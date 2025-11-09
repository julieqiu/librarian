# Library Configuration Design

This document describes how `librarian add` and `librarian generate` work together to configure new libraries and generate scaffolding and code.

## Overview

The workflow is split into two commands:

1. **`librarian add`** - Extracts metadata from BUILD.bazel and creates `.librarian.yaml` entry
2. **`librarian generate`** - Generates scaffolding (first time only) + calls container for code generation

This design keeps container logic minimal - containers only generate code, while all configuration and scaffolding logic lives in librarian itself.

## Workflow

```bash
# Step 1: User adds a new library
librarian add google/cloud/storage/v1

# Creates:
# - .librarian.yaml entry with metadata
# - Commits config only

# Step 2: User generates code
librarian generate {library-name}

# First time:
# - Detects library directory doesn't exist
# - Generates language-specific scaffolding files
# - Calls container to generate client code
# - Commits scaffolding + generated code

# Subsequent times:
# - Calls container to regenerate client code only
# - Commits generated code
```

## Design Principles

1. **Minimal container logic** - Containers should only generate code from protos, nothing more
2. **All scaffolding in librarian** - Makes it easy to refactor without rebuilding containers
3. **Language-specific isolation** - Each language's logic is isolated in its own package
4. **Metadata extraction** - Read BUILD.bazel once, extract all needed metadata
5. **No configure containers** - Eliminates the need for separate configure step
6. **Separation of concerns** - `add` creates config, `generate` creates files

## Metadata Extraction

### Source: BUILD.bazel Files

Each language has a specific Bazel rule that contains metadata:

**Go:** `go_gapic_library`
```python
go_gapic_library(
    name = "secretmanager_go_gapic",
    grpc_service_config = "secretmanager_grpc_service_config.json",
    importpath = "cloud.google.com/go/secretmanager/apiv1;secretmanager",
    release_level = "ga",
    rest_numeric_enums = True,
    service_yaml = "secretmanager_v1.yaml",
    transport = "grpc+rest",
)
```

**Python:** `py_gapic_library`
```python
py_gapic_library(
    name = "secretmanager_py_gapic",
    grpc_service_config = "secretmanager_grpc_service_config.json",
    rest_numeric_enums = True,
    service_yaml = "secretmanager_v1.yaml",
    transport = "grpc+rest",
    opt_args = [
        "warehouse-package-name=google-cloud-secret-manager",
    ],
)
```

**Rust:** No BUILD.bazel
- Rust does not use BUILD.bazel
- Only needs to know API path and service config filename
- Metadata is derived from service YAML and sidekick conventions

### Source: Service YAML Files

Service YAML files provide human-readable metadata:

```yaml
type: google.api.Service
config_version: 3
name: secretmanager.googleapis.com
title: Secret Manager API
documentation:
  summary: |-
    Stores sensitive data such as API keys, passwords, and certificates.
```

## Scaffolding Files by Language

### Go

**Files created by `librarian generate` (first time only):**

1. **`{library}/README.md`** - Generated from template
   ```markdown
   # Secret Manager API

   [Product Documentation](https://cloud.google.com/secret-manager)

   ## Installation

   ```bash
   go get cloud.google.com/go/secretmanager
   ```
   ```

2. **`{library}/CHANGES.md`** - Empty changelog
   ```markdown
   # Changes
   ```

3. **`{library}/internal/version.go`** - Module-level version
   ```go
   package internal

   // Version is the current version of the secretmanager client library.
   const Version = "0.0.0"
   ```

4. **`{library}/{clientDir}/version.go`** - API-level version
   ```go
   package apiv1

   import "cloud.google.com/go/secretmanager/internal"

   // version is the version of this client library.
   var version = internal.Version
   ```

5. **`internal/generated/snippets/go.mod`** - Updated with replace directive
   ```
   replace cloud.google.com/go/secretmanager => ../../../secretmanager
   ```

