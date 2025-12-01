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

package serviceconfig

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewAPI(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		want    *API
	}{
		{
			name:    "default ga API",
			apiPath: "google/cloud/accessapproval/v1",
			want: &API{
				RESTNumericEnums: true,
				Metadata:         true,
				Transport:        "grpc+rest",
				ReleaseLevel:     "ga",
			},
		},
		{
			name:    "default beta API",
			apiPath: "google/cloud/aiplatform/v1beta1",
			want: &API{
				RESTNumericEnums: true,
				Metadata:         true,
				Transport:        "grpc+rest",
				ReleaseLevel:     "beta",
			},
		},
		{
			name:    "default alpha API",
			apiPath: "google/cloud/retail/v2alpha",
			want: &API{
				RESTNumericEnums: true,
				Metadata:         true,
				Transport:        "grpc+rest",
				ReleaseLevel:     "alpha",
			},
		},
		{
			name:    "grpc-only transport override",
			apiPath: "google/bigtable/v2",
			want: &API{
				RESTNumericEnums: true,
				Metadata:         true,
				Transport:        "grpc",
				ReleaseLevel:     "ga",
			},
		},
		{
			name:    "rest-only transport override (non-DIREGAPIC)",
			apiPath: "google/cloud/apihub/v1",
			want: &API{
				RESTNumericEnums: true,
				Metadata:         true,
				Transport:        "rest",
				ReleaseLevel:     "beta",
			},
		},
		{
			name:    "compute v1 - rest + DIREGAPIC",
			apiPath: "google/cloud/compute/v1",
			want: &API{
				RESTNumericEnums: true,
				Metadata:         true,
				Transport:        "rest",
				ReleaseLevel:     "ga",
				DIREGAPIC:        true,
			},
		},
		{
			name:    "release level override - v1 but beta",
			apiPath: "google/ai/generativelanguage/v1",
			want: &API{
				RESTNumericEnums: true,
				Metadata:         true,
				Transport:        "grpc+rest",
				ReleaseLevel:     "beta",
			},
		},
		{
			name:    "release level override - v1beta but ga (DIREGAPIC)",
			apiPath: "google/cloud/compute/v1beta",
			want: &API{
				RESTNumericEnums: true,
				Metadata:         true,
				Transport:        "rest",
				ReleaseLevel:     "ga",
				DIREGAPIC:        true,
			},
		},
		{
			name:    "combined overrides - grpc and beta release level",
			apiPath: "google/monitoring/v3",
			want: &API{
				RESTNumericEnums: true,
				Metadata:         true,
				Transport:        "grpc",
				ReleaseLevel:     "beta",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := NewAPI(test.apiPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveReleaseLevel(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		want    string
	}{
		{"v1", "google/cloud/foo/v1", "ga"},
		{"v2", "google/cloud/foo/v2", "ga"},
		{"v1beta1", "google/cloud/foo/v1beta1", "beta"},
		{"v2beta", "google/cloud/foo/v2beta", "beta"},
		{"v1alpha", "google/cloud/foo/v1alpha", "alpha"},
		{"v1alpha1", "google/cloud/foo/v1alpha1", "alpha"},
		{"no version", "google/longrunning", "ga"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveReleaseLevel(test.apiPath)
			if got != test.want {
				t.Errorf("deriveReleaseLevel(%q) = %q, want %q", test.apiPath, got, test.want)
			}
		})
	}
}

func TestDerivePath(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		want    string
	}{
		{
			name:    "standard pattern",
			apiPath: "google/cloud/accessapproval/v1",
			want:    "google/cloud/accessapproval/v1/accessapproval_v1.yaml",
		},
		{
			name:    "override - billing budgets",
			apiPath: "google/cloud/billing/budgets/v1",
			want:    "google/cloud/billing/budgets/v1/billingbudgets.yaml",
		},
		{
			name:    "override - monitoring",
			apiPath: "google/monitoring/v3",
			want:    "google/monitoring/v3/monitoring.yaml",
		},
		{
			name:    "override - spanner",
			apiPath: "google/spanner/v1",
			want:    "google/spanner/v1/spanner.yaml",
		},
		{
			name:    "override - longrunning",
			apiPath: "google/longrunning",
			want:    "google/longrunning/longrunning.yaml",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DerivePath(test.apiPath)
			if got != test.want {
				t.Errorf("DerivePath(%q) = %q, want %q", test.apiPath, got, test.want)
			}
		})
	}
}
