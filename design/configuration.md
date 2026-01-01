# Librarian Configuration Design

## Objective

Define the configurations used by the Librarian CLI.

## Background

Today, configuring a Google Cloud client library requires stitching together state from multiple disparate files. You might find transport settings in `GAPIC YAML`, file inclusion rules in `BUILD.bazel`, and versioning logic hidden in release scripts.

This fragmentation creates friction. It couples language-neutral concerns, like a service's retry policy, with language-specific decisions, like a Rust crate name. If a service owner updates a deadline, that change currently requires manual updates across every language repository.

We want to make this simpler. We are introducing a unified configuration architecture that decouples these concerns and establishes a clear, predictable flow of information from upstream API definitions to downstream client libraries.

## Overview

Our design structures configuration into four distinct domains of ownership. Each domain has a single authoritative manifest:

1.  **API Definition (`serviceconfig`)**: Service-neutral information owned by the service teams.
2.  **SDK Manifest (`sdk.yaml`)**: Defines the APIs we want to create SDKs for.
3.  **Repository Manifest (`librarian.yaml`)**: Information specific to a language or workspace.
4.  **CLI Dependencies (`tool.yaml`)**: Defines the specifications for the dependencies for the Librarian CLI.

Librarian acts as the integration engine, reconciling these inputs to produce consistent, high-quality client libraries.

## Detailed Design

### 1. The API Definition (`serviceconfig`)

The service configuration defines the surface and behavior of a Google API. This is service-neutral information owned and maintained by the service teams within the `googleapis/googleapis` repository. It is the canonical description of what the API looks like to the tools that generate clients, documentation, and support infrastructure.

Librarian reads this file but does not modify it.

#### Structure
A typical service configuration follows the `google.api.Service` schema and begins by identifying the service. Using Secret Manager as an example:

```yaml
type: google.api.Service
config_version: 3
name: secretmanager.googleapis.com
title: Secret Manager API
```

It includes sections for:
*   **`apis`**: Enumerates the public interfaces provided by the service (e.g., `google.cloud.secretmanager.v1.SecretManagerService`).
*   **`backend`**: Defines execution properties such as request deadlines and retry policies.
*   **`http`**: Maps RPCs to REST endpoints.
*   **`authentication`**: Specifies required OAuth scopes.
*   **`publishing`**: Metadata connecting the API to documentation and issue tracking.

We are migrating language-neutral settings like **Release Level** (Stable/Beta) and **Transport** into this file to further consolidate sources of truth.

### 2. The SDK Manifest (`sdk.yaml`)

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
  - api_path: google/cloud/dialogflow/v2
    service_config_path: google/cloud/dialogflow/v2/dialogflow_v2.yaml
    languages: [go, python, java]
```

*   **`standard`**: A list of APIs that are supported by default.
    *   **`api_path`**: The path to the API in `googleapis` (e.g., `google/cloud/secretmanager/v1`).
    *   **`service_config_path`**: The path to the service configuration file relative to the API path.
*   **`legacy`**: A list of APIs that are maintained for backward compatibility.
    *   **`languages`**: Restricts support for legacy APIs to specific languages.

### 3. The Repository Manifest (`librarian.yaml`)

Each language repository maintains a `librarian.yaml` file in its root directory. This manifest contains information specific to a particular language or workspace. It serves as the authoritative source for how that repository participates in the ecosystem.

#### Structure
The file defines global settings, defaults, and the specific libraries to manage.

**Rust Example (`google-cloud-rust/librarian.yaml`)**:
```yaml
language: rust
repo: googleapis/google-cloud-rust

default:
  output: src/generated/
  transport: grpc+rest

libraries:
  - name: google-cloud-secretmanager
    version: 1.2.0
    rust:
      package_name_override: google-cloud-secretmanager-v1
```

**Python Example (`google-cloud-python/librarian.yaml`)**:
```yaml
language: python
repo: googleapis/google-cloud-python

default:
  transport: grpc+rest

libraries:
  - name: google-cloud-secret-manager
    version: 2.16.0
    python:
      opt_args:
        - "warehouse-package-name=google-cloud-secret-manager"
