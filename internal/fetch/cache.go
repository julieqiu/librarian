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

// Package fetch provides a file-based cache for repositories used by the
// librarian command.
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
//  1. Check if extracted directory exists and contains files
//  2. Check if tarball exists with valid SHA256 in .info file
//  3. Download tarball, compute SHA256, save .info, then extract
package fetch

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const envLibrarianCache = "LIBRARIAN_CACHE"

// Info contains metadata about a downloaded tarball.
type Info struct {
	SHA256 string `json:"sha256"`
}

// RepoDir downloads a repository tarball and returns the path to the extracted
// directory.
//
// It uses the following caching strategy:
//  1. Check if extracted directory exists and contains files. If so, return that path.
//  2. Check if tarball exists with valid SHA256 in .info file. If so, extract and return the path.
//  3. Download tarball, compute SHA256, save .info, extract, and return the path.
//
// The cache directory is determined by LIBRARIAN_CACHE environment variable,
// or defaults to $HOME/.librarian if not set.
func RepoDir(repo, commit string) (string, error) {
	cacheDir, err := cacheDir()
	if err != nil {
		return "", err
	}

	// Step 1: Check if extracted directory exists and contains files
	extractedDir, err := extractDir(cacheDir, repo, commit)
	if err == nil {
		return extractedDir, nil
	}

	// Step 2: Check if tarball exists with valid SHA256, extract if valid
	tarballPath := artifactPath(cacheDir, repo, commit, "tar.gz")
	infoPath := artifactPath(cacheDir, repo, commit, "info")

	if _, err := os.Stat(tarballPath); err == nil {
		if extractedDir, err := extractTarball(repo, commit, tarballPath, infoPath); err == nil {
			return extractedDir, nil
		}
	}

	// Step 3: Download tarball, compute SHA256, save .info, extract
	sourceURL := repo
	if !strings.Contains(repo, "://") {
		sourceURL = "https://" + repo
	}
	sourceURL += fmt.Sprintf("/archive/%s.tar.gz", commit)

	repoPath := stripScheme(repo)
	extractedDir = filepath.Join(cacheDir, fmt.Sprintf("%s@%s", repoPath, commit))

	if err := os.MkdirAll(filepath.Dir(tarballPath), 0755); err != nil {
		return "", fmt.Errorf("failed creating %q: %w", filepath.Dir(tarballPath), err)
	}
	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		return "", fmt.Errorf("failed creating %q: %w", extractedDir, err)
	}
	if err := downloadTarballWithInfo(sourceURL, tarballPath, infoPath); err != nil {
		return "", err
	}
	if err := extract(tarballPath, extractedDir); err != nil {
		return "", fmt.Errorf("failed to extract tarball: %w", err)
	}
	return extractedDir, nil
}

// extractTarball validates and extracts a cached tarball if it exists with a valid SHA256.
// It returns the path to the extracted directory, or an error if validation fails or extraction errors.
func extractTarball(repo, commit, tarballPath, infoPath string) (string, error) {
	b, err := os.ReadFile(infoPath)
	if err != nil {
		return "", err
	}

	var i Info
	if err := json.Unmarshal(b, &i); err != nil {
		return "", err
	}

	sum, err := computeSHA256(tarballPath)
	if err != nil {
		return "", err
	}

	if sum != i.SHA256 {
		return "", fmt.Errorf("SHA256 mismatch: expected %s, got %s", i.SHA256, sum)
	}

	cacheDir, err := cacheDir()
	if err != nil {
		return "", err
	}

	repoPath := stripScheme(repo)
	extractedDir := filepath.Join(cacheDir, fmt.Sprintf("%s@%s", repoPath, commit))

	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		return "", fmt.Errorf("failed creating %q: %w", extractedDir, err)
	}

	if err := extract(tarballPath, extractedDir); err != nil {
		return "", fmt.Errorf("failed to extract tarball: %w", err)
	}

	return extractedDir, nil
}

