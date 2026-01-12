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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/semver"
)

const defaultVersion = "0.1.0"

// ReleaseLibrary bumps version for Cargo.toml and .sidekick.toml files.
// It returns the updated library to make the mutation explicit.
func ReleaseLibrary(library *config.Library) (*config.Library, error) {
	newVersion := defaultVersion
	if library.Version != "" {
		v, err := semver.DeriveNext(semver.Minor, library.Version,
			semver.DeriveNextOptions{
				BumpVersionCore:       true,
				DowngradePreGAChanges: true,
			})
		if err != nil {
			return nil, err
		}
		newVersion = v
	}

	cargoPath := filepath.Join(library.Output, "Cargo.toml")
	if err := updateOrCreateCargo(cargoPath, library.Name, newVersion); err != nil {
		return nil, err
	}
	if err := updateSidekickConfig(cargoPath, newVersion); err != nil {
		return nil, err
	}
	library.Version = newVersion
	return library, nil
}

// updateOrCreateCargo updates the Cargo.toml version, or creates it if it doesn't exist.
func updateOrCreateCargo(cargoPath, packageName, version string) error {
	_, err := os.Stat(cargoPath)
	switch {
	case err != nil && !os.IsNotExist(err):
		return err
	case os.IsNotExist(err):
		cargo := fmt.Sprintf(`[package]
name                   = "%s"
version                = "%s"
edition                = "2021"
`, packageName, version)
		return os.WriteFile(cargoPath, []byte(cargo), 0644)
	default:
		return updateCargoVersion(cargoPath, version)
	}
}
