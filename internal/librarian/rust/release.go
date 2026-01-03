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

// Package rust provides Rust specific functionality for librarian.
package rust

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/semver"
)

const defaultVersion = "0.1.0"

type cargoPackage struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

type cargoManifest struct {
	Package *cargoPackage `toml:"package"`
}

// ReleaseLibrary bumps version for Cargo.toml files and updates librarian config version.
func ReleaseLibrary(library *config.Library, srcPath string) error {
	cargoFile := filepath.Join(srcPath, "Cargo.toml")
	currentVersion := library.Version
	var newVersion string
	if currentVersion == "" {
		newVersion = defaultVersion
		if _, err := os.Stat(cargoFile); os.IsNotExist(err) {
			cargo := fmt.Sprintf(`[package]
name                   = "%s"
version                = "%s"
edition                = "2021"
`, library.Name, newVersion)
			if err := os.WriteFile(cargoFile, []byte(cargo), 0644); err != nil {
				return err
			}
		} else if err := UpdateCargoVersion(cargoFile, newVersion); err != nil {
			return err
		}
	} else {
		var err error
		newVersion, err = semver.DeriveNext(semver.Minor, currentVersion,
			semver.DeriveNextOptions{
				BumpVersionCore:       true,
				DowngradePreGAChanges: true,
			})
		if err != nil {
			return err
		}
		if err := UpdateCargoVersion(cargoFile, newVersion); err != nil {
			return err
		}
	}
	library.Version = newVersion
	return nil
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
		libSrcDir = strings.ReplaceAll(libCfg.Name, "-", "/")
		if cfg.Default == nil {
			return ""
		}
	}
	return DefaultOutput(libSrcDir, cfg.Default.Output)

}
