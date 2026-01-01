Emergency Release Workflow (During Code Freeze)
===============================================

This document outlines the standard operating procedure for performing an emergency release of a specific library while the repository is under a code freeze.

The Challenge
-------------

During a code freeze, the `main` branch is locked to prevent instability. However, critical security patches or hotfixes may be required for individual libraries.

The goal is to release the hotfix without:

1.	Destabilizing the rest of the repository.
2.	Creating configuration drift where `librarian.yaml` does not reflect the reality of the released artifact.

The Workflow: "Branch & Update"
-------------------------------

We prioritize repository consistency over library isolation. We accept that moving the global source forward to fix one library puts other libraries in a "stale" state, which is acceptable during a freeze.

### 1. Create a Hotfix Branch

Checkout a new branch from `main`.

```bash
git checkout -b hotfix/secretmanager-security
```

### 2. Update Global State

Edit `librarian.yaml` to point the global source (`googleapis`) to the new commit needed for the fix.

```yaml
# librarian.yaml
generation:
  sources:
    googleapis:
      commit: Commit_B # Updated from Commit_A
```

### 3. Surgical Generation

Run the generator *only* for the target library to minimize noise and build time.

```bash
librarian generate --library google-cloud-secretmanager
```

-	**Result:** `secretmanager` code is updated to `Commit_B`.
-	**Implication:** Other libraries are now stale (code is from `Commit_A`, config says `Commit_B`). This is acceptable on the hotfix branch.

### 4. Verification & Damage Control

Run tests for the target library.

```bash
# Example
cargo test -p google-cloud-secretmanager
```

**Repo Health Check:** If the new global source (`Commit_B`) causes other libraries (e.g., `google-cloud-aiplatform`) to fail compilation (even without regeneration, perhaps due to shared dependencies), you must **quarantine** them.

Edit `librarian.yaml`:

```yaml
libraries:
  - name: google-cloud-aiplatform
    version: 1.5.0
    generate: false # Disable generation if it won't compile
    # OR
    release: false  # Disable release if it compiles but behaves incorrectly
```

### 5. Commit and Release

Commit the changes:

1.	Updated global `googleapis` commit.
2.	Updated `secretmanager` code.
3.	Any quarantined libraries.

Merge this branch into `main`. The release automation (or manual trigger) should then run `librarian release`.

### 6. Post-Freeze Cleanup

Once the freeze is lifted:

1.	Run `librarian generate` (all) to bring all libraries up to `Commit_B` (eliminating the stale state).
2.	Fix and re-enable any quarantined libraries (`aiplatform`).
