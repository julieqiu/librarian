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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestValidateLibraries(t *testing.T) {
	for _, test := range []struct {
		name      string
		libraries []*config.Library
		wantErr   string
	}{
		{
			name: "valid libraries",
			libraries: []*config.Library{
				{Name: "google-cloud-secretmanager-v1"},
				{Name: "google-cloud-storage-v1"},
			},
		},
		{
			name: "duplicate library names",
			libraries: []*config.Library{
				{Name: "google-cloud-secretmanager-v1"},
				{Name: "google-cloud-secretmanager-v1"},
			},
			wantErr: "duplicate library name: google-cloud-secretmanager-v1 (appears 2 times)",
		},
		{
			name: "duplicate channels via derived name",
			libraries: []*config.Library{
				{Name: "google-cloud-secretmanager-v1"},
				{Name: "google-cloud-secretmanager-v1-alt", Channel: "google/cloud/secretmanager/v1"},
			},
			wantErr: "duplicate channel: google/cloud/secretmanager/v1 (appears 2 times)",
		},
		{
			name: "duplicate explicit channels",
			libraries: []*config.Library{
				{Name: "lib1", Channel: "google/cloud/foo/v1"},
				{Name: "lib2", Channel: "google/cloud/foo/v1"},
			},
			wantErr: "duplicate channel: google/cloud/foo/v1 (appears 2 times)",
		},
		{
			name: "duplicate in Channels list",
			libraries: []*config.Library{
				{Name: "lib1", Channels: []string{"google/cloud/foo/v1", "google/cloud/bar/v1"}},
				{Name: "lib2", Channel: "google/cloud/foo/v1"},
			},
			wantErr: "duplicate channel: google/cloud/foo/v1 (appears 2 times)",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{Libraries: test.libraries}
			err := validateLibraries(cfg)
			if test.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error containing %q, got nil", test.wantErr)
				return
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func TestValidateLibrary(t *testing.T) {
	for _, test := range []struct {
		name    string
		lib     *config.Library
		wantErr string
	}{
		{
			name: "valid protobuf library",
			lib:  &config.Library{Name: "google-cloud-secretmanager-v1"},
		},
		{
			name: "valid discovery library",
			lib: &config.Library{
				Name:                "google-cloud-compute-v1",
				SpecificationFormat: "discovery",
				SpecificationSource: "discoveries/compute.v1.json",
				Output:              "src/generated/cloud/compute/v1",
			},
		},
		{
			name:    "invalid name derivation",
			lib:     &config.Library{Name: "my-custom-library"},
			wantErr: `library "my-custom-library": name cannot be derived into a valid channel`,
		},
		{
			name: "discovery without specification_source",
			lib: &config.Library{
				Name:                "google-cloud-compute-v1",
				SpecificationFormat: "discovery",
				Output:              "src/generated/cloud/compute/v1",
			},
			wantErr: `library "google-cloud-compute-v1": discovery API requires specification_source`,
		},
		{
			name: "discovery without output",
			lib: &config.Library{
				Name:                "google-cloud-compute-v1",
				SpecificationFormat: "discovery",
				SpecificationSource: "discoveries/compute.v1.json",
			},
			wantErr: `library "google-cloud-compute-v1": discovery API requires explicit output path`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateLibrary(test.lib)
			if test.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error containing %q, got nil", test.wantErr)
				return
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func TestValidateIgnored(t *testing.T) {
	for _, test := range []struct {
		name    string
		ignored []string
		wantErr string
	}{
		{
			name:    "valid ignored list",
			ignored: []string{"google/cloud/foo/", "google/cloud/bar"},
		},
		{
			name:    "duplicate patterns",
			ignored: []string{"google/cloud/foo/", "google/cloud/foo/"},
			wantErr: "duplicate ignored pattern: google/cloud/foo/",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{Ignored: test.ignored}
			err := validateIgnored(cfg)
			if test.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error containing %q, got nil", test.wantErr)
				return
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func TestFormatConfig(t *testing.T) {
	cfg := &config.Config{
		Libraries: []*config.Library{
			{Name: "google-cloud-storage-v1", ReleaseLevel: "stable"},
			{Name: "google-cloud-bigquery-v1", ReleaseLevel: "stable"},
			{Name: "google-cloud-storage-v1", ReleaseLevel: "stable"}, // duplicate
			{Name: "google-cloud-secretmanager-v1", ReleaseLevel: "stable"},
		},
		Ignored: []string{
			"google/cloud/foo/",
			"google/cloud/bar/",
			"google/cloud/foo/", // duplicate
		},
	}

	formatConfig(cfg)

	// Libraries should be deduplicated and sorted.
	wantLibNames := []string{
		"google-cloud-bigquery-v1",
		"google-cloud-secretmanager-v1",
		"google-cloud-storage-v1",
	}
	var gotLibNames []string
	for _, lib := range cfg.Libraries {
		gotLibNames = append(gotLibNames, lib.Name)
	}
	if diff := cmp.Diff(wantLibNames, gotLibNames); diff != "" {
		t.Errorf("libraries mismatch (-want +got):\n%s", diff)
	}

	// Ignored should be deduplicated and sorted.
	wantIgnored := []string{
		"google/cloud/bar/",
		"google/cloud/foo/",
	}
	if diff := cmp.Diff(wantIgnored, cfg.Ignored); diff != "" {
		t.Errorf("ignored mismatch (-want +got):\n%s", diff)
	}
}

func TestRemoveDuplicateLibraries(t *testing.T) {
	libs := []*config.Library{
		{Name: "lib1"},
		{Name: "lib2"},
		{Name: "lib1"}, // duplicate
		{Name: "lib3"},
		{Name: "lib2"}, // duplicate
	}

	result := removeDuplicateLibraries(libs)

	wantNames := []string{"lib1", "lib2", "lib3"}
	var gotNames []string
	for _, lib := range result {
		gotNames = append(gotNames, lib.Name)
	}
	if diff := cmp.Diff(wantNames, gotNames); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRemoveDuplicateStrings(t *testing.T) {
	strs := []string{"a", "b", "a", "c", "b", "d"}
	result := removeDuplicateStrings(strs)
	want := []string{"a", "b", "c", "d"}
	if diff := cmp.Diff(want, result); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFmtCommand(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)

	configContent := `language: rust
sources:
  googleapis:
    commit: abc123
libraries:
  - name: google-cloud-storage-v1
    release_level: stable
  - name: google-cloud-bigquery-v1
    release_level: stable
  - name: google-cloud-storage-v1
    release_level: stable
ignored:
  - google/cloud/foo/
  - google/cloud/bar/
  - google/cloud/foo/
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := Run(t.Context(), "librarian", "fmt", "-w", ".")
	if err != nil {
		t.Fatal(err)
	}

	// Read the formatted config.
	cfg, err := config.Read(configPath)
	if err != nil {
		t.Fatal(err)
	}

	// Libraries should be deduplicated and sorted.
	wantLibNames := []string{
		"google-cloud-bigquery-v1",
		"google-cloud-storage-v1",
	}
	var gotLibNames []string
	for _, lib := range cfg.Libraries {
		gotLibNames = append(gotLibNames, lib.Name)
	}
	if diff := cmp.Diff(wantLibNames, gotLibNames); diff != "" {
		t.Errorf("libraries mismatch (-want +got):\n%s", diff)
	}

	// Ignored should be deduplicated and sorted.
	wantIgnored := []string{
		"google/cloud/bar/",
		"google/cloud/foo/",
	}
	if diff := cmp.Diff(wantIgnored, cfg.Ignored); diff != "" {
		t.Errorf("ignored mismatch (-want +got):\n%s", diff)
	}
}

func TestFmtCommandCheckOnly(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)

	configContent := `language: rust
sources:
  googleapis:
    commit: abc123
libraries:
  - name: google-cloud-storage-v1
  - name: google-cloud-bigquery-v1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Get original content.
	originalContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	// Default behavior (no -w flag) should not modify the file.
	err = Run(t.Context(), "librarian", "fmt")
	if err != nil {
		t.Fatal(err)
	}

	// File should not be modified in check mode.
	newContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(originalContent), string(newContent)); diff != "" {
		t.Errorf("file was modified in check mode (-want +got):\n%s", diff)
	}
}

func TestFmtCommandCheckOnlyWithError(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)

	// Library with invalid name that has additional fields (so it won't be removed as name-only).
	configContent := `language: rust
sources:
  googleapis:
    commit: abc123
libraries:
  - name: my-invalid-library
    release_level: stable
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Default behavior (no -w flag) should report errors.
	err := Run(t.Context(), "librarian", "fmt")
	if err == nil {
		t.Error("expected error for invalid library name, got nil")
		return
	}
	if !strings.Contains(err.Error(), "name cannot be derived into a valid channel") {
		t.Errorf("expected error about invalid channel derivation, got: %v", err)
	}
}

func TestIsDiscoveryAPI(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want bool
	}{
		{
			name: "protobuf API",
			lib:  &config.Library{Name: "google-cloud-secretmanager-v1"},
			want: false,
		},
		{
			name: "discovery API",
			lib:  &config.Library{Name: "google-cloud-compute-v1", SpecificationFormat: "discovery"},
			want: true,
		},
		{
			name: "empty specification format",
			lib:  &config.Library{Name: "google-cloud-foo-v1", SpecificationFormat: ""},
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isDiscoveryAPI(test.lib)
			if got != test.want {
				t.Errorf("isDiscoveryAPI() = %v, want %v", got, test.want)
			}
		})
	}
}
