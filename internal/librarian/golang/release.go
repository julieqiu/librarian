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
	"fmt"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
)

// ReleaseLibrary prepares a library for release by:
//   - Updating the library version
//   - Updating internal/version.go with the new version
//   - Updating snippet metadata files with the new version
//
// Note: Changelog updates (CHANGES.md) are no longer handled by this function.
func ReleaseLibrary(library *config.Library, version string) error {
	library.Version = version

	var moduleDir string
	if isRootRepoModule(library) {
		moduleDir = library.Output
	} else {
		moduleDir = filepath.Join(library.Output, library.Name)
	}

	// Update internal/version.go
	if err := generateInternalVersionFile(moduleDir, library.Version); err != nil {
		return fmt.Errorf("failed to update version for %s: %w", library.Name, err)
	}

	// Update snippet metadata (source and dest are the same since updating in place)
	if err := updateSnippetsMetadata(library, library.Output, library.Output); err != nil {
		return fmt.Errorf("failed to update snippet version for %s: %w", library.Name, err)
	}

	return nil
}

// isRootRepoModule returns whether the library is stored at the repository root.
// This applies to single-module repositories like gapic-generator-go, and to the
// special "root-module" library in google-cloud-go (containing civil, rpcreplay, etc).
//
// Edge case: gax-go has v1 code at the root but is NOT a root-module because v2
// lives in a subdirectory. We use library ID "v2" instead of "root-module".
//
// Most google-cloud-go modules live in subdirectories with separate paths for
// production code and generated snippets.
func isRootRepoModule(lib *config.Library) bool {
	return lib.Name == "root-module"
}
