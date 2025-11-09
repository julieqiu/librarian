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
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/gitrepo"
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

	fmt.Printf("Generating code for %s...\n", artifactPath)
	fmt.Printf("  APIs: %d\n", len(artifactState.Generate.APIs))
	fmt.Printf("  Container: %s:%s\n", artifactState.Generate.Container.Image, artifactState.Generate.Container.Tag)
	fmt.Printf("  Googleapis: %s @ %s\n", artifactState.Generate.Googleapis.Repo, artifactState.Generate.Googleapis.Ref)

	// 1. Clone or update googleapis repository
	googleapisRepo, err := r.ensureGoogleapisRepo(artifactState.Generate.Googleapis)
	if err != nil {
		return fmt.Errorf("failed to ensure googleapis repository: %w", err)
	}

	// Get the commit hash from googleapis
	commitHash, err := googleapisRepo.HeadHash()
	if err != nil {
		return fmt.Errorf("failed to get googleapis commit hash: %w", err)
	}
	slog.Info("using googleapis commit", "hash", commitHash)

	// 2. Prepare working directories
	outputDir, err := os.MkdirTemp("", "librarian-generate-*")
	if err != nil {
		return fmt.Errorf("failed to create temp output directory: %w", err)
	}
	defer os.RemoveAll(outputDir)

	// 3. Prepare request files for container
	if err := r.prepareGeneratorInput(fullPath, artifactState); err != nil {
		return fmt.Errorf("failed to prepare generator input: %w", err)
	}

	// 4. Run generator container
	containerImage := fmt.Sprintf("%s:%s", artifactState.Generate.Container.Image, artifactState.Generate.Container.Tag)
	dockerClient, err := docker.New(r.repoRoot, containerImage, &docker.DockerOptions{
		UserUID: os.Getenv("UID"),
		UserGID: os.Getenv("GID"),
	})
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}

	if err := dockerClient.Generate(ctx, &docker.GenerateRequest{
		GoogleapisDir: googleapisRepo.GetDir(),
		Output:        outputDir,
		RepoDir:       fullPath,
		State:         r.convertToLibrarianState(artifactState, artifactPath),
		LibraryID:     artifactPath,
	}); err != nil {
		return fmt.Errorf("failed to run generator container: %w", err)
	}

	// 5. Apply keep/remove/exclude rules and copy files
	if err := r.applyFilesRulesAndCopy(artifactState, fullPath, outputDir); err != nil {
		return fmt.Errorf("failed to apply file rules and copy: %w", err)
	}

	// 6. Update artifact state with generation metadata
	artifactState.Generate.Commit = commitHash
	artifactState.Generate.Librarian = cli.Version()

	if err := config.WriteArtifactState(fullPath, artifactState); err != nil {
		return fmt.Errorf("failed to write artifact state: %w", err)
	}

	slog.Info("updated artifact state", "path", artifactPath)
	fmt.Printf("✓ Generated code for %s\n", artifactPath)

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

// ensureGoogleapisRepo clones or opens the googleapis repository at the specified ref.
func (r *generateNewRunner) ensureGoogleapisRepo(googleapis config.RepositoryRef) (gitrepo.Repository, error) {
	// Determine googleapis directory (use a cache directory)
	googleapisDir := filepath.Join(r.repoRoot, ".librarian-cache", "googleapis")

	// Clone or open the repository
	repo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{
		Dir:          googleapisDir,
		MaybeClone:   true,
		RemoteURL:    googleapis.Repo,
		RemoteBranch: "master", // Default branch
		Depth:        1,        // Shallow clone for speed
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone/open googleapis: %w", err)
	}

	// Checkout the specified ref if provided
	if googleapis.Ref != "" {
		slog.Info("checking out googleapis ref", "ref", googleapis.Ref)
		if err := repo.Checkout(googleapis.Ref); err != nil {
			return nil, fmt.Errorf("failed to checkout ref %s: %w", googleapis.Ref, err)
		}
	}

	return repo, nil
}

// prepareGeneratorInput prepares the generator input directory with API configuration files.
func (r *generateNewRunner) prepareGeneratorInput(artifactPath string, artifactState *config.ArtifactState) error {
	generatorInputDir := filepath.Join(artifactPath, config.GeneratorInputDir)

	// Create generator input directory
	if err := os.MkdirAll(generatorInputDir, 0755); err != nil {
		return fmt.Errorf("failed to create generator input directory: %w", err)
	}

	// Write API configuration files for each API
	for _, api := range artifactState.Generate.APIs {
		apiFile := filepath.Join(generatorInputDir, strings.ReplaceAll(api.Path, "/", "_")+".json")
		apiConfig := map[string]interface{}{
			"api": api.Path,
		}

		data, err := json.MarshalIndent(apiConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal API config: %w", err)
		}

		if err := os.WriteFile(apiFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write API config file: %w", err)
		}
		slog.Debug("created API config file", "api", api, "file", apiFile)
	}

	return nil
}

