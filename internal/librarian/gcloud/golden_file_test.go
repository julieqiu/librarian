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

package gcloud

import (
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/yaml"
)

var updateGolden = flag.Bool("update", false, "update gcloud golden files")

func TestGenerate_Golden(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	googleapisPath, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	var cases []string
	for _, e := range entries {
		if e.IsDir() {
			cases = append(cases, e.Name())
		}
	}

	for _, test := range cases {
		t.Run(test, func(t *testing.T) {
			caseDir := filepath.Join("testdata", test)
			cfgPath := filepath.Join(caseDir, "librarian.yaml")
			cfg, err := yaml.Read[config.Config](cfgPath)
			if err != nil {
				t.Fatal(err)
			}
			if len(cfg.Libraries) != 1 {
				t.Fatalf("expected exactly one library in %s, got %d", cfgPath, len(cfg.Libraries))
			}

			library := cfg.Libraries[0]
			outDir := filepath.Join(t.TempDir(), "out")
			library.Output = outDir
			src := &sources.Sources{
				Googleapis: googleapisPath,
			}
			if err := Generate(t.Context(), library, src); err != nil {
				t.Fatal(err)
			}

			expectedRoot := filepath.Join(caseDir, "expected", "surface")
			if *updateGolden {
				if err := os.RemoveAll(expectedRoot); err != nil && !errors.Is(err, fs.ErrNotExist) {
					t.Fatal(err)
				}
				if err := os.MkdirAll(expectedRoot, 0755); err != nil {
					t.Fatal(err)
				}
				if err := updateGoldenDir(expectedRoot, outDir); err != nil {
					t.Fatal(err)
				}
				return
			}

			if _, err := os.Stat(expectedRoot); errors.Is(err, fs.ErrNotExist) {
				t.Errorf("expected golden directory not found: %s; run with -update after reviewing the generated tree", expectedRoot)
				return
			}
			compareDirectories(t, expectedRoot, outDir)
		})
	}
}

func updateGoldenDir(dest, src string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func compareDirectories(t *testing.T, expectedDir, gotDir string) bool {
	t.Helper()
	allPass := true
	filepath.WalkDir(expectedDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(expectedDir, path)
		gotPath := filepath.Join(gotDir, relPath)
		if _, err := os.Stat(gotPath); errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: missing in output", relPath)
			allPass = false
			return nil
		}

		if !compareFiles(t, path, gotPath, relPath) {
			allPass = false
		}
		return nil
	})

	filepath.WalkDir(gotDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(gotDir, path)
		if err != nil {
			return err
		}

		expectedPath := filepath.Join(expectedDir, relPath)
		if _, err := os.Stat(expectedPath); errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: extra file generated in output", relPath)
			allPass = false
		}
		return nil
	})

	return allPass
}

func compareFiles(t *testing.T, expected, got, rel string) bool {
	t.Helper()
	wantContent, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("%s: failed to read expected file: %v", rel, err)
	}
	gotContent, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("%s: failed to read generated file: %v", rel, err)
	}

	if filepath.Ext(expected) == ".yaml" {
		wantYAML, err := yaml.Unmarshal[any](wantContent)
		if err != nil {
			t.Errorf("%s: failed to unmarshal expected YAML: %v", rel, err)
			return false
		}
		gotYAML, err := yaml.Unmarshal[any](gotContent)
		if err != nil {
			t.Errorf("%s: failed to unmarshal generated YAML: %v", rel, err)
			return false
		}
		if diff := cmp.Diff(*wantYAML, *gotYAML, cmp.AllowUnexported()); diff != "" {
			t.Errorf("%s mismatch (-want +got):\n%s", rel, diff)
			return false
		}
		return true
	}

	wantStr := string(wantContent)
	gotStr := string(gotContent)

	re := regexp.MustCompile(`# Copyright \d{4} Google LLC`)
	wantStr = re.ReplaceAllString(wantStr, `# Copyright <YEAR> Google LLC`)
	gotStr = re.ReplaceAllString(gotStr, `# Copyright <YEAR> Google LLC`)

	if diff := cmp.Diff(wantStr, gotStr); diff != "" {
		t.Errorf("%s mismatch (-want +got):\n%s", rel, diff)
		return false
	}
	return true
}
