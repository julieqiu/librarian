# Python Implementation Design for Librarian

## Objective
This document defines the Python-specific implementation, workflows, and command behaviors for the `librarian` CLI.

## Background
The `librarian` CLI is a language-agnostic tool for managing Google Cloud client libraries. For a high-level overview of the CLI's design, refer to [design/cli.md](./../cli.md), and for a detailed explanation of the configuration architecture, see [design/configuration.md](./../configuration.md). However, to function correctly, it requires language-specific logic to interface with each language's unique ecosystem. This document details that specific implementation for the Python ecosystem, which is centered around `pip`, PyPI, and virtual environments.

## Overview
The Python implementation for `librarian` is designed around a "single source of truth" principle, where the `librarian.yaml` manifest is the authoritative source for all configuration. Package metadata files like `setup.py` and `__init__.py` are treated as generated artifacts, ensuring that all metadata is derived directly from `librarian.yaml`.

The workflow is orchestrated through a series of `librarian` commands that wrap and delegate to standard Python tooling like `gcp-synthtool`, `pip`, `twine`, `black`, and `isort`. This approach ensures that the process is both idiomatic for Python developers and robust enough to handle the complexities of Python packaging.

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
-   **Functionality:** Orchestrates the Python GAPIC generator to produce client library code and package metadata.
-   **Python-Specifics:**
    1.  **Environment:** `librarian` invokes the Python generator within a virtual environment, using the Python version and dependencies specified in `tool.yaml`.
    2.  **Manifest Generation:** Key package files like `setup.py` and version files (e.g., `__init__.py`) are generated from templates. All metadata, including the package name, version, and dependencies, are populated from the configuration in `librarian.yaml`.
    3.  **Generator Execution:** It executes `gcp-synthtool`, passing in the appropriate API protos and service configuration to generate the raw `.py` source files.
    4.  **Post-Processing:** After generation, `librarian` runs standard Python formatters like `black` and `isort` on the generated code to ensure it conforms to `google-cloud-python` style guides.
    5.  **`keep` field:** The `keep` field in `librarian.yaml` is respected. Files listed here (e.g., `noxfile.py`, handwritten samples) are preserved and are not deleted during the pre-generation cleanup of the output directory.

### `librarian release`
-   **Functionality:** Prepares a new release by calculating the next version and updating Python package files to reflect that version. This command modifies local files, which are then expected to be committed before being published with `librarian publish`.
-   **Python-Specifics:**
    1.  **Version Calculation:**
        *   The system first reads the current version of the library from its entry within the `librarian.yaml` manifest.
        *   It then analyzes the library's conventional commit history since the last recorded release. Based on the types of changes (e.g., `feat` for minor, `fix` for patch, `BREAKING CHANGE` for major), it determines the appropriate next semantic version (e.g., 1.0.0 -> 1.0.1 or 1.1.0).
    2.  **Update `librarian.yaml`:**
        *   Once the next semantic version has been calculated, the version field for the specific library within the `librarian.yaml` manifest is updated to this newly determined version. This ensures that `librarian.yaml` remains the authoritative "single source of truth" for the library's version.
    3.  **File Updates:**
        *   The newly calculated version is then propagated and written into several key generated Python package files to ensure consistency across the library's distribution. This includes:
            *   **Version Definition Files:** Updates the `__version__` variable in `gapic_version.py` or `version.py` files (e.g., `my_library/v1/gapic_version.py`). If present, it also updates the `__release_date__` variable to the current date.
            *   **Project Metadata Files:** Modifies the version field within `setup.py` or `pyproject.toml`, which are used for packaging and distribution.
            *   **Snippet Metadata Files:** Updates the `clientLibrary.version` field in `samples/**/snippet_metadata.json` files, ensuring sample code metadata reflects the correct client library version.
    4.  **Validation:** It is expected that project-specific CI pipelines will run `pytest` or `nox` as a separate validation step. `librarian release` focuses solely on versioning and file updates.

### `librarian publish`
-   **Functionality:** Publishes packages that have changed since the last release tag.
-   **Python-Specifics:**
    1.  **Change Detection:** It identifies which libraries are candidates for publishing by finding which library versions in `librarian.yaml` have been updated since the last release tag (using `git diff`).
    2.  **Build:** For each candidate library, it builds the source distribution (`sdist`) and wheel (`bdist_wheel`) using `setup.py`.
    3.  **Publish:** It uses `twine` to upload the built artifacts to the Python Package Index (PyPI).
    4.  **Tagging:** After a successful publish, it creates and pushes a Git tag to the repository, formatted according to the `tag_format` specified in `librarian.yaml` (e.g., `google-cloud-secret-manager/v2.25.0`).

## Alternatives Considered
(This section can be filled in as the design evolves.)