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

package python

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestReleaseLibrary(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "packages", "test-lib")
	if err := os.MkdirAll(srcPath, 0755); err != nil {
		t.Fatal(err)
	}

	versionPy := filepath.Join(srcPath, "google", "cloud", "test_lib", "version.py")
	if err := os.MkdirAll(filepath.Dir(versionPy), 0755); err != nil {
		t.Fatal(err)
	}
	versionContent := `
__version__ = "1.2.3"
__release_date__ = "2024-01-01"
`
	if err := os.WriteFile(versionPy, []byte(versionContent), 0644); err != nil {
		t.Fatal(err)
	}

	snippetJSON := filepath.Join(srcPath, "samples", "snippets", "snippet_metadata.json")
	if err := os.MkdirAll(filepath.Dir(snippetJSON), 0755); err != nil {
		t.Fatal(err)
	}
	snippetContent := `
{
  "clientLibrary": {
    "version": "1.2.3"
  }
}
`
	if err := os.WriteFile(snippetJSON, []byte(snippetContent), 0644); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Name:    "test-lib",
		Version: "1.2.3",
	}

	if err := ReleaseLibrary(lib, srcPath); err != nil {
		t.Fatal(err)
	}

	// Verify version.py
	gotVersionPy, err := os.ReadFile(versionPy)
	if err != nil {
		t.Fatal(err)
	}
	// Expected Minor bump from 1.2.3 is 1.3.0
	if !regexp.MustCompile(`__version__ = "1.3.0"`).Match(gotVersionPy) {
		t.Errorf("version.py mismatch, got:\n%s", string(gotVersionPy))
	}
	if regexp.MustCompile(`__release_date__ = "2024-01-01"`).Match(gotVersionPy) {
		t.Errorf("version.py date not updated, got:\n%s", string(gotVersionPy))
	}

	// Verify snippet_metadata.json
	gotSnippet, err := os.ReadFile(snippetJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`"version": "1.3.0"`).Match(gotSnippet) {
		t.Errorf("snippet_metadata.json mismatch, got:\n%s", string(gotSnippet))
	}

	// Verify library config
	if diff := cmp.Diff("1.3.0", lib.Version); diff != "" {
		t.Errorf("library version mismatch (-want +got):\n%s", diff)
	}
}

func TestDeriveSrcPath(t *testing.T) {
	tests := []struct {
		name     string
		libCfg   *config.Library
		cfg      *config.Config
		wantPath string
	}{
		{
			name: "explicit output",
			libCfg: &config.Library{
				Output: "explicit/path",
			},
			wantPath: "explicit/path",
		},
		{
			name: "default path from name",
			libCfg: &config.Library{
				Name: "test-lib",
			},
			wantPath: filepath.Join("packages", "test-lib"),
		},
		{
			name: "path from channel",
			libCfg: &config.Library{
				Name: "test-lib",
				Channels: []*config.Channel{
					{Path: "google/cloud/test/v1"},
				},
			},
			wantPath: "google/cloud/test/v1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveSrcPath(test.libCfg, test.cfg)
			if diff := cmp.Diff(test.wantPath, got); diff != "" {
				t.Errorf("DeriveSrcPath() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
