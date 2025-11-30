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

// Package command provides helpers to execute external commands with logging.
package command

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Run executes a program (with arguments) and captures any error output.
func Run(command string, arg ...string) error {
	cmd := exec.Command(command, arg...)
	fmt.Fprintf(os.Stderr, "Running: %s\n", cmd.String())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %v\n%s", cmd, err, output)
	}
	return nil
}

// Require checks that the named tools are installed and in PATH.
func Require(names ...string) error {
	var missing []string
	for _, name := range names {
		if _, err := exec.LookPath(name); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("required tools not found in PATH: %s", strings.Join(missing, ", "))
	}
	return nil
}
