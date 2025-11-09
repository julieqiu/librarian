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

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

const (
	dartRepoDir = "/Users/julieqiu/code/googleapis/google-cloud-dart"
	testdataDir = "internal/container/dart/testdata"
)

// Sidekick TOML structures
type SidekickGeneral struct {
	SpecificationFormat string `toml:"specification-format"`
	SpecificationSource string `toml:"specification-source"`
	ServiceConfig       string `toml:"service-config"`
}

type SidekickSource struct {
	DescriptionOverride string `toml:"description-override"`
	TitleOverride       string `toml:"title-override"`
}

type SidekickCodec struct {
	CopyrightYear                 string `toml:"copyright-year"`
	RepositoryURL                 string `toml:"repository-url"`
	ApiKeysEnvironmentVariables   string `toml:"api-keys-environment-variables"`
	DevDependencies               string `toml:"dev-dependencies"`
	ReadmeAfterTitleText          string `toml:"readme-after-title-text"`
	ReadmeQuickstartText          string `toml:"readme-quickstart-text"`
	ReadmeCustomServiceExplanation string `toml:"readme-custom-service-explanation"`
}

type Sidekick struct {
	General SidekickGeneral `toml:"general"`
	Source  SidekickSource  `toml:"source"`
	Codec   SidekickCodec   `toml:"codec"`
}

// Librarian YAML structures
type LibrarianGenerate struct {
	SpecificationFormat string `yaml:"specification_format,omitempty"`
	APIs                []API  `yaml:"apis,omitempty"`
}

type API struct {
	Path          string `yaml:"path"`
	ServiceConfig string `yaml:"service_config,omitempty"`
}

type DartSource struct {
	DescriptionOverride string `yaml:"description_override,omitempty"`
	TitleOverride       string `yaml:"title_override,omitempty"`
}

type DartCodec struct {
	CopyrightYear                  string `yaml:"copyright_year,omitempty"`
	RepositoryURL                  string `yaml:"repository_url,omitempty"`
	ApiKeysEnvironmentVariables    string `yaml:"api_keys_environment_variables,omitempty"`
	DevDependencies                string `yaml:"dev_dependencies,omitempty"`
	ReadmeAfterTitleText           string `yaml:"readme_after_title_text,omitempty"`
	ReadmeQuickstartText           string `yaml:"readme_quickstart_text,omitempty"`
	ReadmeCustomServiceExplanation string `yaml:"readme_custom_service_explanation,omitempty"`
}

type DartConfig struct {
	Source *DartSource `yaml:"source,omitempty"`
	Codec  *DartCodec  `yaml:"codec,omitempty"`
}

type Librarian struct {
	Name     string             `yaml:"name"`
	Version  string             `yaml:"version"`
	Generate *LibrarianGenerate `yaml:"generate,omitempty"`
	Dart     *DartConfig        `yaml:"dart,omitempty"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Find all .sidekick.toml files in google-cloud-dart/generated
	generatedDir := filepath.Join(dartRepoDir, "generated")
	entries, err := os.ReadDir(generatedDir)
	if err != nil {
		return fmt.Errorf("failed to read generated directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sidekickPath := filepath.Join(generatedDir, entry.Name(), ".sidekick.toml")
		if _, err := os.Stat(sidekickPath); os.IsNotExist(err) {
			continue
		}

		librarian, err := convertSidekick(sidekickPath, entry.Name())
		if err != nil {
			log.Printf("Error converting %s: %v", entry.Name(), err)
			continue
		}

		// Create testdata directory for this package
		packageDir := filepath.Join(testdataDir, entry.Name())
		if err := os.MkdirAll(packageDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", packageDir, err)
		}

		// Write .librarian.yaml
		librarianPath := filepath.Join(packageDir, ".librarian.yaml")
		if err := writeLibrarian(librarianPath, librarian); err != nil {
			return fmt.Errorf("failed to write %s: %w", librarianPath, err)
		}

		log.Printf("Converted %s", entry.Name())
	}

	return nil
}

func convertSidekick(sidekickPath, packageName string) (*Librarian, error) {
	data, err := os.ReadFile(sidekickPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var sidekick Sidekick
	if err := toml.Unmarshal(data, &sidekick); err != nil {
		return nil, fmt.Errorf("failed to unmarshal toml: %w", err)
	}

	librarian := &Librarian{
		Name:    packageName,
		Version: "0.0.0", // Default version, should be updated
	}

	// Convert general section to generate.apis
	if sidekick.General.SpecificationSource != "" {
		librarian.Generate = &LibrarianGenerate{
			SpecificationFormat: getSpecificationFormat(sidekick.General.SpecificationFormat),
			APIs: []API{
				{
					Path:          sidekick.General.SpecificationSource,
					ServiceConfig: sidekick.General.ServiceConfig,
				},
			},
		}
	}

	// Convert dart-specific sections
	librarian.Dart = &DartConfig{}

	// Convert source
	if sidekick.Source.DescriptionOverride != "" || sidekick.Source.TitleOverride != "" {
		librarian.Dart.Source = &DartSource{
			DescriptionOverride: sidekick.Source.DescriptionOverride,
			TitleOverride:       sidekick.Source.TitleOverride,
		}
	}

	// Convert codec
	librarian.Dart.Codec = &DartCodec{
		CopyrightYear:                  sidekick.Codec.CopyrightYear,
		RepositoryURL:                  sidekick.Codec.RepositoryURL,
		ApiKeysEnvironmentVariables:    sidekick.Codec.ApiKeysEnvironmentVariables,
		DevDependencies:                sidekick.Codec.DevDependencies,
		ReadmeAfterTitleText:           sidekick.Codec.ReadmeAfterTitleText,
		ReadmeQuickstartText:           sidekick.Codec.ReadmeQuickstartText,
		ReadmeCustomServiceExplanation: sidekick.Codec.ReadmeCustomServiceExplanation,
	}

	// Clean up empty dart section
	if librarian.Dart.Source == nil && librarian.Dart.Codec != nil {
		if librarian.Dart.Codec.CopyrightYear == "" &&
			librarian.Dart.Codec.RepositoryURL == "" &&
			librarian.Dart.Codec.ApiKeysEnvironmentVariables == "" &&
			librarian.Dart.Codec.DevDependencies == "" &&
			librarian.Dart.Codec.ReadmeAfterTitleText == "" &&
			librarian.Dart.Codec.ReadmeQuickstartText == "" &&
			librarian.Dart.Codec.ReadmeCustomServiceExplanation == "" {
			librarian.Dart = nil
		}
	}

	return librarian, nil
}

func getSpecificationFormat(source string) string {
	// Dart uses protobuf for all current packages
	if strings.HasPrefix(source, "google/") {
		return "protobuf"
	}
	return ""
}

func writeLibrarian(path string, librarian *Librarian) error {
	out, err := yaml.Marshal(librarian)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml: %w", err)
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
