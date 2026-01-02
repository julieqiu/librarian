# Librarian CLI Specification

This document outlines the command-line interface and core workflows for `librarian`, a tool for managing Google Cloud client libraries.

## Commands

-   **`add`**: Onboards a new client library.
-   **`generate`**: Generates client library code.
-   **`update`**: Updates external source dependencies.
-   **`stage`**: Prepares a release by bumping versions in local files.
-   **`tag`**: Creates and pushes a Git tag for a staged release.
-   **`publish`**: Publishes a tagged release to a public registry.
-   **`tidy`**: Formats and validates `librarian.yaml`.
-   **`status`**: Checks the health and readiness of libraries.
-   **`check`**: Runs a suite of quality and validation checks.
-   **`version`**: Prints the `librarian` version.
-   **`delete`**: Removes a client library.

## Workflows

### `librarian stage`

The `stage` command prepares a library for a new release. It is the first step in the release process. Its primary responsibility is to determine the correct next semantic version and update all necessary files within the local repository to reflect that new version. It does not create Git tags or interact with remote repositories.

Its workflow is as follows:
1.  It analyzes the Git commit history since the last release tag to identify changes (`feat`, `fix`, `BREAKING CHANGE`).
2.  Based on the commit history, it calculates the next semantic version for each library that has changed.
3.  It updates the `version` field for each library in the `librarian.yaml` manifest.
4.  It propagates the new version into all language-specific package metadata files, preparing them to be committed.

### `librarian tag`

The `tag` command creates the official Git tag for a release. It is the second step in the release process, following `librarian stage` and a `git commit`. This command's only responsibility is to create and push the version-specific Git tag to the remote repository.

Its workflow is as follows:
1.  It identifies which libraries have staged changes by finding which versions in `librarian.yaml` have been updated since the last Git tag.
2.  For each candidate library, it creates and pushes a new Git tag (e.g., `secretmanager/apiv1/v1.5.0`), pointing to the current commit.

### `librarian publish`

The `publish` command executes the final step of a release by uploading artifacts to a public package registry. It follows `librarian tag`.

Its workflow is as follows:
1.  It identifies which libraries are candidates for publishing based on the presence of a new release tag.
2.  It orchestrates the language-specific publishing process:
    *   **Python:** Builds the source and wheel distributions and uploads them to PyPI using `twine`.
    *   **Rust:** Runs `cargo workspaces publish` to upload packages to `crates.io`.
    *   **Go:** This command is a no-op for Go, as the `librarian tag` command is the publishing mechanism for Go modules.

### `librarian check`

The `check` command runs a suite of quality assurance and validation checks for a specific client library. This command is designed to provide a consolidated way for developers to verify their code against project standards without manually executing multiple language-specific tools.

Its workflow is as follows:
1.  It takes a library name as an argument and navigates to its output directory.
2.  It identifies the library's language and orchestrates the appropriate set of validation tools.
    *   **For Rust:**
        -   Executes `cargo test` to run unit and integration tests.
        -   Executes `cargo doc --workspace --no-deps` to check documentation for warnings and errors.
        -   Executes `cargo clippy --workspace -- -D warnings` to run the linter with strict warning policies.
    *   **For Python:**
        -   Executes `nox -s test` to run the test suite defined in `noxfile.py`.
    *   **For Go:**
        -   Executes `go test ./...` to run all tests within the module.
        -   Executes `go vet ./...` to run the Go vet linter for suspicious constructs.