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

package java

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerateClirr(t *testing.T) {
	tmpDir := t.TempDir()
	protoModulePath := filepath.Join(tmpDir, "proto-google-cloud-test-v1")
	srcDir := filepath.Join(protoModulePath, "src", "main", "java", "com", "google", "cloud", "test", "v1")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	orBuilderFile := filepath.Join(srcDir, "TestOrBuilder.java")
	if err := os.WriteFile(orBuilderFile, []byte("package com.google.cloud.test.v1; public interface TestOrBuilder {}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := generateClirr(protoModulePath); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(protoModulePath, "clirr-ignored-differences.xml")
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	goldenPath := filepath.Join("testdata", "clirr", "clirr-ignored-differences.xml")
	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, got, 0644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s\n\nHint: run 'go test ./internal/librarian/java -v -update' to update golden files.", diff)
	}
}

func TestGenerateClirr_SkipExisting(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "clirr-ignored-differences.xml")
	initialContent := "manual content"
	if err := os.WriteFile(outputPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateClirr(tmpDir); err != nil {
		t.Fatal(err)
	}
	newContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(initialContent, string(newContent)); diff != "" {
		t.Errorf("generateClirr modified existing file (-want +got):\n%s", diff)
	}
}