```

*   **`language`**: The programming language of the repository (e.g., `rust`, `python`).
*   **`repo`**: The repository identifier.
*   **`default`**: Shared settings applied to all libraries.
    *   **`output`**: Default directory for generated code.
    *   **`transport`**: Default transport protocols.
*   **`libraries`**: The inventory of managed libraries.
    *   **`name`**: The library identifier.
    *   **`version`**: The current semantic version.
    *   **`overrides`**: Language-specific configuration (e.g., `package_name_override` for Rust, `opt_args` for Python).

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

*   **`version`**: The schema version of the tool manifest.
*   **`<language>`**: Language-specific tooling sections (e.g., `python`, `rust`).
    *   **`version`**: Required runtime version.
    *   **`tools`**: Lists of tools categorized by installer (e.g., `pip`, `cargo`, `curl`).
        *   **`name`**: Package name or download URL.
        *   **`version`**: Version pin or commit hash.
        *   **`sha256`**: Checksum for direct downloads.

## Alternatives Considered

### Merging `tool.yaml` with `librarian.yaml`
We considered defining CLI dependencies directly within the repository manifest. However, we decided to separate them because tooling requirements are dependencies of the Librarian CLI itself. They should be versioned and managed alongside the CLI release, rather than being coupled to the configuration of a specific language repository.

### Using Service Configuration for Onboarding
We considered using the presence of a service configuration in `googleapis/googleapis` as the primary trigger for onboarding new SDKs. However, we decided to use a centralized `sdk.yaml` to ensure a deliberate approval process for new libraries. This model allows the Librarian platform team to validate API maturity and quality before resources are committed to generation and release. Additionally, `sdk.yaml` provides a necessary layer for platform-level overrides—such as mapping to non-standard service configuration paths—that are difficult to manage strictly from upstream sources.

### Status Quo (Distributed Config)
We considered keeping the existing split between `GAPIC YAML`, `BUILD.bazel`, and legacy scripts. We rejected this because it perpetuates the maintenance burden and inconsistency issues we are trying to solve.

### Monolithic Configuration
We considered creating a single massive configuration file for the entire fleet. We rejected this because it would create a bottleneck for language maintainers and obscure the ownership boundaries between platform-wide API definitions and repository-specific build rules.

## Plan

While the long-term goal is a single source of truth, Librarian must operate in an environment where that does not yet fully exist. Our generators were designed to consume configuration from multiple independent sources, and many required fields have not yet been migrated to the upstream service configuration.

To move forward without requiring a massive upstream migration as a prerequisite, a transitional bridge will be implemented in the `googleapis/librarian` repository for data that will eventually live in the `serviceconfig` and `sdk.yaml` in `googleapis/googleapis`.

### The Internal Bridge

*   **Internal Representation**: We use an internal model (`internal/serviceconfig/overrides.go`) to aggregate language-neutral configuration that does not yet have a home in `googleapis/googleapis`.
*   **Legacy Reconstruction**: At generation time, Librarian can reconstruct legacy configuration artifacts (such as `GAPIC YAML`) from this internal model. This allows generator behavior to remain stable while we incrementally consolidate the underlying configuration.
*   **Reconciliation**: Librarian serves as the integration layer, reconciling existing inputs with the emerging unified model until the migration to `serviceconfig` is complete.

### Consolidation Mapping

The following table outlines the planned consolidation of legacy configuration files into the unified manifests:

| Legacy File | Primary Fields | New Location |
| :--- | :--- | :--- |
| **`*_gapic.yaml`** | Package names, class overrides, generation options | `librarian.yaml` |
| **`BUILD.bazel`** | Transport settings, numeric enum behavior | `serviceconfig` |
| **`*_grpc_service_config.json`** | Retry policies, request deadlines | `serviceconfig` |
| **`.sidekick.toml`** | Library inventory, state, Rust-specific settings | `librarian.yaml` |
| **`.librarian/state.yaml`** | Library versions, generated commits, metadata | `librarian.yaml` |
| **API Index / Central Catalog** | API paths, service config locations | `sdk.yaml` |
| **`synthtool` (post-processing)** | Custom file movement, templating logic | `librarian.yaml` (minimized) |

## Deprecation

Once all of the GAPIC generators have migrated, we will delete the legacy configuration files.
