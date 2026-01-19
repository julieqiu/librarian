// Copyright 2026 Google LLC
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

package golang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestFormat_NoChannels(t *testing.T) {
	library := &config.Library{
		Name:     "test",
		Output:   t.TempDir(),
		Channels: []*config.Channel{},
	}

	err := Format(t.Context(), library)
	if err != nil {
		t.Errorf("Format() with no channels should return nil, got: %v", err)
	}
}

func TestFormat_NoVersion(t *testing.T) {
	library := &config.Library{
		Name:   "test",
		Output: t.TempDir(),
		Channels: []*config.Channel{
			{Path: "google/cloud/test/v1"},
		},
		Version: "",
	}

	err := Format(t.Context(), library)
	if err == nil {
		t.Fatal("expected error when version is empty")
	}
	if !strings.Contains(err.Error(), "no version") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBuildModulePath(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    string
	}{
		{
			name: "no version override",
			library: &config.Library{
				Name: "spanner",
			},
			want: "cloud.google.com/go/spanner",
		},
		{
			name: "with version override",
			library: &config.Library{
				Name: "spanner",
				Go: &config.GoModule{
					ModulePathVersion: "v2",
				},
			},
			want: "cloud.google.com/go/spanner/v2",
		},
		{
			name: "empty version override",
			library: &config.Library{
				Name: "firestore",
				Go: &config.GoModule{
					ModulePathVersion: "",
				},
			},
			want: "cloud.google.com/go/firestore",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildModulePath(test.library)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDirExists(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(t *testing.T) string
		want  bool
	}{
		{
			name: "directory does not exist",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			want: false,
		},
		{
			name: "directory exists",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			want: true,
		},
		{
			name: "file exists instead of directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				file := filepath.Join(dir, "file.txt")
				if err := os.WriteFile(file, []byte("content"), 0644); err != nil {
					t.Fatal(err)
				}
				return file
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := test.setup(t)
			got := dirExists(dir)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAPIConfig(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		path    string
		want    *config.GoAPI
	}{
		{
			name: "no go config",
			library: &config.Library{
				Name: "test",
			},
			path: "google/cloud/test/v1",
			want: nil,
		},
		{
			name: "go config but no matching path",
			library: &config.Library{
				Name: "test",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{Path: "google/cloud/other/v1"},
					},
				},
			},
			path: "google/cloud/test/v1",
			want: nil,
		},
		{
			name: "matching path found",
			library: &config.Library{
				Name: "test",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:         "google/cloud/test/v1",
							ProtoPackage: "custom.proto.package",
						},
					},
				},
			},
			path: "google/cloud/test/v1",
			want: &config.GoAPI{
				Path:         "google/cloud/test/v1",
				ProtoPackage: "custom.proto.package",
			},
		},
		{
			name: "multiple configs, finds correct one",
			library: &config.Library{
				Name: "test",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{Path: "google/cloud/test/v1"},
						{
							Path:            "google/cloud/test/v2",
							ClientDirectory: "apiv2",
						},
						{Path: "google/cloud/test/v3"},
					},
				},
			},
			path: "google/cloud/test/v2",
			want: &config.GoAPI{
				Path:            "google/cloud/test/v2",
				ClientDirectory: "apiv2",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := apiConfig(test.library, test.path)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProtoPackage(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		path    string
		want    string
	}{
		{
			name: "no override, simple path",
			library: &config.Library{
				Name: "test",
			},
			path: "google/cloud/test/v1",
			want: "google.cloud.test.v1",
		},
		{
			name: "no override, complex path",
			library: &config.Library{
				Name: "spanner",
			},
			path: "google/spanner/admin/instance/v1",
			want: "google.spanner.admin.instance.v1",
		},
		{
			name: "with override",
			library: &config.Library{
				Name: "test",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:         "google/cloud/test/v1",
							ProtoPackage: "custom.proto.package",
						},
					},
				},
			},
			path: "google/cloud/test/v1",
			want: "custom.proto.package",
		},
		{
			name: "override exists but for different path",
			library: &config.Library{
				Name: "test",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:         "google/cloud/test/v2",
							ProtoPackage: "custom.proto.package",
						},
					},
				},
			},
			path: "google/cloud/test/v1",
			want: "google.cloud.test.v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := protoPackage(test.library, test.path)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClientDirectory(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "simple path with v1",
			library: &config.Library{
				Name: "spanner",
			},
			path: "google/cloud/spanner/v1",
			want: "apiv1",
		},
		{
			name: "admin path",
			library: &config.Library{
				Name: "spanner",
			},
			path: "google/spanner/admin/instance/v1",
			want: "admin/instance/apiv1",
		},
		{
			name: "with override",
			library: &config.Library{
				Name: "spanner",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:            "google/cloud/spanner/v1",
							ClientDirectory: "custom/dir",
						},
					},
				},
			},
			path: "google/cloud/spanner/v1",
			want: "custom/dir",
		},
		{
			name: "path without module name",
			library: &config.Library{
				Name: "spanner",
			},
			path:    "google/cloud/other/v1",
			wantErr: true,
		},
		{
			name: "path with no slash after module name",
			library: &config.Library{
				Name: "spanner",
			},
			path:    "google/spanner",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := clientDirectory(test.library, test.path)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateSnippetsMetadata(t *testing.T) {
	for _, test := range []struct {
		name         string
		library      *config.Library
		snippetFiles map[string]string
		want         map[string]string
		wantErr      bool
	}{
		{
			name: "replace $VERSION placeholder",
			library: &config.Library{
				Name:    "spanner",
				Version: "1.2.3",
				Channels: []*config.Channel{
					{Path: "google/cloud/spanner/v1"},
				},
			},
			snippetFiles: map[string]string{
				"internal/generated/snippets/spanner/apiv1/snippet_metadata.google.cloud.spanner.v1.json": `{"version": "$VERSION"}`,
			},
			want: map[string]string{
				"internal/generated/snippets/spanner/apiv1/snippet_metadata.google.cloud.spanner.v1.json": `{"version": "1.2.3"}`,
			},
		},
		{
			name: "replace existing version number",
			library: &config.Library{
				Name:    "spanner",
				Version: "2.0.0",
				Channels: []*config.Channel{
					{Path: "google/cloud/spanner/v1"},
				},
			},
			snippetFiles: map[string]string{
				"internal/generated/snippets/spanner/apiv1/snippet_metadata.google.cloud.spanner.v1.json": `{"version": "1.0.0"}`,
			},
			want: map[string]string{
				"internal/generated/snippets/spanner/apiv1/snippet_metadata.google.cloud.spanner.v1.json": `{"version": "2.0.0"}`,
			},
		},
		{
			name: "multiple channels",
			library: &config.Library{
				Name:    "spanner",
				Version: "1.5.0",
				Channels: []*config.Channel{
					{Path: "google/cloud/spanner/v1"},
					{Path: "google/spanner/admin/instance/v1"},
				},
			},
			snippetFiles: map[string]string{
				"internal/generated/snippets/spanner/apiv1/snippet_metadata.google.cloud.spanner.v1.json":                         `{"version": "$VERSION"}`,
				"internal/generated/snippets/spanner/admin/instance/apiv1/snippet_metadata.google.spanner.admin.instance.v1.json": `{"version": "1.0.0"}`,
			},
			want: map[string]string{
				"internal/generated/snippets/spanner/apiv1/snippet_metadata.google.cloud.spanner.v1.json":                         `{"version": "1.5.0"}`,
				"internal/generated/snippets/spanner/admin/instance/apiv1/snippet_metadata.google.spanner.admin.instance.v1.json": `{"version": "1.5.0"}`,
			},
		},
		{
			name: "snippet file missing, skips without error",
			library: &config.Library{
				Name:    "spanner",
				Version: "1.0.0",
				Channels: []*config.Channel{
					{Path: "google/cloud/spanner/v1"},
				},
			},
			snippetFiles: map[string]string{},
			want:         map[string]string{},
		},
		{
			name: "no version placeholder or number found",
			library: &config.Library{
				Name:    "spanner",
				Version: "1.0.0",
				Channels: []*config.Channel{
					{Path: "google/cloud/spanner/v1"},
				},
			},
			snippetFiles: map[string]string{
				"internal/generated/snippets/spanner/apiv1/snippet_metadata.google.cloud.spanner.v1.json": `{"version": "unknown"}`,
			},
			wantErr: true,
		},
		{
			name: "with custom proto package",
			library: &config.Library{
				Name:    "test",
				Version: "3.0.0",
				Channels: []*config.Channel{
					{Path: "google/cloud/test/v1"},
				},
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:         "google/cloud/test/v1",
							ProtoPackage: "custom.proto.pkg",
						},
					},
				},
			},
			snippetFiles: map[string]string{
				"internal/generated/snippets/test/apiv1/snippet_metadata.custom.proto.pkg.json": `{"version": "$VERSION"}`,
			},
			want: map[string]string{
				"internal/generated/snippets/test/apiv1/snippet_metadata.custom.proto.pkg.json": `{"version": "3.0.0"}`,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sourceDir := t.TempDir()
			destDir := t.TempDir()

			for path, content := range test.snippetFiles {
				fullPath := filepath.Join(sourceDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			err := updateSnippetsMetadata(test.library, sourceDir, destDir)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			for path, wantContent := range test.want {
				fullPath := filepath.Join(destDir, path)
				gotBytes, err := os.ReadFile(fullPath)
				if err != nil {
					t.Fatal(err)
				}
				got := string(gotBytes)
				if diff := cmp.Diff(wantContent, got); diff != "" {
					t.Errorf("file %s mismatch (-want +got):\n%s", path, diff)
				}
			}
		})
	}
}
