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

package golang

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestBuildProtocArgs(t *testing.T) {
	googleapisDir := "/source/googleapis"
	outputDir := "/output"
	protoFiles := []string{"/source/googleapis/google/cloud/secretmanager/v1/service.proto"}

	boolTrue := true
	boolFalse := false

	for _, test := range []struct {
		name    string
		library *config.Library
		api     *config.API
		want    []string
	}{
		{
			name: "basic GAPIC",
			library: &config.Library{
				Transport:    "grpc+rest",
				ReleaseLevel: "stable",
			},
			api: &config.API{
				Path:          "google/cloud/secretmanager/v1",
				ServiceConfig: "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Go: &config.GoPackage{
					ImportPath: "cloud.google.com/go/secretmanager/apiv1;secretmanager",
				},
			},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_v1_out=/output",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--go_gapic_opt=transport=grpc+rest",
				"--go_gapic_opt=release-level=stable",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
		{
			name: "with go_grpc",
			library: &config.Library{
				Transport: "grpc",
			},
			api: &config.API{
				Path:          "google/cloud/secretmanager/v1",
				ServiceConfig: "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Go: &config.GoPackage{
					ImportPath: "cloud.google.com/go/secretmanager/apiv1;secretmanager",
					GoGRPC:     &boolTrue,
				},
			},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_out=/output",
				"--go-grpc_out=/output",
				"--go-grpc_opt=require_unimplemented_servers=false",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--go_gapic_opt=transport=grpc",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
		{
			name: "with legacy grpc",
			library: &config.Library{
				Transport: "grpc",
			},
			api: &config.API{
				Path:          "google/cloud/secretmanager/v1",
				ServiceConfig: "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Go: &config.GoPackage{
					ImportPath: "cloud.google.com/go/secretmanager/apiv1;secretmanager",
					LegacyGRPC: true,
				},
			},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_v1_out=/output",
				"--go_v1_opt=plugins=grpc",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--go_gapic_opt=transport=grpc",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
		{
			name: "with metadata",
			library: &config.Library{
				Transport: "grpc",
			},
			api: &config.API{
				Path:          "google/cloud/secretmanager/v1",
				ServiceConfig: "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Metadata:      &boolTrue,
				Go: &config.GoPackage{
					ImportPath: "cloud.google.com/go/secretmanager/apiv1;secretmanager",
				},
			},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_v1_out=/output",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--go_gapic_opt=transport=grpc",
				"--go_gapic_opt=metadata",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
		{
			name: "with DIREGAPIC",
			library: &config.Library{
				Transport: "rest",
			},
			api: &config.API{
				Path:             "google/cloud/compute/v1",
				ServiceConfig:    "google/cloud/compute/v1/compute_v1.yaml",
				DIREGAPIC:        true,
				RESTNumericEnums: &boolTrue,
				Go: &config.GoPackage{
					ImportPath: "cloud.google.com/go/compute/apiv1;compute",
				},
			},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_v1_out=/output",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/compute/apiv1;compute",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/compute/v1/compute_v1.yaml",
				"--go_gapic_opt=transport=rest",
				"--go_gapic_opt=diregapic",
				"--go_gapic_opt=rest-numeric-enums",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
		{
			name: "with grpc service config",
			library: &config.Library{
				Transport: "grpc",
			},
			api: &config.API{
				Path:              "google/cloud/asset/v1",
				ServiceConfig:     "google/cloud/asset/v1/cloudasset_v1.yaml",
				GRPCServiceConfig: "cloudasset_grpc_service_config.json",
				Go: &config.GoPackage{
					ImportPath: "cloud.google.com/go/asset/apiv1;asset",
				},
			},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_v1_out=/output",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/asset/apiv1;asset",
				"--go_gapic_opt=api-service-config=/source/googleapis/google/cloud/asset/v1/cloudasset_v1.yaml",
				"--go_gapic_opt=grpc-service-config=/source/googleapis/google/cloud/asset/v1/cloudasset_grpc_service_config.json",
				"--go_gapic_opt=transport=grpc",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
		{
			name:    "metadata false does not add flag",
			library: &config.Library{},
			api: &config.API{
				Path:     "google/cloud/secretmanager/v1",
				Metadata: &boolFalse,
				Go: &config.GoPackage{
					ImportPath: "cloud.google.com/go/secretmanager/apiv1;secretmanager",
				},
			},
			want: []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"--go_v1_out=/output",
				"--go_gapic_out=/output",
				"--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
				"-I=/source/googleapis",
				"/source/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildProtocArgs(test.library, test.api, googleapisDir, outputDir, protoFiles)
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
		configuredLevel string
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
			name:            "alpha configured",
			importPath:      "cloud.google.com/go/secretmanager/apiv1",
			configuredLevel: "alpha",
			want:            "preview",
		},
		{
			name:            "beta configured",
			importPath:      "cloud.google.com/go/secretmanager/apiv1",
			configuredLevel: "beta",
			want:            "preview",
		},
		{
			name:            "path takes precedence",
			importPath:      "cloud.google.com/go/secretmanager/apiv1beta1",
			configuredLevel: "stable",
			want:            "preview",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := determineReleaseLevel(test.importPath, test.configuredLevel)
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
