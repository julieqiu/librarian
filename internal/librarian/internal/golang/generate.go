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

package golang

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	googleapisRepo = "github.com/googleapis/googleapis"
)

// Generate generates a Go client library.
func Generate(ctx context.Context, library *config.Library, sources *config.Sources) error {
	googleapisDir, err := sourceDir(sources.Googleapis, googleapisRepo)
	if err != nil {
		return err
	}
	// Create a temporary directory for protoc output.
	tempDir, err := os.MkdirTemp("", "librarian-go-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	for _, api := range library.APIs {
		if api.DisableGAPIC {
			continue
		}
		if err := invokeProtoc(ctx, library, api, googleapisDir, tempDir); err != nil {
			return fmt.Errorf("protoc failed for api %q: %w", api.Path, err)
		}
		if err := generateRepoMetadata(library, api, googleapisDir, tempDir); err != nil {
			return fmt.Errorf("failed to generate .repo-metadata.json for api %q: %w", api.Path, err)
		}
	}

	if err := flattenOutput(tempDir, library.Output); err != nil {
		return fmt.Errorf("failed to flatten output: %w", err)
	}

	if err := goimports(ctx, library.Output); err != nil {
		return fmt.Errorf("failed to run goimports: %w", err)
	}

	return nil
}

// invokeProtoc runs protoc to generate Go client code for the given API.
func invokeProtoc(ctx context.Context, library *config.Library, api *config.API, googleapisDir, outputDir string) error {
	apiDir := filepath.Join(googleapisDir, api.Path)
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		return fmt.Errorf("failed to read API directory %s: %w", apiDir, err)
	}

	var protoFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".proto" {
			protoFiles = append(protoFiles, filepath.Join(apiDir, entry.Name()))
		}
	}
	if len(protoFiles) == 0 {
		return fmt.Errorf("no .proto files found in %s", apiDir)
	}

	args := buildProtocArgs(library, api, googleapisDir, outputDir, protoFiles)
	return command.Run(args[0], args[1:]...)
}

// buildProtocArgs constructs the protoc command arguments.
func buildProtocArgs(library *config.Library, api *config.API, googleapisDir, outputDir string, protoFiles []string) []string {
	args := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
	}

	// Determine which proto compiler plugins to use.
	goGRPC := api.Go != nil && api.Go.GoGRPC != nil && *api.Go.GoGRPC
	legacyGRPC := api.Go != nil && api.Go.LegacyGRPC

	if goGRPC {
		args = append(args,
			"--go_out="+outputDir,
			"--go-grpc_out="+outputDir,
			"--go-grpc_opt=require_unimplemented_servers=false",
		)
	} else {
		args = append(args, "--go_v1_out="+outputDir)
		if legacyGRPC {
			args = append(args, "--go_v1_opt=plugins=grpc")
		}
	}

	// Add GAPIC plugin arguments.
	args = append(args, "--go_gapic_out="+outputDir)

	// Build GAPIC options.
	if api.Go != nil && api.Go.ImportPath != "" {
		args = append(args, "--go_gapic_opt=go-gapic-package="+api.Go.ImportPath)
	}
	if api.ServiceConfig != "" {
		args = append(args, "--go_gapic_opt=api-service-config="+filepath.Join(googleapisDir, api.ServiceConfig))
	}
	if api.GRPCServiceConfig != "" {
		args = append(args, "--go_gapic_opt=grpc-service-config="+filepath.Join(googleapisDir, api.Path, api.GRPCServiceConfig))
	}
	if library.Transport != "" {
		args = append(args, "--go_gapic_opt=transport="+library.Transport)
	}
	if library.ReleaseLevel != "" {
		args = append(args, "--go_gapic_opt=release-level="+library.ReleaseLevel)
	}
	if api.Metadata != nil && *api.Metadata {
		args = append(args, "--go_gapic_opt=metadata")
	}
	if api.DIREGAPIC {
		args = append(args, "--go_gapic_opt=diregapic")
	}
	if api.RESTNumericEnums != nil && *api.RESTNumericEnums {
		args = append(args, "--go_gapic_opt=rest-numeric-enums")
	}

	// Add include path.
	args = append(args, "-I="+googleapisDir)

	// Add proto files.
	args = append(args, protoFiles...)

	return args
}

// repoMetadata is used for JSON marshaling in .repo-metadata.json.
type repoMetadata struct {
	APIShortname        string `json:"api_shortname"`
	ClientDocumentation string `json:"client_documentation"`
	ClientLibraryType   string `json:"client_library_type"`
	Description         string `json:"description"`
	DistributionName    string `json:"distribution_name"`
	Language            string `json:"language"`
	LibraryType         string `json:"library_type"`
	ReleaseLevel        string `json:"release_level"`
}

