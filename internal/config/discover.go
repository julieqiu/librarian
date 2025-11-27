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

	"github.com/googleapis/librarian/internal/config/cli"
)

// channels scans the googleapis directory and returns all API channel paths.
// It finds directories containing .proto files.
func channels(googleapisDir string) ([]string, error) {
	var channels []string

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

		channelPath, err := filepath.Rel(googleapisDir, path)
		if err != nil {
			return err
		}

		channels = append(channels, channelPath)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return channels, nil
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

// ServiceConfig finds the service config file for a channel path.
// It looks for YAML files containing "type: google.api.Service", skipping
// any files ending in _gapic.yaml.
// The channelPath should be relative to googleapisDir (e.g., "google/cloud/secretmanager/v1").
// Returns the service config path relative to googleapisDir, or empty string if not found.
func ServiceConfig(googleapisDir, channelPath string) (string, error) {
	dir := filepath.Join(googleapisDir, channelPath)
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
			return filepath.Join(channelPath, name), nil
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
	for i := 0; i < 5 && scanner.Scan(); i++ {
		if strings.TrimSpace(scanner.Text()) == "type: google.api.Service" {
			return true, nil
		}
	}
	return false, scanner.Err()
}

// Discover finds channels not covered by existing libraries and
// appends new Library entries to cfg.Libraries.
func (cfg *Config) Discover(googleapisDir string) error {
	channels, err := channels(googleapisDir)
	if err != nil {
		return err
	}

	covered := make(map[string]bool)
	for _, lib := range cfg.Libraries {
		channel := lib.Channel
		if channel == "" && lib.Name != "" {
			// Derive channel from name: google-cloud-foo-v1 -> google/cloud/foo/v1
			channel = strings.ReplaceAll(lib.Name, "-", "/")
		}
		if channel != "" {
			covered[channel] = true
		}
		for _, ch := range lib.Channels {
			covered[ch] = true
		}
	}

	for _, channelPath := range channels {
		if covered[channelPath] {
			continue
		}
		if isIgnored(channelPath, cfg.Language) {
			continue
		}
		serviceConfig, err := ServiceConfig(googleapisDir, channelPath)
		if err != nil {
			return err
		}
		cfg.Libraries = append(cfg.Libraries, &Library{
			Channel:       channelPath,
			ServiceConfig: serviceConfig,
		})
	}
	return nil
}

// isIgnored returns true if channelPath starts with any excluded API prefix.
// It checks both the universal exclusions (cli.ExcludedAPIs.All) and
// language-specific exclusions based on the language parameter.
func isIgnored(channelPath, language string) bool {
	for _, prefix := range cli.ExcludedAPIs.All {
		if strings.HasPrefix(channelPath, prefix) {
			return true
		}
	}
	var langExcluded []string
	switch language {
	case "rust":
		langExcluded = cli.ExcludedAPIs.Rust
	}
	for _, prefix := range langExcluded {
		if strings.HasPrefix(channelPath, prefix) {
			return true
		}
	}
	return false
}
