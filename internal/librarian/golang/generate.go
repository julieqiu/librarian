// Copyright 2026 Google LLC
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
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

var (
	//go:embed template/_README.md.txt
	readmeTmpl string
)

// readmeData contains the template data for generating a README.md file.
type readmeData struct {
	Name       string
	ModulePath string
}

// Generate is the main entrypoint for the `generate` command. It orchestrates
// the entire generation process. The high-level steps are:
//
//  1. Invoke `protoc` for each channel specified in the library, generating Go
//     files into a nested directory structure (e.g.,
//     `/output/cloud.google.com/go/...`).
//  2. Fix the permissions of all generated `.go` files to `0644`.
//  3. Flatten the output directory, moving the generated module(s) to the top
//     level of the output directory (e.g., `/output/chronicle`).
//  4. Generate configuration files (README, version files, go.mod setup).
func Generate(ctx context.Context, library *config.Library, googleapisDir string) error {
	moduleRoot := filepath.Join(library.Output, library.Name)
	isNewLibrary := !dirExists(library.Output)

	if err := os.MkdirAll(moduleRoot, 0755); err != nil {
		return fmt.Errorf("failed to create module directory: %w", err)
	}

	if err := invokeProtoc(ctx, library, googleapisDir); err != nil {
		return fmt.Errorf("gapic generation failed: %w", err)
	}
	if err := flattenOutput(library.Output); err != nil {
		return fmt.Errorf("failed to flatten output: %w", err)
	}

	modPath := buildModulePath(library)
	if err := applyModuleVersion(library.Output, library.Name, modPath); err != nil {
		return fmt.Errorf("failed to apply module version to output directories: %w", err)
	}

	if library.Go != nil {
		if err := deleteOutputPaths(library.Output, library.Go.DeleteGenerationOutputPaths); err != nil {
			return fmt.Errorf("failed to delete paths specified in delete_generation_output_paths: %w", err)
		}
	}

	if err := generateREADME(library, googleapisDir); err != nil {
		return err
	}
	if err := generateInternalVersionFile(moduleRoot, library.Version); err != nil {
		return err
	}

	if isNewLibrary {
		if err := addSnippetsReplaceDirective(ctx, library); err != nil {
			return err
		}
	}

	for _, channel := range library.Channels {
		if err := generateClientVersionFile(library, channel.Path); err != nil {
			return err
		}
	}

	return nil
}

// invokeProtoc handles the protoc GAPIC generation logic for the 'generate' CLI command.
// It reads a library, and for each channel specified, it invokes protoc
// to generate the client library and its corresponding .repo-metadata.json file.
func invokeProtoc(ctx context.Context, library *config.Library, googleapisDir string) error {
	for _, channel := range library.Channels {
		api, err := serviceconfig.Find(googleapisDir, channel.Path)
		if err != nil {
			return fmt.Errorf("failed to find service config: %w", err)
		}

		grpcServiceConfigPath, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, channel.Path)
		if err != nil {
			return fmt.Errorf("failed to find gRPC service config: %w", err)
		}

		var nestedProtos []string
		apiCfg := apiConfig(library, channel.Path)
		if apiCfg != nil {
			nestedProtos = apiCfg.NestedProtos
		}

		var serviceConfigPath string
		if api != nil && api.ServiceConfig != "" {
			serviceConfigPath = filepath.Join(googleapisDir, api.ServiceConfig)
		}

		args, err := buildProtocCommand(library, channel.Path, googleapisDir, serviceConfigPath, grpcServiceConfigPath, nestedProtos)
		if err != nil {
			return fmt.Errorf("failed to build protoc command for channel %q in library %q: %w", channel.Path, library.Name, err)
		}
		if err := command.Run(ctx, args[0], args[1:]...); err != nil {
			return fmt.Errorf("protoc failed for channel %q in library %q: %w", channel.Path, library.Name, err)
		}

		if serviceConfigPath != "" {
			defaultVersion := library.Version
			if defaultVersion == "" {
				defaultVersion = "0.0.0"
			}
			if err := repometadata.Generate(library, "go", "googleapis/google-cloud-go", serviceConfigPath, defaultVersion, library.Output); err != nil {
				return fmt.Errorf("failed to generate .repo-metadata.json for channel %q in library %q: %w", channel.Path, library.Name, err)
			}
		}
	}
	return nil
}

