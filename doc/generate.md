# Code Generation Design

This document describes the design for code generation in Librarian,
including the container architecture and generation flow for each supported language.

## Overview

Librarian uses a **command-based container architecture** where:
- **Host (librarian CLI)** orchestrates the generation flow in pure Go
- **Containers** execute explicit commands provided by the host
- **Communication** happens via command files that specify what to execute

Each language has:
1. A single container image with all dependencies pre-installed
2. Multiple container invocations per library generation (generate → format → test)
3. Host-side orchestration that prepares commands and applies file filtering

## Container Interface

### Command Structure

Containers receive explicit commands to execute via `/commands/commands.json`.
The container reads the commands and executes them sequentially.

### Container Mounts

All containers receive these mounts:

- `/commands/commands.json` - Commands to execute (read-only)
- `/source` - Googleapis repository (read-only)
- `/output` - Directory where generated code is written

### Container Implementation

Each container is a **simple command executor** that:
- Reads commands from `/commands/commands.json`
- Executes each command sequentially
- Exits when all commands complete

**Container maintainers:**
- **All containers**: Maintained by librarian team (simple Go code that executes commands)

**Generator tool maintainers:**
- **gapic-generator-python**: Maintained by Python team
- **gapic-generator-go**: Maintained by Go team
- **Sidekick**: Maintained in this repo under `internal/sidekick`

The container is language-agnostic. The host CLI constructs language-specific commands based on BUILD.bazel configuration.

## Python Container

### Dependencies

