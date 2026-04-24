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

// Package gcloud provides a simple API for generating gcloud commands.
package gcloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	sidekickgcloud "github.com/googleapis/librarian/internal/sidekick/gcloud"
	"github.com/googleapis/librarian/internal/sidekick/gcloud/provider"
	"github.com/googleapis/librarian/internal/sources"
)

// gcloudBaseModule is the Python module path for the generated surfaces. It is
// hardcoded because gcloud.yaml (which could otherwise carry this value) is
// intentionally omitted from the librarian path.
const gcloudBaseModule = "googlecloudsdk"

// Generate generates gcloud commands for every API in library. It is the
// librarian dispatch entry point and infers overrides from the parsed API
// model rather than reading a gcloud.yaml file.
func Generate(ctx context.Context, library *config.Library, src *sources.Sources) error {
	for _, a := range library.APIs {
		if err := generateAPI(a, library, src); err != nil {
			return fmt.Errorf("api %q: %w", a.Path, err)
		}
	}
	return nil
}

func generateAPI(a *config.API, library *config.Library, src *sources.Sources) error {
	apiDir := filepath.Join(src.Googleapis, a.Path)

	includeList, err := collectProtoIncludeList(src.Googleapis, apiDir)
	if err != nil {
		return err
	}

	serviceConfig, err := findServiceConfig(apiDir)
	if err != nil {
		return err
	}

	model, err := provider.CreateAPIModel(src.Googleapis, includeList, serviceConfig, "", "")
	if err != nil {
		return err
	}

	overrides := buildOverrides(model)
	return sidekickgcloud.Generate(model, overrides, library.Output, gcloudBaseModule)
}

// collectProtoIncludeList returns a comma-separated list of .proto files
// under apiDir, each expressed as a path relative to googleapisDir (the
// format CreateAPIModel expects).
func collectProtoIncludeList(googleapisDir, apiDir string) (string, error) {
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		return "", fmt.Errorf("failed to read API directory %s: %w", apiDir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".proto" {
			continue
		}
		abs := filepath.Join(apiDir, entry.Name())
		rel, err := filepath.Rel(googleapisDir, abs)
		if err != nil {
			return "", fmt.Errorf("failed to resolve relative path for %s: %w", abs, err)
		}
		files = append(files, filepath.ToSlash(rel))
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no .proto files found in %s", apiDir)
	}
	return strings.Join(files, ","), nil
}

// findServiceConfig returns the absolute path of the single *.yaml service
// config that lives next to the protos. It returns an error when none or
// more than one yaml file is present.
func findServiceConfig(apiDir string) (string, error) {
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		return "", fmt.Errorf("failed to read API directory %s: %w", apiDir, err)
	}

	var yamls []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		yamls = append(yamls, filepath.Join(apiDir, entry.Name()))
	}
	switch len(yamls) {
	case 0:
		return "", fmt.Errorf("no service config yaml found in %s", apiDir)
	case 1:
		return yamls[0], nil
	default:
		return "", fmt.Errorf("expected exactly one service config yaml in %s, found %d", apiDir, len(yamls))
	}
}

// buildOverrides constructs an in-memory provider.Config inferred from the
// parsed API model. ReleaseTrackGA is hardcoded because gcloud.yaml is
// intentionally omitted from this path.
func buildOverrides(model *api.API) *provider.Config {
	return &provider.Config{
		ServiceName: serviceName(model),
		APIs: []provider.API{
			{
				Name:          model.Name,
				APIVersion:    apiVersion(model),
				ReleaseTracks: []provider.ReleaseTrack{provider.ReleaseTrackGA},
			},
		},
	}
}

// serviceName returns the DefaultHost of the first service in the model
// (e.g. "parallelstore.googleapis.com"). Returns "" if no services exist.
func serviceName(model *api.API) string {
	if len(model.Services) == 0 {
		return ""
	}
	return model.Services[0].DefaultHost
}

// apiVersion returns the trailing version component of the protobuf package
// name (e.g. "google.cloud.parallelstore.v1" -> "v1").
func apiVersion(model *api.API) string {
	parts := strings.Split(model.PackageName, ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
