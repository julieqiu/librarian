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
	"runtime"

	"github.com/googleapis/librarian/internal/fetch"
)

const protocURL = "https://github.com/protocolbuffers/protobuf/releases/download/v%s/protoc-%s-%s-%s.zip"

func installProtoc(ctx context.Context, tool ToolVersion) error {
	arch, err := protocArch()
	if err != nil {
		return err
	}
	osName, err := protocOS()
	if err != nil {
		return err
	}

	url := fmt.Sprintf(protocURL,
		tool.Version, tool.Version, osName, arch)

	tmpDir, err := os.MkdirTemp("", "librarian-protoc-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	zipFile := filepath.Join(tmpDir, "protoc.zip")
	if err := fetch.Download(ctx, zipFile, url, tool.Checksum); err != nil {
		return fmt.Errorf("downloading protoc: %w", err)
	}

	prefix := os.Getenv("PROTOC_PREFIX")
	if prefix == "" {
		prefix = "/usr/local"
	}
	if err := fetch.ExtractZip(zipFile, prefix); err != nil {
		return fmt.Errorf("extracting protoc: %w", err)
	}
	return nil
}

func protocArch() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64", nil
	case "arm64":
		return "aarch_64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}
}

func protocOS() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "osx", nil
	case "linux":
		return "linux", nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
