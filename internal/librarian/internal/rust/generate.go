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
	"context"
	"os"
	"path/filepath"

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
	var extraModules []string
	if library.Rust != nil {
		extraModules = library.Rust.ExtraModules
	}
	if err := cleanOutput(library.Output, extraModules); err != nil {
		return err
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
	// Create stub files for extra_modules if they don't exist.
	// This is needed because rustfmt resolves module declarations and will fail
	// if the corresponding .rs files don't exist. Extra modules are handwritten
	// files that may not exist yet in a fresh output directory.
	if err := createExtraModuleStubs(library.Output, extraModules); err != nil {
		return err
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

// createExtraModuleStubs creates empty stub files for extra_modules if they don't exist.
// This allows rustfmt to succeed even when the handwritten module files are missing.
func createExtraModuleStubs(dir string, extraModules []string) error {
	srcDir := filepath.Join(dir, "src")
	for _, mod := range extraModules {
		modFile := filepath.Join(srcDir, mod+".rs")
		if _, err := os.Stat(modFile); os.IsNotExist(err) {
			// Create an empty stub file.
			if err := os.WriteFile(modFile, []byte("// TODO: implement this module\n"), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// cleanOutput removes all files and directories in the output directory except
// Cargo.toml and files corresponding to extraModules (e.g., "errors" -> "src/errors.rs").
func cleanOutput(dir string, extraModules []string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// Build a set of extra module filenames to keep.
	keepFiles := make(map[string]bool)
	for _, mod := range extraModules {
		keepFiles[mod+".rs"] = true
	}

	for _, entry := range entries {
		if entry.Name() == "Cargo.toml" {
			continue
		}
		// Keep the src directory but clean its contents selectively.
		if entry.Name() == "src" && entry.IsDir() {
			if err := cleanSrcDir(filepath.Join(dir, "src"), keepFiles); err != nil {
				return err
			}
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

// cleanSrcDir removes all files in the src directory except those in keepFiles.
func cleanSrcDir(srcDir string, keepFiles map[string]bool) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if keepFiles[entry.Name()] {
			continue
		}
		if err := os.RemoveAll(filepath.Join(srcDir, entry.Name())); err != nil {
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
