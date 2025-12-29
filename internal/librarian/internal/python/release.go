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

// Package python provides Python specific functionality for librarian.
package python

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/semver"
)

var (
	// versionVarRegex matches `version = "x.y.z"` or `__version__ = "x.y.z"`.
	// Group 1: Prefix (e.g. `version = "`)
	// Group 2: Version string
	// Group 3: Suffix (e.g. `"`)
	versionVarRegex = regexp.MustCompile(`(?m)^(version\s*=\s*["']|__version__\s*=\s*["'])([^"']+)(["'].*)$`)

	// releaseDateRegex matches `__release_date__ = "YYYY-MM-DD"`.
	releaseDateRegex = regexp.MustCompile(`(?m)^(__release_date__\s*=\s*["'])([^"']+)(["'].*)$`)
)

// ReleaseLibrary bumps version for Python files.
func ReleaseLibrary(library *config.Library, srcPath string) error {
	versionFiles, err := findVersionFiles(srcPath)
	if err != nil {
		return err
	}
	if len(versionFiles) == 0 {
		return fmt.Errorf("no version files found in %s", srcPath)
	}

	// Just like Rust, we derive the next version using a hardcoded Minor bump.
	// We extract the current version from the first file found.
	currentVersion, err := extractVersion(versionFiles[0])
	if err != nil {
		return err
	}

	newVersion, err := semver.DeriveNextOptions{
		BumpVersionCore:       true,
		DowngradePreGAChanges: true,
	}.DeriveNext(semver.Minor, currentVersion)
	if err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")
	for _, file := range versionFiles {
		if err := updateVersionFile(file, newVersion, today); err != nil {
			return err
		}
	}

	// Update snippet metadata if it exists.
	if err := updateSnippetMetadata(srcPath, newVersion); err != nil {
		return err
	}

	library.Version = newVersion
	return nil
}

// findVersionFiles locates gapic_version.py, version.py, pyproject.toml, or setup.py.
func findVersionFiles(root string) ([]string, error) {
	var files []string
	var fallbackFiles []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".nox" || name == "build" || name == "dist" || name == "__pycache__" || name == "venv" || name == ".venv" {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		if name == "gapic_version.py" || name == "version.py" {
			if filepath.Base(filepath.Dir(path)) == "types" {
				return nil
			}
			files = append(files, path)
		} else if name == "pyproject.toml" || name == "setup.py" {
			fallbackFiles = append(fallbackFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(files) > 0 {
		return files, nil
	}
	return fallbackFiles, nil
}

func extractVersion(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	matches := versionVarRegex.FindSubmatch(content)
	if len(matches) < 3 {
		return "", fmt.Errorf("version string not found in %s", path)
	}
	return string(matches[2]), nil
}

func updateVersionFile(path, newVersion, date string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	newContent := versionVarRegex.ReplaceAll(content, []byte("${1}"+newVersion+"${3}"))

	if releaseDateRegex.Match(newContent) {
		newContent = releaseDateRegex.ReplaceAll(newContent, []byte("${1}"+date+"${3}"))
	}

	return os.WriteFile(path, newContent, 0644)
}

func updateSnippetMetadata(root, newVersion string) error {
	samplesDir := filepath.Join(root, "samples")
	if _, err := os.Stat(samplesDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.WalkDir(samplesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.Contains(d.Name(), "snippet") && strings.HasSuffix(d.Name(), ".json") {
			return updateJSONVersion(path, newVersion)
		}
		return nil
	})
}

func updateJSONVersion(path, newVersion string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return err
	}

	clientLib, ok := data["clientLibrary"].(map[string]interface{})
	if !ok {
		return nil
	}
	clientLib["version"] = newVersion

	newContent, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	newContent = append(newContent, '\n')

	return os.WriteFile(path, newContent, 0644)
}

// DeriveSrcPath determines what src path library code lives in.
func DeriveSrcPath(libCfg *config.Library, cfg *config.Config) string {
	if libCfg.Output != "" {
		return libCfg.Output
	}

	libSrcDir := ""
	if len(libCfg.Channels) > 0 && libCfg.Channels[0].Path != "" {
		libSrcDir = libCfg.Channels[0].Path
	} else {
		// Default structure: packages/{library_id}
		libSrcDir = filepath.Join("packages", libCfg.Name)
	}

	if cfg != nil && cfg.Default != nil && cfg.Default.Output != "" {
		return strings.ReplaceAll(cfg.Default.Output, "{service}", libSrcDir)
	}

	return libSrcDir
}
