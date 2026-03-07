// Copyright 2026 Google LLC
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

package dotnet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

const googleapisDir = "../../testdata/googleapis"

func TestCopyProtoFiles(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "output")

	if err := os.WriteFile(filepath.Join(src, "service.proto"), []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "types.proto"), []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "README.md"), []byte("# README"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "BUILD.bazel"), []byte("load()"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyProtoFiles(src, dst); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dst)
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	want := []string{"service.proto", "types.proto"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestApplyRenameMessage(t *testing.T) {
	dir := t.TempDir()
	proto := `syntax = "proto3";
package google.cloud.aiplatform.v1;

message Schema {
  string name = 1;
  Schema nested = 2;
}

message PredictRequest {
  Schema schema = 1;
}
`
	if err := os.WriteFile(filepath.Join(dir, "schema.proto"), []byte(proto), 0644); err != nil {
		t.Fatal(err)
	}

	rename := &config.DotnetRenameMessage{
		From: "Schema",
		To:   "OpenApiSchema",
	}
	if err := applyRenameMessage(dir, rename); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "schema.proto"))
	if err != nil {
		t.Fatal(err)
	}
	want := `syntax = "proto3";
package google.cloud.aiplatform.v1;

message OpenApiSchema {
  string name = 1;
  OpenApiSchema nested = 2;
}

message PredictRequest {
  OpenApiSchema schema = 1;
}
`
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestApplyRemoveField(t *testing.T) {
	dir := t.TempDir()
	proto := `syntax = "proto3";
package google.cloud.aiplatform.v1;

message QueryDeployedModelsResponse {
  string name = 1;
  repeated DeployedModel deployed_models = 2;
  string next_page_token = 3;
}
`
	if err := os.WriteFile(filepath.Join(dir, "service.proto"), []byte(proto), 0644); err != nil {
		t.Fatal(err)
	}

	remove := &config.DotnetRemoveField{
		Message: "QueryDeployedModelsResponse",
		Field:   "deployed_models",
	}
	if err := applyRemoveField(dir, remove); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "service.proto"))
	if err != nil {
		t.Fatal(err)
	}
	want := `syntax = "proto3";
package google.cloud.aiplatform.v1;

message QueryDeployedModelsResponse {
  string name = 1;
  string next_page_token = 3;
}
`
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestApplyRenameRPC(t *testing.T) {
	dir := t.TempDir()
	proto := `syntax = "proto3";
package google.cloud.logging.v2;

service ConfigServiceV2 {
  rpc UpdateBucketAsync(UpdateBucketRequest) returns (Operation);
  rpc GetBucket(GetBucketRequest) returns (LogBucket);
}
`
	if err := os.WriteFile(filepath.Join(dir, "logging.proto"), []byte(proto), 0644); err != nil {
		t.Fatal(err)
	}

	rename := &config.DotnetRenameRPC{
		From: "UpdateBucketAsync",
		To:   "UpdateBucketLongRunning",
	}
	if err := applyRenameRPC(dir, rename); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "logging.proto"))
	if err != nil {
		t.Fatal(err)
	}
	want := `syntax = "proto3";
package google.cloud.logging.v2;

service ConfigServiceV2 {
  rpc UpdateBucketLongRunning(UpdateBucketRequest) returns (Operation);
  rpc GetBucket(GetBucketRequest) returns (LogBucket);
}
`
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestApplyRPCWireNameFixes(t *testing.T) {
	dir := t.TempDir()
	libName := "Google.Cloud.Logging.V2"
	libDir := filepath.Join(dir, libName)
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}

	csContent := `static readonly Method<UpdateBucketRequest, Operation> __Method_UpdateBucketLongRunning = new Method<UpdateBucketRequest, Operation>("UpdateBucketLongRunning");`
	if err := os.WriteFile(filepath.Join(libDir, "ServiceGrpc.g.cs"), []byte(csContent), 0644); err != nil {
		t.Fatal(err)
	}

	pregens := []*config.DotnetPregeneration{
		{
			RenameRPC: &config.DotnetRenameRPC{
				From:     "UpdateBucketAsync",
				To:       "UpdateBucketLongRunning",
				WireName: "UpdateBucketAsync",
			},
		},
	}
	if err := applyRPCWireNameFixes(dir, libName, pregens); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(libDir, "ServiceGrpc.g.cs"))
	if err != nil {
		t.Fatal(err)
	}
	want := `static readonly Method<UpdateBucketRequest, Operation> __Method_UpdateBucketLongRunning = new Method<UpdateBucketRequest, Operation>("UpdateBucketAsync");`
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateAPI(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("slow test: .NET code generation")
	}
	testhelper.RequireCommand(t, "protoc")

	outdir := t.TempDir()
	cfg := &config.Config{Language: config.LanguageDotnet}
	library := &config.Library{
		Name:   "Google.Cloud.SecretManager.V1",
		Output: outdir,
		// Use proto-only mode because gapic-generator-csharp has stricter
		// proto requirements than our shared testdata supports.
		Dotnet: &config.DotnetPackage{Generator: "proto"},
	}
	api := &config.API{Path: "google/cloud/secretmanager/v1"}
	if err := generateAPI(t.Context(), cfg, api, library, googleapisDir, outdir); err != nil {
		t.Fatal(err)
	}
	libDir := filepath.Join(outdir, library.Name)
	entries, err := os.ReadDir(libDir)
	if err != nil {
		t.Fatal(err)
	}
	var csFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".g.cs") {
			csFiles = append(csFiles, e.Name())
		}
	}
	if len(csFiles) == 0 {
		t.Error("expected generated .g.cs files, got none")
	}
}

