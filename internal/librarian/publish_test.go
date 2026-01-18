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
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestPublish(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	testhelper.SetupForVersionBump(t, "v1.0.0")

	cfg := &config.Config{
		Language: languageFake,
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
		},
	}
	if err := publish(t.Context(), cfg, false, false, false); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile("PUBLISHED")
	if err != nil {
		t.Fatal(err)
	}
	want := "published\n"
	if diff := cmp.Diff(want, string(content)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPublishCommand(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
	}{
		{
			name: "no flags",
			args: []string{"librarian", "publish"},
		},
		{
			name: "dry run flag",
			args: []string{"librarian", "publish", "--dry-run"},
		},
		{
			name: "dry run keep going flag",
			args: []string{"librarian", "publish", "--dry-run-keep-going"},
		},
		{
			name: "skip semver checks flag",
			args: []string{"librarian", "publish", "--skip-semver-checks"},
		},
		{
			name: "all flags",
			args: []string{"librarian", "publish", "--dry-run", "--dry-run-keep-going", "--skip-semver-checks"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			testhelper.SetupForVersionBump(t, "v1.0.0")

			configContent := `language: fake
release:
  remote: origin
  branch: main
`
			if err := os.WriteFile("librarian.yaml", []byte(configContent), 0644); err != nil {
				t.Fatal(err)
			}

			err := Run(t.Context(), test.args...)
			if err != nil {
				t.Fatal(err)
			}

			content, err := os.ReadFile("PUBLISHED")
			if err != nil {
				t.Fatal(err)
			}
			want := "published\n"
			if diff := cmp.Diff(want, string(content)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPublishUnsupportedLanguage(t *testing.T) {
	cfg := &config.Config{
		Language: "unsupported",
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
		},
	}

	err := publish(t.Context(), cfg, false, false, false)
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
	want := `publish not implemented for "unsupported"`
	if diff := cmp.Diff(want, err.Error()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
