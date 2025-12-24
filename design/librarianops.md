# LibrarianOps Automation Design

This document outlines the architecture for `librarianops`, the automation engine that drives the `librarian` ecosystem. It is designed as a **Service-First** system to enable event-driven responsiveness, while retaining a **CLI** for operator control and debugging.

## Architecture

The system follows a "Core Logic as Library" pattern:

1.  **Core Logic (`internal/ops`)**: Contains the business logic for reconciling repository state (e.g., `SyncSources`, `ReleaseRepository`). It is agnostic to *how* it is triggered (HTTP request vs CLI command).
2.  **The Service (`cmd/librarianserver`)**: A long-running web service (deployed to Cloud Run) that listens for events (GitHub Webhooks, Cloud Pub/Sub, Cloud Scheduler triggers) and invokes the Core Logic.
3.  **The Admin CLI (`cmd/librarianops`)**: A command-line tool that wraps the Core Logic. Used by engineers for manual interventions, testing, and "break-glass" scenarios.

## Core Operations

These operations are implemented in the Core Logic and exposed via both the Service and CLI.

### 1. `Sync Sources`
*   **Goal:** Keep downstream language repositories up-to-date with upstream `googleapis` commits.
*   **Trigger (Service):**
    *   **Event:** `push` webhook from `googleapis/googleapis`.
    *   **Schedule:** Daily cron (safety net).
*   **Workflow:**
    1.  Detect new commit.
    2.  For each repo: Create bot branch -> Run `librarian update` -> Open PR.

### 2. `Generate All`
*   **Goal:** Ensure generated code is fresh (e.g., after a tool update or source sync merge).
*   **Trigger (Service):**
    *   **Event:** `pull_request` merge event on a language repo (if it touched `librarian.yaml`).
    *   **Schedule:** Nightly.
*   **Workflow:**
    1.  For each repo: Create bot branch -> Run `librarian generate` -> Open PR.

### 3. `Onboard APIs`
*   **Goal:** Automatically onboard new APIs defined in `catalog.yaml`.
*   **Trigger (Service):**
    *   **Event:** `push` webhook to `googleapis/librarian` (modifying `catalog.yaml`).
*   **Workflow:**
    1.  Diff `catalog.yaml` vs Repo `librarian.yaml`.
    2.  For new APIs: Create bot branch -> Run `librarian create` -> Open PR.

### 4. `Release`
*   **Goal:** Prepare release artifacts.
*   **Trigger (Service):**
    *   **Schedule:** Weekly "Release Train".
    *   **Manual API Call:** "Trigger Release" button in internal dashboard.
*   **Workflow:**
    1.  For each repo: Create release branch -> Run `librarian release` -> Open PR.

### 5. `Sync Catalog`
*   **Goal:** Keep `catalog.yaml` in sync with `googleapis`.
*   **Trigger (Service):**
    *   **Schedule:** Daily.
    *   **Event:** `push` webhook from `googleapis/googleapis`.
*   **Workflow:**
    1.  Scan `googleapis` -> Propose updates to `catalog.yaml` -> Open PR.

## Operational Model

*   **Routine Operations:** Handled entirely by the **Service** reacting to GitHub events or Cron schedules.
*   **Emergency / Debug:** Handled by the **Admin CLI** running locally on an engineer's machine.
    *   *Example:* `librarianops sync-sources --repo googleapis/google-cloud-rust --force` (Bypasses event queue to fix a specific repo immediately).
