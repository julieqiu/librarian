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

package config

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Config represents a librarian.yaml configuration file.
type Config struct {
	// Language is one of "go", "python", or "rust".
	Language string `yaml:"language"`

	// Repo is the repository path, such as "googleapis/google-cloud-python".
	Repo string `yaml:"repo,omitempty"`

	// Sources references external source repositories.
	Sources *Sources `yaml:"sources,omitempty"`

	// Default contains default settings for all libraries.
	Default *Default `yaml:"defaults,omitempty"`

	// Libraries contains library configurations.
	Libraries []*Library `yaml:"libraries,omitempty"`
}

// Sources references external source repositories.
type Sources struct {
	// Discovery is the discovery-artifact-manager repository configuration.
	Discovery *Source `yaml:"discovery,omitempty"`

	// Googleapis is the googleapis repository configuration.
	Googleapis *Source `yaml:"googleapis,omitempty"`
}

// Source represents a source repository.
type Source struct {
	// Commit is the git commit hash or tag to use.
	Commit string `yaml:"commit"`

	// SHA256 is the expected hash of the tarball for this commit.
	SHA256 string `yaml:"sha256,omitempty"`

	// Dir is a local directory path to use instead of fetching. If set, Commit
	// and SHA256 are ignored.
	Dir string `yaml:"dir,omitempty"`
}

// Default contains default settings for all libraries.
type Default struct {
	// Output is the directory where generated code is written.
	Output string `yaml:"output,omitempty"`

	// Transport is the transport protocol, such as "grpc+rest" or "grpc".
	Transport string `yaml:"transport,omitempty"`

	// ReleaseLevel is either "stable" or "preview".
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// TagFormat is the template for git tags, such as "{name}/v{version}".
	TagFormat string `yaml:"tag_format,omitempty"`

	// Remote is the git remote name, such as "upstream" or "origin".
	Remote string `yaml:"remote,omitempty"`

	// Branch is the release branch, such as "main" or "master".
	Branch string `yaml:"branch,omitempty"`

	// Rust contains Rust-specific default configuration.
	Rust *RustDefault `yaml:"rust,omitempty"`
}

// Library represents a library configuration.
type Library struct {
	// Name is the library name, such as "secretmanager" or "storage".
	Name string `yaml:"name"`

	// APIs lists the APIs to include in this library.
	APIs []*API `yaml:"apis,omitempty"`

	// Version is the library version.
	Version string `yaml:"version,omitempty"`

	// SkipGenerate disables code generation for this library.
	SkipGenerate bool `yaml:"skip_generate,omitempty"`

	// SkipRelease disables releasing for this library.
	SkipRelease bool `yaml:"skip_release,omitempty"`

	// SkipPublish disables publishing for this library.
	SkipPublish bool `yaml:"skip_publish,omitempty"`

	// Output is the directory where generated code is written.
	Output string `yaml:"output,omitempty"`

	// Keep lists files and directories to preserve during regeneration.
	Keep []string `yaml:"keep,omitempty"`

	// CopyrightYear is the copyright year for the library.
	CopyrightYear string `yaml:"copyright_year,omitempty"`

	// Transport is the transport protocol, such as "grpc+rest" or "grpc".
	Transport string `yaml:"transport,omitempty"`

	// ReleaseLevel is either "stable" or "preview".
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// Rust contains Rust-specific library configuration.
	Rust *RustCrate `yaml:"rust,omitempty"`
}

// API describes an API to include in a library.
type API struct {
	// Path is the path to the API specification, such as
	// "google/cloud/secretmanager/v1".
	Path string `yaml:"path"`

	// ServiceConfig is the path to the service config file.
	ServiceConfig string `yaml:"service_config,omitempty"`

	// Format is the API specification format, either "protobuf" (default) or
	// "discovery".
	Format string `yaml:"format,omitempty"`
}

