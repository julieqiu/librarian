Milestone: Unified Configuration (Milestone 72)
===============================================

**Status:** In Progress **GitHub Milestone:** [Milestone 72](https://github.com/googleapis/librarian/milestone/72)

Objective
---------

Streamline and centralize configuration management by eliminating redundant files and separating language-neutral from language-specific settings.

Tracked Issues
--------------

| Issue                                                        | Title                                                                 | Status             | Impact on Design                                                                |
|:-------------------------------------------------------------|:----------------------------------------------------------------------|:-------------------|:--------------------------------------------------------------------------------|
| [#3002](https://github.com/googleapis/librarian/issues/3002) | config: how do we specify how to remove previously-generated files    | Defined            | Replaced "preserve" regexes with `delete_patterns` in `librarian.yaml`.         |
| [#3003](https://github.com/googleapis/librarian/issues/3003) | librarian: do we still need repo-metadata.json?                       | Evaluating         | Exploring language-neutral methods to expose this metadata.                     |
| [#3004](https://github.com/googleapis/librarian/issues/3004) | config: remove gRPC service config JSON files                         | Planned            | Absorbed retry/timeout settings into `serviceconfig.yaml`.                      |
| [#3005](https://github.com/googleapis/librarian/issues/3005) | config: remove configuration from BUILD.bazel                         | Planned            | Moved `transports` and `rest_numeric_enums` to `serviceconfig.yaml`.            |
| [#3006](https://github.com/googleapis/librarian/issues/3006) | config: remove the API index and its generator                        | Planned            | APIs are now resolved directly via `sdk.yaml` and source.                       |
| [#3008](https://github.com/googleapis/librarian/issues/3008) | config: remove synthtool                                              | Planned            | Minimized post-processing; logic moved to `librarian.yaml` or generator.        |
| [#3040](https://github.com/googleapis/librarian/issues/3040) | config: plan for language-specific and language-neutral configuration | **Core Principle** | Defined the boundary: Neutral (Service Config) vs. Specific (`librarian.yaml`). |
| [#3041](https://github.com/googleapis/librarian/issues/3041) | config: remove xyz_gapic.yaml files                                   | Planned            | All overrides consolidated into the `libraries` block of `librarian.yaml`.      |
| [#3042](https://github.com/googleapis/librarian/issues/3042) | config: evaluate value of gapic_metadata.json files                   | Evaluating         | Potentially generating these from the unified model.                            |

Key Design Decisions
--------------------

1.	**The Boundary:** API-producer-owned data (timeouts, retries, transports) stays in the upstream Service Config. Language-specific data (package names, output paths) stays in the repository's `librarian.yaml`.
2.	**Safe Removal:** We explicitly define what to delete (`delete_patterns`) to ensure no stale code remains while protecting handwritten files.
3.	**Environment Separation:** Tooling requirements are moved to a standalone `tool.yaml` to decouple the build environment from the library configuration.
