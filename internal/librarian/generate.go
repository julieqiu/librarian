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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/librarian/internal/golang"
	"github.com/googleapis/librarian/internal/librarian/internal/rust"
	"github.com/googleapis/librarian/internal/yaml"
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
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return err
	}
	if cfg.Sources == nil {
		return errEmptySources
	}
	var generatedDirs []string
	if all {
		dirs, err := generateAll(ctx, cfg)
		if err != nil {
			return err
		}
		generatedDirs = dirs
	} else {
		dir, err := generateLibrary(ctx, cfg, libraryName)
		if err != nil {
			return err
		}
		generatedDirs = []string{dir}
	}
	// Format generated files at the end.
	if cfg.Language == "rust" && len(generatedDirs) > 0 {
		// Filter to only directories that exist.
		var existingDirs []string
		for _, dir := range generatedDirs {
			if _, err := os.Stat(dir); err == nil {
				existingDirs = append(existingDirs, dir)
			}
		}
		if len(existingDirs) > 0 {
			fmt.Println("Formatting generated files...")
			if err := rust.Format(existingDirs...); err != nil {
				return err
			}
		}
	}
	return nil
}

func generateAll(ctx context.Context, cfg *config.Config) ([]string, error) {
	googleapisDir, err := fetchGoogleapisDir(cfg.Sources)
	if err != nil {
		return nil, err
	}
	// Apply defaults to populate API paths before checking coverage.
	// Skip libraries marked with skip_generate.
	for _, lib := range cfg.Libraries {
		if !lib.SkipGenerate {
			applyDefault(lib, cfg.Default)
		}
	}
	uncoveredAPIs := findUncoveredAPIs(cfg, googleapisDir)
	var generatedDirs []string
	for _, lib := range cfg.Libraries {
		dir, err := generateLibrary(ctx, cfg, lib.Name)
		if err != nil {
			return nil, err
		}
		generatedDirs = append(generatedDirs, dir)
	}
	// Report uncovered APIs that should be added to exclusion list or librarian.yaml.
	if len(uncoveredAPIs) > 0 {
		fmt.Println("\nAPIs found in googleapis but not in librarian.yaml or exclusion list:")
		for _, api := range uncoveredAPIs {
			fmt.Printf("  - %s\n", api)
		}
		fmt.Println("Add these to config.ExcludedAPIs or create library entries in librarian.yaml.")
	}
	return generatedDirs, nil
}

func generateLibrary(ctx context.Context, cfg *config.Config, libraryName string) (string, error) {
	googleapisDir, err := fetchGoogleapisDir(cfg.Sources)
	if err != nil {
		return "", err
	}
	for _, lib := range cfg.Libraries {
		if lib.Name == libraryName {
			if lib.SkipGenerate {
				fmt.Printf("skipping %s: skip_generate is set\n", lib.Name)
				return lib.Output, nil
			}
			applyDefault(lib, cfg.Default)
			for _, api := range lib.APIs {
				if api.ServiceConfig == "" {
					serviceConfig, err := findServiceConfig(googleapisDir, api.Path)
					if err != nil {
						return "", err
					}
					api.ServiceConfig = serviceConfig
				}
			}
			if err := generate(ctx, cfg.Language, lib, cfg.Sources); err != nil {
				return "", err
			}
			return lib.Output, nil
		}
	}
	return "", fmt.Errorf("library %q not found", libraryName)
}

func generate(ctx context.Context, language string, library *config.Library, sources *config.Sources) error {
	var err error
	switch language {
	case "testhelper":
		err = testGenerate(library)
	case "go":
		if err := golang.RequireTools(); err != nil {
			return err
		}
		keep := append(library.Keep,
			"CHANGES.md",
			"README.md",
			"go.mod",
			"internal/generated/snippets/go.mod",
			"internal/version.go",
		)
		if err := cleanOutput(library.Output, keep); err != nil {
			return err
		}
		err = golang.Generate(ctx, library, sources)
	case "rust":
		if err := rust.RequireTools(); err != nil {
			return err
		}
		// Always keep Cargo.toml for Rust libraries.
		keep := append(library.Keep, "Cargo.toml")
		if err := cleanOutput(library.Output, keep); err != nil {
			return err
		}
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

func fetchGoogleapisDir(sources *config.Sources) (string, error) {
	if sources == nil || sources.Googleapis == nil {
		return "", errors.New("googleapis source is required")
	}
	if sources.Googleapis.Dir != "" {
		return sources.Googleapis.Dir, nil
	}
	return fetch.RepoDir(googleapisRepo, sources.Googleapis.Commit, sources.Googleapis.SHA256)
}
