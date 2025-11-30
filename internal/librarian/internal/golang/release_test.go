// Copyright 2025 Google LLC
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

package golang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

const (
	storageDir      = "storage"
	storageName     = "storage"
	storageInitial  = "1.0.0"
	storageReleased = "1.0.1"

	secretmanagerDir      = "secretmanager"
	secretmanagerName     = "secretmanager"
	secretmanagerInitial  = "1.5.3"
	secretmanagerReleased = "1.5.4"
)

func TestReleaseAll(t *testing.T) {
	cfg := setupRelease(t)
	got, err := ReleaseAll(cfg)
	if err != nil {
		t.Fatal(err)
	}

	checkVersionFile(t, storageDir, storageReleased)
	checkVersionFile(t, secretmanagerDir, secretmanagerReleased)
	checkChangelog(t, storageDir, storageReleased)
	checkChangelog(t, secretmanagerDir, secretmanagerReleased)
	want := map[string]string{
		storageName:       storageReleased,
		secretmanagerName: secretmanagerReleased,
	}
	if diff := cmp.Diff(want, libraryVersions(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestReleaseOne(t *testing.T) {
	cfg := setupRelease(t)
	got, err := ReleaseLibrary(cfg, storageName)
	if err != nil {
		t.Fatal(err)
	}

	checkVersionFile(t, storageDir, storageReleased)
	checkChangelog(t, storageDir, storageReleased)
	// secretmanager should not be updated.
	if _, err := os.Stat(filepath.Join(secretmanagerDir, "internal", "version.go")); !os.IsNotExist(err) {
		t.Error("secretmanager version.go should not exist")
	}
	want := map[string]string{
		storageName:       storageReleased,
		secretmanagerName: secretmanagerInitial,
	}
	if diff := cmp.Diff(want, libraryVersions(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestReleaseSkipped(t *testing.T) {
	cfg := setupRelease(t)
	cfg.Libraries[0].SkipRelease = true
	got, err := ReleaseAll(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// storage should not be updated.
	if _, err := os.Stat(filepath.Join(storageDir, "internal", "version.go")); !os.IsNotExist(err) {
		t.Error("storage version.go should not exist")
	}
	checkVersionFile(t, secretmanagerDir, secretmanagerReleased)
	want := map[string]string{
		storageName:       storageInitial,
		secretmanagerName: secretmanagerReleased,
	}
	if diff := cmp.Diff(want, libraryVersions(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdateChangelog(t *testing.T) {
	// Use fixed time for deterministic output.
	now = func() time.Time {
		return time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	}
	defer func() { now = time.Now }()

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := os.MkdirAll("testlib", 0755); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Name:      "testlib",
		Version:   "1.2.3",
		TagFormat: "{name}/v{version}",
	}

	if err := updateChangelog(lib, now().UTC()); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile("testlib/CHANGES.md")
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	if !strings.Contains(got, "## [1.2.3]") {
		t.Errorf("changelog missing version header:\n%s", got)
	}
	if !strings.Contains(got, "2025-01-15") {
		t.Errorf("changelog missing date:\n%s", got)
	}
	if !strings.Contains(got, "testlib%2Fv1.2.3") {
		t.Errorf("changelog missing encoded tag:\n%s", got)
	}
}

func TestUpdateVersionFile(t *testing.T) {
	// Use fixed time for deterministic output.
	now = func() time.Time {
		return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	defer func() { now = time.Now }()

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if err := os.MkdirAll("testlib", 0755); err != nil {
		t.Fatal(err)
	}

	if err := updateVersionFile("testlib", "1.2.3"); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile("testlib/internal/version.go")
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	if !strings.Contains(got, `const Version = "1.2.3"`) {
		t.Errorf("version.go missing version constant:\n%s", got)
	}
	if !strings.Contains(got, "Copyright 2025") {
		t.Errorf("version.go missing copyright year:\n%s", got)
	}
}

func TestUpdateSnippetsMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	snippetsDir := filepath.Join("internal", "generated", "snippets", "testlib", "apiv1")
	if err := os.MkdirAll(snippetsDir, 0755); err != nil {
		t.Fatal(err)
	}

	snippetFile := filepath.Join(snippetsDir, "snippet_metadata.google.cloud.testlib.v1.json")
	content := `{"version": "$VERSION"}`
	if err := os.WriteFile(snippetFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := updateSnippetsMetadata("testlib", "1.2.3"); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(snippetFile)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"version": "1.2.3"}`
	if string(got) != want {
		t.Errorf("got %q, want %q", string(got), want)
	}
}

func setupRelease(t *testing.T) *config.Config {
	t.Helper()
	// Use fixed time for deterministic output.
	now = func() time.Time {
		return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { now = time.Now })

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	createModule(t, storageDir, storageName, storageInitial)
	createModule(t, secretmanagerDir, secretmanagerName, secretmanagerInitial)
	return &config.Config{
		Libraries: []*config.Library{
			{Name: storageName, Output: storageDir, Version: storageInitial},
			{Name: secretmanagerName, Output: secretmanagerDir, Version: secretmanagerInitial},
		},
	}
}

func createModule(t *testing.T, dir, name, version string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
}

func checkVersionFile(t *testing.T, dir, wantVersion string) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(dir, "internal", "version.go"))
	if err != nil {
		t.Fatal(err)
	}
	wantLine := `const Version = "` + wantVersion + `"`
	got := string(contents)
	if !strings.Contains(got, wantLine) {
		t.Errorf("%s version.go mismatch:\nwant line: %q\ngot:\n%s", dir, wantLine, got)
	}
}

func checkChangelog(t *testing.T, dir, wantVersion string) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(dir, "CHANGES.md"))
	if err != nil {
		t.Fatal(err)
	}
	wantLine := "## [" + wantVersion + "]"
	got := string(contents)
	if !strings.Contains(got, wantLine) {
		t.Errorf("%s CHANGES.md mismatch:\nwant line: %q\ngot:\n%s", dir, wantLine, got)
	}
}

func libraryVersions(cfg *config.Config) map[string]string {
	m := make(map[string]string)
	for _, lib := range cfg.Libraries {
		m[lib.Name] = lib.Version
	}
	return m
}
