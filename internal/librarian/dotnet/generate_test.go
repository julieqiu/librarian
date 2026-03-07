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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

const googleapisDir = "../../testdata/googleapis"

func TestGenerateAPI(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("slow test: .NET GAPIC code generation")
	}
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "grpc_csharp_plugin")
	testhelper.RequireCommand(t, "Google.Api.Generator")

	outdir := t.TempDir()
	cfg := &config.Config{
		Language: config.LanguageDotnet,
	}
	library := &config.Library{
		Name:   "Google.Cloud.SecretManager.V1",
		Output: outdir,
	}
	api := &config.API{Path: "google/cloud/secretmanager/v1"}
	if err := generateAPI(t.Context(), cfg, api, library, googleapisDir, outdir); err != nil {
		t.Fatal(err)
	}
	// Verify that some generated .g.cs files exist in the library output.
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
	testhelper.RequireCommand(t, "grpc_csharp_plugin")
	testhelper.RequireCommand(t, "Google.Api.Generator")

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

