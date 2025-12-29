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
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	errDuplicateLibraryName = errors.New("duplicate library name")
	errDuplicateChannelPath = errors.New("duplicate channel path")
)

func tidyCommand() *cli.Command {
	return &cli.Command{
		Name:      "tidy",
		Usage:     "format and validate librarian.yaml",
		UsageText: "librarian tidy [path]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return RunTidy(ctx)
		},
	}
}

// RunTidy formats and validates the librarian configuration file.
func RunTidy(ctx context.Context) error {
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return err
	}
	if err := validateLibraries(cfg); err != nil {
		return err
	}

	var googleapisDir string
	if cfg.Sources != nil && cfg.Sources.Googleapis != nil {
		var err error
		googleapisDir, err = fetchGoogleapisDir(ctx, cfg.Sources)
		if err != nil {
			return err
		}
	}

	for _, lib := range cfg.Libraries {
		if lib.Output != "" && len(lib.Channels) == 1 && isDerivableOutput(cfg, lib) {
			lib.Output = ""
		}
		for _, ch := range lib.Channels {
			fixIncorrectServiceConfig(googleapisDir, cfg.Language, lib, ch)
			if isDerivableChannelPath(googleapisDir, cfg.Language, lib, ch) {
				ch.Path = ""
			}
			if isDerivableServiceConfig(googleapisDir, cfg.Language, lib, ch) {
				ch.ServiceConfig = ""
			}
		}
		lib.Channels = slices.DeleteFunc(lib.Channels, func(ch *config.Channel) bool {
			return ch.Path == "" && ch.ServiceConfig == ""
		})

		tidyLanguageConfig(lib, cfg.Language)
	}
	return yaml.Write(librarianConfigPath, formatConfig(cfg))
}

func isDerivableOutput(cfg *config.Config, lib *config.Library) bool {
	derivedOutput := defaultOutput(cfg.Language, lib.Channels[0].Path, cfg.Default.Output)
	return lib.Output == derivedOutput
}

func resolvedPath(language string, lib *config.Library, ch *config.Channel) string {
	if ch.Path != "" {
		return ch.Path
	}
	return deriveChannelPath(language, lib)
}

func fixIncorrectServiceConfig(googleapisDir, language string, lib *config.Library, ch *config.Channel) {
	if googleapisDir == "" || ch.ServiceConfig == "" {
		return
	}
	path := resolvedPath(language, lib, ch)
	correctConfig, err := serviceconfig.Find(googleapisDir, path)
	if err != nil || correctConfig == "" || ch.ServiceConfig == correctConfig {
		return
	}
	ch.ServiceConfig = ""
}

func isDerivableChannelPath(googleapisDir, language string, lib *config.Library, ch *config.Channel) bool {
	derivedPath := deriveChannelPath(language, lib)
	if ch.Path != derivedPath {
		return false
	}

	// Validate path exists if googleapis available
	if googleapisDir != "" {
		fullPath := filepath.Join(googleapisDir, derivedPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			return false
		}
		if !info.IsDir() {
			return false
		}
	}

	return true
}

func isDerivableServiceConfig(googleapisDir, language string, lib *config.Library, ch *config.Channel) bool {
	path := resolvedPath(language, lib, ch)

	// Fall back to heuristic if googleapis not available
	if googleapisDir == "" {
		return ch.ServiceConfig != "" && ch.ServiceConfig == deriveServiceConfig(path)
	}

	// Use serviceconfig.Find() for actual validation
	foundConfig, err := serviceconfig.Find(googleapisDir, path)
	if err != nil || foundConfig == "" {
		return false
	}

	return ch.ServiceConfig == foundConfig
}

// deriveServiceConfig returns the conventionally derived service config
// path for a given channel as a fallback when the googleapis directory
// is not available. For example, "google/cloud/speech/v1" derives to
// "google/cloud/speech/v1/speech_v1.yaml".
//
// It returns an empty string if the resolved path does not contain sufficient
// components or if the version component does not start with 'v'.
func deriveServiceConfig(resolvedPath string) string {
	parts := strings.Split(resolvedPath, "/")
	if len(parts) >= 2 {
		version := parts[len(parts)-1]
		service := parts[len(parts)-2]
		if strings.HasPrefix(version, "v") {
			return fmt.Sprintf("%s/%s_%s.yaml", resolvedPath, service, version)
		}
	}
	return ""
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
		for _, ch := range lib.Channels {
			if ch.Path != "" {
				pathCount[ch.Path]++
			}
		}
	}
	for name, count := range nameCount {
		if count > 1 {
			errs = append(errs, fmt.Errorf("%w: %s (appears %d times)", errDuplicateLibraryName, name, count))
		}
	}
	for path, count := range pathCount {
		if count > 1 {
			errs = append(errs, fmt.Errorf("%w: %s (appears %d times)", errDuplicateChannelPath, path, count))
		}
	}
	return errors.Join(errs...)
}

func tidyLanguageConfig(lib *config.Library, language string) {
	switch language {
	case languageRust:
		tidyRustConfig(lib)
	}
}

func tidyRustConfig(lib *config.Library) {
	if lib.Rust != nil && lib.Rust.Modules != nil {
		lib.Rust.Modules = slices.DeleteFunc(lib.Rust.Modules, func(module *config.RustModule) bool {
			return module.Source == "none" && module.Template == ""
		})
	}
}

func formatConfig(cfg *config.Config) *config.Config {
	if cfg.Default != nil && cfg.Default.Rust != nil {
		slices.SortFunc(cfg.Default.Rust.PackageDependencies, func(a, b *config.RustPackageDependency) int {
			return strings.Compare(a.Name, b.Name)
		})
	}

	slices.SortFunc(cfg.Libraries, func(a, b *config.Library) int {
		return strings.Compare(a.Name, b.Name)
	})
	for _, lib := range cfg.Libraries {
		slices.SortFunc(lib.Channels, func(a, b *config.Channel) int {
			return strings.Compare(a.Path, b.Path)
		})
		if lib.Rust != nil {
			slices.SortFunc(lib.Rust.PackageDependencies, func(a, b *config.RustPackageDependency) int {
				return strings.Compare(a.Name, b.Name)
			})
		}
	}
	return cfg
}
