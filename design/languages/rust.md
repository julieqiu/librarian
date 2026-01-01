# Rust Implementation Design

This document details the architectural design of the Rust support in `librarian`. It covers the generation strategy, configuration mapping, and the release pipeline implementation.

## Overview

The Rust implementation (formerly "Sidekick") focuses on generating idiomatic Rust crates from Google APIs. It relies heavily on `Cargo.toml` as the source of truth for package metadata and `librarian.yaml` for generation configuration.

## Configuration Architecture

### `librarian.yaml` mapping
The unified `librarian.yaml` schema is mapped to internal Rust structures:
*   **Source of Truth**: `librarian.yaml` replaces the legacy `.sidekick.toml`.
*   **Veneers**: Libraries with `veneer: true` in `librarian.yaml` use the `rust.modules` list to define multiple generation targets (e.g., separate proto and GAPIC crates) within a single package.
*   **Overrides**: Rust-specific overrides (e.g., `disabled_rustdoc_warnings`, `package_dependencies`) are defined in the `rust` block of the library configuration.

## Generation Pipeline

1.  **Preparation**:
    *   `googleapis` is fetched to the commit specified in `sources`.
    *   Output directories are cleaned, respecting the `keep` list.
2.  **Execution**:
    *   Librarian invokes `protoc` with the Rust plugins.
    *   It handles complex logic for "veneer" libraries, allowing handwritten layers to wrap generated code.
3.  **Post-Processing**:
    *   `cargo fmt` is executed to ensure code quality.
    *   `README.md` generation is handled (if applicable).

## Release Pipeline

The Rust release pipeline is split into distinct "Prepare" and "Publish" phases.

### 1. Preparation (`librarian release`)
*   **Change Detection**: Uses `git diff <last_tag>..HEAD` to detect changes in the crate directory.
*   **Versioning**:
    *   Parses `Cargo.toml` to find the current version.
    *   Calculates the next version using strict SemVer rules (defaulting to Minor bump for pre-1.0 features).
    *   **In-Place Update**: Uses line-based replacement in `Cargo.toml` to preserve comments and formatting.
*   **Verification**: Runs `cargo semver-checks` to detect accidental breaking changes.

### 2. Publication (`librarian publish`)
*   **Planning**: Invokes `cargo workspaces plan` to determine the topological order of publication and valid publish sets.
*   **Safety Check**: Validates that the `librarian` calculated release set matches the `cargo` plan.
*   **Execution**: Runs `cargo workspaces publish` to push crates to crates.io.

## Constraints & Assumptions
*   **Monorepo**: Assumes a workspace-based structure (`google-cloud-rust`).
*   **Cargo**: Relies on `cargo` and `cargo-workspaces` being available.