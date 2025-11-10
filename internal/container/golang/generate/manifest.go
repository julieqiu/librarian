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
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/container/go/config"
	"github.com/googleapis/librarian/internal/container/go/request"
	"gopkg.in/yaml.v3"
)

// manifestEntry is used for JSON marshaling in manifest.
type manifestEntry struct {
	APIShortname        string `json:"api_shortname"`
	ClientDocumentation string `json:"client_documentation"`
	ClientLibraryType   string `json:"client_library_type"`
	Description         string `json:"description"`
	DistributionName    string `json:"distribution_name"`
	Language            string `json:"language"`
	LibraryType         string `json:"library_type"`
	ReleaseLevel        string `json:"release_level"`
}

const gapicAutoLibraryType = "GAPIC_AUTO"

// generateRepoMetadata generates a .repo-metadata.json file for a given API.
// It gathers metadata from the service YAML and artifact configuration.
// The generated file is written to the appropriate location within the output directory,
// following the expected structure for .repo-metadata.json files.
func generateRepoMetadata(ctx context.Context, cfg *Config, lib *request.Library, api *request.API, moduleConfig *config.ModuleConfig) error {
	if api.ServiceConfig == "" {
		slog.Info("librariangen: no service config for API, skipping .repo-metadata.json generation", "api_path", api.Path)
		return nil
	}
	apiServiceDir := filepath.Join(cfg.SourceDir, api.Path)
	yamlPath := filepath.Join(apiServiceDir, api.ServiceConfig)

	yamlFile, err := os.Open(yamlPath)
	if err != nil {
		return fmt.Errorf("librariangen: failed to open service YAML file %s: %w", yamlPath, err)
	}
	defer yamlFile.Close()

	yamlConfig := struct {
		Title    string `yaml:"title"`
		NameFull string `yaml:"name"`
	}{}
	if err := yaml.NewDecoder(yamlFile).Decode(&yamlConfig); err != nil {
		return fmt.Errorf("librariangen: failed to decode service YAML: %w", err)
	}

	// Use the GAPIC import path from the API configuration
	importPath := api.GAPICImportPath
	if i := strings.Index(importPath, ";"); i != -1 {
		importPath = importPath[:i]
	}

	docURL, err := docURL(moduleConfig.GetModulePath(), importPath)
	if err != nil {
		return fmt.Errorf("librariangen: unable to build docs URL: %w", err)
	}

	// Use the release level from the API configuration, defaulting to "stable" if not set
	releaseLevel := api.ReleaseLevel
	if releaseLevel == "" {
		releaseLevel = "stable"
	}

	apiShortname := apiShortname(yamlConfig.NameFull)

	entry := manifestEntry{
		APIShortname:        apiShortname,
		ClientDocumentation: docURL,
		ClientLibraryType:   "generated",
		Description:         yamlConfig.Title,
		DistributionName:    importPath,
		Language:            "go",
		LibraryType:         gapicAutoLibraryType,
		ReleaseLevel:        releaseLevel,
	}

	// Determine output path from the import path.
	outputPath := filepath.Join(cfg.OutputDir, filepath.FromSlash(importPath), ".repo-metadata.json")

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("librariangen: error creating directory for %s: %w", outputPath, err)
	}

	jsonData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("librariangen: error marshalling data for API %s: %w", api.Path, err)
	}
	jsonData = append(jsonData, '\n')
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("librariangen: error writing file %s: %w", outputPath, err)
	}
	slog.Debug("librariangen: generated .repo-metadata.json", "path", outputPath)
	return nil
}

// Name is of form secretmanager.googleapis.com api_shortname
// should be prefix secretmanager.
func apiShortname(nameFull string) string {
	nameParts := strings.Split(nameFull, ".")
	return nameParts[0]
}

func docURL(modulePath, importPath string) (string, error) {
	pkgPath := strings.TrimPrefix(strings.TrimPrefix(importPath, modulePath), "/")
	return "https://cloud.google.com/go/docs/reference/" + modulePath + "/latest/" + pkgPath, nil
}
