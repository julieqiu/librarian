# Release Workflow

This document outlines the standard workflow for preparing and publishing releases for client libraries managed by `librarian`.

## Goal
To reliably update library versions, generate release artifacts (e.g., changelogs), and publish them to their respective package registries (e.g., PyPI, crates.io).

## Prerequisites

1.  **Repository Configuration**: `librarian.yaml` is correctly configured for all libraries intended for release.
2.  **Clean State**: The repository is in a clean state, with all desired code changes merged into `main`.
3.  **Tooling Installed**: The `librarian` CLI tool is installed and accessible.

## Release Workflow

### 1. Create a Release Branch
All release preparations should occur on a dedicated release branch, isolating changes until they are ready for merge.

```bash
git checkout -b release/v<next-version-or-date> # e.g., release/v1.2.3 or release/2025-01-01
```

### 2. Prepare Release Artifacts (`librarian release`)
Use `librarian release` to calculate new versions, update `librarian.yaml`, and generate release-related files like `CHANGELOG.md` and package manifests (e.g., `Cargo.toml`, `setup.py`).

```bash
# To prepare all libraries that have detected changes:
librarian release --all

# To prepare a specific library (e.g., for a targeted hotfix):
librarian release google-cloud-secretmanager
```

*   **Actions by `librarian release`:**
    1.  **Version Calculation**: Determines the next semantic version for each eligible library based on detected changes and current version in `librarian.yaml`.
    2.  **Config Update**: Updates the `version` field for relevant libraries in `librarian.yaml`.
    3.  **Manifest Update**: Bumps versions in language-specific package manifest files (e.g., `Cargo.toml` for Rust, `setup.py` for Python).
    4.  **Changelog Generation**: Updates `CHANGELOG.md` files (global or per-library, as configured) with new release entries.
    5.  **Generated Code Refresh (Optional but Recommended)**: After version bumps and changelog updates, some generated code (e.g., READMEs that embed version numbers) might need to be refreshed. A subsequent `librarian generate --all` ensures consistency.

### 3. Regenerate Code (if necessary)
As noted above, if release preparation (e.g., updating `README.md` based on new versions) requires regenerating parts of the code, run a full generation step.

```bash
librarian generate --all
# Or for a specific library:
librarian generate google-cloud-secretmanager

# Also run language-specific formatters
cargo fmt # for Rust
# black . && isort . # for Python
```

### 4. Review and Commit Changes
Review all changes made by `librarian release` and `librarian generate`.

```bash
git status
git diff
```

Commit the changes to the release branch. The commit message should clearly indicate this is a release preparation.

```bash
git commit -m "chore: prepare for v1.2.3 release"
```

### 5. Open Pull Request and Merge

Push your release branch and open a Pull Request targeting the `main` branch. The PR should be reviewed and merged.

### 6. Publish Release Artifacts (`librarian publish`)

After the release branch is merged into `main`, a CI/CD job (triggered by the merge or a new tag) will run the `librarian publish` command.

```bash
# This command is typically run by automation in a CI/CD pipeline
librarian publish --all

# Or for a specific library:
librarian publish google-cloud-secretmanager
```

*   **Actions by `librarian publish`:**
    1.  **Artifact Upload**: Uploads the finalized package artifacts (e.g., `.crate` files to crates.io, `.whl` files to PyPI).
    2.  **Tagging**: Creates a Git tag (e.g., `google-cloud-secretmanager/v1.2.1`) on the `main` branch, as configured by `release.tag_format`.

### 7. Post-Release

Verify that the new versions are available in the respective package registries and that the Git tags have been created correctly.
