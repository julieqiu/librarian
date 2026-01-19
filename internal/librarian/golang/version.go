// Copyright 2026 Google LLC
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

package golang

import (
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/googleapis/librarian/internal/config"
)

var (
	//go:embed template/_internal_version.go.txt
	internalVersionTmpl string

	//go:embed template/_version.go.txt
	versionTmpl string
)

// clientVersionData contains the template data for generating a version.go file.
type clientVersionData struct {
	Year               int
	Package            string
	ModuleRootInternal string
}

// internalVersionData contains the template data for generating an internal/version.go file.
type internalVersionData struct {
	Year    int
	Version string
}

// generateClientVersionFile creates a version.go file for a client.
func generateClientVersionFile(library *config.Library, channelPath string) error {
	dir, err := clientDirectory(library, channelPath)
	if err != nil {
		return err
	}

	modPath := buildModulePath(library)
	fullClientDir := filepath.Join(library.Output, library.Name, dir)
	if err := os.MkdirAll(fullClientDir, 0755); err != nil {
		return err
	}
	versionPath := filepath.Join(fullClientDir, "version.go")
	t := template.Must(template.New("version").Parse(versionTmpl))
	data := clientVersionData{
		Year:               time.Now().Year(),
		Package:            filepath.Base(filepath.Dir(fullClientDir)), // The package name is the name of the directory containing the client directory (e.g. `apiv1beta1`).
		ModuleRootInternal: modPath + "/internal",
	}
	f, err := os.Create(versionPath)
	if err != nil {
		return err
	}
	err = t.Execute(f, data)
	cerr := f.Close()
	if err != nil {
		return err
	}
	return cerr
}

// generateInternalVersionFile creates an internal/version.go file for the module.
func generateInternalVersionFile(moduleDir, version string) error {
	internalDir := filepath.Join(moduleDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		return err
	}
	versionPath := filepath.Join(internalDir, "version.go")
	t := template.Must(template.New("internal_version").Parse(internalVersionTmpl))
	data := internalVersionData{
		Year:    time.Now().Year(),
		Version: version,
	}
	if err := os.MkdirAll(filepath.Dir(versionPath), 0755); err != nil {
		return fmt.Errorf("creating directory for version file: %w", err)
	}

	f, err := os.Create(versionPath)
	if err != nil {
		return err
	}
	err = t.Execute(f, data)
	cerr := f.Close()
	if err != nil {
		return err
	}
	return cerr
}
