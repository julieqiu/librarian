# Container Configuration Specification

This document describes the configuration specifications for each language container and the data extracted from BUILD.bazel files during testdata population.

## Overview

Each language container processes `.librarian.yaml` files to generate client libraries. The configuration is divided into:

1. **General section** - Language-agnostic API and metadata configuration
2. **Language-specific section** - Language-specific build and filtering rules

## General Configuration (All Languages)

### `generate.apis` Array

Each API entry contains configuration extracted from BUILD.bazel files:

- `path` - API path relative to googleapis root (e.g., `google/cloud/secretmanager/v1`)
- `service_config` - Service configuration YAML file (e.g., `secretmanager_v1.yaml`)
- `grpc_service_config` - gRPC retry configuration JSON file (e.g., `secretmanager_grpc_service_config.json`)
- `rest_numeric_enums` - Whether to use numeric enums in REST (boolean)
- `transport` - Transport protocol (`grpc`, `rest`, or `grpc+rest`)

### Additional Fields by Language

Different `*_gapic_library` rules in BUILD.bazel provide different fields:

**Go (`go_gapic_library`):**
- `importpath` - Go import path (e.g., `cloud.google.com/go/secretmanager/apiv1;secretmanager`)
- `release_level` - Release level (`ga`, `beta`, `alpha`)

**Python (`py_gapic_library`):**
- `opt_args` - Additional generator options (array of strings, e.g., `["autogen-snippets=False"]`)

## Python Configuration

### BUILD.bazel Data

**Statistics:**
- Total Python libraries: 231
- Libraries with BUILD.bazel data: 21 GAPIC libraries
- Libraries without BUILD.bazel data: 210 (proto-only or handwritten libraries)

### Example Configuration

```yaml
name: google-cloud-secret-manager
version: 1.0.0
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
        - path: google/cloud/secretmanager/v1beta2
          service_config: secretmanager_v1beta2.yaml
          grpc_service_config: secretmanager_grpc_service_config.json
          rest_numeric_enums: true
          transport: grpc+rest
          opt_args:
              - warehouse-package-name=google-cloud-secret-manager
python:
    remove:
        - packages/google-cloud-secret-manager/google/cloud/secretmanager.py
        - packages/google-cloud-secret-manager/google/cloud/secretmanager_v1/__init__.py
```

### Python-Specific Fields

- `python.remove` - Files to delete after generation (explicit file list, no regex)

### Regex Expansion

**Original state:** 4 libraries used regex patterns in `python.remove`
**Expanded state:** All regex patterns converted to explicit file lists

- `google-cloud-audit-log`: 4 files
- `google-cloud-access-context-manager`: 12 files
- `googleapis-common-protos`: 64 files
- `grpc-google-iam-v1`: 9 files

**Total:** 89 files matched by regex patterns, all converted to explicit lists

**Rationale:** Regex support removed to simplify the container implementation and make file filtering transparent.

## Go Configuration

### BUILD.bazel Data

**Statistics:**
- Total Go libraries: 183
- Libraries with BUILD.bazel data: 172
- Libraries without BUILD.bazel data: 11 (have `disable_gapic: true` or no `go_gapic_library` rule)

### Example Configuration

```yaml
name: secretmanager
version: 1.0.0
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
          client_directory: secretmanager
          disable_gapic: false
          nested_protos:
              - google/iam/v1
          proto_package: cloud.google.com/go/secretmanager/apiv1/secretmanagerpb
go:
    source_roots:
        - secretmanager
        - internal/generated/snippets/secretmanager
    keep:
        - secretmanager/apiv1/iam_policy_client.go
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
    release_exclude_paths:
        - internal/generated/snippets/secretmanager/
    tag_format: '{id}/v{version}'
    module_path_version: /v2
    delete_generation_output_paths:
        - internal/generated/snippets/secretmanager/
```

### Go-Specific Fields

**From state.yaml:**
- `go.source_roots` - Directories containing source code (array)
- `go.release_exclude_paths` - Paths excluded from releases (array)
- `go.tag_format` - Git tag format template

**From generator-input/repo-config.yaml:**
- `generate.apis[].client_directory` - Client directory name
- `generate.apis[].disable_gapic` - Whether GAPIC generation is disabled (boolean)
- `generate.apis[].nested_protos` - Nested proto packages (array)
- `generate.apis[].proto_package` - Go proto package import path
- `go.module_path_version` - Module path version suffix (e.g., `/v2`)
- `go.delete_generation_output_paths` - Paths to delete before generation (array)

**File filtering:**
- `go.keep` - Files to preserve during generation (explicit file list)
- `go.remove_regex` - Files to remove after generation (regex patterns)

