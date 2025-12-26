# Release Workflow

This document outlines the standard workflow for preparing and publishing client library releases for repositories managed by `librarian`, primarily driven by `librarianops` automation, but fully supporting manual developer workflows.

## Goal
To reliably update library versions, generate release artifacts (e.g., changelogs), and publish them to their respective package registries (e.g., PyPI, crates.io), with flexible manual override capabilities.

## Prerequisites

1.  **Repository Configuration**: `librarian.yaml` is correctly configured for all libraries intended for release.
2.  **Code Freshness**: All desired code changes for the release (including generated code) are merged into `main`.
3.  **Tooling Operational**: The `librarianops` service (or CLI) and `librarian` CLI are installed and accessible.

## Release Workflow (Automation-Driven)

### 1. `librarianops release` Execution (Platform Team / Automation)

A Platform Team member (or scheduled `librarianops` automation) runs the `librarianops release` command.

*   **Action:**
    1.  `librarianops` iterates through all configured language repositories.
    2.  For each repository, it checks for detected changes in libraries that are eligible for release (i.e., `release: true` in `librarian.yaml`).
    3.  For eligible libraries, it creates a dedicated release branch (e.g., `release/google-cloud-secretmanager-v1.2.1`).
    4.  It executes `librarian release --all` (or `librarian release <name>`) within that repository.
        *   **Staggered Rollouts:** If a massive update has occurred (e.g., updating the generator image affecting 200+ libraries), `librarianops` can be configured to use `librarian release --all --limit 10`. This processes only the first 10 pending libraries, creating a manageable batch for release and minimizing the blast radius of potential issues.
        *   **Version Calculation**: Determines the next semantic version for each eligible library.
        *   **Config Update**: Updates the `version` field for relevant libraries in `librarian.yaml`.
        *   **Manifest Update**: Bumps versions in language-specific package manifest files (e.g., `Cargo.toml` for Rust, `setup.py` for Python).
        *   **Changelog Generation**: Updates `CHANGELOG.md` files (global or per-library) with new release entries.
    5.  It executes `librarian generate --all` to refresh any generated artifacts (e.g., READMEs) that embed version numbers or other release data.
    6.  It creates or updates a Pull Request (PR) in the language repository, proposing the release preparation changes.

### 2. Language Team Review and Merge

The respective Language Team reviews the automatically generated Release PR.

*   **Action:** The language team verifies the version bumps, changelog entries, and ensures all tests pass.
*   **Outcome:** Merging the PR into `main` finalizes the release preparation in the repository.

### 3. `librarian publish` Execution (CI/CD Automation)

After the Release PR is merged into `main`, a CI/CD job (typically triggered by the merge event or a new Git tag) will automatically run the `librarian publish` command.

*   **Action:**
    1.  The CI/CD pipeline runs `librarian publish --all` (or `librarian publish <name>`).
    2.  **Artifact Upload**: Uploads the finalized package artifacts (e.g., `.crate` files to crates.io, `.whl` files to PyPI) to their respective registries.
    3.  **Tagging**: Creates a Git tag (e.g., `google-cloud-secretmanager/v1.2.1` based on `release.tag_format`) on the `main` branch, pointing to the merged commit.
    4.  **Verification**: The CI/CD pipeline performs post-publish checks.

### 4. Post-Release Verification

*   **Action:** The Platform Team and Language Teams verify that the new versions are available in the respective package registries and that the Git tags have been created correctly.

## Manual Release Workflows (Developer-Driven)

Language team members can run `librarian release` directly for local testing, debugging, or managing highly specific release scenarios (e.g., a targeted hotfix).

### 1. Local Release Preparation

1.  **Create a Release Branch:**
    ```bash
git checkout -b release/v1.2.3-manual
    ```
2.  **Run `librarian release`:**
    ```bash
librarian release google-cloud-secretmanager # For a specific library
# OR
librarian release --all # For all libraries
    ```
3.  **Regenerate Code & Format (if needed):**
    ```bash
librarian generate --all # If READMEs/other generated files depend on new versions
cargo fmt # Or language-specific formatter
    ```
4.  **Review, Commit, and Open PR:** Review changes, commit, and open a PR to `main`.

### 2. Manual Publish (for Debugging/Override)

In rare cases, a manual `librarian publish` might be needed (e.g., re-publishing a corrupted artifact, though this should be avoided).

1.  **Ensure Prepared State:** Make sure the library is in a released state (versions bumped, changelogs updated).
2.  **Run Publish:**
    ```bash
librarian publish google-cloud-secretmanager
    ```
