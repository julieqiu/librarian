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

package librarian

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	cmdtest "github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

func TestRunReleaseAll(t *testing.T) {
	cmdtest.RequireCommand(t, "cargo")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	createCrate(t, "src/storage", "google-cloud-storage", "1.0.0")
	createCrate(t, "src/secretmanager", "google-cloud-secretmanager-v1", "1.5.3")

	cfg := &config.Config{
		Version:  "v1",
		Language: "rust",
		Versions: map[string]string{
			"google-cloud-storage":          "1.0.0",
			"google-cloud-secretmanager-v1": "1.5.3",
		},
	}
	if err := cfg.Write(librarianConfigPath); err != nil {
		t.Fatal(err)
	}

	if err := runRelease(t.Context(), "", true, false); err != nil {
		t.Fatal(err)
	}

	checkCargoVersion(t, "src/storage/Cargo.toml", "1.1.0")
	checkCargoVersion(t, "src/secretmanager/Cargo.toml", "1.6.0")

	updatedCfg, err := config.Read(librarianConfigPath)
	if err != nil {
		t.Fatal(err)
	}

	wantVersions := map[string]string{
		"google-cloud-storage":          "1.1.0",
		"google-cloud-secretmanager-v1": "1.6.0",
	}
	if diff := cmp.Diff(wantVersions, updatedCfg.Versions); diff != "" {
		t.Errorf("versions mismatch (-want +got):\n%s", diff)
	}
}

func TestRunReleaseSingleLibrary(t *testing.T) {
	cmdtest.RequireCommand(t, "cargo")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	createCrate(t, "src/storage", "google-cloud-storage", "1.0.0")
	createCrate(t, "src/secretmanager", "google-cloud-secretmanager-v1", "1.5.3")

	cfg := &config.Config{
		Version:  "v1",
		Language: "rust",
		Versions: map[string]string{
			"google-cloud-storage":          "1.0.0",
			"google-cloud-secretmanager-v1": "1.5.3",
		},
	}
	if err := cfg.Write(librarianConfigPath); err != nil {
		t.Fatal(err)
	}

	if err := runRelease(t.Context(), "google-cloud-storage", false, false); err != nil {
		t.Fatal(err)
	}

	checkCargoVersion(t, "src/storage/Cargo.toml", "1.1.0")
	checkCargoVersion(t, "src/secretmanager/Cargo.toml", "1.5.3")

	updatedCfg, err := config.Read(librarianConfigPath)
	if err != nil {
		t.Fatal(err)
	}

	if updatedCfg.Versions["google-cloud-storage"] != "1.1.0" {
		t.Errorf("google-cloud-storage version = %s; want 1.1.0", updatedCfg.Versions["google-cloud-storage"])
	}
	if updatedCfg.Versions["google-cloud-secretmanager-v1"] != "1.5.3" {
		t.Errorf("google-cloud-secretmanager-v1 version = %s; want 1.5.3 (should not have changed)", updatedCfg.Versions["google-cloud-secretmanager-v1"])
	}
}

func TestRunReleaseValidation(t *testing.T) {
	for _, test := range []struct {
		name    string
		libName string
		all     bool
		wantErr string
	}{
		{
			name:    "both name and all",
			libName: "google-cloud-storage",
			all:     true,
			wantErr: "cannot specify both library name and --all flag",
		},
		{
			name:    "neither name nor all",
			libName: "",
			all:     false,
			wantErr: "must specify either a library name or --all flag",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := runRelease(t.Context(), test.libName, test.all, false)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != test.wantErr {
				t.Errorf("got error %q; want %q", err.Error(), test.wantErr)
			}
		})
	}
}

func createCrate(t *testing.T, dir, name, version string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	cargo := fmt.Sprintf(`[package]
name                   = "%s"
version                = "%s"
edition                = "2021"
`, name, version)

	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargo), 0644); err != nil {
		t.Fatal(err)
	}
}

func checkCargoVersion(t *testing.T, path, wantVersion string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	contentsStr := string(contents)
	wantSingle := fmt.Sprintf("version = '%s'", wantVersion)
	wantDouble := fmt.Sprintf(`version                = "%s"`, wantVersion)
	if !contains(contentsStr, wantSingle) && !contains(contentsStr, wantDouble) {
		t.Errorf("%s does not contain version %s\nGot:\n%s", path, wantVersion, contentsStr)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
