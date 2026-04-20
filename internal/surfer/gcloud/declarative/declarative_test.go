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

package declarative

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestCommandMarshalDefault(t *testing.T) {
	for _, test := range []struct {
		name string
		cmd  *Command
		want string
	}{
		{
			name: "unset",
			cmd: &Command{
				HelpText: HelpText{Brief: "b", Description: "d"},
				Arguments: Arguments{Params: []Argument{
					{ArgName: "x", HelpText: "h", Default: Default{}},
				}},
			},
			want: `help_text:
    brief: b
    description: d
arguments:
    params:
        - arg_name: x
          help_text: h
          is_positional: false
          required: false
`,
		},
		{
			name: "set_nil",
			cmd: &Command{
				HelpText: HelpText{Brief: "b", Description: "d"},
				Arguments: Arguments{Params: []Argument{
					{ArgName: "x", HelpText: "h", Default: Default{Set: true}},
				}},
			},
			want: `help_text:
    brief: b
    description: d
arguments:
    params:
        - arg_name: x
          help_text: h
          is_positional: false
          required: false
          default: null
`,
		},
		{
			name: "set_value",
			cmd: &Command{
				HelpText: HelpText{Brief: "b", Description: "d"},
				Arguments: Arguments{Params: []Argument{
					{ArgName: "x", HelpText: "h", Default: Default{Set: true, Value: "hello"}},
				}},
			},
			want: `help_text:
    brief: b
    description: d
arguments:
    params:
        - arg_name: x
          help_text: h
          is_positional: false
          required: false
          default: hello
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b, err := yaml.Marshal(test.cmd)
			if err != nil {
				t.Fatal(err)
			}
			got := string(b)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
