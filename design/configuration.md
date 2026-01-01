Librarian Configuration Design
==============================

Objective
---------

Define the configurations used by the Librarian CLI.

Background
----------

Today, configuring a Google Cloud client library requires stitching together state from multiple disparate files. You might find transport settings in `GAPIC YAML`, file inclusion rules in `BUILD.bazel`, and versioning logic hidden in release scripts.

This fragmentation creates friction. It couples language-neutral concerns, like a service's retry policy, with language-specific decisions, like a Rust crate name. If a service owner updates a deadline, that change currently requires manual updates across every language repository.

We want to make this simpler. We are introducing a unified configuration architecture that decouples these concerns and establishes a clear, predictable flow of information from upstream API definitions to downstream client libraries.

Overview
--------

Our design structures configuration into four distinct domains of ownership. Each domain has a single authoritative manifest:

1.	**API Definition (`serviceconfig`\)**: Service-neutral information owned by the service teams.
2.	**SDK Manifest (`sdk.yaml`\)**: Defines the APIs we want to create SDKs for.
3.	**Repository Manifest (`librarian.yaml`\)**: Information specific to a language or workspace.
4.	**CLI Dependencies (`tool.yaml`\)**: Defines the specifications for the dependencies for the Librarian CLI.

Librarian acts as the integration engine, reconciling these inputs to produce consistent, high-quality client libraries.

Detailed Design
---------------

### 1. The API Definition (`serviceconfig`\)

The service configuration defines the surface and behavior of a Google API. This is service-neutral information owned and maintained by the service teams within the `googleapis/googleapis` repository. It is the canonical description of what the API looks like to the tools that generate clients, documentation, and support infrastructure.

Librarian reads this file but does not modify it.

```yaml
# A typical service configuration follows the google.api.Service schema.
type: google.api.Service
config_version: 3
name: secretmanager.googleapis.com
title: Secret Manager API

# NEW: release_level defines the stability of the API (e.g., STABLE, BETA, ALPHA).
release_level: STABLE

# apis enumerates the public interfaces provided by the service.
apis:
  - name: google.cloud.secretmanager.v1.SecretManagerService

# backend defines execution properties such as request deadlines and retry policies.
backend:
  rules:
  - selector: google.cloud.secretmanager.v1.SecretManagerService.*
    deadline: 60.0

# http maps RPCs to REST endpoints.
http:
  rules:
  - selector: google.cloud.secretmanager.v1.SecretManagerService.AccessSecretVersion
    get: /v1/{name=projects/*/secrets/*/versions/*}:access

# authentication specifies required OAuth scopes.
authentication:
  rules:
  - selector: google.cloud.secretmanager.v1.SecretManagerService.*
    oauth:
      canonical_scopes: https://www.googleapis.com/auth/cloud-platform

# publishing metadata connects the API to documentation and issue tracking.
publishing:
  documentation_uri: https://cloud.google.com/secret-manager/docs
  github_label: api: secretmanager
  organization: CLOUD

  # NEW: transports defines the supported transport protocols (e.g., GRPC, REST).
  transports: [GRPC, REST]
```

### 2. The SDK Manifest (`sdk.yaml`\)

The `sdk.yaml` file (formerly `catalog.yaml`) defines the set of APIs for which we want to create SDKs. It serves as the central registry that Librarian uses to validate, resolve, and enumerate supported APIs across the ecosystem.

#### Structure

The file defines canonical API identities and maps them to their upstream source locations.

```yaml
# Standard APIs: The active, supported surface.
standard:
  - api_path: google/cloud/secretmanager/v1
    service_config_path: google/cloud/secretmanager/v1/secretmanager_v1.yaml

# Legacy APIs: Maintained for legacy reasons.
legacy:
  - api_path: google/cloud/secretmanager/v1beta1
    service_config_path: google/cloud/secretmanager/v1beta1/secretmanager_v1beta1.yaml
    languages: [go, python]
```

