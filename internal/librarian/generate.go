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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/librarian/internal/rust"
	"github.com/urfave/cli/v3"
)

const googleapisRepo = "github.com/googleapis/googleapis"

var (
	errMissingLibraryOrAllFlag = errors.New("must specify library name or use --all flag")
	errBothLibraryAndAllFlag   = errors.New("cannot specify both library name and --all flag")
	errEmptySources            = errors.New("sources field is required in librarian.yaml: specify googleapis and/or discovery source commits")
)

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate a client library",
		UsageText: "librarian generate [library] [--all]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "generate all libraries",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			all := cmd.Bool("all")
			libraryName := cmd.Args().First()
			if !all && libraryName == "" {
				return errMissingLibraryOrAllFlag
			}
			if all && libraryName != "" {
				return errBothLibraryAndAllFlag
			}
			return runGenerate(ctx, all, libraryName)
		},
	}
}

func runGenerate(ctx context.Context, all bool, libraryName string) error {
	cfg, err := config.Read(librarianConfigPath)
	if err != nil {
		return err
	}
	if cfg.Sources == nil {
		return errEmptySources
	}
	if all {
		return generateAll(ctx, cfg)
	}
	return generateLibrary(ctx, cfg, libraryName)
}

func generateAll(ctx context.Context, cfg *config.Config) error {
	googleapisDir, err := googleapisDir(cfg.Sources.Googleapis)
	if err != nil {
		return err
	}

	if err := cfg.Discover(googleapisDir); err != nil {
		return err
	}
	for _, lib := range cfg.Libraries {
		lib.Fill(cfg.Default)
	}

	for _, lib := range cfg.Libraries {
		if lib.Channel != "" && lib.ServiceConfig == "" {
			serviceConfig, err := config.ServiceConfig(googleapisDir, lib.Channel)
			if err != nil {
				return err
			}
			lib.ServiceConfig = serviceConfig
		}
		for _, ch := range lib.Channels {
			if lib.APIServiceConfigs == nil {
				lib.APIServiceConfigs = make(map[string]string)
			}
			if lib.APIServiceConfigs[ch] == "" {
				serviceConfig, err := config.ServiceConfig(googleapisDir, ch)
				if err != nil {
					return err
				}
				lib.APIServiceConfigs[ch] = serviceConfig
			}
		}
	}

	var errs []error
	for _, lib := range cfg.Libraries {
		if lib.Generate != nil && lib.Generate.Disabled {
			continue
		}
		if err := generate(ctx, cfg.Language, lib, cfg.Sources); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func generateLibrary(ctx context.Context, cfg *config.Config, libraryName string) error {
	var library *config.Library
	for _, lib := range cfg.Libraries {
		if lib.Name == libraryName {
			library = lib
			break
		}
	}
	if library == nil {
		library = &config.Library{Name: libraryName}
	}
	library.Fill(cfg.Default)

	googleapisDir, err := googleapisDir(cfg.Sources.Googleapis)
	if err != nil {
		return err
	}

	if library.ServiceConfig == "" {
		serviceConfig, err := config.ServiceConfig(googleapisDir, library.Channel)
		if err != nil {
			return err
		}
		library.ServiceConfig = serviceConfig
	}

	library.Fill(cfg.Default)
	return generate(ctx, cfg.Language, library, cfg.Sources)
}

func generate(ctx context.Context, language string, library *config.Library, sources *config.Sources) error {
	var err error
	switch language {
	case "testhelper":
		err = testGenerate(library)
	case "rust":
		err = rust.Generate(ctx, library, sources)
	default:
		err = fmt.Errorf("generate not implemented for %q", language)
	}

	if err != nil {
		fmt.Printf("✗ Error generating %s: %v\n", library.Name, err)
		return err
	}
	fmt.Printf("✓ Successfully generated %s\n", library.Name)
	return nil
}

// googleapisDir returns the local directory path for the googleapis repository.
func googleapisDir(source *config.Source) (string, error) {
	if source == nil {
		return "", errors.New("googleapis source is required")
	}
	if source.Dir != "" {
		return source.Dir, nil
	}
	return fetch.RepoDir(googleapisRepo, source.Commit, source.SHA256)
}
