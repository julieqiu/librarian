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
A typical service configuration follows the `google.api.Service` schema and begins by identifying the service:

```yaml
type: google.api.Service
config_version: 3
name: example.googleapis.com
title: Example API
```

It includes sections for:
*   **`apis`**: Enumerates the public interfaces provided by the service.
*   **`backend`**: Defines execution properties such as request deadlines and retry policies.
*   **`http`**: Maps RPCs to REST endpoints.
*   **`authentication`**: Specifies required OAuth scopes.
*   **`publishing`**: Metadata connecting the API to documentation and issue tracking.

We are migrating language-neutral settings like **Release Level** (Stable/Beta) and **Transport** into this file to further consolidate sources of truth.

### 2. The SDK Manifest (`sdk.yaml`)

The `sdk.yaml` file (formerly `catalog.yaml`) defines the set of APIs for which we want to create SDKs. It serves as the central registry that Librarian uses to validate, resolve, and enumerate supported APIs across the ecosystem.

This manifest defines:
*   **Canonical API identities**: Unique identifiers for each API.
*   **API maturity**: The status of an API (e.g., GA, Beta, Alpha).
*   **Mapping between APIs and their source locations**: Links APIs to their definitions in the `googleapis` repository.

By centralizing this list, we decouple the *existence* of an API in `googleapis` from its *consumption* by the Librarian system.

### 3. The Repository Manifest (`librarian.yaml`)

Each language repository maintains a `librarian.yaml` file in its root directory. This manifest contains information specific to a particular language or workspace. It serves as the authoritative source for how that repository participates in the ecosystem.

This manifest defines:
*   **Identity**: The repository name (e.g., `google-cloud-go`).
*   **Tooling**: The specific versions of tools, like `protoc`, required for the build.
*   **Inventory**: The list of libraries to be generated for this specific repository.
*   **Overrides**: Language-specific deviations from defaults, such as renaming a package to avoid a keyword collision.

### 4. CLI Dependencies (`tool.yaml`)

The Librarian CLI repository contains a `tool.yaml` file that defines the specifications for the dependencies required by the CLI itself.

This manifest ensures that the environment used to run Librarian is consistent and reproducible by declaring:
*   **Required Runtimes**: Language versions (e.g., Python 3.14, Go 1.25) needed for generators.
*   **Tooling Specs**: Precise versions and installation sources for external dependencies like `gcp-synthtool` or `cargo-semver-checks`.

### 5. Unifying Configurations (Incremental Consolidation)

Our long-term goal is a single source of truth, but we must also support the present. Librarian invokes language-specific generators that were designed before this unified model existed. These generators expect configuration in specific legacy formats (like `*_gapic.yaml` or `BUILD.bazel`) and require inputs that don't yet exist in the upstream `serviceconfig`.

To bridge this gap, Librarian implements an internal aggregation layer:

*   **Internal Representation**: We define an internal model (`internal/serviceconfig/overrides.go`) to aggregate configuration that doesn't yet have a home in `googleapis/googleapis`.
*   **Legacy Compatibility**: At generation time, Librarian reconstructs the necessary legacy artifacts from this model. This keeps existing generators working stably while we migrate fields to the new system.
*   **Reconciliation**: Until the migration is complete, Librarian serves as the integration layer, reconciling existing inputs with the emerging unified model.

## Alternatives Considered

### Status Quo (Distributed Config)
We considered keeping the existing split between `GAPIC YAML`, `BUILD.bazel`, and legacy scripts. We rejected this because it perpetuates the maintenance burden and inconsistency issues we are trying to solve.

### Merging `tool.yaml` with `librarian.yaml`
We considered defining CLI dependencies directly within the repository manifest. However, we decided to separate them because tooling requirements are dependencies of the Librarian CLI itself. They should be versioned and managed alongside the CLI release, rather than being coupled to the configuration of a specific language repository.

### Using Service Configuration for Onboarding
We considered using the presence of a service configuration in `googleapis/googleapis` as the primary trigger for onboarding new SDKs. However, we decided to use a centralized `sdk.yaml` to ensure a deliberate approval process for new libraries. This model allows the Librarian platform team to validate API maturity and quality before resources are committed to generation and release. Additionally, `sdk.yaml` provides a necessary layer for platform-level overrides—such as mapping to non-standard service configuration paths—that are difficult to manage strictly from upstream sources.

### Monolithic Configuration
We considered creating a single massive configuration file for the entire fleet. We rejected this because it would create a bottleneck for language maintainers and obscure the ownership boundaries between platform-wide API definitions and repository-specific build rules.
