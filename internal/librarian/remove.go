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
)

type removeRunner struct {
	path     string
	repoRoot string
}

func newRemoveRunner(args []string) (*removeRunner, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("missing required argument <path>")
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments, expected: librarian remove <path>")
	}

	path := args[0]

	// Get current working directory as repo root
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return &removeRunner{
		path:     path,
		repoRoot: repoRoot,
	}, nil
}

func (r *removeRunner) run(ctx context.Context) error {
	_ = ctx
	artifactPath := filepath.Join(r.repoRoot, r.path)
	configPath := filepath.Join(artifactPath, ".librarian.yaml")

	// Check if .librarian.yaml exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf(".librarian.yaml does not exist at %s", configPath)
	}

	// Remove the file
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to remove .librarian.yaml: %w", err)
	}

	slog.Info("removed .librarian.yaml", "path", configPath)
	fmt.Printf("Removed %s from librarian management\n", r.path)
	fmt.Println("Note: Source code was not modified")

	return nil
}
