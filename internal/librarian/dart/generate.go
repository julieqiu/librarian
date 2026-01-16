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

package dart

import (
	"context"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	sidekickdart "github.com/googleapis/librarian/internal/sidekick/dart"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

// Generate generates a Dart client library.
func Generate(ctx context.Context, library *config.Library, googleapisDir string) error {
	sidekickConfig, err := toSidekickConfig(library, library.Channels[0], googleapisDir)
	if err != nil {
		return err
	}
	model, err := parser.CreateModel(sidekickConfig)
	if err != nil {
		return err
	}
	if err := sidekickdart.Generate(ctx, model, library.Output, sidekickConfig); err != nil {
		return err
	}
	return nil
}

// Format formats a generated Dart library.
func Format(ctx context.Context, library *config.Library) error {
	if err := command.Run(ctx, "dart", "format", library.Output); err != nil {
		return err
	}
	return nil
}

func toSidekickConfig(library *config.Library, ch *config.Channel, googleapisDir string) (*sidekickconfig.Config, error) {
	source := map[string]string{
		"googleapis-root": googleapisDir,
	}

	channel, err := serviceconfig.Find(googleapisDir, ch.Path)
	if err != nil {
		return nil, err
	}

	sidekickCfg := &sidekickconfig.Config{
		General: sidekickconfig.GeneralConfig{
			Language:            "dart",
			SpecificationFormat: "protobuf",
			ServiceConfig:       channel.ServiceConfig,
			SpecificationSource: ch.Path,
		},
		Source: source,
	}
	return sidekickCfg, nil
}
