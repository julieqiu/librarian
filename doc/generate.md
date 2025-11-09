# Code Generation Design

This document describes the design for code generation in Librarian,
including the container architecture and generation flow for each supported language.

## Overview

Librarian uses a **command-based container architecture** where:
- **Host (librarian CLI)** orchestrates the generation flow in pure Go
- **Containers** execute external tools that require language-specific dependencies
- **Communication** happens via commands.json files containing commands to execute

Each language has:
1. A single container image with all dependencies pre-installed
2. Multiple container invocations during generation (each with different commands)
3. Host-side orchestration that manages the full generation lifecycle

## Container Interface

### Command Structure

All containers receive a commands.json file that lists commands to execute.
Each command specifies the executable name and its arguments.

The container executes each command in order and exits.

### Container Mounts

All containers receive these mounts:

- `/commands` - Contains commands.json (read-only)
- `/config` - Contains language-specific dependency files (requirements.txt, go.mod, etc.) (read-only)
- `/source` - Googleapis repository (read-only)
- `/output` - Directory where generated code is written

### Container Implementation

The container implementation is **language-agnostic**.
Each language container uses the same Go code that reads the commands.json
file and executes each command sequentially.

The only difference between language containers is which dependencies are installed in the Dockerfile.

### Configuration Files

Each language maintains its dependency files in `internal/container/{language}/config/`:

- **Python**: `internal/container/python/config/requirements.txt`
- **Go**: `internal/container/go/config/` (for tool dependencies)
- **Rust**: `internal/container/rust/config/` (for toolchain configuration)

These files are mounted at `/config` in the container and can be referenced by commands.

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

#### Invocation 1: Code Generation

The host prepares a commands.json file that tells the container to run gapic-generator-python
(via the grpc_tools.protoc module).
This command includes all the protobuf files to process and configuration
options like service configs,
retry configs, transport settings, and package naming.

Example commands.json:
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
        "/source/google/cloud/secretmanager/v1/resources.proto",
        "/source/google/cloud/secretmanager/v1/service.proto"
      ]
    }
  ]
}
```

The container executes this command to generate Python client library code.

Between invocations, the **host** applies file filtering rules (removes
namespace package files according to the python.remove configuration).

#### Invocation 2: Post-Processing

The host prepares a commands.json file that tells the container to run synthtool.
This Python script applies templates, fixes import headers in generated protobuf files,
and performs other code cleanup tasks.

Example commands.json:
```json
{
  "commands": [
    {
      "command": "python3",
      "args": [
        "-c",
        "from pathlib import Path; from synthtool.languages import python; output = Path('/output'); python.py_samples(output); python.fix_pb2_headers(); python.fix_pb2_grpc_headers()"
      ]
    }
  ]
}
```

#### Invocation 3: Testing

The host prepares a commands.json file that tells the container to run nox
with the unit test session.
This executes the test suite to verify the generated library works correctly.

Example commands.json:
```json
{
  "commands": [
    {
      "command": "nox",
      "args": [
        "-s", "unit-3.14(protobuf_implementation='upb')",
        "-f", "/output/packages/google-cloud-secret-manager/noxfile.py"
      ]
    }
  ]
}
```

### Host Responsibilities

Between container invocations, the host:
1. Applies `python.remove` file filtering rules
2. Copies generated code from staging to final location
3. Manages staging directories

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

#### Invocation 1: Code Generation

The host prepares a commands.json file that tells the container to run protoc
with Go-specific plugins (protoc-gen-go,
protoc-gen-go-grpc, and protoc-gen-go_gapic).
The command includes all protobuf files to process and configuration options like package names,
service configs, and transport settings.

Example commands.json:
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
        "/source/google/cloud/secretmanager/v1/resources.proto",
        "/source/google/cloud/secretmanager/v1/service.proto"
      ]
    }
  ]
}
```

The container executes protoc to generate Go client library code,
including protocol buffer types, gRPC stubs,
and GAPIC client code.

Between invocations, the **host**:
1. Flattens the output directory (protoc generates nested paths like cloud.google.com/go/secretmanager
which need to be moved to the root)
2. Applies go.remove_regex file filtering rules to remove generated files that should not be kept

#### Invocation 2: Post-Processing

The host prepares a commands.json file with multiple commands to format
code and initialize the Go module (for new modules only).

Example commands.json:
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

#### Invocation 3: Build and Test

The host prepares a commands.json file with commands to build and test the generated module.

Example commands.json:
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

### Host Responsibilities

