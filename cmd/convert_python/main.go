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

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Python state.yaml structures
type PythonAPI struct {
	Path              string `yaml:"path"`
	ServiceConfig     string `yaml:"service_config"`
	GRPCServiceConfig string `yaml:"grpc_service_config"`
}

type PythonLibrary struct {
	ID                    string      `yaml:"id"`
	Version               string      `yaml:"version"`
	LastGeneratedCommit   string      `yaml:"last_generated_commit"`
	APIs                  []PythonAPI `yaml:"apis"`
	SourceRoots           []string    `yaml:"source_roots"`
	PreserveRegex         []string    `yaml:"preserve_regex"`
	RemoveRegex           []string    `yaml:"remove_regex"`
	ReleaseExcludePaths   []string    `yaml:"release_exclude_paths"`
	TagFormat             string      `yaml:"tag_format"`
}

type PythonState struct {
	Image     string          `yaml:"image"`
	Libraries []PythonLibrary `yaml:"libraries"`
}

// Python .repo-metadata.json structure
type RepoMetadata struct {
	Name                  string `json:"name"`
	NamePretty            string `json:"name_pretty"`
	ProductDocumentation  string `json:"product_documentation"`
	ClientDocumentation   string `json:"client_documentation"`
	IssueTracker          string `json:"issue_tracker"`
	ReleaseLevel          string `json:"release_level"`
	Language              string `json:"language"`
	LibraryType           string `json:"library_type"`
	Repo                  string `json:"repo"`
	DistributionName      string `json:"distribution_name"`
	APIID                 string `json:"api_id"`
	APIShortname          string `json:"api_shortname"`
	DefaultVersion        string `json:"default_version"`
	APIDescription        string `json:"api_description"`
}

// Librarian YAML structures
type LibrarianGenerate struct {
	SpecificationFormat string `yaml:"specification_format,omitempty"`
	APIs                []API  `yaml:"apis,omitempty"`
}

type API struct {
	Path          string `yaml:"path"`
	ServiceConfig string `yaml:"service_config"`
}

type PythonConfig struct {
	Keep     []string        `yaml:"keep,omitempty"`
	Remove   []string        `yaml:"remove,omitempty"`
	Exclude  []string        `yaml:"exclude,omitempty"`
	Metadata PythonMetadata  `yaml:"metadata,omitempty"`
}

type PythonMetadata struct {
	NamePretty           string `yaml:"name_pretty,omitempty"`
	Description          string `yaml:"description,omitempty"`
	DistributionName     string `yaml:"distribution_name,omitempty"`
	APIID                string `yaml:"api_id,omitempty"`
	DefaultVersion       string `yaml:"default_version,omitempty"`
	ProductDocumentation string `yaml:"product_documentation,omitempty"`
	ClientDocumentation  string `yaml:"client_documentation,omitempty"`
	IssueTracker         string `yaml:"issue_tracker,omitempty"`
	ReleaseLevel         string `yaml:"release_level,omitempty"`
	LibraryType          string `yaml:"library_type,omitempty"`
	Repo                 string `yaml:"repo,omitempty"`
}

type Librarian struct {
	Name     string            `yaml:"name"`
	Version  string            `yaml:"version"`
	Generate LibrarianGenerate `yaml:"generate"`
	Python   *PythonConfig     `yaml:"python,omitempty"`
}

func convertPythonLibrary(lib PythonLibrary, metadataPath string) (*Librarian, error) {
	librarian := &Librarian{
		Name:    lib.ID,
		Version: lib.Version,
		Generate: LibrarianGenerate{
			SpecificationFormat: "protobuf",
		},
	}

	// Convert APIs
	if len(lib.APIs) > 0 {
		for _, api := range lib.APIs {
			librarian.Generate.APIs = append(librarian.Generate.APIs, API{
				Path:          api.Path,
				ServiceConfig: api.ServiceConfig,
			})
		}
	}

	// Python configuration
	python := &PythonConfig{
		Keep:   lib.PreserveRegex,
		Remove: lib.RemoveRegex,
	}

	// Read .repo-metadata.json if it exists
	if metadataPath != "" {
		data, err := os.ReadFile(metadataPath)
		if err == nil {
			var metadata RepoMetadata
			if err := json.Unmarshal(data, &metadata); err == nil {
				python.Metadata = PythonMetadata{
					NamePretty:           metadata.NamePretty,
					Description:          metadata.APIDescription,
					DistributionName:     metadata.DistributionName,
					APIID:                metadata.APIID,
					DefaultVersion:       metadata.DefaultVersion,
					ProductDocumentation: metadata.ProductDocumentation,
					ClientDocumentation:  metadata.ClientDocumentation,
					IssueTracker:         metadata.IssueTracker,
					ReleaseLevel:         metadata.ReleaseLevel,
					LibraryType:          metadata.LibraryType,
					Repo:                 metadata.Repo,
				}
			}
		}
	}

	librarian.Python = python

	return librarian, nil
}

func main() {
	pythonRepo := "/Users/julieqiu/code/googleapis/google-cloud-python"
	testdataDir := "internal/container/python/testdata"

	// Create testdata directory
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		log.Fatal(err)
	}

	// Read state.yaml
	stateData, err := os.ReadFile(filepath.Join(pythonRepo, ".librarian", "state.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	var state PythonState
	if err := yaml.Unmarshal(stateData, &state); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d libraries in state.yaml\n", len(state.Libraries))

	successCount := 0
	for _, lib := range state.Libraries {
		// Find .repo-metadata.json
		metadataPath := filepath.Join(pythonRepo, ".librarian", "generator-input", "packages", lib.ID, ".repo-metadata.json")
		if _, err := os.Stat(metadataPath); err != nil {
			metadataPath = "" // Doesn't exist
		}

		librarian, err := convertPythonLibrary(lib, metadataPath)
		if err != nil {
			fmt.Printf("✗ %s: %v\n", lib.ID, err)
			continue
		}

		// Create output directory
		outputDir := filepath.Join(testdataDir, lib.ID)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("✗ %s: %v\n", lib.ID, err)
			continue
		}

		outputPath := filepath.Join(outputDir, ".librarian.yaml")

		// Write YAML
		yamlData, err := yaml.Marshal(librarian)
		if err != nil {
			fmt.Printf("✗ %s: %v\n", lib.ID, err)
			continue
		}

		if err := os.WriteFile(outputPath, yamlData, 0644); err != nil {
			fmt.Printf("✗ %s: %v\n", lib.ID, err)
			continue
		}

		fmt.Printf("✓ %s\n", lib.ID)
		successCount++
	}

	fmt.Printf("\nGenerated %d .librarian.yaml files in %s\n", successCount, testdataDir)
}
