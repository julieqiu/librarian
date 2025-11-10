// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package golang implements the Go language container.
//
// The container provides the runtime environment with all Go dependencies
// (protoc, go tools, etc.) pre-installed. The actual generation logic lives
// in the generate/ package.
//
// Container mounts:
// - /librarian - Contains generate-request.json
// - /input - .librarian/generator-input directory from the language repository
// - /source - Googleapis repository (read-only)
// - /output - Directory where generated code is written
package golang

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/googleapis/librarian/internal/container/golang/generate"
)

// Install is a no-op for the Go container.
// All dependencies are installed at Docker image build time via the Dockerfile.
// The Dockerfile installs:
//   - Go 1.24.0
//   - protoc 25.7
//   - protoc-gen-go v1.35.2
//   - protoc-gen-go-grpc v1.3.0
//   - protoc-gen-go_gapic v0.54.0
//   - goimports (latest)
//   - staticcheck v2023.1.6
//
// This function exists to satisfy the container interface but does nothing
// since installation happens during `docker build`, not at runtime.
func Install() error {
	slog.Info("golang container: dependencies are pre-installed in the Docker image")
	return nil
}

// Generate executes the Go code generation workflow.
//
// This function provides the container interface that delegates to the
// generate.Generate() function which contains all the actual logic:
//  1. Read generate-request.json
//  2. Parse BUILD.bazel files
//  3. Invoke protoc with gapic plugins
//  4. Run post-processing (goimports, go mod init, go mod tidy)
//  5. Apply file permissions and flattening
//
// The container provides the runtime environment with all dependencies
// installed, while the generate package provides the orchestration logic.
func Generate(ctx context.Context) error {
	slog.Info("golang container: starting generation")

	// Create configuration from standard container mount points
	cfg := &generate.Config{
		LibrarianDir: "/librarian",
		InputDir:     "/input",
		OutputDir:    "/output",
		SourceDir:    "/source",
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		slog.Error("golang container: invalid configuration", "error", err)
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Execute generation logic
	if err := generate.Generate(ctx, cfg); err != nil {
		slog.Error("golang container: generation failed", "error", err)
		return fmt.Errorf("generation failed: %w", err)
	}

	slog.Info("golang container: generation completed successfully")
	return nil
}
