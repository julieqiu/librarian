# Python Implementation for Librarian

This document details the Python-specific behavior and implementation of the `librarian` CLI commands.

## Library Naming and Path Inference
-   **Library Naming:** Python libraries follow the `google-cloud-<service>` naming convention (e.g., `google-cloud-secret-manager`). The `-` is a significant separator.
-   **Path Inference:** When using `librarian add`, the tool can infer the `api_path` from the library name. It does this by replacing `-` with `/` and taking the last two segments. For example, `google-cloud-secret-manager` would suggest an inference of `secretmanager/v1` if a matching entry exists in `sdk.yaml`.

## Command-Specific Behavior

### `librarian add`
-   **Functionality:** Adds a new library entry to `librarian.yaml`.
-   **Python-Specifics:**
    -   Supports adding multiple API versions (`channels`) to a single library, which is a common pattern in Python libraries.
    -   The `output` directory, if not specified, defaults to `packages/<library-name>`.

### `librarian generate`
-   **Functionality:** Orchestrates the Python GAPIC generator to produce client library code.
-   **Python-Specifics:**
    1.  **Environment:** `librarian` invokes the Python generator within a virtual environment defined by `tool.yaml`.
    2.  **Generator Execution:** It executes `gcp-synthtool` with the appropriate API protos and service configuration.
    3.  **Post-Processing:** After generation, `librarian` runs standard Python formatters like `black` and `isort` on the generated code to ensure it conforms to `google-cloud-python` style guides.
    4.  **`keep` field:** The `keep` field in `librarian.yaml` is respected. Files listed here (e.g., `noxfile.py`, handwritten samples) are preserved and are not deleted during the pre-generation cleanup of the output directory.

### `librarian release`
-   **Functionality:** Prepares a new release by updating version numbers.
-   **Python-Specifics:**
    1.  **Version Update:** The version number in `librarian.yaml` is incremented based on conventional commits.
    2.  **File Updates:** It may update version numbers in other files, such as `__init__.py` or `setup.py`, if required by the repository's conventions.

### `librarian publish`
-   **Functionality:** Publishes the prepared library package to a registry.
-   **Python-Specifics:**
    1.  **Build:** It builds the source distribution (`sdist`) and wheel (`bdist_wheel`) for the package using `setup.py`.
    2.  **Publish:** It uses `twine` to upload the built artifacts to the Python Package Index (PyPI).
    3.  **Tagging:** It creates and pushes a Git tag to the repository, formatted according to the `tag_format` specified in `librarian.yaml` (e.g., `google-cloud-secret-manager/v2.25.0`).