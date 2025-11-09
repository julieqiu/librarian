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
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"gopkg.in/yaml.v3"
)

type editRunner struct {
	path     string
	metadata []string
	language string
	keep     []string
	remove   []string
	exclude  []string
	repoRoot string
}

func newEditRunner(args []string, metadata []string, language string, keep, remove, exclude []string) (*editRunner, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("missing required argument <path>")
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments, expected: librarian edit <path> [flags]")
	}

	path := args[0]

	// Get current working directory as repo root
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return &editRunner{
		path:     path,
		metadata: metadata,
		language: language,
		keep:     keep,
		remove:   remove,
		exclude:  exclude,
		repoRoot: repoRoot,
	}, nil
}

func (r *editRunner) run(ctx context.Context) error {
	_ = ctx
	artifactPath := filepath.Join(r.repoRoot, r.path)
	configPath := filepath.Join(artifactPath, ".librarian.yaml")

	// Check if .librarian.yaml exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf(".librarian.yaml does not exist at %s", configPath)
	}

	// Read the artifact state
	artifactState, err := config.ReadArtifactState(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to read .librarian.yaml: %w", err)
	}

	// If no flags provided, display current configuration
	if len(r.metadata) == 0 && r.language == "" && len(r.keep) == 0 && len(r.remove) == 0 && len(r.exclude) == 0 {
		return r.displayConfig(artifactState)
	}

	// Apply edits
	modified := false

	// Update metadata
	if len(r.metadata) > 0 {
		if artifactState.Generate == nil {
			return fmt.Errorf("artifact does not have a generate section (metadata only applies to generated code)")
		}
		if err := r.updateMetadata(artifactState); err != nil {
			return err
		}
		modified = true
	}

	// Update language-specific metadata
	if r.language != "" {
		if artifactState.Generate == nil {
			return fmt.Errorf("artifact does not have a generate section (language metadata only applies to generated code)")
		}
		if err := r.updateLanguage(artifactState); err != nil {
			return err
		}
		modified = true
	}

	// Update keep list
	if len(r.keep) > 0 {
		if artifactState.Generate == nil {
			return fmt.Errorf("artifact does not have a generate section (keep list only applies to generated code)")
		}
		artifactState.Generate.Keep = append(artifactState.Generate.Keep, r.keep...)
		modified = true
		slog.Info("added to keep list", "files", r.keep)
	}

	// Update remove list
	if len(r.remove) > 0 {
		if artifactState.Generate == nil {
			return fmt.Errorf("artifact does not have a generate section (remove list only applies to generated code)")
		}
		artifactState.Generate.Remove = append(artifactState.Generate.Remove, r.remove...)
		modified = true
		slog.Info("added to remove list", "files", r.remove)
	}

	// Update exclude list
	if len(r.exclude) > 0 {
		if artifactState.Generate == nil {
			return fmt.Errorf("artifact does not have a generate section (exclude list only applies to generated code)")
		}
		artifactState.Generate.Exclude = append(artifactState.Generate.Exclude, r.exclude...)
		modified = true
		slog.Info("added to exclude list", "files", r.exclude)
	}

	// Write back if modified
	if modified {
		if err := config.WriteArtifactState(artifactPath, artifactState); err != nil {
			return fmt.Errorf("failed to write .librarian.yaml: %w", err)
		}
		slog.Info("updated .librarian.yaml", "path", configPath)
		fmt.Printf("Updated configuration for %s\n", r.path)
	}

	return nil
}

func (r *editRunner) updateMetadata(artifactState *config.ArtifactState) error {
	if artifactState.Generate.Metadata == nil {
		artifactState.Generate.Metadata = &config.ArtifactMetadata{}
	}

	for _, item := range r.metadata {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid metadata format %q, expected KEY=VALUE", item)
		}
		key, value := parts[0], parts[1]

		// Set the metadata field
		switch key {
		case "name_pretty":
			artifactState.Generate.Metadata.NamePretty = value
		case "product_documentation":
			artifactState.Generate.Metadata.ProductDocumentation = value
		case "client_documentation":
			artifactState.Generate.Metadata.ClientDocumentation = value
		case "issue_tracker":
			artifactState.Generate.Metadata.IssueTracker = value
		case "release_level":
			if value != "stable" && value != "preview" {
				return fmt.Errorf("release_level must be 'stable' or 'preview', got %q", value)
			}
			artifactState.Generate.Metadata.ReleaseLevel = value
		case "library_type":
			artifactState.Generate.Metadata.LibraryType = value
		case "api_id":
			artifactState.Generate.Metadata.APIID = value
		case "api_shortname":
			artifactState.Generate.Metadata.APIShortname = value
		case "api_description":
			artifactState.Generate.Metadata.APIDescription = value
		case "default_version":
			artifactState.Generate.Metadata.DefaultVersion = value
		default:
			return fmt.Errorf("unknown metadata field %q", key)
		}

		slog.Info("set metadata", "key", key, "value", value)
	}

	return nil
}

func (r *editRunner) updateLanguage(artifactState *config.ArtifactState) error {
	// Parse language metadata: LANG:KEY=VALUE
	parts := strings.SplitN(r.language, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid language format %q, expected LANG:KEY=VALUE", r.language)
	}

	langParts := strings.SplitN(parts[1], "=", 2)
	if len(langParts) != 2 {
		return fmt.Errorf("invalid language format %q, expected LANG:KEY=VALUE", r.language)
	}

	lang := parts[0]
	key := langParts[0]
	value := langParts[1]

	// Validate language matches expected values
	validKeys := map[string][]string{
		"go":     {"module"},
		"python": {"package"},
		"rust":   {"crate"},
		"dart":   {"package"},
	}

	expectedKeys, ok := validKeys[lang]
	if !ok {
		return fmt.Errorf("invalid language %q, must be one of: go, python, rust, dart", lang)
	}

	validKey := false
	for _, k := range expectedKeys {
		if key == k {
			validKey = true
			break
		}
	}
	if !validKey {
		return fmt.Errorf("invalid key %q for language %s, expected: %v", key, lang, expectedKeys)
	}

	// Initialize language map if needed
	if artifactState.Generate.Language == nil {
		artifactState.Generate.Language = make(map[string]string)
	}

	artifactState.Generate.Language[key] = value
	slog.Info("set language metadata", "language", lang, "key", key, "value", value)

	return nil
}

func (r *editRunner) displayConfig(artifactState *config.ArtifactState) error {
	fmt.Printf("Configuration for %s:\n\n", r.path)

	data, err := yaml.Marshal(artifactState)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	fmt.Println(string(data))
	return nil
}
