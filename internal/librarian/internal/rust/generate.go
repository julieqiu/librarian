// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rust

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
)

const (
	googleapisRepo = "github.com/googleapis/googleapis"
	discoveryRepo  = "github.com/googleapis/discovery-artifact-manager"
)

// Generate generates a Rust client library.
func Generate(ctx context.Context, library *config.Library, sources *config.Sources) error {
	// Read copyright year and version from existing Cargo.toml if not set in config.
	// This avoids storing these values in librarian.yaml while preserving existing ones.
	if library.CopyrightYear == "" {
		library.CopyrightYear = readCopyrightYear(library.Output)
	}
	if library.Version == "" {
		library.Version = readVersion(library.Output)
	}
	googleapisDir, err := sourceDir(sources.Googleapis, googleapisRepo)
	if err != nil {
		return err
	}
	discoveryDir, err := sourceDir(sources.Discovery, discoveryRepo)
	if err != nil {
		return err
	}
	sidekickConfig := toSidekickConfig(library, library.ServiceConfig, googleapisDir, discoveryDir)
	model, err := parser.CreateModel(sidekickConfig)
	if err != nil {
		return err
	}
	if err := sidekickrust.Generate(model, library.Output, sidekickConfig); err != nil {
		return err
	}
	if err := command.Run("taplo", "fmt", filepath.Join(library.Output, "Cargo.toml")); err != nil {
		return err
	}
	// Create stub files for kept .rs files if they don't exist.
	// This is needed because rustfmt resolves module declarations and will fail
	// if the corresponding .rs files don't exist. Kept files are handwritten
	// files that may not exist yet in a fresh output directory.
	if library.Generate != nil {
		if err := createStubFiles(library.Output, library.Generate.Keep); err != nil {
			return err
		}
	}
	rsFiles, err := filepath.Glob(filepath.Join(library.Output, "src", "*.rs"))
	if err != nil {
		return err
	}
	if len(rsFiles) > 0 {
		args := append([]string{"--edition", "2024"}, rsFiles...)
		if err := command.Run("rustfmt", args...); err != nil {
			return err
		}
	}
	return nil
}

// createStubFiles creates empty stub files for .rs files in the keep list if they don't exist.
// This allows rustfmt to succeed even when the handwritten module files are missing.
func createStubFiles(dir string, keep []string) error {
	for _, k := range keep {
		if !strings.HasSuffix(k, ".rs") {
			continue
		}
		filePath := filepath.Join(dir, k)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// Ensure parent directory exists.
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return err
			}
			// Create an empty stub file.
			if err := os.WriteFile(filePath, []byte("// TODO: implement this module\n"), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func sourceDir(source *config.Source, repo string) (string, error) {
	if source == nil {
		return "", nil
	}
	if source.Dir != "" {
		return source.Dir, nil
	}
	return fetch.RepoDir(repo, source.Commit, source.SHA256)
}

// copyrightYearRegex matches "# Copyright YYYY" at the start of Cargo.toml.
var copyrightYearRegex = regexp.MustCompile(`^# Copyright (\d{4})`)

// readCopyrightYear reads the copyright year from an existing Cargo.toml.
// Returns the year as a string, or the current year if the file doesn't exist
// or doesn't have a copyright header.
func readCopyrightYear(dir string) string {
	cargoPath := filepath.Join(dir, "Cargo.toml")
	f, err := os.Open(cargoPath)
	if err != nil {
		return currentYear()
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		if matches := copyrightYearRegex.FindStringSubmatch(scanner.Text()); len(matches) == 2 {
			return matches[1]
		}
	}
	return currentYear()
}

func currentYear() string {
	return time.Now().Format("2006")
}

// versionRegex matches 'version = "X.Y.Z"' in Cargo.toml.
var versionRegex = regexp.MustCompile(`^version\s*=\s*"([^"]+)"`)

// readVersion reads the version from an existing Cargo.toml.
// Returns the version as a string, or empty string if not found.
func readVersion(dir string) string {
	cargoPath := filepath.Join(dir, "Cargo.toml")
	f, err := os.Open(cargoPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if matches := versionRegex.FindStringSubmatch(scanner.Text()); len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}
