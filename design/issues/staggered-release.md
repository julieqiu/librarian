# Staggered Release Workflow

This document addresses the problem of managing large-scale releases, particularly when a single change (e.g., a generator update) affects numerous client libraries simultaneously. It outlines a strategy for a "staggered rollout" to mitigate risk.

## The Problem: High Blast Radius Releases

When a core component like the code generator is updated, it often results in generated code changes across **many (e.g., 200+) client libraries** in a repository. If `librarian release --all` is executed in such a scenario, it would prepare and trigger the publication of new versions for *all* affected libraries simultaneously.

This creates a significant **"blast radius"**:

*   **Massive User Impact:** If a subtle bug is introduced by the generator update (and missed during automated testing), releasing all libraries at once means the bug is immediately exposed to the entire user base of those 200+ clients.
*   **Difficult Rollback:** Rolling back 200 published packages is a complex and time-consuming operation.
*   **Overwhelming CI/CD:** A single, giant release PR or a single release job processing hundreds of artifacts can strain CI/CD systems and increase the chance of unrelated failures.

## The Solution: Staggered Rollouts with `--limit`

To mitigate this risk, `librarian` supports a **staggered release** workflow, where libraries are released in smaller, controlled batches over time. This reduces the blast radius and allows for early detection of issues.

### Mechanism: `librarian release --limit <N>`

The `librarian release` command includes a `--limit <N>` flag. This flag instructs `librarian` to process only up to `N` libraries that have detected changes, even when `--all` is specified. Libraries are typically selected based on a deterministic order (e.g., alphabetical by name).

### Workflow

1.  **Initial Trigger (Generator Update):**
    A generator update is merged (e.g., via `librarianops sync-sources` and `librarianops generate-all`). This results in new generated code for many libraries, creating pending release changes.

2.  **Automated Staggered Release (`librarianops release`):**
    The `librarianops release` automation is configured to run on a schedule (e.g., daily) with the `--limit` flag.
    ```bash
    # Example: Release 10 libraries per day
    librarianops release --all --limit 10
    ```

    *   **Day 1:** `librarianops` detects 200 changed libraries. It selects the first 10 (alphabetically) and creates a Release PR for them.
    *   **Review & Merge:** The language team reviews and merges this PR.
    *   **Publish:** The CI/CD pipeline publishes these 10 libraries.

3.  **Monitor and Iterate:**
    *   **Day 2 (and subsequent days):** `librarianops` runs again. It sees that the first 10 libraries have now been released. The remaining 190 still have pending changes.
    *   It selects the *next* 10 pending libraries and creates a new Release PR.
    *   This process continues until all libraries have been released.

### Benefits

*   **Reduced Blast Radius:** If a bug is present, it affects only a small subset of users (e.g., 10 libraries) rather than the entire ecosystem.
*   **Early Detection:** Issues are more likely to be found in smaller batches, allowing for a quick pause and fix before widespread impact.
*   **Manageable Review:** Release PRs are smaller and easier for language teams to review.
*   **Steady Flow:** Maintains a continuous, predictable flow of updates rather than infrequent, massive releases.

## Related Issues

*   [GitHub Issue #3254: Support staggered rollout for mass releases](https://github.com/googleapis/librarian/issues/3254)