// convertToLibrarianState converts ArtifactState to the old LibrarianState format
// needed by the docker container interface.
func (r *generateNewRunner) convertToLibrarianState(artifactState *config.ArtifactState, artifactPath string) *config.LibrarianState {
	// For now, use the artifact path as the source root
	// TODO(https://github.com/googleapis/librarian/issues/<number>): Extract from metadata or configuration
	sourceRoots := []string{artifactPath}

	// Convert APIConfig to API format expected by docker
	var apis []*config.API
	for _, apiConfig := range artifactState.Generate.APIs {
		apis = append(apis, &config.API{
			Path:          apiConfig.Path,
			ServiceConfig: apiConfig.GRPCServiceConfig,
		})
	}

	libraryState := &config.LibraryState{
		ID:          artifactPath,
		APIs:        apis,
		SourceRoots: sourceRoots,
	}

	return &config.LibrarianState{
		Libraries: []*config.LibraryState{libraryState},
	}
}

// applyFilesRulesAndCopy applies keep/remove/exclude rules and copies generated files.
func (r *generateNewRunner) applyFilesRulesAndCopy(artifactState *config.ArtifactState, artifactPath, outputDir string) error {
	// Get file patterns from artifact state
	keepPatterns := artifactState.Generate.Keep
	removePatterns := artifactState.Generate.Remove
	excludePatterns := artifactState.Generate.Exclude

	slog.Info("applying file rules", "keep", len(keepPatterns), "remove", len(removePatterns), "exclude", len(excludePatterns))

	// 1. First, remove files matching remove patterns from the artifact directory
	if err := r.removeFiles(artifactPath, removePatterns, keepPatterns); err != nil {
		return fmt.Errorf("failed to remove files: %w", err)
	}

	// 2. Copy generated files from output directory to artifact directory, excluding files that match exclude patterns
	if err := r.copyGeneratedFiles(outputDir, artifactPath, excludePatterns); err != nil {
		return fmt.Errorf("failed to copy generated files: %w", err)
	}

	return nil
}

// removeFiles removes files from the artifact directory based on remove patterns, preserving keep patterns.
func (r *generateNewRunner) removeFiles(artifactPath string, removePatterns, keepPatterns []string) error {
	if len(removePatterns) == 0 {
		return nil
	}

	// Compile patterns
	var removeRegexes []*regexp.Regexp
	for _, pattern := range removePatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid remove pattern %q: %w", pattern, err)
		}
		removeRegexes = append(removeRegexes, re)
	}

	var keepRegexes []*regexp.Regexp
	for _, pattern := range keepPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid keep pattern %q: %w", pattern, err)
		}
		keepRegexes = append(keepRegexes, re)
	}

	// Walk the artifact directory and remove matching files
	return filepath.Walk(artifactPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the artifact directory itself
		if path == artifactPath {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(artifactPath, path)
		if err != nil {
			return err
		}

		// Check if file should be kept
		for _, keepRe := range keepRegexes {
			if keepRe.MatchString(relPath) {
				slog.Debug("keeping file (matches keep pattern)", "file", relPath)
				return nil
			}
		}

		// Check if file should be removed
		for _, removeRe := range removeRegexes {
			if removeRe.MatchString(relPath) {
				slog.Debug("removing file", "file", relPath)
				if err := os.RemoveAll(path); err != nil {
					return fmt.Errorf("failed to remove %s: %w", path, err)
				}
				// If we removed a directory, skip walking into it
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		return nil
	})
}

// copyGeneratedFiles copies files from output directory to artifact directory, excluding files matching exclude patterns.
func (r *generateNewRunner) copyGeneratedFiles(outputDir, artifactPath string, excludePatterns []string) error {
	// Compile exclude patterns
	var excludeRegexes []*regexp.Regexp
	for _, pattern := range excludePatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid exclude pattern %q: %w", pattern, err)
		}
		excludeRegexes = append(excludeRegexes, re)
	}

	// Walk the output directory and copy files
	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the output directory itself
		if path == outputDir {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(outputDir, path)
		if err != nil {
			return err
		}

		// Check if file should be excluded
		for _, excludeRe := range excludeRegexes {
			if excludeRe.MatchString(relPath) {
				slog.Debug("excluding file from copy", "file", relPath)
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Determine destination path
		destPath := filepath.Join(artifactPath, relPath)

		if info.IsDir() {
			// Create directory
			if err := os.MkdirAll(destPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			slog.Debug("created directory", "dir", relPath)
		} else {
			// Copy file
			if err := r.copyFile(path, destPath); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", relPath, err)
			}
			slog.Debug("copied file", "file", relPath)
		}

		return nil
	})
}

// copyFile copies a file from src to dst.
func (r *generateNewRunner) copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}