**Why these files?**
- README provides initial documentation
- CHANGES.md is updated during releases
- version.go files track library version
- go.mod replace allows snippets to import the local module

---

### Python

**Files created by `librarian generate` (first time only):**

1. **`packages/{library}/CHANGELOG.md`** - Package changelog
   ```markdown
   # Changelog

   [PyPI History][1]

   [1]: https://pypi.org/project/google-cloud-secret-manager/#history
   ```

2. **`packages/{library}/docs/CHANGELOG.md`** - Duplicate for docs
   ```markdown
   # Changelog

   [PyPI History][1]

   [1]: https://pypi.org/project/google-cloud-secret-manager/#history
   ```

3. **`CHANGELOG.md`** - Global changelog (updated, not created)
   - Adds entry for new library in alphabetical order

**Why these files?**
- CHANGELOG.md is required by Python release process
- Duplicate in docs/ for documentation generation
- Global CHANGELOG tracks all libraries in monorepo

---

### Rust

**Files created by `librarian generate` (first time only):**

All files created by `sidekick generate` - complete Rust crate including README, Cargo.toml, all source files, and .sidekick.toml.

**Why no scaffolding?**
- `sidekick generate` creates a complete Rust crate in one step
- Includes README, Cargo.toml, all source files, and .sidekick.toml
- Simpler to let sidekick handle everything than duplicate logic

## Configuration File Structure

### `.librarian.yaml` Entry

Each library gets an entry in `.librarian.yaml`:

**Go Example:**
```yaml
name: secretmanager
version: 0.0.0
generate:
    specification_format: protobuf
    apis:
        - path: google/cloud/secretmanager/v1
          service_config: secretmanager_v1.yaml
          grpc_service_config: secretmanager_grpc_service_config.json
          rest_numeric_enums: true
          transport: grpc+rest
          importpath: cloud.google.com/go/secretmanager/apiv1;secretmanager
          release_level: ga
go:
    source_roots:
        - secretmanager
        - internal/generated/snippets/secretmanager
    tag_format: '{id}/v{version}'
    remove_regex:
        - ^internal/generated/snippets/secretmanager/
        - ^secretmanager/apiv1/[^/]*_client\.go$
        - ^secretmanager/apiv1/[^/]*_client_example_go123_test\.go$
        - ^secretmanager/apiv1/[^/]*_client_example_test\.go$
        - ^secretmanager/apiv1/auxiliary\.go$
        - ^secretmanager/apiv1/auxiliary_go123\.go$
        - ^secretmanager/apiv1/doc\.go$
        - ^secretmanager/apiv1/gapic_metadata\.json$
        - ^secretmanager/apiv1/helpers\.go$
        - ^secretmanager/apiv1/secretmanagerpb/.*$
        - ^secretmanager/apiv1/\.repo-metadata\.json$
```

**Python Example:**
```yaml
name: google-cloud-secret-manager
version: 0.0.0
generate:
    specification_format: protobuf
    apis:
        - path: google/cloud/secretmanager/v1
          service_config: secretmanager_v1.yaml
          grpc_service_config: secretmanager_grpc_service_config.json
          rest_numeric_enums: true
          transport: grpc+rest
          opt_args:
              - warehouse-package-name=google-cloud-secret-manager
python:
    remove:
        - packages/google-cloud-secret-manager/google/cloud/secretmanager.py
```

**Rust Example:**
```yaml
name: cloud-storage-v1
version: 0.0.0
generate:
    specification_format: protobuf
    apis:
        - path: google/cloud/storage/v1
          service_config: storage_v1.yaml
rust:
    codec:
        copyright_year: "2025"
```

## Library Name Derivation

Library names are derived from API paths using language-specific conventions:

### Go
```
google/cloud/secretmanager/v1 → secretmanager
google/bigtable/admin/v2 → admin
```
Rules: Use the second-to-last path component (before version)

