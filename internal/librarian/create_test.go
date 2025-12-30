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

package librarian

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestCreateLibrary(t *testing.T) {
	for _, test := range []struct {
		name            string
		libName         string
		existingLibrary *config.Library
		wantOutput      string
		wantErr         error
	}{
		{
			name:       "create new library",
			libName:    "newlib",
			wantOutput: "output", // Defaults to cfg.Default.Output
		},
		{
			name:    "regenerate existing library",
			libName: "existinglib",
			existingLibrary: &config.Library{
				Name:   "existinglib",
				Output: "existing-output",
			},
			wantErr: errors.New(`"existinglib" already exists`),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			// Setup a fake googleapis repo structure
			// librarian expects: <googleapis>/<channel>/<service>.yaml
			// For languageFake, channel derived from name is just the name.
			serviceConfigDir := filepath.Join(tmpDir, test.libName)
			if err := os.MkdirAll(serviceConfigDir, 0755); err != nil {
				t.Fatal(err)
			}
			serviceConfigPath := filepath.Join(serviceConfigDir, test.libName+".yaml")
			serviceConfigContent := "type: google.api.Service\nconfig_version: 3\n"
			if err := os.WriteFile(serviceConfigPath, []byte(serviceConfigContent), 0644); err != nil {
				t.Fatal(err)
			}

			cfg := &config.Config{
				Language: languageFake,
				Default: &config.Default{
					Output: "output",
				},
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Dir: tmpDir, // Pointing to temp root as googleapis root
					},
				},
			}
			if test.existingLibrary != nil {
				cfg.Libraries = []*config.Library{test.existingLibrary}
			}
			if err := yaml.Write(librarianConfigPath, cfg); err != nil {
				t.Fatal(err)
			}

			err := runCreate(t.Context(), test.libName)
			if test.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, got nil", test.wantErr)
				}
				if err.Error() != test.wantErr.Error() {
					t.Errorf("want error %v, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			// Verify Config Update
			cfg, err = yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				t.Fatal(err)
			}

			var found *config.Library
			for _, lib := range cfg.Libraries {
				if lib.Name == test.libName {
					found = lib
					break
				}
			}
			if found == nil {
				t.Fatal("library not found in config")
			}

			if found.Output != test.wantOutput {
				t.Errorf("output = %q, want %q", found.Output, test.wantOutput)
			}

			// Verify File Generation
			readmePath := filepath.Join(test.wantOutput, "README.md")
			if _, err := os.Stat(readmePath); err != nil {
				t.Errorf("expected README.md at %s: %v", readmePath, err)
			}

			versionPath := filepath.Join(test.wantOutput, "VERSION")
			content, err := os.ReadFile(versionPath)
			if err != nil {
				t.Fatal(err)
			}
			want := "0.0.0"
			if diff := cmp.Diff(want, string(content)); diff != "" {
				t.Errorf("VERSION mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateLibraryNoYaml(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	err := runCreate(t.Context(), "newlib", "", "", "output/newlib", "protobuf")
	if !errors.Is(err, errNoYaml) {
		t.Errorf("want error %v, got %v", errNoYaml, err)
	}
}

func TestCreateCommand(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		wantErr error
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "create"},
			wantErr: errMissingLibraryName,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Run(t.Context(), test.args...)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("want error %v, got %v", test.wantErr, err)
				}
				return
			}
		})
	}
}