The Python container includes:
- Python 3.14
- `protoc` (Protocol Buffer compiler)
- `grpc-tools` (includes `protoc-gen-python` and `protoc-gen-grpc-python`)
- `gapic-generator-python` (Google API client generator)
- `synthtool` (Google's synthesis tool for templates and post-processing)
- `nox` (Testing framework)

### Generation Flow

**Container invocations**: 3

The host prepares `/commands/commands.json` for each phase.

**Phase 1: Code Generation**

```json
{
  "commands": [
    {
      "command": "python3",
      "args": [
        "-m", "grpc_tools.protoc",
        "--proto_path=/source",
        "--python_gapic_out=/output",
        "--python_gapic_opt=service-config=/source/google/cloud/secretmanager/v1/secretmanager_v1.yaml",
        "--python_gapic_opt=retry-config=/source/google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
        "--python_gapic_opt=transport=grpc+rest",
        "--python_gapic_opt=rest-numeric-enums",
        "--python_gapic_opt=warehouse-package-name=google-cloud-secret-manager",
        "/source/google/cloud/secretmanager/v1/service.proto"
      ]
    }
  ]
}
```

**Phase 2: Post-processing**

```json
{
  "commands": [
    {
      "command": "python3",
      "args": ["-m", "synthtool", "--templates", "/source/google/cloud/secretmanager/v1"]
    }
  ]
}
```

**Phase 3: Testing**

```json
{
  "commands": [
    {
      "command": "nox",
      "args": ["-s", "unit"]
    }
  ]
}
```

The host constructs these commands from BUILD.bazel configuration. The container just executes them.

### Host Responsibilities

After container exits, the host:
1. Applies `python.remove` file filtering rules
2. Applies `keep` rules
3. Copies generated code from `/output` to final location

## Go Container

### Dependencies

The Go container includes:
- Go 1.23
- `protoc` (Protocol Buffer compiler)
- `protoc-gen-go` (Go protocol buffer plugin)
- `protoc-gen-go-grpc` (Go gRPC plugin)
- `protoc-gen-go_gapic` (Google API client generator for Go)
- `goimports` (Go import formatter)

### Generation Flow

**Container invocations**: 3

The host prepares `/commands/commands.json` for each phase.

**Phase 1: Code Generation**

```json
{
  "commands": [
    {
      "command": "protoc",
      "args": [
        "--proto_path=/source",
        "--go_out=/output",
        "--go-grpc_out=/output",
        "--go_gapic_out=/output",
        "--go_gapic_opt=go-gapic-package=cloud.google.com/go/secretmanager/apiv1;secretmanager",
        "--go_gapic_opt=grpc-service-config=/source/google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
        "--go_gapic_opt=api-service-config=/source/google/cloud/secretmanager/v1/secretmanager_v1.yaml",
        "--go_gapic_opt=transport=grpc+rest",
        "/source/google/cloud/secretmanager/v1/service.proto"
      ]
    }
  ]
}
```

**Phase 2: Formatting and Build**

```json
{
  "commands": [
    {
      "command": "goimports",
      "args": ["-w", "."]
    },
    {
      "command": "go",
      "args": ["mod", "init", "cloud.google.com/go/secretmanager"]
    },
    {
      "command": "go",
      "args": ["mod", "tidy"]
    }
  ]
}
```

**Phase 3: Testing**

```json
{
  "commands": [
    {
      "command": "go",
      "args": ["build", "./..."]
    },
    {
      "command": "go",
      "args": ["test", "./...", "-short"]
    }
  ]
}
```

The host constructs these commands from BUILD.bazel configuration. The container just executes them.

### Host Responsibilities

After container exits, the host:
1. Applies `go.remove_regex` file filtering patterns
2. Applies `go.keep` file preservation rules
3. Copies generated code from `/output` to final location

## Rust Container

### Dependencies

The Rust container includes:
- Rust 1.75 toolchain
- `cargo` (Rust build tool)
- `taplo-cli` (TOML formatter)
- `typos-cli` (Spell checker)
- Sidekick code generator (pure Go, embedded in container)

### Generation Flow

**Container invocations**: 3

The host prepares `/commands/commands.json` for each phase.

**Phase 1: Code Generation**

```json
{
  "commands": [
    {
      "command": "sidekick",
      "args": [
        "generate",
        "--api-path=/source/google/bigtable/admin/v2",
        "--service-config=/source/google/bigtable/admin/v2/bigtableadmin_v2.yaml",
        "--output=/output"
      ]
    }
  ]
}
```

**Phase 2: Formatting**

```json
{
  "commands": [
    {
      "command": "cargo",
      "args": ["fmt"]
    },
    {
      "command": "taplo",
      "args": ["fmt", "Cargo.toml"]
    }
  ]
}
```

**Phase 3: Testing and Validation**

```json
{
  "commands": [
    {
      "command": "cargo",
      "args": ["test"]
    },
    {
      "command": "cargo",
      "args": ["clippy"]
    },
    {
      "command": "cargo",
      "args": ["doc"]
    },
    {
      "command": "typos",
      "args": ["."]
    }
  ]
}
```

**Note**: Rust is unique because the code generator (Sidekick) is pure Go, while Python and Go use external generators. Sidekick lives in `internal/sidekick` and is embedded in the Rust container.

The host constructs these commands from BUILD.bazel configuration. The container just executes them.

### Host Responsibilities

After container exits, the host:
1. Applies any file filtering rules (Rust typically doesn't need filtering)
2. Copies generated code from `/output` to final location

## Summary Comparison

| Language | Code Generation | Container Runs | Container Maintained By | Generator Tool Maintained By |
|----------|----------------|----------------|------------------------|------------------------------|
| **Python** | gapic-generator-python | 3 | Librarian team | Python team (gapic-generator-python) |
| **Go** | gapic-generator-go | 3 | Librarian team | Go team (gapic-generator-go) |
| **Rust** | Sidekick | 3 | Librarian team | Librarian team (internal/sidekick) |

## Key Design Principles

### 1. Container Executes Commands

The container is a simple command executor:
- Reads commands from `/commands/commands.json`
- Executes each command sequentially
- Exits when all commands complete

**Benefit**: Container is language-agnostic. No language expertise needed to maintain it.

### 2. Multiple Container Invocations

The container is called multiple times per library generation:
- Invocation 1: Code generation (run protoc/sidekick)
- Invocation 2: Formatting and build
- Invocation 3: Testing and validation

**Benefit**: Each phase is explicit. Easy to debug by inspecting commands.json for each phase.

### 3. Commands, Not Configuration

The host passes **commands** (how to generate), not configuration (what to generate):
- Host: "Run these exact protoc commands"
- Container: "Executing commands..."

**Benefit**: Explicit over implicit. You can see exactly what will run. Host constructs commands from BUILD.bazel configuration, so language-specific knowledge lives in BUILD.bazel, not Go code.

### 4. Host Constructs Commands

The host (librarian CLI) is responsible for:
- Reading `.librarian.yaml` configuration
- Ensuring googleapis is available
- Parsing BUILD.bazel to extract generator configuration
- Constructing commands.json for each phase
- Calling the container for each phase
- Applying keep/remove file rules between phases

**Benefit**: Host has minimal language-specific knowledge (just command construction from BUILD.bazel config). Adding new languages means adding command construction logic, not container logic.

### 5. Clear Ownership

- **Container**: Librarian team owns (simple command executor in Go)
- **Generator tools**: Language teams own (gapic-generator-python, gapic-generator-go, Sidekick)
- **BUILD.bazel configuration**: Language teams define what flags/options generators need
- **Scaffolding, versioning, releases**: Librarian team owns (generic across languages)

**Benefit**: Each team maintains what they know best. Librarian team can evolve scaffolding/releases in Go without language expertise.

## Container Implementation

The container is a simple command executor. It reads `/commands/commands.json` and executes each command sequentially.

**Example container code (cmd/container/main.go):**

```go
package main

import (
	"encoding/json"
	"os"
	"os/exec"
)

type CommandsFile struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func main() {
	// Read commands from /commands/commands.json
	data, err := os.ReadFile("/commands/commands.json")
	if err != nil {
		panic(err)
	}

	var cmds CommandsFile
	if err := json.Unmarshal(data, &cmds); err != nil {
		panic(err)
	}

	// Execute each command
	for _, cmd := range cmds.Commands {
		c := exec.Command(cmd.Command, cmd.Args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			panic(err)
		}
	}
}
```

**Dockerfile (language-specific):**

```dockerfile
# Python container
FROM python:3.14

# Install dependencies
RUN pip install protoc grpc-tools gapic-generator-python synthtool nox

# Copy command executor
COPY cmd/container/main /usr/local/bin/container

ENTRYPOINT ["/usr/local/bin/container"]
```

```dockerfile
# Go container
FROM golang:1.23

# Install dependencies
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN go install github.com/googleapis/gapic-generator-go/cmd/protoc-gen-go_gapic@latest
RUN go install golang.org/x/tools/cmd/goimports@latest

# Copy command executor
COPY cmd/container/main /usr/local/bin/container

ENTRYPOINT ["/usr/local/bin/container"]
```

```dockerfile
# Rust container
FROM rust:1.75

# Install dependencies
RUN cargo install taplo-cli typos-cli

# Copy Sidekick binary
COPY internal/sidekick/sidekick /usr/local/bin/sidekick

# Copy command executor
COPY cmd/container/main /usr/local/bin/container

ENTRYPOINT ["/usr/local/bin/container"]
```

**Benefits:**
- Language-agnostic container (same executor for all languages)
- Simple implementation (~30 lines of Go)
- Easy to debug (inspect commands.json to see what runs)
- Explicit over implicit

## Production Deployment

In production (CI/CD), librarian CLI calls containers:

```bash
# Build container (done by librarian team)
docker build -f Dockerfile.python -t us-central1-docker.pkg.dev/.../python-generator:v1.0.0 .
docker push us-central1-docker.pkg.dev/.../python-generator:v1.0.0

# Librarian CLI calls container (multiple times per library)
docker run \
    -v /tmp/commands.json:/commands/commands.json:ro \
    -v /path/to/googleapis:/source:ro \
    -v /path/to/output:/output \
    us-central1-docker.pkg.dev/.../python-generator:v1.0.0
```

The container ensures:
- Hermetic builds (pinned dependency versions)
- No pollution of host environment
- Consistent results across different developer machines and CI
- Librarian team controls container implementation (simple command executor)
