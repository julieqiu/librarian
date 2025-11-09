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

package python

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GenerateScaffolding creates the initial scaffolding files for a new Python library.
// This should be called by librarian generate on first-time generation only.
func GenerateScaffolding(repoRoot, libraryID string) error {
	libraryPath := filepath.Join(repoRoot, "packages", libraryID)

	// Check if this is first-time generation
	if pathExists(libraryPath) {
		// Directory exists - not first time, skip scaffolding
		return nil
	}

	// Create packages/{library_id}/CHANGELOG.md
	if err := generateChangelog(libraryPath, libraryID); err != nil {
		return err
	}

	// Create packages/{library_id}/docs/CHANGELOG.md (duplicate)
	if err := generateDocsChangelog(libraryPath, libraryID); err != nil {
		return err
	}

	// Update global CHANGELOG.md
	if err := updateGlobalChangelog(repoRoot, libraryID); err != nil {
		return err
	}

	return nil
}

func generateChangelog(libraryPath, libraryID string) error {
	if err := os.MkdirAll(libraryPath, 0755); err != nil {
		return err
	}

	changelogPath := filepath.Join(libraryPath, "CHANGELOG.md")
	content := fmt.Sprintf(`# Changelog

[PyPI History][1]

[1]: https://pypi.org/project/%s/#history
`, libraryID)

	return os.WriteFile(changelogPath, []byte(content), 0644)
}

func generateDocsChangelog(libraryPath, libraryID string) error {
	docsPath := filepath.Join(libraryPath, "docs")
	if err := os.MkdirAll(docsPath, 0755); err != nil {
		return err
	}

	changelogPath := filepath.Join(docsPath, "CHANGELOG.md")
	content := fmt.Sprintf(`# Changelog

[PyPI History][1]

[1]: https://pypi.org/project/%s/#history
`, libraryID)

	return os.WriteFile(changelogPath, []byte(content), 0644)
}

func updateGlobalChangelog(repoRoot, libraryID string) error {
	changelogPath := filepath.Join(repoRoot, "CHANGELOG.md")

	// Read existing changelog
	data, err := os.ReadFile(changelogPath)
	if err != nil {
		return fmt.Errorf("failed to read global CHANGELOG.md: %w", err)
	}

	content := string(data)

	// Create new entry
	newEntry := fmt.Sprintf("- [%s==0.0.0](https://pypi.org/project/%s)\n", libraryID, libraryID)

	// Find the "Supported Python Versions" or libraries section
	// We'll insert in alphabetical order after finding existing library entries

	// Look for lines that match the pattern: - [library-name==version](url)
	libraryPattern := regexp.MustCompile(`^- \[([a-z0-9-]+)==[0-9.]+\]`)

	lines := strings.Split(content, "\n")
	insertIndex := -1
	inLibrarySection := false

	for i, line := range lines {
		// Check if this is a library entry
		if libraryPattern.MatchString(line) {
			inLibrarySection = true
			matches := libraryPattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				existingLibrary := matches[1]
				// If new library should come before this one alphabetically
				if libraryID < existingLibrary {
					insertIndex = i
					break
				}
			}
		} else if inLibrarySection && !strings.HasPrefix(line, "-") {
			// We've reached the end of the library section
			// Insert at the end of the section (before this line)
			insertIndex = i
			break
		}
	}

	// If we found a place to insert
	if insertIndex >= 0 {
		// Insert the new entry
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:insertIndex]...)
		newLines = append(newLines, newEntry)
		newLines = append(newLines, lines[insertIndex:]...)
		content = strings.Join(newLines, "\n")
	} else {
		// If no library section found, append at the end
		// (This shouldn't normally happen in a well-formed CHANGELOG)
		content = content + "\n" + newEntry
	}

	return os.WriteFile(changelogPath, []byte(content), 0644)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
