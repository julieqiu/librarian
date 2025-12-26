# Migration Execution Plan

This document outlines the operational plan for unifying the fractured release
tooling ecosystem into the single `librarian` system.

## The Ecosystem
We are classifying systems into three migration categories:
1.  **Legacy Core (`legacylibrarian`)**: Used by Go and Python. High complexity, stateful migration required.
2.  **Sidekick (`sidekick`)**: Used by Rust. Structured configuration migration required.
3.  **Disparate Systems**: .NET (`librarian@v0.1.0`),
Ruby, PHP, Node, Java.
These use various custom scripts or older tools.

## Strategy 1: Go & Python (The Dual-Write Bridge)
**Risk:** High (Production critical, massive surface area).
**Method:** "Strangler Fig" / Dual-Write.

1.  **Code Integration**: Cherry-pick `legacylibrarian` logic into `main` to create a unified binary.
2.  **Dual-Write**: Update `legacylibrarian` command to write a shadow `librarian.yaml`
(v2) whenever it updates `.librarian/state.yaml`.
    *   *Safety:* Header `# GENERATED FILE...` added.
3.  **Validation**: Verify shadow files in production.
4.  **Cutover**: Switch tooling to read `librarian.yaml`.

## Strategy 2: Rust (Config Translation)
**Risk:** Medium.
**Method:** Configuration Mapping.

1.  **Tooling**: Use `migrate-sidekick` to deterministicly map `.sidekick.toml` to `librarian.yaml`.
2.  **Execution**: Run migration, delete old config, swap binary in CI, and commit in a single PR per repository.

## Strategy 3: .NET, Java, Ruby, et al. (Onboarding)
**Risk:** Variable.
**Method:** Fresh Onboarding (Greenfields).

We do not attempt to "migrate" the configuration of these disparate tools. Instead, we treat them as new onboardings.

1.  **Inventory**: Identify the active libraries in the repository.
2.  **Init**: Run `librarian init` (or script `librarian create` loops)
to generate a pristine `librarian.yaml` that reflects the current repo state.
3.  **Swap**: Replace the legacy build/release scripts in CI with `librarian` commands.
