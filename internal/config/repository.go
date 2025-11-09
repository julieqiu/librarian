// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LibrarianConfig represents the repository-level configuration stored in .librarian.yaml
// at the repository root.
type LibrarianConfig struct {
	Librarian LibrarianSection `yaml:"librarian"`
	Generate  *GenerateSection `yaml:"generate,omitempty"`
	Release   *ReleaseSection  `yaml:"release,omitempty"`
}

// LibrarianSection contains metadata about the librarian version and language.
type LibrarianSection struct {
	Version  string `yaml:"version"`
	Language string `yaml:"language,omitempty"`
}

// GenerateSection contains configuration for code generation.
type GenerateSection struct {
	Container  ContainerConfig `yaml:"container"`
	Googleapis RepositoryRef   `yaml:"googleapis"`
	Discovery  *RepositoryRef  `yaml:"discovery,omitempty"`
	Dir        string          `yaml:"dir,omitempty"`
}

// ContainerConfig specifies the container image for code generation.
type ContainerConfig struct {
	Image string `yaml:"image"`
	Tag   string `yaml:"tag"`
}

// RepositoryRef specifies a repository location and git reference.
type RepositoryRef struct {
	Repo string `yaml:"repo"`
	Ref  string `yaml:"ref,omitempty"`
}

// ReleaseSection contains configuration for releases.
type ReleaseSection struct {
	TagFormat string `yaml:"tag_format"`
}

// ReadLibrarianConfig reads the .librarian.yaml file from the repository root.
func ReadLibrarianConfig(repoRoot string) (*LibrarianConfig, error) {
	configPath := filepath.Join(repoRoot, ".librarian.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .librarian.yaml: %w", err)
	}

	var config LibrarianConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse .librarian.yaml: %w", err)
	}

	return &config, nil
}

// WriteLibrarianConfig writes the LibrarianConfig to .librarian.yaml in the repository root.
func WriteLibrarianConfig(repoRoot string, config *LibrarianConfig) error {
	configPath := filepath.Join(repoRoot, ".librarian.yaml")
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal .librarian.yaml: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write .librarian.yaml: %w", err)
	}

	return nil
}

// Validate checks that the LibrarianConfig is valid.
func (c *LibrarianConfig) Validate() error {
	if c.Librarian.Version == "" {
		return fmt.Errorf("librarian.version is required")
	}

	if c.Generate != nil {
		if c.Generate.Container.Image == "" {
			return fmt.Errorf("generate.container.image is required when generate section is present")
		}
		if c.Generate.Container.Tag == "" {
			return fmt.Errorf("generate.container.tag is required when generate section is present")
		}
		if c.Generate.Googleapis.Repo == "" {
			return fmt.Errorf("generate.googleapis.repo is required when generate section is present")
		}
	}

	if c.Release != nil {
		if c.Release.TagFormat == "" {
			return fmt.Errorf("release.tag_format is required when release section is present")
		}
	}

	return nil
}

// HasGenerate returns true if the repository supports code generation.
func (c *LibrarianConfig) HasGenerate() bool {
	return c.Generate != nil
}

// HasRelease returns true if the repository supports releases.
func (c *LibrarianConfig) HasRelease() bool {
	return c.Release != nil
}

// ArtifactState represents the per-artifact state stored in <path>/.librarian.yaml.
type ArtifactState struct {
	Generate *ArtifactGenerateState `yaml:"generate,omitempty"`
	Release  *ArtifactReleaseState  `yaml:"release,omitempty"`
}

// ArtifactGenerateState contains generation configuration and history for an artifact.
type ArtifactGenerateState struct {
	APIs       []APIConfig       `yaml:"apis,omitempty"`
	Commit     string            `yaml:"commit,omitempty"`
	Librarian  string            `yaml:"librarian,omitempty"`
	Container  ContainerConfig   `yaml:"container,omitempty"`
	Googleapis RepositoryRef     `yaml:"googleapis,omitempty"`
	Discovery  *RepositoryRef    `yaml:"discovery,omitempty"`
	Metadata   *ArtifactMetadata `yaml:"metadata,omitempty"`
	Language   map[string]string `yaml:"language,omitempty"`
	Keep       []string          `yaml:"keep,omitempty"`
	Remove     []string          `yaml:"remove,omitempty"`
	Exclude    []string          `yaml:"exclude,omitempty"`
}

// APIConfig represents configuration for a single API version.
type APIConfig struct {
	Path              string   `yaml:"path"`
	GRPCServiceConfig string   `yaml:"grpc_service_config,omitempty"`
	ServiceYAML       string   `yaml:"service_yaml,omitempty"`
	Transport         string   `yaml:"transport,omitempty"`
	RestNumericEnums  bool     `yaml:"rest_numeric_enums,omitempty"`
	OptArgs           []string `yaml:"opt_args,omitempty"`
}

// ArtifactMetadata contains library metadata used for documentation and packaging.
type ArtifactMetadata struct {
	NamePretty           string `yaml:"name_pretty,omitempty"`
	ProductDocumentation string `yaml:"product_documentation,omitempty"`
	ClientDocumentation  string `yaml:"client_documentation,omitempty"`
	IssueTracker         string `yaml:"issue_tracker,omitempty"`
	ReleaseLevel         string `yaml:"release_level,omitempty"`
	LibraryType          string `yaml:"library_type,omitempty"`
	APIID                string `yaml:"api_id,omitempty"`
	APIShortname         string `yaml:"api_shortname,omitempty"`
	APIDescription       string `yaml:"api_description,omitempty"`
	DefaultVersion       string `yaml:"default_version,omitempty"`
}

// ArtifactReleaseState contains release state for an artifact.
type ArtifactReleaseState struct {
	Version  *string          `yaml:"version"`
	Prepared *PreparedRelease `yaml:"prepared,omitempty"`
}

// PreparedRelease represents a prepared but not yet published release.
type PreparedRelease struct {
	Version string `yaml:"version"`
	Commit  string `yaml:"commit"`
}

// ReadArtifactState reads the .librarian.yaml file from an artifact directory.
func ReadArtifactState(artifactPath string) (*ArtifactState, error) {
	configPath := filepath.Join(artifactPath, ".librarian.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .librarian.yaml: %w", err)
	}

	var state ArtifactState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse .librarian.yaml: %w", err)
	}

	return &state, nil
}

// WriteArtifactState writes the ArtifactState to .librarian.yaml in the artifact directory.
func WriteArtifactState(artifactPath string, state *ArtifactState) error {
	configPath := filepath.Join(artifactPath, ".librarian.yaml")
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal .librarian.yaml: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write .librarian.yaml: %w", err)
	}

	return nil
}

// HasGenerate returns true if the artifact supports code generation.
func (a *ArtifactState) HasGenerate() bool {
	return a.Generate != nil
}

// HasRelease returns true if the artifact supports releases.
func (a *ArtifactState) HasRelease() bool {
	return a.Release != nil
}
