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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

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
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(rename.From) + `\b`)
	return walkProtoFiles(protoDir, func(content string) (string, error) {
		content = re.ReplaceAllString(content, rename.To)
		return content, nil
	})
}

// applyRemoveField removes a field from a message in all proto files in the
// given directory.
func applyRemoveField(protoDir string, remove *config.DotnetRemoveField) error {
	messagePattern := regexp.MustCompile(`^\s*message\s+` + regexp.QuoteMeta(remove.Message) + `\s*\{`)
	fieldPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(remove.Field) + `\b`)
	return walkProtoFiles(protoDir, func(content string) (string, error) {
		lines := strings.Split(content, "\n")
		var result []string
		inMessage := false
		braceDepth := 0
		for _, line := range lines {
			if !inMessage && messagePattern.MatchString(line) {
				inMessage = true
				braceDepth = 0
			}
			if inMessage {
				braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
				// Remove the field line within the target message (at depth 1).
				if braceDepth == 1 && fieldPattern.MatchString(line) {
					continue
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
	re := regexp.MustCompile(`\brpc\s+` + regexp.QuoteMeta(rename.From) + `\s*\(`)
	return walkProtoFiles(protoDir, func(content string) (string, error) {
		content = re.ReplaceAllString(content, "rpc "+rename.To+"(")
		return content, nil
	})
}

// walkProtoFiles reads each .proto file in the directory, applies the
// transform function, and writes the result back if changed.
func walkProtoFiles(dir string, transform func(content string) (string, error)) error {
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
		modified, err := transform(original)
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
