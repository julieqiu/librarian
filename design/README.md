Librarian Configuration Design
==============================

This directory contains the design specifications for the `librarian.yaml` configuration schema, CLI tooling, automation, and associated workflows.

Unified Configuration Architecture
----------------------------------

Librarian implements a "Single Source of Truth" model that centralizes configuration across the ecosystem.

### Key Files


Directory Contents
------------------

Below is a complete list of all design documents in this directory.

### Design
-   [cli.md](./cli.md): CLI command specification.
-   [configuration.md](./configuration.md): Overview of the configuration files.
    -   [librarian.yaml](./librarian.yaml): The manifest for language-specific configuration.
    -   [sdk.yaml](./sdk.yaml): The master list of all APIs for which we create SDKs.
    -   [tool.yaml](./tool.yaml): Declarative manifest of the CLI dependencies.
    -   [serviceconfig.yaml](./serviceconfig.yaml): The authoritative language-neutral configuration.

### Root Directory
-   [branches.md](./branches.md): Branching strategy.
-   [contributor.md](./contributor.md): General overview for contributors.
-   [delete.md](./delete.md): Design for the library deletion process.
-   [engplan.md](./engplan.md): Engineering execution plan.
-   [freeze.md](./freeze.md): Design for freezing generated code.
-   [googleapis.md](./googleapis.md): Details on the googleapis submodule interaction.
-   [librarianops.md](./librarianops.md): Operational guide for librarian tooling.
-   [migrate.md](./migrate.md): Design for migrating existing libraries.
-   [onboarding.md](./onboarding.md): Workflow for onboarding new client libraries.
-   [README.md](./README.md): This file.
-   [test.md](./test.md): The end-to-end testing plan.
### Contributor Guides (`contributor/`)
-   [python.md](./contributor/python.md): Contributor guide for `google-cloud-python`.
-   [rust.md](./contributor/rust.md): Contributor guide for `google-cloud-rust`.

### Open Issues (`issues/`)
-   [multiple-runtimes.md](./issues/multiple-runtimes.md): Issue on supporting multiple Python runtimes.
-   [release-level-inference.md](./issues/release-level-inference.md): Issue on inferring release levels.
-   [release-ownership.md](./issues/release-ownership.md): Issue on release ownership.
-   [staggered-release.md](./issues/staggered-release.md): Issue on staggered releases.
-   [unified-config.md](./issues/unified-config.md): Issue on unified configuration.

### Language-Specific Designs (`languages/`)
-   [python.md](./languages/python.md): Architecture of the Python generator and release pipeline.
-   [rust.md](./languages/rust.md): Architecture of the Rust generator and release pipeline.
