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

package setup

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func goInstallTools(ctx context.Context) error {
	return command.Run(ctx, "go", "install", "tool")
}

// ToolVersion describes a tool and its version to install.
type ToolVersion struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version,omitempty"`
	Checksum string `yaml:"checksum,omitempty"`
}

// LanguageVersions holds the protoc and tool versions for a language.
type LanguageVersions struct {
	Protoc ToolVersion   `yaml:"protoc"`
	Tools  []ToolVersion `yaml:"tools,omitempty"`
}

//go:embed versions.yaml
var versionsData []byte

// Run installs all external dependencies for the given language.
func Run(ctx context.Context, language string) error {
	languages, err := yaml.Unmarshal[map[string]LanguageVersions](versionsData)
	if err != nil {
		return fmt.Errorf("parsing versions.yaml: %w", err)
	}
	v, ok := (*languages)[language]
	if !ok {
		return fmt.Errorf("unsupported language: %s", language)
	}
	if err := installProtoc(ctx, v.Protoc); err != nil {
		return fmt.Errorf("installing protoc: %w", err)
	}
	if language == config.LanguageGo {
		return goInstallTools(ctx)
	}
	for _, tool := range v.Tools {
		if err := installTool(ctx, language, tool); err != nil {
			return fmt.Errorf("installing %s: %w", tool.Name, err)
		}
	}
	return installConfigTools(ctx, language)
}

// installConfigTools reads librarian.yaml (if present) and installs any
// tools defined in release.tools, dispatching by installer name.
func installConfigTools(ctx context.Context, language string) error {
	if _, err := os.Stat(config.LibrarianYAML); err != nil {
		return nil
	}
	cfg, err := yaml.Read[config.Config](config.LibrarianYAML)
	if err != nil {
		return fmt.Errorf("reading %s: %w", config.LibrarianYAML, err)
	}
	if cfg.Release == nil || len(cfg.Release.Tools) == 0 {
		return nil
	}
	for installer, tools := range cfg.Release.Tools {
		for _, tool := range tools {
			tv := ToolVersion{Name: tool.Name, Version: tool.Version}
			if err := installConfigTool(ctx, language, installer, tv); err != nil {
				return fmt.Errorf("installing %s: %w", tool.Name, err)
			}
		}
	}
	return nil
}

func installConfigTool(ctx context.Context, language, installer string, tool ToolVersion) error {
	switch installer {
	case "cargo":
		return cargoInstall(ctx, tool)
	case "npm":
		return npmInstall(ctx, tool)
	case "pip":
		return pipInstall(ctx, tool)
	default:
		return fmt.Errorf("unknown installer: %s", installer)
	}
}

func installTool(ctx context.Context, language string, tool ToolVersion) error {
	switch language {
	case config.LanguageJava:
		return installJavaTool(ctx, tool)
	case config.LanguageNodejs:
		return installNodejsTool(ctx, tool)
	case config.LanguagePython:
		return installPythonTool(ctx, tool)
	case config.LanguageRust:
		return installRustTool(ctx, tool)
	default:
		return nil
	}
}