-	**`standard`**: A list of APIs that are supported by default. These are APIs that will be automatically generated if `librarian create --all` is executed.

	-	**`api_path`**: The path to the API in `googleapis` (e.g., `google/cloud/secretmanager/v1`).

	-	**`service_config_path`**: The path to the service configuration file relative to the API path.

-	**`legacy`**: A list of APIs that are maintained for backward compatibility, but are no longer intended to be created for new SDKs. These will be skipped by `librarian create --all`.

	-	**`languages`**: Restricts support for legacy APIs to specific languages.

### 3. The Repository Manifest (`librarian.yaml`\)

Each language repository maintains a `librarian.yaml` file in its root directory. This manifest contains information specific to a particular language or workspace. It serves as the authoritative source for how that repository participates in the ecosystem.

#### Specification

**Top-Level Configuration:**

-	**`language`**: The programming language of the repository (e.g., `rust`, `python`, `go`).
-	**`repo`**: The repository identifier (e.g., `googleapis/google-cloud-rust`).
-	**`default`**: A block defining shared defaults for all libraries.
	-	**`output`**: Default directory for generated code.
	-	**`transport`**: Default transport protocols.
	-	**`release_level`**: Default release stability (e.g., `stable`).
	-	**`tag_format`**: Template for git tags.
-	**`sources`**: External dependencies.
	-	**`googleapis`**: Specific commit and SHA256 of the upstream definitions.
-	**`libraries`**: A list of managed libraries.

**Library Configuration:** Each entry in `libraries` defines a single package:

-	**`name`**: The library identifier.
-	**`version`**: The current semantic version.
-	**`output`**: Overrides the default output directory.
-	**`channels`**: A list of API versions to include (e.g., `google/cloud/secretmanager/v1`).
-	**`veneer`**: Boolean indicating if this is a wrapper library with handwritten components.

**Rust Example (`google-cloud-rust/librarian.yaml`\):**

```yaml
language: rust
repo: googleapis/google-cloud-rust

default:
  output: src/generated/
  release_level: stable
  rust:
    package_dependencies:
      - name: api
        package: google-cloud-api
        source: google.api
      - name: bytes
        package: bytes
        force_used: true
      - name: gax
        package: google-cloud-gax
        used_if: services
    disabled_rustdoc_warnings:
      - redundant_explicit_links
      - broken_intra_doc_links
    generate_setter_samples: "true"

libraries:
  - name: google-cloud-secretmanager
    version: 1.2.0
    rust:
      package_name_override: google-cloud-secretmanager-v1
```

**Python Example (`google-cloud-python/librarian.yaml`\):**

```yaml
language: python
repo: googleapis/google-cloud-python

default:
  tag_format: '{name}/v{version}'

libraries:
  - name: google-cloud-secret-manager
    version: 2.25.0
    channels:
      - path: google/cloud/secretmanager/v1
      - path: google/cloud/secretmanager/v1beta2
      - path: google/cloud/secrets/v1beta1
    keep:
      - packages/google-cloud-secret-manager/CHANGELOG.md
      - docs/CHANGELOG.md
      - samples/README.txt
      - samples/snippets/README.rst
      - tests/system
```

**Python-Specific Configuration (`python` block):**

-	**`opt_args`**: A list of strings passed as options to the Python GAPIC generator (e.g., `warehouse-package-name=...`).
-	**`opt_args_by_channel`**: A map of API paths to option lists, allowing per-version generator overrides.

**Rust-Specific Configuration (`rust` block):**

-	**`modules`**: For veneer crates, defines multiple generation targets (e.g., separate proto and GAPIC outputs).
	-	`source`, `template`, `output` per module.
-	**`package_dependencies`**: Defines external crate dependencies.
-	**`discovery`**: Configuration for Long-Running Operation (LRO) polling.
-	**`documentation_overrides`**: Fixes for upstream documentation issues.
-	**`disabled_rustdoc_warnings`**: Suppresses specific rustdoc lints.
-	**`generate_setter_samples`**: Toggles generation of sample code.

**Go-Specific Configuration (`go` block):**