### Regex Usage Statistics

**preserve_regex (now renamed to `keep`):**
- Total libraries using preserve_regex: 6
- All 6 converted from regex patterns to explicit file lists

Libraries with `keep` files:
1. `batch`: 1 file (`batch/apiv1/iam_policy_client.go`)
2. `bigquery`: 39 files (snippet files in `internal/generated/snippets/bigquery/v2/`)
3. `datacatalog`: 1 file (`datacatalog/apiv1/iam_policy_client.go`)
4. `datastream`: 1 file (`datastream/apiv1/iam_policy_client.go`)
5. `run`: 1 file (`run/apiv2/locations_client.go`)
6. `vmmigration`: 1 file (`vmmigration/apiv1/iam_policy_client.go`)

**remove_regex:**
- Total libraries using remove_regex: 175
- Status: **Kept as regex patterns** (not expanded to explicit lists)

### Remove Regex Pattern Analysis

All 175 Go libraries follow a **highly consistent pattern** for `remove_regex`. Each API version has the same set of ~10 generated files to remove:

**Pattern structure for each API version:**
```
^{library}/{api_version}/[^/]*_client\.go$
^{library}/{api_version}/[^/]*_client_example_go123_test\.go$
^{library}/{api_version}/[^/]*_client_example_test\.go$
^{library}/{api_version}/auxiliary\.go$
^{library}/{api_version}/auxiliary_go123\.go$
^{library}/{api_version}/doc\.go$
^{library}/{api_version}/gapic_metadata\.json$
^{library}/{api_version}/helpers\.go$
^{library}/{api_version}/{proto_package}pb/.*$
^{library}/{api_version}/\.repo-metadata\.json$
```

Plus one global pattern per library:
```
^internal/generated/snippets/{library}/
```

**Example for vmmigration:**
```yaml
remove_regex:
    - ^internal/generated/snippets/vmmigration/
    - ^vmmigration/apiv1/[^/]*_client\.go$
    - ^vmmigration/apiv1/[^/]*_client_example_go123_test\.go$
    - ^vmmigration/apiv1/[^/]*_client_example_test\.go$
    - ^vmmigration/apiv1/auxiliary\.go$
    - ^vmmigration/apiv1/auxiliary_go123\.go$
    - ^vmmigration/apiv1/doc\.go$
    - ^vmmigration/apiv1/gapic_metadata\.json$
    - ^vmmigration/apiv1/helpers\.go$
    - ^vmmigration/apiv1/vmmigrationpb/.*$
    - ^vmmigration/apiv1/\.repo-metadata\.json$
```

**Example for run:**
```yaml
remove_regex:
    - ^internal/generated/snippets/run/
    - ^run/apiv2/[^/]*_client\.go$
    - ^run/apiv2/[^/]*_client_example_go123_test\.go$
    - ^run/apiv2/[^/]*_client_example_test\.go$
    - ^run/apiv2/auxiliary\.go$
    - ^run/apiv2/auxiliary_go123\.go$
    - ^run/apiv2/doc\.go$
    - ^run/apiv2/gapic_metadata\.json$
    - ^run/apiv2/helpers\.go$
    - ^run/apiv2/runpb/.*$
    - ^run/apiv2/\.repo-metadata\.json$
```

### Recommendation: Move remove_regex Logic to Generator

**Key insight:** The `remove_regex` patterns are entirely predictable based on:
- Library name (from `name:` field)
- API version (from `generate.apis[].path` - last segment)
- Proto package name (derived from path or `generate.apis[].proto_package`)

**Proposed change:** Instead of requiring users to specify these patterns in `.librarian.yaml`, the Go generator container should:

1. **Know which files it generates** - The generator knows it creates `*_client.go`, `doc.go`, `auxiliary.go`, etc.
2. **Automatically clean them up before regeneration** - Delete these files before running protoc
3. **Only require `keep` for exceptions** - Users only specify the small number of files to preserve (like `iam_policy_client.go`)

**Benefits:**
- Eliminates ~10 regex patterns × 175 libraries = 1,750+ configuration lines
- Reduces `.librarian.yaml` file size and complexity
- Makes the configuration more maintainable
- Prevents configuration errors (typos in regex patterns)
- Follows the principle: "The generator knows what it generates"

**Current state:** All 175 libraries have `remove_regex` patterns that follow this exact template, differing only in the library name, API version, and proto package name - all of which are already known to the generator.

## Rust Configuration

### Data Source

Rust configurations are converted from Sidekick `.sidekick.toml` files using `cmd/convert_sidekick/main.go`.

