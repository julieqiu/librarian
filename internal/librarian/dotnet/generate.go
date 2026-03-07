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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// GenerateLibraries generates all the given libraries in sequence.
func GenerateLibraries(ctx context.Context, cfg *config.Config, libraries []*config.Library, googleapisDir string) error {
	for _, library := range libraries {
		if err := generate(ctx, cfg, library, googleapisDir); err != nil {
			return err
		}
	}
	return nil
}

// generate generates a .NET client library.
func generate(ctx context.Context, cfg *config.Config, library *config.Library, googleapisDir string) error {
	if len(library.APIs) == 0 {
		return fmt.Errorf("failed to generate library: no apis configured for library %q", library.Name)
	}
	outdir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory path: %w", err)
	}
	googleapisDir, err = filepath.Abs(googleapisDir)
	if err != nil {
		return fmt.Errorf("failed to resolve googleapis directory path: %w", err)
	}
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	for _, api := range library.APIs {
		if err := generateAPI(ctx, cfg, api, library, googleapisDir, outdir); err != nil {
			return fmt.Errorf("failed to generate api %q: %w", api.Path, err)
		}
	}
	if err := runPostgeneration(ctx, cfg, library, googleapisDir, outdir); err != nil {
		return fmt.Errorf("failed to run postgeneration for library %q: %w", library.Name, err)
	}
	return nil
}

func generateAPI(ctx context.Context, cfg *config.Config, api *config.API, library *config.Library, googleapisDir, outdir string) error {
	// Determine the proto source directory. If pregeneration mutations are
	// configured, copy protos to a temp directory and apply mutations there.
	protoDir := googleapisDir
	if library.Dotnet != nil && len(library.Dotnet.Pregeneration) > 0 {
		tmpDir, err := os.MkdirTemp("", "dotnet-protoc-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		if err := copyProtoFiles(filepath.Join(googleapisDir, api.Path), filepath.Join(tmpDir, api.Path)); err != nil {
			return fmt.Errorf("failed to copy proto files: %w", err)
		}
		if err := applyPregeneration(filepath.Join(tmpDir, api.Path), library.Dotnet.Pregeneration); err != nil {
			return fmt.Errorf("failed to apply pregeneration: %w", err)
		}
		protoDir = tmpDir
	}

	apiProtos, err := filepath.Glob(filepath.Join(protoDir, api.Path, "*.proto"))
	if err != nil {
		return fmt.Errorf("failed to find protos: %w", err)
	}
	if len(apiProtos) == 0 {
		return fmt.Errorf("no protos found in api %q", api.Path)
	}

	var preinstalled map[string]string
	if cfg.Release != nil {
		preinstalled = cfg.Release.Preinstalled
	}

	isProtoOnly := library.Dotnet != nil && library.Dotnet.Generator == "proto"

	// Pass 1: proto messages and gRPC stubs (proto-only skips gRPC).
	pass1Args := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
	}
	// If protos were mutated in a temp dir, add it as an include path so
	// protoc can resolve the modified files.
	if protoDir != googleapisDir {
		pass1Args = append(pass1Args, "-I="+protoDir)
	}
	pass1Args = append(pass1Args,
		fmt.Sprintf("--csharp_out=%s", filepath.Join(outdir, library.Name)),
		fmt.Sprintf("--csharp_opt=base_namespace=%s,file_extension=.g.cs", library.Name),
	)
	if !isProtoOnly {
		grpcPlugin := command.GetExecutablePath(preinstalled, "grpc_csharp_plugin")
		pass1Args = append(pass1Args,
			fmt.Sprintf("--grpc_out=%s", filepath.Join(outdir, library.Name)),
			fmt.Sprintf("--grpc_opt=base_namespace=%s,file_suffix=Grpc.g.cs", library.Name),
			fmt.Sprintf("--plugin=protoc-gen-grpc=%s", grpcPlugin),
		)
	}
	pass1Args = append(pass1Args, apiProtos...)

	if err := command.Run(ctx, pass1Args[0], pass1Args[1:]...); err != nil {
		return fmt.Errorf("failed to run protoc (pass 1): %w", err)
	}

	// Pass 2: GAPIC client. Skipped for proto-only libraries.
	if isProtoOnly {
		return nil
	}

	gapicPlugin := command.GetExecutablePath(preinstalled, "Google.Api.Generator")
	pass2Args := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
		"-I=" + googleapisDir,
	}
	if protoDir != googleapisDir {
		pass2Args = append(pass2Args, "-I="+protoDir)
	}
	pass2Args = append(pass2Args,
		fmt.Sprintf("--gapic_out=%s", outdir),
		fmt.Sprintf("--plugin=protoc-gen-gapic=%s", gapicPlugin),
		"--gapic_opt=transport=grpc+rest",
		"--gapic_opt=rest-numeric-enums=true",
	)

	apiCfg, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageDotnet)
	if err != nil {
		return fmt.Errorf("failed to find service config: %w", err)
	}
	if apiCfg != nil && apiCfg.ServiceConfig != "" {
		pass2Args = append(pass2Args, fmt.Sprintf("--gapic_opt=service-config=%s", filepath.Join(googleapisDir, apiCfg.ServiceConfig)))
	}

	pass2Args = append(pass2Args, apiProtos...)

	if err := command.Run(ctx, pass2Args[0], pass2Args[1:]...); err != nil {
		return fmt.Errorf("failed to run protoc (pass 2): %w", err)
	}

	// If any RPC renames specified a WireName, fix the generated gRPC stubs.
	if library.Dotnet != nil {
		if err := applyRPCWireNameFixes(outdir, library.Name, library.Dotnet.Pregeneration); err != nil {
			return fmt.Errorf("failed to apply RPC wire name fixes: %w", err)
		}
	}
	return nil
}

