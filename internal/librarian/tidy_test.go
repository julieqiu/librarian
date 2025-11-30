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

package librarian

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
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

func TestFormatConfig(t *testing.T) {
	cfg := &config.Config{
		Libraries: []*config.Library{
			{Name: "google-cloud-storage-v1", Version: "1.0.0"},
			{Name: "google-cloud-bigquery-v1", Version: "2.0.0"},
			{Name: "google-cloud-secretmanager-v1", Version: "3.0.0"},
			{Name: "google-cloud-default-v1"}, // no config, should be removed
		},
	}

	formatConfig(cfg)

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
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestTidyCommand(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)

	configContent := `language: rust
sources:
  googleapis:
    commit: abc123
libraries:
  - name: google-cloud-storage-v1
    version: "1.0.0"
  - name: google-cloud-bigquery-v1
    version: "2.0.0"
  - name: google-cloud-default-v1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Run(t.Context(), "librarian", "tidy"); err != nil {
		t.Fatal(err)
	}

	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatal(err)
	}

	// google-cloud-default-v1 should be removed (no config beyond name)
	wantLibNames := []string{
		"google-cloud-bigquery-v1",
		"google-cloud-storage-v1",
	}
	var gotLibNames []string
	for _, lib := range cfg.Libraries {
		gotLibNames = append(gotLibNames, lib.Name)
	}
	if diff := cmp.Diff(wantLibNames, gotLibNames); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestTidyCommandDuplicateError(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)

	configContent := `language: rust
sources:
  googleapis:
    commit: abc123
libraries:
  - name: google-cloud-storage-v1
  - name: google-cloud-storage-v1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := Run(t.Context(), "librarian", "tidy")
	if err == nil {
		t.Fatal("expected error for duplicate library")
	}
	if !strings.Contains(err.Error(), "duplicate library name") {
		t.Errorf("expected duplicate error, got: %v", err)
	}
}
