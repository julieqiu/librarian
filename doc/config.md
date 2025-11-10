# New Configuration System

This document describes the new configuration system for Librarian, which uses
`.librarian.yaml` files instead of flags for all configuration.

## Overview

The new configuration system has two levels:

1. **Repository configuration** (`.librarian/config.yaml`) - Defines repository-wide
   settings like language, container images, and googleapis references
2. **Artifact configuration** (`<artifact>/.librarian.yaml`) - Defines artifact-specific
   settings like APIs to generate, metadata, and file filtering rules

This design eliminates the need for command-line flags and makes all configuration
transparent and version-controlled.

## Repository Configuration

The repository configuration file lives at `.librarian/config.yaml` and defines
repository-wide settings.

### Example: Release-only repository

```yaml
librarian:
  version: v0.5.0

release:
  tag_format: '{name}-v{version}'
```

**What this enables:**
- `librarian add <path>` - Track handwritten code for release
- `librarian release prepare` - Prepare releases
- `librarian release tag` - Create git tags
- `librarian release publish` - Publish to registries

### Example: Repository with code generation

```yaml
librarian:
  version: v0.5.0
  language: python

generate:
  container:
    image: us-central1-docker.pkg.dev/cloud-sdk-librarian-prod/images-prod/python-librarian-generator
    tag: latest
  googleapis:
    path: https://github.com/googleapis/googleapis/archive/a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0.tar.gz
    sha256: 81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98
  discovery:
    path: https://github.com/googleapis/discovery-artifact-manager/archive/f9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0.tar.gz
    sha256: 867048ec8f0850a4d77ad836319e4c0a0c624928611af8a900cd77e676164e8e
  dir: packages/

release:
  tag_format: '{name}-v{version}'
```

**What this enables:**
- `librarian add <path> <api>` - Generate code from API definitions
- `librarian generate <path>` - Regenerate code
- `librarian release prepare` - Prepare releases
- `librarian release tag` - Create git tags
- `librarian release publish` - Publish to registries

### Configuration Fields

#### `librarian` section

- `version` - Version of librarian that created this config
- `language` - Repository language (`go`, `python`, `rust`, `dart`)

#### `generate` section (optional)

When present, enables code generation commands. This section defines the generation infrastructure (container images, googleapis location, etc.).

- `container.image` - Container registry path (without tag)
- `container.tag` - Container image tag (e.g., `latest`, `v1.0.0`)
- `googleapis.path` - Local directory path OR tarball URL (e.g., `/Users/name/googleapis` or `https://github.com/googleapis/googleapis/archive/{commit}.tar.gz`)
- `googleapis.sha256` - SHA256 hash for integrity verification (required when `path` is a URL, ignored for local directories)
- `discovery.path` - Local directory path OR tarball URL for discovery-artifact-manager
- `discovery.sha256` - SHA256 hash (required when `path` is a URL)
- `dir` - Directory where generated code is written (relative to repository root,
  with trailing `/`)

**Design rationale**: The `path` field supports both local directories and tarball URLs:
- **Local development** - Point to your local clone (e.g., `/Users/name/googleapis`) to test changes without downloading
- **Production/CI** - Use immutable tarballs with SHA256 verification for reproducibility
- **Caching** - Downloads are cached by SHA256 in `~/Library/Caches/librarian/downloads/` to avoid repeated downloads
- **No race conditions** - Multiple concurrent generations verify the same immutable tarball
- **Single source of truth** - All artifacts use the same googleapis version from the repository config

#### `release` section (optional)

When present, enables release commands.

- `tag_format` - Template for git tags (e.g., `'{name}-v{version}'` or `'{id}/v{version}'`)
  - Supported placeholders: `{id}`, `{name}`, and `{version}`
  - The global default is `'{id}/v{version}'` for Go repositories
  - Some modules require custom formats to avoid double version paths. These exceptions should be handled in code:
    - `bigquery/v2` uses `bigquery/v{version}` (instead of `bigquery/v2/v{version}`)
    - `pubsub/v2` uses `pubsub/v{version}` (instead of `pubsub/v2/v{version}`)
    - `root-module` uses `v{version}` (no id prefix)

## Artifact Configuration

Each artifact has its own `.librarian.yaml` file that defines artifact-specific settings.

### Example: Handwritten code (release-only)

```yaml
release:
  version: null
```

This artifact only has a `release` section, so it can be released but not regenerated.

### Example: Generated code

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
  metadata:
    name_pretty: "Secret Manager"
    product_documentation: "https://cloud.google.com/secret-manager/docs"
    release_level: "stable"
    api_description: "Store and manage secrets"
  language:
    python:
      package: google-cloud-secret-manager
  keep:
    - README.md
    - docs/
  remove:
    - temp.txt
  exclude:
    - tests/

release:
  version: null
