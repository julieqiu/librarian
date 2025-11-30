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

// Package golang implements Go client library generation.
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
	// Skip libraries with no generatable APIs (handwritten libraries like auth).
	// Check for APIs with valid googleapis paths (starting with "google/").
	hasGeneratableAPIs := false
	for _, api := range library.APIs {
		if strings.HasPrefix(api.Path, "google/") && !api.DisableGAPIC {
			hasGeneratableAPIs = true
			break
		}
	}
	if !hasGeneratableAPIs {
		fmt.Printf("skipping %s: no APIs to generate\n", library.Name)
		return nil
	}

	googleapisDir, err := sourceDir(sources.Googleapis, googleapisRepo)
	if err != nil {
		return err
	}

	// Derive output path from library name if not set.
	// For google-cloud-go, libraries are at the root with the library name as directory.
	outputDir := library.Output
	if outputDir == "" {
		outputDir = library.Name
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
		// Skip APIs without valid googleapis paths.
		if !strings.HasPrefix(api.Path, "google/") {
			continue
		}
		if err := invokeProtoc(ctx, library, api, googleapisDir, tempDir); err != nil {
			return fmt.Errorf("protoc failed for api %q: %w", api.Path, err)
		}
		if err := generateRepoMetadata(library, api, googleapisDir, tempDir); err != nil {
			return fmt.Errorf("failed to generate .repo-metadata.json for api %q: %w", api.Path, err)
		}
	}

	modulePath := ""
	if library.Go != nil {
		modulePath = library.Go.ModulePath
	}
	if err := flattenOutput(tempDir, library.Name, modulePath, outputDir); err != nil {
		return fmt.Errorf("failed to flatten output: %w", err)
	}

	if err := goimports(ctx, outputDir); err != nil {
		// Log but don't fail - generated code may have issues from the gapic generator
		fmt.Printf("warning: goimports failed: %v\n", err)
	}

	return nil
}

