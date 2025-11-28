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

package rust

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	cmdtest "github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

func TestGenerate(t *testing.T) {
	cmdtest.RequireCommand(t, "protoc")
	cmdtest.RequireCommand(t, "rustfmt")
	cmdtest.RequireCommand(t, "taplo")
	testdataDir, err := filepath.Abs("../../../sidekick/testdata")
	if err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	googleapisDir := filepath.Join(testdataDir, "googleapis")
	library := &config.Library{
		Name:          "secretmanager",
		Version:       "0.1.0",
		Output:        outDir,
		ReleaseLevel:  "preview",
		CopyrightYear: "2025",
		APIs: []*config.API{
			{
				Path:          "google/cloud/secretmanager/v1",
				ServiceConfig: "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		Rust: &config.RustCrate{
			RustDefault: config.RustDefault{
				PackageDependencies: []*config.RustPackageDependency{
					{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
					{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
					{Name: "location", Package: "google-cloud-location", Source: "google.cloud.location"},
				},
			},
		},
	}
	sources := &config.Sources{
		Googleapis: &config.Source{Dir: googleapisDir},
	}

	if err := Generate(t.Context(), library, sources); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		path string
		want string
	}{
		{filepath.Join(outDir, "Cargo.toml"), "name"},
		{filepath.Join(outDir, "Cargo.toml"), "secretmanager"},
		{filepath.Join(outDir, "README.md"), "# Google Cloud Client Libraries for Rust - Secret Manager API"},
		{filepath.Join(outDir, "src", "lib.rs"), "pub mod model;"},
		{filepath.Join(outDir, "src", "lib.rs"), "pub mod client;"},
	} {
		t.Run(test.path, func(t *testing.T) {
			if _, err := os.Stat(test.path); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(got), test.want) {
				t.Errorf("%q missing expected string: %q", test.path, test.want)
			}
		})
	}
}

func TestCleanOutput(t *testing.T) {
	for _, test := range []struct {
		name  string
		files []string
		want  []string
	}{
		{
			name:  "removes all except Cargo.toml",
			files: []string{"Cargo.toml", "README.md", "src/lib.rs"},
			want:  []string{"Cargo.toml"},
		},
		{
			name:  "empty directory",
			files: []string{},
			want:  []string{},
		},
		{
			name:  "only Cargo.toml",
			files: []string{"Cargo.toml"},
			want:  []string{"Cargo.toml"},
		},
		{
			name:  "no Cargo.toml",
			files: []string{"README.md", "src/lib.rs"},
			want:  []string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range test.files {
				path := filepath.Join(dir, f)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := cleanOutput(dir); err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, e := range entries {
				got = append(got, e.Name())
			}
			slices.Sort(got)
			slices.Sort(test.want)
			if !slices.Equal(got, test.want) {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestCleanOutput_NonExistentDir(t *testing.T) {
	if err := cleanOutput("/nonexistent/path"); err != nil {
		t.Errorf("expected nil error for nonexistent dir, got %v", err)
	}
}

func TestReadCopyrightYear(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "valid copyright header",
			content: "# Copyright 2024\n[package]\nname = \"test\"",
			want:    "2024",
		},
		{
			name:    "no copyright header",
			content: "[package]\nname = \"test\"",
			want:    currentYear(),
		},
		{
			name:    "empty file",
			content: "",
			want:    currentYear(),
		},
		{
			name:    "copyright not on first line",
			content: "[package]\n# Copyright 2020\nname = \"test\"",
			want:    currentYear(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			if got := readCopyrightYear(dir); got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestReadCopyrightYear_NoFile(t *testing.T) {
	dir := t.TempDir()
	if got, want := readCopyrightYear(dir), currentYear(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
