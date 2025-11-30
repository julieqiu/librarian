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

package rust

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
	toml "github.com/pelletier/go-toml/v2"
)

// Create creates a new Rust client library from scratch.
// This is equivalent to `sidekick rust-generate`.
func Create(ctx context.Context, cfg *config.Config, libraryName string, apis []*config.API) error {
	if len(apis) == 0 {
		return fmt.Errorf("at least one API is required")
	}

	// Use first API path to derive output directory.
	// Output: src/generated/<path-without-google-prefix>
	apiPath := apis[0].Path
	output := path.Join("src/generated", strings.TrimPrefix(apiPath, "google/"))

	slog.Info("creating new Rust library", "name", libraryName, "output", output)

	// Prepare cargo workspace by creating a new library crate.
	if err := prepareCargoWorkspace(output); err != nil {
		return err
	}

	// Generate the library code for each API.
	slog.Info("generating library code")
	if err := createGenerate(ctx, cfg, apis, output); err != nil {
		return err
	}

	// Run post-generation tasks.
	return postGenerate(output)
}

// prepareCargoWorkspace creates a new cargo package in the specified output directory.
func prepareCargoWorkspace(outputDir string) error {
	slog.Info("preparing cargo workspace", "output", outputDir)
	if err := command.Run("cargo", "new", "--vcs", "none", "--lib", outputDir); err != nil {
		return fmt.Errorf("failed to create cargo package: %w", err)
	}
	if err := command.Run("taplo", "fmt", "Cargo.toml"); err != nil {
		return fmt.Errorf("failed to format Cargo.toml: %w", err)
	}
	return nil
}

// createGenerate generates the library code using sidekick.
func createGenerate(ctx context.Context, cfg *config.Config, apis []*config.API, output string) error {
	// Get the googleapis directory.
	if cfg.Sources == nil || cfg.Sources.Googleapis == nil {
		return fmt.Errorf("googleapis source is required in librarian.yaml")
	}

	googleapisDir, err := getGoogleapisDir(cfg.Sources.Googleapis)
	if err != nil {
		return err
	}

	// Generate code for each API.
	for _, api := range apis {
		// Create sidekick config for this API.
		year, _, _ := time.Now().Date()
		sidekickCfg := &sidekickconfig.Config{
			General: sidekickconfig.GeneralConfig{
				Language:            "rust",
				SpecificationFormat: "protobuf",
				SpecificationSource: api.Path,
				ServiceConfig:       api.ServiceConfig,
			},
			Codec: map[string]string{
				"copyright-year": fmt.Sprintf("%04d", year),
			},
			Source: map[string]string{
				"googleapis": googleapisDir,
			},
		}

		// Write the .sidekick.toml file.
		if err := sidekickconfig.WriteSidekickToml(output, sidekickCfg); err != nil {
			return fmt.Errorf("failed to write .sidekick.toml: %w", err)
		}

		// Create the API model and generate the code.
		model, err := parser.CreateModel(sidekickCfg)
		if err != nil {
			return fmt.Errorf("failed to create API model for %s: %w", api.Path, err)
		}

		if err := sidekickrust.Generate(model, output, sidekickCfg); err != nil {
			return fmt.Errorf("failed to generate Rust code for %s: %w", api.Path, err)
		}
	}

	return nil
}

// getGoogleapisDir returns the path to the googleapis directory.
func getGoogleapisDir(source *config.Source) (string, error) {
	if source.Dir != "" {
		return source.Dir, nil
	}
	return fetch.RepoDir("github.com/googleapis/googleapis", source.Commit, source.SHA256)
}

// postGenerate runs post-generation tasks on the specified output directory.
func postGenerate(outdir string) error {
	slog.Info("running post-generation tasks")

	// Format the code.
	if err := command.Run("cargo", "fmt"); err != nil {
		return fmt.Errorf("cargo fmt failed: %w", err)
	}

	// Get the package name from Cargo.toml.
	packageName, err := getPackageName(outdir)
	if err != nil {
		return err
	}
	slog.Info("generated new client library", "package", packageName)

	// Run tests.
	slog.Info("running cargo test")
	if err := command.Run("cargo", "test", "--package", packageName); err != nil {
		return fmt.Errorf("cargo test failed: %w", err)
	}

	// Build documentation.
	slog.Info("running cargo doc")
	if err := command.Run("env", "RUSTDOCFLAGS=-D warnings", "cargo", "doc", "--package", packageName, "--no-deps"); err != nil {
		return fmt.Errorf("cargo doc failed: %w", err)
	}

	// Run clippy.
	slog.Info("running cargo clippy")
	if err := command.Run("cargo", "clippy", "--package", packageName, "--", "--deny", "warnings"); err != nil {
		return fmt.Errorf("cargo clippy failed: %w", err)
	}

	// Run typos checker.
	slog.Info("running typos")
	if err := command.Run("typos"); err != nil {
		slog.Warn("typos check failed - please add typos to .typos.toml and fix upstream")
		return fmt.Errorf("typos failed: %w", err)
	}

	return nil
}

// getPackageName reads the package name from Cargo.toml.
func getPackageName(output string) (string, error) {
	filename := filepath.Join(output, "Cargo.toml")
	contents, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", filename, err)
	}

	var manifest cargoManifest
	if err := toml.Unmarshal(contents, &manifest); err != nil {
		return "", fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	if manifest.Package == nil {
		return "", fmt.Errorf("no [package] section in %s", filename)
	}

	return manifest.Package.Name, nil
}
