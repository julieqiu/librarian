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
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	testdataDir   = "internal/container/python/testdata"
	googleapisDir = "/Users/julieqiu/code/googleapis/googleapis"
)

type Librarian struct {
	Name     string            `yaml:"name"`
	Version  string            `yaml:"version"`
	Generate LibrarianGenerate `yaml:"generate"`
	Python   *Python           `yaml:"python,omitempty"`
}

type LibrarianGenerate struct {
	SpecificationFormat string `yaml:"specification_format,omitempty"`
	APIs                []API  `yaml:"apis,omitempty"`
}

type API struct {
	Path              string   `yaml:"path"`
	ServiceConfig     string   `yaml:"service_config,omitempty"`
	GRPCServiceConfig string   `yaml:"grpc_service_config,omitempty"`
	RestNumericEnums  *bool    `yaml:"rest_numeric_enums,omitempty"`
	Transport         string   `yaml:"transport,omitempty"`
	OptArgs           []string `yaml:"opt_args,omitempty"`
}

type PyGapicLibrary struct {
	GRPCServiceConfig string   `yaml:"grpc_service_config,omitempty"`
	RestNumericEnums  *bool    `yaml:"rest_numeric_enums,omitempty"`
	Transport         string   `yaml:"transport,omitempty"`
	OptArgs           []string `yaml:"opt_args,omitempty"`
}

type Python struct {
	Keep     []string       `yaml:"keep,omitempty"`
	Remove   []string       `yaml:"remove,omitempty"`
	Metadata PythonMetadata `yaml:"metadata,omitempty"`
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

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		return fmt.Errorf("failed to read testdata directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		librarianPath := filepath.Join(testdataDir, entry.Name(), ".librarian.yaml")
		if err := processLibrarian(librarianPath); err != nil {
			log.Printf("Error processing %s: %v", entry.Name(), err)
			continue
		}
		log.Printf("Processed %s", entry.Name())
	}

	return nil
}

func processLibrarian(librarianPath string) error {
	// Read the .librarian.yaml file
	data, err := os.ReadFile(librarianPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var librarian Librarian
	if err := yaml.Unmarshal(data, &librarian); err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// For each API, try to read BUILD.bazel and extract py_gapic_library data
	modified := false
	for i := range librarian.Generate.APIs {
		api := &librarian.Generate.APIs[i]
		buildPath := filepath.Join(googleapisDir, api.Path, "BUILD.bazel")

		pyGapicData, err := parseBuildBazel(buildPath)
		if err != nil {
			// If BUILD.bazel doesn't exist or can't be parsed, skip
			continue
		}

		// Add py_gapic_library data directly to the API
		if pyGapicData != nil {
			api.GRPCServiceConfig = pyGapicData.GRPCServiceConfig
			api.RestNumericEnums = pyGapicData.RestNumericEnums
			api.Transport = pyGapicData.Transport
			api.OptArgs = pyGapicData.OptArgs
			modified = true
		}
	}

	// Write back if modified
	if modified {
		out, err := yaml.Marshal(&librarian)
		if err != nil {
			return fmt.Errorf("failed to marshal yaml: %w", err)
		}

		if err := os.WriteFile(librarianPath, out, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}

// parseBuildBazel parses a BUILD.bazel file and extracts py_gapic_library configuration.
// For now, this is a simplified parser that looks for specific patterns.
// In production, we should use a proper Starlark parser.
func parseBuildBazel(buildPath string) (*PyGapicLibrary, error) {
	content, err := os.ReadFile(buildPath)
	if err != nil {
		return nil, err
	}

	text := string(content)

	// Look for py_gapic_library rule (not py_gapic_assembly_pkg)
	// The pattern is: any line containing "_py_gapic" but not "py_gapic_assembly_pkg"
	if !strings.Contains(text, "_py_gapic") {
		return nil, nil // No py_gapic_library found
	}

	// Check if this is actually a py_gapic_library (not just assembly_pkg)
	lines := strings.Split(text, "\n")
	inPyGapic := false
	var pyGapic PyGapicLibrary

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Start of py_gapic_library block
		if strings.HasPrefix(trimmed, "py_gapic_library(") ||
			(strings.Contains(trimmed, "_py_gapic") && strings.Contains(trimmed, "=") && !strings.Contains(trimmed, "py_gapic_assembly_pkg")) {
			inPyGapic = true
			continue
		}

		if !inPyGapic {
			continue
		}

		// End of rule
		if strings.HasPrefix(trimmed, ")") {
			break
		}

		// Parse attributes
		if strings.Contains(trimmed, "grpc_service_config") {
			val := extractValue(trimmed)
			pyGapic.GRPCServiceConfig = val
		} else if strings.Contains(trimmed, "rest_numeric_enums") {
			val := extractValue(trimmed)
			boolVal := val == "True"
			pyGapic.RestNumericEnums = &boolVal
		} else if strings.Contains(trimmed, "transport") && !strings.Contains(trimmed, "//") {
			val := extractValue(trimmed)
			pyGapic.Transport = val
		} else if strings.Contains(trimmed, "opt_args") {
			// Check if opt_args is on a single line: opt_args = ["foo", "bar"]
			if strings.Contains(trimmed, "[") && strings.Contains(trimmed, "]") {
				// Single-line case
				start := strings.Index(trimmed, "[")
				end := strings.Index(trimmed, "]")
				listContent := trimmed[start+1 : end]
				// Split by comma and extract values
				parts := strings.Split(listContent, ",")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if part != "" {
						arg := strings.Trim(part, "\"'")
						pyGapic.OptArgs = append(pyGapic.OptArgs, arg)
					}
				}
			} else {
				// Multi-line case: look ahead for list items
				args := []string{}
				for j := i + 1; j < len(lines); j++ {
					argLine := strings.TrimSpace(lines[j])
					if strings.HasPrefix(argLine, "]") {
						break
					}
					if strings.HasPrefix(argLine, "\"") || strings.HasPrefix(argLine, "'") {
						arg := extractValue(argLine)
						args = append(args, arg)
					}
				}
				pyGapic.OptArgs = args
			}
		}
	}

	if pyGapic.GRPCServiceConfig == "" && pyGapic.RestNumericEnums == nil && pyGapic.Transport == "" && len(pyGapic.OptArgs) == 0 {
		return nil, nil // Found py_gapic but no attributes
	}

	return &pyGapic, nil
}

// extractValue extracts the string value from a line like 'key = "value",' or 'key = value,'.
func extractValue(line string) string {
	// Remove leading/trailing whitespace
	line = strings.TrimSpace(line)

	// Split by '='
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		// Handle simple quoted string case
		line = strings.Trim(line, ",")
		line = strings.Trim(line, "\"'")
		return line
	}

	val := strings.TrimSpace(parts[1])
	// Remove trailing comma
	val = strings.TrimSuffix(val, ",")
	// Remove quotes
	val = strings.Trim(val, "\"'")

	return val
}
