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

package librarian

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// addLibraries adds library entries to cfg for APIs not covered by existing
// libraries.
func addLibraries(cfg *config.Config, googleapisDir string) error {
	allAPIs, err := listAPIs(googleapisDir)
	if err != nil {
		return err
	}

	covered := make(map[string]bool)
	for _, lib := range cfg.Libraries {
		for _, api := range lib.APIs {
			covered[api.Path] = true
		}
		// If no APIs defined, derive path from name: google-cloud-foo-v1 ->
		// google/cloud/foo/v1.
		if len(lib.APIs) == 0 && lib.Name != "" {
			covered[strings.ReplaceAll(lib.Name, "-", "/")] = true
		}
	}
	for _, apiPath := range allAPIs {
		if covered[apiPath] {
			continue
		}
		serviceConfig, err := findServiceConfig(googleapisDir, apiPath)
		if err != nil {
			return err
		}
		cfg.Libraries = append(cfg.Libraries, &config.Library{
			APIs: []*config.API{{
				Path:          apiPath,
				ServiceConfig: serviceConfig,
			}},
		})
	}
	return nil
}

// excludedAPIPrefixes contains directory prefixes that should not be scanned
// for APIs. These are typically tooling or metadata directories.
var excludedAPIPrefixes = []string{
	"gapic",
}

// listAPIs scans the googleapis directory and returns all API paths. It finds
// directories containing .proto files.
func listAPIs(googleapisDir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(googleapisDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		apiPath, err := filepath.Rel(googleapisDir, path)
		if err != nil {
			return err
		}
		// Skip excluded directories.
		for _, prefix := range excludedAPIPrefixes {
			if strings.HasPrefix(apiPath, prefix) {
				return filepath.SkipDir
			}
		}
		if !hasProtoFiles(path) {
			return nil
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

// knownServiceConfigs maps API paths to their service config files for cases
// where the service config is not in the same directory as the API.
var knownServiceConfigs = map[string]string{
	"google/cloud/aiplatform/v1/schema/predict/instance":       "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/v1/schema/predict/params":         "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/v1/schema/predict/prediction":     "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/v1/schema/trainingjob/definition": "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
}

// findServiceConfig finds the service config file for a channel path. It looks
// for YAML files containing "type: google.api.Service", skipping any files
// ending in _gapic.yaml.
//
// The apiPath should be relative to googleapisDir (e.g.,
// "google/cloud/secretmanager/v1"). Returns the service config path relative
// to googleapisDir, or empty string if not found.
func findServiceConfig(googleapisDir, apiPath string) (string, error) {
	// Check known service config mappings first.
	if sc, ok := knownServiceConfigs[apiPath]; ok {
		return sc, nil
	}
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

// applyDefault populates empty library fields from the provided defaults.
func applyDefault(lib *config.Library, d *config.Default) {
	// Ensure each library has at least one API with a path.
	if len(lib.APIs) == 0 {
		lib.APIs = []*config.API{{}}
	}
	if lib.APIs[0].Path == "" && lib.Name != "" {
		lib.APIs[0].Path = strings.ReplaceAll(lib.Name, "-", "/")
	}

	if d == nil {
		return
	}
	if lib.Output == "" && d.Output != "" {
		// Derive output path from default output and API path.
		// e.g., "src/generated" + "cloud/accessapproval/v1" for API path "google/cloud/accessapproval/v1"
		// or "src/generated" + "grafeas/v1" for API path "grafeas/v1"
		apiPath := lib.APIs[0].Path
		lib.Output = filepath.Join(d.Output, strings.TrimPrefix(apiPath, "google/"))
	}
	if lib.ReleaseLevel == "" {
		lib.ReleaseLevel = d.ReleaseLevel
	}
	if d.Rust == nil {
		return
	}
	if lib.Rust == nil {
		lib.Rust = &config.RustCrate{}
	}
	lib.Rust.PackageDependencies = mergePackageDependencies(
		d.Rust.PackageDependencies,
		lib.Rust.PackageDependencies,
	)
	if len(lib.Rust.DisabledRustdocWarnings) == 0 {
		lib.Rust.DisabledRustdocWarnings = d.Rust.DisabledRustdocWarnings
	}
}

// mergePackageDependencies merges default and library package dependencies,
// with library dependencies taking precedence for duplicates.
func mergePackageDependencies(defaults, lib []*config.RustPackageDependency) []*config.RustPackageDependency {
	seen := make(map[string]bool)
	var result []*config.RustPackageDependency
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
