# Googleapis Repository and Central Catalog

This document describes the role of the `googleapis/googleapis` repository
and the `catalog.yaml` file in the `librarian` ecosystem.

## The `googleapis/googleapis` Repository

The `googleapis/googleapis` repository (typically located at `https://github.com/googleapis/googleapis`)
serves as the **upstream source of truth** for Google Cloud API definitions.

### Structure and Content
*   **Monorepo:** It's a large monorepo containing `.proto` files and associated
API configuration files for almost all Google Cloud services.
*   **API Definitions:** `.proto` files define the RPC interfaces, messages, and enums for each API.
*   **Service Configuration:** Many APIs also include `service_config.yaml`
files (formerly `.proto.yaml` files in older conventions) that define RPC behavior,
authentication scopes, client library generation hints,
and other service-specific metadata.
These files are crucial inputs for client library generators.
*   **Organization:** APIs are typically organized by `google/<cloud_product>/<version>/`.
For example, `google/cloud/secretmanager/v1/secretmanager.proto` and `google/cloud/secretmanager/v1/secretmanager_v1.yaml`.

### Role in Librarian
`googleapis/googleapis` is the primary "source" (as referenced in `librarian.yaml`
under `generation.sources.googleapis`) from which client libraries are generated.
`librarian` pulls specific commits of this repository to ensure deterministic
and reproducible builds.

## The Central Catalog (`catalog.yaml`)

The `catalog.yaml` file acts as the **central manifest of all APIs available
for client library generation** across the Google Cloud ecosystem.
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
exist for historical reasons but might not be recommended for new onboardings).

### Structure
The `catalog.yaml` does **not** define library names,
as these vary by language (e.g., `google-cloud-secret-manager` in Python
vs `google-cloud-secretmanager` in Rust).
Instead, it lists the raw API definitions.

The catalog is split into two primary sections: `standard` and `legacy`.

```yaml
# librarian/catalog.yaml

# APIs that SHOULD be onboarded by language repositories.
# These represent the active, supported surface of Google Cloud APIs.
standard:
  - api_path: google/cloud/secretmanager/v1
    service_config_path: google/cloud/secretmanager/v1/secretmanager_v1.yaml

  - api_path: google/pubsub/v1
    service_config_path: google/pubsub/v1/pubsub_v1.yaml

  - api_path: google/cloud/aiplatform/v1
    service_config_path: google/cloud/aiplatform/v1/aiplatform_v1.yaml

# APIs that are maintained for legacy reasons but may not be recommended for new adoption.
# Language repositories may choose to support these if they have existing clients.
legacy:
  - api_path: google/cloud/dialogflow/v2
    service_config_path: google/cloud/dialogflow/v2/dialogflow_v2.yaml
    reason: "Replaced by Dialogflow CX (v3)"

  - api_path: google/datastore/v1beta3
    service_config_path: google/datastore/v1beta3/datastore_v1beta3.yaml
    reason: "Deprecated version"
```

### Role in Librarian Workflow
Language repositories define their own mapping in their local `librarian.yaml`.
The `librarian` tool uses the `catalog.yaml` to resolve the `api_path` and
`service_config_path` for a given library.

For example, `google-cloud-rust/librarian.yaml` might have:
```yaml
libraries:
  - name: google-cloud-secretmanager
    # Implicitly maps to 'google/cloud/secretmanager/v1' found in catalog.yaml
```

The `librarian` tool performs this lookup by searching `catalog.yaml` for
an entry where the `api_path` matches the library's expected path (or by
some other heuristic defined in the tool).
If a library in `librarian.yaml` cannot be matched to an entry in `catalog.yaml`
(either `standard` or `legacy`),
generation may fail or require explicit overrides.