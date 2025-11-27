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
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/urfave/cli/v3"
)

func fmtCommand() *cli.Command {
	return &cli.Command{
		Name:      "fmt",
		Usage:     "format and validate librarian.yaml",
		UsageText: "librarian fmt [-w] [path]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "w",
				Usage: "write result to file",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			write := cmd.Bool("w")
			path := cmd.Args().First()
			if write && path == "" {
				return errors.New("path required when using -w flag (use '.' for current directory)")
			}
			return runFmt(write, path)
		},
	}
}

func runFmt(write bool, path string) error {
	// Determine the config file path.
	configPath := librarianConfigPath
	if path != "" && path != "." {
		configPath = path
	}

	cfg, err := config.Read(configPath)
	if err != nil {
		return err
	}

	// Format the config first (remove duplicates, sort, etc.).
	formatConfig(cfg)

	// Then validate the formatted config.
	var errs []error
	if err := validateLibraries(cfg); err != nil {
		errs = append(errs, err)
	}
	if err := validateIgnored(cfg); err != nil {
		errs = append(errs, err)
	}

	if !write {
		// Check only mode (default).
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
		fmt.Println("librarian.yaml is valid")
		return nil
	}

	// Write the formatted config.
	if err := cfg.Write(configPath); err != nil {
		return err
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	fmt.Println("librarian.yaml formatted successfully")
	return nil
}

// validateLibraries checks for issues in library configurations.
func validateLibraries(cfg *config.Config) error {
	var errs []error

	// Check for duplicate library names.
	nameCount := make(map[string]int)
	for _, lib := range cfg.Libraries {
		if lib.Name != "" {
			nameCount[lib.Name]++
		}
	}
	for name, count := range nameCount {
		if count > 1 {
			errs = append(errs, fmt.Errorf("duplicate library name: %s (appears %d times)", name, count))
		}
	}

	// Check for duplicate channels.
	channelCount := make(map[string]int)
	for _, lib := range cfg.Libraries {
		channel := lib.Channel
		if channel == "" && lib.Name != "" && !isDiscoveryAPI(lib) {
			// Derive channel from name like Fill() does.
			channel = strings.ReplaceAll(lib.Name, "-", "/")
		}
		if channel != "" {
			channelCount[channel]++
		}
		for _, ch := range lib.Channels {
			channelCount[ch]++
		}
	}
	for channel, count := range channelCount {
		if count > 1 {
			errs = append(errs, fmt.Errorf("duplicate channel: %s (appears %d times)", channel, count))
		}
	}

	// Validate each library.
	for _, lib := range cfg.Libraries {
		if err := validateLibrary(lib); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// validateLibrary checks a single library configuration for issues.
func validateLibrary(lib *config.Library) error {
	var errs []error

	// Check that name can be derived into a valid channel (for protobuf APIs).
	if lib.Name != "" && lib.Channel == "" && !isDiscoveryAPI(lib) {
		derivedChannel := strings.ReplaceAll(lib.Name, "-", "/")
		// Channel should start with "google/" for googleapis.
		if !strings.HasPrefix(derivedChannel, "google/") {
			errs = append(errs, fmt.Errorf("library %q: name cannot be derived into a valid channel (got %q, expected prefix 'google/')", lib.Name, derivedChannel))
		}
	}

	// Check that Discovery APIs have required fields.
	if isDiscoveryAPI(lib) {
		if lib.SpecificationSource == "" {
			errs = append(errs, fmt.Errorf("library %q: discovery API requires specification_source", lib.Name))
		}
		if lib.Output == "" {
			errs = append(errs, fmt.Errorf("library %q: discovery API requires explicit output path (cannot derive from channel)", lib.Name))
		}
	}

	// Check that libraries with extra_modules have an output path.
	if lib.Rust != nil && len(lib.Rust.ExtraModules) > 0 && lib.Output == "" && lib.Channel == "" {
		errs = append(errs, fmt.Errorf("library %q: has extra_modules but no output path or channel", lib.Name))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// validateIgnored checks for issues in the ignored list.
func validateIgnored(cfg *config.Config) error {
	var errs []error

	// Check for duplicates in ignored list.
	seen := make(map[string]bool)
	for _, pattern := range cfg.Ignored {
		if seen[pattern] {
			errs = append(errs, fmt.Errorf("duplicate ignored pattern: %s", pattern))
		}
		seen[pattern] = true
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// formatConfig cleans up and normalizes the configuration.
func formatConfig(cfg *config.Config) {
	// Remove duplicate libraries (keep first occurrence).
	cfg.Libraries = removeDuplicateLibraries(cfg.Libraries)

	// Sort libraries by name.
	slices.SortFunc(cfg.Libraries, func(a, b *config.Library) int {
		return strings.Compare(a.Name, b.Name)
	})

	// Remove duplicate ignored patterns.
	cfg.Ignored = removeDuplicateStrings(cfg.Ignored)

	// Sort ignored patterns.
	slices.Sort(cfg.Ignored)
}

// removeDuplicateLibraries removes duplicate libraries by name, keeping the first occurrence.
func removeDuplicateLibraries(libs []*config.Library) []*config.Library {
	seen := make(map[string]bool)
	result := make([]*config.Library, 0, len(libs))
	for _, lib := range libs {
		if lib.Name == "" || !seen[lib.Name] {
			result = append(result, lib)
			if lib.Name != "" {
				seen[lib.Name] = true
			}
		}
	}
	return result
}

// removeDuplicateStrings removes duplicate strings, keeping the first occurrence.
func removeDuplicateStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(strs))
	for _, s := range strs {
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}
	return result
}

// isDiscoveryAPI checks if a library uses the Discovery API format.
func isDiscoveryAPI(lib *config.Library) bool {
	return lib.SpecificationFormat == "discovery"
}
