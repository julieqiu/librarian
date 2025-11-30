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

package golang

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/semver"
)

//go:embed _internal_version.go.txt
var internalVersionTmpl string

// now is a variable for testing.
var now = time.Now

// ReleaseAll updates versions and changelogs for all libraries.
func ReleaseAll(cfg *config.Config) (*config.Config, error) {
	return release(cfg, "")
}

// ReleaseLibrary updates versions and changelog for a specific library.
func ReleaseLibrary(cfg *config.Config, name string) (*config.Config, error) {
	return release(cfg, name)
}

func release(cfg *config.Config, name string) (*config.Config, error) {
	shouldRelease := func(libName string) bool {
		if name == "" {
			return true
		}
		return name == libName
	}

	var found bool
	for _, lib := range cfg.Libraries {
		if !shouldRelease(lib.Name) {
			continue
		}
		if lib.SkipRelease {
			fmt.Printf("skipping %s: skip_release is set\n", lib.Name)
			continue
		}

		found = true
		moduleDir := lib.Output
		if moduleDir == "" {
			moduleDir = lib.Name
		}
		// Handle root-module case.
		if lib.Name == "root-module" {
			moduleDir = "."
		}

		// Read current version from config first, then from file.
		currentVersion := lib.Version
		if currentVersion == "" {
			currentVersion = readVersion(moduleDir)
		}
		if currentVersion == "" {
			currentVersion = "0.0.0"
		}

		// Bump version using semver.DeriveNext with Patch level.
		newVersion, err := semver.DeriveNext(semver.Patch, currentVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to bump version for %s: %w", lib.Name, err)
		}
		lib.Version = newVersion

		// Update changelog.
		if err := updateChangelog(lib, now().UTC()); err != nil {
			return nil, fmt.Errorf("failed to update changelog for %s: %w", lib.Name, err)
		}

		// Update internal/version.go.
		if err := updateVersionFile(moduleDir, newVersion); err != nil {
			return nil, fmt.Errorf("failed to update version file for %s: %w", lib.Name, err)
		}

		// Update snippet metadata.
		if err := updateSnippetsMetadata(lib.Name, newVersion); err != nil {
			return nil, fmt.Errorf("failed to update snippets metadata for %s: %w", lib.Name, err)
		}

		fmt.Printf("✓ Released %s v%s\n", lib.Name, newVersion)
	}

	if name != "" && !found {
		return nil, fmt.Errorf("library %q not found", name)
	}
	return cfg, nil
}

// updateVersionFile creates or updates the internal/version.go file.
func updateVersionFile(moduleDir, version string) error {
	internalDir := filepath.Join(moduleDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		return err
	}

	versionPath := filepath.Join(internalDir, "version.go")
	t := template.Must(template.New("internal_version").Parse(internalVersionTmpl))

	data := struct {
		Year    int
		Version string
	}{
		Year:    now().Year(),
		Version: version,
	}

	f, err := os.Create(versionPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, data)
}

// changelogSections defines the order of sections in the changelog.
var changelogSections = []struct {
	Type    string
	Section string
}{
	{Type: "feat", Section: "Features"},
	{Type: "fix", Section: "Bug Fixes"},
	{Type: "perf", Section: "Performance Improvements"},
	{Type: "revert", Section: "Reverts"},
	{Type: "docs", Section: "Documentation"},
}

// updateChangelog updates the CHANGES.md file with a new version entry.
func updateChangelog(lib *config.Library, t time.Time) error {
	var changelogPath string
	if lib.Name == "root-module" || lib.Output == "." {
		changelogPath = "CHANGES.md"
	} else {
		dir := lib.Output
		if dir == "" {
			dir = lib.Name
		}
		changelogPath = filepath.Join(dir, "CHANGES.md")
	}

	oldContent, err := os.ReadFile(changelogPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading changelog: %w", err)
	}

	// Check if version already exists.
	versionString := fmt.Sprintf("## [%s]", lib.Version)
	if bytes.Contains(oldContent, []byte(versionString)) {
		return nil
	}

	var newEntry bytes.Buffer

	// Build tag URL.
	tagFormat := lib.TagFormat
	if tagFormat == "" {
		tagFormat = "{name}/v{version}"
	}
	tag := strings.NewReplacer("{name}", lib.Name, "{version}", lib.Version).Replace(tagFormat)
	encodedTag := strings.ReplaceAll(tag, "/", "%2F")
	releaseURL := "https://github.com/googleapis/google-cloud-go/releases/tag/" + encodedTag
	date := t.Format("2006-01-02")
	fmt.Fprintf(&newEntry, "## [%s](%s) (%s)\n\n", lib.Version, releaseURL, date)

	// For now, just add a placeholder since we don't have commit info.
	// The real implementation would parse git commits.
	newEntry.WriteString("### Miscellaneous Chores\n\n")
	newEntry.WriteString("* release\n\n")

	// Find insertion point after "# Changes" title.
	insertionPoint := 0
	titleMarker := []byte("# Changes")
	if i := bytes.Index(oldContent, titleMarker); i != -1 {
		searchStart := i + len(titleMarker)
		nonWhitespaceIdx := bytes.IndexFunc(oldContent[searchStart:], func(r rune) bool {
			return !bytes.ContainsRune([]byte{' ', '\t', '\n', '\r'}, r)
		})
		if nonWhitespaceIdx != -1 {
			insertionPoint = searchStart + nonWhitespaceIdx
		} else {
			insertionPoint = len(oldContent)
		}
	} else if len(oldContent) == 0 {
		// No existing changelog, create with header.
		newEntry.Reset()
		fmt.Fprintf(&newEntry, "# Changes\n\n## [%s](%s) (%s)\n\n", lib.Version, releaseURL, date)
		newEntry.WriteString("### Miscellaneous Chores\n\n")
		newEntry.WriteString("* release\n\n")
	}

	// Build new content.
	var newContent []byte
	newContent = append(newContent, oldContent[:insertionPoint]...)
	newContent = append(newContent, newEntry.Bytes()...)
	newContent = append(newContent, oldContent[insertionPoint:]...)

	// Ensure directory exists.
	if err := os.MkdirAll(filepath.Dir(changelogPath), 0755); err != nil {
		return fmt.Errorf("creating directory for changelog: %w", err)
	}
	return os.WriteFile(changelogPath, newContent, 0644)
}

// updateSnippetsMetadata updates the version in snippet metadata files.
func updateSnippetsMetadata(libraryName, version string) error {
	snippetsDir := filepath.Join("internal", "generated", "snippets", libraryName)
	if _, err := os.Stat(snippetsDir); os.IsNotExist(err) {
		// No snippets directory, nothing to update.
		return nil
	}

	// Find all snippet_metadata.*.json files.
	return filepath.Walk(snippetsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasPrefix(info.Name(), "snippet_metadata.") || !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		contentStr := string(content)
		var newContent string

		// Replace $VERSION placeholder or existing version.
		if strings.Contains(contentStr, "$VERSION") {
			newContent = strings.Replace(contentStr, "$VERSION", version, 1)
		} else {
			// Replace existing version number.
			re := regexp.MustCompile(`\d+\.\d+\.\d+`)
			if foundVersion := re.FindString(contentStr); foundVersion != "" {
				newContent = strings.Replace(contentStr, foundVersion, version, 1)
			}
		}

		if newContent != "" && newContent != contentStr {
			if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
				return err
			}
		}
		return nil
	})
}