// Fill populates empty library fields from the provided defaults.
func (lib *Library) Fill(d *Default) {
	if d == nil {
		return
	}
	if lib.Output == "" {
		lib.Output = d.Output
	}
	if lib.ReleaseLevel == "" {
		lib.ReleaseLevel = d.ReleaseLevel
	}
	if d.Rust == nil {
		return
	}
	if lib.Rust == nil {
		lib.Rust = &RustCrate{}
	}
	lib.Rust.PackageDependencies = mergePackageDependencies(
		d.Rust.PackageDependencies,
		lib.Rust.PackageDependencies,
	)
	if len(lib.Rust.DisabledRustdocWarnings) == 0 {
		lib.Rust.DisabledRustdocWarnings = d.Rust.DisabledRustdocWarnings
	}
}

// AddLibraries adds library entries for APIs not covered by existing
// libraries.
func (cfg *Config) AddLibraries(googleapisDir string) error {
	allAPIs, err := apis(googleapisDir)
	if err != nil {
		return err
	}

	covered := make(map[string]bool)
	for _, lib := range cfg.Libraries {
		for _, api := range lib.APIs {
			covered[api.Path] = true
		}
		// If no APIs defined, derive path from name: google-cloud-foo-v1 -> google/cloud/foo/v1
		if len(lib.APIs) == 0 && lib.Name != "" {
			covered[strings.ReplaceAll(lib.Name, "-", "/")] = true
		}
	}

	for _, apiPath := range allAPIs {
		if covered[apiPath] {
			continue
		}
		serviceConfig, err := FindServiceConfig(googleapisDir, apiPath)
		if err != nil {
			return err
		}
		cfg.Libraries = append(cfg.Libraries, &Library{
			APIs: []*API{{
				Path:          apiPath,
				ServiceConfig: serviceConfig,
			}},
		})
	}
	return nil
}

// mergePackageDependencies merges default and library package dependencies,
// with library dependencies taking precedence for duplicates.
func mergePackageDependencies(defaults, lib []*RustPackageDependency) []*RustPackageDependency {
	seen := make(map[string]bool)
	var result []*RustPackageDependency
	for _, dep := range lib {
		seen[dep.Name] = true
		result = append(result, dep)
	}
	for _, dep := range defaults {
		if seen[dep.Name] {
			continue
		}
		copied := *dep
		result = append(result, &copied)
	}
	return result
}

// apis scans the googleapis directory and returns all API paths.
// It finds directories containing .proto files.
func apis(googleapisDir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(googleapisDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if !hasProtoFiles(path) {
			return nil
		}
		apiPath, err := filepath.Rel(googleapisDir, path)
		if err != nil {
			return err
		}
		paths = append(paths, apiPath)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

// hasProtoFiles returns true if the directory contains any .proto files.
func hasProtoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".proto") {
			return true
		}
	}
	return false
}

// FindServiceConfig finds the service config file for a channel path.  It
// looks for YAML files containing "type: google.api.Service", skipping any
// files ending in _gapic.yaml.
//
// The apiPath should be relative to googleapisDir (e.g.,
// "google/cloud/secretmanager/v1"). Returns the service config path relative
// to googleapisDir, or empty string if not found.
func FindServiceConfig(googleapisDir, apiPath string) (string, error) {
	dir := filepath.Join(googleapisDir, apiPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		if strings.HasSuffix(name, "_gapic.yaml") {
			continue
		}

		path := filepath.Join(dir, name)
		isServiceConfig, err := isServiceConfigFile(path)
		if err != nil {
			return "", err
		}
		if isServiceConfig {
			return filepath.Join(apiPath, name), nil
		}
	}
	return "", nil
}

// isServiceConfigFile checks if the file contains "type: google.api.Service".
func isServiceConfigFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for i := 0; i < 20 && scanner.Scan(); i++ {
		if strings.TrimSpace(scanner.Text()) == "type: google.api.Service" {
			return true, nil
		}
	}
	return false, scanner.Err()
}