// invokeProtoc runs protoc to generate Go client code for the given API.
func invokeProtoc(_ context.Context, library *config.Library, api *config.API, googleapisDir, outputDir string) error {
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

// protocPath returns the protoc binary path, checking PROTOC env var first.
func protocPath() string {
	if p := os.Getenv("PROTOC"); p != "" {
		return p
	}
	return "protoc"
}

// buildProtocArgs constructs the protoc command arguments.
func buildProtocArgs(library *config.Library, api *config.API, googleapisDir, outputDir string, protoFiles []string) []string {
	args := []string{
		protocPath(),
		"--experimental_allow_proto3_optional",
	}

	// Determine which proto compiler plugins to use.
	// By default, use modern plugins (protoc-gen-go + protoc-gen-go-grpc).
	// If legacy_grpc is set, use the v1 plugin with plugins=grpc option.
	legacyGRPC := api.Go != nil && api.Go.LegacyGRPC

	if legacyGRPC {
		args = append(args,
			"--go_v1_out="+outputDir,
			"--go_v1_opt=plugins=grpc",
		)
	} else {
		args = append(args,
			"--go_out="+outputDir,
			"--go-grpc_out="+outputDir,
			"--go-grpc_opt=require_unimplemented_servers=false",
		)
	}

	// Add GAPIC plugin arguments.
	args = append(args, "--go_gapic_out="+outputDir)

	// Build GAPIC options.
	// go-gapic-package is required. Use explicit value if set, otherwise derive from library name and API path.
	importPath := ""
	if api.Go != nil && api.Go.ImportPath != "" {
		importPath = api.Go.ImportPath
	} else {
		importPath = deriveGoGapicPackage(library.Name, api.Path)
	}
	if importPath != "" {
		args = append(args, "--go_gapic_opt=go-gapic-package="+importPath)
	}
	if api.ServiceConfig != "" {
		args = append(args, "--go_gapic_opt=api-service-config="+filepath.Join(googleapisDir, api.ServiceConfig))
	}
	if api.GRPCServiceConfig != "" {
		args = append(args, "--go_gapic_opt=grpc-service-config="+filepath.Join(googleapisDir, api.Path, api.GRPCServiceConfig))
	}
	if api.Transport != "" {
		args = append(args, "--go_gapic_opt=transport="+api.Transport)
	}
	if api.ReleaseLevel != "" {
		args = append(args, "--go_gapic_opt=release-level="+api.ReleaseLevel)
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

// deriveGoGapicPackage derives the go-gapic-package value from the API path.
// The format is: "cloud.google.com/go/{path}/api{version};{packagename}"
//
// Examples:
//   - "google/cloud/accessapproval/v1" → "cloud.google.com/go/accessapproval/apiv1;accessapproval"
//   - "google/ai/generativelanguage/v1" → "cloud.google.com/go/ai/generativelanguage/apiv1;generativelanguage"
//   - "google/cloud/bigquery/connection/v1" → "cloud.google.com/go/bigquery/connection/apiv1;connection"
func deriveGoGapicPackage(_, apiPath string) string {
	if apiPath == "" {
		return ""
	}

	// Strip "google/" prefix.
	path := strings.TrimPrefix(apiPath, "google/")
	// Strip "cloud/" prefix (for google/cloud/... paths).
	path = strings.TrimPrefix(path, "cloud/")

	// Split into components.
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}

	// Find the version component (starts with "v" followed by digit).
	versionIdx := -1
	for i, part := range parts {
		if len(part) > 1 && part[0] == 'v' && part[1] >= '0' && part[1] <= '9' {
			versionIdx = i
			break
		}
	}
	if versionIdx == -1 || versionIdx == 0 {
		return ""
	}

	// Path components before version.
	pathParts := parts[:versionIdx]
	version := parts[versionIdx]

	// Package name is the last component before version.
	packageName := pathParts[len(pathParts)-1]
	packageName = strings.ReplaceAll(packageName, "-", "")

	// Build import path.
	importPath := strings.Join(pathParts, "/")
	return fmt.Sprintf("cloud.google.com/go/%s/api%s;%s", importPath, version, packageName)
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
	} else if importPath != "" {
		// Derive module path from import path by stripping the apiv* suffix.
		// e.g., "cloud.google.com/go/accessapproval/apiv1" -> "cloud.google.com/go/accessapproval"
		modulePath = deriveModulePath(importPath)
	}

	// Build doc URL.
	docURL := buildDocURL(modulePath, importPath)

	// Determine release level.
	releaseLevel := determineReleaseLevel(importPath, api.ReleaseLevel)

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

// deriveModulePath derives the Go module path from an import path.
// Go modules in google-cloud-go are always at cloud.google.com/go/{name}.
// For example:
//   - "cloud.google.com/go/accessapproval/apiv1" -> "cloud.google.com/go/accessapproval"
//   - "cloud.google.com/go/ai/generativelanguage/apiv1" -> "cloud.google.com/go/ai"
func deriveModulePath(importPath string) string {
	parts := strings.Split(importPath, "/")
	// Go modules are always cloud.google.com/go/{name}, so take first 3 parts.
	// parts[0] = "cloud.google.com", parts[1] = "go", parts[2] = "{name}"
	if len(parts) >= 3 {
		return strings.Join(parts[:3], "/")
	}
	return importPath
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
// and API-level release level (from BUILD.bazel).
func determineReleaseLevel(importPath, apiReleaseLevel string) string {
	// Check import path for alpha/beta.
	if i := strings.LastIndex(importPath, "/"); i != -1 {
		lastElem := importPath[i+1:]
		if strings.Contains(lastElem, "alpha") || strings.Contains(lastElem, "beta") {
			return "preview"
		}
	}

	// Check API-level release level (from BUILD.bazel).
	if apiReleaseLevel == "alpha" || apiReleaseLevel == "beta" {
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
// structure to the output directory.
// For versioned modules (e.g., dataproc with module path cloud.google.com/go/dataproc/v2),
// protoc generates to cloud.google.com/go/dataproc/v2/apiv1/... and we need to
// flatten that to outputDir/apiv1/... (removing the v2 level).
func flattenOutput(tempDir, libraryName, modulePath, outputDir string) error {
	// Determine the source directory based on module path.
	// For versioned modules like cloud.google.com/go/dataproc/v2, the generated files
	// are at tempDir/cloud.google.com/go/dataproc/v2/...
	// For non-versioned modules, they're at tempDir/cloud.google.com/go/libraryName/...
	var srcDir string
	if modulePath != "" {
		parts := strings.Split(modulePath, "/")
		// Module path format: cloud.google.com/go/{name} or cloud.google.com/go/{name}/{version}
		if len(parts) == 4 {
			// Versioned module: cloud.google.com/go/{name}/{version}
			srcDir = filepath.Join(tempDir, "cloud.google.com", "go", parts[2], parts[3])
		} else if len(parts) == 3 {
			// Non-versioned module: cloud.google.com/go/{name}
			srcDir = filepath.Join(tempDir, "cloud.google.com", "go", parts[2])
		}
	}
	if srcDir == "" {
		srcDir = filepath.Join(tempDir, "cloud.google.com", "go", libraryName)
	}

	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", srcDir)
	}

	// Move files from source to output directory.
	if err := moveFiles(srcDir, outputDir); err != nil {
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

	// Ensure target directory exists.
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
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

// goimports runs goimports on the generated files using the go tool directive.
func goimports(_ context.Context, dir string) error {
	return command.Run("go", "tool", "goimports", "-w", dir)
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
