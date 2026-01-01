Librarian Contributor Guide
===========================

Welcome to the Librarian contributor documentation. This guide covers the standard workflows for maintaining client libraries across all supported languages.

General Workflows
-----------------

The `librarian` CLI provides a unified interface for managing client libraries:

-	**`librarian generate`**: Generates client code from upstream protos.
-	**`librarian create`**: Onboards a new API/library.
-	**`librarian release`**: Prepares a library for release (version bumping, changelogs).
-	**`librarian publish`**: Uploads artifacts to package registries.

Language-Specific Guides
------------------------

For detailed instructions specific to your language ecosystem, please refer to:

-	**[Python Guide](./contributor/python.md)**: Setup, workflows, and troubleshooting for `google-cloud-python`.
-	**[Rust Guide](./contributor/rust.md)**: Setup and workflows for `google-cloud-rust`.

Getting Help
------------

If you encounter issues, please file a bug in the `librarian` repository.