```

### Artifact Configuration Fields

#### `generate` section (optional)

When present, this artifact can be regenerated with `librarian generate`.

**API Configuration** (`apis` array):

Each API entry contains configuration extracted from BUILD.bazel during `librarian add`:

- `path` - API path relative to googleapis root (e.g., `secretmanager/v1`)
- `grpc_service_config` - Retry configuration file path (relative to API directory)
- `service_yaml` - Service configuration file path
- `transport` - Transport protocol (e.g., `grpc+rest`, `grpc`)
- `rest_numeric_enums` - Whether to use numeric enums in REST
- `opt_args` - Additional generator options (array of strings)

**Metadata** (`metadata` object):

Library metadata used to generate documentation and configure the package:

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

**Language-specific metadata** (`language` object):

Language-specific configuration that matches the repository's language:

```yaml
# For Go repositories
language:
  go:
    module: github.com/user/repo

# For Python repositories
language:
  python:
    package: my-package

# For Rust repositories
language:
  rust:
    crate: my_crate

# For Dart repositories
language:
  dart:
    package: my_package
```

**File filtering**:

- `keep` - Files/directories not overwritten during generation (array of regex patterns)
- `remove` - Files/directories deleted after generation (array of regex patterns)
- `exclude` - Files/directories not included in releases (array of regex patterns)

**Note**: The artifact's `.librarian.yaml` does NOT store googleapis URL/SHA256 or generation history. These are stored only in the repository-level `.librarian.yaml` to ensure all artifacts use the same googleapis version. This design:
- Prevents duplication across hundreds of artifact configs
- Ensures consistency - all artifacts generated from the same googleapis version
- Prevents race conditions - no per-artifact googleapis state to get out of sync
- Simplifies updates - change googleapis version in one place, regenerate all artifacts

#### `release` section (optional)

When present, this artifact can be released with `librarian release prepare`, `librarian release tag`, and `librarian release publish`.

- `version` - Current released version (null if never released)
- `prepared.version` - Next version being prepared (present only when a release is prepared)
- `prepared.commit` - Commit SHA at which the release was prepared

## How Configuration Works

### Adding an artifact without APIs (release-only)

```bash
librarian add packages/my-tool
```

This creates `packages/my-tool/.librarian.yaml`:

```yaml
release:
  version: null
```

The artifact can be released but not regenerated.

### Adding an artifact with APIs (generated code)

```bash
librarian add packages/google-cloud-secret-manager secretmanager/v1 secretmanager/v1beta2
```

This:

1. Reads `.librarian/config.yaml` to get googleapis location
2. Clones googleapis repository if needed
3. For each API path (`secretmanager/v1`, `secretmanager/v1beta2`):
   - Reads `BUILD.bazel` file in that directory
   - Extracts configuration from language-specific gapic rule (e.g., `py_gapic_library`)
   - Saves configuration to `.librarian.yaml`
4. Copies container, googleapis, and discovery settings from repository config
5. Creates `packages/google-cloud-secret-manager/.librarian.yaml` with all extracted config

**Key insight**: BUILD.bazel parsing happens only once during `librarian add`. The
extracted configuration is saved to `.librarian.yaml` and reused for all subsequent
`librarian generate` commands. This makes generation faster and ensures reproducibility
even if BUILD.bazel files change upstream.

### Generating code

```bash
librarian generate packages/google-cloud-secret-manager
```

This:

1. Reads `.librarian/config.yaml` (repository config with `infrastructure` section)
2. Reads `packages/google-cloud-secret-manager/.librarian.yaml` (artifact config with `generate` section)
3. Ensures googleapis is available:
   - If `googleapis.path` is a local directory → uses it directly
   - If `googleapis.path` is a URL → downloads tarball, verifies SHA256, caches by SHA256
4. Builds `generate.json` from API configurations in `.librarian.yaml`
5. Runs generator container **once** with `generate.json`
6. Applies keep/remove/exclude rules after container exits
7. Copies final output to artifact directory

**No generation state** is written to the artifact's `.librarian.yaml`. The repository config already contains the googleapis path/SHA256, which serves as the single source of truth for what was used.

## Container Interface

The generator container is a Docker image that implements the Librarian container
contract using a **command-based architecture**.

### What the container receives

**Mounts:**

- `/commands/commands.json` - Contains commands to execute (read-only)
- `/source` - Read-only googleapis repository
- `/output` - Directory where container writes generated code

**Container execution:**

The librarian CLI invokes the container multiple times during generation, each time
with a different commands.json file. The container reads the commands and executes
them sequentially.

**Example invocation:**

```bash
docker run \
  -v /path/to/commands.json:/commands/commands.json:ro \
  -v /path/to/googleapis:/source:ro \
  -v /path/to/output:/output \
  python-generator:latest
