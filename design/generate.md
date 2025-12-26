# Generation Workflow

This document outlines the standard workflow for generating and updating
client library code managed by `librarian`,
primarily driven by `librarianops` automation,
but fully supporting manual developer workflows.

## Goal
To continuously keep client libraries up-to-date with upstream API definitions
(Protos or Discovery documents) and generator logic,
ensuring code freshness and consistency across all managed repositories,
with flexible manual override capabilities.

## Prerequisites

1.  **Repository Configuration**: `librarian.yaml` is correctly configured in each language repository.
2.  **Tooling Installed**: The `librarian` CLI tool is installed.
3.  **Clean State**: It is recommended to start from a clean Git state on a fresh branch.
4.  **API Sources Synced**: `generation.sources` in `librarian.yaml` are
up-to-date (managed by `librarianops sync-sources` or manually updated).

## Automated Generation Workflow (`librarianops generate-all`)

This workflow is the primary mechanism for updating generated code across the ecosystem.
It is typically triggered by `librarianops` (Platform Team automation) after
upstream source updates or new generator logic is merged.

### 1. `librarianops generate-all` Execution

A platform team member (or scheduled `librarianops` automation) runs the `librarianops generate-all` command.

*   **Action:**
    1.  `librarianops` iterates through all configured language repositories.
    2.  For each repository, it creates a bot branch (e.g., `owlbot/generate-code-<timestamp>`).
    3.  It executes `librarian generate --all` within that repository, regenerating code for all libraries as defined in its `librarian.yaml`.
    4.  It runs language-specific formatters (e.g., `cargo fmt` for Rust, `black . && isort .` for Python).
    5.  It creates or updates a Pull Request (PR) in the language repository, proposing the generated code changes.

### 2. Language Team Review and Merge

The respective Language Team reviews the automatically generated PR.

*   **Action:** The language team verifies the generated code,
runs tests, and ensures no regressions or unexpected changes are introduced.
*   **Outcome:** Merging the PR integrates the updated generated code into the `main` branch.

## Manual Generation Workflows (Developer-Driven)

Language team members can run `librarian generate` whenever needed for local development,
debugging, or specific iteration cycles.

### 1. Local Iteration on Generator Logic / Proto Changes
When developing changes to the `librarian` generator itself,
or working with local proto changes, you can refresh code directly on your development branch.

1.  **Create a Branch:** Develop on a feature branch.
2.  **Run Generation:** Use `librarian generate --all` (or targeted) locally
to refresh code with your new generator logic or local proto changes.
    ```bash
librarian generate --all
# Or for a specific library:
librarian generate google-cloud-secretmanager
    ```
3.  **Verify & Iterate:** Test the generated code,
commit changes, and repeat until the generator logic/proto changes are correct.

### 2. Targeted Regeneration (Single Library)
For quick local checks or debugging, you can regenerate only a specific library.

1.  **Run Generation:**
    ```bash
librarian generate google-cloud-secretmanager
    ```
    *   *Note:* This uses the global sources defined in `librarian.yaml`.

### 3. Handling "Dirty" States (Advanced)
As described in [Freeze Workflow](freeze.md),
regenerating *only* one library but updating the global source (e.g.,
for a hotfix) leaves a "dirty" state where `librarian.yaml` source versions
might not match the generated code of other libraries.
This is acceptable for temporary hotfix branches but should be resolved
by a full `librarian generate --all` when merging back to `main` or lifting a freeze.

## Validation

After any generation step (automated or manual), always validate the changes:

1.  **Diff Check:** Ensure the changes in generated code look correct and expected.
2.  **Compilation/Build:** Run the language build tool (e.g., `cargo build`, `python setup.py build`).
3.  **Tests:** Run unit and integration tests.
