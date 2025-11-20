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

package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDir(t *testing.T) {
	for _, test := range []struct {
		name    string
		env     string
		wantDir string
	}{
		{
			name:    "uses LIBRARIAN_CACHE when set",
			env:     "/custom/cache",
			wantDir: "/custom/cache",
		},
		{
			name:    "uses HOME/.librarian when LIBRARIAN_CACHE not set",
			env:     "",
			wantDir: filepath.Join(os.Getenv("HOME"), ".librarian"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.env != "" {
				t.Setenv("LIBRARIAN_CACHE", test.env)
			}

			got, err := Dir()
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.wantDir, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPath(t *testing.T) {
	t.Setenv("LIBRARIAN_CACHE", "/tmp/cache")

	for _, test := range []struct {
		name   string
		repo   string
		commit string
		suffix string
		want   string
	}{
		{
			name:   "simple repo",
			repo:   "github.com/googleapis/googleapis",
			commit: "abc123",
			suffix: "tar.gz",
			want:   "/tmp/cache/download/github.com/googleapis/googleapis@abc123.tar.gz",
		},
		{
			name:   "info file",
			repo:   "github.com/googleapis/googleapis",
			commit: "abc123",
			suffix: "info",
			want:   "/tmp/cache/download/github.com/googleapis/googleapis@abc123.info",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Path(test.repo, test.commit, test.suffix)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDownloadDir(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("LIBRARIAN_CACHE", cacheDir)

	repo := "github.com/googleapis/googleapis"
	commit := "abc123"

	dir := filepath.Join(cacheDir, "github.com/googleapis/googleapis@abc123")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := DownloadDir(repo, commit)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(dir, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDownloadDir_Empty(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("LIBRARIAN_CACHE", cacheDir)

	_, err := DownloadDir("github.com/googleapis/googleapis", "abc123")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}