**Statistics:**
- Total Rust libraries: 200
- Libraries with service_config: 155
- Libraries without service_config: 45 (proto-only packages)
- Specification formats:
  - Protobuf: 165 libraries
  - Discovery: 1 library
  - OpenAPI: 1 library
  - Proto-only (no format specified): 33 libraries

### Example Configuration (Protobuf)

```yaml
name: bigtable-admin-v2
version: 1.1.0
generate:
    specification_format: protobuf
    apis:
        - path: google/bigtable/admin/v2
          service_config: google/bigtable/admin/v2/bigtableadmin_v2.yaml
rust:
    codec:
        copyright_year: "2025"
```

### Example Configuration (Discovery)

```yaml
name: cloud-compute-v1
version: 0.2.1
generate:
    specification_format: disco
    apis:
        - path: discoveries/compute.v1.json
          service_config: google/cloud/compute/v1/compute_v1.yaml
rust:
    source:
        roots: discovery,googleapis
    codec:
        copyright_year: "2025"
        package_name_override: google-cloud-compute-v1
        per_service_features: true
        disabled_rustdoc_warnings: bare_urls,broken_intra_doc_links,redundant_explicit_links
        disabled_clippy_warnings: doc_lazy_continuation
    discovery:
        operation_id: .google.cloud.compute.v1.Operation
        pollers:
            - prefix: compute/v1/projects/{project}/zones/{zone}
              method_id: .google.cloud.compute.v1.zoneOperations.get
            - prefix: compute/v1/projects/{project}/regions/{region}
              method_id: .google.cloud.compute.v1.regionOperations.get
            - prefix: compute/v1/projects/{project}
              method_id: .google.cloud.compute.v1.globalOperations.get
            - prefix: compute/v1/
              method_id: .google.cloud.compute.v1.globalOrganizationOperations.get
```

### Rust-Specific Fields

**Source configuration (`rust.source`):**
- `description_override` - Override API description
- `title_override` - Override API title (14 libraries use this)
- `roots` - Source roots (e.g., `discovery,googleapis`)
- `project_root` - Project root directory
- `include_list` - Files to include
- `included_ids` - IDs to include
- `skipped_ids` - IDs to skip

**Codec configuration (`rust.codec`):**
- `copyright_year` - Copyright year (all 200 libraries have this set to "2025")
- `package_name_override` - Override package name
- `name_overrides` - Map of name overrides
- `module_path` - Module path
- `root_name` - Root module name
- `template_override` - Template override path
- `not_for_publication` - Whether library should be published (boolean)
- `has_veneer` - Whether library has handwritten veneer code (boolean)
- `per_service_features` - Enable per-service Cargo features (boolean)
- `default_features` - Default Cargo features (array)
- `extra_modules` - Additional modules to include (array)
- `generate_setter_samples` - Generate setter code samples (boolean)
- `detailed_tracing_attributes` - Enable detailed tracing (boolean)
- `include_grpc_only_methods` - Include gRPC-only methods (boolean)
- `disabled_rustdoc_warnings` - Rustdoc warnings to disable (comma-separated string)
- `disabled_clippy_warnings` - Clippy warnings to disable (comma-separated string)
- `routing_required` - Whether routing is required (boolean)
- `post_process_protos` - Whether to post-process protos (boolean)
- `packages` - Package dependencies map

**Documentation overrides (`rust.documentation_overrides`):**
- `id` - Element ID to override
- `match` - Regex pattern to match
- `replace` - Replacement text

**Discovery configuration (`rust.discovery`):**
- `operation_id` - Operation resource ID
- `pollers` - Long-running operation pollers (array)
  - `prefix` - URL prefix to match
  - `method_id` - Method ID for polling

**Pagination overrides (`rust.pagination_overrides`):**
- `id` - Method ID
- `item_field` - Field containing items

### File Filtering

**No file filtering patterns:** Unlike Python and Go, Rust configurations do not use `keep`, `remove`, or regex patterns for file filtering. All 200 libraries have no file filtering configuration.

This suggests the Rust generator either:
1. Has built-in knowledge of which files to keep/remove
2. Generates a clean output directory without needing filtering
3. Handles file management differently than Python/Go containers

### Regex Usage Statistics

- Libraries using `keep`: 0
- Libraries using `remove`: 0
- Libraries using `keep_regex`: 0
- Libraries using `remove_regex`: 0

**Rationale:** The Rust container manages file generation internally without requiring explicit file filtering rules in the configuration.

## Dart Configuration

### Data Source

Dart configurations are converted from Sidekick `.sidekick.toml` files using `cmd/convert_dart/main.go`.

