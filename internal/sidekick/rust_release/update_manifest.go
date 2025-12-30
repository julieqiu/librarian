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

package rustrelease

import (
	"os"

	"github.com/googleapis/librarian/internal/semver"
	"github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/pelletier/go-toml/v2"
	"github.com/googleapis/librarian/internal/librarian/rust"
)

func updateManifest(config *config.Release, lastTag, manifest string) ([]string, error) {
	needsBump, err := rust.ManifestVersionNeedsBump(gitExe(config), lastTag, manifest)
	if err != nil {
		return nil, err
	}
	if !needsBump {
		return nil, nil
	}
	contents, err := os.ReadFile(manifest)
	if err != nil {
		return nil, err
	}
	info := rust.Cargo{
		Package: &rust.CrateInfo{
			Publish: true,
		},
	}
	if err := toml.Unmarshal(contents, &info); err != nil {
		return nil, err
	}
	if !info.Package.Publish {
		return nil, nil
	}
	newVersion, err := semver.DeriveNextOptions{
		BumpVersionCore:       true,
		DowngradePreGAChanges: true,
	}.DeriveNext(semver.Minor, info.Package.Version)
	if err != nil {
		return nil, err
	}
	if err := rust.UpdateCargoVersion(manifest, newVersion); err != nil {
		return nil, err
	}
	if err := updateSidekickConfig(manifest, newVersion); err != nil {
		return nil, err
	}
	return []string{info.Package.Name}, nil
}
