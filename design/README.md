# Librarian Configuration Design

This directory contains the design specifications for the `librarian.yaml` configuration schema, CLI tooling, automation, and associated workflows.

## Unified Configuration Architecture

Librarian implements a "Single Source of Truth" model that centralizes configuration across the ecosystem.

### Key Files
*   **[librarian.yaml](./librarian.yaml)**: The manifest for language-specific configuration.
*   **[sdk.yaml](./sdk.yaml)**: The master list of all APIs for which we create SDKs.
*   **[tool.yaml](./tool.yaml)**: Declarative manifest of the CLI dependencies.
*   **[serviceconfig.yaml](./serviceconfig.yaml)**: The authoritative language-neutral configuration.

## Documentation Index

### Contributor Guides (How-To)
*   **[Contributor Guide](./contributor.md)**: General overview.
*   **[Python Guide](./contributor/python.md)**: For `google-cloud-python`.
*   **[Rust Guide](./contributor/rust.md)**: For `google-cloud-rust`.

### Design Documents (Architecture)
*   **[Python Design](./languages/python.md)**: Architecture of the Python generator and release pipeline.
*   **[Rust Design](./languages/rust.md)**: Architecture of the Rust generator and release pipeline.

### Workflows & Processes
*   **[onboarding.md](./onboarding.md)**: Workflow for onboarding new client libraries.
*   **[generate.md](./generate.md)**: Code generation workflow details.
*   **[release.md](./release.md)**: Release, publish, and automation workflows.
*   **[branches.md](./branches.md)**: Branching strategy.
*   **[cli.md](./cli.md)**: CLI command specification.
*   **[engplan.md](./engplan.md)**: Engineering execution plan.

### Open Issues
*   **[Unified Configuration](./issues/unified-config.md)**
*   **[Release Ownership](./issues/release-ownership.md)**
*   **[Staggered Release](./issues/staggered-release.md)**
*   **[Multiple Python Runtimes](./issues/multiple-runtimes.md)**
*   **[Release Level Inference](./issues/release-level-inference.md)**