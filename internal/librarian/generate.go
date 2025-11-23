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

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate a client library",
		UsageText: "librarian generate <name>",
		Description: `Generate a client library from googleapis.

For Rust (channel-level libraries):
  librarian generate google-cloud-secretmanager-v1`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return fmt.Errorf("generate requires a library name argument")
			}
			libraryName := cmd.Args().Get(0)

			cfg, err := config.Read("librarian.yaml")
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			if cfg.Language == "" {
				return fmt.Errorf("language must be set in librarian.yaml")
			}

			if cfg.Default == nil || cfg.Default.Generate == nil || cfg.Default.Generate.OneLibraryPer == "" {
				return fmt.Errorf("one_library_per must be set in librarian.yaml under default.generate.one_library_per")
			}

			if cfg.Sources == nil || cfg.Sources.Googleapis == nil {
				return fmt.Errorf("no googleapis source configured in librarian.yaml")
			}

			commit := cfg.Sources.Googleapis.Commit
			if commit == "" {
				return fmt.Errorf("no commit specified for googleapis source in librarian.yaml")
			}

			googleapisDir, err := googleapisDir(commit)
			if err != nil {
				return err
			}

			// Find library by name
			var library *config.Library
			for _, lib := range cfg.Libraries {
				if lib.Name == libraryName {
					library = lib
					break
				}
			}
			if library == nil {
				return fmt.Errorf("library %q not found in librarian.yaml", libraryName)
			}

			// Generate the library
			if err := Generate(ctx, cfg.Default.Generate.OneLibraryPer, cfg.Language, library, googleapisDir); err != nil {
				return fmt.Errorf("failed to generate library: %w", err)
			}

			fmt.Printf("✓ Successfully generated %s\n", libraryName)
			return nil
		},
	}
}

// Generate generates a library.
func Generate(ctx context.Context, oneLibraryPer, lang string, library *config.Library, googleapisDir string) error {
	return language.Generate(ctx, lang, library, googleapisDir)
}
