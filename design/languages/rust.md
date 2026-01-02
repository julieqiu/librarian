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
-   **Functionality:** Orchestrates the Rust generator to produce client library code, ensuring the output directory is a valid and up-to-date Rust crate. `librarian` executes all commands within the appropriate toolkit container; it does not manage the Rust toolchain directly, relying on the container's pre-configured `PATH`.
-   **Rust-Specifics:**
    1.  **First-Time Crate Initialization:** The command first checks if the library's output directory exists.
        -   **If the output directory does NOT exist:** It performs a one-time scaffolding step by running `cargo new --lib --vcs none <output-dir>`. This creates the directory, a placeholder `Cargo.toml`, and a `src/lib.rs`, establishing a valid Rust crate.
    2.  **`Cargo.toml` Generation:** On every run (including the first), it generates the `Cargo.toml` file from a template (`Cargo.toml.mustache`), populating it with the specific crate name, version, authors, and dependencies defined in `librarian.yaml`. This overwrites any placeholder `Cargo.toml`, ensuring it is always synchronized with the configuration.
    3.  **Generator Execution:** It invokes the Rust generator, passing in the API protos and service configuration. The generator produces the raw `.rs` source files, overwriting any placeholder `src/lib.rs`.
    4.  **Post-Processing:** After generation, `librarian` runs `cargo fmt` on the source code and `taplo fmt` on the `Cargo.toml` file to ensure all generated artifacts conform to Rust style conventions.
    5.  **`veneer` and `modules`:** For complex "veneer" libraries, the `rust.modules` configuration in `librarian.yaml` is used to orchestrate multiple generation steps into different subdirectories, ensuring each module is correctly generated within its respective subdirectory.

### `librarian stage`
-   **Functionality:** Prepares a new release by calculating the next version and updating local package files. This command modifies local files, which are then expected to be committed.
-   **Rust-Specifics:**
    1.  **Version Calculation:** For each library, `librarian` reads the current version from its `Cargo.toml` file. It then analyzes the Git commit history since the last release tag to determine the next semantic version.
    2.  **Update `librarian.yaml`:** Updates the `version` field for the specific library within the `librarian.yaml` manifest.
    3.  **File Updates:** Propagates the new version back into the `Cargo.toml` file using a line-based update mechanism that preserves formatting and comments. This ensures `librarian.yaml` and `Cargo.toml` are synchronized.
    4.  **Validation:** `librarian stage` does not run `cargo test`. Validation is expected to be a separate step in the CI pipeline.

### `librarian tag`
-   **Functionality:** Creates and pushes the Git tag for a staged package. This command is run after `librarian stage` and a `git commit`.
-   **Rust-Specifics:**
    1.  **Change Detection:** Identifies which libraries are candidates for tagging by finding which library versions in `librarian.yaml` have been updated since the last release tag.
    2.  **Tagging:** Creates and pushes a Git tag to the repository, formatted as `<crate-name>-v<version>`.

### `librarian publish`
-   **Functionality:** Publishes a tagged crate to `crates.io`, with a strong focus on workspace integrity and safety.
-   **Rust-Specifics:**
    The `publish` command is a multi-step orchestration that delegates heavily to the `cargo-workspaces` and `cargo-semver-checks` tools.
    1.  **Change Detection:** It identifies which crates are candidates for publishing based on the presence of a new release tag.
    2.  **Publication Planning:** It runs `cargo workspaces plan --skip-published` to generate a publication plan, which analyzes the entire workspace dependency graph.
    3.  **Plan Validation:** The list of changed crates found via Git is compared against the publication plan from `cargo workspaces plan`. The command fails if these lists do not match, preventing accidental releases.
    4.  **Semantic Versioning Checks:** For each crate in the publication plan, it runs `cargo semver-checks` (unless `--skip-semver-checks` is used) to validate that the version bump is appropriate for the code changes.
    5.  **Publishing:** It executes `cargo workspaces publish`, which topologically sorts the workspace dependencies and publishes the crates to `crates.io` in the correct order.

## Alternatives Considered

