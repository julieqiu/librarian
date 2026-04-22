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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	sidekickgcloud "github.com/googleapis/librarian/internal/sidekick/gcloud"
	"github.com/googleapis/librarian/internal/sidekick/gcloud/provider"
	"github.com/googleapis/librarian/internal/sources"
)

// Generate generates gcloud commands for the given library.
func Generate(_ context.Context, library *config.Library, src *sources.Sources) error {
	if len(library.APIs) != 1 {
		return fmt.Errorf("the gcloud generator only supports a single api per library")
	}
	g := library.Gcloud
	svcConfig, err := serviceconfig.Find(src.Googleapis, library.APIs[0].Path, config.LanguageGcloud)
	if err != nil {
		return err
	}
	model, err := provider.CreateAPIModel(src.Googleapis, g.IncludeList, svcConfig.ServiceConfig, g.DescriptorFiles, g.DescriptorFilesToGenerate)
	if err != nil {
		return err
	}
	return sidekickgcloud.Generate(model, nil, library.Output, g.BaseModule)
}
