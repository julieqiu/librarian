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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestBuildProtocArgs(t *testing.T) {
	googleapisDir := "/source/googleapis"
	outputDir := "/output"

	for _, test := range []struct {
		name       string
		library    *config.Library
		api        *config.API
		protoFiles []string
		want       []string
	}{
		{
			name:    "basic GAPIC with serviceconfig defaults",
			library: &config.Library{},
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
				Go: &config.GoPackage{
					ImportPath: "cloud.google.com/go/secretmanager/apiv1;secretmanager",
				},
			},
			protoFiles: []string{"/source/googleapis/google/cloud/secretmanager/v1/service.proto"},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_out=/output",
				"--go-grpc_out=/output",
				"--go-grpc_opt=require_unimplemented_servers=false",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--go_gapic_opt=grpc-service-config=/source/googleapis/google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
				"--go_gapic_opt=transport=grpc+rest",
				"--go_gapic_opt=release-level=stable",
				"--go_gapic_opt=metadata",
				"--go_gapic_opt=rest-numeric-enums",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
		{
			name: "derived import path",
			library: &config.Library{
				Name: "accessapproval",
			},
			api: &config.API{
				Path: "google/cloud/accessapproval/v1",
			},
			protoFiles: []string{"/source/googleapis/google/cloud/accessapproval/v1/service.proto"},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_out=/output",
				"--go-grpc_out=/output",
				"--go-grpc_opt=require_unimplemented_servers=false",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/accessapproval/apiv1;accessapproval",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/accessapproval/v1/accessapproval_v1.yaml",
				"--go_gapic_opt=grpc-service-config=/source/googleapis/google/cloud/accessapproval/v1/accessapproval_grpc_service_config.json",
				"--go_gapic_opt=transport=grpc+rest",
				"--go_gapic_opt=release-level=stable",
				"--go_gapic_opt=metadata",
				"--go_gapic_opt=rest-numeric-enums",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/accessapproval/v1/service.proto",
			},
		},
		{
			name:    "with legacy grpc",
			library: &config.Library{},
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
				Go: &config.GoPackage{
					ImportPath: "cloud.google.com/go/secretmanager/apiv1;secretmanager",
					LegacyGRPC: true,
				},
			},
			protoFiles: []string{"/source/googleapis/google/cloud/secretmanager/v1/service.proto"},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_v1_out=/output",
				"--go_v1_opt=plugins=grpc",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--go_gapic_opt=grpc-service-config=/source/googleapis/google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
				"--go_gapic_opt=transport=grpc+rest",
				"--go_gapic_opt=release-level=stable",
				"--go_gapic_opt=metadata",
				"--go_gapic_opt=rest-numeric-enums",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
		{
			name:    "with DIREGAPIC for compute",
			library: &config.Library{},
			api: &config.API{
				Path: "google/cloud/compute/v1",
				Go: &config.GoPackage{
					ImportPath: "cloud.google.com/go/compute/apiv1;compute",
				},
			},
			protoFiles: []string{"/source/googleapis/google/cloud/compute/v1/compute.proto"},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_out=/output",
				"--go-grpc_out=/output",
				"--go-grpc_opt=require_unimplemented_servers=false",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/compute/apiv1;compute",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/compute/v1/compute_v1.yaml",
				"--go_gapic_opt=grpc-service-config=/source/googleapis/google/cloud/compute/v1/compute_grpc_service_config.json",
				"--go_gapic_opt=transport=rest",
				"--go_gapic_opt=release-level=stable",
				"--go_gapic_opt=metadata",
				"--go_gapic_opt=diregapic",
				"--go_gapic_opt=rest-numeric-enums",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/compute/v1/compute.proto",
			},
		},
		{
			name:    "with grpc service config",
			library: &config.Library{},
			api: &config.API{
				Path: "google/cloud/asset/v1",
				Go: &config.GoPackage{
					ImportPath: "cloud.google.com/go/asset/apiv1;asset",
				},
			},
			protoFiles: []string{"/source/googleapis/google/cloud/asset/v1/service.proto"},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_out=/output",
				"--go-grpc_out=/output",
				"--go-grpc_opt=require_unimplemented_servers=false",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/asset/apiv1;asset",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/asset/v1/asset_v1.yaml",
				"--go_gapic_opt=grpc-service-config=/source/googleapis/google/cloud/asset/v1/cloudasset_grpc_service_config.json",
				"--go_gapic_opt=transport=grpc+rest",
				"--go_gapic_opt=release-level=stable",
				"--go_gapic_opt=metadata",
				"--go_gapic_opt=rest-numeric-enums",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/asset/v1/service.proto",
			},
		},
		{
			name:    "with proto_package",
			library: &config.Library{},
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
				Go: &config.GoPackage{
					ImportPath:   "cloud.google.com/go/secretmanager/apiv1;secretmanager",
					ProtoPackage: "google.cloud.secretmanager.v1",
				},
			},
			protoFiles: []string{"/source/googleapis/google/cloud/secretmanager/v1/service.proto"},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_out=/output",
				"--go-grpc_out=/output",
				"--go-grpc_opt=require_unimplemented_servers=false",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--go_gapic_opt=grpc-service-config=/source/googleapis/google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
				"--go_gapic_opt=transport=grpc+rest",
				"--go_gapic_opt=release-level=stable",
				"--go_gapic_opt=module=google.cloud.secretmanager.v1",
				"--go_gapic_opt=metadata",
				"--go_gapic_opt=rest-numeric-enums",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
		{
			name:    "beta api has preview release level",
			library: &config.Library{},
			api: &config.API{
				Path: "google/cloud/secretmanager/v1beta1",
				Go: &config.GoPackage{
					ImportPath: "cloud.google.com/go/secretmanager/apiv1beta1;secretmanager",
				},
			},
			protoFiles: []string{"/source/googleapis/google/cloud/secretmanager/v1beta1/service.proto"},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_out=/output",
				"--go-grpc_out=/output",
				"--go-grpc_opt=require_unimplemented_servers=false",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager/apiv1beta1;secretmanager",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/secretmanager/v1beta1/secretmanager_v1beta1.yaml",
				"--go_gapic_opt=transport=grpc+rest",
				"--go_gapic_opt=release-level=preview",
				"--go_gapic_opt=metadata",
				"--go_gapic_opt=rest-numeric-enums",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/secretmanager/v1beta1/service.proto",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildProtocArgs(test.library, test.api, googleapisDir, outputDir, test.protoFiles)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDetermineReleaseLevel(t *testing.T) {
	for _, test := range []struct {
		name            string
		importPath      string
		apiReleaseLevel string
		want            string
	}{
		{
			name:       "stable path",
			importPath: "cloud.google.com/go/secretmanager/apiv1",
			want:       "stable",
		},
		{
			name:       "alpha path",
			importPath: "cloud.google.com/go/secretmanager/apiv1alpha",
			want:       "preview",
		},
		{
			name:       "beta path",
			importPath: "cloud.google.com/go/secretmanager/apiv1beta1",
			want:       "preview",
		},
		{
			name:            "api alpha",
			importPath:      "cloud.google.com/go/secretmanager/apiv1",
			apiReleaseLevel: "alpha",
			want:            "preview",
		},
		{
			name:            "api beta",
			importPath:      "cloud.google.com/go/secretmanager/apiv1",
			apiReleaseLevel: "beta",
			want:            "preview",
		},
		{
			name:            "path takes precedence over api",
			importPath:      "cloud.google.com/go/secretmanager/apiv1beta1",
			apiReleaseLevel: "ga",
			want:            "preview",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := determineReleaseLevel(test.importPath, test.apiReleaseLevel)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestExtractAPIShortname(t *testing.T) {
	for _, test := range []struct {
		nameFull string
		want     string
	}{
		{"secretmanager.googleapis.com", "secretmanager"},
		{"compute.googleapis.com", "compute"},
		{"storage", "storage"},
	} {
		t.Run(test.nameFull, func(t *testing.T) {
			got := extractAPIShortname(test.nameFull)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestBuildDocURL(t *testing.T) {
	for _, test := range []struct {
		name       string
		modulePath string
		importPath string
		want       string
	}{
		{
			name:       "basic",
			modulePath: "cloud.google.com/go/secretmanager",
			importPath: "cloud.google.com/go/secretmanager/apiv1",
			want:       "https://cloud.google.com/go/docs/reference/cloud.google.com/go/secretmanager/latest/apiv1",
		},
		{
			name:       "empty module path",
			modulePath: "",
			importPath: "cloud.google.com/go/secretmanager/apiv1",
			want:       "",
		},
		{
			name:       "empty import path",
			modulePath: "cloud.google.com/go/secretmanager",
			importPath: "",
			want:       "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildDocURL(test.modulePath, test.importPath)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestDeriveGoGapicPackage(t *testing.T) {
	for _, test := range []struct {
		name        string
		libraryName string
		apiPath     string
		want        string
	}{
		{
			name:        "basic google/cloud path",
			libraryName: "accessapproval",
			apiPath:     "google/cloud/accessapproval/v1",
			want:        "cloud.google.com/go/accessapproval/apiv1;accessapproval",
		},
		{
			name:        "beta version",
			libraryName: "secretmanager",
			apiPath:     "google/cloud/secretmanager/v1beta1",
			want:        "cloud.google.com/go/secretmanager/apiv1beta1;secretmanager",
		},
		{
			name:        "alpha version",
			libraryName: "aiplatform",
			apiPath:     "google/cloud/aiplatform/v1alpha",
			want:        "cloud.google.com/go/aiplatform/apiv1alpha;aiplatform",
		},
		{
			name:        "nested path under google/cloud",
			libraryName: "bigquery",
			apiPath:     "google/cloud/bigquery/connection/v1",
			want:        "cloud.google.com/go/bigquery/connection/apiv1;connection",
		},
		{
			name:        "non-cloud path (google/ai)",
			libraryName: "ai",
			apiPath:     "google/ai/generativelanguage/v1",
			want:        "cloud.google.com/go/ai/generativelanguage/apiv1;generativelanguage",
		},
		{
			name:        "non-cloud path (google/analytics)",
			libraryName: "analytics",
			apiPath:     "google/analytics/admin/v1alpha",
			want:        "cloud.google.com/go/analytics/admin/apiv1alpha;admin",
		},
		{
			name:        "empty api path",
			libraryName: "secretmanager",
			apiPath:     "",
			want:        "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveGoGapicPackage(test.libraryName, test.apiPath)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestReadVersion(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "standard format",
			content: "package internal\n\nconst Version = \"1.2.3\"",
			want:    "1.2.3",
		},
		{
			name:    "with comments",
			content: "package internal\n\n// Version is the current release.\nconst Version = \"0.5.0\"",
			want:    "0.5.0",
		},
		{
			name:    "no version",
			content: "package internal\n\nconst Foo = \"bar\"",
			want:    "",
		},
		{
			name:    "empty file",
			content: "",
			want:    "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			internalDir := filepath.Join(dir, "internal")
			if err := os.MkdirAll(internalDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(internalDir, "version.go"), []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			if got := readVersion(dir); got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestReadVersion_NoFile(t *testing.T) {
	dir := t.TempDir()
	if got := readVersion(dir); got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestMoveSnippets(t *testing.T) {
	// Create a temporary directory to serve as the working directory.
	// This ensures internal/generated/snippets is created within the temp dir.
	workDir := t.TempDir()
	t.Chdir(workDir)

	outputDir := filepath.Join(workDir, "testlib")
	srcSnippetsDir := filepath.Join(outputDir, "internal", "snippets")
	if err := os.MkdirAll(srcSnippetsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test snippet files.
	snippetContent := "package snippets\n\nfunc Example() {}\n"
	if err := os.WriteFile(filepath.Join(srcSnippetsDir, "example.go"), []byte(snippetContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcSnippetsDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcSnippetsDir, "subdir", "nested.go"), []byte(snippetContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := moveSnippets(outputDir, "testlib"); err != nil {
		t.Fatal(err)
	}

	// Verify snippets moved to destination.
	destDir := filepath.Join("internal", "generated", "snippets", "testlib")
	if _, err := os.Stat(filepath.Join(destDir, "example.go")); err != nil {
		t.Errorf("expected example.go at destination: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "subdir", "nested.go")); err != nil {
		t.Errorf("expected subdir/nested.go at destination: %v", err)
	}

	// Verify source directory removed.
	if _, err := os.Stat(srcSnippetsDir); !os.IsNotExist(err) {
		t.Errorf("expected source snippets directory to be removed")
	}
}

func TestMoveSnippets_NoSnippets(t *testing.T) {
	outputDir := t.TempDir()
	// No internal/snippets directory exists.
	if err := moveSnippets(outputDir, "testlib"); err != nil {
		t.Errorf("expected no error when no snippets exist: %v", err)
	}
}

func TestCollectProtoFiles(t *testing.T) {
	dir := t.TempDir()

	// Create test proto files.
	if err := os.WriteFile(filepath.Join(dir, "service.proto"), []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "types.proto"), []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a non-proto file that should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# README"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := collectProtoFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 proto files, got %d", len(files))
	}
}

func TestCollectProtoFiles_SingleFile(t *testing.T) {
	dir := t.TempDir()
	protoFile := filepath.Join(dir, "service.proto")
	if err := os.WriteFile(protoFile, []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := collectProtoFiles(protoFile)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 proto file, got %d", len(files))
	}
	if files[0] != protoFile {
		t.Errorf("expected %s, got %s", protoFile, files[0])
	}
}

func TestCollectProtoFiles_NonProtoFile(t *testing.T) {
	dir := t.TempDir()
	txtFile := filepath.Join(dir, "readme.txt")
	if err := os.WriteFile(txtFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := collectProtoFiles(txtFile)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files for non-proto file, got %d", len(files))
	}
}

func TestCollectProtoFiles_NotExists(t *testing.T) {
	_, err := collectProtoFiles("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

