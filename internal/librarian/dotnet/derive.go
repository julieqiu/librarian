// Copyright 2026 Google LLC
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

// Package dotnet provides .NET specific functionality for librarian.
package dotnet

import (
	"path/filepath"
	"strings"
)

// DeriveAPIPath derives an API path from a NuGet package ID.
// For example: Google.Cloud.SecretManager.V1 -> google/cloud/secretmanager/v1.
func DeriveAPIPath(name string) string {
	name = strings.TrimPrefix(name, "Google.")
	name = strings.ReplaceAll(name, ".", "/")
	return strings.ToLower(name)
}

// DefaultOutput returns the default output directory for a .NET library.
func DefaultOutput(name, defaultOutput string) string {
	return filepath.Join(defaultOutput, name)
}
