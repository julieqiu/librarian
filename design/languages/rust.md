# Rust Implementation Design for Librarian

## Objective
This document defines the Rust-specific implementation, workflows, and command behaviors for the `librarian` CLI.

## Background
The `librarian` CLI is a language-agnostic tool for managing Google Cloud client libraries. For a high-level overview of the CLI's design, refer to [design/cli.md](./../cli.md), and for a detailed explanation of the configuration architecture, see [design/configuration.md](./../configuration.md). However, to function correctly, it requires language-specific logic to interface with each language's unique ecosystem, including its build system, package manager, and naming conventions. This document details that specific implementation for the Rust ecosystem, which is centered around Cargo, `crates.io`, and workspace-based repositories.

## Overview
The Rust implementation for `librarian` is designed around a "single source of truth" principle, where the `librarian.yaml` manifest is the authoritative source for all configuration. The `Cargo.toml` file for each crate is treated as a generated artifact, ensuring that all metadata is derived directly from `librarian.yaml`.

The workflow is orchestrated through a series of `librarian` commands that wrap and delegate to standard Rust tooling like `cargo`, `cargo-workspaces`, `cargo-semver-checks`, and `taplo`. This approach ensures that the process is both idiomatic for Rust developers and robust enough to handle the complexities of a Cargo workspace.

## Detailed Design

### Library Naming and Path Inference
-   **Library Naming:** Rust libraries (crates) follow the `google-cloud-<service>-<version>` naming convention (e.g., `google-cloud-secretmanager-v1`).
-   **Path Inference:** When using `librarian add`, the tool can infer the `api_path` directly from the library name by replacing `-` with `/` after the `google-cloud-` prefix. For example, `google-cloud-secretmanager-v1` is unambiguously inferred to correspond to the API path `secretmanager/v1`.

### `librarian add`
-   **Functionality:** Adds a new library entry to `librarian.yaml`.
-   **Rust-Specifics:**
    -   Typically, a single Rust crate maps to a single API version (`channel`), so multiple `api_path` arguments are uncommon.
    -   The `output` directory, if not specified, defaults to a path derived from the library name by replacing hyphens with slashes (e.g., `google-cloud-secretmanager` would default to `google/cloud/secretmanager`).

### `librarian generate`
-   **Functionality:** Orchestrates the Rust generator to produce client library code and the `Cargo.toml` manifest.
-   **Rust-Specifics:**
    1.  **Environment:** `librarian` uses the `rust` toolchain and `cargo` binary specified in `tool.yaml`.
    2.  **`Cargo.toml` Generation:** The crate's manifest file, `Cargo.toml`, is generated from a template (`Cargo.toml.mustache`). All metadata, including the crate name, version, authors, and dependencies, are populated from the configuration in `librarian.yaml`.
    3.  **Generator Execution:** It invokes the Rust generator, passing in the API protos and service configuration. The generator produces the raw `.rs` source files.
    4.  **Post-Processing:** After generation, `librarian` runs `cargo fmt` on the source code and `taplo fmt` on the `Cargo.toml` file to ensure all generated artifacts conform to Rust style conventions.
    5.  **`veneer` and `modules`:** For complex "veneer" libraries, the `rust.modules` configuration in `librarian.yaml` is used to orchestrate multiple generation steps into different subdirectories.

### `librarian release`
-   **Functionality:** Orchestrates the versioning process for Rust crates within a `librarian` managed repository. The command iterates through all configured libraries in `librarian.yaml`.
-   **Rust-Specifics:**
    1.  **Skip Release:** If a library's configuration in `librarian.yaml` includes `skip_release: true`, that library is skipped entirely for the current release process.
    2.  **Version Determination (Cargo.toml as Source):** For libraries not skipped, `librarian release` reads the *current* version directly from the crate's `Cargo.toml` file. It then uses `internal/semver` (and git history, if available) to calculate the next appropriate semantic version.
    3.  **Update `Cargo.toml`:** The calculated new version is written directly back to the `Cargo.toml` file, using a line-based update mechanism that preserves existing comments, formatting, and any hand-authored sections. This is crucial because `Cargo.toml` files often contain valuable developer-added context.
    4.  **Update `librarian.yaml`:** The `version` field for the corresponding library in `librarian.yaml` is updated to reflect this new version. This ensures that `librarian.yaml` remains synchronized with the authoritative version in `Cargo.toml`.
    5.  **Validation:** `librarian release` itself does not run `cargo test`. It is expected that `cargo test` and other comprehensive validation will be executed as a separate step in the broader CI/release pipeline to ensure library quality. This design choice prevents `librarian release` from becoming overly complex and allows for independent validation strategies.

### `librarian publish`
-   **Functionality:** Publishes crates that have changed since the last release tag, with a strong focus on workspace integrity and safety.
-   **Rust-Specifics:**
    The `publish` command is a multi-step orchestration that delegates heavily to the `cargo-workspaces` and `cargo-semver-checks` tools to ensure a safe and correct release.
    1.  **Change Detection:** It identifies which crates are candidates for publishing by finding which `Cargo.toml` files have been modified since the last release tag (using `git diff`).
    2.  **Publication Planning:** It runs `cargo workspaces plan --skip-published` to generate a publication plan, which analyzes the entire workspace dependency graph.
    3.  **Plan Validation:** The list of changed crates found via `git` is compared against the publication plan from `cargo workspaces plan`. The command fails if these lists do not match, preventing accidental releases with missing dependency version bumps.
    4.  **Semantic Versioning Checks:** For each crate in the publication plan, it runs `cargo semver-checks` (unless `--skip-semver-checks` is used) to validate that the version bump is appropriate for the code changes.
    5.  **Publishing:** It executes `cargo workspaces publish`, which automatically handles the topological sort of the workspace dependencies and publishes the crates to `crates.io` in the correct order.
    6.  **Tagging:** After a successful publish, it creates and pushes a Git tag for each published crate, formatted as `<crate-name>-v<version>`.

## Alternatives Considered

