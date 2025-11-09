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

// SidekickGeneral represents Sidekick TOML general configuration.
type SidekickGeneral struct {
	SpecificationFormat string `toml:"specification-format"`
	SpecificationSource string `toml:"specification-source"`
	ServiceConfig       string `toml:"service-config"`
	Language            string `toml:"language"`
}

type SidekickSource struct {
	DescriptionOverride string `toml:"description-override"`
	TitleOverride       string `toml:"title-override"`
	Roots               string `toml:"roots"`
	ProjectRoot         string `toml:"project-root"`
}

type SidekickCodec struct {
	Version                 string `toml:"version"`
	CopyrightYear           string `toml:"copyright-year"`
	PackageNameOverride     string `toml:"package-name-override"`
	ModulePath              string `toml:"module-path"`
	RootName                string `toml:"root-name"`
	TemplateOverride        string `toml:"template-override"`
	PerServiceFeatures      string `toml:"per-service-features"`
	DisabledRustdocWarnings string `toml:"disabled-rustdoc-warnings"`
	DisabledClippyWarnings  string `toml:"disabled-clippy-warnings"`
}

type DocumentationOverride struct {
	Id      string `toml:"id"`
	Match   string `toml:"match"`
	Replace string `toml:"replace"`
}

type DiscoveryPoller struct {
	Prefix   string `toml:"prefix"`
	MethodId string `toml:"method-id"`
}

type SidekickDiscovery struct {
	OperationId string            `toml:"operation-id"`
	Pollers     []DiscoveryPoller `toml:"pollers"`
}

type PaginationOverride struct {
	Id        string `toml:"id"`
	ItemField string `toml:"item-field"`
}

type Sidekick struct {
	General               SidekickGeneral         `toml:"general"`
	Source                SidekickSource          `toml:"source"`
	Codec                 SidekickCodec           `toml:"codec"`
	DocumentationOverride []DocumentationOverride `toml:"documentation-overrides"`
	Discovery             SidekickDiscovery       `toml:"discovery"`
	PaginationOverrides   []PaginationOverride    `toml:"pagination-overrides"`
	// Extra holds package dependencies like 'package:lro'
	Extra map[string]interface{} `toml:",inline"`
}

// LibrarianGenerate represents Librarian YAML generation configuration.
type LibrarianGenerate struct {
	SpecificationFormat string `yaml:"specification_format,omitempty"`
	APIs                []API  `yaml:"apis,omitempty"`
}

type API struct {
	Path          string `yaml:"path"`
	ServiceConfig string `yaml:"service_config,omitempty"`
}

type RustSource struct {
	DescriptionOverride string   `yaml:"description_override,omitempty"`
	TitleOverride       string   `yaml:"title_override,omitempty"`
	Roots               string   `yaml:"roots,omitempty"`
	ProjectRoot         string   `yaml:"project_root,omitempty"`
	IncludeList         []string `yaml:"include_list,omitempty"`
	IncludedIds         []string `yaml:"included_ids,omitempty"`
	SkippedIds          []string `yaml:"skipped_ids,omitempty"`
}

type RustCodec struct {
	CopyrightYear             string            `yaml:"copyright_year,omitempty"`
	PackageNameOverride       string            `yaml:"package_name_override,omitempty"`
	NameOverrides             map[string]string `yaml:"name_overrides,omitempty"`
	ModulePath                string            `yaml:"module_path,omitempty"`
	RootName                  string            `yaml:"root_name,omitempty"`
	TemplateOverride          string            `yaml:"template_override,omitempty"`
	NotForPublication         bool              `yaml:"not_for_publication,omitempty"`
	HasVeneer                 bool              `yaml:"has_veneer,omitempty"`
	PerServiceFeatures        bool              `yaml:"per_service_features,omitempty"`
	DefaultFeatures           []string          `yaml:"default_features,omitempty"`
	ExtraModules              []string          `yaml:"extra_modules,omitempty"`
	GenerateSetterSamples     bool              `yaml:"generate_setter_samples,omitempty"`
	DetailedTracingAttributes bool              `yaml:"detailed_tracing_attributes,omitempty"`
	IncludeGrpcOnlyMethods    bool              `yaml:"include_grpc_only_methods,omitempty"`
	DisabledRustdocWarnings   string            `yaml:"disabled_rustdoc_warnings,omitempty"`
	DisabledClippyWarnings    string            `yaml:"disabled_clippy_warnings,omitempty"`
	RoutingRequired           bool              `yaml:"routing_required,omitempty"`
	PostProcessProtos         bool              `yaml:"post_process_protos,omitempty"`
	Packages                  map[string]string `yaml:"packages,omitempty"`
}

