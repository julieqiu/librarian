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
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// cleanOutput removes all files and directories in dir except those specified
// in the keep list. The keep list can contain both files and directories.
// Paths in keep are relative to dir and can include subdirectories
// (e.g., "src/errors.rs").
func cleanOutput(dir string, keep []string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	// Build a set of top-level entries that contain kept files.
	keepDirs := make(map[string][]string)
	for _, k := range keep {
		// Split on path separator to get first component.
		first := k
		rest := ""
		if idx := findPathSep(k); idx != -1 {
			first = k[:idx]
			rest = k[idx+1:]
		}
		if rest != "" {
			keepDirs[first] = append(keepDirs[first], rest)
		} else {
			// Keep the entire entry.
			keepDirs[first] = nil
		}
	}
	for _, e := range entries {
		subKeep, hasKeep := keepDirs[e.Name()]
		if hasKeep && subKeep == nil {
			// Keep entire entry.
			continue
		}
		if hasKeep && e.IsDir() {
			// Recursively clean directory while preserving kept files.
			if err := cleanOutput(filepath.Join(dir, e.Name()), subKeep); err != nil {
				return err
			}
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// findPathSep returns the index of the first path separator in s, or -1.
func findPathSep(s string) int {
	for i, c := range s {
		if c == '/' || c == filepath.Separator {
			return i
		}
	}
	return -1
}
