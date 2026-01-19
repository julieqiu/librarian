// Copyright 2026 Google LLC
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

package golang

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

// Format is the entrypoint for formatting generated files. It runs
// formatters and other tools to ensure code quality. The high-level steps are:
//
//  1. Modify the generated snippets to specify the current version
//  2. Run `goimports` to format the code.
//  3. For new modules only, run "go mod init" and "go mod tidy"
func Format(ctx context.Context, library *config.Library) error {
	outputDir := library.Output
	moduleDir := outputDir + "/" + library.Name

	if len(library.Channels) == 0 {
		return nil
	}

	if library.Version == "" {
		return fmt.Errorf("no version for library: %s (required for post-processing)", library.Name)
	}

	if err := updateSnippetsMetadata(library, outputDir, outputDir); err != nil {
		return fmt.Errorf("failed to update snippets metadata: %w", err)
	}

	if err := goimports(ctx, outputDir); err != nil {
		return fmt.Errorf("failed to run 'goimports': %w", err)
	}

	// If we have a single channel, and it's new, then this must be the first time generating this library.
	// We run go mod init and go mod tidy *only* this time. We can only run this once because once go.mod and go.sum have
	// been created, Librarian should refuse to copy it over unless the old version is deleted first...
	// and we *don't* want to run it every time (partly because generate shouldn't be updating dependencies,
	// and partly because there might be handwritten code in the library, which generate can't "see").
	// When configuring the first generated channel for a library, we assume the whole library is new.
	//
	// We can't even run "go mod init" from configure and just "go mod tidy" here, as files written
	// by the configure command aren't available during generate.
	isNewLibrary := len(library.Channels) == 1 && !dirExists(library.Output)
	if isNewLibrary {
		modPath := buildModulePath(library)
		if err := goModInit(ctx, moduleDir, modPath); err != nil {
			return fmt.Errorf("failed to run 'go mod init': %w", err)
		}
		if err := goModTidy(ctx, moduleDir); err != nil {
			return fmt.Errorf("failed to run 'go mod tidy': %w", err)
		}
	}

	return nil
}

// updateSnippetsMetadata updates all snippet files to populate the $VERSION placeholder, reading them from
// the sourceDir and writing them to the destDir. These two may be the same, but don't have to be.
func updateSnippetsMetadata(lib *config.Library, sourceDir string, destDir string) error {
	moduleName := lib.Name
	version := lib.Version

	snpDir := snippetsDir("", moduleName)

	for _, channel := range lib.Channels {
		clientDirName, err := clientDirectory(lib, channel.Path)
		if err != nil {
			return err
		}

		protoPkg := protoPackage(lib, channel.Path)
		snippetFile := "snippet_metadata." + protoPkg + ".json"
		path := filepath.Join(snpDir, clientDirName, snippetFile)
		read, err := os.ReadFile(filepath.Join(sourceDir, path))
		if err != nil {
			// If the snippet metadata doesn't exist, that's probably because this API path
			// is proto-only (so the GAPIC generator hasn't run). Continue to the next API path.
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}

		content := string(read)
		var newContent string

		if strings.Contains(content, "$VERSION") {
			newContent = strings.Replace(content, "$VERSION", version, 1)
		} else {
			// This regex finds a version string like "1.2.3".
			re := regexp.MustCompile(`\d+\.\d+\.\d+`)
			if foundVersion := re.FindString(content); foundVersion != "" {
				newContent = strings.Replace(content, foundVersion, version, 1)
			}
		}

		if newContent == "" {
			return fmt.Errorf("no version number or placeholder found in '%s'", snippetFile)
		}

		destPath := filepath.Join(destDir, path)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("creating directory for snippet file: %w", err)
		}
		err = os.WriteFile(destPath, []byte(newContent), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

// goimports runs the goimports tool on a directory to format Go files and
// manage imports.
func goimports(ctx context.Context, dir string) error {
	// The `.` argument will make goimports process all go files in the directory
	// and its subdirectories. The -w flag writes results back to source files.
	return command.RunInDir(ctx, dir, "goimports", "-w", ".")
}

// goModInit runs "go mod init" on a directory to initialize the module.
func goModInit(ctx context.Context, dir, modulePath string) error {
	return command.RunInDir(ctx, dir, "go", "mod", "init", modulePath)
}

// goModTidy runs "go mod tidy" on a directory to add appropriate dependencies.
func goModTidy(ctx context.Context, dir string) error {
	return command.RunInDir(ctx, dir, "go", "mod", "tidy")
}
