# Python Implementation Design for Librarian

## Objective
This document defines the Python-specific implementation, workflows, and command behaviors for the `librarian` CLI.

## Background
The `librarian` CLI is a language-agnostic tool for managing Google Cloud client libraries. For a high-level overview of the CLI's design, refer to [design/cli.md](./../cli.md), and for a detailed explanation of the configuration architecture, see [design/configuration.md](./../configuration.md). However, to function correctly, it requires language-specific logic to interface with each language's unique ecosystem. This document details that specific implementation for the Python ecosystem, which is centered around `pip`, PyPI, and virtual environments.

## Overview
The Python implementation for `librarian` is designed around a "single source of truth" principle, where the `librarian.yaml` manifest is the authoritative source for all configuration. The `librarian` Go binary acts as a smart **orchestrator**, delegating language-specific tasks to a toolkit of standard Python tools.

Package metadata files like `setup.py` and `__init__.py` are treated as generated artifacts, ensuring that all metadata is derived directly from `librarian.yaml`. The workflow is an explicit sequence of commands executed by `librarian` (e.g., `protoc`, `black`, `isort`, `twine`), making the process transparent, debuggable, and idiomatic for Python developers.

## Detailed Design

### Library Naming and Path Inference
-   **Library Naming:** Python libraries follow the `google-cloud-<service>` naming convention (e.g., `google-cloud-secret-manager`). The `-` is a significant separator.
-   **Path Inference:** When using `librarian add`, the tool can infer the `api_path` from the library name. It does this by replacing `-` with `/` and taking the last two segments. For example, `google-cloud-secret-manager` would suggest an inference of `secretmanager/v1` if a matching entry exists in `sdk.yaml`.

### `librarian add`
-   **Functionality:** Adds a new library entry to `librarian.yaml`.
-   **Python-Specifics:**
    -   Supports adding multiple API versions (`channels`) to a single library entry, which is a common pattern in Python libraries.
    -   The `output` directory, if not specified, defaults to `packages/<library-name>`.

### `librarian generate`
-   **Functionality:** Orchestrates a sequence of tools to produce client library code and package metadata. `librarian` executes all commands within the appropriate toolkit container. It does not manage virtual environments itself; it runs commands like `python3`, `protoc`, `black`, and `isort` directly, relying on the container's pre-configured `PATH`.
-   **Python-Specifics:**
    1.  **Pre-generation Cleanup:** `librarian` cleans the output directory. Crucially, it reads the `keep` field from `librarian.yaml` and preserves the specified files and directories (e.g., `noxfile.py`, handwritten samples) before deleting the rest.
    2.  **Code Generation via `protoc`:** `librarian` executes `protoc` with the `--python_gapic_out` plugin. The Go binary is responsible for reading `librarian.yaml` and constructing the correct command-line arguments, including transport options and service configs.
    3.  **Manifest & Metadata Generation:** `librarian` executes a dedicated Python script provided by the toolkit (e.g., `python3 -m synthtool.manifest_generator`). The Go binary passes metadata from `librarian.yaml` (like the library name, version, and dependencies) to this script via command-line arguments. This keeps the Python packaging logic in a maintainable Python script.
    4.  **Code Formatting:** After code and manifests are generated, `librarian` explicitly executes standard Python formatters on the output directory, in sequence: first `black .`, then `isort .`. This makes the formatting step transparent and auditable.

### `librarian stage`
-   **Functionality:** Prepares a new release by calculating the next version and updating Python package files. This command modifies local files, which are then expected to be committed.
-   **Python-Specifics:**
    1.  **Version Calculation:** Analyzes the library's conventional commit history since the last recorded release to determine the next semantic version.
    2.  **Update `librarian.yaml`:** Updates the `version` field for the specific library within the `librarian.yaml` manifest.
    3.  **File Updates:** `librarian` delegates the propagation of the new version into Python-specific files (e.g., `setup.py`, `__init__.py`) by executing a dedicated versioning script from the toolkit.

### `librarian tag`
-   **Functionality:** Creates and pushes the Git tag for a staged Python package.
-   **Python-Specifics:**
    1.  **Change Detection:** Identifies which libraries are candidates for tagging by finding which library versions in `librarian.yaml` have been updated since the last release tag.
    2.  **Tagging:** Creates and pushes a Git tag to the repository, formatted according to the `tag_format` specified in `librarian.yaml` (e.g., `google-cloud-secret-manager/v2.25.0`).

### `librarian publish`
-   **Functionality:** Publishes a tagged Python package to PyPI.
-   **Python-Specifics:**
    1.  **Change Detection:** Identifies which libraries are candidates for publishing based on the presence of a new release tag.
    2.  **Build:** `librarian` orchestrates the standard Python build process by executing `python3 setup.py sdist bdist_wheel` in the library's output directory.
    3.  **Publish:** `librarian` executes `twine upload dist/*` to upload the built artifacts to the Python Package Index (PyPI).

## Alternatives Considered
(This section can be filled in as the design evolves.)