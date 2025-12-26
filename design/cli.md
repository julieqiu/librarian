# Librarian CLI Specification

This document describes the command-line interface for the new `librarian` tool,
which replaces the legacy `sidekick` utility.
The design emphasizes consistency and clear resource management verbs (Create,
Generate, Update, Release, Publish).

## Command Reference

### `librarian create`
**Usage:** `librarian create <name> [apis...] [flags]`
*   **Purpose:** Adds a new library to the `librarian.yaml` configuration and performs the initial code generation.
*   **Arguments:**
    *   `<name>`: The name of the library to create (e.g., `google-cloud-secretmanager`).
    *   `[apis...]`: One or more API paths (e.g., `google/cloud/secretmanager/v1`) that define the channels for this library. These are looked up in `catalog.yaml`.

### `librarian generate`
**Usage:** `librarian generate [<name> | --all] [flags]`
*   **Purpose:** Regenerates the code for managed libraries using the current configuration and sources.
*   **Arguments:**
    *   `<name>`: (Optional) The name of a specific library to regenerate. If omitted, `--all` must be used.
*   **Flags:**
    *   `--all`: Regenerate *all* libraries listed in `librarian.yaml`. Exclusive with `<name>` argument.
    *   `--check`: Verify that the generated code matches the current configuration without modifying files. Returns a non-zero exit code if changes are detected.

### `librarian update`
**Usage:** `librarian update [<source> | --all] [flags]`
*   **Purpose:** Updates the internal state or global dependencies,
such as the `googleapis` commit hash in `librarian.yaml`.
*   **Arguments:**
    *   `<source>`: (Optional) The name of a specific source to update (e.g., `googleapis`, `protobuf`). If omitted, `--all` must be used.
*   **Flags:**
    *   `--all`: Update *all* global sources to their latest valid versions. Exclusive with `<source>` argument.

### `librarian release`
**Usage:** `librarian release [<name> | --all] [flags]`
*   **Purpose:** Prepares libraries for release.
This typically involves calculating the next semantic version,
updating `CHANGELOG.md`, and bumping versions in manifest files (e.g., `Cargo.toml`).
*   **Arguments:**
    *   `<name>`: (Optional) The name of a specific library to prepare for release. If omitted, `--all` must be used.
*   **Flags:**
    *   `--all`: Prepare release for *all* libraries that have changes. Exclusive with `<name>` argument.
    *   `--skip-semver-checks`: Skip semantic version compliance checks (e.g., `cargo semver-checks`) during release preparation. Use with caution.

### `librarian publish`
**Usage:** `librarian publish [<name> | --all] [flags]`
*   **Purpose:** Uploads the prepared artifacts to the package registry.
*   **Arguments:**
    *   `<name>`: (Optional) The name of a specific artifact to publish. If omitted, `--all` must be used.
*   **Flags:**
    *   `--all`: Publish *all* released artifacts. Exclusive with `<name>` argument.