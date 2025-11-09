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

type generateNewRunner struct {
	path       string
	all        bool
	commit     bool
	repoRoot   string
	repoConfig *config.LibrarianConfig
}

func newGenerateNewRunner(args []string, all, commit bool) (*generateNewRunner, error) {
	if !all && len(args) < 1 {
		return nil, fmt.Errorf("missing required argument <path> or --all flag")
	}
	if all && len(args) > 0 {
		return nil, fmt.Errorf("cannot specify both <path> and --all flag")
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments, expected: librarian generate <path>")
	}

	var path string
	if len(args) > 0 {
		path = args[0]
	}

	// Get current working directory as repo root
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Read repository config
	repoConfig, err := config.ReadLibrarianConfig(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read .librarian.yaml: %w", err)
	}

	// Verify repository supports generation
	if !repoConfig.HasGenerate() {
		return nil, fmt.Errorf("repository does not support code generation (no generate section in .librarian.yaml)")
	}

	return &generateNewRunner{
		path:       path,
		all:        all,
		commit:     commit,
		repoRoot:   repoRoot,
		repoConfig: repoConfig,
	}, nil
}

func (r *generateNewRunner) run(ctx context.Context) error {
	if r.all {
		return r.generateAll(ctx)
	}
	return r.generateSingle(ctx, r.path)
}

func (r *generateNewRunner) generateSingle(ctx context.Context, artifactPath string) error {
	fullPath := filepath.Join(r.repoRoot, artifactPath)

	// Read artifact state
	artifactState, err := config.ReadArtifactState(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact state at %s: %w", artifactPath, err)
	}

	// Verify artifact has generate section
	if !artifactState.HasGenerate() {
		return fmt.Errorf("artifact %s does not have a generate section", artifactPath)
	}

	slog.Info("generating code for artifact", "path", artifactPath)

	// TODO: Implement actual generation
	// This would involve:
	// 1. Clone googleapis at the specified ref (if not already cloned)
	// 2. Prepare request files for the container with API configurations
	// 3. Run the generator container with appropriate mounts
	// 4. Apply keep/remove/exclude rules to the output
	// 5. Update .librarian.yaml with generation metadata (commit, librarian version)

	fmt.Printf("Generating code for %s...\n", artifactPath)
	fmt.Printf("  APIs: %d\n", len(artifactState.Generate.APIs))
	fmt.Printf("  Container: %s:%s\n", artifactState.Generate.Container.Image, artifactState.Generate.Container.Tag)
	fmt.Printf("  Googleapis: %s @ %s\n", artifactState.Generate.Googleapis.Repo, artifactState.Generate.Googleapis.Ref)

	// Update artifact state with generation metadata
	artifactState.Generate.Commit = "TODO: actual commit SHA after generation"
	artifactState.Generate.Librarian = cli.Version()

	// Write back
	if err := config.WriteArtifactState(fullPath, artifactState); err != nil {
		return fmt.Errorf("failed to write artifact state: %w", err)
	}

	slog.Info("updated artifact state", "path", artifactPath)
	fmt.Printf("✓ Generated code for %s (placeholder - full implementation pending)\n", artifactPath)

	if r.commit {
		slog.Warn("--commit flag not yet implemented")
		fmt.Println("Note: --commit flag not yet implemented")
	}

	return nil
}

func (r *generateNewRunner) generateAll(ctx context.Context) error {
	// Find all artifacts with .librarian.yaml that have a generate section
	artifacts, err := r.findGeneratableArtifacts()
	if err != nil {
		return err
	}

	if len(artifacts) == 0 {
		fmt.Println("No artifacts with generate section found")
		return nil
	}

	fmt.Printf("Found %d artifact(s) with generate section\n", len(artifacts))

	successCount := 0
	failCount := 0

	for _, artifact := range artifacts {
		fmt.Printf("\n=== Generating %s ===\n", artifact)
		if err := r.generateSingle(ctx, artifact); err != nil {
			fmt.Printf("✗ Failed to generate %s: %v\n", artifact, err)
			failCount++
		} else {
			successCount++
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Success: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failCount)

	if failCount > 0 {
		return fmt.Errorf("failed to generate %d artifact(s)", failCount)
	}

	return nil
}

func (r *generateNewRunner) findGeneratableArtifacts() ([]string, error) {
	var artifacts []string

	// Walk the repository looking for .librarian.yaml files
	err := filepath.Walk(r.repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root .librarian.yaml
		if path == filepath.Join(r.repoRoot, ".librarian.yaml") {
			return nil
		}

		// Look for .librarian.yaml files
		if !info.IsDir() && info.Name() == ".librarian.yaml" {
			// Read the artifact state
			artifactDir := filepath.Dir(path)
			artifactState, err := config.ReadArtifactState(artifactDir)
			if err != nil {
				slog.Warn("failed to read artifact state", "path", artifactDir, "error", err)
				return nil // Continue walking
			}

			// Check if it has a generate section
			if artifactState.HasGenerate() {
				// Get relative path from repo root
				relPath, err := filepath.Rel(r.repoRoot, artifactDir)
				if err != nil {
					return err
				}
				artifacts = append(artifacts, relPath)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find artifacts: %w", err)
	}

	return artifacts, nil
}
