// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
)

type addRunner struct {
	path         string
	apis         []string
	commit       bool
	repoRoot     string
	repoConfig   *config.LibrarianConfig
}

func newAddRunner(args []string, commit bool) (*addRunner, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("missing required argument <path>")
	}

	path := args[0]
	apis := args[1:]

	// Get current working directory as repo root
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Read repository config
	repoConfig, err := config.ReadLibrarianConfig(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read .librarian.yaml: %w (run 'librarian init' first)", err)
	}

	return &addRunner{
		path:       path,
		apis:       apis,
		commit:     commit,
		repoRoot:   repoRoot,
		repoConfig: repoConfig,
	}, nil
}

func (r *addRunner) run(ctx context.Context) error {
	artifactPath := filepath.Join(r.repoRoot, r.path)

	// Check if directory exists
	if stat, err := os.Stat(artifactPath); err != nil {
		if os.IsNotExist(err) {
			// Create the directory
			if err := os.MkdirAll(artifactPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", artifactPath, err)
			}
			slog.Info("created directory", "path", artifactPath)
		} else {
			return fmt.Errorf("failed to check directory %s: %w", artifactPath, err)
		}
	} else if !stat.IsDir() {
		return fmt.Errorf("%s exists but is not a directory", artifactPath)
	}

	// Check if .librarian.yaml already exists in the artifact directory
	configPath := filepath.Join(artifactPath, ".librarian.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf(".librarian.yaml already exists at %s", configPath)
	}

	// Create the artifact state
	artifactState := &config.ArtifactState{}

	// If APIs are provided AND repo has generate section, add generate state
	if len(r.apis) > 0 && r.repoConfig.HasGenerate() {
		if err := r.addGenerateState(ctx, artifactState); err != nil {
			return err
		}
	}

	// If repo has release section, add release state
	if r.repoConfig.HasRelease() {
		artifactState.Release = &config.ArtifactReleaseState{
			Version: nil, // null initially
		}
	}

	// Validate the state
	if artifactState.Generate == nil && artifactState.Release == nil {
		return fmt.Errorf("artifact must have either generate or release section (repository config determines capabilities)")
	}

	// Write the artifact state
	if err := config.WriteArtifactState(artifactPath, artifactState); err != nil {
		return fmt.Errorf("failed to write .librarian.yaml: %w", err)
	}

	slog.Info("created .librarian.yaml", "path", configPath)

	// Print summary
	fmt.Printf("Added %s to librarian management\n", r.path)
	if artifactState.Generate != nil {
		fmt.Printf("  - generate section with %d API(s)\n", len(artifactState.Generate.APIs))
	}
	if artifactState.Release != nil {
		fmt.Printf("  - release section (version: null)\n")
	}

	// TODO: Handle --commit flag to create git commit
	if r.commit {
		slog.Warn("--commit flag not yet implemented")
	}

	return nil
}

func (r *addRunner) addGenerateState(ctx context.Context, artifactState *config.ArtifactState) error {
	if r.repoConfig.Librarian.Language == "" {
		return fmt.Errorf("repository language not set in .librarian.yaml (required for API generation)")
	}

	// TODO: Clone googleapis if needed
	// For now, assume googleapis is available or will be cloned later

	generateState := &config.ArtifactGenerateState{
		APIs:       make([]config.APIConfig, 0, len(r.apis)),
		Librarian:  cli.Version(),
		Container:  r.repoConfig.Generate.Container,
		Googleapis: r.repoConfig.Generate.Googleapis,
		Discovery:  r.repoConfig.Generate.Discovery,
	}

	// Parse BUILD.bazel for each API
	for _, apiPath := range r.apis {
		slog.Info("parsing BUILD.bazel", "api", apiPath)

		// TODO: Actually parse BUILD.bazel from googleapis
		// For now, create a placeholder API config
		apiConfig := config.APIConfig{
			Path: apiPath,
		}

		// If we had googleapis cloned, we would do:
		// googleapisRoot := ... (path to cloned googleapis)
		// parsedConfig, err := config.ParseBUILDFile(googleapisRoot, apiPath, r.repoConfig.Librarian.Language)
		// if err != nil {
		//     return fmt.Errorf("failed to parse BUILD.bazel for %s: %w", apiPath, err)
		// }
		// apiConfig = *parsedConfig

		generateState.APIs = append(generateState.APIs, apiConfig)
	}

	artifactState.Generate = generateState
	return nil
}
