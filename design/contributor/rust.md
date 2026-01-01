Rust Contributor Guide
======================

This guide is intended for contributors to the `google-cloud-rust` repository.

Prerequisites
-------------

-	**Librarian CLI**: The latest `librarian` binary.
-	**Rust Toolchain**: `cargo`, `rustc`.
-	**Protoc**: Protocol Buffer compiler (v23.0+).

Workflows
---------

### Generate a New Library

```bash
git checkout -b feat-new-library
librarian create google-cloud-secretmanager google/cloud/secretmanager/v1
```

### Regenerate Libraries

Use this when you've updated `librarian.yaml` (e.g., bumped the `googleapis` commit).

```bash
librarian generate --all
# Or for a single crate:
librarian generate google-cloud-secretmanager
```

### Release

The release workflow is responsible for updating crate versions and preparing them for publication. It is primarily driven by the `librarian release` command.

**Versioning Workflow:**
1.  **`librarian release` Execution:** When you run `librarian release <crate-name>` (or `--all` for all crates):
    *   The tool reads the *current* version of the `<crate-name>` directly from its `Cargo.toml` file.
    *   It calculates the next semantic version based on conventional commits in the Git history.
    *   The new version is then written back to the `Cargo.toml` file, carefully preserving any comments or custom formatting.
    *   Crucially, this new version is *also* used to update the corresponding `version` field for the crate in `librarian.yaml`, ensuring this central configuration remains synchronized with the published crate's actual version.

2.  **Skipping Releases:** If a crate should not be released (e.g., it's a test utility or temporarily unstable), you can set `skip_release: true` in its entry within `librarian.yaml`. The `librarian release --all` command will automatically ignore such crates.

3.  **Verification (Separate Step):** After running `librarian release`, it is essential to perform comprehensive validation.
    *   `cargo check`: Ensure the code still compiles.
    *   `cargo test`: Run unit and integration tests to verify functionality.

4.  **Publish (Usually CI only):**
    ```bash
    librarian publish google-cloud-secretmanager
    ```

Tips
----

-	**Formatting**: `librarian generate` automatically runs `cargo fmt`. You don't need to run it manually.
-	**Veneers**: If you are working on a library with handwritten code ("veneer"), ensure your changes in `src/` don't conflict with the `generated/` directory structure.
