# Librarian Configuration Design

This directory contains the design specifications for the `librarian.yaml` configuration schema, CLI tooling, automation, and associated workflows.

## Unified Configuration Architecture

Librarian implements a "Single Source of Truth" model that centralizes configuration across the ecosystem, eliminating redundant files (e.g., `BUILD.bazel`, `GAPIC YAML`, `API Index`) and separating concerns.

### 1. [librarian.yaml](./librarian.yaml) (The Manifest)
**Owner:** Language Repository.
**Role:** Authoritative source for **language-specific** configuration and repository state.
**Key Features:** 
- Library versions and inventory.
- Language-specific generation overrides.
- **Safe Removal Policy**: Uses explicit `delete_patterns` to clean stale generated code.

### 2. [catalog.yaml](./catalog.yaml) (The Registry)
**Owner:** Platform / Central.
**Role:** Master list of **all APIs** available for generation.
**Key Features:**
- Canonical API identities.
- API maturity tracking.
- Mapping to upstream source locations.

### 3. [tools.yaml](./tools.yaml) (The Environment)
**Owner:** Language Maintainer.
**Role:** Declarative manifest of the **build environment**.
**Key Features:**
- Required language runtimes (e.g., Python 3.14, Rust 1.76).
- Tooling dependencies (e.g., `protoc`, `cargo-semver-checks`).

### 4. [serviceconfig.yaml](./serviceconfig.yaml) (The API Definition)
**Owner:** API Producer (Upstream).
**Role:** Authoritative **language-neutral** configuration for an API.
**Key Features:**
- Transports and numeric enum settings.
- Retry policies and request deadlines (merged from gRPC JSON).
- LRO (Long Running Operation) polling settings.

## Design Principles

1.  **Strict Separation of Concerns**: Language-neutral settings live in the Service Config; language-specific settings live in `librarian.yaml`.
2.  **Explicit Intent**: Use explicit `delete_patterns` for cleanup rather than implicit "preserve" lists to prevent stale code persistence.
3.  **No Redundancy**: If data exists in the Service Config, it should not be duplicated in `librarian.yaml`.
4.  **Consumptive Stability**: Librarian acts as the integration layer, reconciling legacy inputs with the emerging unified model during the migration phase.

## Documents

### Configuration
*   **[librarian.yaml](./librarian.yaml)**: The reference specification and annotated example of the new configuration schema.
*   **[catalog.yaml](./catalog.yaml)**: The central manifest of all available APIs that can be onboarded.
*   **[tools.yaml](./tools.yaml)**: Specification for the build environment and tooling requirements.
*   **[serviceconfig.yaml](./serviceconfig.yaml)**: The upstream Service Configuration file (from `googleapis/googleapis`).
*   **[googleapis.md](./googleapis.md)**: Describes the upstream repository and the role of the central catalog.

### Open Issues & Considerations
*   **[Unified Configuration](./issues/unified-config.md)**: Tracks the "Single Source of Truth" initiative (Milestone 72), including the consolidation of `BUILD.bazel`, `GAPIC YAML`, and `gRPC JSON`.
*   **[Release Ownership and Control](./issues/release-ownership.md)**: Empowering language teams with control over release timing.
*   **[Staggered Release](./issues/staggered-release.md)**: Managing large-scale releases using the `--limit` flag.
*   **[Multiple Python Runtimes](./issues/multiple-runtimes.md)**: Supporting multiple runtimes vs. consolidating to 3.14.
*   **[Release Level Inference](./issues/release-level-inference.md)**: Automatically setting `release_level` based on API versions.

### Workflows & Processes
*   **[onboarding.md](./onboarding.md)**: Workflow for onboarding new client libraries.
*   **[generate.md](./generate.md)**: Code generation workflow details.
*   **[release.md](./release.md)**: Release, publish, and automation workflows.
*   **[branches.md](./branches.md)**: Branching strategy (Release, Bot, Feature).
*   **[cli.md](./cli.md)**: Specification for the `librarian` CLI commands.
*   **[engplan.md](./engplan.md)**: Phased engineering execution plan.
