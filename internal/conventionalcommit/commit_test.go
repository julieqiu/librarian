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

package conventionalcommit

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	parser := NewParser()

	for _, test := range []struct {
		name    string
		message string
		want    *Commit
		wantErr bool
	}{
		{
			name:    "simple feat",
			message: "feat: add new feature",
			want: &Commit{
				Type:        "feat",
				Description: "add new feature",
				Footers:     map[string]string{},
			},
		},
		{
			name:    "fix with scope",
			message: "fix(api): correct validation",
			want: &Commit{
				Type:        "fix",
				Scope:       "api",
				Description: "correct validation",
				Footers:     map[string]string{},
			},
		},
		{
			name:    "breaking change with exclamation",
			message: "feat!: remove deprecated API",
			want: &Commit{
				Type:        "feat",
				Description: "remove deprecated API",
				IsBreaking:  true,
				Footers:     map[string]string{},
			},
		},
		{
			name: "with body and footer",
			message: `feat: add new endpoint

This adds a new REST endpoint for users.

Reviewed-by: Alice`,
			want: &Commit{
				Type:        "feat",
				Description: "add new endpoint",
				Body:        "This adds a new REST endpoint for users.",
				Footers: map[string]string{
					"reviewed-by": "Alice",
				},
			},
		},
		{
			name: "breaking change footer",
			message: `feat: change API signature

BREAKING CHANGE: The API now requires authentication`,
			want: &Commit{
				Type:        "feat",
				Description: "change API signature",
				IsBreaking:  true,
				Footers: map[string]string{
					"breaking-change": "The API now requires authentication",
				},
			},
		},
		{
			name:    "invalid - missing colon",
			message: "feat add something",
			wantErr: true,
		},
		{
			name:    "invalid - empty",
			message: "",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parser.Parse(test.message)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommitMethods(t *testing.T) {
	for _, test := range []struct {
		name          string
		commit        *Commit
		wantIsFeat    bool
		wantIsFix     bool
		wantHasFooter bool
	}{
		{
			name: "feat commit",
			commit: &Commit{
				Type:    "feat",
				Footers: map[string]string{},
			},
			wantIsFeat: true,
		},
		{
			name: "fix commit",
			commit: &Commit{
				Type:    "fix",
				Footers: map[string]string{},
			},
			wantIsFix: true,
		},
		{
			name: "commit with footer",
			commit: &Commit{
				Type: "feat",
				Footers: map[string]string{
					"reviewed-by": "Bob",
				},
			},
			wantIsFeat:    true,
			wantHasFooter: true,
		},
		{
			name: "docs commit",
			commit: &Commit{
				Type:    "docs",
				Footers: map[string]string{},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.commit.IsFeat(); got != test.wantIsFeat {
				t.Errorf("IsFeat() = %v, want %v", got, test.wantIsFeat)
			}
			if got := test.commit.IsFix(); got != test.wantIsFix {
				t.Errorf("IsFix() = %v, want %v", got, test.wantIsFix)
			}
			if got := test.commit.HasFooter(); got != test.wantHasFooter {
				t.Errorf("HasFooter() = %v, want %v", got, test.wantHasFooter)
			}
		})
	}
}
