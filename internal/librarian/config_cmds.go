// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
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
	"log/slog"
	"os"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

type configGetRunner struct {
	key      string
	repoRoot string
}

func newConfigGetRunner(args []string) (*configGetRunner, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("missing required argument <key>")
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments, expected: librarian config get <key>")
	}

	// Get current working directory as repo root
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return &configGetRunner{
		key:      args[0],
		repoRoot: repoRoot,
	}, nil
}

func (r *configGetRunner) run(ctx context.Context) error {
	_ = ctx
	// Read repository config
	librarianConfig, err := config.ReadLibrarianConfig(r.repoRoot)
	if err != nil {
		return fmt.Errorf("failed to read .librarian.yaml: %w", err)
	}

	// Get the value for the key
	value, err := getConfigValue(librarianConfig, r.key)
	if err != nil {
		return err
	}

	fmt.Println(value)
	return nil
}

type configSetRunner struct {
	key      string
	value    string
	repoRoot string
}

func newConfigSetRunner(args []string) (*configSetRunner, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("missing required arguments, expected: librarian config set <key> <value>")
	}
	if len(args) > 2 {
		return nil, fmt.Errorf("too many arguments, expected: librarian config set <key> <value>")
	}

	// Get current working directory as repo root
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return &configSetRunner{
		key:      args[0],
		value:    args[1],
		repoRoot: repoRoot,
	}, nil
}

func (r *configSetRunner) run(ctx context.Context) error {
	_ = ctx
	// Read repository config
	librarianConfig, err := config.ReadLibrarianConfig(r.repoRoot)
	if err != nil {
		return fmt.Errorf("failed to read .librarian.yaml: %w", err)
	}

	// Set the value
	if err := setConfigValue(librarianConfig, r.key, r.value); err != nil {
		return err
	}

	// Write back
	if err := config.WriteLibrarianConfig(r.repoRoot, librarianConfig); err != nil {
		return fmt.Errorf("failed to write .librarian.yaml: %w", err)
	}

	slog.Info("updated .librarian.yaml", "key", r.key, "value", r.value)
	fmt.Printf("Set %s = %s\n", r.key, r.value)
	return nil
}

type configUpdateRunner struct {
	key      string
	all      bool
	repoRoot string
}

func newConfigUpdateRunner(args []string, all bool) (*configUpdateRunner, error) {
	if !all && len(args) < 1 {
		return nil, fmt.Errorf("missing required argument <key> or --all flag")
	}
	if all && len(args) > 0 {
		return nil, fmt.Errorf("cannot specify both <key> and --all flag")
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments, expected: librarian config update [key]")
	}

	var key string
	if len(args) > 0 {
		key = args[0]
	}

	// Get current working directory as repo root
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return &configUpdateRunner{
		key:      key,
		all:      all,
		repoRoot: repoRoot,
	}, nil
}

func (r *configUpdateRunner) run(ctx context.Context) error {
	_ = ctx
	// Read repository config
	librarianConfig, err := config.ReadLibrarianConfig(r.repoRoot)
	if err != nil {
		return fmt.Errorf("failed to read .librarian.yaml: %w", err)
	}

	// Determine which keys to update
	var keysToUpdate []string
	if r.all {
		keysToUpdate = []string{"generate.container", "generate.googleapis", "generate.discovery"}
	} else {
		keysToUpdate = []string{r.key}
	}

	// Update each key
	updated := false
	for _, key := range keysToUpdate {
		// Skip if the section doesn't exist
		if strings.HasPrefix(key, "generate.") && librarianConfig.Generate == nil {
			if r.all {
				continue // Skip silently for --all
			}
			return fmt.Errorf("generate section does not exist in .librarian.yaml")
		}

		switch key {
		case "generate.container":
			// TODO: Fetch latest container tag from registry
			fmt.Printf("Updating %s to latest... (not yet implemented)\n", key)
			slog.Warn("update to latest not yet implemented", "key", key)

		case "generate.googleapis":
			// TODO: Fetch latest commit from googleapis
			fmt.Printf("Updating %s to latest... (not yet implemented)\n", key)
			slog.Warn("update to latest not yet implemented", "key", key)

		case "generate.discovery":
			// TODO: Fetch latest commit from discovery-artifact-manager
			if librarianConfig.Generate.Discovery != nil {
				fmt.Printf("Updating %s to latest... (not yet implemented)\n", key)
				slog.Warn("update to latest not yet implemented", "key", key)
			}

		default:
			return fmt.Errorf("unsupported key for update: %s (supported: generate.container, generate.googleapis, generate.discovery)", key)
		}
	}

	if updated {
		// Write back
		if err := config.WriteLibrarianConfig(r.repoRoot, librarianConfig); err != nil {
			return fmt.Errorf("failed to write .librarian.yaml: %w", err)
		}
		fmt.Println("Updated .librarian.yaml")
	}

	return nil
}

