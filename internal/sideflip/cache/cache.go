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

// Package cache provides functions for managing the librarian cache.
//
// The librarian cache is the directory where the librarian command stores
// downloaded files.
//
// The default location of the librarian cache is $HOME/.librarian. To use a
// different location, set the LIBRARIAN_CACHE environment variable.
//
// The diagrams below explains the structure of the librarian cache.
//
// For each path, $repo is a repository path (i.e. github.com/googleapis/googleapis),
// and $commit is a commit hash in that repository.
//
// Cache structure:
//
//	$LIBRARIAN_CACHE/
//	├── download/                  # Downloaded artifacts
//	│   └── $repo@$commit.tar.gz   # Source tarball
//	│   └── $repo@$commit.info     # Metadata (SHA256)
//	└── $repo@$commit/             # Extracted source files
//	    └── {files...}
//
// Example for github.com/googleapis/googleapis at commit abc123:
//
//	$HOME/.librarian/
//	├── download/
//	│   └── github.com/googleapis/googleapis@abc123.tar.gz
//	│   └── github.com/googleapis/googleapis@abc123.info
//	└── github.com/googleapis/googleapis@abc123/
//	    └── google/
//	        └── api/
//	            └── annotations.proto
//
// When downloading a repository, the cache is checked in this order:
//
//  1. Check if extracted directory exists and contains files.
//  2. Check if tarball exists with valid SHA256 in .info file.
//  3. Download tarball, compute SHA256, save .info, then extract.
package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Dir returns the root cache directory for librarian operations.
// It checks the LIBRARIAN_CACHE environment variable, falling back to
// $HOME/.librarian/ if not set.
func Dir() (string, error) {
	if cache := os.Getenv("LIBRARIAN_CACHE"); cache != "" {
		return cache, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".librarian"), nil
}

// Path returns the path to a cached download artifact for the given repo and commit.
// The suffix specifies the file type (e.g., "tar.gz", "info").
//
// Returns: $LIBRARIAN_CACHE/download/$repo@$commit.$suffix
func Path(repo, commit, suffix string) (string, error) {
	cache, err := Dir()
	if err != nil {
		return "", err
	}
	repoPath := filepath.Join(strings.Split(repo, "/")...)
	downloadDir := filepath.Join(cache, "download", filepath.Dir(repoPath))
	return filepath.Join(downloadDir, fmt.Sprintf("%s@%s.%s", filepath.Base(repoPath), commit, suffix)), nil
}

// DownloadDir returns the directory containing the extracted files for the given repo and commit.
// It validates that the directory exists and contains files.
//
// Returns: $LIBRARIAN_CACHE/$repo@$commit/
func DownloadDir(repo, commit string) (string, error) {
	cache, err := Dir()
	if err != nil {
		return "", err
	}
	repoPath := filepath.Join(strings.Split(repo, "/")...)
	dir := filepath.Join(cache, fmt.Sprintf("%s@%s", repoPath, commit))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("directory %q does not exist or is empty: %w", dir, err)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("directory %q does not exist or is empty", dir)
	}
	return dir, nil
}
