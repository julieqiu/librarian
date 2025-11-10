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
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/container/go/config"
	"github.com/googleapis/librarian/internal/container/go/execv"
	"github.com/googleapis/librarian/internal/container/go/module"
	"github.com/googleapis/librarian/internal/container/go/request"
	"gopkg.in/yaml.v3"
)

// External string template vars.
var (
	//go:embed _README.md.txt
	readmeTmpl string
	//go:embed _version.go.txt
	versionTmpl string
)

// NewAPIStatus is the API.Status value used to represent "this is a new API being configured".
const NewAPIStatus = "new"

// Test substitution vars.
var (
	execvRun     = execv.Run
	requestParse = Parse
	responseSave = saveResponse
)

// Configure configures a new library by generating initial files.
// This includes README.md, CHANGES.md, version files, and go.mod updates.
//
// This function is called by `librarian generate` the first time a library
// is generated. It should only create configuration files, not generated code.
//
// Parameters:
//   - outputDir: Directory where files will be written (e.g., the library root)
//   - sourceDir: Path to the googleapis repository for reading service YAML files
//   - libraryID: Library name (e.g., "secretmanager")
//   - modulePath: Go module path (e.g., "cloud.google.com/go/secretmanager")
//   - firstAPIPath: Path to the first API (e.g., "google/cloud/secretmanager/v1")
//   - firstAPIServiceConfig: Service config file for the first API (e.g., "secretmanager_v1.yaml")
//   - clientDir: Client directory (e.g., "apiv1")
//   - snippetsGoModPath: Path to internal/generated/snippets/go.mod (optional, can be empty)
func Configure(ctx context.Context, outputDir, sourceDir, libraryID, modulePath, firstAPIPath, firstAPIServiceConfig, clientDir, snippetsGoModPath string) error {
	slog.Debug("librariangen: configuring new library", "library", libraryID)

	// Create library directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("librariangen: failed to create library directory: %w", err)
	}

	// Generate README.md
	if err := createReadme(outputDir, sourceDir, libraryID, modulePath, firstAPIPath, firstAPIServiceConfig); err != nil {
		return fmt.Errorf("librariangen: failed to generate README: %w", err)
	}

	// Generate CHANGES.md
	if err := createChanges(outputDir); err != nil {
		return fmt.Errorf("librariangen: failed to generate CHANGES: %w", err)
	}

	// Generate internal/version.go
	if err := module.GenerateInternalVersionFile(outputDir, "0.0.0"); err != nil {
		return fmt.Errorf("librariangen: failed to generate internal version file: %w", err)
	}

	// Generate client version file (e.g., apiv1/version.go)
	if err := createClientVersionFile(outputDir, modulePath, clientDir); err != nil {
		return fmt.Errorf("librariangen: failed to generate client version file: %w", err)
	}

	// Update snippets go.mod with replace directive
	if snippetsGoModPath != "" {
		if err := updateSnippetsGoMod(ctx, snippetsGoModPath, modulePath, libraryID); err != nil {
			return fmt.Errorf("librariangen: failed to update snippets go.mod: %w", err)
		}
	}

	slog.Debug("librariangen: configuration complete", "library", libraryID)
	return nil
}

// readConfigureReq reads generate-request.json from the librarian-tool input directory.
// The request file tells librariangen which library and APIs to generate.
// It is prepared by the Librarian tool and mounted at /librarian.
func readConfigureReq(librarianDir string) (*Request, error) {
	reqPath := filepath.Join(librarianDir, "configure-request.json")
	slog.Debug("librariangen: reading configure request", "path", reqPath)

	configureReq, err := requestParse(reqPath)
	if err != nil {
		return nil, err
	}
	slog.Debug("librariangen: successfully unmarshalled request")
	return configureReq, nil
}

// saveConfigureResp saves the response in configure-response.json in the librarian-tool input directory.
// The response file tells Librarian how to reconfigure the library in its state file.
func saveConfigureResp(resp *request.Library, librarianDir string) error {
	respPath := filepath.Join(librarianDir, "configure-response.json")
	slog.Debug("librariangen: saving configure response", "path", respPath)

	if err := responseSave(resp, respPath); err != nil {
		return err
	}
	slog.Debug("librariangen: successfully marshalled response")
	return nil
}

