// Copyright 2025 Google LLC
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

package config

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestFill(t *testing.T) {
	defaults := &Default{
		Output: "src/generated/",
		Generate: &DefaultGenerate{
			ReleaseLevel: "stable",
		},
	}

	for _, test := range []struct {
		name string
		lib  *Library
		want *Library
	}{
		{
			name: "fills empty fields",
			lib:  &Library{},
			want: &Library{
				// Output is not filled because Channel is empty.
				// Output is derived from channel, not directly from default.output.
				ReleaseLevel: "stable",
			},
		},
		{
			name: "preserves existing values",
			lib: &Library{
				Output:       "custom/output/",
				ReleaseLevel: "preview",
			},
			want: &Library{
				Output:       "custom/output/",
				ReleaseLevel: "preview",
			},
		},
		{
			name: "partial fill",
			lib: &Library{
				Output: "custom/output/",
			},
			want: &Library{
				Output:       "custom/output/",
				ReleaseLevel: "stable",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.lib.Fill(defaults)
			if diff := cmp.Diff(test.want, test.lib); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFill_NoDefault(t *testing.T) {
	lib := &Library{Output: "foo/"}
	lib.Fill(nil)
	if lib.Output != "foo/" {
		t.Errorf("got %q, want %q", lib.Output, "foo/")
	}
}

func TestFill_Rust(t *testing.T) {
	defaults := &Default{
		Rust: &RustDefault{
			PackageDependencies: []*RustPackageDependency{
				{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
				{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
			},
			DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
		},
	}

	for _, test := range []struct {
		name string
		lib  *Library
		want *Library
	}{
		{
			name: "fills rust defaults",
			lib:  &Library{},
			want: &Library{
				Rust: &RustCrate{
					PackageDependencies: []RustPackageDependency{
						{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
						{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
					},
					DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
				},
			},
		},
		{
			name: "merges rust defaults with existing values",
			lib: &Library{
				Rust: &RustCrate{
					PackageDependencies: []RustPackageDependency{
						{Name: "custom", Package: "custom-pkg"},
					},
					DisabledRustdocWarnings: []string{"custom_warning"},
				},
			},
			want: &Library{
				Rust: &RustCrate{
					// Default deps are prepended to custom deps.
					PackageDependencies: []RustPackageDependency{
						{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
						{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						{Name: "custom", Package: "custom-pkg"},
					},
					// Custom warnings are preserved (not overwritten by defaults).
					DisabledRustdocWarnings: []string{"custom_warning"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.lib.Fill(defaults)
			if diff := cmp.Diff(test.want, test.lib); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadWrite(t *testing.T) {
	cfg, err := Read("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	enc := yaml.NewEncoder(&got)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		t.Fatal(err)
	}

	wantCfg, err := Read("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var gotCfg Config
	if err := yaml.Unmarshal(got.Bytes(), &gotCfg); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(wantCfg, &gotCfg); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
