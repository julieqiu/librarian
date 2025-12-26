# Release Level Inference for New Libraries

## The Problem

Currently, when Librarian generates a client library for a new API, it defaults the `release_level` in `.repo-metadata.json` to `"preview"`. This is a safe default, but it often misaligns with the actual maturity of the API being onboarded.

If a new client library is generated for a mature, stable API (e.g., `v1`), manually updating the release level to "stable" is an extra step that can be missed or creates unnecessary toil. Conversely, automatically setting "stable" for a beta API would be misleading.

## Goal

To automate the determination of the correct `release_level` ("stable" or "preview") for newly generated client libraries based on the intrinsic properties of the API itself, specifically its version string and launch stage.

## Proposed Logic

The `librarian create` command (and the underlying generation logic) should infer the release level using the following precedence:

### 1. API Path Version Analysis
The primary signal is the version segment in the API's proto path (e.g., `google/cloud/secretmanager/v1`).

*   **Stable:** Versions matching `v[0-9]+` (e.g., `v1`, `v2`).
*   **Preview:** Versions containing `alpha` or `beta` (e.g., `v1beta1`, `v1alpha2`).

### 2. API Launch Stage (Secondary Signal)
Many Google Cloud APIs define a `launch_stage` in their `service_config.yaml` or directly in the `.proto` file options.

*   If `launch_stage` is `GA`, `GA_WITH_DEPRECATION`, or similar, it strongly suggests a **Stable** release level.
*   If `launch_stage` is `BETA`, `ALPHA`, or `EARLY_ACCESS`, it mandates a **Preview** release level.

### Decision Algorithm

1.  **Check `service_config.yaml`**: If a `launch_stage` is explicitly defined, use it to determine stability.
    *   `GA` -> `stable`
    *   `BETA`/`ALPHA` -> `preview`
2.  **Fallback to Path Parsing**: If `launch_stage` is absent or ambiguous, parse the API path version.
    *   Regex `^v\d+$` (e.g., `v1`) -> `stable`
    *   Regex `.*(alpha|beta).*` -> `preview`
3.  **Default**: If neither signal is decisive, default to `preview` (current behavior) for safety.

## Configuration Override

As always, `librarian.yaml` serves as the final source of truth. The inferred release level is used to populate the initial entry in `librarian.yaml`.

```yaml
libraries:
  - name: google-cloud-newservice
    version: 0.1.0
    release_level: stable # Inferred by Librarian during 'create'
```

If the inference is incorrect for a specific case, the repository maintainer can manually override the `release_level` field in `librarian.yaml`.

## Related Issues

*   [GitHub Issue #2352: Determine release level based on API path and launch stage](https://github.com/googleapis/librarian/issues/2352)
