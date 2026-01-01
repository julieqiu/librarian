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

The release workflow handles version bumping and changelog generation.

```bash
# 1. Prepare Release
librarian release google-cloud-secretmanager

# 2. Verify
cargo check
cargo test

# 3. Publish (Usually CI only)
librarian publish google-cloud-secretmanager
```

Tips
----

-	**Formatting**: `librarian generate` automatically runs `cargo fmt`. You don't need to run it manually.
-	**Veneers**: If you are working on a library with handwritten code ("veneer"), ensure your changes in `src/` don't conflict with the `generated/` directory structure.
