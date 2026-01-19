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

package librarianops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
)

func TestGenerateCommand(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, repoFake)
	if err := os.Mkdir(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	setupGitRepo(t, repoDir)
	if err := command.Run(t.Context(), "git", "-C", repoDir, "checkout", "-b", "main"); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	googleapisDir := filepath.Join(wd, "..", "testdata", "googleapis")
	configContent := fmt.Sprintf(`language: fake
sources:
  googleapis:
    dir: %s
libraries:
  - name: test-library
    output: output
    channels:
      - path: google/cloud/secretmanager/v1
`, googleapisDir)
	if err := os.WriteFile(filepath.Join(repoDir, "librarian.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatal(err)
	}

	args := []string{"librarianops", "generate", "-C", repoDir}
	if err := Run(t.Context(), args...); err != nil {
		t.Fatal(err)
	}
	readmePath := filepath.Join(repoDir, "output", "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		t.Errorf("expected README.md to be generated: %v", err)
	}
}

func TestGenerateCommand_Errors(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
	}{
		{
			name: "both repo and all flag",
			args: []string{"librarianops", "generate", "--all", repoFake},
		},
		{
			name: "neither repo nor all flag",
			args: []string{"librarianops", "generate"},
		},
		{
			name: "all flag with C flag",
			args: []string{"librarianops", "generate", "--all", "-C", "/tmp/foo"},
		},
		{
			name: "unsupported repo",
			args: []string{"librarianops", "generate", "unsupported-repo"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Run(t.Context(), test.args...)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestGetRepoConfig(t *testing.T) {
	for _, test := range []struct {
		name string
		repo string
		want repoConfig
	}{
		{
			name: "google-cloud-rust",
			repo: repoRust,
			want: repoConfig{
				updateDiscovery: true,
				runCargoUpdate:  true,
			},
		},
		{
			name: "fake-repo",
			repo: repoFake,
			want: repoConfig{
				updateDiscovery: false,
				runCargoUpdate:  false,
			},
		},
		{
			name: "unknown repo",
			repo: "unknown-repo",
			want: repoConfig{
				updateDiscovery: false,
				runCargoUpdate:  false,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := getRepoConfig(test.repo)
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(repoConfig{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateCommand_InferRepoName(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, repoFake)
	if err := os.Mkdir(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	setupGitRepo(t, repoDir)
	if err := command.Run(t.Context(), "git", "-C", repoDir, "checkout", "-b", "main"); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	googleapisDir := filepath.Join(wd, "..", "testdata", "googleapis")
	configContent := fmt.Sprintf(`language: fake
sources:
  googleapis:
    dir: %s
libraries:
  - name: test-library
    output: output
    channels:
      - path: google/cloud/secretmanager/v1
`, googleapisDir)
	if err := os.WriteFile(filepath.Join(repoDir, "librarian.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatal(err)
	}

	args := []string{"librarianops", "generate", "-C", repoDir}
	if err := Run(t.Context(), args...); err != nil {
		t.Fatal(err)
	}
	readmePath := filepath.Join(repoDir, "output", "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		t.Errorf("expected README.md to be generated: %v", err)
	}
}

func TestGenerateCommand_InferRepoNameFromCurrentDir(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, repoFake)
	if err := os.Mkdir(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	setupGitRepo(t, repoDir)
	if err := command.Run(t.Context(), "git", "-C", repoDir, "checkout", "-b", "main"); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	googleapisDir := filepath.Join(wd, "..", "testdata", "googleapis")
	configContent := fmt.Sprintf(`language: fake
sources:
  googleapis:
    dir: %s
libraries:
  - name: test-library
    output: output
    channels:
      - path: google/cloud/secretmanager/v1
`, googleapisDir)
	if err := os.WriteFile(filepath.Join(repoDir, "librarian.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatal(err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWD)

	args := []string{"librarianops", "generate", "-C", "."}
	if err := Run(t.Context(), args...); err != nil {
		t.Fatal(err)
	}
	readmePath := filepath.Join(repoDir, "output", "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		t.Errorf("expected README.md to be generated: %v", err)
	}
}

func TestCommitChanges(t *testing.T) {
	repoDir := t.TempDir()
	setupGitRepo(t, repoDir)

	testFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWD)

	if err := commitChanges(t.Context()); err != nil {
		t.Fatal(err)
	}

	cmd := exec.CommandContext(t.Context(), "git", "log", "-1", "--pretty=%s")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(output))
	want := "chore: run librarian update and generate --all"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestCreateBranch(t *testing.T) {
	repoDir := t.TempDir()
	setupGitRepo(t, repoDir)

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWD)

	fixedDate := time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC)
	if err := createBranch(t.Context(), fixedDate); err != nil {
		t.Fatal(err)
	}

	cmd := exec.CommandContext(t.Context(), "git", "branch", "--show-current")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(output))
	want := "librarianops-generateall-2026-01-19"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestProcessRepo_WorkingDirectoryRestoration(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, repoFake)
	if err := os.Mkdir(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	setupGitRepo(t, repoDir)
	if err := command.Run(t.Context(), "git", "-C", repoDir, "checkout", "-b", "main"); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	googleapisDir := filepath.Join(wd, "..", "testdata", "googleapis")
	configContent := fmt.Sprintf(`language: fake
sources:
  googleapis:
    dir: %s
libraries:
  - name: test-library
    output: output
    channels:
      - path: google/cloud/secretmanager/v1
`, googleapisDir)
	if err := os.WriteFile(filepath.Join(repoDir, "librarian.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", repoDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatal(err)
	}

	if err := processRepo(t.Context(), repoFake, repoDir); err != nil {
		t.Fatal(err)
	}

	currentWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(originalWD, currentWD); diff != "" {
		t.Errorf("working directory not restored (-want +got):\n%s", diff)
	}
}

func TestProcessRepo_TempDirectoryCleanup(t *testing.T) {
	err := processRepo(t.Context(), repoFake, "")
	if err == nil {
		t.Fatal("expected error when cloning non-existent repo")
	}
}

func setupGitRepo(t *testing.T, dir string) {
	t.Helper()
	if err := command.Run(t.Context(), "git", "init", dir); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", dir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "-C", dir, "config", "user.name", "Test User"); err != nil {
		t.Fatal(err)
	}
}
