package bazel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/bazel"
)

func TestReleaseLevel(t *testing.T) {
	tests := []struct {
		name         string
		bazelRL      string
		docGoContent string
		want         string
	}{
		{"bazel_ga", "ga", "", "stable"},
		{"bazel_alpha", "alpha", "", "preview"},
		{"bazel_beta", "beta", "", "preview"},
		{"import_path_alpha", "", "", "preview"},
		{"import_path_beta", "", "", "preview"},
		{"default_stable", "", "", "stable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			docGoPath := filepath.Join(tmpDir, "doc.go")
			if tt.docGoContent != "" {
				if err := os.WriteFile(docGoPath, []byte(tt.docGoContent), 0644); err != nil {
					t.Fatalf("writing doc.go: %v", err)
				}
			}

			bazelConfig, err := bazel.Parse(createFakeBazelFile(t, tt.bazelRL))
			if err != nil {
				t.Fatalf("bazel.Parse() failed: %v", err)
			}

			importPath := "cloud.google.com/go/foo/apiv1"
			if tt.name == "import_path_alpha" {
				importPath = "cloud.google.com/go/foo/apiv1alpha1"
			}
			if tt.name == "import_path_beta" {
				importPath = "cloud.google.com/go/foo/apiv1beta"
			}

			got, err := releaseLevel(importPath, bazelConfig)
			if err != nil {
				t.Fatalf("releaseLevel() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("releaseLevel() = %q, want %q", got, tt.want)
			}
		})
	}
}
