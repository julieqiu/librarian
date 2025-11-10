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

package generate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/container/go/request"
	"github.com/googleapis/librarian/internal/container/golang/execv"
)

// Test substitution vars.
var (
	execvRun = execv.Run
)

// Config holds the internal librariangen configuration for the generate command.
type Config struct {
	// LibrarianDir is the path to the librarian-tool input directory.
	// It is expected to contain the generate-request.json file.
	LibrarianDir string
	// InputDir is the path to the .librarian/generator-input directory from the
	// language repository.
	InputDir string
	// OutputDir is the path to the empty directory where librariangen writes
	// its output.
	OutputDir string
	// SourceDir is the path to a complete checkout of the googleapis repository.
	SourceDir string
	// DisablePostProcessor controls whether the post-processor is run.
	// This should always be false in production.
	DisablePostProcessor bool
}

// Validate ensures that the configuration is valid.
func (c *Config) Validate() error {
	if c.LibrarianDir == "" {
		return errors.New("librariangen: librarian directory must be set")
	}
	if c.InputDir == "" {
		return errors.New("librariangen: input directory must be set")
	}
	if c.OutputDir == "" {
		return errors.New("librariangen: output directory must be set")
	}
	if c.SourceDir == "" {
		return errors.New("librariangen: source directory must be set")
	}
	return nil
}

// Generate is the main entrypoint for the `generate` command. It orchestrates
// the entire generation process. The high-level steps are:
//
//  1. Validate the configuration.
//  2. Invoke `protoc` for each API specified in the request, generating Go
//     files into a nested directory structure (e.g.,
//     `/output/cloud.google.com/go/...`).
//  3. Fix the permissions of all generated `.go` files to `0644`.
//  4. Flatten the output directory, moving the generated module(s) to the top
//     level of the output directory (e.g., `/output/chronicle`).
//  5. If the `DisablePostProcessor` flag is false, run the post-processor on the
//     generated module(s), updating versions for snippet metadata,
//     running go mod tidy etc.
//
// The `DisablePostProcessor` flag should always be false in production. It can be
// true during development to inspect the "raw" protoc output before any
// post-processing is applied.
func Generate(ctx context.Context, artifact *config.ArtifactState, googleapisDir string, outDir string) error {
	if artifact.Generate == nil {
		return errors.New("librariangen: artifact has no generate configuration")
	}
	if len(artifact.Generate.APIs) == 0 {
		return errors.New("librariangen: no APIs in artifact configuration")
	}

	// Phase 1: Code Generation
	for _, api := range artifact.Generate.APIs {
		apiServiceDir := filepath.Join(googleapisDir, api.Path)
		args, err := buildProtocCommand(&api, googleapisDir, outDir)
		if err != nil {
			return fmt.Errorf("librariangen: failed to build protoc command for api %q: %w", api.Path, err)
		}

		// Run protoc
		if err := execvRun(ctx, args, outDir); err != nil {
			return fmt.Errorf("librariangen: protoc failed for api %q: %w", api.Path, err)
		}
	}

	// Flatten output directory structure
	if err := flattenOutput(outDir); err != nil {
		return fmt.Errorf("librariangen: failed to flatten output: %w", err)
	}

	// Phase 2: Formatting and Build
	if err := formatAndBuild(ctx, outDir); err != nil {
		return fmt.Errorf("librariangen: formatting and build failed: %w", err)
	}

	// Phase 3: Testing is handled by the Build command (see builder.go)
	if err := goBuild(ctx, moduleDir); err != nil {
		return fmt.Errorf("librariangen: failed to run 'go build': %w", err)
	}
	if err := goTest(ctx, moduleDir); err != nil {
		return fmt.Errorf("librariangen: failed to run 'go test': %w", err)
	}
	return nil
}

