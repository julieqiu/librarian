# Librarian Configuration Design

This directory contains the design specifications for the `librarian.yaml`
configuration schema,
CLI tooling, automation, and associated workflows.

## Documents

### Configuration
*   **[librarian.yaml](./librarian.yaml)**:
The reference specification and annotated example of the new configuration schema.
This defines the structure for global settings,
tooling, generation inputs, and the release inventory.
*   **[catalog.yaml](./catalog.yaml)**: The central manifest of all available
APIs (Standard and Legacy) that can be onboarded for client library generation.
*   **[serviceconfig.yaml](./serviceconfig.yaml)**:
An example of the upstream Service Configuration file (found in `googleapis/googleapis`)
which serves as a primary input for generation.
*   **[googleapis.md](./googleapis.md)**:
Describes the upstream `googleapis` repository and the role of the central catalog,
referencing `catalog.yaml` and `serviceconfig.yaml` for their detailed structures.

### Automation
*   **[librarianops.md](./librarianops.md)**:
Architecture for the automation engine (Service + CLI) that drives synchronization,
onboarding, and releases.

### Workflows & Processes
*   **[onboarding.md](./onboarding.md)**:
Describes the workflow for onboarding new client libraries,
driven by `librarian create` and `librarianops onboard-apis`.
*   **[generate.md](./generate.md)**: Details the code generation workflow,
using `librarian generate` and `librarianops generate-all`.
*   **[release.md](./release.md)**: Outlines the release workflow,
including `librarian release`, `librarian publish`,
and `librarianops release`.
*   **[branches.md](./branches.md)**: Defines the branching strategy,
covering Release branches (for standard and hotfix releases),
Bot branches, and Feature branches.
*   **[delete.md](./delete.md)**: Describes the workflow for deleting a client library from `librarian`'s management.
*   **[engplan.md](./engplan.md)**: The phased engineering execution plan
for building and deploying the `librarian` ecosystem.
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