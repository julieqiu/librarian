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

package librarian

import (
	"testing"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
)

func TestGenerateAction(t *testing.T) {
	t.Parallel()
	testActionConfig(t, cmdGenerate)
}

func TestInitAction(t *testing.T) {
	t.Parallel()
	testActionConfig(t, cmdInit)
}

func TestTagAndReleaseAction(t *testing.T) {
	t.Parallel()
	testActionConfig(t, cmdTagAndRelease)
}

func testActionConfig(t *testing.T, cmd *cli.Command) {
	t.Helper()
	for _, test := range []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "invalid work root",
			cfg: &config.Config{
				WorkRoot: "  ",
			},
		},
		{
			name: "invalid repo",
			cfg: &config.Config{
				Repo: "  ",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cmd.Config = test.cfg
			err := cmd.Action(t.Context(), cmd)
			if err == nil {
				t.Errorf("error should not nil")
			}
		})
	}
}