// findLibraryAndAPIToConfigure examines a request, and finds a single library
// containing a single new API, returning both of them. An error is returned
// if there is not exactly one library containing exactly one new API.
func findLibraryAndAPIToConfigure(req *Request) (*request.Library, *request.API, error) {
	var library *request.Library
	var api *request.API
	for _, candidate := range req.Libraries {
		var newAPI *request.API
		for _, api := range candidate.APIs {
			if api.Status == NewAPIStatus {
				if newAPI != nil {
					return nil, nil, fmt.Errorf("librariangen: library %s has multiple new APIs", candidate.ID)
				}
				newAPI = &api
			}
		}

		if newAPI != nil {
			if library != nil {
				return nil, nil, fmt.Errorf("librariangen: multiple libraries have new APIs (at least %s and %s)", library.ID, candidate.ID)
			}
			library = candidate
			api = newAPI
		}
	}
	if library == nil {
		return nil, nil, fmt.Errorf("librariangen: no libraries have new APIs")
	}
	return library, api, nil
}

// configureLibrary performs the real work of configuring a new or updated module,
// creating files and populating the state file entry.
// In theory we could just have a return type of "error", but logically this is
// returning the configure-response... it just happens to be "the library being configured"
// at the moment. If the format of configure-response ever changes, we'll need fewer
// changes if we don't make too many assumptions now.
func configureLibrary(ctx context.Context, cfg *Config, library *request.Library, api *request.API) (*request.Library, error) {
	// It's just *possible* the new path has a manually configured
	// client directory - but even if not, RepoConfig has the logic
	// for figuring out the client directory. Even if the new path
	// doesn't have a custom configuration, we can use this to
	// work out the module path, e.g. if there's a major version other
	// than v1.
	repoConfig, err := config.LoadRepoConfig(cfg.LibrarianDir)
	if err != nil {
		return nil, err
	}
	var moduleConfig = repoConfig.GetModuleConfig(library.ID)

	moduleRoot := filepath.Join(cfg.OutputDir, library.ID)
	if err := os.Mkdir(moduleRoot, 0755); err != nil {
		return nil, err
	}
	// Only a single API path can be added on each configure call, so we can tell
	// if this is a new library if it's got exactly one API path.
	// In that case, we need to add:
	// - CHANGES.md (static text: "# Changes")
	// - README.md
	// - internal/version.go
	// - go.mod
	if len(library.APIs) == 1 {
		library.SourcePaths = []string{library.ID, "internal/generated/snippets/" + library.ID}
		library.RemoveRegex = []string{"^internal/generated/snippets/" + library.ID + "/"}
		library.TagFormat = "{id}/v{version}"
		library.Version = "0.0.0"
		if err := createReadme(cfg, library); err != nil {
			return nil, err
		}
		if err := createChanges(cfg, library); err != nil {
			return nil, err
		}
		if err := module.GenerateInternalVersionFile(moduleRoot, library.Version); err != nil {
			return nil, err
		}
		if err := goModEditReplaceInSnippets(ctx, cfg, moduleConfig.GetModulePath(), "../../../"+library.ID); err != nil {
			return nil, err
		}
		// The postprocessor for the generate command will run "go mod init" and "go mod tidy"
		// - because it has the source code at that point. It *won't* have the version files we've
		// created here though. That's okay so long as our version.go files don't have any dependencies.
	}

	// Whether it's a new library or not, generate a version file for the new client directory.
	if err := createClientVersionFile(cfg, moduleConfig, api.Path); err != nil {
		return nil, err
	}

	// Make changes in the Library object, to communicate state file changes back to
	// Librarian.
	if err := updateLibraryState(moduleConfig, library, api); err != nil {
		return nil, err
	}

	return library, nil
}

// createReadme generates a README.md file in the module's root directory,
// using the service config for the first API in the library to obtain the
// service's title.
func createReadme(outputDir, sourceDir, libraryID, modulePath, firstAPIPath, firstAPIServiceConfig string) error {
	readmePath := filepath.Join(outputDir, "README.md")
	serviceYAMLPath := filepath.Join(sourceDir, firstAPIPath, firstAPIServiceConfig)
	title, err := readTitleFromServiceYAML(serviceYAMLPath)
	if err != nil {
		return fmt.Errorf("librariangen: failed to read title from service yaml: %w", err)
	}

	slog.Info("librariangen: creating file", "path", readmePath)
	readmeFile, err := os.Create(readmePath)
	if err != nil {
		return err
	}
	defer readmeFile.Close()
	t := template.Must(template.New("readme").Parse(readmeTmpl))
	readmeData := struct {
		Name       string
		ModulePath string
	}{
		Name:       title,
		ModulePath: modulePath,
	}
	return t.Execute(readmeFile, readmeData)
}

