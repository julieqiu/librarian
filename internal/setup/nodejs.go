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
	npmGAPICGeneratorTypescript  = "gapic-generator-typescript"
	googleCloudNodeCore = "google-cloud-node-core"

	githubTarballURL = "https://github.com/googleapis/%s/archive/%s.tar.gz"
)

func installNodejsTool(ctx context.Context, tool ToolVersion) error {
	switch tool.Name {
	case npmGAPICGeneratorTypescript:
		return installGapicGeneratorTypescript(ctx, tool)
	case pipSynthtool:
		return pipInstallSynthtool(ctx, tool)
	default:
		return npmInstall(ctx, tool)
	}
}

func npmInstall(ctx context.Context, tool ToolVersion) error {
	pkg := tool.Name + "@" + tool.Version
	return command.Run(ctx, "npm", "install", "-g", pkg)
}

func installGapicGeneratorTypescript(ctx context.Context, tool ToolVersion) error {
	tmpDir, err := os.MkdirTemp("", npmGAPICGeneratorTypescript+"-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tarball := filepath.Join(tmpDir, googleCloudNodeCore+".tar.gz")
	url := fmt.Sprintf(githubTarballURL, googleCloudNodeCore, tool.Version)
	if err := fetch.Download(ctx, tarball, url, tool.Checksum); err != nil {
		return fmt.Errorf("downloading %s: %w", googleCloudNodeCore, err)
	}

	extractDir := filepath.Join(tmpDir, "src")
	if err := fetch.ExtractTarball(tarball, extractDir); err != nil {
		return fmt.Errorf("extracting %s: %w", googleCloudNodeCore, err)
	}

	dir := filepath.Join(extractDir, "generator", npmGAPICGeneratorTypescript)
	if err := command.RunInDir(ctx, dir, "npm", "install"); err != nil {
		return err
	}
	if err := command.RunInDir(ctx, dir, "npm", "run", "compile"); err != nil {
		return err
	}
	return command.RunInDir(ctx, dir, "npm", "link")
}
