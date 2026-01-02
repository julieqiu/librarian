Migrate `release_level` configuration from `.sidekick.toml` to `sdk.yaml`.

- **Comparison:** Compare existing `release_level` values in `.sidekick.toml` with those defined in `go_gapic_library` rules within `BUILD.bazel` files to identify any discrepancies or conflicts before migration.
- **Implementation:**
    - Identify where `release_level` is currently used in the codebase (likely in Sidekick-related parsing).
    - Modify the configuration loading logic to read `release_level` from `sdk.yaml` instead.
    - Ensure all parts of the system that rely on `release_level` are updated to use the new source.