// createChanges generates a CHANGES.md file at the root of the module.
func createChanges(outputDir string) error {
	changesPath := filepath.Join(outputDir, "CHANGES.md")
	slog.Info("librariangen: creating file", "path", changesPath)
	content := "# Changes\n"
	return os.WriteFile(changesPath, []byte(content), 0644)
}

// createClientVersionFile creates a version.go file for a client.
func createClientVersionFile(outputDir, modulePath, clientDir string) error {
	fullClientDir := filepath.Join(outputDir, clientDir)
	if err := os.MkdirAll(fullClientDir, 0755); err != nil {
		return err
	}
	versionPath := filepath.Join(fullClientDir, "version.go")
	slog.Info("librariangen: creating file", "path", versionPath)
	t := template.Must(template.New("version").Parse(versionTmpl))
	versionData := struct {
		Year               int
		Package            string
		ModuleRootInternal string
	}{
		Year:               time.Now().Year(),
		Package:            filepath.Base(clientDir), // The package name is the client directory name (e.g. `apiv1`).
		ModuleRootInternal: modulePath + "/internal",
	}
	f, err := os.Create(versionPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, versionData)
}

// updateSnippetsGoMod updates internal/generated/snippets/go.mod with a replace directive
// to point to the local module.
func updateSnippetsGoMod(ctx context.Context, snippetsGoModPath, modulePath, libraryID string) error {
	snippetsDir := filepath.Dir(snippetsGoModPath)
	relativeDir := "../../../" + libraryID
	replaceStr := fmt.Sprintf("%s=%s", modulePath, relativeDir)
	args := []string{"go", "mod", "edit", "-replace", replaceStr}
	slog.Info("librariangen: running go mod edit -replace", "replace", replaceStr, "directory", snippetsDir)
	return execvRun(ctx, args, snippetsDir)
}

// updateLibraryState updates the library to add any required removal/preservation
// regexes for the specified API.
func updateLibraryState(moduleConfig *config.ModuleConfig, library *request.Library, api *request.API) error {
	apiConfig := moduleConfig.GetAPIConfig(api.Path)
	clientDirectory, err := apiConfig.GetClientDirectory()
	if err != nil {
		return err
	}
	apiParts := strings.Split(api.Path, "/")
	protobufDir := apiParts[len(apiParts)-2] + "pb/.*"
	generatedPaths := []string{
		"[^/]*_client\\.go",
		"[^/]*_client_example_go123_test\\.go",
		"[^/]*_client_example_test\\.go",
		"auxiliary\\.go",
		"auxiliary_go123\\.go",
		"doc\\.go",
		"gapic_metadata\\.json",
		"helpers\\.go",
		"\\.repo-metadata\\.json",
		protobufDir,
	}
	for _, generatedPath := range generatedPaths {
		library.RemoveRegex = append(library.RemoveRegex, "^"+path.Join(library.ID, clientDirectory, generatedPath)+"$")
	}
	return nil
}

// readTitleFromServiceYAML reads the service YAML file and returns the title.
func readTitleFromServiceYAML(path string) (string, error) {
	slog.Info("librariangen: reading service yaml", "path", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("librariangen: failed to read service yaml file: %w", err)
	}
	var serviceConfig struct {
		Title string `yaml:"title"`
	}
	if err := yaml.Unmarshal(data, &serviceConfig); err != nil {
		return "", fmt.Errorf("librariangen: failed to unmarshal service yaml: %w", err)
	}
	if serviceConfig.Title == "" {
		return "", errors.New("librariangen: title not found in service yaml")
	}
	return serviceConfig.Title, nil
}

// Request corresponds to a librarian configure request.
// It is unmarshalled from the configure-request.json file. Note that
// this request is in a different form from most other requests, as it
// contains all libraries.
type Request struct {
	// All libraries configured within the repository.
	Libraries []*request.Library `json:"libraries"`
}

// Parse reads a configure-request.json file from the given path and unmarshals
// it into a ConfigureRequest struct.
func Parse(path string) (*Request, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("librariangen: failed to read request file from %s: %w", path, err)
	}

	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("librariangen: failed to unmarshal request file %s: %w", path, err)
	}

	return &req, nil
}
