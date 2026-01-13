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

package serviceconfig

import (
	"testing"
)

func TestAPIsNoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, api := range APIs {
		if seen[api.Path] {
			t.Errorf("duplicate API path: %s", api.Path)
		}
		seen[api.Path] = true
	}
}

func TestAPIsAlphabeticalOrder(t *testing.T) {
	for i := 1; i < len(APIs); i++ {
		prev := APIs[i-1].Path
		curr := APIs[i].Path
		if prev > curr {
			t.Errorf("APIs not in alphabetical order: %q comes after %q", prev, curr)
		}
	}
}

func TestFindTitle(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{
			name: "apps script types",
			path: "google/apps/script/type/gmail",
			want: "Google Apps Script Types",
		},
		{
			name: "gke hub types",
			path: "google/cloud/gkehub/v1/configmanagement",
			want: "GKE Hub Types",
		},
		{
			name: "no title override",
			path: "google/cloud/secretmanager/v1",
			want: "",
		},
		{
			name: "nonexistent path",
			path: "google/nonexistent/v1",
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ""
			for _, api := range APIs {
				if api.Path == test.path {
					got = api.Title
					break
				}
			}
			if got != test.want {
				t.Errorf("APIs[%q].Title = %q, want %q", test.path, got, test.want)
			}
		})
	}
}
