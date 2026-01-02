# Hotfix Workflow

### Objective
This document outlines the recommended workflow for handling hotfixes using the new Librarian CLI, emphasizing the separation of concerns between library management and Git operations.

### Background
In the previous Librarian CLI design, a `--branch` flag was necessary. This flag served a dual purpose: it instructed the tool which remote branch to clone as its working basis and which branch to target as the base for automated pull requests. This was due to the old CLI acting as an orchestrator for both library generation and Git workflows.

With the evolution of the Librarian CLI, the design philosophy shifted towards adherence to the Unix philosophy, where each tool performs a single, well-defined task. The new CLI is designed to operate within an existing Git repository, focusing solely on the internal state and configuration of the library. Consequently, the explicit `--branch` flag is no longer part of the `librarian` command arguments.

### Overview
The new hotfix strategy leverages standard Git commands for version control (branching, checking out, merging, creating pull requests) and dedicates the Librarian CLI to its core function: managing the client library's configuration, generation, and release processes. This approach enhances flexibility, transparency, and aligns with common developer workflows.

### Detailed Design

Consider a scenario where a critical bug is discovered in a production version (e.g., `v1.5.0`), corresponding to a stable release branch named `release-v1.5`, while active development is ongoing in the `main` branch for `v2.0.0`. The goal is to release `v1.5.1` with the hotfix.

The hotfix workflow would be executed as follows:

1.  **Checkout the Stable Release Branch**: The user or CI/CD system first switches to the stable branch that requires the hotfix.
    ```bash
    git checkout release-v1.5
    git pull origin release-v1.5
    ```

2.  **Create a Dedicated Hotfix Branch**: A new branch is created from the stable release branch for the hotfix work.
    ```bash
    git checkout -b hotfix/bug-123-release-v1.5
    ```

3.  **Apply Fixes and Run Librarian Commands**: Manual code changes are applied, or configuration updates are made to address the bug. After these changes, the relevant Librarian commands are executed to regenerate code, update versions, or prepare release artifacts.
    ```bash
    # (Developer applies necessary code or configuration changes, e.g., updating an API source)

    # Example: Update external dependencies if the fix comes from an updated source
    librarian update googleapis

    # Regenerate the specific library affected by the hotfix
    librarian generate google-cloud-affected-service

    # Prepare the patch release, which will calculate and bump the version (e.g., to v1.5.1)
    librarian release google-cloud-affected-service
    ```
    At this stage, all generated code, manifest updates, and version bumps are correctly applied to the codebase of the `hotfix/bug-123-release-v1.5` branch, which is based on `release-v1.5`.

4.  **Commit and Push Changes**: The developer or CI/CD system commits the changes and pushes the hotfix branch to the remote repository.
    ```bash
    git add .
    git commit -m "fix(affected-service): resolve critical bug 123 in v1.5"
    git push origin hotfix/bug-123-release-v1.5
    ```

5.  **Create a Pull Request**: A pull request is created against the original stable release branch (`release-v1.5`) on the Git hosting platform (e.g., GitHub, GitLab, Bitbucket). The base branch of this PR is explicitly set to `release-v1.5`.
    This step is external to the Librarian CLI and is typically performed manually or by a CI/CD script that interacts with the Git hosting service's API.

6.  **Review, Merge, and Tag**: The pull request is reviewed, merged into `release-v1.5`, and a new release tag (e.g., `v1.5.1`) is created.

### Alternatives Considered

The primary alternative was the old CLI design where a `--branch` flag handled both cloning and setting the base for pull requests. This approach was rejected in favor of the current design due to its inherent inflexibility, reduced transparency, and the tight coupling between library management and Git workflow orchestration. The new design promotes a clearer separation of concerns, making the CLI more predictable, composable, and easier to integrate into diverse CI/CD environments and developer workflows.
