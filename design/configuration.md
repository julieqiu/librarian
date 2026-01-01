# Librarian Configuration Design

**Author:** Librarian Team  
**Status:** Draft

## Objective

To implement a unified configuration architecture that coordinates the generation and release of Google Cloud client libraries by establishing authoritative sources of truth.

## Background

Generating a client library today requires coordinating state across multiple disparate files. Settings for supported transports live in `GAPIC YAML`, file inclusion rules in `BUILD.bazel`, and versioning information in various release scripts.

This fragmentation couples language-neutral concerns (like a service's retry policy) with language-specific concerns (like a Rust crate name). When a service-level setting changes, that change currently must be manually propagated to every language-specific configuration file.

We need a system that decouples these concerns and provides a clear, predictable flow of information from upstream API definitions to downstream client libraries.

## Overview

The configuration for Librarian is structured into three distinct domains of ownership, each with a single authoritative manifest:

1.  **Repository Manifest (`librarian.yaml`)**: Defines *how* a language repository builds its libraries.
2.  **Registry (`catalog.yaml`)**: Defines *what* APIs are available to be built.
3.  **API Definition (`serviceconfig.yaml`)**: Defines *how* an upstream API behaves.

Librarian acts as the integration engine, reading these inputs to produce consistent client libraries.

## Detailed Design

### 1. The Repository Manifest (`librarian.yaml`)

Each language repository maintains a `librarian.yaml` file in its root directory. This file is the authoritative source for that repository's participation in the ecosystem. It defines the repository's identity, required tooling versions, the list of managed libraries, and any language-specific overrides.

### 2. The Registry (`catalog.yaml`)

Librarian maintains a central `catalog.yaml` file to resolve the identity and location of target APIs. The catalog lists every API available for generation, defining its canonical identity and mapping it to its source definition in the `googleapis` repository. This decouples the *existence* of an API from its *consumption* by a specific language.

### 3. The API Definition (`serviceconfig.yaml`)

The service configuration file defines the surface and behavior of a Google API. It is the canonical description of what the API looks like to the tools that generate clients, documentation, and support infrastructure. The configuration is expressed using the `google.api.Service` schema. Librarian reads this file but does not modify it.

#### Structure
A typical service configuration begins by identifying the service:
```yaml
type: google.api.Service
config_version: 3
name: example.googleapis.com
title: Example API
```
*   `type`: Identifies the file as a service configuration.
*   `config_version`: Specifies the schema version.
*   `name`: The globally unique service name.
*   `title`: A human-readable label.

#### APIs
```yaml
apis:
- name: google.cloud.example.v1.ExampleService
```
The `apis` section enumerates the public interfaces provided by the service. The name must match the fully qualified protobuf service name. These definitions determine which RPCs are exposed and form the basis for all generated clients.

#### Documentation
```yaml
documentation:
  summary: An example API for demonstration.
```
Documentation fields provide human-readable descriptions used in generated reference material. These fields are used by documentation generators but do not affect runtime behavior.

#### Backend
```yaml
backend:
  rules:
  - selector: google.cloud.example.v1.ExampleService.*
    deadline: 60.0
```
Backend rules define execution properties such as request deadlines. Rules are applied by the selector and may target individual methods or entire services.

#### HTTP
```yaml
http:
  rules:
  - selector: google.iam.v1.IAMPolicy.GetIamPolicy
    get: /v1/{resource=projects/*/resources/*}
```
HTTP rules map RPCs to REST endpoints. They define HTTP methods, paths, and request bodies for REST-based access.

#### Authentication
```yaml
authentication:
  rules:
  - selector: google.cloud.example.v1.ExampleService.*
    oauth:
      canonical_scopes: https://www.googleapis.com/auth/cloud-platform
```
Authentication rules specify required OAuth scopes. They apply at the method or service level and are enforced by infrastructure.

#### Publishing
```yaml
publishing:
  documentation_uri: https://cloud.google.com/example/docs
  github_label: api: example
  organization: CLOUD
  library_settings:
  - version: google.cloud.example.v1
    java_settings:
      library_package: com.google.cloud.example.v1
```
Publishing metadata connects the API to documentation, issue tracking, and client generation. Language-specific settings allow per-language customization without altering the API definition.

#### Migration Targets
The following configuration items are planned to be moved into the service configuration:
*   **Release Level**: Defining whether an API is Stable, Beta, or Alpha.
*   **Transport**: Specifying supported transport protocols (gRPC, REST).

### 4. Unifying Configurations (Incremental Consolidation)

While the long-term goal is to have a single source of truth, Librarian must operate in an environment where that does not yet exist. Librarian invokes language-specific generators that were designed to consume configuration from multiple independent sources rather than a single unified model.

In addition, the service configuration does not yet contain all of the information required by existing generators. Specifically, required inputs remain defined in generator-specific files such as `*_grpc_service_config.json`, `*_gapic.yaml`, and `BUILD.bazel`.

To avoid blocking progress on a complete migration to the service configuration file, Librarian implements an internal aggregation layer:

*   **Internal Representation**: Librarian defines an internal model (`internal/serviceconfig/overrides.go`) that aggregates language-neutral configuration that does not yet have a home in `googleapis/googleapis`.
*   **Legacy Compatibility**: Librarian may reconstruct legacy configuration artifacts (e.g., GAPIC YAML) from this aggregated model at generation time. This allows generator behavior to remain stable while configuration is incrementally consolidated.
*   **Reconciliation**: Until the migration to the service configuration is complete, Librarian serves as the integration layer that reconciles existing inputs with the emerging unified model.

## Alternatives Considered

### Status Quo (Distributed Config)
We considered maintaining the existing split between `GAPIC YAML`, `BUILD.bazel`, and `sidekick.toml`. However, this was rejected because it perpetuates the maintenance burden and data inconsistency problems described in the Background.

### Monolithic Configuration
We considered a single massive configuration file for the entire fleet. This was rejected because it would create a bottleneck for language maintainers and obscure the ownership boundaries between platform-wide API definitions and repository-specific build rules.