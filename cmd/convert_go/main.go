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

const (
	stateFile      = "/Users/julieqiu/code/googleapis/google-cloud-go/.librarian/state.yaml"
	repoConfigFile = "/Users/julieqiu/code/googleapis/google-cloud-go/.librarian/generator-input/repo-config.yaml"
	outputDir      = "internal/container/go/testdata"
	googleapisDir  = "/Users/julieqiu/code/googleapis/googleapis"
)

type State struct {
	Image     string      `yaml:"image"`
	Libraries []GoLibrary `yaml:"libraries"`
}

type GoLibrary struct {
	ID                  string   `yaml:"id"`
	Version             string   `yaml:"version"`
	LastGeneratedCommit string   `yaml:"last_generated_commit"`
	APIs                []API    `yaml:"apis"`
	SourceRoots         []string `yaml:"source_roots"`
	PreserveRegex       []string `yaml:"preserve_regex"`
	RemoveRegex         []string `yaml:"remove_regex"`
	ReleaseExcludePaths []string `yaml:"release_exclude_paths"`
	TagFormat           string   `yaml:"tag_format"`
}

type API struct {
	Path          string `yaml:"path"`
	ServiceConfig string `yaml:"service_config"`
}

type Librarian struct {
	Name     string            `yaml:"name"`
	Version  string            `yaml:"version"`
	Generate LibrarianGenerate `yaml:"generate"`
	Go       *GoConfig         `yaml:"go,omitempty"`
}

type LibrarianGenerate struct {
	SpecificationFormat string         `yaml:"specification_format,omitempty"`
	APIs                []LibrarianAPI `yaml:"apis,omitempty"`
}

type LibrarianAPI struct {
	Path            string   `yaml:"path"`
	ServiceConfig   string   `yaml:"service_config,omitempty"`
	ClientDirectory string   `yaml:"client_directory,omitempty"`
	DisableGapic    bool     `yaml:"disable_gapic,omitempty"`
	NestedProtos    []string `yaml:"nested_protos,omitempty"`
	ProtoPackage    string   `yaml:"proto_package,omitempty"`
}

type GoConfig struct {
	SourceRoots                 []string `yaml:"source_roots,omitempty"`
	PreserveRegex               []string `yaml:"preserve_regex,omitempty"`
	RemoveRegex                 []string `yaml:"remove_regex,omitempty"`
	ReleaseExcludePaths         []string `yaml:"release_exclude_paths,omitempty"`
	TagFormat                   string   `yaml:"tag_format,omitempty"`
	ModulePathVersion           string   `yaml:"module_path_version,omitempty"`
	DeleteGenerationOutputPaths []string `yaml:"delete_generation_output_paths,omitempty"`
}

type RepoConfig struct {
	Modules []RepoModule `yaml:"modules"`
}

type RepoModule struct {
	Name                        string    `yaml:"name"`
	APIs                        []RepoAPI `yaml:"apis"`
	ModulePathVersion           string    `yaml:"module_path_version,omitempty"`
	DeleteGenerationOutputPaths []string  `yaml:"delete_generation_output_paths,omitempty"`
}

type RepoAPI struct {
	Path            string   `yaml:"path"`
	ClientDirectory string   `yaml:"client_directory,omitempty"`
	DisableGapic    bool     `yaml:"disable_gapic,omitempty"`
	NestedProtos    []string `yaml:"nested_protos,omitempty"`
	ProtoPackage    string   `yaml:"proto_package,omitempty"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Read state.yaml
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := yaml.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Read repo-config.yaml
	var repoConfig RepoConfig
	repoData, err := os.ReadFile(repoConfigFile)
	if err != nil {
		log.Printf("Warning: failed to read repo config file: %v", err)
	} else {
		if err := yaml.Unmarshal(repoData, &repoConfig); err != nil {
			return fmt.Errorf("failed to unmarshal repo config: %w", err)
		}
	}

	// Create a map of repo config by module name for easy lookup
	repoConfigMap := make(map[string]*RepoModule)
	for i := range repoConfig.Modules {
		repoConfigMap[repoConfig.Modules[i].Name] = &repoConfig.Modules[i]
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Convert each library
	for _, lib := range state.Libraries {
		// Merge with repo config if available
		repoModule := repoConfigMap[lib.ID]
		librarian := convertGoLibrary(lib, repoModule)

		// Create directory for this library
		libDir := filepath.Join(outputDir, lib.ID)
		if err := os.MkdirAll(libDir, 0755); err != nil {
			log.Printf("Failed to create directory for %s: %v", lib.ID, err)
			continue
		}

		// Write .librarian.yaml
		out, err := yaml.Marshal(&librarian)
		if err != nil {
			log.Printf("Failed to marshal %s: %v", lib.ID, err)
			continue
		}

		outPath := filepath.Join(libDir, ".librarian.yaml")
		if err := os.WriteFile(outPath, out, 0644); err != nil {
			log.Printf("Failed to write %s: %v", lib.ID, err)
			continue
		}

		log.Printf("Converted %s", lib.ID)
	}

	log.Printf("Converted %d libraries", len(state.Libraries))
	return nil
}

func convertGoLibrary(lib GoLibrary, repoModule *RepoModule) *Librarian {
	librarian := &Librarian{
		Name:    lib.ID,
		Version: lib.Version,
		Generate: LibrarianGenerate{
			SpecificationFormat: "protobuf",
		},
	}

	// Create a map of repo API config by path for easy lookup
	repoAPIMap := make(map[string]*RepoAPI)
	if repoModule != nil {
		for i := range repoModule.APIs {
			repoAPIMap[repoModule.APIs[i].Path] = &repoModule.APIs[i]
		}
	}

	// Convert APIs
	if len(lib.APIs) > 0 {
		for _, api := range lib.APIs {
			libAPI := LibrarianAPI{
				Path:          api.Path,
				ServiceConfig: api.ServiceConfig,
			}

			// Merge with repo config if available
			if repoAPI, ok := repoAPIMap[api.Path]; ok {
				libAPI.ClientDirectory = repoAPI.ClientDirectory
				libAPI.DisableGapic = repoAPI.DisableGapic
				libAPI.NestedProtos = repoAPI.NestedProtos
				libAPI.ProtoPackage = repoAPI.ProtoPackage
			}

			librarian.Generate.APIs = append(librarian.Generate.APIs, libAPI)
		}
	}

	// Go configuration
	go_config := &GoConfig{
		SourceRoots:         lib.SourceRoots,
		PreserveRegex:       lib.PreserveRegex,
		RemoveRegex:         lib.RemoveRegex,
		ReleaseExcludePaths: lib.ReleaseExcludePaths,
		TagFormat:           lib.TagFormat,
	}

	// Add repo-level config
	if repoModule != nil {
		go_config.ModulePathVersion = repoModule.ModulePathVersion
		go_config.DeleteGenerationOutputPaths = repoModule.DeleteGenerationOutputPaths
	}

	librarian.Go = go_config

	return librarian
}
