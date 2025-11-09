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
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestGenerateNewRunner_Single(t *testing.T) {
	t.Skip("Skipping integration test - requires Docker and network access")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Init python repository
	initRunner, err := newInitRunner([]string{"python"}, "python")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	// Add artifact with APIs
	addRunner, err := newAddRunner([]string{"packages/my-lib", "api/v1"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	// Generate the artifact
	generateRunner, err := newGenerateNewRunner([]string{"packages/my-lib"}, false, false)
	if err != nil {
		t.Fatalf("newGenerateNewRunner() error = %v", err)
	}
	if err := generateRunner.run(context.Background()); err != nil {
		t.Fatalf("generate run() error = %v", err)
	}

	// Verify artifact state was updated
	artifactState, err := config.ReadArtifactState(tmpDir + "/packages/my-lib")
	if err != nil {
		t.Fatalf("failed to read artifact state: %v", err)
	}

	if artifactState.Generate.Librarian == "" {
		t.Error("librarian version should be set after generation")
	}
	if artifactState.Generate.Commit == "" {
		t.Error("commit should be set after generation")
	}
}

func TestGenerateNewRunner_All(t *testing.T) {
	t.Skip("Skipping integration test - requires Docker and network access")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Init python repository
	initRunner, err := newInitRunner([]string{"python"}, "python")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	// Add multiple artifacts with APIs
	artifacts := []string{"packages/lib1", "packages/lib2", "packages/lib3"}
	for _, artifact := range artifacts {
		addRunner, err := newAddRunner([]string{artifact, "api/v1"}, false)
		if err != nil {
			t.Fatalf("newAddRunner() error = %v", err)
		}
		if err := addRunner.run(context.Background()); err != nil {
			t.Fatalf("add run() error = %v", err)
		}
	}

	// Add one release-only artifact (should be skipped)
	addRunner, err := newAddRunner([]string{"packages/tool"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	// Generate all artifacts
	generateRunner, err := newGenerateNewRunner([]string{}, true, false)
	if err != nil {
		t.Fatalf("newGenerateNewRunner() error = %v", err)
	}
	if err := generateRunner.run(context.Background()); err != nil {
		t.Fatalf("generate run() error = %v", err)
	}

	// Verify all generate artifacts were updated
	for _, artifact := range artifacts {
		artifactState, err := config.ReadArtifactState(tmpDir + "/" + artifact)
		if err != nil {
			t.Fatalf("failed to read artifact state for %s: %v", artifact, err)
		}

		if artifactState.Generate.Librarian == "" {
			t.Errorf("%s: librarian version should be set after generation", artifact)
		}
		if artifactState.Generate.Commit == "" {
			t.Errorf("%s: commit should be set after generation", artifact)
		}
	}

	// Verify release-only artifact was not affected
	toolState, err := config.ReadArtifactState(tmpDir + "/packages/tool")
	if err != nil {
		t.Fatalf("failed to read tool state: %v", err)
	}
	if toolState.Generate != nil {
		t.Error("release-only artifact should not have generate section")
	}
}

func TestGenerateNewRunner_MissingPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	_, err := newGenerateNewRunner([]string{}, false, false)
	if err == nil {
		t.Error("newGenerateNewRunner() should return error when path is missing and --all not specified")
	}
}

func TestGenerateNewRunner_BothPathAndAll(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	_, err := newGenerateNewRunner([]string{"packages/my-lib"}, true, false)
	if err == nil {
		t.Error("newGenerateNewRunner() should return error when both path and --all are specified")
	}
}

func TestGenerateNewRunner_NoGenerateSection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Init release-only repository
	initRunner, err := newInitRunner([]string{}, "")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	// Try to create generate runner (should fail)
	_, err = newGenerateNewRunner([]string{"packages/my-tool"}, false, false)
	if err == nil {
		t.Error("newGenerateNewRunner() should return error when repository doesn't support generation")
	}
}

func TestGenerateNewRunner_ArtifactNoGenerateSection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Init python repository
	initRunner, err := newInitRunner([]string{"python"}, "python")
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

	// Try to generate (should fail)
	generateRunner, err := newGenerateNewRunner([]string{"packages/my-tool"}, false, false)
	if err != nil {
		t.Fatalf("newGenerateNewRunner() error = %v", err)
	}
	err = generateRunner.run(context.Background())
	if err == nil {
		t.Error("generate run() should return error when artifact doesn't have generate section")
	}
}

func TestGenerateNewRunner_FindGeneratableArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Init python repository
	initRunner, err := newInitRunner([]string{"python"}, "python")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	// Add 2 generate artifacts and 1 release-only
	addRunner1, err := newAddRunner([]string{"packages/lib1", "api/v1"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner1.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	addRunner2, err := newAddRunner([]string{"packages/lib2", "api/v2"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner2.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	addRunner3, err := newAddRunner([]string{"packages/tool"}, false)
	if err != nil {
		t.Fatalf("newAddRunner() error = %v", err)
	}
	if err := addRunner3.run(context.Background()); err != nil {
		t.Fatalf("add run() error = %v", err)
	}

	// Find generatable artifacts
	generateRunner, err := newGenerateNewRunner([]string{}, true, false)
	if err != nil {
		t.Fatalf("newGenerateNewRunner() error = %v", err)
	}

	artifacts, err := generateRunner.findGeneratableArtifacts()
	if err != nil {
		t.Fatalf("findGeneratableArtifacts() error = %v", err)
	}

	if len(artifacts) != 2 {
		t.Errorf("findGeneratableArtifacts() found %d artifacts, want 2", len(artifacts))
	}
}