-	**`module_path_version`**: The Go module version suffix (e.g., `/v2`).
-   **`go_apis`**: Overrides for specific API paths within the module (e.g., `proto_package` renaming).

### 4. CLI Dependencies (`tool.yaml`)

The Librarian CLI repository contains a `tool.yaml` file that defines the specifications for the dependencies required by the CLI itself.

#### Structure

The file declares required runtimes and external tools.

```yaml
version: v1

python:
  version: "3.14"
  tools:
    pip:
      - name: gcp-synthtool
        version: git+https://...

rust:
  version: "1.76"
  tools:
    cargo:
      - name: cargo-semver-checks
        version: "0.44.0"
```

-	**`version`**: The schema version of the tool manifest.
-	**`<language>`**: Language-specific tooling sections (e.g., `python`, `rust`).
	-	**`version`**: Required runtime version.
	-	**`tools`**: Lists of tools categorized by installer (e.g., `pip`, `cargo`, `curl`).
		-	**`name`**: Package name or download URL.
		-	**`version`**: Version pin or commit hash.
		-	**`sha256`**: Checksum for direct downloads.

### 5. Library Types

Different libraries have different implementation strategies. A library might be fully generated from protobuf definitions, or it might combine generated code with handwritten helpers, or it might be entirely handwritten but depend on generated subpackages. Librarian needs to know which kind of library it is working with to make the right decisions about what to generate and what to preserve.

We distinguish five library types. The type is inferred from the presence or absence of specific configuration fields. This keeps the common case simple while allowing explicit control when needed.

The classification logic proceeds in order:

1.	If `veneer` is true and the library has language-specific modules, it is a **veneer with generated layers**.
2.	If `veneer` is true but the library has no modules, it is a **purely handwritten veneer**.
3.	If the library has a `keep` field, it is **hybrid**.
4.	If a service configuration exists for the API path, it is **autogenerated with service config**.
5.	Otherwise, it is **autogenerated from proto only**.

#### Autogenerated: Proto Only

The library is generated entirely from protobuf definitions. No service configuration exists, so the generator produces only the basic types and serialization code.

The output directory is cleaned before each generation. Every file is replaced. This ensures a clean build and prevents orphaned files from accumulating when the API changes.

```yaml
libraries:
  - name: google-cloud-type
    version: 1.2.0
```

#### Autogenerated: With Service Config

The library is generated from both protobuf definitions and a service configuration file. The service configuration supplies retry policies, timeouts, authentication scopes, and HTTP bindings. The generator produces a complete client library with transport-aware retry logic and request routing.

Like proto-only libraries, the output directory is cleaned before generation. Every file is replaced.

```yaml
libraries:
  - name: google-cloud-secretmanager-v1
    version: 1.2.0
    channels:
      - path: google/cloud/secretmanager/v1
```

#### Hybrid

The library is generated, but some files are handwritten. The `keep` field lists the handwritten files. Before generation, Librarian removes all files in the output directory except those in the `keep` list. After generation, the handwritten files remain alongside the generated code.

This is useful for libraries that need custom error types, helper functions, or integration code that the generator cannot produce automatically.

```yaml
libraries:
  - name: google-cloud-compute-v1
    version: 2.0.0
    keep:
      - src/errors.rs
      - src/operation.rs
```

The `keep` field is a list of paths relative to the library's output directory. Each path names a file or directory to preserve. If a path names a directory, all files in that directory are preserved recursively.

#### Veneer: Purely Handwritten

The library is entirely handwritten. Setting `veneer: true` without providing any modules signals that no generation should occur. Librarian skips the generation step entirely and leaves all files untouched.

This is used for foundational libraries like authentication helpers or test utilities that do not correspond to any generated API surface.

```yaml
libraries:
  - name: google-cloud-auth
    version: 1.3.0
    veneer: true
```

#### Veneer: With Generated Layers

The library is primarily handwritten, but it contains generated submodules. Each module specifies a source, template, and output directory. Librarian generates code into each module's output directory while preserving everything outside those directories.

This allows fine-grained control over what gets generated. One module might generate protobuf types, another might generate a gRPC client, and a third might generate conversion helpers. The handwritten veneer code orchestrates these pieces into a coherent public API.

