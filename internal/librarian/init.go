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
	"path/filepath"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
)

const (
	defaultGenerateDir = "packages/"
	defaultTagFormat   = "{name}-v{version}"
)

// Default container images for each language
var defaultContainerImages = map[string]struct{ image, tag string }{
	"go": {
		image: "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/go-librarian-generator",
		tag:   "latest",
	},
	"python": {
		image: "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/python-librarian-generator",
		tag:   "latest",
	},
	"rust": {
		image: "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/rust-librarian-generator",
		tag:   "latest",
	},
	"dart": {
		image: "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/dart-librarian-generator",
		tag:   "latest",
	},
}

type initRunner struct {
	language string
	repoRoot string
}

func newInitRunner(args []string, language string) (*initRunner, error) {
	// Get current working directory as repo root
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// If language provided, validate it
	if language != "" {
		validLanguages := []string{"go", "python", "rust", "dart"}
		valid := false
		for _, lang := range validLanguages {
			if language == lang {
				valid = true
				break
			}
		}
		if !valid {
			return nil, fmt.Errorf("invalid language %q, must be one of: go, python, rust, dart", language)
		}
	}

	return &initRunner{
		language: language,
		repoRoot: repoRoot,
	}, nil
}

func (r *initRunner) run(ctx context.Context) error {
	configPath := filepath.Join(r.repoRoot, ".librarian.yaml")

	// Check if .librarian.yaml already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf(".librarian.yaml already exists at %s", configPath)
	}

	// Create the LibrarianConfig
	librarianConfig := &config.LibrarianConfig{
		Librarian: config.LibrarianSection{
			Version: cli.Version(),
		},
		Release: &config.ReleaseSection{
			TagFormat: defaultTagFormat,
		},
	}

	// Add language and generate section if language is specified
	if r.language != "" {
		librarianConfig.Librarian.Language = r.language

		containerInfo, ok := defaultContainerImages[r.language]
		if !ok {
			return fmt.Errorf("no default container image configured for language %q", r.language)
		}

		librarianConfig.Generate = &config.GenerateSection{
			Container: config.ContainerConfig{
				Image: containerInfo.image,
				Tag:   containerInfo.tag,
			},
			Googleapis: config.RepositoryRef{
				Repo: "github.com/googleapis/googleapis",
				// Ref is intentionally omitted - will be set during first 'add' command
			},
			Dir: defaultGenerateDir,
		}

		// Add Discovery for Python and Go
		if r.language == "python" || r.language == "go" {
			librarianConfig.Generate.Discovery = &config.RepositoryRef{
				Repo: "github.com/googleapis/discovery-artifact-manager",
				// Ref is intentionally omitted
			}
		}
	}

	// Validate the config
	if err := librarianConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Write the config file
	if err := config.WriteLibrarianConfig(r.repoRoot, librarianConfig); err != nil {
		return fmt.Errorf("failed to write .librarian.yaml: %w", err)
	}

	slog.Info("created .librarian.yaml", "path", configPath)

	if r.language != "" {
		fmt.Printf("Initialized repository for %s with generation and release support\n", r.language)
		fmt.Printf("Created .librarian.yaml with:\n")
		fmt.Printf("  - librarian.version: %s\n", cli.Version())
		fmt.Printf("  - librarian.language: %s\n", r.language)
		fmt.Printf("  - generate section with container image\n")
		fmt.Printf("  - release section with tag format: %s\n", defaultTagFormat)
	} else {
		fmt.Println("Initialized repository for release management only")
		fmt.Printf("Created .librarian.yaml with:\n")
		fmt.Printf("  - librarian.version: %s\n", cli.Version())
		fmt.Printf("  - release section with tag format: %s\n", defaultTagFormat)
	}

	return nil
}
