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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/internal/golang"
	"github.com/googleapis/librarian/internal/librarian/internal/rust"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var errMissingLibraryName = errors.New("library name is required")

func createCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "create a new client library",
		UsageText: "librarian create <library> [api-paths...]",
		Description: `Creates a new client library.

The library name is required (e.g., "google-cloud-secretmanager-v1").

API paths are optional. If not provided, the path will be derived from the
library name by replacing "-" with "/" (e.g., "google-cloud-secretmanager-v1"
becomes "google/cloud/secretmanager/v1").

Examples:
  librarian create google-cloud-secretmanager-v1
  librarian create secretmanager google/cloud/secretmanager/v1
  librarian create spanner google/spanner/v1 google/spanner/admin/database/v1`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				return errMissingLibraryName
			}
			libraryName := args[0]
			apiPaths := args[1:]
			return runCreate(ctx, libraryName, apiPaths)
		},
	}
}

func runCreate(ctx context.Context, libraryName string, apiPaths []string) error {
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return err
	}
	if cfg.Sources == nil {
		return errEmptySources
	}

	// Get googleapis directory.
	googleapisDir, err := fetchGoogleapisDir(cfg.Sources)
	if err != nil {
		return err
	}

	// Derive API path from library name if not provided.
	// Same logic as applyDefault: replace "-" with "/".
	if len(apiPaths) == 0 {
		apiPaths = []string{strings.ReplaceAll(libraryName, "-", "/")}
	}

	// Build API configs with service configs.
	var apis []*config.API
	for _, apiPath := range apiPaths {
		serviceConfig, err := findServiceConfig(googleapisDir, apiPath)
		if err != nil {
			return fmt.Errorf("failed to find service config for %s: %w", apiPath, err)
		}
		apis = append(apis, &config.API{
			Path:          apiPath,
			ServiceConfig: serviceConfig,
		})
	}

	switch cfg.Language {
	case "testhelper":
		return testCreate(libraryName, apis)
	case "rust":
		if err := rust.RequireTools(); err != nil {
			return err
		}
		return rust.Create(ctx, cfg, libraryName, apis)
	case "go":
		if err := golang.RequireTools(); err != nil {
			return err
		}
		return golang.Create(ctx, libraryName, apis, googleapisDir)
	default:
		return fmt.Errorf("create not implemented for %q", cfg.Language)
	}
}
