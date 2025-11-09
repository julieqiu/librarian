# Librarian

Librarian automates the maintenance and release of versioned directories in a
repository. A directory managed by Librarian may contain either generated code
(for a client library) or handwritten code (for a tool or service).

**Repository model**: Each repository supports a single language (Go,
Python, Rust, or Dart) and can contain multiple artifacts for that language.
Repository capabilities are determined by which sections exist in `.librarian/config.yaml`:

- `generate` section present → repository supports code generation
- `release` section present → repository supports release management
- Both sections present → repository supports both

Each artifact can independently have generation and/or release enabled based
on which sections are present in its `.librarian.yaml` file.

Librarian records generation input, release state, and version history, and
provides commands to regenerate and release the code in a repeatable way.

## Overview

### Librarian

**Core commands**

- [librarian init](#repository-setup): Initialize repository for library management
- [librarian add](#managing-directories): Track a directory for management
- [librarian edit](#editing-artifact-configuration): Edit artifact configuration (metadata, keep, remove, exclude)
- [librarian remove](#removing-a-directory): Stop tracking a directory
- [librarian generate](#generating-a-client-library): Generate or regenerate code for tracked directories
- [librarian prepare](#preparing-a-release): Prepare a release with version updates and notes
- [librarian release](#publishing-a-release): Tag and publish a prepared release

**Configuration commands**

- [librarian config get](#configuration): Read a configuration value
- [librarian config set](#configuration): Set a configuration value
- [librarian config update](#configuration): Update toolchain versions to latest

**Inspection commands**

- [librarian list](#inspection): List all tracked directories
- [librarian status](#inspection): Show generation and release status
- [librarian history](#inspection): View release history

### Librarianops

**Automation commands**

- [librarianops generate](#automate-code-generation): Automate code generation workflow
- [librarianops prepare](#automate-release-preparation): Automate release preparation workflow
- [librarianops release](#automate-release-publishing): Automate release publishing workflow

## Repository Setup

```bash
librarian init [language]
```

Initializes a repository for library management. Repository capabilities are determined by which sections are created.

**Languages supported:**
- `go` - Uses Go generator container
- `python` - Uses Python generator container
- `rust` - Uses Rust generator container
- `dart` - Uses Dart generator container

**Example: Release-only repository**

```bash
librarian init
```

**Example** `.librarian/config.yaml`:

```yaml
librarian:
  version: v0.5.0

release:
  tag_format: '{name}-v{version}'
```

**What this enables:**
- `librarian add <path>` - Track handwritten code for release
- `librarian prepare <path>` - Prepare releases
- `librarian release <path>` - Publish releases

**Example: Repository with code generation and releases**

```bash
librarian init python
```

**Example** `.librarian/config.yaml`:

```yaml
librarian:
  version: v0.5.0
  language: python

generate:
  container:
    image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/python-librarian-generator
    tag: latest
  googleapis:
    repo: github.com/googleapis/googleapis
    ref: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0
  discovery:
    repo: github.com/googleapis/discovery-artifact-manager
    ref: f9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0
  dir: packages/

release:
  tag_format: '{name}-v{version}'
```

**What this enables:**
- `librarian add <path> <api>` - Generate code from API definitions
- `librarian generate <path>` - Regenerate code
- `librarian prepare <path>` - Prepare releases
- `librarian release <path>` - Publish releases

**Configuration fields:**

- `container.image` - Container registry path (without tag)
- `container.tag` - Container image tag (e.g., `latest`, `v1.0.0`)
- `googleapis.repo` - Repository location for googleapis (GitHub path or local directory relative to `.librarian/`)
- `googleapis.ref` - Git reference (commit SHA, branch name, or tag). Optional; if omitted, uses HEAD of default branch
- `discovery.repo` - Repository location for discovery-artifact-manager
- `discovery.ref` - Git reference. Optional; if omitted, uses HEAD of default branch
- `dir` - Directory where generated code is written (relative to repository root, with trailing `/`)

**Note**: The presence of the `generate` section enables generation commands.
The presence of the `release` section enables release commands.

## Managing Directories

### Adding a Directory

```bash
librarian add <path> [api...]
```

Tracks a directory for management. The sections created in `<path>/.librarian.yaml` depend on:
1. Which sections exist in `.librarian/config.yaml`
2. Whether APIs are provided

**In a release-only repository** (no `generate` section in config):
```bash
librarian add packages/my-tool
```

**Example** `packages/my-tool/.librarian.yaml`:
```yaml
release:
  version: null
```

**In a repository with generation** (has `generate` section in config):
```bash
# Add handwritten code (no APIs)
librarian add packages/my-tool
```

**Example** `packages/my-tool/.librarian.yaml`:
```yaml
release:
  version: null
```

```bash
# Add generated code (with APIs)
librarian add packages/google-cloud-secret-manager secretmanager/v1 secretmanager/v1beta2
```

When adding APIs, `librarian add` automatically:
1. Reads the BUILD.bazel file for each API path
2. Extracts generation configuration from the language-specific gapic rule (e.g., `py_gapic_library`)
3. Saves the configuration to `.librarian.yaml` for reproducible generation

**Example** `packages/google-cloud-secret-manager/.librarian.yaml`:
```yaml
generate:
  apis:
    - path: secretmanager/v1
      grpc_service_config: secretmanager_grpc_service_config.json
      service_yaml: secretmanager_v1.yaml
      transport: grpc+rest
      rest_numeric_enums: true
      opt_args:
        - warehouse-package-name=google-cloud-secret-manager
    - path: secretmanager/v1beta2
      grpc_service_config: secretmanager_grpc_service_config.json
      service_yaml: secretmanager_v1beta2.yaml
      transport: grpc+rest
      rest_numeric_enums: true
      opt_args:
        - warehouse-package-name=google-cloud-secret-manager
  commit: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0
  librarian: v0.5.0
  container:
    image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/python-librarian-generator
    tag: latest
  googleapis:
    repo: github.com/googleapis/googleapis
    ref: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0
  discovery:
    repo: github.com/googleapis/discovery-artifact-manager
    ref: f9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0

release:
  version: null
```

**API configuration fields** (extracted from BUILD.bazel):
- `grpc_service_config` - Retry configuration file path (relative to API directory)
- `service_yaml` - Service configuration file path
- `transport` - Transport protocol (e.g., `grpc+rest`, `grpc`)
- `rest_numeric_enums` - Whether to use numeric enums in REST
- `opt_args` - Additional generator options

These fields can be manually edited if you need to override the BUILD.bazel configuration.

**Note**: The `generate` section is only created when APIs are provided
AND the repository has a `generate` section in its config.
The `release` section is created if the repository has a `release` section in its config.

`--commit` writes a standard commit message for the change.

### Removing a Directory

```bash
librarian remove <path>
```

Removes `<path>/.librarian.yaml`. Source code is not modified.

## Editing Artifact Configuration

```bash
librarian edit <path> [flags]
```

Configure artifact-specific settings:

### Set Library Metadata

Library metadata is used to generate documentation and configure the package.

```bash
# Set metadata fields
librarian edit <path> \
  --metadata name_pretty="Secret Manager" \
  --metadata release_level=stable \
  --metadata product_documentation="https://cloud.google.com/secret-manager/docs" \
  --metadata api_description="Store and manage secrets"
```

**Available metadata fields:**
- `name_pretty` - Human-readable name (e.g., "Secret Manager")
- `product_documentation` - URL to product documentation
- `client_documentation` - URL to client library documentation
- `issue_tracker` - URL to issue tracker
- `release_level` - Release level: `stable` or `preview`
- `library_type` - Library type: `GAPIC_AUTO` or `GAPIC_COMBO`
- `api_id` - API ID (e.g., `secretmanager.googleapis.com`)
- `api_shortname` - Short API name (e.g., `secretmanager`)
- `api_description` - Description of the API
- `default_version` - Default API version (e.g., `v1`)

**Example** `packages/google-cloud-secret-manager/.librarian.yaml`:
```yaml
generate:
  apis:
    - path: secretmanager/v1
  metadata:
    name_pretty: "Secret Manager"
    product_documentation: "https://cloud.google.com/secret-manager/docs"
    release_level: "stable"
    api_description: "Store and manage secrets"
```

### Set Language-Specific Metadata

The language metadata should match the repository's language (set via `librarian init`).

```bash
# For Go repositories
librarian edit <path> --language go:module=github.com/user/repo

# For Python repositories
librarian edit <path> --language python:package=my-package

# For Rust repositories
librarian edit <path> --language rust:crate=my_crate

# For Dart repositories
librarian edit <path> --language dart:package=my_package
```

Language-specific metadata is used by generators and tooling for proper
package/module configuration.
The format is `--language LANG:KEY=VALUE` where LANG matches your repository's
language and KEY is the property name (module, package, or crate).

### Keep Files During Generation

```bash
librarian edit <path> --keep README.md --keep docs
```

Files and directories in the keep list are not overwritten during code generation.

### Remove Files After Generation

```bash
librarian edit <path> --remove temp.txt --remove build
```

Files in the remove list are deleted after code generation completes.

### Exclude Files From Release

```bash
librarian edit <path> --exclude tests --exclude .gitignore
```

Files in the exclude list are not included when creating releases.

### View Current Configuration

```bash
librarian edit <path>
```

Running `edit` without flags displays the current configuration for the artifact.

## Generating a Client Library

For artifacts with a `generate` section in their `.librarian.yaml`:

```bash
librarian generate <path>
```

Generates or regenerates code using the container and configuration from `.librarian/config.yaml`.
Librarian updates the artifact's `.librarian.yaml` automatically.

### How Generation Works

1. **librarian CLI** (Go binary):
   - Reads `.librarian/config.yaml` and artifact's `.librarian.yaml`
   - Clones googleapis at the specified commit SHA
   - Prepares request files for the container with API configurations from `.librarian.yaml`
   - Runs the generator container with appropriate mounts
   - Applies keep/remove/exclude rules to the output
   - Updates `.librarian.yaml` with generation metadata

2. **Generator container** (language-specific):
   - Reads request files from mounted directory (includes pre-parsed API configurations)
   - Executes protoc with language-specific plugins using the provided configurations
   - Runs post-processors (formatting, templates, etc.)
   - Runs validation and tests
   - Writes generated code to output directory

3. **librarian CLI** (continues):
   - Copies generated code to artifact directory
   - Preserves files in "keep" list
   - Removes files in "remove" list
   - Updates `.librarian.yaml` state

**Note**: BUILD.bazel parsing happens only once during `librarian add`. The extracted configuration is saved to `.librarian.yaml` and reused for all subsequent `librarian generate` commands. This makes generation faster and ensures reproducibility even if BUILD.bazel files change upstream.

`--commit` writes a standard commit message for the change.

### Regenerate All Artifacts

```bash
librarian generate --all
```

Regenerates all artifacts that have a `generate` section.

**Note**: This command only works in repositories that have a `generate`
section in `.librarian/config.yaml`,
and only affects artifacts that have a `generate` section in their `.librarian.yaml`.

## Releasing

### Preparing a Release

For artifacts with a `release` section in their `.librarian.yaml`:

```bash
librarian prepare <path>
```

Determines the next version, updates metadata, and prepares release notes.
Does not tag or publish.

**Example** `packages/google-cloud-secret-manager/.librarian.yaml`:

```yaml
release:
  version: v1.2.0
  prepared:
    version: v1.3.0
    commit: e4d5c6b7a8f9e0d1c2b3a4f5e6d7c8b9a0f1e2d3
```

Prepare all artifacts that have a `release` section:

```bash
librarian prepare --all
```

`--commit` writes a standard commit message for the change.

**Note**: This command only works in repositories that have a `release`
section in `.librarian/config.yaml`,
and only affects artifacts that have a `release` section in their `.librarian.yaml`.

### Publishing a Release

For artifacts with a `release` section and a prepared release:

```bash
librarian release <path>
```

Tags the prepared version and updates recorded release state. If no prepared
release exists, the command does nothing.

Release all prepared artifacts:

```bash
librarian release --all
```

**Example** `packages/google-cloud-secret-manager/.librarian.yaml` after release:

```yaml
release:
  version: v1.3.0
```

## Configuration

### Update Versions in config.yaml

Update toolchain information to latest:

```bash
librarian config update [key]
librarian config update --all
```

Supported keys:

- `generate.container` - Update container image to latest
- `generate.googleapis` - Update googleapis to latest commit
- `generate.discovery` - Update discovery-artifact-manager to latest commit

Set a configuration key explicitly:

```bash
librarian config set <key> <value>
```

Supported keys:

- `librarian.language` - Repository language
- `generate.dir` - Default generation directory (default: "generated")
- `generate.container.image` - Container image name
- `generate.container.tag` - Container image tag
- `generate.container` - Container image and tag (syntactic sugar)
- `generate.googleapis.repo` - Googleapis repository location
- `generate.googleapis.ref` - Googleapis git reference
- `generate.discovery.repo` - Discovery repository location
- `generate.discovery.ref` - Discovery git reference
- `release.tag_format` - Release tag format template

**Example: Set global generation directory**

```bash
librarian config set generate.dir packages
```

**Example: Update container image and tag**

```bash
# Set both image and tag at once (syntactic sugar)
librarian config set generate.container python-gen:v1.2.0

# Or set them independently
librarian config set generate.container.image python-gen
librarian config set generate.container.tag v1.2.0
```

## Inspection

View information about tracked directories and their release history.

List all tracked directories:

```bash
librarian list
```

Show the current generation and release status for a directory:

```bash
librarian status <path>
```

View the release history for a directory:

```bash
librarian history <path>
```

## Automation with librarianops

The `librarianops` command automates common librarian workflows for CI/CD pipelines.

### Configuration

**Flags:**

- `--project` - GCP project ID (default: `cloud-sdk-librarian-prod`)
- `--dry-run` - Print commands without executing them

```bash
# Use custom project
librarianops --project my-project generate

# Dry run to see what would be executed
librarianops --dry-run generate
```

### Automate Code Generation

```bash
librarianops generate
```

This runs:
1. `librarian generate --all --commit` - Regenerate all artifacts
2. `gh pr create --with-token=$(fetch token) --fill` - Create pull request

### Automate Release Preparation

```bash
librarianops prepare
```

This runs:
1. `librarian prepare --all --commit` - Prepare all artifacts
2. `gh pr create --with-token=$(fetch token) --fill` - Create pull request

### Automate Release Publishing

```bash
librarianops release
```

This runs:
1. `librarian release --all` - Release all prepared artifacts
2. `gh release create --with-token=$(fetch token) --notes-from-tag` - Create GitHub releases

## Architecture

### Container Boundary

Librarian uses a container-based architecture to isolate language-specific generation logic:

**librarian CLI (Go)** - Runs on host machine:
- Configuration management (read/write YAML files)
- Git operations (clone, checkout, commit, tag)
- BUILD.bazel parsing (during `librarian add` only)
- Orchestration (prepare directories, run container, apply rules)
- State management (update `.librarian.yaml` files)

**Generator Container (language-specific)** - Runs in Docker:
- Execute protoc with language-specific plugins
- Run post-processors (formatters, templates)
- Run validation and tests
- Write generated code to output directory

**Container interface:**
- Input: Request JSON files with pre-parsed API configurations
- Mounts: googleapis source, configuration, output directory
- Output: Generated code in output directory

This separation allows:
- **Language independence** - CLI doesn't need Python/Rust/etc. installed
- **Isolation** - Container has all dependencies, no conflicts with host
- **Reproducibility** - Same container + same inputs = identical output
- **Flexibility** - Swap container images without changing CLI
- **Simplicity** - Container doesn't need BUILD.bazel parsing logic

**Why parse BUILD.bazel in the CLI?**
- Parsing happens once during `librarian add`, not on every generation
- Configuration is saved in `.librarian.yaml` for transparency and reproducibility
- Users can manually edit configuration if BUILD.bazel is incorrect
- Container remains simple - just executes protoc with provided options
- Go has excellent Bazel parsing libraries (`github.com/bazelbuild/buildtools/build`)

### Local Development

**For library users** (no container/Python knowledge needed):
```bash
# Just works - container is pulled automatically
librarian generate --all
```

**For librarian developers** (Go development):
```bash
# Standard Go development workflow
go run ./cmd/librarian/main.go generate --all

# Or for faster iteration
go install ./cmd/librarian
librarian generate --all
```

**For container developers** (Python generator logic):
```bash
# Build custom container
docker build -t python-gen:dev -f .generator/Dockerfile .

# Point librarian at custom container
librarian config set generate.container python-gen:dev

# Test
librarian generate packages/google-cloud-secret-manager
```

## Notes

- Librarian does not modify code outside the tracked directories.
- Librarian records only information required for reproducibility and release
  automation.
- The system is designed so that `git log` and `.librarian.yaml` describe the
  full history of generation inputs and release versions.
- All configuration lives in YAML files - no hidden state or external databases.
