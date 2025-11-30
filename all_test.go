// Copyright 2024 Google LLC
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
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestGolangCILint(t *testing.T) {
	rungo(t, "tool", "golangci-lint", "run")
}

func TestGoImports(t *testing.T) {
	rungo(t, "tool", "goimports", "-d", ".")
}

func TestGoModTidy(t *testing.T) {
	rungo(t, "mod", "tidy", "-diff")
}

func TestYAMLFormat(t *testing.T) {
	rungo(t, "tool", "yamlfmt", "-lint", ".")
}

func rungo(t *testing.T, args ...string) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmdStr := fmt.Sprintf("go %s", strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s: %v\nStdout:\n%s\nStderr:\n%s", cmdStr, err, stdout.String(), stderr.String())
	}
	if stdout.Len() > 0 {
		t.Logf("%s:\n%s", cmdStr, stdout.String())
	}
}
