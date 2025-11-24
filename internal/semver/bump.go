// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// BumpMinor is a convenience function that always bumps the minor version.
// This is equivalent to DeriveNext(Minor, version).
func BumpMinor(version string) (string, error) {
	return DeriveNext(Minor, version)
}

// BumpMinorPreservingPrerelease bumps the minor version but preserves the
// pre-release suffix unchanged. This is used for Rust crate version bumping
// where the pre-release label should remain stable across minor version bumps.
//
// Examples:
//   - "1.2.3" -> "1.3.0"
//   - "0.1.2" -> "0.2.0"
//   - "0.1.2-alpha" -> "0.2.0-alpha"
//
// If the version string is not in X.Y.Z format, it returns unchanged.
func BumpMinorPreservingPrerelease(version string) (string, error) {
	components := strings.SplitN(version, ".", 3)
	if len(components) != 3 {
		return version, nil
	}
	n, err := strconv.Atoi(components[1])
	if err != nil {
		return "", err
	}
	components[1] = fmt.Sprintf("%d", n+1)
	v := strings.Split(components[2], "-")
	v[0] = "0"
	components[2] = strings.Join(v, "-")
	return strings.Join(components, "."), nil
}
