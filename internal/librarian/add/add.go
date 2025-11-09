// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package add

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Config holds the configuration for adding a new library.
type Config struct {
	// GoogleAPIsRoot is the path to the googleapis repository.
	GoogleAPIsRoot string
	// RepoRoot is the path to the language repository.
	RepoRoot string
	// Language is the target language (go, python, rust).
	Language string
}

// Add adds a new library by extracting metadata from googleapis and creating a .librarian.yaml entry.
// It does NOT generate any scaffolding files - that happens in librarian generate.
func Add(cfg *Config, apiPath string) error {
	// 1. Validate API path exists
	fullPath := filepath.Join(cfg.GoogleAPIsRoot, apiPath)
	if !pathExists(fullPath) {
		return fmt.Errorf("API path not found: %s", apiPath)
	}

	// 2. Read BUILD.bazel (except for Rust which doesn't use it)
	var buildContent []byte
	var err error
	if cfg.Language != "rust" {
		buildPath := filepath.Join(fullPath, "BUILD.bazel")
		buildContent, err = os.ReadFile(buildPath)
		if err != nil {
			return fmt.Errorf("failed to read BUILD.bazel: %w", err)
		}
	}

	// 3. Extract metadata (language-specific)
	var metadata Metadata
	switch cfg.Language {
	case "go":
		metadata, err = extractGoMetadata(apiPath, buildContent)
	case "python":
		metadata, err = extractPythonMetadata(apiPath, buildContent)
	case "rust":
		metadata, err = extractRustMetadata(apiPath, cfg.GoogleAPIsRoot)
	default:
		return fmt.Errorf("unsupported language: %s", cfg.Language)
	}
	if err != nil {
		return fmt.Errorf("failed to extract metadata: %w", err)
	}

	// 4. Create library config
	library, err := createLibraryConfig(cfg.Language, metadata)
	if err != nil {
		return fmt.Errorf("failed to create library config: %w", err)
	}

	// 5. Save to .librarian.yaml
	if err := saveLibraryConfig(cfg.RepoRoot, library); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// 6. Commit config only
	if err := commitConfig(cfg.RepoRoot, library); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// 7. Print next steps
	fmt.Printf("✓ Created .librarian.yaml entry for %s\n", library.Name())
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  librarian generate %s\n", library.Name())

	return nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func commitConfig(repoRoot string, library Library) error {
	// Add config file(s)
	exec.Command("git", "-C", repoRoot, "add", ".librarian.yaml").Run()
	exec.Command("git", "-C", repoRoot, "add", ".librarian/*.yaml").Run()

	// Commit
	cmd := exec.Command("git", "-C", repoRoot, "commit", "-m",
		fmt.Sprintf("feat(%s): add library configuration", library.Name()))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