func TestGenerateAPI_ProtoOnly(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("slow test: .NET proto-only code generation")
	}
	testhelper.RequireCommand(t, "protoc")

	outdir := t.TempDir()
	cfg := &config.Config{
		Language: config.LanguageDotnet,
	}
	library := &config.Library{
		Name:   "Google.Cloud.SecretManager.V1",
		Output: outdir,
		Dotnet: &config.DotnetPackage{
			Generator: "proto",
		},
	}
	api := &config.API{Path: "google/cloud/secretmanager/v1"}
	if err := generateAPI(t.Context(), cfg, api, library, googleapisDir, outdir); err != nil {
		t.Fatal(err)
	}
	// Verify that proto .g.cs files exist but no Grpc .g.cs files.
	libDir := filepath.Join(outdir, library.Name)
	entries, err := os.ReadDir(libDir)
	if err != nil {
		t.Fatal(err)
	}
	var hasProtoCS, hasGrpcCS bool
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".g.cs") && !strings.Contains(e.Name(), "Grpc") {
			hasProtoCS = true
		}
		if strings.Contains(e.Name(), "Grpc.g.cs") {
			hasGrpcCS = true
		}
	}
	if !hasProtoCS {
		t.Error("expected proto .g.cs files, got none")
	}
	if hasGrpcCS {
		t.Error("proto-only generation should not produce Grpc .g.cs files")
	}
}

func TestGenerate_NoAPIs(t *testing.T) {
	cfg := &config.Config{
		Language: config.LanguageDotnet,
	}
	library := &config.Library{
		Name:   "Google.Cloud.SecretManager.V1",
		Output: t.TempDir(),
	}
	err := generate(t.Context(), cfg, library, googleapisDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGenerateLibraries(t *testing.T) {
	if testing.Short() {
		t.Skip("slow test: .NET code generation")
	}
	testhelper.RequireCommand(t, "protoc")

	outdir := t.TempDir()
	cfg := &config.Config{
		Language: config.LanguageDotnet,
	}
	libraries := []*config.Library{
		{
			Name:   "Google.Cloud.SecretManager.V1",
			Output: filepath.Join(outdir, "Google.Cloud.SecretManager.V1"),
			APIs: []*config.API{
				{Path: "google/cloud/secretmanager/v1"},
			},
			// Use proto-only mode because gapic-generator-csharp has stricter
			// proto requirements than our shared testdata supports.
			Dotnet: &config.DotnetPackage{Generator: "proto"},
		},
	}
	if err := GenerateLibraries(t.Context(), cfg, libraries, googleapisDir); err != nil {
		t.Fatal(err)
	}
	libDir := filepath.Join(outdir, "Google.Cloud.SecretManager.V1", "Google.Cloud.SecretManager.V1")
	if _, err := os.Stat(libDir); err != nil {
		t.Errorf("expected library output directory %s to exist: %v", libDir, err)
	}
}

func TestGenerateLibraries_Error(t *testing.T) {
	cfg := &config.Config{
		Language: config.LanguageDotnet,
	}
	libraries := []*config.Library{
		{
			Name:   "Google.Cloud.SecretManager.V1",
			Output: "/../bad-output",
			APIs: []*config.API{
				{Path: "google/cloud/secretmanager/v1"},
			},
		},
	}
	err := GenerateLibraries(t.Context(), cfg, libraries, googleapisDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestApplyPregeneration(t *testing.T) {
	dir := t.TempDir()
	proto := `syntax = "proto3";
package google.cloud.aiplatform.v1;

message Schema {
  string name = 1;
  Schema nested = 2;
  repeated DeployedModel deployed_models = 3;
}

service PredictionService {
  rpc UpdateBucketAsync(UpdateBucketRequest) returns (Operation);
}
`
	if err := os.WriteFile(filepath.Join(dir, "service.proto"), []byte(proto), 0644); err != nil {
		t.Fatal(err)
	}

	pregens := []*config.DotnetPregeneration{
		{
			RenameMessage: &config.DotnetRenameMessage{
				From: "Schema",
				To:   "OpenApiSchema",
			},
		},
		{
			RemoveField: &config.DotnetRemoveField{
				Message: "OpenApiSchema",
				Field:   "deployed_models",
			},
		},
		{
			RenameRPC: &config.DotnetRenameRPC{
				From: "UpdateBucketAsync",
				To:   "UpdateBucketLongRunning",
			},
		},
	}
	if err := applyPregeneration(dir, pregens); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "service.proto"))
	if err != nil {
		t.Fatal(err)
	}
	want := `syntax = "proto3";
package google.cloud.aiplatform.v1;

message OpenApiSchema {
  string name = 1;
  OpenApiSchema nested = 2;
}

service PredictionService {
  rpc UpdateBucketLongRunning(UpdateBucketRequest) returns (Operation);
}
`
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
