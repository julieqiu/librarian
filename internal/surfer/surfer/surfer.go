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

// Package surfer provides the core implementation for the surfer CLI tool.
package surfer

import (
	"context"
	"fmt"
	"os"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/surfer/gcloud"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// Run executes the surfer CLI with the given command line arguments.
func Run(ctx context.Context, args ...string) error {
	cmd := &cli.Command{
		Name:        "surfer",
		Usage:       "generates gcloud command YAML files",
		UsageText:   "surfer generate [arguments]",
		Description: "surfer generates gcloud command YAML files",
		Commands: []*cli.Command{
			generateCommand(),
		},
	}
	return cmd.Run(ctx, args)
}

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generates gcloud commands",
		UsageText: "surfer generate <path to gcloud.yaml> --googleapis <path> [--out <path>]",
		Description: `generate generates gcloud command files from protobuf API specifications,
service config yaml, and gcloud.yaml.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "googleapis",
				Value: "https://github.com/googleapis/googleapis",
				Usage: "URL or directory path to googleapis",
			},
			&cli.StringFlag{
				Name:  "out",
				Value: ".",
				Usage: "output directory",
			},
			&cli.StringFlag{
				Name:  "proto-files-include-list",
				Value: "google/cloud/parallelstore/v1/parallelstore.proto",
				Usage: "comma-separated list of protobuf files used to generate the gcloud commands",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() == 0 {
				return fmt.Errorf("path to gcloud.yaml is required")
			}
			gcloudConfig := cmd.Args().First()
			googleapis := cmd.String("googleapis")
			out := cmd.String("out")
			includeList := cmd.String("proto-files-include-list")
			return Generate(googleapis, gcloudConfig, out, includeList)
		},
	}
}

// Generate generates gcloud commands for a service.
func Generate(googleapis, gcloudConfigPath, output, includeList string) error {
	overrides, err := readGcloudConfig(gcloudConfigPath)
	if err != nil {
		return err
	}

	model, err := createAPIModel(googleapis, includeList)
	if err != nil {
		return err
	}

	if len(model.Services) == 0 {
		return fmt.Errorf("no services found in the provided protos")
	}

	for _, service := range model.Services {
		// TODO(https://github.com/googleapis/librarian/issues/3291): Ensure output directories don't collide if multiple services share a name.
		if err := gcloud.GenerateService(service, overrides, model, output); err != nil {
			return fmt.Errorf("failed to generate commands for service %q: %w", service.Name, err)
		}
	}
	return nil
}

// createAPIModel parses the service specification and creates the API model.
func createAPIModel(googleapisPath, includeList string) (*api.API, error) {
	parserConfig := &config.Config{
		General: config.GeneralConfig{
			SpecificationFormat: "protobuf",
		},
		Source: map[string]string{
			"local-root":   googleapisPath,
			"include-list": includeList,
		},
	}

	// We use `parser.CreateModel` instead of calling the individual parsing and processing
	// functions directly because CreateModel is the designated entry point that ensures
	// the API model is not only parsed but also fully linked (cross-referenced), validated,
	// and processed with all necessary configuration overrides. This guarantees a complete
	// and consistent model for the generator without code duplication.
	model, err := parser.CreateModel(parserConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create API model: %w", err)
	}
	return model, nil
}

// readGcloudConfig loads the gcloud configuration from a gcloud.yaml file.
func readGcloudConfig(path string) (*gcloud.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read gcloud config file: %w", err)
	}

	var cfg gcloud.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse gcloud config YAML: %w", err)
	}
	return &cfg, nil
}