```yaml
libraries:
  - name: google-cloud-storage
    version: 1.6.0
    veneer: true
    rust:
      modules:
        - output: src/generated/gapic
          source: google/storage/v2
          template: grpc-client
        - output: src/generated/protos
          source: google/storage/v2
          template: prost
```

Before generating each module, Librarian cleans only that module's output directory. Files outside the module directories are never touched. This ensures that the handwritten veneer code remains intact while the generated layers are rebuilt from scratch.

Alternatives Considered
-----------------------

### Merging `tool.yaml` with `librarian.yaml`

We considered defining CLI dependencies directly within the repository manifest. However, we decided to separate them because tooling requirements are dependencies of the Librarian CLI itself. They should be versioned and managed alongside the CLI release, rather than being coupled to the configuration of a specific language repository.

### Using Service Configuration for Onboarding

We considered using the presence of a service configuration in `googleapis/googleapis` as the primary trigger for onboarding new SDKs. However, we decided to use a centralized `sdk.yaml` to ensure a deliberate approval process for new libraries. This model allows the Librarian platform team to validate API maturity and quality before resources are committed to generation and release. Additionally, `sdk.yaml` provides a necessary layer for platform-level overrides—such as mapping to non-standard service configuration paths—that are difficult to manage strictly from upstream sources.

### Status Quo (Distributed Config)

We considered keeping the existing split between `GAPIC YAML`, `BUILD.bazel`, and legacy scripts. We rejected this because it perpetuates the maintenance burden and inconsistency issues we are trying to solve.

### Monolithic Configuration

We considered creating a single massive configuration file for the entire fleet. We rejected this because it would create a bottleneck for language maintainers and obscure the ownership boundaries between platform-wide API definitions and repository-specific build rules.

Migration Plan
--------------

While the long-term goal is a single upstream source of truth, Librarian must operate in a transitional environment. To bridge this gap without hard-coding logic, we will use an enhanced `sdk.yaml` as the central, declarative configuration hub.

-   `sdk.yaml` will live in `googleapis/librarian`, and contain temporary fields. The long-term goal is to migrate this file to `googleapis/googleapis`.
-   These temporary fields will aggregate language-neutral configuration (e.g., retry policies, deadlines) that have not yet been migrated to their final destination in the upstream `serviceconfig.yaml` files.
-   **Synthesized Configuration:** Librarian will be the sole orchestrator of configuration. It will read the local `sdk.yaml`, the repository's `librarian.yaml`, and the upstream `serviceconfig.yaml` files. From these sources, it will synthesize the final configuration artifacts (such as legacy `GAPIC YAML` files) required by the language-specific generators.
-   **Generator Decoupling:** The generators will be modified to no longer read from any legacy upstream configuration files. Their only input will be the configuration artifacts constructed and passed to them by Librarian, ensuring a single, consistent source of truth for the generation process.



### Consolidation Mapping

The following table outlines the planned consolidation of legacy configuration files into the unified manifests:

| Legacy File                       | Primary Fields                                     | New Location                 |
|:----------------------------------|:---------------------------------------------------|:-----------------------------|
| **`*_gapic.yaml`**                | Package names, class overrides, generation options | `librarian.yaml`             |
| **`BUILD.bazel`**                 | Transport settings, numeric enum behavior          | `serviceconfig`              |
| **`*_grpc_service_config.json`**  | Retry policies, request deadlines                  | `serviceconfig`              |
| **`.sidekick.toml`**              | Library inventory, state, Rust-specific settings   | `librarian.yaml`             |
| **`.librarian/state.yaml`**       | Library versions, generated commits, metadata      | `librarian.yaml`             |
| **API Index / Central Catalog**   | API paths, service config locations                | `sdk.yaml`                   |
| **`synthtool` (post-processing)** | Custom file movement, templating logic             | `librarian.yaml` (minimized) |

Deprecation
-----------

Once all of the GAPIC generators have migrated, we will delete the legacy configuration files.