// generateREADME creates a README.md file in the module's root directory,
// using the service config for the first channel in the library to obtain the
// service's title.
func generateREADME(library *config.Library, googleapisDir string) error {
	if len(library.Channels) == 0 {
		return errors.New("cannot generate README without any channels")
	}

	api, err := serviceconfig.Find(googleapisDir, library.Channels[0].Path)
	if err != nil {
		return fmt.Errorf("failed to find service config: %w", err)
	}
	if api == nil || api.Title == "" {
		return errors.New("no title found in service config")
	}

	readmePath := filepath.Join(library.Output, library.Name, "README.md")
	modPath := buildModulePath(library)

	readmeFile, err := os.Create(readmePath)
	if err != nil {
		return err
	}
	t := template.Must(template.New("readme").Parse(readmeTmpl))
	data := readmeData{
		Name:       api.Title,
		ModulePath: modPath,
	}
	err = t.Execute(readmeFile, data)
	cerr := readmeFile.Close()
	if err != nil {
		return err
	}
	return cerr
}

// addSnippetsReplaceDirective adds a go.mod replace directive in the snippets
// directory to reference the library module locally.
func addSnippetsReplaceDirective(ctx context.Context, library *config.Library) error {
	outputSnippetsDir := snippetsDir(library.Output, "")
	if err := os.MkdirAll(outputSnippetsDir, 0755); err != nil {
		return err
	}

	modPath := buildModulePath(library)
	relativeDir := "../../../" + library.Name
	replaceStr := fmt.Sprintf("%s=%s", modPath, relativeDir)
	return command.RunInDir(ctx, outputSnippetsDir, "go", "mod", "edit", "-replace", replaceStr)
}

// snippetsDir returns the path to the snippets directory for a library.
// If outputDir is empty, returns a relative path. If libraryName is empty,
// returns the path to the snippets root directory.
func snippetsDir(outputDir, libraryName string) string {
	parts := []string{"internal", "generated", "snippets"}
	if outputDir != "" {
		parts = append([]string{outputDir}, parts...)
	}
	if libraryName != "" {
		parts = append(parts, libraryName)
	}
	return filepath.Join(parts...)
}

// dirExists checks if the directory exists.
func dirExists(dir string) bool {
	_, err := os.Stat(dir)
	return !os.IsNotExist(err)
}

// buildModulePath returns the module path for the library, applying
// any configured version.
func buildModulePath(library *config.Library) string {
	prefix := "cloud.google.com/go/" + library.Name
	if library.Go != nil && library.Go.ModulePathVersion != "" {
		return prefix + "/" + library.Go.ModulePathVersion
	}

	// No override: assume implicit v1.
	return prefix
}

// apiConfig returns the Go-specific configuration for the API identified by
// its path within googleapis (e.g. "google/cloud/functions/v2").
// If no API-specific configuration is found, nil is returned.
func apiConfig(library *config.Library, path string) *config.GoAPI {
	if library.Go == nil {
		return nil
	}
	for _, cfg := range library.Go.GoAPIs {
		if cfg.Path == path {
			return cfg
		}
	}
	return nil
}

// protoPackage returns the protobuf package for the API,
// which is derived from the path unless overridden in GoAPI config.
func protoPackage(library *config.Library, path string) string {
	cfg := apiConfig(library, path)
	if cfg != nil && cfg.ProtoPackage != "" {
		return cfg.ProtoPackage
	}

	// No override: derive the value.
	return strings.ReplaceAll(path, "/", ".")
}

// clientDirectory returns the directory for the clients of this
// API, relative to the module root.
func clientDirectory(library *config.Library, path string) (string, error) {
	cfg := apiConfig(library, path)
	if cfg != nil && cfg.ClientDirectory != "" {
		return cfg.ClientDirectory, nil
	}

	// No override: derive the value.
	startOfModuleName := strings.Index(path, library.Name+"/")
	if startOfModuleName == -1 {
		return "", fmt.Errorf("unexpected API path format: %s", path)
	}

	// google/spanner/v1 => ["google", "spanner", "v1"]
	// google/spanner/admin/instance/v1 => ["google", "spanner", "admin", "instance", "v1"]
	parts := strings.Split(path, "/")
	moduleIndex := slices.Index(parts, library.Name)
	if moduleIndex == -1 {
		return "", fmt.Errorf("module name '%s' not found in API path '%s'", library.Name, path)
	}

	// Remove everything up to and include the module name.
	// google/spanner/v1 => ["v1"]
	// google/spanner/admin/instance/v1 => ["admin", "instance", "v1"]
	parts = parts[moduleIndex+1:]
	parts[len(parts)-1] = "api" + parts[len(parts)-1]
	return strings.Join(parts, "/"), nil
}

