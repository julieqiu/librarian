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

package librarian

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCleanOutput(t *testing.T) {
	for _, test := range []struct {
		name  string
		files []string
		keep  []string
		want  []string
	}{
		{
			name:  "removes all files",
			files: []string{"README.md", "src/lib.rs"},
			want:  []string{},
		},
		{
			name:  "empty directory",
			files: []string{},
			want:  []string{},
		},
		{
			name:  "keep single file",
			files: []string{"Cargo.toml", "README.md"},
			keep:  []string{"Cargo.toml"},
			want:  []string{"Cargo.toml"},
		},
		{
			name:  "keep file in subdirectory",
			files: []string{"Cargo.toml", "src/lib.rs", "src/errors.rs", "src/other.rs"},
			keep:  []string{"Cargo.toml", "src/errors.rs"},
			want:  []string{"Cargo.toml", "src"},
		},
		{
			name:  "keep multiple files in subdirectory",
			files: []string{"Cargo.toml", "src/lib.rs", "src/errors.rs", "src/operation.rs"},
			keep:  []string{"Cargo.toml", "src/errors.rs", "src/operation.rs"},
			want:  []string{"Cargo.toml", "src"},
		},
		{
			name:  "keep entire directory",
			files: []string{"Cargo.toml", "src/lib.rs", "custom/file.rs"},
			keep:  []string{"Cargo.toml", "custom"},
			want:  []string{"Cargo.toml", "custom"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range test.files {
				path := filepath.Join(dir, f)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := cleanOutput(dir, test.keep); err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, e := range entries {
				got = append(got, e.Name())
			}
			slices.Sort(got)
			slices.Sort(test.want)
			if !slices.Equal(got, test.want) {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestCleanOutput_NonExistentDir(t *testing.T) {
	if err := cleanOutput("/nonexistent/path", nil); err != nil {
		t.Errorf("expected nil error for nonexistent dir, got %v", err)
	}
}

func TestCleanOutput_PreservesKeptFiles(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"Cargo.toml",
		"src/lib.rs",
		"src/errors.rs",
		"src/operation.rs",
		"src/other.rs",
	}
	for _, f := range files {
		path := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test content for "+f), 0644); err != nil {
			t.Fatal(err)
		}
	}

	keep := []string{"Cargo.toml", "src/errors.rs", "src/operation.rs"}
	if err := cleanOutput(dir, keep); err != nil {
		t.Fatal(err)
	}

	// Verify kept files still exist with their content.
	for _, k := range keep {
		path := filepath.Join(dir, k)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("kept file %q was deleted: %v", k, err)
			continue
		}
		if want := "test content for " + k; string(content) != want {
			t.Errorf("kept file %q has wrong content: got %q, want %q", k, content, want)
		}
	}

	// Verify deleted files are gone.
	for _, f := range []string{"src/lib.rs", "src/other.rs"} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("file %q should have been deleted but still exists", f)
		}
	}
}
