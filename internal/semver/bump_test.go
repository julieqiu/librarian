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

import "testing"

func TestBumpMinorPreservingPrerelease(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{"1.2", "1.2"},
		{"1.2.3", "1.3.0"},
		{"0.1.2", "0.2.0"},
		{"0.1.2-alpha", "0.2.0-alpha"},
		{"1.5.10-beta.2", "1.6.0-beta.2"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, err := BumpMinorPreservingPrerelease(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("mismatch, want=%s, got=%s", test.want, got)
			}
		})
	}
}

func TestBumpMinor(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{"1.2.3", "1.3.0"},
		{"0.1.2", "0.2.0"},
		{"1.2.3-alpha.1", "1.2.3-alpha.2"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, err := BumpMinor(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("mismatch, want=%s, got=%s", test.want, got)
			}
		})
	}
}
