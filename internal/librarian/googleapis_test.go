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

package librarian

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestDeriveGoLibraryName(t *testing.T) {
	for _, test := range []struct {
		apiPath string
		want    string
	}{
		{"google/cloud/speech/v1", "speech"},
		{"google/cloud/speech/v1p1beta1", "speech"},
		{"google/cloud/bigquery/storage/v1", "bigquery"},
		{"google/ai/generativelanguage/v1", "ai"},
		{"google/devtools/cloudbuild/v1", "devtools"},
		{"grafeas/v1", "grafeas"},
		{"", ""},
	} {
		t.Run(test.apiPath, func(t *testing.T) {
			got := deriveGoLibraryName(test.apiPath)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestDiscoverLibraries(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *config.Config
		want []string // library names
	}{
		{
			name: "discovers all",
			cfg:  &config.Config{},
			want: []string{"library", "grafeas", "speech"},
		},
		{
			name: "skips existing library by name",
			cfg: &config.Config{
				Libraries: []*config.Library{
					{Name: "speech"},
				},
			},
			want: []string{"library", "grafeas"},
		},
		{
			name: "skips existing library by API path",
			cfg: &config.Config{
				Libraries: []*config.Library{
					{APIs: []*config.API{{Path: "google/cloud/speech/v1"}}},
				},
			},
			// speech is still discovered because the library has no name,
			// but the v1 API path is skipped
			want: []string{"library", "grafeas", "speech"},
		},
		{
			name: "all covered by name",
			cfg: &config.Config{
				Libraries: []*config.Library{
					{Name: "speech"},
					{Name: "grafeas"},
					{Name: "library"},
				},
			},
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := discoverLibraries(test.cfg, "testdata/googleapis")
			var names []string
			for _, lib := range got {
				names = append(names, lib.Name)
			}
			slices.Sort(names)
			slices.Sort(test.want)
			if diff := cmp.Diff(test.want, names); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindUncoveredAPIs(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *config.Config
		want []string
	}{
		{
			name: "all uncovered",
			cfg:  &config.Config{},
			want: []string{
				"google/cloud/speech/v1",
				"google/cloud/speech/v1p1beta1",
				"google/cloud/speech/v2",
				"grafeas/v1",
				"library/two",
			},
		},
		{
			name: "some covered",
			cfg: &config.Config{
				Libraries: []*config.Library{
					{APIs: []*config.API{{Path: "google/cloud/speech/v1"}}},
					{APIs: []*config.API{{Path: "grafeas/v1"}}},
				},
			},
			want: []string{
				"google/cloud/speech/v1p1beta1",
				"google/cloud/speech/v2",
				"library/two",
			},
		},
		{
			name: "all covered",
			cfg: &config.Config{
				Libraries: []*config.Library{
					{APIs: []*config.API{{Path: "google/cloud/speech/v1"}}},
					{APIs: []*config.API{{Path: "google/cloud/speech/v1p1beta1"}}},
					{APIs: []*config.API{{Path: "google/cloud/speech/v2"}}},
					{APIs: []*config.API{{Path: "grafeas/v1"}}},
					{APIs: []*config.API{{Path: "library/two"}}},
				},
			},
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := findUncoveredAPIs(test.cfg, "testdata/googleapis")
			slices.Sort(got)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindServiceConfig(t *testing.T) {
	got, err := findServiceConfig("testdata/googleapis", "google/cloud/speech/v1")
	if err != nil {
		t.Fatal(err)
	}
	want := "google/cloud/speech/v1/speech_v1.yaml"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFill(t *testing.T) {
	defaults := &config.Default{
		Output: "src/generated/",
	}
	for _, test := range []struct {
		name     string
		defaults *config.Default
		lib      *config.Library
		want     *config.Library
	}{
		{
			name:     "fills empty fields",
			defaults: defaults,
			lib:      &config.Library{},
			want: &config.Library{
				APIs:   []*config.API{{}},
				Output: "src/generated",
			},
		},
		{
			name:     "preserves existing values",
			defaults: defaults,
			lib: &config.Library{
				Output: "custom/output/",
			},
			want: &config.Library{
				APIs:   []*config.API{{}},
				Output: "custom/output/",
			},
		},
		{
			name:     "partial fill",
			defaults: defaults,
			lib:      &config.Library{Output: "custom/output/"},
			want: &config.Library{
				APIs:   []*config.API{{}},
				Output: "custom/output/",
			},
		},
		{
			name:     "nil defaults",
			defaults: nil,
			lib:      &config.Library{Output: "foo/"},
			want:     &config.Library{APIs: []*config.API{{}}, Output: "foo/"},
		},
		{
			name:     "derives output from name",
			defaults: defaults,
			lib:      &config.Library{Name: "google-cloud-speech-v1"},
			want: &config.Library{
				Name:   "google-cloud-speech-v1",
				APIs:   []*config.API{{Path: "google/cloud/speech/v1"}},
				Output: "src/generated/cloud/speech/v1",
			},
		},
		{
			name:     "non-google API path",
			defaults: defaults,
			lib:      &config.Library{APIs: []*config.API{{Path: "grafeas/v1"}}},
			want: &config.Library{
				APIs:   []*config.API{{Path: "grafeas/v1"}},
				Output: "src/generated/grafeas/v1",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			applyDefault(test.lib, test.defaults)
			if diff := cmp.Diff(test.want, test.lib); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFill_Rust(t *testing.T) {
	testDefault := &config.Default{
		Rust: &config.RustDefault{
			PackageDependencies: []*config.RustPackageDependency{
				{Name: "wkt", Package: "google-cloud-wkt", Source: "google.protobuf"},
				{Name: "iam_v1", Package: "google-cloud-iam-v1", Source: "google.iam.v1"},
			},
			DisabledRustdocWarnings: []string{"broken_intra_doc_links"},
		},
	}
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "fills rust defaults",
			lib:  &config.Library{},
			want: &config.Library{
				APIs: []*config.API{{}},
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
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
			lib: &config.Library{
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "custom", Package: "custom-pkg"},
						},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{{}},
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
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
			lib: &config.Library{
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
							{Name: "wkt", Package: "custom-wkt"},
						},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{{}},
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
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
			lib: &config.Library{
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						DisabledRustdocWarnings: []string{"custom_warning"},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{{}},
				Rust: &config.RustCrate{
					RustDefault: config.RustDefault{
						PackageDependencies: []*config.RustPackageDependency{
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
			applyDefault(test.lib, testDefault)
			if diff := cmp.Diff(test.want, test.lib); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
