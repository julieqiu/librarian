Analyze and unify transport configuration across languages.

- **Analysis:** Investigate how `transport` is currently configured for all supported languages (Go, Python, Rust).
- **Sources to Check:** Review `BUILD.bazel` files, `.sidekick.toml`, and any existing language-specific configurations.
- **Identify Inconsistencies:** Document any discrepancies in how `transport` is specified, defaulted, or interpreted between the different language toolchains.
- **Propose Unification:** Define a single, consistent approach for managing the `transport` setting in `sdk.yaml` or `librarian.yaml` that can be applied globally.
