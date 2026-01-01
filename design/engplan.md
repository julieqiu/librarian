# Engineering Execution Plan

This document outlines the phased roadmap for building and deploying the new `librarian` ecosystem.

## Phase 1: Design & Specification
**Goal:** Define the system architecture, configuration schema, and workflows.
*   **Status:** Complete.
*   **Deliverables:**
    *   `design/librarian.yaml` (Schema)
    *   `design/cli.md` (Tooling Interface)
    *   `design/migrate.md` (Migration Strategy)
    *   `design/branches.md` & `design/freeze.md` (Workflows)
    *   `design/googleapis.md` (Upstream Data Model)

## Phase 2: Foundation (Rust Parity)
**Goal:** Build the `librarian` binary to feature parity with `sidekick` (Rust). Prove the design works for one language.
*   **Focus:** `librarian create`, `generate`, `update`, `release` for Rust.
*   **Milestones:**
    *   [Milestone 73](https://github.com/googleapis/librarian/milestone/73)
    *   [Milestone 74](https://github.com/googleapis/librarian/milestone/74)
    *   [Milestone 67](http://github.com/googleapis/librarian/milestone/67)

## Phase 3: Migration Bridge (Dual-Write)
**Goal:** Secure the legacy production path for Go and Python before attempting to port them.
*   **Task:** Implement the "Dual-Write" strategy in `legacylibrarian`.
*   **Deliverable:** Production releases of Go/Python libraries automatically generate a shadow `librarian.yaml` (v2), enabling validation without disruption.

## Phase 4: Infrastructure (The Central Catalog)
**Goal:** Implement the centralized API definition system to support multi-language scaling.
*   **Dependency:** Required before onboarding Python/Go/.NET to avoid configuration duplication.
*   **Tasks:**
    *   Create `sdk.yaml` in the `librarian` repo.
    *   Populate it with Standard and Legacy APIs.
    *   Update `librarian` CLI to resolve generation targets via the Catalog.

## Phase 5: Expansion (Python, Go, .NET)
**Goal:** Port the remaining languages to the new `librarian` tool.
*   **Dependency:** Requires Phase 3 (Validation) and Phase 4 (Catalog).
*   **Tasks:**
    *   **Python:** Port logic from `legacylibrarian` / `.generator`.
    *   **Go:** Port logic from `legacylibrarian`.
    *   **.NET:** Implement generation logic (replacing `librarian@v0.1.0`).
*   **Deliverable:** A single `librarian generate` command works consistently across all 4 languages.

## Phase 6: Automation (LibrarianOps)
**Goal:** Automate the manual CLI workflows defined in previous phases.
*   **Dependency:** Requires a stable CLI (Phase 2 & 5).
*   **Design:**
    *   Define automation scope (PR creation, updates, release tagging).
    *   Design `sdk.yaml` update triggers.
    *   Specify Bot Branch handling.
*   **Implementation:** Build the `librarianops` bot.
