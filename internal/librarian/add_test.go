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

	"github.com/googleapis/librarian/internal/config"
)

func TestAddRunner_ReleaseOnly(t *testing.T) {
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

	// Init release-only repository
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

	// Verify artifact config
	artifactPath := filepath.Join(tmpDir, "packages/my-tool")
	artifactState, err := config.ReadArtifactState(artifactPath)
	if err != nil {
		t.Fatalf("failed to read artifact state: %v", err)
	}

	if artifactState.Generate != nil {
		t.Error("generate section should not exist for release-only artifact")
	}
	if artifactState.Release == nil {
		t.Fatal("release section should exist")
	}
	if artifactState.Release.Version != nil {
		t.Errorf("release.version = %v, want nil", artifactState.Release.Version)
	}
}

func TestAddRunner_WithAPIs(t *testing.T) {
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

	// Init python repository
	initRunner, err := newInitRunner([]string{"python"}, "python")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	// Add artifact with APIs
	apis := []string{"secretmanager/v1", "secretmanager/v1beta2"}
	addRunner, err := newAddRunner(append([]string{"packages/google-cloud-secret-manager"}, apis...), false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	// Verify artifact config
	artifactPath := filepath.Join(tmpDir, "packages/google-cloud-secret-manager")
	artifactState, err := config.ReadArtifactState(artifactPath)
	if err != nil {
		t.Fatalf("failed to read artifact state: %v", err)
	}

	if artifactState.Generate == nil {
		t.Fatal("generate section should exist")
	}
	if len(artifactState.Generate.APIs) != len(apis) {
		t.Errorf("len(APIs) = %d, want %d", len(artifactState.Generate.APIs), len(apis))
	}
	for i, api := range apis {
		if artifactState.Generate.APIs[i].Path != api {
			t.Errorf("APIs[%d].Path = %q, want %q", i, artifactState.Generate.APIs[i].Path, api)
		}
	}

	if artifactState.Release == nil {
		t.Fatal("release section should exist")
	}
}

func TestAddRunner_MissingPath(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	_, err = newAddRunner([]string{}, false)
	if err == nil {
		t.Error("newAddRunner() should return error when path is missing")
	}
}

func TestAddRunner_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Init repository
	initRunner, err := newInitRunner([]string{}, "")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	// Add artifact first time
	addRunner, err := newAddRunner([]string{"packages/my-tool"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	// Try to add again
	addRunner2, err := newAddRunner([]string{"packages/my-tool"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	err = addRunner2.run(context.Background())
	if err == nil {
		t.Error("add run() should return error when .librarian.yaml already exists")
	}
}

func TestAddRunner_NoRepositoryConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Don't init repository
	_, err = newAddRunner([]string{"packages/my-tool"}, false)
	if err == nil {
		t.Error("newAddRunner() should return error when repository is not initialized")
	}
}