type RustDocumentationOverride struct {
	Id      string `yaml:"id"`
	Match   string `yaml:"match"`
	Replace string `yaml:"replace"`
}

type RustDiscoveryPoller struct {
	Prefix   string `yaml:"prefix"`
	MethodId string `yaml:"method_id"`
}

type RustDiscovery struct {
	OperationId string                `yaml:"operation_id,omitempty"`
	Pollers     []RustDiscoveryPoller `yaml:"pollers,omitempty"`
}

type RustPaginationOverride struct {
	Id        string `yaml:"id"`
	ItemField string `yaml:"item_field"`
}

type Rust struct {
	Language              string                      `yaml:"language,omitempty"`
	Source                *RustSource                 `yaml:"source,omitempty"`
	Codec                 *RustCodec                  `yaml:"codec,omitempty"`
	DocumentationOverride []RustDocumentationOverride `yaml:"documentation_overrides,omitempty"`
	Discovery             *RustDiscovery              `yaml:"discovery,omitempty"`
	PaginationOverrides   []RustPaginationOverride    `yaml:"pagination_overrides,omitempty"`
}

type Librarian struct {
	Name     string            `yaml:"name"`
	Version  string            `yaml:"version"`
	Generate LibrarianGenerate `yaml:"generate"`
	Rust     *Rust             `yaml:"rust,omitempty"`
}

func parseNameOverrides(s string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}

