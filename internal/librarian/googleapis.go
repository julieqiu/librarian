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

package librarian

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// findUncoveredAPIs returns API paths found in googleapis that are not covered
// by existing libraries. It expects applyDefault to have been called on all
// libraries first to populate API paths.
func findUncoveredAPIs(cfg *config.Config, googleapisDir string) []string {
	allAPIs, err := listAPIs(googleapisDir, cfg.Language)
	if err != nil {
		return nil
	}

	covered := make(map[string]bool)
	for _, lib := range cfg.Libraries {
		for _, api := range lib.APIs {
			covered[api.Path] = true
		}
	}
	var uncovered []string
	for _, apiPath := range allAPIs {
		if !covered[apiPath] {
			uncovered = append(uncovered, apiPath)
		}
	}
	return uncovered
}

// listAPIs scans the googleapis directory and returns all API paths. It finds
// directories containing .proto files, excluding paths in config.ExcludedAPIs.
func listAPIs(googleapisDir, language string) ([]string, error) {
	excluded := buildExclusionSet(language)
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
		if excluded.matchesPrefix(apiPath) {
			return filepath.SkipDir
		}
		if !hasProtoFiles(path) {
			return nil
		}
		// Skip exact matches.
		if excluded.matchesExact(apiPath) {
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

// exclusionSet holds prefix and exact match exclusions for efficient lookup.
type exclusionSet struct {
	prefixes []string
	exact    map[string]bool
}

// buildExclusionSet creates an exclusion set for the given language.
func buildExclusionSet(language string) *exclusionSet {
	es := &exclusionSet{
		exact: make(map[string]bool),
	}
	// Add "All" exclusions.
	es.prefixes = append(es.prefixes, config.ExcludedAPIs.All...)
	for _, p := range config.ExactExcludedAPIs.All {
		es.exact[p] = true
	}
	// Add language-specific exclusions.
	switch language {
	case "go":
		es.prefixes = append(es.prefixes, config.ExcludedAPIs.Go...)
		for _, p := range config.ExactExcludedAPIs.Go {
			es.exact[p] = true
		}
	case "rust":
		es.prefixes = append(es.prefixes, config.ExcludedAPIs.Rust...)
		for _, p := range config.ExactExcludedAPIs.Rust {
			es.exact[p] = true
		}
	}
	return es
}

func (es *exclusionSet) matchesPrefix(path string) bool {
	for _, prefix := range es.prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func (es *exclusionSet) matchesExact(path string) bool {
	return es.exact[path]
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

// findServiceConfig finds the service config file for a channel path. It looks
// for YAML files containing "type: google.api.Service", skipping any files
// ending in _gapic.yaml.
//
// The apiPath should be relative to googleapisDir (e.g.,
// "google/cloud/secretmanager/v1"). Returns the service config path relative
// to googleapisDir, or empty string if not found.
func findServiceConfig(googleapisDir, apiPath string) (string, error) {
	// Check known service config full path overrides first (for APIs with
	// service config in a different directory).
	if sc, ok := serviceconfig.PathOverride(apiPath); ok {
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
