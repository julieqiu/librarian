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

package generate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/container/go/config"
	"github.com/googleapis/librarian/internal/container/go/request"
)

func TestApiShortname(t *testing.T) {
	nameFull := "secretmanager.googleapis.com"
	want := "secretmanager"
	if got := apiShortname(nameFull); got != want {
		t.Errorf("apiShortname() = %v, want %v", got, want)
	}
}

func TestDocURL(t *testing.T) {
	modulePath := "cloud.google.com/go/secretmanager"
	importPath := "cloud.google.com/go/secretmanager/apiv1"
	want := "https://cloud.google.com/go/docs/reference/cloud.google.com/go/secretmanager/latest/apiv1"
	got, err := docURL(modulePath, importPath)
	if err != nil {
		t.Fatalf("docURL() unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("docURL() = %v, want %v", got, want)
	}
}

func TestGenerateRepoMetadata(t *testing.T) {
	testCases := []struct {
		name            string
		serviceConfig   string
		expectFile      bool
		expectedContent manifestEntry
	}{
		{
			name:          "with service config",
			serviceConfig: "testlib_v1.yaml",
			expectFile:    true,
			expectedContent: manifestEntry{
				APIShortname:        "test",
				ClientDocumentation: "https://cloud.google.com/go/docs/reference/cloud.google.com/go/testlib/latest/apiv1",
				ClientLibraryType:   "generated",
				Description:         "Test API",
				DistributionName:    "cloud.google.com/go/testlib/apiv1",
				Language:            "go",
				LibraryType:         "GAPIC_AUTO",
				ReleaseLevel:        "stable",
			},
		},
		{
			name:          "without service config",
			serviceConfig: "",
			expectFile:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sourceDir := filepath.Join(tmpDir, "source")
			outputDir := filepath.Join(tmpDir, "output")
			if err := os.MkdirAll(filepath.Join(sourceDir, "google/cloud/testlib/v1"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				t.Fatal(err)
			}

			if err := os.WriteFile(filepath.Join(sourceDir, "google/cloud/testlib/v1", "testlib_v1.yaml"), []byte("name: test.googleapis.com\ntitle: Test API"), 0644); err != nil {
				t.Fatal(err)
			}

			cfg := &Config{
				SourceDir: sourceDir,
				OutputDir: outputDir,
			}
			lib := &request.Library{
				ID: "testlib",
			}
			api := &request.API{
				Path:            "google/cloud/testlib/v1",
				ServiceConfig:   tc.serviceConfig,
				GAPICImportPath: "cloud.google.com/go/testlib/apiv1",
				ReleaseLevel:    "stable",
			}
			moduleConfig := &config.ModuleConfig{
				Name: "testlib",
			}

			if err := generateRepoMetadata(context.Background(), cfg, lib, api, moduleConfig); err != nil {
				t.Fatalf("generateRepoMetadata() failed: %v", err)
			}

			filePath := filepath.Join(outputDir, "cloud.google.com/go/testlib/apiv1/.repo-metadata.json")
			if !tc.expectFile {
				if _, err := os.Stat(filePath); !os.IsNotExist(err) {
					t.Errorf("expected file to not exist, but it does")
				}
				return
			}

			got, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("os.ReadFile() failed: %v", err)
			}

			var gotEntry manifestEntry
			if err := json.Unmarshal(got, &gotEntry); err != nil {
				t.Fatalf("json.Unmarshal() failed: %v", err)
			}

			if diff := cmp.Diff(tc.expectedContent, gotEntry); diff != "" {
				t.Errorf("generateRepoMetadata() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
