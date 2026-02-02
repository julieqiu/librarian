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

// Package golang provides functionality for generating and releasing Go client
// libraries.
package golang

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
)

// Generate generates a Go client library from protobuf definitions.
func Generate(ctx context.Context, library *config.Library, googleapisDir string) error {
	moduleRoot := filepath.Join(library.Output, library.Name)
	if err := os.MkdirAll(moduleRoot, 0755); err != nil {
		return fmt.Errorf("create module directory: %w", err)
	}

	// Generate code from protobuf definitions.
	if err := runProtoc(ctx, library, googleapisDir); err != nil {
		return fmt.Errorf("run protoc: %w", err)
	}

	// Reorganize output into expected layout.
	if err := moveToRoot(library.Output); err != nil {
		return fmt.Errorf("move to root: %w", err)
	}
	if err := applyModuleVersion(library.Output, library.Name, modulePath(library)); err != nil {
		return fmt.Errorf("apply module version: %w", err)
	}
	if library.Go != nil && len(library.Go.DeleteGenerationOutputPaths) > 0 {
		if err := cleanup(library.Output, library.Go.DeleteGenerationOutputPaths); err != nil {
			return fmt.Errorf("cleanup: %w", err)
		}
	}

	// Write module files.
	if err := writeReadme(library, googleapisDir); err != nil {
		return fmt.Errorf("write readme: %w", err)
	}
	if err := writeVersionFile(moduleRoot, library.Version); err != nil {
		return fmt.Errorf("write version file: %w", err)
	}

	isNewLibrary := !dirExists(library.Output)
	if isNewLibrary {
		if err := configureSnippets(ctx, library); err != nil {
			return fmt.Errorf("configure snippets: %w", err)
		}
	}

	for _, api := range library.APIs {
		if err := writeClientVersion(library, api.Path); err != nil {
			return fmt.Errorf("write client version for %s: %w", api.Path, err)
		}
	}
	return nil
}

// runProtoc invokes protoc for each API in the library.
//
// TODO(https://github.com/googleapis/librarian/issues/3617): implement this function
func runProtoc(_ context.Context, _ *config.Library, _ string) error {
	return nil
}

// moveToRoot moves generated files from cloud.google.com/go/ to the output root.
//
// TODO(https://github.com/googleapis/librarian/issues/3617): implement this function
func moveToRoot(_ string) error {
	return nil
}

// applyModuleVersion reorganizes the (already flattened) output directory
// appropriately for versioned modules. For a module path of the form
// cloud.google.com/go/{module-id}/{version}, we expect to find
// /output/{id}/{version} and /output/internal/generated/snippets/{module-id}/{version}.
// In most cases, we only support a single major version of the module, rooted at
// /{module-id} in the repository, so the content of these directories are moved into
// /output/{module-id} and /output/internal/generated/snippets/{id}.
//
// However, when we need to support multiple major versions, we use {module-id}/{version}
// as the *library* ID (in the state file etc). That indicates that the module is rooted
// in that versioned directory (e.g. "pubsub/v2"). In that case, the flattened code is
// already in the right place, so this function doesn't need to do anything.
//
// TODO(https://github.com/googleapis/librarian/issues/3617): implement this function
func applyModuleVersion(_, _, _ string) error {
	return nil
}

// configureSnippets adds a go.mod replace directive for local development.
//
// TODO(https://github.com/googleapis/librarian/issues/3617): implement this function
func configureSnippets(_ context.Context, _ *config.Library) error {
	return nil
}

// writeReadme creates the module's README.md.
//
// TODO(https://github.com/googleapis/librarian/issues/3617): implement this function
func writeReadme(_ *config.Library, _ string) error {
	return nil
}

// writeVersionFile creates internal/version.go with the module version.
//
// TODO(https://github.com/googleapis/librarian/issues/3617): implement this function
func writeVersionFile(_, _ string) error {
	return nil
}

// writeClientVersion creates version info for a specific API client.
//
// TODO(https://github.com/googleapis/librarian/issues/3617): implement this function
func writeClientVersion(_ *config.Library, _ string) error {
	return nil
}

func dirExists(dir string) bool {
	_, err := os.Stat(dir)
	return !os.IsNotExist(err)
}

func modulePath(library *config.Library) string {
	path := "cloud.google.com/go/" + library.Name
	if library.Go != nil && library.Go.ModulePathVersion != "" {
		path += "/" + library.Go.ModulePathVersion
	}
	return path
}

// cleanup removes unwanted paths from the output directory.
//
// TODO(https://github.com/googleapis/librarian/issues/3617): implement this function
func cleanup(_ string, _ []string) error {
	return nil
}
