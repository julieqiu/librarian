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
	"fmt"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language"
	"github.com/urfave/cli/v3"
)

func releaseCommand() *cli.Command {
	return &cli.Command{
		Name:      "release",
		Usage:     "bump versions for release",
		UsageText: "librarian release [name] [--all] [--execute]",
		Description: `Bump versions and create release artifacts.

Specify a library name to release a single library, or use --all to release all libraries.

By default, this is a dry run. Use --execute to create and push git tags.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "release all libraries",
			},
			&cli.BoolFlag{
				Name:  "execute",
				Usage: "create and push git tags (default is dry run)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			var name string
			if cmd.Args().Len() > 0 {
				name = cmd.Args().Get(0)
			}
			return runRelease(ctx, name, cmd.Bool("all"), cmd.Bool("execute"))
		},
	}
}

func runRelease(ctx context.Context, name string, all bool, execute bool) error {
	if name != "" && all {
		return fmt.Errorf("cannot specify both library name and --all flag")
	}
	if name == "" && !all {
		return fmt.Errorf("must specify either a library name or --all flag")
	}

	cfg, err := config.Read(librarianConfigPath)
	if err != nil {
		return err
	}

	if name != "" {
		fmt.Printf("Bumping version for %s...\n", name)
	} else {
		fmt.Println("Bumping versions...")
	}

	cfg, err = language.Release(ctx, cfg, name)
	if err != nil {
		return err
	}

	if err := cfg.Write(librarianConfigPath); err != nil {
		return err
	}
	fmt.Println("✓ Updated release artifacts.")

	if !execute {
		fmt.Println("\nDry run complete. Run with --execute to create and push tags.")
		return nil
	}

	// TODO(https://github.com/googleapis/librarian/issues/2966): implement
	// --execute mode
	return nil
}