// copyProtoFiles copies all .proto files from src to dst.
func copyProtoFiles(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".proto") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, entry.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dst, entry.Name()), data, 0644); err != nil {
			return err
		}
	}
	return nil
}

// applyPregeneration applies declarative proto mutations to proto files
// in the given directory.
func applyPregeneration(protoDir string, pregens []*config.DotnetPregeneration) error {
	for _, pregen := range pregens {
		if pregen.RenameMessage != nil {
			if err := applyRenameMessage(protoDir, pregen.RenameMessage); err != nil {
				return err
			}
		}
		if pregen.RemoveField != nil {
			if err := applyRemoveField(protoDir, pregen.RemoveField); err != nil {
				return err
			}
		}
		if pregen.RenameRPC != nil {
			if err := applyRenameRPC(protoDir, pregen.RenameRPC); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyRenameMessage renames a message definition and all references to it
// in all proto files in the given directory.
func applyRenameMessage(protoDir string, rename *config.DotnetRenameMessage) error {
	return walkProtoFiles(protoDir, func(path string, content string) (string, error) {
		// Rename the message definition.
		content = strings.ReplaceAll(content,
			"message "+rename.From+" {",
			"message "+rename.To+" {")
		content = strings.ReplaceAll(content,
			"message "+rename.From+"{",
			"message "+rename.To+"{")

		// Rename references as field types. Match the old name when it
		// appears as a type (preceded by a space or the start of a line,
		// followed by a space).
		re := regexp.MustCompile(`(\s)` + regexp.QuoteMeta(rename.From) + `(\s)`)
		content = re.ReplaceAllString(content, "${1}"+rename.To+"${2}")
		return content, nil
	})
}

// applyRemoveField removes a field from a message in all proto files in the
// given directory.
func applyRemoveField(protoDir string, remove *config.DotnetRemoveField) error {
	return walkProtoFiles(protoDir, func(path string, content string) (string, error) {
		lines := strings.Split(content, "\n")
		var result []string
		inMessage := false
		braceDepth := 0
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !inMessage && (trimmed == "message "+remove.Message+" {" || trimmed == "message "+remove.Message+"{") {
				inMessage = true
				braceDepth = 0
			}
			if inMessage {
				braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
				// Remove the field line within the target message (at depth 1).
				if braceDepth == 1 && strings.Contains(trimmed, remove.Field) {
					// Check that this looks like a field declaration containing the field name.
					fieldPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(remove.Field) + `\b`)
					if fieldPattern.MatchString(trimmed) {
						continue
					}
				}
				if braceDepth <= 0 {
					inMessage = false
				}
			}
			result = append(result, line)
		}
		return strings.Join(result, "\n"), nil
	})
}

// applyRenameRPC renames an RPC method in all proto files in the given
// directory.
func applyRenameRPC(protoDir string, rename *config.DotnetRenameRPC) error {
	return walkProtoFiles(protoDir, func(path string, content string) (string, error) {
		content = strings.ReplaceAll(content,
			"rpc "+rename.From+"(",
			"rpc "+rename.To+"(")
		return content, nil
	})
}

// walkProtoFiles reads each .proto file in the directory, applies the
// transform function, and writes the result back if changed.
func walkProtoFiles(dir string, transform func(path string, content string) (string, error)) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".proto") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		original := string(data)
		modified, err := transform(path, original)
		if err != nil {
			return err
		}
		if modified != original {
			if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyRPCWireNameFixes fixes gRPC method descriptors in generated .g.cs files
// when an RPC has been renamed but needs to keep the original wire name.
func applyRPCWireNameFixes(outdir, libraryName string, pregens []*config.DotnetPregeneration) error {
	for _, pregen := range pregens {
		if pregen.RenameRPC == nil || pregen.RenameRPC.WireName == "" {
			continue
		}
		rename := pregen.RenameRPC
		// Walk all .g.cs files and replace the renamed method name in gRPC
		// method descriptors with the original wire name.
		libDir := filepath.Join(outdir, libraryName)
		err := filepath.WalkDir(libDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".g.cs") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			original := string(data)
			// In gRPC generated code, the method descriptor contains the
			// method name as a string literal. Replace the renamed name
			// with the wire name in method descriptor strings.
			modified := strings.ReplaceAll(original,
				fmt.Sprintf(`"%s"`, rename.To),
				fmt.Sprintf(`"%s"`, rename.WireName))
			if modified != original {
				if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// runPostgeneration runs post-generation actions for a library.
func runPostgeneration(ctx context.Context, cfg *config.Config, library *config.Library, googleapisDir, outdir string) error {
	if library.Dotnet == nil {
		return nil
	}
	var preinstalled map[string]string
	if cfg.Release != nil {
		preinstalled = cfg.Release.Preinstalled
	}

	for _, post := range library.Dotnet.Postgeneration {
		if post.Run != "" {
			if err := command.Run(ctx, "bash", "-c", post.Run); err != nil {
				return fmt.Errorf("failed to run postgeneration command %q: %w", post.Run, err)
			}
		}
		if post.ExtraProto != "" {
			grpcPlugin := command.GetExecutablePath(preinstalled, "grpc_csharp_plugin")
			protoPath := filepath.Join(googleapisDir, post.ExtraProto)
			args := []string{
				"protoc",
				"--experimental_allow_proto3_optional",
				"-I=" + googleapisDir,
				fmt.Sprintf("--csharp_out=%s", filepath.Join(outdir, library.Name)),
				fmt.Sprintf("--csharp_opt=base_namespace=%s,file_extension=.g.cs", library.Name),
				fmt.Sprintf("--grpc_out=%s", filepath.Join(outdir, library.Name)),
				fmt.Sprintf("--grpc_opt=base_namespace=%s,file_suffix=Grpc.g.cs", library.Name),
				fmt.Sprintf("--plugin=protoc-gen-grpc=%s", grpcPlugin),
				protoPath,
			}
			if err := command.Run(ctx, args[0], args[1:]...); err != nil {
				return fmt.Errorf("failed to compile extra proto %q: %w", post.ExtraProto, err)
			}
		}
	}
	return nil
}
