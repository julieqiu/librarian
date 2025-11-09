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
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveRunner(t *testing.T) {
	// Create temp directory and init repository
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Init repository
	initRunner, err := newInitRunner([]string{}, "")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	// Add artifact
	addRunner, err := newAddRunner([]string{"packages/my-tool"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	// Verify .librarian.yaml exists
	configPath := filepath.Join(tmpDir, "packages/my-tool", ".librarian.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf(".librarian.yaml should exist before remove: %v", err)
	}

	// Remove artifact
	removeRunner, err := newRemoveRunner([]string{"packages/my-tool"})
	if err != nil {
		t.Fatalf("newRemoveRunner() error = %v", err)
	}
	if err := removeRunner.run(context.Background()); err != nil {
		t.Fatalf("remove run() error = %v", err)
	}

	// Verify .librarian.yaml is removed
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error(".librarian.yaml should be removed")
	}

	// Verify directory still exists
	dirPath := filepath.Join(tmpDir, "packages/my-tool")
	if _, err := os.Stat(dirPath); err != nil {
		t.Error("directory should still exist after remove")
	}
}

func TestRemoveRunner_MissingPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	_, err := newRemoveRunner([]string{})
	if err == nil {
		t.Error("newRemoveRunner() should return error when path is missing")
	}
}

func TestRemoveRunner_TooManyArgs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	_, err := newRemoveRunner([]string{"path1", "path2"})
	if err == nil {
		t.Error("newRemoveRunner() should return error when too many arguments")
	}
}

func TestRemoveRunner_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	removeRunner, err := newRemoveRunner([]string{"packages/nonexistent"})
	if err != nil {
		t.Fatalf("newRemoveRunner() error = %v", err)
	}

	err = removeRunner.run(context.Background())
	if err == nil {
		t.Error("remove run() should return error when .librarian.yaml does not exist")
	}
}
