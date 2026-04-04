// Copyright 2026 Google LLC
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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var errMultipleGRPCServiceConfigs = errors.New("found multiple gRPC service config files")

func importGRPCServiceConfigCommand() *cli.Command {
	return &cli.Command{
		Name:      "import-grpc-service-config",
		Usage:     "import gRPC service config JSON data into sdk.yaml",
		UsageText: "import-grpc-service-config <googleapis-dir>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 1 {
				return fmt.Errorf("expected 1 argument, got %d", cmd.NArg())
			}
			return runImportGRPCServiceConfig("internal/serviceconfig/sdk.yaml", cmd.Args().First())
		},
	}
}

func runImportGRPCServiceConfig(sdkYaml, googleapisDir string) error {
	apis, err := yaml.Read[[]*serviceconfig.API](sdkYaml)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", sdkYaml, err)
	}
	for _, api := range *apis {
		matches, err := filepath.Glob(filepath.Join(googleapisDir, api.Path, "*_grpc_service_config.json"))
		if err != nil {
			return fmt.Errorf("failed to glob for gRPC service config in %s: %w", api.Path, err)
		}
		if len(matches) == 0 {
			continue
		}
		if len(matches) > 1 {
			return fmt.Errorf("%w in %s: %v", errMultipleGRPCServiceConfigs, api.Path, matches)
		}
		data, err := os.ReadFile(matches[0])
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", matches[0], err)
		}
		var cfg serviceconfig.GRPCServiceConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("failed to unmarshal %s: %w", matches[0], err)
		}
		if len(cfg.MethodConfig) == 0 {
			continue
		}
		api.GRPCServiceConfig = &cfg
	}
	sort.Slice(*apis, func(i, j int) bool {
		return (*apis)[i].Path < (*apis)[j].Path
	})
	return yaml.Write(sdkYaml, *apis)
}
