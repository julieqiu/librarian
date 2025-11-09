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
- `librarian prepare <path>` - Prepare releases
- `librarian release <path>` - Publish releases

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

### Configuration Fields

#### `librarian` section

- `version` - Version of librarian that created this config
- `language` - Repository language (`go`, `python`, `rust`, `dart`)

#### `generate` section (optional)

When present, enables code generation commands.

- `container.image` - Container registry path (without tag)
- `container.tag` - Container image tag (e.g., `latest`, `v1.0.0`)
- `googleapis.repo` - Repository location for googleapis (GitHub path or local directory
  relative to `.librarian/`)
- `googleapis.ref` - Git reference (commit SHA, branch name, or tag). Optional; if
  omitted, uses HEAD of default branch
- `discovery.repo` - Repository location for discovery-artifact-manager
- `discovery.ref` - Git reference. Optional; if omitted, uses HEAD of default branch
- `dir` - Directory where generated code is written (relative to repository root,
  with trailing `/`)

#### `release` section (optional)

When present, enables release commands.

- `tag_format` - Template for git tags (e.g., `'{name}-v{version}'`)

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

**Generation state** (populated by `librarian generate`):

- `commit` - Googleapis commit SHA used for generation
- `librarian` - Librarian version used for generation
- `container.image` - Container image used (copied from repository config)
- `container.tag` - Container tag used (copied from repository config)
- `googleapis.repo` - Googleapis repo location (copied from repository config)
- `googleapis.ref` - Googleapis ref used (copied from repository config)
- `discovery.repo` - Discovery repo location (copied from repository config)
- `discovery.ref` - Discovery ref used (copied from repository config)

#### `release` section (optional)

When present, this artifact can be released with `librarian prepare` and `librarian release`.

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

1. Reads `.librarian/config.yaml` (repository config)
2. Reads `packages/google-cloud-secret-manager/.librarian.yaml` (artifact config)
3. Clones googleapis at the specified ref (or updates existing clone)
4. Prepares generate-request.json with API configurations from `.librarian.yaml`
5. Runs generator container with appropriate mounts
6. Applies keep/remove/exclude rules to the output
7. Updates `.librarian.yaml` with generation metadata (commit, librarian version)

## Container Interface

### What the container receives

The generator container is a Docker image that implements the Librarian container
contract. The container receives:

**Mounts:**

- `/request` - Contains generate-request.json (derived from the artifact's `.librarian.yaml`)
- `/output` - Empty directory where container writes generated code
- `/source` - Read-only googleapis repository at specified commit

**Command arguments:**

```bash
docker run \
  -v /path/to/artifact/.librarian:/request \
  -v /tmp/output:/output \
  -v /path/to/googleapis:/source:ro \
  python-generator:latest \
  generate \
  --request=/request \
  --output=/output \
  --source=/source
```

Note: `/path/to/artifact/.librarian` is the `.librarian` directory inside the specific artifact being generated (e.g., `packages/google-cloud-secret-manager/.librarian`), not the repository-level `.librarian` directory.

### generate-request.json

The container reads `/request/generate-request.json` which contains the library
configuration.

**Example (Python):**

```json
{
  "id": "packages/google-cloud-secret-manager",
  "version": "1.0.0",
  "apis": [
    {
      "path": "google/cloud/secretmanager/v1",
      "service_config": "secretmanager_grpc_service_config.json"
    },
    {
      "path": "google/cloud/secretmanager/v1beta2",
      "service_config": "secretmanager_grpc_service_config.json"
    }
  ],
  "metadata": {
    "name_pretty": "Secret Manager",
    "product_documentation": "https://cloud.google.com/secret-manager/docs",
    "release_level": "stable",
    "api_description": "Store and manage secrets"
  },
  "language": {
    "python": {
      "package": "google-cloud-secret-manager"
    }
  }
}
```

**Example (Go):**

```json
{
  "id": "packages/secretmanager",
  "version": "1.0.0",
  "apis": [
    {
      "path": "google/cloud/secretmanager/v1",
      "service_config": "secretmanager_grpc_service_config.json"
    }
  ],
  "source_roots": ["packages/secretmanager"],
  "preserve_regex": ["README\\.md", "docs/"],
  "remove_regex": ["temp\\.txt"],
  "metadata": {
    "name_pretty": "Secret Manager",
    "product_documentation": "https://cloud.google.com/secret-manager/docs",
    "release_level": "stable"
  },
  "language": {
    "go": {
      "module": "cloud.google.com/go/secretmanager"
    }
  }
}
```

### Container responsibilities

The container must:

1. Read `/request/generate-request.json`
2. Parse API configurations
3. Execute protoc with language-specific plugins using provided configurations
4. Run post-processors (formatting, templates, etc.)
5. Run validation and tests
6. Write generated code to `/output`

The container does NOT:

- Parse BUILD.bazel files (already done by librarian CLI)
- Clone googleapis (already mounted at `/source`)
- Read from `/input` (no longer needed - all data is in generate-request.json)
- Apply keep/remove/exclude rules (done by librarian CLI after container exits)
- Update `.librarian.yaml` (done by librarian CLI after container exits)

## Key Design Decisions

### Why parse BUILD.bazel in the CLI?

- Parsing happens once during `librarian add`, not on every generation
- Configuration is saved in `.librarian.yaml` for transparency and reproducibility
- Users can manually edit configuration if BUILD.bazel is incorrect
- Container remains simple - just executes protoc with provided options
- Go has excellent Bazel parsing libraries (`github.com/bazelbuild/buildtools/build`)

### Why copy container/googleapis/discovery settings to artifact config?

This provides **explicit versioning** - each artifact records exactly which container
and googleapis version was used for generation. This enables:

- **Reproducibility** - Can regenerate with exact same inputs
- **Transparency** - Users can see what was used by reading `.librarian.yaml`
- **Per-artifact control** - Different artifacts can use different container versions
- **History** - Git log shows when container/googleapis versions changed

Without this, we'd need to look at git history to know what was in the repository
config at generation time.

### Why use regex patterns for keep/remove/exclude?

Regex patterns provide:

- **Flexibility** - Match patterns like `.*_test\.py` or `docs/.*\.md`
- **Precision** - Exact control over what files are affected
- **Simplicity** - Single pattern can match many files
- **Familiarity** - Developers understand regex

### Why mount /request instead of /librarian?

The mount contains the request data (`generate-request.json`) that tells the container
what to generate. The name `/request` clearly indicates this is the input request for
the generation operation, and avoids confusion with the CLI tool name.

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

### ✅ Eliminated /input mount

The `/input` mount has been removed. Previously, it was mounted to `.librarian/generator-input/`
which contained files like `repo-config.yaml` (Go) or `.repo-metadata.json` (Python).
All necessary data is now in `generate-request.json`, making the container interface simpler.

### ✅ Renamed ApiRoot to GoogleapisDir

The current field name `ApiRoot` in the Go code is confusing - it sounds like the
root of a single API, but it's actually the root of the googleapis repository.
Renaming to `GoogleapisDir` would be clearer:

```go
// Before
type GenerateRequest struct {
    ApiRoot string  // Confusing - root of what?
}

// After
type GenerateRequest struct {
    GoogleapisDir string  // Clear - googleapis directory
}
```
