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

func TestRead(t *testing.T) {
	got, err := Read("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := &Config{
		Language: "rust",
		Sources: &Sources{
			Discovery: &Source{
				Commit: "b27c80574e918a7e2a36eb21864d1d2e45b8c032",
				SHA256: "67c8d3792f0ebf5f0582dce675c379d0f486604eb0143814c79e788954aa1212",
			},
			Googleapis: &Source{
				Commit: "ded7ed1e4cce7c165c56a417572cebea9bc1d82c",
				SHA256: "839e897c39cada559b97d64f90378715a4a43fbc972d8cf93296db4156662085",
			},
		},
		Default: &Default{
			Output:       "src/generated/",
			ReleaseLevel: "stable",
			TagFormat:    "{name}/v{version}",
			Rust: &RustDefault{
				DisabledRustdocWarnings: []string{
					"redundant_explicit_links",
					"broken_intra_doc_links",
				},
				PackageDependencies: []*RustPackageDependency{
					{Name: "bytes", Package: "bytes", ForceUsed: true},
					{Name: "serde", Package: "serde", ForceUsed: true},
				},
			},
		},
		Libraries: []*Library{
			{
				Name:    "google-cloud-secretmanager-v1",
				Version: "0.1.0",
				APIs: []*API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			{
				Name:    "google-cloud-storage-v2",
				Version: "0.2.0",
				APIs: []*API{
					{Path: "google/cloud/storage/v2"},
				},
				Rust: &RustCrate{
					RustDefault: RustDefault{
						DisabledRustdocWarnings: []string{"rustdoc::bare_urls"},
					},
				},
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestWrite(t *testing.T) {
	want, err := Read("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(want); err != nil {
		t.Fatal(err)
	}
	var got Config
	if err := yaml.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, &got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFill(t *testing.T) {
	defaults := &Default{
		Output:       "src/generated/",
		ReleaseLevel: "stable",
	}
	for _, test := range []struct {
		name     string
		defaults *Default
		lib      *Library
		want     *Library
	}{
		{
			name:     "fills empty fields",
			defaults: defaults,
			lib:      &Library{},
			want: &Library{
				Output:       "src/generated/",
				ReleaseLevel: "stable",
			},
		},
		{
			name:     "preserves existing values",
			defaults: defaults,
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
			name:     "partial fill",
			defaults: defaults,
			lib:      &Library{Output: "custom/output/"},
			want: &Library{
				Output:       "custom/output/",
				ReleaseLevel: "stable",
			},
		},
		{
			name:     "nil defaults",
			defaults: nil,
			lib:      &Library{Output: "foo/"},
			want:     &Library{Output: "foo/"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.lib.Fill(test.defaults)
			if diff := cmp.Diff(test.want, test.lib); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
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
					RustDefault: RustDefault{
						PackageDependencies: []*RustPackageDependency{
							{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
					},
				},
			},
		},
		{
			name: "merges package dependencies",
			lib: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						PackageDependencies: []*RustPackageDependency{
							{Name: "custom", Package: "custom-pkg"},
						},
					},
				},
			},
			want: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						PackageDependencies: []*RustPackageDependency{
							{Name: "custom", Package: "custom-pkg"},
							{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
					},
				},
			},
		},
		{
			name: "library overrides default",
			lib: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						PackageDependencies: []*RustPackageDependency{
							{Name: "wkt", Package: "custom-wkt"},
						},
					},
				},
			},
			want: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						PackageDependencies: []*RustPackageDependency{
							{Name: "wkt", Package: "custom-wkt"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
					},
				},
			},
		},
		{
			name: "preserves existing warnings",
			lib: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						DisabledRustdocWarnings: []string{"custom_warning"},
					},
				},
			},
			want: &Library{
				Rust: &RustCrate{
					RustDefault: RustDefault{
						PackageDependencies: []*RustPackageDependency{
							{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
							{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
						},
						DisabledRustdocWarnings: []string{"custom_warning"},
					},
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
