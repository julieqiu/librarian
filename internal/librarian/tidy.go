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
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

func tidyCommand() *cli.Command {
	return &cli.Command{
		Name:      "tidy",
		Usage:     "format and validate librarian.yaml",
		UsageText: "librarian tidy [path]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runTidy(cmd.Args().First())
		},
	}
}

func runTidy(path string) error {
	configPath := librarianConfigPath
	if path != "" && path != "." {
		configPath = path
	}

	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		return err
	}

	var errs []error
	if err := validateLibraries(cfg); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return yaml.Write(configPath, formatConfig(cfg))
}

func validateLibraries(cfg *config.Config) error {
	var (
		errs      []error
		nameCount = make(map[string]int)
		pathCount = make(map[string]int)
	)
	for _, lib := range cfg.Libraries {
		if lib.Name != "" {
			nameCount[lib.Name]++
		}
		for _, api := range lib.APIs {
			if api.Path != "" {
				pathCount[api.Path]++
			}
		}
		if err := validateLibrary(lib); err != nil {
			errs = append(errs, err)
		}
	}
	for name, count := range nameCount {
		if count > 1 {
			errs = append(errs, fmt.Errorf("duplicate library name: %s (appears %d times)", name, count))
		}
	}
	for path, count := range pathCount {
		if count > 1 {
			errs = append(errs, fmt.Errorf("duplicate API path: %s (appears %d times)", path, count))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func validateLibrary(lib *config.Library) error {
	var errs []error
	if lib.Name != "" && !isDiscoveryAPI(lib) {
		derivedChannel := strings.ReplaceAll(lib.Name, "-", "/")
		if !strings.HasPrefix(derivedChannel, "google/") {
			errs = append(errs, fmt.Errorf("library %q: name cannot be derived into a valid channel (got %q, expected prefix 'google/')", lib.Name, derivedChannel))
		}
	}
	if isDiscoveryAPI(lib) {
		if lib.Output == "" {
			errs = append(errs, fmt.Errorf("library %q: discovery API requires explicit output path", lib.Name))
		}
	}
	if lib.Rust != nil && len(lib.Rust.ExtraModules) > 0 && lib.Output == "" {
		errs = append(errs, fmt.Errorf("library %q: has extra_modules but no output path", lib.Name))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func formatConfig(cfg *config.Config) *config.Config {
	// Sort default package dependencies.
	if cfg.Default != nil && cfg.Default.Rust != nil {
		slices.SortFunc(cfg.Default.Rust.PackageDependencies, func(a, b *config.RustPackageDependency) int {
			return strings.Compare(a.Name, b.Name)
		})
	}

	slices.SortFunc(cfg.Libraries, func(a, b *config.Library) int {
		return strings.Compare(a.Name, b.Name)
	})
	var defaultOutput string
	if cfg.Default != nil {
		defaultOutput = cfg.Default.Output
	}
	result := make([]*config.Library, 0, len(cfg.Libraries))
	for _, lib := range cfg.Libraries {
		// Sort APIs by path.
		slices.SortFunc(lib.APIs, func(a, b *config.API) int {
			return strings.Compare(a.Path, b.Path)
		})
		// Sort rust package dependencies.
		if lib.Rust != nil {
			slices.SortFunc(lib.Rust.PackageDependencies, func(a, b *config.RustPackageDependency) int {
				return strings.Compare(a.Name, b.Name)
			})
		}
		newlib := removeRedundantFields(lib, defaultOutput)
		if !isDefaultLibrary(newlib) {
			result = append(result, newlib)
		}
	}
	cfg.Libraries = result
	return cfg
}

func removeRedundantFields(lib *config.Library, defaultOutput string) *config.Library {
	if isDiscoveryAPI(lib) {
		return lib
	}
	var apiPath string
	if len(lib.APIs) > 0 {
		apiPath = lib.APIs[0].Path
	}
	if apiPath == "" {
		return lib
	}
	if lib.Output != "" {
		derivedOutput := defaultOutput + strings.TrimPrefix(apiPath, "google/")
		if lib.Output == derivedOutput {
			lib.Output = ""
		}
	}
	return lib
}

func isDefaultLibrary(lib *config.Library) bool {
	if lib.Name == "" {
		return false
	}
	return reflect.DeepEqual(lib, &config.Library{Name: lib.Name})
}

func isDiscoveryAPI(lib *config.Library) bool {
	for _, api := range lib.APIs {
		if api.Format == "discovery" {
			return true
		}
	}
	return false
}
