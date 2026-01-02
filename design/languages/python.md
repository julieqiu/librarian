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

### `librarian stage`
-   **Functionality:** Prepares a new release by calculating the next version and updating Python package files. This command modifies local files, which are then expected to be committed.
-   **Python-Specifics:**
    1.  **Version Calculation:** Analyzes the library's conventional commit history since the last recorded release to determine the next semantic version.
    2.  **Update `librarian.yaml`:** Updates the `version` field for the specific library within the `librarian.yaml` manifest.
    3.  **File Updates:** Propagates the new version into key generated Python package files, including `setup.py`, `gapic_version.py`, and `samples/**/snippet_metadata.json`.

### `librarian tag`
-   **Functionality:** Creates and pushes the Git tag for a staged Python package.
-   **Python-Specifics:**
    1.  **Change Detection:** Identifies which libraries are candidates for tagging by finding which library versions in `librarian.yaml` have been updated since the last release tag.
    2.  **Tagging:** Creates and pushes a Git tag to the repository, formatted according to the `tag_format` specified in `librarian.yaml` (e.g., `google-cloud-secret-manager/v2.25.0`).

### `librarian publish`
-   **Functionality:** Publishes a tagged Python package to PyPI.
-   **Python-Specifics:**
    1.  **Change Detection:** Identifies which libraries are candidates for publishing based on the presence of a new release tag.
    2.  **Build:** Builds the source distribution (`sdist`) and wheel (`bdist_wheel`) using `setup.py`.
    3.  **Publish:** Uses `twine` to upload the built artifacts to the Python Package Index (PyPI).

## Alternatives Considered
(This section can be filled in as the design evolves.)