// extract extracts a tarball to the specified directory, removing the GitHub
// top-level "repo-<commit>/" prefix from file paths.
func extract(tarballPath, destDir string) error {
	f, err := os.Open(tarballPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// Remove the GitHub top-level "repo-<commit>/" prefix
		name := hdr.Name
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			name = parts[1]
		} else {
			continue
		}

		target := filepath.Join(destDir, name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			out, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}

			out.Close()
		}
	}
}

func computeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Latest returns the latest commit SHA on the default branch for the given repo.
func Latest(repo string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/commits/HEAD", repo)
	return latestCommit(apiURL)
}

// LatestGoogleapis returns the latest commit SHA on master for googleapis/googleapis.
func LatestGoogleapis() (string, error) {
	return Latest("googleapis/googleapis")
}

func latestCommit(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get latest SHA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP error fetching latest SHA: %s", resp.Status)
	}

	var body struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if body.SHA == "" {
		return "", fmt.Errorf("no SHA found in GitHub response")
	}
	return body.SHA, nil
}

func downloadTarballWithInfo(sourceURL, tarballPath, infoPath string) error {
	resp, err := http.Get(sourceURL)
	if err != nil {
		return fmt.Errorf("failed downloading tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tarball download failed: HTTP %d %s (%q)", resp.StatusCode, resp.Status, sourceURL)
	}

	h := sha256.New()
	out, err := os.Create(tarballPath)
	if err != nil {
		return fmt.Errorf("failed creating tarball file: %w", err)
	}

	tee := io.TeeReader(resp.Body, h)
	if _, err := io.Copy(out, tee); err != nil {
		out.Close()
		return fmt.Errorf("failed writing tarball: %w", err)
	}
	out.Close()

	sha := fmt.Sprintf("%x", h.Sum(nil))
	i := Info{SHA256: sha}
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(infoPath), 0755); err != nil {
		return fmt.Errorf("failed to create info dir: %w", err)
	}
	if err := os.WriteFile(infoPath, b, 0644); err != nil {
		return fmt.Errorf("failed to write .info file: %w", err)
	}
	return nil
}

// cacheDir returns the root cache directory for librarian operations.
// It checks the $LIBRARIAN_CACHE environment variable, falling back to $HOME/.librarian/
// if not set.
func cacheDir() (string, error) {
	if cache := os.Getenv(envLibrarianCache); cache != "" {
		return cache, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".librarian"), nil
}

// artifactPath returns the path to a cached download artifact for the given repo and commit.
// The suffix specifies the file type (e.g., "tar.gz", "info").
//
// Returns: $LIBRARIAN_CACHE/download/$repo@$commit.$suffix
func artifactPath(cacheDir, repo, commit, suffix string) string {
	repoPath := stripScheme(repo)
	downloadDir := filepath.Join(cacheDir, "download", filepath.Dir(repoPath))
	return filepath.Join(downloadDir, fmt.Sprintf("%s@%s.%s", filepath.Base(repoPath), commit, suffix))
}

// stripScheme removes the URL scheme (e.g., "https://") from a repo path
// and returns a filepath-safe version.
func stripScheme(repo string) string {
	repoPath := repo
	if idx := strings.Index(repo, "://"); idx != -1 {
		repoPath = repo[idx+3:]
	}
	return filepath.Join(strings.Split(repoPath, "/")...)
}

// extractDir returns the directory containing the extracted files for the
// given repo and commit. It validates that the directory exists and contains
// files.
//
// Returns: $LIBRARIAN_CACHE/$repo@$commit/
func extractDir(cacheDir, repo, commit string) (string, error) {
	repoPath := stripScheme(repo)
	extractedDir := filepath.Join(cacheDir, fmt.Sprintf("%s@%s", repoPath, commit))

	entries, err := os.ReadDir(extractedDir)
	if err != nil {
		return "", fmt.Errorf("directory %q does not exist or is empty: %w", extractedDir, err)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("directory %q does not exist or is empty", extractedDir)
	}
	return extractedDir, nil
}
