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
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
)

// buildProtocCommand constructs the full protoc command arguments for a given API.
func buildProtocCommand(library *config.Library, channelPath, sourceDir, serviceConfigPath, grpcServiceConfigPath string, nestedProtos []string) ([]string, error) {
	// Gather all .proto files in the API's source directory (but not in subdirectories).
	apiServiceDir := filepath.Join(sourceDir, channelPath)
	entries, err := os.ReadDir(apiServiceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read API source directory %s: %w", apiServiceDir, err)
	}

	var protoFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".proto" {
			protoFiles = append(protoFiles, filepath.Join(apiServiceDir, entry.Name()))
		}
	}

	for _, nestedProto := range nestedProtos {
		protoFiles = append(protoFiles, filepath.Join(apiServiceDir, nestedProto))
	}
	if len(protoFiles) == 0 {
		return nil, fmt.Errorf("no .proto files found in %s", apiServiceDir)
	}

	// Get Go-specific config
	hasGAPIC := library.Go != nil && library.Go.HasGAPIC
	hasGoGRPC := library.Go != nil && library.Go.HasGoGRPC
	modulePath := library.Go.ModulePath
	if modulePath == "" && library.Go != nil {
		// Fallback if not set
		modulePath = "cloud.google.com/go/" + library.Name
	}

	// Construct the protoc command arguments.
	var gapicOpts []string
	if hasGAPIC {
		gapicOpts = append(gapicOpts, "go-gapic-package="+modulePath)
		if serviceConfigPath != "" {
			gapicOpts = append(gapicOpts, fmt.Sprintf("api-service-config=%s", serviceConfigPath))
		}
		if grpcServiceConfigPath != "" {
			gapicOpts = append(gapicOpts, fmt.Sprintf("grpc-service-config=%s", grpcServiceConfigPath))
		}
		if library.Transport != "" {
			gapicOpts = append(gapicOpts, fmt.Sprintf("transport=%s", library.Transport))
		}
		if library.ReleaseLevel != "" {
			gapicOpts = append(gapicOpts, fmt.Sprintf("release-level=%s", library.ReleaseLevel))
		}
		if library.Go != nil && library.Go.Metadata {
			gapicOpts = append(gapicOpts, "metadata")
		}
		if library.Go != nil && library.Go.Diregapic {
			gapicOpts = append(gapicOpts, "diregapic")
		}
		if library.Go != nil && library.Go.RESTNumericEnums {
			gapicOpts = append(gapicOpts, "rest-numeric-enums")
		}
	}

	args := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
	}
	// All generated files are written to the output directory.
	args = append(args, "--go_out="+library.Output)
	if hasGoGRPC {
		args = append(args, "--go-grpc_out="+library.Output, "--go-grpc_opt=require_unimplemented_servers=false")
	}
	if hasGAPIC {
		args = append(args, "--go_gapic_out="+library.Output)

		for _, opt := range gapicOpts {
			args = append(args, "--go_gapic_opt="+opt)
		}
	}
	args = append(args,
		// The -I flag specifies the import path for protoc. All protos
		// and their dependencies must be findable from this path.
		// The /source mount contains the complete googleapis repository.
		"-I="+sourceDir,
	)

	args = append(args, protoFiles...)

	return args, nil
}
