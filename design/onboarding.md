New Library Onboarding Flow
===========================

This document describes the process for onboarding a new client library into a `librarian`-managed repository. The flow leverages the centralized `sdk.yaml` and `librarianops` automation to ensure consistency and streamline configuration.

Goal
----

To efficiently add a new library to `librarian.yaml` in language repositories, generate its initial code, and prepare it for its first release, primarily driven by automation, but with full support for manual developer intervention.

Prerequisites
-------------

1.	**API Available**: The API definition (protos, service config) must exist in `googleapis/googleapis`.
2.	**Development Environment**: A properly set up development environment as described in the contributor guides (e.g., `design/rust.md`).

Onboarding Workflow (Automation-Driven)
---------------------------------------

### 1. Service Team Completes API Definition

The service team completes their API definition work, ensuring all necessary `.proto` files and `service_config.yaml` are committed and available in the `googleapis/googleapis` repository.

### 2. Platform Team Updates Catalog (via `librarianops sync-catalog`\)

Someone on the Librarian Platform Team (or a scheduled `librarianops` automation) runs `librarianops sync-catalog`.

-	**Action:** This command scans `googleapis/googleapis`, identifies new APIs, and proposes updates to `librarian/sdk.yaml` in the `librarian` repository. A Pull Request is created for review.
-	**Role:** The platform team reviews and merges this PR to update the central catalog.

### 3. `librarianops onboard-apis` Runs

After `sdk.yaml` is updated (e.g., its PR is merged), the `librarianops onboard-apis` command runs (typically scheduled or event-triggered).

-	**Action:**
	1.	`librarianops` compares the updated `sdk.yaml` with the `libraries` list in each language repository's `librarian.yaml`.
	2.	For each new API in `sdk.yaml` not yet onboarded in a language repository:
		-	A bot branch is created (e.g., `feat/onboard-<library-name>`) in the target language repo.
		-	`librarian create <library-name> [api-paths...]` is executed to add the library to `librarian.yaml` and (if applicable) generate its initial code.
		-	A Pull Request is opened in the language repository, proposing the onboarding of the new client library.

### 4. Language Team Review and Merge

The respective Language Team (e.g., `google-cloud-rust` team) reviews the automatically generated PR from `librarianops`.

-	**Action:** The language team verifies the `librarian.yaml` changes and the initially generated code.
-	**Outcome:** Merging the PR integrates the new library into the repository, making it subject to future `librarian generate` and `librarian release` operations.

Manual Onboarding (Developer-Driven)
------------------------------------

Language team members can also manually onboard new libraries for development, testing, or specific non-standard configurations.

### 1. Create a Feature Branch

```bash
git checkout -b feat/manual-onboard-myservice
```

### 2. Run `librarian create`

Execute `librarian create` directly on your local branch.

```bash
librarian create google-cloud-myservice google/cloud/myservice/v1
# ... or with custom config if needed ...
```

### 3. Review, Commit, and Open PR

Review the changes, commit them, and open a PR for review. This allows for direct developer control over the onboarding process.
