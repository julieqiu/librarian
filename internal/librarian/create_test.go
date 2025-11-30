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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateCommand(t *testing.T) {
	// Get absolute path to testdata before changing directory.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	googleapisDir := filepath.Join(wd, "testdata", "googleapis")

	for _, test := range []struct {
		name       string
		args       []string
		wantErr    error
		wantOutput string
		wantAPIs   []string
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "create"},
			wantErr: errMissingLibraryName,
		},
		{
			name:       "library name only - derives API path",
			args:       []string{"librarian", "create", "google-cloud-speech-v1"},
			wantOutput: "src/generated/google-cloud-speech-v1",
			wantAPIs:   []string{"google/cloud/speech/v1"},
		},
		{
			name:       "library name with explicit API path",
			args:       []string{"librarian", "create", "speech", "google/cloud/speech/v1"},
			wantOutput: "src/generated/speech",
			wantAPIs:   []string{"google/cloud/speech/v1"},
		},
		{
			name:       "library name with multiple API paths",
			args:       []string{"librarian", "create", "speech", "google/cloud/speech/v1", "google/cloud/speech/v2"},
			wantOutput: "src/generated/speech",
			wantAPIs:   []string{"google/cloud/speech/v1", "google/cloud/speech/v2"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			configContent := fmt.Sprintf(`language: testhelper
sources:
  googleapis:
    dir: %s
`, googleapisDir)
			if err := os.WriteFile(librarianConfigPath, []byte(configContent), 0644); err != nil {
				t.Fatal(err)
			}

			err := Run(t.Context(), test.args...)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("want error %v, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			// Check that output directory was created.
			readmePath := filepath.Join(test.wantOutput, "README.md")
			content, err := os.ReadFile(readmePath)
			if err != nil {
				t.Fatalf("expected README.md to exist at %s: %v", readmePath, err)
			}

			// Verify the content contains expected API paths.
			for _, apiPath := range test.wantAPIs {
				if !strings.Contains(string(content), apiPath) {
					t.Errorf("expected content to contain API path %q, got: %s", apiPath, content)
				}
			}
		})
	}
}

func TestCreateCommand_MissingSources(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	configContent := `language: testhelper
`
	if err := os.WriteFile(librarianConfigPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := Run(t.Context(), "librarian", "create", "test-library")
	if !errors.Is(err, errEmptySources) {
		t.Errorf("want error %v, got %v", errEmptySources, err)
	}
}