**Statistics:**
- Total Dart libraries: 13
- Libraries with service_config: 12
- Libraries without service_config: 1
- All libraries use protobuf specification format

### Example Configuration (Basic)

```yaml
name: google_cloud_secretmanager_v1
version: 0.0.0
generate:
    apis:
        - path: google/cloud/secretmanager/v1
          service_config: google/cloud/secretmanager/v1/secretmanager_v1.yaml
dart:
    codec:
        copyright_year: "2025"
        repository_url: https://github.com/googleapis/google-cloud-dart/tree/main/generated/google_cloud_secretmanager_v1
```

### Example Configuration (With README customization)

```yaml
name: google_cloud_ai_generativelanguage_v1beta
version: 0.0.0
generate:
    apis:
        - path: google/ai/generativelanguage/v1beta
          service_config: google/ai/generativelanguage/v1beta/generativelanguage_v1beta.yaml
dart:
    codec:
        copyright_year: "2025"
        repository_url: https://github.com/googleapis/google-cloud-dart/tree/main/generated/google_cloud_ai_generativelanguage_v1beta
        api_keys_environment_variables: GOOGLE_API_KEY,GEMINI_API_KEY
        dev_dependencies: googleapis_auth,test,test_utils
        readme_after_title_text: |
            > [!TIP]
            > Flutter applications should use Firebase AI Logic.
        readme_quickstart_text: |
            ## Quickstart
            [Full quickstart documentation...]
```

### Example Configuration (With description override)

```yaml
name: google_cloud_common
version: 0.0.0
generate:
    apis:
        - path: google/cloud/common
          service_config: google/cloud/common/common.yaml
dart:
    source:
        description_override: Additional metadata for operations.
    codec:
        copyright_year: "2025"
        repository_url: https://github.com/googleapis/google-cloud-dart/tree/main/generated/google_cloud_common
```

### Dart-Specific Fields

**Source configuration (`dart.source`):**
- `description_override` - Override package description (1 library uses this)
- `title_override` - Override package title

**Codec configuration (`dart.codec`):**
- `copyright_year` - Copyright year (all 13 libraries have this set to "2025")
- `repository_url` - GitHub repository URL (all 13 libraries have this)
- `api_keys_environment_variables` - Environment variables for API keys (comma-separated)
- `dev_dependencies` - Development dependencies (comma-separated)
- `readme_after_title_text` - Additional text to insert after README title
- `readme_quickstart_text` - Custom quickstart section for README
- `readme_custom_service_explanation` - Custom service explanation for README

### File Filtering

**No file filtering patterns:** Like Rust, Dart configurations do not use `keep`, `remove`, or regex patterns for file filtering. All 13 libraries have no file filtering configuration.

This suggests the Dart generator either:
1. Has built-in knowledge of which files to keep/remove
2. Generates a clean output directory without needing filtering
3. Handles file management differently than Python/Go containers

### Regex Usage Statistics

- Libraries using `keep`: 0
- Libraries using `remove`: 0
- Libraries using `keep_regex`: 0
- Libraries using `remove_regex`: 0

**Rationale:** The Dart container manages file generation internally without requiring explicit file filtering rules in the configuration.

## Summary

| Language | Total Libraries | With BUILD.bazel | With service_config |
|----------|----------------|------------------|---------------------|
| Python   | 231            | 21               | 224                 |
| Go       | 183            | 172              | 175                 |
| Rust     | 200            | 0 (N/A)          | 155                 |
| Dart     | 13             | 0 (N/A)          | 12                  |

### Key Findings

1. **Python:** All regex patterns successfully expanded to explicit file lists (89 total files)
2. **Go keep:** Only 6 libraries need to preserve specific files (44 total files)
3. **Go remove_regex:** All 175 libraries follow identical pattern structure - should be moved to generator logic
4. **Rust:** No file filtering needed - generator handles file management internally; does not use BUILD.bazel (converted from Sidekick)
5. **Dart:** No file filtering needed - generator handles file management internally; does not use BUILD.bazel (converted from Sidekick)
6. **BUILD.bazel coverage:**
   - Python: 21/231 (9%) have `py_gapic_library` rules
   - Go: 172/183 (94%) have `go_gapic_library` rules
   - Rust: N/A (does not use BUILD.bazel)
   - Dart: N/A (does not use BUILD.bazel)
7. **service_config coverage:**
   - Python: 224/231 (97%) - includes state.yaml data for all API libraries
   - Go: 175/183 (96%)
   - Rust: 155/200 (78%) - includes proto-only packages
   - Dart: 12/13 (92%)
