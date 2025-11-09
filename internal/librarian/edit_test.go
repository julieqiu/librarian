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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestEditRunner_Metadata(t *testing.T) {
	// Create temp directory and init repository
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Init python repository and add artifact with APIs
	initRunner, err := newInitRunner([]string{"python"}, "python")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	addRunner, err := newAddRunner([]string{"packages/my-lib", "api/v1"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	// Edit metadata
	metadata := []string{"name_pretty=My Library", "release_level=stable"}
	editRunner, err := newEditRunner([]string{"packages/my-lib"}, metadata, "", nil, nil, nil)
	if err != nil {
		t.Fatalf("newEditRunner() error = %v", err)
	}
	if err := editRunner.run(context.Background()); err != nil {
		t.Fatalf("edit run() error = %v", err)
	}

	// Verify metadata was set
	artifactPath := filepath.Join(tmpDir, "packages/my-lib")
	artifactState, err := config.ReadArtifactState(artifactPath)
	if err != nil {
		t.Fatalf("failed to read artifact state: %v", err)
	}

	if artifactState.Generate.Metadata == nil {
		t.Fatal("metadata should be set")
	}
	if artifactState.Generate.Metadata.NamePretty != "My Library" {
		t.Errorf("metadata.name_pretty = %q, want %q", artifactState.Generate.Metadata.NamePretty, "My Library")
	}
	if artifactState.Generate.Metadata.ReleaseLevel != "stable" {
		t.Errorf("metadata.release_level = %q, want %q", artifactState.Generate.Metadata.ReleaseLevel, "stable")
	}
}

func TestEditRunner_Language(t *testing.T) {
	// Create temp directory and init repository
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Init python repository and add artifact
	initRunner, err := newInitRunner([]string{"python"}, "python")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	addRunner, err := newAddRunner([]string{"packages/my-lib", "api/v1"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	// Edit language metadata
	editRunner, err := newEditRunner([]string{"packages/my-lib"}, nil, "python:package=my-package", nil, nil, nil)
	if err != nil {
		t.Fatalf("newEditRunner() error = %v", err)
	}
	if err := editRunner.run(context.Background()); err != nil {
		t.Fatalf("edit run() error = %v", err)
	}

	// Verify language metadata was set
	artifactPath := filepath.Join(tmpDir, "packages/my-lib")
	artifactState, err := config.ReadArtifactState(artifactPath)
	if err != nil {
		t.Fatalf("failed to read artifact state: %v", err)
	}

	if artifactState.Generate.Language == nil {
		t.Fatal("language should be set")
	}
	if pkg, ok := artifactState.Generate.Language["package"]; !ok || pkg != "my-package" {
		t.Errorf("language[package] = %q, want %q", pkg, "my-package")
	}
}

func TestEditRunner_KeepRemoveExclude(t *testing.T) {
	// Create temp directory and init repository
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Init repository and add artifact
	initRunner, err := newInitRunner([]string{"python"}, "python")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	addRunner, err := newAddRunner([]string{"packages/my-lib", "api/v1"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	// Edit keep/remove/exclude lists
	keep := []string{"README.md", "docs/"}
	remove := []string{"temp.txt"}
	exclude := []string{"tests/", ".gitignore"}
	editRunner, err := newEditRunner([]string{"packages/my-lib"}, nil, "", keep, remove, exclude)
	if err != nil {
		t.Fatalf("newEditRunner() error = %v", err)
	}
	if err := editRunner.run(context.Background()); err != nil {
		t.Fatalf("edit run() error = %v", err)
	}

	// Verify lists were set
	artifactPath := filepath.Join(tmpDir, "packages/my-lib")
	artifactState, err := config.ReadArtifactState(artifactPath)
	if err != nil {
		t.Fatalf("failed to read artifact state: %v", err)
	}

	if diff := cmp.Diff(keep, artifactState.Generate.Keep); diff != "" {
		t.Errorf("keep list mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(remove, artifactState.Generate.Remove); diff != "" {
		t.Errorf("remove list mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(exclude, artifactState.Generate.Exclude); diff != "" {
		t.Errorf("exclude list mismatch (-want +got):\n%s", diff)
	}
}

func TestEditRunner_InvalidMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Init and add artifact
	initRunner, err := newInitRunner([]string{"python"}, "python")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	addRunner, err := newAddRunner([]string{"packages/my-lib", "api/v1"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	// Try to set invalid release_level
	metadata := []string{"release_level=invalid"}
	editRunner, err := newEditRunner([]string{"packages/my-lib"}, metadata, "", nil, nil, nil)
	if err != nil {
		t.Fatalf("newEditRunner() error = %v", err)
	}

	err = editRunner.run(context.Background())
	if err == nil {
		t.Error("edit run() should return error for invalid release_level")
	}
}

func TestEditRunner_MissingPath(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	_, err = newEditRunner([]string{}, nil, "", nil, nil, nil)
	if err == nil {
		t.Error("newEditRunner() should return error when path is missing")
	}
}

func TestEditRunner_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	editRunner, err := newEditRunner([]string{"packages/nonexistent"}, nil, "", nil, nil, nil)
	if err != nil {
		t.Fatalf("newEditRunner() error = %v", err)
	}

	err = editRunner.run(context.Background())
	if err == nil {
		t.Error("edit run() should return error when .librarian.yaml does not exist")
	}
}

func TestEditRunner_ReleaseOnlyArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Init release-only repository
	initRunner, err := newInitRunner([]string{}, "")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	// Add release-only artifact
	addRunner, err := newAddRunner([]string{"packages/my-tool"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	// Try to set metadata (should fail for release-only artifacts)
	metadata := []string{"name_pretty=My Tool"}
	editRunner, err := newEditRunner([]string{"packages/my-tool"}, metadata, "", nil, nil, nil)
	if err != nil {
		t.Fatalf("newEditRunner() error = %v", err)
	}

	err = editRunner.run(context.Background())
	if err == nil {
		t.Error("edit run() should return error when trying to set metadata on release-only artifact")
	}
}