```

### commands.json

The container reads `/commands/commands.json` which contains explicit commands to execute.

**Example (Python code generation):**

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

**Example (Go code generation):**

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

### Container responsibilities

The container must:

1. Read `/commands/commands.json`
2. Execute each command sequentially
3. Exit when all commands complete

The container does NOT:

- Parse BUILD.bazel files (already done by librarian CLI)
- Clone googleapis (already mounted at `/source`)
- Parse `.librarian.yaml` files (already done by librarian CLI - commands are pre-built)
- Apply keep/remove/exclude rules (done by librarian CLI after container exits)
- Update `.librarian.yaml` (done by librarian CLI after container exits)

### Multiple invocations

The host CLI calls the container multiple times during generation, each with different commands:

1. **Code generation** - Run protoc/generators
2. **Post-processing** - Run formatters, templates
3. **Testing** - Run tests and validation

Between invocations, the host applies file filtering rules and manages staging directories.

See [doc/generate.md](generate.md) for detailed generation flows for Python, Go, and Rust.

## Key Design Decisions

### Why parse BUILD.bazel in the CLI?

- Parsing happens once during `librarian add`, not on every generation
- Configuration is saved in `.librarian.yaml` for transparency and reproducibility
- Users can manually edit configuration if BUILD.bazel is incorrect
- Container remains simple - just executes protoc with provided options
- Go has excellent Bazel parsing libraries (`github.com/bazelbuild/buildtools/build`)

### Why NOT copy container/googleapis/discovery settings to artifact config?

The new design stores googleapis URL/SHA256 **only in the repository config**, not in each artifact's config. This provides:

- **Single source of truth** - One place to update googleapis version for all artifacts
- **No duplication** - Don't repeat the same URL/SHA256 across hundreds of artifact configs
- **Prevents race conditions** - No mutable shared cache or per-artifact googleapis state
- **Consistent generation** - All artifacts always use the same googleapis version
- **Simpler updates** - Change googleapis in one file, regenerate all artifacts
- **Git history still works** - `git log .librarian.yaml` shows when googleapis changed for the entire repository

This follows the same pattern as sidekick (see `.sidekick.toml` in google-cloud-rust), where the root config contains `googleapis-root` and `googleapis-sha256`, and per-library configs contain only API-specific settings.

### Why use regex patterns for keep/remove/exclude?

Regex patterns provide:

- **Flexibility** - Match patterns like `.*_test\.py` or `docs/.*\.md`
- **Precision** - Exact control over what files are affected
- **Simplicity** - Single pattern can match many files
- **Familiarity** - Developers understand regex

### Why use a command-based architecture?

The command-based architecture provides:

- **Simplicity** - Container just executes commands, no parsing/interpretation needed
- **Language-agnostic container code** - Same Go code runs in all language containers
- **Explicit control** - Host decides exactly what commands to run
- **Debuggability** - Commands are transparent and can be inspected
- **Flexibility** - Easy to add new commands or change execution order

## Migration from Old System

The old system used flags and `.repo-metadata.json` files. The new system uses
`.librarian.yaml` files for all configuration.

**Old system:**

```bash
# Flags required for every command
librarian generate \
  --api secretmanager/v1 \
  --api-source=/path/to/googleapis \
  --library secretmanager \
  --image python-gen:latest
```

**New system:**

```bash
# All configuration in .librarian.yaml files
librarian generate packages/google-cloud-secret-manager
```

**Migration steps:**

1. Run `librarian init <language>` to create `.librarian/config.yaml`
2. For each existing library:
   - Run `librarian add <path> <apis>` to create `.librarian.yaml`
   - Verify configuration matches old `.repo-metadata.json`
3. Delete old `.repo-metadata.json` files
4. Update CI/CD pipelines to use new commands without flags

## Completed Improvements

### ✅ Adopted command-based container architecture

The container interface uses a command-based architecture:
- `/commands/commands.json` contains **explicit commands to execute**
- Container is a simple command executor (language-agnostic)
- Multiple container invocations per library (generate → format → test)
- Librarian team maintains container (simple Go code)
- Language teams maintain generator tools (gapic-generator-python, gapic-generator-go, Sidekick)

**Key benefits:**
- **Simplicity**: Container is ~30 lines of Go, no language expertise needed
- **Explicit**: Commands are visible in commands.json, easy to debug
- **Ownership**: Librarian team owns container, language teams own generators
- **Flexibility**: Easy to add/remove/reorder commands without changing container

### ✅ Consistent configuration naming

Both repository-level and artifact-level configuration use `generate`:

**Repository config** (`.librarian/config.yaml`):
```yaml
generate:  # How to generate (container, googleapis)
  container:
    image: python-generator
  googleapis:
    path: https://...
```

**Artifact config** (`<path>/.librarian.yaml`):
```yaml
generate:  # What to generate (APIs, metadata)
  apis:
    - path: secretmanager/v1
```

**Benefit**: Consistent naming throughout the configuration hierarchy.