// getConfigValue retrieves a configuration value by key path.
func getConfigValue(cfg *config.LibrarianConfig, key string) (string, error) {
	parts := strings.Split(key, ".")

	switch parts[0] {
	case "librarian":
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid key: %s", key)
		}
		switch parts[1] {
		case "version":
			return cfg.Librarian.Version, nil
		case "language":
			return cfg.Librarian.Language, nil
		default:
			return "", fmt.Errorf("unknown librarian field: %s", parts[1])
		}

	case "generate":
		if cfg.Generate == nil {
			return "", fmt.Errorf("generate section does not exist")
		}
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid key: %s", key)
		}

		switch parts[1] {
		case "dir":
			return cfg.Generate.Dir, nil
		case "container":
			if len(parts) == 2 {
				return fmt.Sprintf("%s:%s", cfg.Generate.Container.Image, cfg.Generate.Container.Tag), nil
			}
			switch parts[2] {
			case "image":
				return cfg.Generate.Container.Image, nil
			case "tag":
				return cfg.Generate.Container.Tag, nil
			default:
				return "", fmt.Errorf("unknown container field: %s", parts[2])
			}
		case "googleapis":
			if len(parts) == 2 {
				return cfg.Generate.Googleapis.Path, nil
			}
			switch parts[2] {
			case "repo":
				return cfg.Generate.Googleapis.Path, nil
			case "ref":
				return cfg.Generate.Googleapis.Ref, nil
			default:
				return "", fmt.Errorf("unknown googleapis field: %s", parts[2])
			}
		case "discovery":
			if cfg.Generate.Discovery == nil {
				return "", fmt.Errorf("discovery section does not exist")
			}
			if len(parts) == 2 {
				return cfg.Generate.Discovery.Path, nil
			}
			switch parts[2] {
			case "repo":
				return cfg.Generate.Discovery.Path, nil
			case "ref":
				return cfg.Generate.Discovery.Ref, nil
			default:
				return "", fmt.Errorf("unknown discovery field: %s", parts[2])
			}
		default:
			return "", fmt.Errorf("unknown generate field: %s", parts[1])
		}

	case "release":
		if cfg.Release == nil {
			return "", fmt.Errorf("release section does not exist")
		}
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid key: %s", key)
		}
		switch parts[1] {
		case "tag_format":
			return cfg.Release.TagFormat, nil
		default:
			return "", fmt.Errorf("unknown release field: %s", parts[1])
		}

	default:
		return "", fmt.Errorf("unknown config section: %s", parts[0])
	}
}

// setConfigValue sets a configuration value by key path.
func setConfigValue(cfg *config.LibrarianConfig, key, value string) error {
	parts := strings.Split(key, ".")

	switch parts[0] {
	case "librarian":
		if len(parts) < 2 {
			return fmt.Errorf("invalid key: %s", key)
		}
		switch parts[1] {
		case "language":
			cfg.Librarian.Language = value
		default:
			return fmt.Errorf("cannot set librarian.%s", parts[1])
		}

	case "generate":
		if cfg.Generate == nil {
			return fmt.Errorf("generate section does not exist (run 'librarian init <language>' first)")
		}
		if len(parts) < 2 {
			return fmt.Errorf("invalid key: %s", key)
		}

		switch parts[1] {
		case "dir":
			cfg.Generate.Dir = value
		case "container":
			if len(parts) == 2 {
				// Syntactic sugar: parse image:tag
				colonIdx := strings.LastIndex(value, ":")
				if colonIdx == -1 {
					return fmt.Errorf("invalid container format, expected image:tag")
				}
				cfg.Generate.Container.Image = value[:colonIdx]
				cfg.Generate.Container.Tag = value[colonIdx+1:]
			} else if len(parts) == 3 {
				switch parts[2] {
				case "image":
					cfg.Generate.Container.Image = value
				case "tag":
					cfg.Generate.Container.Tag = value
				default:
					return fmt.Errorf("unknown container field: %s", parts[2])
				}
			} else {
				return fmt.Errorf("invalid key: %s", key)
			}
		case "googleapis":
			if len(parts) < 3 {
				return fmt.Errorf("invalid key: %s", key)
			}
			switch parts[2] {
			case "repo":
				cfg.Generate.Googleapis.Path = value
			case "ref":
				cfg.Generate.Googleapis.Ref = value
			default:
				return fmt.Errorf("unknown googleapis field: %s", parts[2])
			}
		case "discovery":
			if cfg.Generate.Discovery == nil {
				cfg.Generate.Discovery = &config.RepositoryRef{}
			}
			if len(parts) < 3 {
				return fmt.Errorf("invalid key: %s", key)
			}
			switch parts[2] {
			case "repo":
				cfg.Generate.Discovery.Path = value
			case "ref":
				cfg.Generate.Discovery.Ref = value
			default:
				return fmt.Errorf("unknown discovery field: %s", parts[2])
			}
		default:
			return fmt.Errorf("unknown generate field: %s", parts[1])
		}

	case "release":
		if cfg.Release == nil {
			return fmt.Errorf("release section does not exist")
		}
		if len(parts) < 2 {
			return fmt.Errorf("invalid key: %s", key)
		}
		switch parts[1] {
		case "tag_format":
			cfg.Release.TagFormat = value
		default:
			return fmt.Errorf("unknown release field: %s", parts[1])
		}

	default:
		return fmt.Errorf("unknown config section: %s", parts[0])
	}

	return nil
}
