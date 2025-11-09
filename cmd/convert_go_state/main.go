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
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// StateYAML represents the structure of state.yaml.
type StateYAML struct {
	Image     string    `yaml:"image"`
	Libraries []Library `yaml:"libraries"`
}

// Library represents a library entry in state.yaml.
type Library struct {
	ID                  string   `yaml:"id"`
	Version             string   `yaml:"version"`
	APIs                []API    `yaml:"apis"`
	RemoveRegex         []string `yaml:"remove_regex"`
	ReleaseExcludePaths []string `yaml:"release_exclude_paths"`
}

// API represents an API configuration.
type API struct {
	Path          string `yaml:"path"`
	ServiceConfig string `yaml:"service_config"`
}

// LibrarianYAML represents the structure of .librarian.yaml.
type LibrarianYAML struct {
	Generate Generate `yaml:"generate"`
	Metadata Metadata `yaml:"metadata"`
	Go       GoConfig `yaml:"go"`
}

// Generate represents the generate section.
type Generate struct {
	SpecificationFormat string `yaml:"specification_format"`
	SpecificationSource string `yaml:"specification_source"`
	ServiceConfig       string `yaml:"service_config"`
}

// Metadata represents the metadata section.
type Metadata struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// GoConfig represents the go section.
type GoConfig struct {
	APIs    []API    `yaml:"apis,omitempty"`
	Remove  []string `yaml:"remove,omitempty"`
	Exclude []string `yaml:"exclude,omitempty"`
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: go run main.go <state.yaml> <output-dir>")
	}

	stateFile := os.Args[1]
	outputDir := os.Args[2]

	// Read state.yaml
	data, err := os.ReadFile(stateFile)
	if err != nil {
		log.Fatalf("Failed to read state.yaml: %v", err)
	}

	var state StateYAML
	if err := yaml.Unmarshal(data, &state); err != nil {
		log.Fatalf("Failed to parse state.yaml: %v", err)
	}

	fmt.Printf("Found %d libraries\n", len(state.Libraries))

	// Process each library
	successCount := 0
	for i, lib := range state.Libraries {
		if err := processLibrary(lib, outputDir); err != nil {
			log.Printf("Failed to process library %s: %v", lib.ID, err)
			continue
		}
		successCount++
		if (i+1)%10 == 0 {
			fmt.Printf("Processed %d/%d libraries...\n", i+1, len(state.Libraries))
		}
	}

	fmt.Printf("\nSuccessfully processed %d/%d libraries\n", successCount, len(state.Libraries))
}

func processLibrary(lib Library, outputDir string) error {
	// Skip libraries with no APIs
	if len(lib.APIs) == 0 {
		return fmt.Errorf("no APIs defined")
	}

	// Create the directory structure
	libDir := filepath.Join(outputDir, lib.ID)
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Build the .librarian.yaml structure
	libYAML := LibrarianYAML{
		Generate: Generate{
			SpecificationFormat: "protobuf",
			SpecificationSource: lib.APIs[0].Path,
			ServiceConfig:       lib.APIs[0].ServiceConfig,
		},
		Metadata: Metadata{
			Name:    lib.ID,
			Version: lib.Version,
		},
		Go: GoConfig{
			APIs:    lib.APIs,
			Remove:  lib.RemoveRegex,
			Exclude: lib.ReleaseExcludePaths,
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(&libYAML)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write to file
	outputFile := filepath.Join(libDir, ".librarian.yaml")
	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
