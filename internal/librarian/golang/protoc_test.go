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

func TestBuildProtocCommand(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "google/cloud/secretmanager/v1")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "service.proto"), []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "resources.proto"), []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name                  string
		library               *config.Library
		channelPath           string
		serviceConfigPath     string
		grpcServiceConfigPath string
		nestedProtos          []string
		wantContains          []string
		wantNotContains       []string
	}{
		{
			name: "basic proto only",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "/output",
				Go: &config.GoModule{
					ModulePath: "cloud.google.com/go/secretmanager",
				},
			},
			channelPath: "google/cloud/secretmanager/v1",
			wantContains: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_out=/output",
				"-I=" + tmpDir,
			},
			wantNotContains: []string{
				"--go_gapic_out",
				"--go-grpc_out",
			},
		},
		{
			name: "with GAPIC",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "/output",
				Go: &config.GoModule{
					ModulePath: "cloud.google.com/go/secretmanager",
					HasGAPIC:   true,
				},
			},
			channelPath: "google/cloud/secretmanager/v1",
			wantContains: []string{
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager",
			},
		},
		{
			name: "with go-grpc",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "/output",
				Go: &config.GoModule{
					ModulePath: "cloud.google.com/go/secretmanager",
					HasGoGRPC:  true,
				},
			},
			channelPath: "google/cloud/secretmanager/v1",
			wantContains: []string{
				"--go-grpc_out=/output",
				"--go-grpc_opt=require_unimplemented_servers=false",
			},
		},
		{
			name: "with service configs",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "/output",
				Go: &config.GoModule{
					ModulePath: "cloud.google.com/go/secretmanager",
					HasGAPIC:   true,
				},
			},
			channelPath:           "google/cloud/secretmanager/v1",
			serviceConfigPath:     "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			grpcServiceConfigPath: "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
			wantContains: []string{
				"--go_gapic_opt=api-service-config=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--go_gapic_opt=grpc-service-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
			},
		},
		{
			name: "with transport and release level",
			library: &config.Library{
				Name:         "secretmanager",
				Output:       "/output",
				Transport:    "grpc+rest",
				ReleaseLevel: "stable",
				Go: &config.GoModule{
					ModulePath: "cloud.google.com/go/secretmanager",
					HasGAPIC:   true,
				},
			},
			channelPath: "google/cloud/secretmanager/v1",
			wantContains: []string{
				"--go_gapic_opt=transport=grpc+rest",
				"--go_gapic_opt=release-level=stable",
			},
		},
		{
			name: "with Go options",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "/output",
				Go: &config.GoModule{
					ModulePath:       "cloud.google.com/go/secretmanager",
					HasGAPIC:         true,
					Metadata:         true,
					Diregapic:        true,
					RESTNumericEnums: true,
				},
			},
			channelPath: "google/cloud/secretmanager/v1",
			wantContains: []string{
				"--go_gapic_opt=metadata",
				"--go_gapic_opt=diregapic",
				"--go_gapic_opt=rest-numeric-enums",
			},
		},
		{
			name: "with nested protos",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "/output",
				Go: &config.GoModule{
					ModulePath: "cloud.google.com/go/secretmanager",
				},
			},
			channelPath:  "google/cloud/secretmanager/v1",
			nestedProtos: []string{"nested/extra.proto"},
			wantContains: []string{
				filepath.Join(tmpDir, "google/cloud/secretmanager/v1/nested/extra.proto"),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			args, err := buildProtocCommand(test.library, test.channelPath, tmpDir, test.serviceConfigPath, test.grpcServiceConfigPath, test.nestedProtos)
			if err != nil {
				t.Fatal(err)
			}

			argsStr := strings.Join(args, " ")
			for _, want := range test.wantContains {
				if !strings.Contains(argsStr, want) {
					t.Errorf("args missing %q\ngot: %v", want, args)
				}
			}
			for _, notWant := range test.wantNotContains {
				if strings.Contains(argsStr, notWant) {
					t.Errorf("args should not contain %q\ngot: %v", notWant, args)
				}
			}
		})
	}
}

func TestBuildProtocCommand_NoProtos(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "google/cloud/empty/v1")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name:   "empty",
		Output: "/output",
		Go: &config.GoModule{
			ModulePath: "cloud.google.com/go/empty",
		},
	}

	_, err := buildProtocCommand(library, "google/cloud/empty/v1", tmpDir, "", "", nil)
	if err == nil {
		t.Fatal("expected error for directory with no proto files")
	}
	if !strings.Contains(err.Error(), "no .proto files found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBuildProtocCommand_ModulePathFallback(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "google/cloud/test/v1")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "test.proto"), []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name:   "test",
		Output: "/output",
		Go: &config.GoModule{
			HasGAPIC: true,
		},
	}

	args, err := buildProtocCommand(library, "google/cloud/test/v1", tmpDir, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "go-gapic-package=cloud.google.com/go/test") {
		t.Errorf("expected fallback module path, got: %v", args)
	}
}