// buildProtocCommand constructs the protoc command arguments for a given API.
func buildProtocCommand(api *config.APIConfig, googleapisDir string, outDir string) ([]string, error) {
	args := []string{
		"protoc",
		"--proto_path=" + googleapisDir,
		"--go_out=" + outDir,
		"--go-grpc_out=" + outDir,
		"--go_gapic_out=" + outDir,
	}

	// Add GAPIC import path if specified
	if api.GAPICImportPath != "" {
		args = append(args, "--go_gapic_opt=go-gapic-package="+api.GAPICImportPath)
	}

	// Add gRPC service config if specified
	if api.GRPCServiceConfig != "" {
		configPath := filepath.Join(googleapisDir, api.Path, api.GRPCServiceConfig)
		args = append(args, "--go_gapic_opt=grpc-service-config="+configPath)
	}

	// Add service YAML if specified
	if api.ServiceYAML != "" {
		yamlPath := filepath.Join(googleapisDir, api.Path, api.ServiceYAML)
		args = append(args, "--go_gapic_opt=api-service-config="+yamlPath)
	}

	// Add transport option if specified
	if api.Transport != "" {
		args = append(args, "--go_gapic_opt=transport="+api.Transport)
	}

	// Add rest-numeric-enums if specified
	if api.RestNumericEnums {
		args = append(args, "--go_gapic_opt=rest-numeric-enums")
	}

	// Add any additional optional arguments
	for _, optArg := range api.OptArgs {
		args = append(args, "--go_gapic_opt="+optArg)
	}

	// Add the proto files for this API
	// TODO: Find all .proto files in the API path
	protoFiles := filepath.Join(googleapisDir, api.Path, "*.proto")
	args = append(args, protoFiles)

	return args, nil
}

// formatAndBuild runs formatting tools and initializes go modules.
func formatAndBuild(ctx context.Context, outDir string) error {
	// Run goimports to format code
	goimportsArgs := []string{"goimports", "-w", "."}
	if err := execvRun(ctx, goimportsArgs, outDir); err != nil {
		return fmt.Errorf("goimports failed: %w", err)
	}

	// Run go mod tidy
	goModTidyArgs := []string{"go", "mod", "tidy"}
	if err := execvRun(ctx, goModTidyArgs, outDir); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	return nil
}

// flattenOutput moves the contents of /output/cloud.google.com/go/ to the top
// level of /output.
func flattenOutput(outputDir string) error {
	goDir := filepath.Join(outputDir, "cloud.google.com", "go")
	if _, err := os.Stat(goDir); os.IsNotExist(err) {
		// Directory doesn't exist, nothing to flatten
		return nil
	}

	files, err := os.ReadDir(goDir)
	if err != nil {
		return fmt.Errorf("librariangen: failed to read dir %s: %w", goDir, err)
	}

	for _, f := range files {
		oldPath := filepath.Join(goDir, f.Name())
		newPath := filepath.Join(outputDir, f.Name())
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("librariangen: failed to move %s to %s: %w", oldPath, newPath, err)
		}
	}

	// Remove the now-empty cloud.google.com directory.
	if err := os.RemoveAll(filepath.Join(outputDir, "cloud.google.com")); err != nil {
		return fmt.Errorf("librariangen: failed to remove cloud.google.com: %w", err)
	}
	return nil
}

// goBuild builds all the code under the specified directory
func goBuild(ctx context.Context, dir, module string) error {
	args := []string{"go", "build", "./..."}
	return execvRun(ctx, args, dir)
}

// goTest builds all the code under the specified directory
func goTest(ctx context.Context, dir, module string) error {
	args := []string{"go", "test", "./...", "-short"}
	return execvRun(ctx, args, dir)
}

// readBuildReq reads generate-request.json from the librarian-tool input directory.
// The request file tells librariangen which library and APIs to generate.
// It is prepared by the Librarian tool and mounted at /librarian.
func readBuildReq(librarianDir string) (*request.Library, error) {
	reqPath := filepath.Join(librarianDir, "build-request.json")

	buildReq, err := requestParse(reqPath)
	if err != nil {
		return nil, err
	}
	return buildReq, nil
