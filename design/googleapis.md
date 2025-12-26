# Googleapis Repository and Central Catalog

This document describes the role of the `googleapis/googleapis` repository
and the centralized API catalog within the `librarian` ecosystem.

## The `googleapis/googleapis` Repository

The `googleapis/googleapis` repository (typically located at `https://github.com/googleapis/googleapis`)
serves as the **upstream source of truth** for Google Cloud API definitions.

### Structure and Content
*   **Monorepo:** It's a large monorepo containing `.proto` files and associated
API configuration files for almost all Google Cloud services.
*   **API Definitions:** `.proto` files define the RPC interfaces, messages, and enums for each API.
*   **Service Configuration:** Many APIs include `service_config.yaml` files.
The `publishing:` section within these files is **deprecated** and its responsibilities
for release metadata,
issue tracking, and documentation links are now handled by `librarian.yaml` and `catalog.yaml`.
*   **Organization:** APIs are typically organized by `google/<cloud_product>/<version>/`.
For example, `google/cloud/secretmanager/v1/secretmanager.proto` and `google/cloud/secretmanager/v1/secretmanager_v1.yaml`.

### Role in Librarian
`googleapis/googleapis` is the primary "source" (as referenced in `librarian.yaml`
under `generation.sources.googleapis`) from which client libraries are generated.
`librarian` pulls specific commits of this repository to ensure deterministic
and reproducible builds.

## The Central Catalog

The Central Catalog (`catalog.yaml`) acts as the **central manifest of all
APIs available for client library generation** across the Google Cloud ecosystem.
It lives within the `googleapis/librarian` repository and is maintained
by the Librarian team.

### Purpose
*   **Discovery:** It provides a discoverable list of all APIs that are
"onboarded" and available for client library generation.
*   **Source of Truth for API Locations:** It defines the canonical API
source paths (`api_path`) and service configuration paths (`service_config_path`)
within the `googleapis/googleapis` repository.
*   **Categorization:** It distinguishes between "Standard" APIs (which
should generally be onboarded by all languages) and "Legacy" APIs (which
exist for historical reasons but might only be supported by a subset of languages).

### Structure (Refer to `design/catalog.yaml`)

The `catalog.yaml` does **not** define library names,
as these vary by language (e.g., `google-cloud-secret-manager` in Python
vs `google-cloud-secretmanager` in Rust).
Instead, it lists raw API definitions, categorized into `standard` and `legacy` sections.
The `legacy` section includes a `languages` field to indicate which languages
support that particular legacy API.

### Example Service Configuration (Refer to `design/serviceconfig.yaml`)

This is an example of a `service_config.yaml` file found in the `googleapis/googleapis` repository,
defining how an API is exposed and providing hints for client library generation.
The `publishing:` section is deprecated.

### Role in Librarian Workflow
Language repositories define their own mapping in their local `librarian.yaml`.
The `librarian` tool uses the `catalog.yaml` to resolve the `api_path` and
`service_config_path` for a given library.
If a library in `librarian.yaml` cannot be matched to an entry in `catalog.yaml`
(either `standard` or `legacy`),
generation may fail or require explicit overrides.
