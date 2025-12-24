# Generation Workflow

This document outlines the standard workflow for generating and updating client library code managed by `librarian`.

## Goal
To keep client libraries up-to-date with upstream API definitions (Protos or Discovery documents) and generator logic.

## Prerequisites

1.  **Repository Configuration**: `librarian.yaml` is correctly configured with `generation` settings and a `libraries` list.
2.  **Tooling Installed**: The `librarian` CLI tool is installed.
3.  **Clean State**: It is recommended to start from a clean Git state on a fresh branch.

## Generation Workflows

### 1. Routine Updates (Protos / Discovery Docs)
This workflow is used when upstream API definitions (`googleapis/googleapis` or Discovery docs) have changed, and you need to update your client libraries to reflect those changes.

1.  **Create a Branch:**
    ```bash
    git checkout -b chore/update-deps-$(date +%Y-%m-%d)
    ```

2.  **Update Global Sources:**
    Use `librarian update` to fetch the latest commit hashes for your configured sources (e.g., `googleapis`). This updates `librarian.yaml`.
    ```bash
    librarian update --all
    ```

3.  **Regenerate Code:**
    Run `librarian generate` to rebuild all libraries using the new source versions.
    ```bash
    librarian generate --all
    ```

4.  **Format Code:**
    Run your language-specific formatter (e.g., `cargo fmt`, `black .`).

5.  **Commit and PR:**
    ```bash
    git add .
    git commit -m "chore: update upstream dependencies"
    git push -u origin HEAD
    # Open PR against main
    ```

### 2. Generator Logic Updates
When the underlying code generator (e.g., the template logic inside the `librarian` tool or its docker images) is updated, you need to refresh the generated code without changing the upstream sources.

1.  **Create a Branch:**
    ```bash
    git checkout -b chore/refresh-generator
    ```

2.  **Regenerate Code:**
    Simply run generation without updating sources.
    ```bash
    librarian generate --all
    ```

3.  **Commit and PR:**
    ```bash
    git commit -am "chore: regenerate code with latest generator"
    ```

### 3. Targeted Regeneration
Sometimes you only want to regenerate a specific library (e.g., to test a fix or because the generator is slow).

1.  **Regenerate Single Library:**
    ```bash
    librarian generate google-cloud-secretmanager
    ```
    *   *Note:* This uses the global sources defined in `librarian.yaml`.

### 4. Handling "Dirty" States (Advanced)
If you need to regenerate *only* one library but update the global source (e.g., for a [Hotfix](freeze.md)), be aware that this leaves the repository in a "dirty" state where `librarian.yaml` source versions might not match the generated code of other libraries. This is acceptable for temporary hotfix branches but should be resolved by a full `librarian generate --all` when merging back to main or lifting a freeze.

## Validation

After any generation step, always validate the changes:

1.  **Diff Check:** Ensure the changes in generated code look correct and expected.
2.  **Compilation/Build:** Run the language build tool (e.g., `cargo build`, `python setup.py build`).
3.  **Tests:** Run unit and integration tests.