Between container invocations, the host:
1. Flattens output directory structure
2. Applies `go.remove_regex` patterns
3. Applies `go.keep` file preservation rules
4. Copies generated code from staging to final location

## Rust Container

### Dependencies

The Rust container includes:
- Rust 1.75 toolchain
- `cargo` (Rust build tool)
- `taplo-cli` (TOML formatter)
- `typos-cli` (Spell checker)

### Generation Flow

**Container invocations**: 2

**Note**: Unlike Python and Go, Rust code generation happens **entirely
on the host** using Sidekick (pure Go).
The container is only used for formatting and validation.

#### Host Code Generation

The host runs Sidekick, which is pure Go code that:
1. Parses protobuf/OpenAPI/Discovery specifications
2. Generates Rust code using Go templates
3. Writes output to staging directory

No container is needed for code generation!

#### Invocation 1: Formatting

The host prepares a commands.json file with commands to format the generated Rust code.

Example commands.json:
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

#### Invocation 2: Testing and Validation

The host prepares a commands.json file with multiple validation commands to ensure code quality.

Example commands.json:
```json
{
  "commands": [
    {
      "command": "cargo",
      "args": ["test", "--package", "google-cloud-bigtable-admin-v2"]
    },
    {
      "command": "cargo",
      "args": ["clippy", "--package", "google-cloud-bigtable-admin-v2", "--", "--deny", "warnings"]
    },
    {
      "command": "env",
      "args": [
        "RUSTDOCFLAGS=-D warnings",
        "cargo", "doc", "--package", "google-cloud-bigtable-admin-v2", "--no-deps"
      ]
    },
    {
      "command": "typos",
      "args": []
    }
  ]
}
```

### Host Responsibilities

The host:
1. Generates all Rust code using Sidekick
2. Applies any file filtering rules
3. Copies generated code from staging to final location

## Summary Comparison

| Language | Code Generation | Container Runs | Total Commands | What Container Does |
|----------|----------------|----------------|----------------|-------------------|
| **Python** | Container (protoc + gapic-gen) | 3 | ~10 | Generate → Post-process → Test |
| **Go** | Container (protoc + plugins) | 3 | ~8 | Generate → Format → Build & Test |
| **Rust** | **Host** (sidekick in Go) | 2 | ~5 | Format → Test & Validate |

## Key Design Principles

### 1. Container is Stateless

Each container invocation is independent. The container:
- Reads `commands.json`
- Executes commands
- Exits

No state is preserved between invocations.

### 2. Host Orchestrates

The host (librarian CLI) is responsible for:
- Reading `.librarian.yaml` configuration
- Building command lists from configuration
- Writing `commands.json` files
- Calling the container multiple times
- Managing file operations between container runs

### 3. Commands are Explicit

The `commands.json` file contains **exact commands** to run,
not high-level instructions.
The container doesn't make decisions - it just executes.

### 4. Language-Agnostic Container Code

The Go code that runs inside containers is identical across languages. Only the Dockerfile (dependencies) differs.

### 5. Pure Go Where Possible

Rust demonstrates that when generation can be done in pure Go (Sidekick),
the container is only needed for tooling (formatting, testing).
This is preferable to external code generators.

## Local Development

For local development without Docker:

### Install Dependencies

```bash
# Python
go run cmd/container/main.go install --language=python

# Go
go run cmd/container/main.go install --language=go

# Rust
go run cmd/container/main.go install --language=rust
```

This reads dependency files from `internal/container/{language}/config/` and runs the appropriate install command:
- Python: `pip install -r internal/container/python/config/requirements.txt`
- Go: `go install` for tools listed in config
- Rust: `cargo install` or `rustup` for toolchain setup

### Generate Without Container

Once dependencies are installed, you can run generation locally:

```bash
go run cmd/container/main.go generate --language=python \
    --librarian=./testdata/.librarian \
    --source=./testdata/source \
    --output=./testdata/output
```

This executes the same commands that would run in the container, but on your local machine.

## Production Deployment

In production (CI/CD), containers are used:

```bash
# Build container
docker build --build-arg LANGUAGE=python -t python-container .

# Run generation (called by librarian CLI)
docker run \
    -v /path/to/commands:/commands:ro \
    -v /path/to/librarian/internal/container/python/config:/config:ro \
    -v /path/to/googleapis:/source:ro \
    -v /path/to/output:/output \
    python-container \
    generate --language=python
```

The container ensures:
- Hermetic builds (pinned dependency versions)
- No pollution of host environment
- Consistent results across different developer machines and CI
