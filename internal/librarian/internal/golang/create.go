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
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

var (
	//go:embed tmpl/_version.go.txt
	versionTmpl string

	//go:embed tmpl/_README.md.txt
	readmeTmpl string
)

// Create creates a new Go client library from scratch.
func Create(ctx context.Context, libraryName string, apis []*config.API, googleapisDir string) error {
	if len(apis) == 0 {
		return fmt.Errorf("at least one API is required")
	}

	// Derive module path from library name.
	// Convention: cloud.google.com/go/{libraryName}
	modulePath := "cloud.google.com/go/" + libraryName

	if err := os.MkdirAll(libraryName, 0755); err != nil {
		return fmt.Errorf("failed to create library directory: %w", err)
	}

	title := libraryName
	if apis[0].ServiceConfig != "" {
		serviceYAMLPath := filepath.Join(googleapisDir, apis[0].ServiceConfig)
		cfg, err := serviceconfig.Read(serviceYAMLPath)
		if err != nil {
			return err
		}
		title = cfg.GetTitle()
	}

	if err := generateReadme(libraryName, title, modulePath); err != nil {
		return err
	}
	if err := generateChangelog(libraryName); err != nil {
		return err
	}
	if err := generateInternalVersionFile(libraryName); err != nil {
		return err
	}

	library := &config.Library{
		Name:    libraryName,
		Output:  libraryName,
		APIs:    apis,
		Version: "0.0.0",
		Go: &config.GoModule{
			ModulePath: modulePath,
		},
	}
	if err := Generate(ctx, library, googleapisDir); err != nil {
		return fmt.Errorf("failed to generate Go code: %w", err)
	}
	return postGenerate(libraryName, modulePath)
}

// generateReadme creates a README.md file for the library.
func generateReadme(dir, title, modulePath string) error {
	readmePath := filepath.Join(dir, "README.md")
	readmeFile, err := os.Create(readmePath)
	if err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}
	defer readmeFile.Close()

	t := template.Must(template.New("readme").Parse(readmeTmpl))
	data := struct {
		Name       string
		ModulePath string
	}{
		Name:       title,
		ModulePath: modulePath,
	}
	return t.Execute(readmeFile, data)
}

// generateChangelog creates a CHANGES.md file for the library.
func generateChangelog(dir string) error {
	changesPath := filepath.Join(dir, "CHANGES.md")
	content := "# Changes\n"
	return os.WriteFile(changesPath, []byte(content), 0644)
}

// generateInternalVersionFile generates internal/version.go for the library.
func generateInternalVersionFile(libraryName string) error {
	internalDir := filepath.Join(libraryName, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		return fmt.Errorf("failed to create internal directory: %w", err)
	}

	versionPath := filepath.Join(internalDir, "version.go")
	versionFile, err := os.Create(versionPath)
	if err != nil {
		return fmt.Errorf("failed to create version.go: %w", err)
	}
	defer versionFile.Close()

	t := template.Must(template.New("version").Parse(versionTmpl))
	data := struct {
		Year    int
		Version string
	}{
		Year:    time.Now().Year(),
		Version: "0.0.0",
	}
	return t.Execute(versionFile, data)
}

// postGenerate runs post-generation tasks on the library.
func postGenerate(libraryName, modulePath string) error {
	if err := command.RunIn(libraryName, "go", "mod", "init", modulePath); err != nil {
		return fmt.Errorf("go mod init failed: %w", err)
	}
	if err := command.RunIn(libraryName, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}
	if err := command.Run("go", "tool", "goimports", "-w", libraryName); err != nil {
		return fmt.Errorf("goimports failed: %w", err)
	}
	return nil
}