// generateRepoMetadata generates a .repo-metadata.json file for the API.
func generateRepoMetadata(library *config.Library, api *config.API, googleapisDir, outputDir string) error {
	if api.ServiceConfig == "" {
		return nil
	}

	// Read the service YAML to get title and name.
	serviceYAMLPath := filepath.Join(googleapisDir, api.ServiceConfig)
	serviceConfig, err := yaml.Read[serviceYAML](serviceYAMLPath)
	if err != nil {
		return fmt.Errorf("failed to read service YAML %s: %w", serviceYAMLPath, err)
	}

	// Extract import path (without the package alias).
	importPath := ""
	if api.Go != nil && api.Go.ImportPath != "" {
		importPath = api.Go.ImportPath
		if i := strings.Index(importPath, ";"); i != -1 {
			importPath = importPath[:i]
		}
	}

	// Determine module path.
	modulePath := ""
	if library.Go != nil && library.Go.ModulePath != "" {
		modulePath = library.Go.ModulePath
	}

	// Build doc URL.
	docURL := buildDocURL(modulePath, importPath)

	// Determine release level.
	releaseLevel := determineReleaseLevel(importPath, library.ReleaseLevel)

	// Extract API shortname from the full name.
	apiShortname := extractAPIShortname(serviceConfig.Name)

	metadata := repoMetadata{
		APIShortname:        apiShortname,
		ClientDocumentation: docURL,
		ClientLibraryType:   "generated",
		Description:         serviceConfig.Title,
		DistributionName:    importPath,
		Language:            "go",
		LibraryType:         "GAPIC_AUTO",
		ReleaseLevel:        releaseLevel,
	}

	// Determine output path from the import path.
	outputPath := filepath.Join(outputDir, filepath.FromSlash(importPath), ".repo-metadata.json")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", outputPath, err)
	}

	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	jsonData = append(jsonData, '\n')

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	return nil
}

// serviceYAML holds the fields we need from the service YAML file.
type serviceYAML struct {
	Title string `yaml:"title"`
	Name  string `yaml:"name"`
}

// buildDocURL constructs the documentation URL for the API.
func buildDocURL(modulePath, importPath string) string {
	if modulePath == "" || importPath == "" {
		return ""
	}
	pkgPath := strings.TrimPrefix(strings.TrimPrefix(importPath, modulePath), "/")
	return "https://cloud.google.com/go/docs/reference/" + modulePath + "/latest/" + pkgPath
}

// determineReleaseLevel determines the release level based on the import path
// and the configured release level.
func determineReleaseLevel(importPath, configuredLevel string) string {
	// Check import path for alpha/beta.
	if i := strings.LastIndex(importPath, "/"); i != -1 {
		lastElem := importPath[i+1:]
		if strings.Contains(lastElem, "alpha") || strings.Contains(lastElem, "beta") {
			return "preview"
		}
	}

	// Check configured release level.
	if configuredLevel == "alpha" || configuredLevel == "beta" {
		return "preview"
	}

	return "stable"
}

// extractAPIShortname extracts the API shortname from the full service name.
// For example, "secretmanager.googleapis.com" returns "secretmanager".
func extractAPIShortname(nameFull string) string {
	parts := strings.Split(nameFull, ".")
	return parts[0]
}

// flattenOutput moves generated files from the nested cloud.google.com/go/...
// structure to the top level of the output directory.
func flattenOutput(tempDir, outputDir string) error {
	goDir := filepath.Join(tempDir, "cloud.google.com", "go")
	if _, err := os.Stat(goDir); os.IsNotExist(err) {
		return fmt.Errorf("go directory does not exist: %s", goDir)
	}

	if err := moveFiles(goDir, outputDir); err != nil {
		return err
	}

	return nil
}

// moveFiles moves all files and directories from sourceDir to targetDir.
func moveFiles(sourceDir, targetDir string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", sourceDir, err)
	}

	for _, entry := range entries {
		oldPath := filepath.Join(sourceDir, entry.Name())
		newPath := filepath.Join(targetDir, entry.Name())

		// If target exists and is a directory, merge contents recursively.
		if entry.IsDir() {
			if info, err := os.Stat(newPath); err == nil && info.IsDir() {
				if err := moveFiles(oldPath, newPath); err != nil {
					return err
				}
				continue
			}
		}

		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to move %s to %s: %w", oldPath, newPath, err)
		}
	}

	return nil
}

// goimports runs goimports on the generated files.
func goimports(ctx context.Context, dir string) error {
	return command.Run("goimports", "-w", dir)
}

func sourceDir(source *config.Source, repo string) (string, error) {
	if source == nil {
		return "", errors.New("source is required")
	}
	if source.Dir != "" {
		return source.Dir, nil
	}
	return fetch.RepoDir(repo, source.Commit, source.SHA256)
}