func convertSidekick(sidekickPath string) (*Librarian, error) {
	data, err := os.ReadFile(sidekickPath)
	if err != nil {
		return nil, err
	}

	var sidekick Sidekick
	if err := toml.Unmarshal(data, &sidekick); err != nil {
		return nil, err
	}

	// Derive name from path
	dir := filepath.Dir(sidekickPath)
	relPath, _ := filepath.Rel("/Users/julieqiu/code/googleapis/google-cloud-rust", dir)
	name := strings.ReplaceAll(strings.TrimPrefix(relPath, "src/generated/"), "/", "-")

	librarian := &Librarian{
		Name:    name,
		Version: sidekick.Codec.Version,
		Generate: LibrarianGenerate{
			SpecificationFormat: sidekick.General.SpecificationFormat,
			APIs: []API{
				{
					Path:          sidekick.General.SpecificationSource,
					ServiceConfig: sidekick.General.ServiceConfig,
				},
			},
		},
	}

	// Default specification format
	if librarian.Generate.SpecificationFormat == "" {
		librarian.Generate.SpecificationFormat = "protobuf"
	}

	// Build rust section
	rust := &Rust{}
	hasRust := false

	// Language
	if sidekick.General.Language != "" && sidekick.General.Language != "rust" {
		rust.Language = sidekick.General.Language
		hasRust = true
	}

	// Source
	source := &RustSource{
		DescriptionOverride: sidekick.Source.DescriptionOverride,
		TitleOverride:       sidekick.Source.TitleOverride,
		Roots:               sidekick.Source.Roots,
		ProjectRoot:         sidekick.Source.ProjectRoot,
	}

	// Extract source fields from Extra map
	for key, value := range sidekick.Extra {
		switch key {
		case "include-list":
			if arrValue, ok := value.([]interface{}); ok {
				for _, item := range arrValue {
					if str, ok := item.(string); ok {
						source.IncludeList = append(source.IncludeList, str)
					}
				}
			}
		case "included-ids":
			if arrValue, ok := value.([]interface{}); ok {
				for _, item := range arrValue {
					if str, ok := item.(string); ok {
						source.IncludedIds = append(source.IncludedIds, str)
					}
				}
			}
		case "skipped-ids":
			if strValue, ok := value.(string); ok {
				source.SkippedIds = strings.Split(strValue, ",")
			} else if arrValue, ok := value.([]interface{}); ok {
				for _, item := range arrValue {
					if str, ok := item.(string); ok {
						source.SkippedIds = append(source.SkippedIds, str)
					}
				}
			}
		}
	}

	if source.DescriptionOverride != "" ||
		source.TitleOverride != "" ||
		source.Roots != "" ||
		source.ProjectRoot != "" ||
		len(source.IncludeList) > 0 ||
		len(source.IncludedIds) > 0 ||
		len(source.SkippedIds) > 0 {
		rust.Source = source
		hasRust = true
	}

	// Codec - extract from sidekick.Extra since fields are interface{}
	codec := &RustCodec{
		CopyrightYear:           sidekick.Codec.CopyrightYear,
		PackageNameOverride:     sidekick.Codec.PackageNameOverride,
		ModulePath:              sidekick.Codec.ModulePath,
		RootName:                sidekick.Codec.RootName,
		TemplateOverride:        sidekick.Codec.TemplateOverride,
		DisabledRustdocWarnings: sidekick.Codec.DisabledRustdocWarnings,
		DisabledClippyWarnings:  sidekick.Codec.DisabledClippyWarnings,
	}

	// Extract other codec fields from Extra map
	for key, value := range sidekick.Extra {
		switch key {
		case "name-overrides":
			if strValue, ok := value.(string); ok {
				codec.NameOverrides = parseNameOverrides(strValue)
			}
		case "not-for-publication":
			if boolValue, ok := value.(bool); ok {
				codec.NotForPublication = boolValue
			} else if strValue, ok := value.(string); ok {
				codec.NotForPublication = strValue == "true"
			}
		case "has-veneer":
			if boolValue, ok := value.(bool); ok {
				codec.HasVeneer = boolValue
			}
		case "default-features":
			if strValue, ok := value.(string); ok {
				codec.DefaultFeatures = strings.Split(strValue, ",")
			} else if arrValue, ok := value.([]interface{}); ok {
				for _, item := range arrValue {
					if str, ok := item.(string); ok {
						codec.DefaultFeatures = append(codec.DefaultFeatures, str)
					}
				}
			}
		case "extra-modules":
			if strValue, ok := value.(string); ok {
				codec.ExtraModules = strings.Split(strValue, ",")
			} else if arrValue, ok := value.([]interface{}); ok {
				for _, item := range arrValue {
					if str, ok := item.(string); ok {
						codec.ExtraModules = append(codec.ExtraModules, str)
					}
				}
			}
		case "generate-setter-samples":
			if boolValue, ok := value.(bool); ok {
				codec.GenerateSetterSamples = boolValue
			} else if strValue, ok := value.(string); ok {
				codec.GenerateSetterSamples = strValue == "true"
			}
		case "detailed-tracing-attributes":
			if boolValue, ok := value.(bool); ok {
				codec.DetailedTracingAttributes = boolValue
			}
		case "include-grpc-only-methods":
			if boolValue, ok := value.(bool); ok {
				codec.IncludeGrpcOnlyMethods = boolValue
			}
		case "routing-required":
			if boolValue, ok := value.(bool); ok {
				codec.RoutingRequired = boolValue
			}
		case "post-process-protos":
			if boolValue, ok := value.(bool); ok {
				codec.PostProcessProtos = boolValue
			}
		}
	}

	// Convert per-service-features string to bool
	if sidekick.Codec.PerServiceFeatures == "true" {
		codec.PerServiceFeatures = true
	}

	// Extract package dependencies
	packages := make(map[string]string)
	for key, value := range sidekick.Extra {
		if strings.HasPrefix(key, "package:") {
			pkgName := strings.TrimPrefix(key, "package:")
			if strValue, ok := value.(string); ok {
				packages[pkgName] = strValue
			}
		}
	}
	if len(packages) > 0 {
		codec.Packages = packages
	}

	if codec.CopyrightYear != "" || codec.PackageNameOverride != "" ||
		len(codec.NameOverrides) > 0 || codec.ModulePath != "" ||
		codec.RootName != "" || codec.TemplateOverride != "" ||
		codec.NotForPublication || codec.HasVeneer ||
		codec.PerServiceFeatures || len(codec.DefaultFeatures) > 0 ||
		len(codec.ExtraModules) > 0 || codec.GenerateSetterSamples ||
		codec.DetailedTracingAttributes || codec.IncludeGrpcOnlyMethods ||
		codec.DisabledRustdocWarnings != "" || codec.DisabledClippyWarnings != "" ||
		codec.RoutingRequired || codec.PostProcessProtos ||
		len(codec.Packages) > 0 {
		rust.Codec = codec
		hasRust = true
	}

	// Documentation overrides
	if len(sidekick.DocumentationOverride) > 0 {
		for _, override := range sidekick.DocumentationOverride {
			rust.DocumentationOverride = append(rust.DocumentationOverride, RustDocumentationOverride(override))
		}
		hasRust = true
	}

	// Discovery
	if sidekick.Discovery.OperationId != "" || len(sidekick.Discovery.Pollers) > 0 {
		discovery := &RustDiscovery{
			OperationId: sidekick.Discovery.OperationId,
		}
		for _, poller := range sidekick.Discovery.Pollers {
			discovery.Pollers = append(discovery.Pollers, RustDiscoveryPoller(poller))
		}
		rust.Discovery = discovery
		hasRust = true
	}

	// Pagination overrides
	if len(sidekick.PaginationOverrides) > 0 {
		for _, override := range sidekick.PaginationOverrides {
			rust.PaginationOverrides = append(rust.PaginationOverrides, RustPaginationOverride(override))
		}
		hasRust = true
	}

	if hasRust {
		librarian.Rust = rust
	}

	return librarian, nil
}