// flattenOutput moves the contents of /output/cloud.google.com/go/ to the top
// level of /output.
//
// Failure here may be indicative that input artifacts did NOT generate artifacts
// in the expected location (e.g. wrong go_package path, etc).
func flattenOutput(outputDir string) error {
	goDir := filepath.Join(outputDir, "cloud.google.com", "go")
	if _, err := os.Stat(goDir); os.IsNotExist(err) {
		return fmt.Errorf("go directory does not exist in path: %s", goDir)
	}
	if err := moveFiles(goDir, outputDir); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(outputDir, "cloud.google.com")); err != nil {
		return fmt.Errorf("failed to remove cloud.google.com: %w", err)
	}
	return nil
}

// applyModuleVersion reorganizes the (already flattened) output directory
// appropriately for versioned modules. For a module path of the form
// cloud.google.com/go/{module-id}/{version}, we expect to find
// /output/{id}/{version} and /output/internal/generated/snippets/{module-id}/{version}.
// In most cases, we only support a single major version of the module, rooted at
// /{module-id} in the repository, so the content of these directories are moved into
// /output/{module-id} and /output/internal/generated/snippets/{id}.
//
// However, when we need to support multiple major versions, we use {module-id}/{version}
// as the *library* ID (in the state file etc). That indicates that the module is rooted
// in that versioned directory (e.g. "pubsub/v2"). In that case, the flattened code is
// already in the right place, so this function doesn't need to do anything.
func applyModuleVersion(outputDir, libraryID, modulePath string) error {
	parts := strings.Split(modulePath, "/")
	if len(parts) == 3 {
		return nil
	}
	if len(parts) != 4 {
		return fmt.Errorf("unexpected module path format: %s", modulePath)
	}
	id := parts[2]      // e.g. dataproc
	version := parts[3] // e.g. v2

	if libraryID == id+"/"+version {
		return nil
	}

	srcDir := filepath.Join(outputDir, id)
	srcVersionDir := filepath.Join(srcDir, version)
	snpDir := snippetsDir(outputDir, id)
	snippetsVersionDir := filepath.Join(snpDir, version)

	if err := moveFiles(srcVersionDir, srcDir); err != nil {
		return err
	}
	if err := os.RemoveAll(srcVersionDir); err != nil {
		return fmt.Errorf("failed to remove %s: %w", srcVersionDir, err)
	}

	if err := moveFiles(snippetsVersionDir, snpDir); err != nil {
		return err
	}
	if err := os.RemoveAll(snippetsVersionDir); err != nil {
		return fmt.Errorf("failed to remove %s: %w", snippetsVersionDir, err)
	}
	return nil
}

// moveFiles moves all files (and directories) from sourceDir to targetDir.
func moveFiles(sourceDir, targetDir string) error {
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read dir %s: %w", sourceDir, err)
	}
	for _, f := range files {
		oldPath := filepath.Join(sourceDir, f.Name())
		newPath := filepath.Join(targetDir, f.Name())
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to move %s to %s: %w", oldPath, newPath, err)
		}
	}
	return nil
}

// deleteOutputPaths deletes the specified paths, which may be files
// or directories, relative to the output directory. This is an emergency
// escape hatch for situations where files are generated that we don't want
// to include, such as the internal/generated/snippets/storage/internal directory.
// This is configured in librarian.yaml at the library level with the key
// delete_generation_output_paths.
func deleteOutputPaths(outputDir string, pathsToDelete []string) error {
	for _, path := range pathsToDelete {
		if err := os.RemoveAll(filepath.Join(outputDir, path)); err != nil {
			return err
		}
	}
	return nil
}
