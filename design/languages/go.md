# Go Implementation Design for Librarian

## Objective
This document defines the Go-specific implementation, workflows, and command behaviors for the `librarian` CLI. It ensures consistency with the overall `librarian` design principles while adhering to idiomatic Go development practices observed in the `google-cloud-go` repository.

## Background
The `librarian` CLI is a language-agnostic tool for managing Google Cloud client libraries. For a high-level overview of the CLI's design, refer to [design/cli.md](./../cli.md), and for a detailed explanation of the configuration architecture, see [design/configuration.md](./../configuration.md). To function correctly, it requires language-specific logic to interface with each language's unique ecosystem. This document details that specific implementation for the Go ecosystem, which is centered around Go modules, `go.mod`, and the Go Proxy.

## Overview
The Go implementation for `librarian` operates on a "single source of truth" principle, where the `librarian.yaml` manifest is the authoritative source for all configuration. Go source files and version files (`internal/version.go`) are treated as generated or managed artifacts, ensuring that all metadata and content are derived directly from `librarian.yaml` and Git history.

Key Go-specific concepts from the legacy system, such as handling "hybrid" (handwritten) libraries and managing shared, repository-level files, will be represented in `librarian.yaml`. For example, a setting like `skip_release: true` will be used to prevent automated releases of libraries that require manual intervention.

The workflow is orchestrated through `librarian` commands that wrap standard Go tooling (`protoc-gen-go-gapic`, `go mod tidy`) and Git, ensuring the process is idiomatic for Go developers.

## Detailed Design

### Library Naming and Path Inference
-   **Library Naming:** Go libraries are identified by their module path relative to the repository root (e.g., `secretmanager/apiv1`). A special ID, `root-module`, is used for repository-wide concerns that don't belong to a specific module.
-   **Path Inference:** When using `librarian add`, the tool can infer the API path (e.g., `google/cloud/secretmanager/v1`) from a Go module path.

### `librarian add`
-   **Functionality:** Adds a new library entry to `librarian.yaml`.
-   **Go-Specifics:**
    -   Configures the Go module path and relevant API paths.
    -   The `output` directory defaults to the module path relative to the repository root (e.g., `secretmanager/apv1`).

### `librarian generate`
-   **Functionality:** Orchestrates the Go GAPIC generator to produce client library code, ensuring the output directory is a valid and up-to-date Go module.
-   **Go-Specifics:**
    1.  **First-Time Module Initialization:** The command first checks if a `go.mod` file exists in the output directory.
        -   **If `go.mod` does NOT exist:** It performs a one-time scaffolding step by running `go mod init <module-path>` to make the directory a valid Go module.
    2.  **`go.mod` Generation:** On every run (including the first), it generates the `go.mod` file from a template, populating it with the specific module path and dependencies defined in `librarian.yaml`. This overwrites the file created during initialization, ensuring it is always synchronized with the configuration.
    3.  **Code Generation:** It executes `protoc` with `protoc-gen-go-gapic` to generate the `.go` source files from protocol buffers.
    4.  **Output Flattening:** It flattens the initial nested generator output to the correct, final module structure.
    5.  **Dependency Management:** It runs `go mod tidy` to ensure the `go.sum` file is correct and all dependencies are properly recorded.
    6.  **Global File Modifications:** Respects a configuration setting that permits modifications to shared, repository-level files like `internal/generated/snippets/go.mod`.

### `librarian stage`
-   **Functionality:** Prepares a new release by calculating the next version and updating Go module files. This command modifies local files, which are then expected to be committed.
-   **Go-Specifics:**
    1.  **Version Calculation:** Analyzes the Git commit history since the last release tag to determine the next semantic version.
    2.  **Update `librarian.yaml`:** Modifies `librarian.yaml` to set the `version` field for the library to the newly calculated version.
    3.  **Internal Version File Generation:** Creates or updates `internal/version.go` within the module's directory, embedding the new version string directly into the Go source code.
    4.  **Snippet Metadata Update:** Updates the `clientLibrary.version` field in all relevant `snippet_metadata.json` files with the new version.

### `librarian tag`
-   **Functionality:** Creates and pushes the Git tag for a staged Go module.
-   **Go-Specifics:**
    1.  **Change Detection:** Identifies which Go modules are candidates for tagging by checking for updated versions in `librarian.yaml` since the last Git tag.
    2.  **Tagging:** For each candidate module, it creates and pushes a new Git tag. The tag's format is derived from the module's path and version (e.g., `secretmanager/apiv1/v1.5.0`).

### `librarian publish`
-   **Functionality:** Finalizes the publication of Go modules by triggering Go Proxy discovery.
-   **Go-Specifics:**
    1.  **Trigger Go Proxy Discovery:** For each newly tagged module, it runs a `go get <module_path>@<version>` command (e.g., `go get cloud.google.com/go/secretmanager/apiv1@v1.5.0`). This explicit `go get` operation signals the Go Proxy to discover and index the newly released module version.

## Alternatives Considered
(This section can be filled in as the design evolves.)