### Python
```
google/cloud/secretmanager/v1 → google-cloud-secret-manager
google/bigtable/admin/v2 → google-bigtable-admin
```
Rules:
1. Check opt_args for `warehouse-package-name` (highest priority)
2. If path starts with `google/cloud/`: `google-cloud-{service}`
3. Otherwise: `google-{service}`

### Rust
```
google/cloud/storage/v1 → cloud-storage-v1
google/bigtable/admin/v2 → bigtable-admin-v2
```
Rules: Join all path components after "google" with hyphens

## Client Directory Derivation

Go needs to determine the client directory (e.g., `apiv1`, `apiv1beta1`):

```
google/cloud/secretmanager/v1 → apiv1
google/cloud/secretmanager/v1beta1 → apiv1beta1
google/cloud/secretmanager/v2 → apiv2
```

Rules: `api` + version (last path component)

## Remove Regex Pattern Generation

Go's `remove_regex` patterns follow a consistent structure based on the library name and API path.

**Template:**
```
^internal/generated/snippets/{library}/
^{library}/{clientDir}/[^/]*_client\.go$
^{library}/{clientDir}/[^/]*_client_example_go123_test\.go$
^{library}/{clientDir}/[^/]*_client_example_test\.go$
^{library}/{clientDir}/auxiliary\.go$
^{library}/{clientDir}/auxiliary_go123\.go$
^{library}/{clientDir}/doc\.go$
^{library}/{clientDir}/gapic_metadata\.json$
^{library}/{clientDir}/helpers\.go$
^{library}/{clientDir}/{protoPackage}/.*$
^{library}/{clientDir}/\.repo-metadata\.json$
```

**Variables:**
- `{library}` - Library name (e.g., `secretmanager`)
- `{clientDir}` - Client directory (e.g., `apiv1`)
- `{protoPackage}` - Proto package name (e.g., `secretmanagerpb`)

**Proto package derivation:**
```
google/cloud/secretmanager/v1 → secretmanagerpb
google/bigtable/admin/v2 → adminpb
```
Rules: Service name + `pb`

## Implementation Structure

```
internal/librarian/
├── add/
│   ├── add.go           # Main add logic (config only)
│   ├── metadata.go      # BUILD.bazel parsing
│   └── config.go        # .librarian.yaml creation
└── generate/
    ├── generate.go      # Main generate logic
    ├── scaffolding.go   # Scaffolding generation (first-time detection)
    ├── go.go            # Go scaffolding
    ├── python.go        # Python scaffolding
    └── rust.go          # Rust (no scaffolding, just container call)
```

## Error Handling

`librarian add` should fail fast with clear errors:

1. **API path not found** - `google/cloud/invalid/v1` doesn't exist
2. **BUILD.bazel missing** - Required for Go and Python
3. **Service YAML missing** - Required for metadata extraction
4. **Parse failures** - Invalid BUILD.bazel syntax
5. **Invalid metadata** - Missing required fields

## Future Enhancements

1. **Support adding API to existing library** - Currently only supports new libraries
2. **Validate generated files** - Ensure templates render correctly
3. **Custom templates** - Allow users to override default templates
4. **Dry-run mode** - Show what would be created without creating files
5. **Interactive mode** - Prompt for missing metadata

## Testing Strategy

1. **Unit tests** - Test metadata extraction for each language
2. **Integration tests** - Test full add workflow with real googleapis data
3. **Golden files** - Compare generated scaffolding against expected output
4. **Testdata** - Use internal/container/go/testdata structure

## Migration from Configure Containers

This design eliminates the need for configure containers:

**Before:**
```
librarian add → call configure container → merge response → commit config + scaffolding
```

**After:**
```
librarian add → save config → commit config only
librarian generate → generate scaffolding (first time) → call container → commit all
```

**Benefits:**
1. Simpler - No configure container needed
2. Faster - No extra container call for configuration
3. More maintainable - All logic in one place (Go code)
4. Easier to test - Pure functions, no Docker required
5. Clear separation - `add` = config, `generate` = files
