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
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestConfigGetRunner(t *testing.T) {
	for _, test := range []struct {
		name     string
		key      string
		language string
		want     string
		wantErr  bool
	}{
		{
			name:     "librarian.version",
			key:      "librarian.version",
			language: "",
			want:     "not available",
			wantErr:  false,
		},
		{
			name:     "librarian.language",
			key:      "librarian.language",
			language: "python",
			want:     "python",
			wantErr:  false,
		},
		{
			name:     "generate.dir",
			key:      "generate.dir",
			language: "python",
			want:     "packages/",
			wantErr:  false,
		},
		{
			name:     "generate.container.image",
			key:      "generate.container.image",
			language: "python",
			want:     "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/python-librarian-generator",
			wantErr:  false,
		},
		{
			name:     "generate.container (combined)",
			key:      "generate.container",
			language: "python",
			want:     "us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/python-librarian-generator:latest",
			wantErr:  false,
		},
		{
			name:     "release.tag_format",
			key:      "release.tag_format",
			language: "",
			want:     "{name}-v{version}",
			wantErr:  false,
		},
		{
			name:     "invalid key",
			key:      "invalid.key",
			language: "",
			wantErr:  true,
		},
		{
			name:     "generate section missing",
			key:      "generate.dir",
			language: "",
			wantErr:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			// Init repository
			initRunner, err := newInitRunner([]string{test.language}, test.language)
			if err != nil {
				t.Fatalf("newInitRunner() error = %v", err)
			}
			if err := initRunner.run(context.Background()); err != nil {
				t.Fatalf("init run() error = %v", err)
			}

			// Get config value
			librarianConfig, err := config.ReadLibrarianConfig(tmpDir)
			if err != nil {
				t.Fatalf("failed to read config: %v", err)
			}

			got, err := getConfigValue(librarianConfig, test.key)
			if (err != nil) != test.wantErr {
				t.Fatalf("getConfigValue() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}

			if got != test.want {
				t.Errorf("getConfigValue() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestConfigSetRunner(t *testing.T) {
	for _, test := range []struct {
		name     string
		key      string
		value    string
		language string
		verify   func(*testing.T, *config.LibrarianConfig)
		wantErr  bool
	}{
		{
			name:     "generate.dir",
			key:      "generate.dir",
			value:    "libs/",
			language: "python",
			verify: func(t *testing.T, cfg *config.LibrarianConfig) {
				if cfg.Generate.Dir != "libs/" {
					t.Errorf("generate.dir = %q, want %q", cfg.Generate.Dir, "libs/")
				}
			},
			wantErr: false,
		},
		{
			name:     "generate.container (combined)",
			key:      "generate.container",
			value:    "my-image:v1.0.0",
			language: "python",
			verify: func(t *testing.T, cfg *config.LibrarianConfig) {
				if cfg.Generate.Container.Image != "my-image" {
					t.Errorf("container.image = %q, want %q", cfg.Generate.Container.Image, "my-image")
				}
				if cfg.Generate.Container.Tag != "v1.0.0" {
					t.Errorf("container.tag = %q, want %q", cfg.Generate.Container.Tag, "v1.0.0")
				}
			},
			wantErr: false,
		},
		{
			name:     "generate.container.image",
			key:      "generate.container.image",
			value:    "new-image",
			language: "python",
			verify: func(t *testing.T, cfg *config.LibrarianConfig) {
				if cfg.Generate.Container.Image != "new-image" {
					t.Errorf("container.image = %q, want %q", cfg.Generate.Container.Image, "new-image")
				}
			},
			wantErr: false,
		},
		{
			name:     "generate.container.tag",
			key:      "generate.container.tag",
			value:    "v2.0.0",
			language: "python",
			verify: func(t *testing.T, cfg *config.LibrarianConfig) {
				if cfg.Generate.Container.Tag != "v2.0.0" {
					t.Errorf("container.tag = %q, want %q", cfg.Generate.Container.Tag, "v2.0.0")
				}
			},
			wantErr: false,
		},
		{
			name:     "generate.googleapis.ref",
			key:      "generate.googleapis.ref",
			value:    "abc123",
			language: "python",
			verify: func(t *testing.T, cfg *config.LibrarianConfig) {
				if cfg.Generate.Googleapis.Ref != "abc123" {
					t.Errorf("googleapis.ref = %q, want %q", cfg.Generate.Googleapis.Ref, "abc123")
				}
			},
			wantErr: false,
		},
		{
			name:     "release.tag_format",
			key:      "release.tag_format",
			value:    "v{version}",
			language: "",
			verify: func(t *testing.T, cfg *config.LibrarianConfig) {
				if cfg.Release.TagFormat != "v{version}" {
					t.Errorf("tag_format = %q, want %q", cfg.Release.TagFormat, "v{version}")
				}
			},
			wantErr: false,
		},
		{
			name:     "generate section missing",
			key:      "generate.dir",
			value:    "libs/",
			language: "",
			wantErr:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			// Init repository
			initRunner, err := newInitRunner([]string{test.language}, test.language)
			if err != nil {
				t.Fatalf("newInitRunner() error = %v", err)
			}
			if err := initRunner.run(context.Background()); err != nil {
				t.Fatalf("init run() error = %v", err)
			}

			// Set config value
			setRunner, err := newConfigSetRunner([]string{test.key, test.value})
			if err != nil {
				t.Fatalf("newConfigSetRunner() error = %v", err)
			}

			err = setRunner.run(context.Background())
			if (err != nil) != test.wantErr {
				t.Fatalf("setConfigValue() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}

			// Verify
			librarianConfig, err := config.ReadLibrarianConfig(tmpDir)
			if err != nil {
				t.Fatalf("failed to read config: %v", err)
			}

			if test.verify != nil {
				test.verify(t, librarianConfig)
			}
		})
	}
}

func TestConfigGetRunner_MissingArgs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	_, err := newConfigGetRunner([]string{})
	if err == nil {
		t.Error("newConfigGetRunner() should return error when key is missing")
	}
}

func TestConfigSetRunner_MissingArgs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	_, err := newConfigSetRunner([]string{"key"})
	if err == nil {
		t.Error("newConfigSetRunner() should return error when value is missing")
	}
}

func TestConfigUpdateRunner(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Init repository
	initRunner, err := newInitRunner([]string{"python"}, "python")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	// Test update with --all (should not error, even though not fully implemented)
	updateRunner, err := newConfigUpdateRunner([]string{}, true)
	if err != nil {
		t.Fatalf("newConfigUpdateRunner() error = %v", err)
	}
	if err := updateRunner.run(context.Background()); err != nil {
		t.Fatalf("update run() error = %v", err)
	}
}

func TestConfigUpdateRunner_InvalidKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Init repository
	initRunner, err := newInitRunner([]string{"python"}, "python")
	if err != nil {
		t.Fatalf("newInitRunner() error = %v", err)
	}
	if err := initRunner.run(context.Background()); err != nil {
		t.Fatalf("init run() error = %v", err)
	}

	// Test update with invalid key
	updateRunner, err := newConfigUpdateRunner([]string{"invalid.key"}, false)
	if err != nil {
		t.Fatalf("newConfigUpdateRunner() error = %v", err)
	}
	err = updateRunner.run(context.Background())
	if err == nil {
		t.Error("update run() should return error for invalid key")
	}
}

func TestConfigUpdateRunner_BothKeyAndAll(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	_, err := newConfigUpdateRunner([]string{"generate.container"}, true)
	if err == nil {
		t.Error("newConfigUpdateRunner() should return error when both key and --all are specified")
	}
}
