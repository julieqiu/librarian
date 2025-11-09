// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestInitRunner(t *testing.T) {
	for _, test := range []struct {
		name     string
		language string
		wantErr  bool
	}{
		{
			name:     "no language",
			language: "",
			wantErr:  false,
		},
		{
			name:     "python",
			language: "python",
			wantErr:  false,
		},
		{
			name:     "go",
			language: "go",
			wantErr:  false,
		},
		{
			name:     "rust",
			language: "rust",
			wantErr:  false,
		},
		{
			name:     "dart",
			language: "dart",
			wantErr:  false,
		},
		{
			name:     "invalid language",
			language: "java",
			wantErr:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get current directory: %v", err)
			}
			defer os.Chdir(origDir)

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to change to temp directory: %v", err)
			}

			// Create runner
			var args []string
			if test.language != "" {
				args = []string{test.language}
			}
			runner, err := newInitRunner(args, test.language)
			if (err != nil) != test.wantErr {
				t.Fatalf("newInitRunner() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}

			// Run init
			if err := runner.run(context.Background()); err != nil {
				t.Fatalf("run() error = %v", err)
			}

			// Verify .librarian.yaml was created
			configPath := filepath.Join(tmpDir, ".librarian.yaml")
			if _, err := os.Stat(configPath); err != nil {
				t.Fatalf(".librarian.yaml was not created: %v", err)
			}

			// Read and verify config
			librarianConfig, err := config.ReadLibrarianConfig(tmpDir)
			if err != nil {
				t.Fatalf("failed to read .librarian.yaml: %v", err)
			}

			// Verify librarian section
			if librarianConfig.Librarian.Version == "" {
				t.Error("librarian.version should be set")
			}
			if test.language != "" && librarianConfig.Librarian.Language != test.language {
				t.Errorf("librarian.language = %q, want %q", librarianConfig.Librarian.Language, test.language)
			}

			// Verify generate section
			if test.language != "" {
				if librarianConfig.Generate == nil {
					t.Fatal("generate section should exist when language is provided")
				}
				if librarianConfig.Generate.Container.Image == "" {
					t.Error("generate.container.image should be set")
				}
				if librarianConfig.Generate.Container.Tag == "" {
					t.Error("generate.container.tag should be set")
				}
				if librarianConfig.Generate.Googleapis.Repo == "" {
					t.Error("generate.googleapis.repo should be set")
				}
				if librarianConfig.Generate.Dir == "" {
					t.Error("generate.dir should be set")
				}

				// Verify discovery for python and go
				if test.language == "python" || test.language == "go" {
					if librarianConfig.Generate.Discovery == nil {
						t.Error("generate.discovery should exist for python and go")
					} else if librarianConfig.Generate.Discovery.Repo == "" {
						t.Error("generate.discovery.repo should be set")
					}
				} else {
					if librarianConfig.Generate.Discovery != nil {
						t.Errorf("generate.discovery should not exist for %s", test.language)
					}
				}
			} else {
				if librarianConfig.Generate != nil {
					t.Error("generate section should not exist when no language is provided")
				}
			}

			// Verify release section
			if librarianConfig.Release == nil {
				t.Fatal("release section should always exist")
			}
			if librarianConfig.Release.TagFormat == "" {
				t.Error("release.tag_format should be set")
			}
		})
	}
}

func TestInitRunner_AlreadyExists(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create .librarian.yaml
	configPath := filepath.Join(tmpDir, ".librarian.yaml")
	if err := os.WriteFile(configPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create existing .librarian.yaml: %v", err)
	}

	// Try to init
	runner, err := newInitRunner([]string{}, "")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}

	err = runner.run(context.Background())
	if err == nil {
		t.Error("run() should return error when .librarian.yaml already exists")
	}
}
