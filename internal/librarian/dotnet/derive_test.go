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

package dotnet

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDeriveAPIPath(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "secret_manager",
			input: "Google.Cloud.SecretManager.V1",
			want:  "cloud/secretmanager/v1",
		},
		{
			name:  "ai_platform",
			input: "Google.Cloud.AIPlatform.V1",
			want:  "cloud/aiplatform/v1",
		},
		{
			name:  "apps_card",
			input: "Google.Apps.Card.V1",
			want:  "apps/card/v1",
		},
		{
			name:  "long_running",
			input: "Google.LongRunning",
			want:  "longrunning",
		},
		{
			name:  "spanner_admin_database",
			input: "Google.Cloud.Spanner.Admin.Database.V1",
			want:  "cloud/spanner/admin/database/v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveAPIPath(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDefaultOutput(t *testing.T) {
	got := DefaultOutput("Google.Cloud.SecretManager.V1", "apis")
	want := "apis/Google.Cloud.SecretManager.V1"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