func main() {
	rustRepo := "/Users/julieqiu/code/googleapis/google-cloud-rust"
	testdataDir := "internal/container/rust/testdata"

	// Create testdata directory
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		log.Fatal(err)
	}

	// Find all .sidekick.toml files in src/generated
	var sidekickFiles []string
	err := filepath.Walk(filepath.Join(rustRepo, "src", "generated"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == ".sidekick.toml" {
			sidekickFiles = append(sidekickFiles, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d .sidekick.toml files\n", len(sidekickFiles))

	successCount := 0
	for _, sidekickPath := range sidekickFiles {
		librarian, err := convertSidekick(sidekickPath)
		if err != nil {
			fmt.Printf("✗ %s: %v\n", sidekickPath, err)
			continue
		}

		// Create output path in testdata (flatten src/generated)
		relPath, _ := filepath.Rel(rustRepo, filepath.Dir(sidekickPath))
		// Remove src/generated/ prefix
		relPath = strings.TrimPrefix(relPath, "src/generated/")
		outputDir := filepath.Join(testdataDir, relPath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("✗ %s: %v\n", sidekickPath, err)
			continue
		}

		outputPath := filepath.Join(outputDir, ".librarian.yaml")

		// Write YAML
		yamlData, err := yaml.Marshal(librarian)
		if err != nil {
			fmt.Printf("✗ %s: %v\n", sidekickPath, err)
			continue
		}

		if err := os.WriteFile(outputPath, yamlData, 0644); err != nil {
			fmt.Printf("✗ %s: %v\n", sidekickPath, err)
			continue
		}

		fmt.Printf("✓ %s\n", relPath)
		successCount++
	}

	fmt.Printf("\nGenerated %d .librarian.yaml files in %s\n", successCount, testdataDir)
}
