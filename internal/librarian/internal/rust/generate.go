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
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
	for _, api := range library.APIs {
		sidekickConfig := toSidekickConfig(library, api, googleapisDir, discoveryDir)
		model, err := parser.CreateModel(sidekickConfig)
		if err != nil {
			return err
		}
		if err := sidekickrust.Generate(model, library.Output, sidekickConfig); err != nil {
			return err
		}
	}
	return nil
}

// Format formats all Cargo.toml and .rs files in the given directories.
func Format(dirs ...string) error {
	var tomlFiles, rsFiles []string
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if d.Name() == "Cargo.toml" {
				tomlFiles = append(tomlFiles, path)
			} else if filepath.Ext(path) == ".rs" {
				rsFiles = append(rsFiles, path)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	// Format all Cargo.toml files with taplo.
	if len(tomlFiles) > 0 {
		args := append([]string{"fmt"}, tomlFiles...)
		if err := command.Run("taplo", args...); err != nil {
			return err
		}
	}

	// Format all .rs files with rustfmt.
	if len(rsFiles) > 0 {
		// rustfmt defaults to 2015 edition when run directly on files. Specify
		// 2024 to match the edition in Cargo.toml.
		args := append([]string{"--edition", "2024"}, rsFiles...)
		if err := command.Run("rustfmt", args...); err != nil {
			return err
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

// versionRegex matches 'version = "X.Y.Z"' in Cargo.toml.
var versionRegex = regexp.MustCompile(`^version\s*=\s*"([^"]+)"`)

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

// readVersion reads the version from an existing Cargo.toml.
// Returns the version string, or empty string if the file doesn't exist
// or doesn't have a version field.
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
