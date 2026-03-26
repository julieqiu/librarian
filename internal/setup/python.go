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
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/fetch"
)

const (
	pipBlack     = "black"
	pipPandoc    = "pandoc"
	pipSynthtool = "synthtool"

	pandocURL    = "https://github.com/jgm/pandoc/releases/download/%s/pandoc-%s-linux-amd64.tar.gz"
	synthtoolURL = "gcp-synthtool@git+https://github.com/googleapis/synthtool@%s"
)

func installPythonTool(ctx context.Context, tool ToolVersion) error {
	if tool.Name == pipPandoc {
		return installPandoc(ctx, tool)
	}
	return pipInstall(ctx, tool)
}

func pipInstall(ctx context.Context, tool ToolVersion) error {
	switch tool.Name {
	case pipBlack:
		pkg := "black[jupyter]==" + tool.Version
		return command.Run(ctx, "pip", "install", "--break-system-packages", pkg)
	case pipSynthtool:
		return pipInstallSynthtool(ctx, tool)
	default:
		pkg := tool.Name + "==" + tool.Version
		return command.Run(ctx, "pip", "install", "--break-system-packages", pkg)
	}
}

func pipInstallSynthtool(ctx context.Context, tool ToolVersion) error {
	pkg := fmt.Sprintf(synthtoolURL, tool.Version)
	return command.Run(ctx, "pip", "install", "--break-system-packages", pkg)
}

func installPandoc(ctx context.Context, tool ToolVersion) error {
	tmpDir, err := os.MkdirTemp("", "librarian-pandoc-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	url := fmt.Sprintf(pandocURL, tool.Version, tool.Version)
	tarball := filepath.Join(tmpDir, "pandoc.tar.gz")
	if err := fetch.Download(ctx, tarball, url, tool.Checksum); err != nil {
		return fmt.Errorf("downloading pandoc: %w", err)
	}

	prefix := "/usr/local"
	if err := fetch.ExtractTarball(tarball, prefix); err != nil {
		return fmt.Errorf("extracting pandoc: %w", err)
	}
	return nil
}
