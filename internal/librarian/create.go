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
	"sort"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/serviceconfig"

	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	errUnsupportedLanguage = errors.New("library creation is not supported for the specified language")
	errMissingLibraryName  = errors.New("must provide library name as argument to create a new library")
	errNoYaml              = errors.New("unable to read librarian.yaml")
)

func createCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "create a new client library",
		UsageText: "librarian create <library> [apis...]",
		Action: func(ctx context.Context, c *cli.Command) error {
			name := c.Args().First()
			if name == "" {
				return errMissingLibraryName
			}
			return runCreate(ctx, name, c.Args().Slice()[1:]...)
		},
	}
}

func runCreate(ctx context.Context, name string, channel ...string) error {
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return fmt.Errorf("%w: %v", errNoYaml, err)
	}

	if cfg.Sources == nil || cfg.Sources.Googleapis == nil {
		return errNoGoogleapiSourceInfo
	}
	googleapisDir, err := fetchSource(ctx, cfg.Sources.Googleapis, googleapisRepo)
	if err != nil {
		return err
	}
	if err := addLibraryToConfig(cfg, name, googleapisDir, channel...); err != nil {
		return err
	}

	c := cfg.Libraries[0].Channels[0].Path
	output := defaultOutput(cfg.Language, c, cfg.Default.Output)
	switch cfg.Language {
	case languageFake:
		if err := runGenerate(ctx, false, name); err != nil {
			return err
		}
	case languageRust:
		if err := rust.PrepareCargoWorkspace(ctx, output); err != nil {
			return err
		}
		if err := runGenerate(ctx, false, name); err != nil {
			return err
		}
		if err := rust.FormatAndValidateLibrary(ctx, output); err != nil {
			return err
		}
	default:
		return errUnsupportedLanguage
	}

	if err := tidyConfig(ctx, cfg, googleapisDir); err != nil {
		return err
	}
	return yaml.Write(librarianConfigPath, formatConfig(cfg))
}

func addLibraryToConfig(cfg *config.Config, name, googleapisDir string, channel ...string) error {
	for _, lib := range cfg.Libraries {
		if lib.Name == name {
			return fmt.Errorf("%q already exists", name)
		}
	}

	lib := &config.Library{
		Name:    name,
		Version: "0.1.0",
	}

	if len(channel) > 0 {
		for _, c := range channel {
			sc, err := serviceconfig.Find(googleapisDir, c)
			if err != nil {
				return err
			}
			lib.Channels = append(lib.Channels, &config.Channel{
				Path:          c,
				ServiceConfig: sc,
			})
		}
	} else {
		c := deriveChannelPath(cfg.Language, lib)
		sc, err := serviceconfig.Find(googleapisDir, c)
		if err != nil {
			return err
		}
		lib.Channels = append(lib.Channels, &config.Channel{
			Path:          c,
			ServiceConfig: sc,
		})
	}
	if len(lib.Channels) > 1 {
		sort.Slice(lib, func(i, j int) bool {
			return lib.Channels[i].Path < lib.Channels[j].Path
		})
	}

	cfg.Libraries = append(cfg.Libraries, lib)
	sort.Slice(cfg.Libraries, func(i, j int) bool {
		return cfg.Libraries[i].Name < cfg.Libraries[j].Name
	})
	return nil
}
