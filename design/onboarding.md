# New Library Onboarding Flow

This document describes the process for onboarding a new client library into a `librarian`-managed repository. The flow leverages the centralized `sdk.yaml` and `librarianops` automation to ensure consistency and streamline configuration.

## Goal

To efficiently add a new library's configuration to `librarian.yaml`, generate its initial code, and prepare it for its first release. The process is primarily driven by automation but provides full support for manual developer intervention, adhering to a clear separation between configuration management and code generation.

## Prerequisites

1.  **API Available**: The API definition (protos, service config) must exist in `googleapis/googleapis`.
2.  **Development Environment**: A properly set up development environment as described in the contributor guides (e.g., `design/rust.md`).

## Onboarding Workflow (Automation-Driven)

### 1. Service Team Completes API Definition

The service team completes their API definition work, ensuring all necessary `.proto` files and `service_config.yaml` are committed and available in the `googleapis/googleapis` repository.

### 2. Platform Team Updates Catalog (via `librarianops sync-catalog`)

Someone on the Librarian Platform Team (or a scheduled `librarianops` automation) runs `librarianops sync-catalog`.

-   **Action:** This command scans `googleapis/googleapis`, identifies new APIs, and proposes updates to `librarian/sdk.yaml` in the `librarian` repository. A Pull Request is created for review.
-   **Role:** The platform team reviews and merges this PR to update the central catalog, making new APIs available for onboarding.

### 3. `librarianops onboard-apis` Runs

After `sdk.yaml` is updated (e.g., its PR is merged), the `librarianops onboard-apis` command runs (typically scheduled or event-triggered).

-   **Action:**
    1.  `librarianops` compares the updated `sdk.yaml` with the `libraries` list in each language repository's `librarian.yaml`.
    2.  For each new API in `sdk.yaml` not yet onboarded in a language repository:
        -   A bot branch is created (e.g., `feat/onboard-<library-name>`) in the target language repo.
        -   **`librarian add <library-name> [api-paths...]`** is executed to add the library's configuration to `librarian.yaml`.
        -   **`librarian generate <library-name>`** is executed to generate the initial source code.
        -   A Pull Request is opened in the language repository, proposing the configuration change and the newly generated code.

### 4. Language Team Review and Merge

The respective Language Team (e.g., `google-cloud-rust` team) reviews the automatically generated PR from `librarianops`.

-   **Action:** The language team verifies the changes to `librarian.yaml` and the initially generated code.
-   **Outcome:** Merging the PR integrates the new library into the repository, making it subject to future `librarian generate` and `librarian release` operations.

## Manual Onboarding (Developer-Driven)

Language team members can also manually onboard new libraries for development, testing, or specific non-standard configurations. This process follows the same separation of configuration and generation.

### 1. Create a Feature Branch

```bash
git checkout -b feat/manual-onboard-myservice
```

### 2. Add the Library Configuration

Run `librarian add` to add the new library to the `librarian.yaml` manifest. This command **only modifies configuration**; it does not generate any code.

```bash
# Librarian infers the API path from the library name and sdk.yaml
librarian add google-cloud-myservice

# Or, specify the API path explicitly
librarian add google-cloud-myservice google/cloud/myservice/v1
```

### 3. Generate the Library Code

Once the library is configured in `librarian.yaml`, run `librarian generate` to create the initial source code.

```bash
librarian generate google-cloud-myservice
```

### 4. Review, Commit, and Open PR

Review the changes to `librarian.yaml` and the newly generated code, commit them, and open a PR for review. This allows for direct developer control over the onboarding process while maintaining the designed workflow.
