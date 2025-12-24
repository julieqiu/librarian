# New Library Onboarding Flow

This document describes the process for onboarding a new client library into a `librarian`-managed repository. The flow leverages the centralized `catalog.yaml` to ensure consistency and streamline configuration.

## Goal
To add a new library to a `librarian.yaml` managed repository, generate its initial code, and prepare it for its first release.

## Prerequisites

1.  **API in Catalog**: The API for the new library must already be defined in the central `librarian/catalog.yaml` (e.g., under `standard:` or `legacy:`). This ensures Librarian knows where to find its upstream proto sources and service configuration.
2.  **Development Environment**: A properly set up development environment as described in the contributor guides (e.g., `design/rust.md`).

## Onboarding Workflow

### 1. Create a Feature Branch
Start by creating a new Git branch for your onboarding work. This ensures your changes are isolated and can be reviewed via a Pull Request.

```bash
git checkout -b feat/onboard-<new-library-name>
```

### 2. Run `librarian create`
Use the `librarian create` command to add the new library to your repository's `librarian.yaml` and perform its initial code generation. The command will look up the API details in `catalog.yaml`.

```bash
librarian create <new-library-name> [apis...]
# Example for a standard API:
librarian create google-cloud-newservice

# Example for an API requiring explicit channels (if not in catalog or override needed):
librarian create google-cloud-anotherservice google/cloud/anotherservice/v1beta google/cloud/anotherservice/v1
```

*   **Arguments:**
    *   `<new-library-name>`: The canonical name you want for your client library in this language repository (e.g., `google-cloud-newservice`). This will be used as the `name` field in the `libraries` list of `librarian.yaml`.
    *   `[apis...]`: (Optional) One or more API paths (e.g., `google/cloud/newservice/v1`) that define the channels for this library. If omitted for a standard API, `librarian` will attempt to infer the API path(s) from the `catalog.yaml` based on the library name.

*   **Actions by `librarian create`:**
    1.  **Catalog Lookup**: If `apis` are not provided, `librarian` will attempt to resolve the primary API path and service configuration path by looking up `<new-library-name>` (or inferring from it) in `catalog.yaml`.
    2.  **`librarian.yaml` Update**: A new entry for `<new-library-name>` is added to the `libraries` list in your `librarian.yaml` with an initial `version` (e.g., `0.1.0`). If `apis` were provided, they are added as `channels` within the library definition.
    3.  **Code Generation**: The initial client library code is generated into the appropriate `output` directory (as defined in `generation.output` or overridden locally).
    4.  **Local Repository Integration**: The command performs necessary steps to integrate the new library into the local repository (e.g., updating `Cargo.toml` for Rust, `setup.py` for Python, adding generated files to Git staging).

### 3. Review and Commit Changes

Review the changes made by `librarian create`.

```bash
git status
git diff
```

Commit the changes, including the updated `librarian.yaml` and the newly generated code.

```bash
git commit -m "feat(<new-library-name>): onboard new client library"
```

### 4. Open a Pull Request

Push your branch and open a Pull Request against the `main` branch of your repository. The PR will be reviewed by maintainers to ensure:
*   The `librarian.yaml` entry is correct.
*   The generated code adheres to style guidelines.
*   All tests pass.

### 5. Final Steps

Once the PR is merged, the new library is successfully onboarded. It will be included in subsequent `librarian generate --all` runs and will be eligible for release via `librarian release --all`.

