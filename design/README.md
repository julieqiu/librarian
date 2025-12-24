# Librarian Configuration Design (v2)

This directory contains the design specifications for the version 2 `librarian.yaml`
configuration schema,
CLI tooling, and associated workflows.

## Documents

### Configuration
*   **[librarian.yaml](./librarian.yaml)**:
The reference specification and annotated example of the new configuration schema.
This defines the structure for global settings,
tooling, generation inputs, and the release inventory.
*   **[googleapis.md](./googleapis.md)**:
Describes the upstream `googleapis` repository and the centralized `catalog.yaml`
which defines the "menu" of available APIs for all languages.

### Workflows & Processes
*   **[branches.md](./branches.md)**: Defines the branching strategy,
covering Release branches (for standard and hotfix releases),
Bot branches, and Feature branches.
*   **[freeze.md](./freeze.md)**: A detailed workflow guide for handling
emergency releases (hotfixes) during a repository code freeze.
*   **[cli.md](./cli.md)**: Specification for the new `librarian` CLI commands (`create`,
`generate`, `update`, `release`, `publish`) and their relationship to the
legacy `sidekick` tool.
*   **[rust.md](./rust.md)**: A concrete "How-To" guide for Rust contributors,
demonstrating the new CLI commands in practice.

## Key Design Principles

1.  **Single Source of Truth:** A unified `libraries` list defines both
the generation inputs and the release artifacts,
avoiding split-brain configuration states.
2.  **Explicit Intent:** We use explicit flags (`generate:
false`) rather than implicit "modes" to handle handwritten or broken libraries.
3.  **Separation of Concerns:** Global `generation` and `release` settings are isolated in their own top-level blocks.
4.  **Safety:** The design prioritizes repository consistency and explicit
opt-ins for complex behaviors like "Veneer" libraries.
