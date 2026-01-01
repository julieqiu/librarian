Branching Strategy
==================

This document outlines the branching strategy for repositories managed by `librarian`. The workflow uses **Release Branches** for all release activities (including emergency hotfixes) and **Bot Branches** for automated maintenance.

Core Principle
--------------

**`librarian` operates on the current working directory.** The CLI tool itself does not create, switch, or merge branches. It modifies the files in the current checkout. Branch management is the responsibility of the user (developer) or the automation bot wrapper (e.g., `librarianops`).

Branch Types
------------

### 1. Release Branches (`release/...`\)

Used for **all** release activities, including standard releases and emergency hotfixes during a freeze. A release branch is the staging area where version numbers are bumped, changelogs are updated, and `librarian.yaml` is finalized for the release tag.

-	**Naming Convention:** `release/<library>-v<version>` (e.g., `release/google-cloud-secretmanager-v1.2.1`\)
-	**Workflow:**
	1.	**Create Branch:** Create a branch from `main`.
	2.	**Run Release:** Run `librarian release` (often via automation).
		-	This updates `librarian.yaml` (versions).
		-	This updates manifest files (`Cargo.toml`, `setup.py`).
		-	This updates `CHANGELOG.md`.
	3.	**Commit & Push:** Commit these changes.
	4.	**Tag:** The git tag (e.g., `google-cloud-secretmanager/v1.2.1`) is created pointing to the commit on this release branch.
	5.	**Merge:** The branch is merged back into `main` to preserve the history.

**Why use Release Branches for Hotfixes?** During a code freeze, `main` cannot be used directly. By doing the work on a `release/...` branch, we isolate the changes. The tag is created on this branch, allowing the artifact to be published immediately. The merge to `main` can happen immediately or be queued until the freeze lifts, but the artifact is already safe because it was built from the tagged commit on the release branch.

### 2. Bot Branches (`chore/...` or `owlbot/...`\)

These branches are created by automation (e.g., `librarianops` or OwlBot) to propose updates to the repository. They are used for generating code updates from new upstream sources.

-	**Naming Convention:** `owlbot/update-googleapis-<commit-hash>` or `chore/update-deps-<date>`
-	**Workflow:**
	1.	**Detection:** The bot detects a new commit in `googleapis/googleapis`.
	2.	**Branching:** The bot creates a branch in the language repo (e.g., `googleapis/google-cloud-python`).
	3.	**Execution:** The bot runs:
		-	`librarian update --all` (updates `librarian.yaml` source hashes)
		-	`librarian generate --all` (regenerates code)
	4.	**PR Creation:** The bot opens a Pull Request targeting `main`.
	5.	**Merge:** A human reviews and merges the PR.

### 3. Feature Branches (`feat/...`\)

Used by developers for manual changes, such as onboarding new libraries or writing handwritten code.

-	**Naming Convention:** `feat/<description>`
-	**Workflow:** Standard PR workflow targeting `main`.

### 4. Specialized Branches (Interface Based Versioning)

These branches are used for specific, long-term versioning strategies, particularly for client libraries that need to track a specific version of an API's interface independently from the `main` branch of `googleapis`. This is known as "Interface Based Versioning".

-	**Naming Convention:** `api-v<major>/<minor>` or `lts/<api-name>-v<major>`
-	**Purpose:** To allow a client library to be generated against a fixed or slower-moving API definition, providing stability for specific client versions without forcing them to constantly update to the latest API changes on `googleapis` main. This might be used for Long-Term Support (LTS) client libraries or for clients that require a specific API compatibility level.
-	**Workflow:**
	1.	A specialized branch is created (e.g., `api-v1.x`) from `main` at a specific `googleapis` commit.
	2.	`librarian.yaml` on this branch would explicitly pin its `generation.sources.googleapis.commit` to the desired API interface version, potentially overriding the default `main` branch commit. This is an explicit exception to the global source policy on the `main` branch.
	3.	Client libraries on *this branch* would then be generated and released independently, providing a stable API surface for consumers of that client version.
