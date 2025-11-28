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

package rust

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
)

const (
	googleapisRepo = "github.com/googleapis/googleapis"
	discoveryRepo  = "github.com/googleapis/discovery-artifact-manager"
)

// Generate generates a Rust client library.
func Generate(ctx context.Context, library *config.Library, sources *config.Sources) error {
	if err := cleanOutput(library.Output); err != nil {
		return err
	}
	googleapisDir, err := sourceDir(sources.Googleapis, googleapisRepo)
	if err != nil {
		return err
	}
	discoveryDir, err := sourceDir(sources.Discovery, discoveryRepo)
	if err != nil {
		return err
	}
	for _, api := range library.APIs {
		sidekickConfig := toSidekickConfig(library, api, googleapisDir, discoveryDir)
		model, err := parser.CreateModel(sidekickConfig)
		if err != nil {
			return err
		}
		if err := sidekickrust.Generate(model, library.Output, sidekickConfig); err != nil {
			return err
		}
	}
	if err := command.Run("taplo", "fmt", filepath.Join(library.Output, "Cargo.toml")); err != nil {
		return err
	}
	rsFiles, err := filepath.Glob(filepath.Join(library.Output, "src", "*.rs"))
	if err != nil {
		return err
	}
	if len(rsFiles) > 0 {
		// rustfmt defaults to 2015 edition when run directly on files. Specify
		// 2024 to match the edition in Cargo.toml.
		args := append([]string{"--edition", "2024"}, rsFiles...)
		if err := command.Run("rustfmt", args...); err != nil {
			return err
		}
	}
	return nil
}

// cleanOutput removes all files and directories in dir except Cargo.toml.
func cleanOutput(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() == "Cargo.toml" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func sourceDir(source *config.Source, repo string) (string, error) {
	if source == nil {
		return "", nil
	}
	if source.Dir != "" {
		return source.Dir, nil
	}
	return fetch.RepoDir(repo, source.Commit, source.SHA256)
}
