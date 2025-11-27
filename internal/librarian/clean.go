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
)

// CleanOutput removes all files and directories in the output directory except
// those specified in the keep list. The keep list can contain both top-level
// files/directories and files within subdirectories (e.g., "src/errors.rs").
func CleanOutput(dir string, keep []string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// Build maps for top-level keeps and per-directory keeps.
	topLevelKeep := make(map[string]bool)
	subdirKeep := make(map[string]map[string]bool) // subdir -> filename -> true

	for _, k := range keep {
		dir, file := filepath.Split(k)
		if dir == "" {
			topLevelKeep[file] = true
		} else {
			dir = filepath.Clean(dir)
			if subdirKeep[dir] == nil {
				subdirKeep[dir] = make(map[string]bool)
			}
			subdirKeep[dir][file] = true
		}
	}

	for _, entry := range entries {
		name := entry.Name()

		// Keep top-level files/directories explicitly listed.
		if topLevelKeep[name] {
			continue
		}

		// If this is a directory with files to keep, clean it selectively.
		if entry.IsDir() {
			if keepFiles, ok := subdirKeep[name]; ok {
				if err := cleanDir(filepath.Join(dir, name), keepFiles); err != nil {
					return err
				}
				continue
			}
		}

		// Remove everything else.
		if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	return nil
}

// cleanDir removes all files in a directory except those in keepFiles.
func cleanDir(dir string, keepFiles map[string]bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if keepFiles[entry.Name()] {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
