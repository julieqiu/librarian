# Librarian CLI Specification

This document describes the command-line interface for the new `librarian` tool,
which replaces the legacy `sidekick` utility.
The design emphasizes consistency and clear resource management verbs (Create,
Generate, Update, Release, Publish).

## Command Reference

### `librarian create`
**Usage:** `librarian create --library <name> [flags]`
*   **Purpose:** Adds a new library to the `librarian.yaml` configuration and performs the initial code generation.
*   **Flags:**
    *   `--library`: The name of the library to create (e.g., `google-cloud-secretmanager`).
    *   `--source`: (Optional) Override the default API source path.

### `librarian generate`
**Usage:** `librarian generate [flags]`
*   **Purpose:** Regenerates the code for managed libraries using the current configuration and sources.
*   **Flags:**
    *   `--library <name>`: Regenerate *only* the specified library.
    *   `--all`: Regenerate *all* libraries listed in `librarian.yaml` (default behavior might be all, or require this flag for safety).

### `librarian update`
**Usage:** `librarian update [flags]`
*   **Purpose:** Updates the internal state or global dependencies,
such as the `googleapis` commit hash in `librarian.yaml`.
*   **Flags:**
    *   `--all`: Update all global sources to their latest valid versions.

### `librarian release`
**Usage:** `librarian release [flags]`
*   **Purpose:** Prepares libraries for release.
This typically involves calculating the next semantic version,
updating `CHANGELOG.md`, and bumping versions in manifest files (e.g., `Cargo.toml`).
*   **Flags:**
    *   `--library <name>`: Prepare release for *only* the specified library.
    *   `--all`: Prepare release for all libraries that have changes.

### `librarian publish`
**Usage:** `librarian publish [flags]`
*   **Purpose:** Uploads the prepared artifacts to the package registry.
*   **Flags:**
    *   `--dry-run`: Perform all checks but do not actually upload.