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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Configure creates the initial scaffolding files for a new Go library.
// This should be called by librarian generate on first-time generation only.
func Configure(repoRoot, googleapisRoot, libraryID string, apis []API) error {
	libraryPath := filepath.Join(repoRoot, libraryID)

	// Check if this is first-time generation
	if pathExists(libraryPath) {
		// Directory exists - not first time, skip scaffolding
		return nil
	}

	// Read service title from first API's service config
	if len(apis) == 0 {
		return fmt.Errorf("no APIs configured for library %s", libraryID)
	}

	serviceYAMLPath := filepath.Join(googleapisRoot, apis[0].Path, apis[0].ServiceConfig)
	title, err := readServiceTitle(serviceYAMLPath)
	if err != nil {
		return fmt.Errorf("failed to read service title: %w", err)
	}

	// Create README.md
	if err := generateReadme(libraryPath, libraryID, title); err != nil {
		return err
	}

	// Create CHANGES.md
	if err := generateChanges(libraryPath); err != nil {
		return err
	}

	// Create internal/version.go
	if err := generateInternalVersion(libraryPath, libraryID); err != nil {
		return err
	}

	// Create {clientDir}/version.go for each API
	for _, api := range apis {
		clientDir := deriveClientDir(api.Path)
		if err := generateClientVersion(libraryPath, libraryID, clientDir); err != nil {
			return err
		}
	}

	// Update internal/generated/snippets/go.mod
	if err := updateSnippetsGoMod(repoRoot, libraryID); err != nil {
		return err
	}

	return nil
}

// API represents a single API configuration.
type API struct {
	Path          string
	ServiceConfig string
}

var readmeTemplate = template.Must(template.New("readme").Parse(`# {{.Title}}

[Product Documentation](https://cloud.google.com/{{.Product}})

## Installation

` + "```bash" + `
go get cloud.google.com/go/{{.LibraryID}}
` + "```" + `

## Example Usage

[See pkg.go.dev](https://pkg.go.dev/cloud.google.com/go/{{.LibraryID}}) for examples.
`))

var internalVersionTemplate = template.Must(template.New("version").Parse(`// Copyright {{.Year}} Google LLC
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

package internal

// Version is the current version of the {{.LibraryID}} client library.
const Version = "{{.Version}}"
`))

var clientVersionTemplate = template.Must(template.New("clientversion").Parse(`// Copyright {{.Year}} Google LLC
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

package {{.Package}}

import "cloud.google.com/go/{{.LibraryID}}/internal"

// version is the version of this client library.
var version = internal.Version
`))

func generateReadme(libraryPath, libraryID, title string) error {
	readmePath := filepath.Join(libraryPath, "README.md")

	if err := os.MkdirAll(libraryPath, 0755); err != nil {
		return err
	}

	f, err := os.Create(readmePath)
	if err != nil {
		return err
	}
	defer f.Close()

	data := struct {
		Title     string
		LibraryID string
		Product   string
	}{
		Title:     title,
		LibraryID: libraryID,
		Product:   deriveProductName(libraryID),
	}

	return readmeTemplate.Execute(f, data)
}

func generateChanges(libraryPath string) error {
	changesPath := filepath.Join(libraryPath, "CHANGES.md")
	return os.WriteFile(changesPath, []byte("# Changes\n"), 0644)
}

func generateInternalVersion(libraryPath, libraryID string) error {
	internalPath := filepath.Join(libraryPath, "internal")
	if err := os.MkdirAll(internalPath, 0755); err != nil {
		return err
	}

	versionPath := filepath.Join(internalPath, "version.go")
	f, err := os.Create(versionPath)
	if err != nil {
		return err
	}
	defer f.Close()

	data := struct {
		Year      int
		LibraryID string
		Version   string
	}{
		Year:      time.Now().Year(),
		LibraryID: libraryID,
		Version:   "0.0.0",
	}

	return internalVersionTemplate.Execute(f, data)
}

func generateClientVersion(libraryPath, libraryID, clientDir string) error {
	clientPath := filepath.Join(libraryPath, clientDir)
	if err := os.MkdirAll(clientPath, 0755); err != nil {
		return err
	}

	versionPath := filepath.Join(clientPath, "version.go")
	f, err := os.Create(versionPath)
	if err != nil {
		return err
	}
	defer f.Close()

	data := struct {
		Year      int
		Package   string
		LibraryID string
	}{
		Year:      time.Now().Year(),
		Package:   filepath.Base(clientDir), // e.g., "apiv1"
		LibraryID: libraryID,
	}

	return clientVersionTemplate.Execute(f, data)
}

func updateSnippetsGoMod(repoRoot, libraryID string) error {
	snippetsDir := filepath.Join(repoRoot, "internal/generated/snippets")
	modulePath := fmt.Sprintf("cloud.google.com/go/%s", libraryID)
	relativeDir := fmt.Sprintf("../../../%s", libraryID)

	cmd := exec.Command("go", "mod", "edit", "-replace",
		fmt.Sprintf("%s=%s", modulePath, relativeDir))
	cmd.Dir = snippetsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func readServiceTitle(serviceYAMLPath string) (string, error) {
	// This is a simplified version - you may want to use a YAML parser
	data, err := os.ReadFile(serviceYAMLPath)
	if err != nil {
		return "", err
	}

	// Simple parse for "title: " field
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "title:") {
			title := strings.TrimPrefix(line, "title:")
			return strings.TrimSpace(title), nil
		}
	}

	return "", fmt.Errorf("title not found in service YAML")
}

func deriveClientDir(apiPath string) string {
	parts := strings.Split(apiPath, "/")
	version := parts[len(parts)-1]
	return "api" + version
}

func deriveProductName(libraryID string) string {
	// Convert library ID to product URL fragment
	// e.g., "secretmanager" → "secret-manager"
	// This is a simplified version
	return strings.ReplaceAll(libraryID, "_", "-")
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
