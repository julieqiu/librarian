Librarian CLI Specification
===========================

Librarian is a tool for managing Google Cloud client libraries.

It provides a unified interface for onboarding, generating, releasing, and publishing client libraries across multiple languages.

Usage:

```
librarian <command> [arguments]
```

## Library Naming Conventions

Librarian derives library names based on language-specific conventions when not explicitly provided. This ensures consistency within each language ecosystem.

*   **Rust:** Library names follow the `google-cloud-<service>-<version>` pattern. For example, from a library name like `google-cloud-secretmanager-v1`, the API path can be inferred as `google/cloud/secretmanager/v1`.
*   **Go:** [Details on Go naming conventions to be added here]
*   **Python:** [Details on Python naming conventions to be added here]

The commands are:

-	**[add](#add)**: add a new client library to `librarian.yaml`
-	**[generate](#generate)**: generate client library code
-	**[update](#update)**: update sources to the latest version
-	**[release](#release)**: prepare libraries for release
-	**[publish](#publish)**: publish client libraries
-	**[tidy](#tidy)**: format and validate librarian.yaml
-	**[status](#status)**: check the health and readiness of libraries
-	**[version](#version)**: print the librarian version

## Add

`librarian add <library> [api_path...]`

Add onboards a new client library by adding it to `librarian.yaml`.

If one or more `api_path` arguments are provided, they will be used directly.

If no `api_path` is provided, Librarian will attempt to infer the API path based on the library name and the repository's configured language naming conventions. It will consult `sdk.yaml` for available standard APIs. If a unique API path can be unambiguously inferred, it will be used. If no unique API path can be inferred, the command will fail.

In adherence with Unix philosophy, this command produces no output on success and exits with a status code of 0. On failure (e.g., an ambiguous API path), it will print a concise error message to standard error and exit with a non-zero status code.

### Options

```
      --output <path>
          The directory where the library should be generated. If omitted, it is derived from defaults.
```

### Examples

```
# The CLI interface for adding a library should be:
librarian add <library> <api_path> [api_path...]
# For languages other than Rust (for example, Go and Python), multiple channels may be supported. In those cases, users can specify multiple API paths.

# For example, any of these commands would work:

librarian add google-cloud-secret-manager google/cloud/secretmanager/v1
librarian add google-cloud-secret-manager google/cloud/secretmanager/v1  --output packages/google-cloud-secret-manager
librarian add google-cloud-secret-manager \
  google/cloud/secretmanager/v1 \
  google/cloud/secretmanager/v1beta2 \
  google/cloud/secrets/v1beta1 \
  --output packages/google-cloud-secret-manager
```

Generate
--------

`librarian generate <library> | --all [flags]`

Generate generates client library code for managed libraries using the current configuration and sources. It performs the initial code generation for newly added libraries and regenerates code for existing ones. Either the library argument or the --all flag is required.

### Options

```
      --all
          Regenerate all libraries listed in librarian.yaml. Exclusive with the library argument.
```

### Examples

```
# Regenerate all libraries
$ librarian generate --all

# Regenerate a single library
$ librarian generate google-cloud-secretmanager
```

Update
------

`librarian update <source> | --all [flags]`

Update updates external dependencies, such as the commit hash for `googleapis` in `librarian.yaml`, to the latest available version. Either the source argument or the --all flag is required.

### Options

```
      --all
          Update all configured sources. Exclusive with the source argument.
```

### Examples

```
# Update the googleapis source commit
$ librarian update googleapis

# Update all sources
$ librarian update --all
```

Release
-------

`librarian release <library> | --all [flags]`

Release updates versions and prepares release artifacts. It calculates the next semantic version based on changes, updates manifest files. Either the library argument or the --all flag is required.

When using `--all`, the command will attempt to process all eligible libraries. If a release for a specific library fails, the command will report the error, but it will not halt the entire batch operation. It will continue attempting to release the remaining libraries. A summary of all successes and failures will be provided upon completion.

### Options

```
      --all
          Process all libraries that have changed since the last release.
```

### Examples

```
# Prepare release for all eligible libraries
$ librarian release --all

# Prepare release for a specific library
$ librarian release google-cloud-secretmanager
```

Publish
-------

`librarian publish <library> | --all [flags]`

Publish uploads prepared artifacts to package registries. Either the library argument or the --all flag is required.

### Options

```
      --all
          Publish all released artifacts.

      --dry-run
          Print the publish commands without executing them.

      --skip-semver-checks
          Skip semantic versioning checks.
```

### Examples

```
# Publish all artifacts
$ librarian publish --all

# Publish a specific library (dry run)
$ librarian publish google-cloud-secretmanager --dry-run
```

Tidy
----

`librarian tidy`

Tidy formats and validates the `librarian.yaml` configuration file. It simplifies entries by removing fields that can be derived from defaults.

### Examples

```
# Format and validate librarian.yaml
$ librarian tidy
```

Status
------

`librarian status [library]`

Status performs a health check on the managed libraries in the repository. It provides a high-level overview of the state of each library, helping developers understand which libraries need attention.

If a library name is provided, it will check the status of only that library. Otherwise, it will check all libraries in `librarian.yaml`.

The command checks for:
- **New Libraries Available:** Whether there are new standard APIs in `sdk.yaml` that are not yet configured in `librarian.yaml`.
- **Configuration:** Whether a configured library has not been generated yet.
- **Local Changes:** Whether there are local modifications in a library's directory that would be overwritten by a `generate` command.
- **Release Readiness:** Whether a library has changed since its last release and is ready to be released.

### Examples

```
# Check the status of all libraries
$ librarian status

# Example output:
# New libraries available to be added: google-cloud-new-service-v1
# LIBRARY                         STATUS
# google-cloud-secretmanager      Ready to release
# google-cloud-pubsub             Modified (will be overwritten by generate)
# google-cloud-storage            Not generated

# Check the status of a single library
$ librarian status google-cloud-secretmanager
```


Version
-------

`librarian version`

Version prints the version of the librarian binary.

### Examples

```

$ librarian version

librarian version 0.7.0

```

Delete
------

`librarian delete <library>`

Delete removes a client library from `librarian.yaml` and deletes its generated
code from the repository.

This command is designed to be safe and will **only** delete files that would
be managed by the generation process. It respects the library's `keep` rules
and other configuration that preserve handwritten code. This command is
typically used when an API is deprecated or a library is no longer maintained.

### Examples

```bash

# Remove the secretmanager library

librarian delete secretmanager

```

Alternatives Considered
-----------------------

We considered having a single `librarian create` command that would add a new client library to `librarian.yaml` and immediately perform its initial code generation. This was attractive because it offered a single, atomic step for users to get a new, fully generated library, simplifying the "happy path" and reducing the chance of users forgetting to run generation after configuration.

However, we ultimately went with separating this functionality into two commands: `librarian add` and `librarian generate`. This approach was chosen because it provides a clearer separation of concerns, making the CLI more predictable and flexible. The `librarian add` command now focuses solely on configuring `librarian.yaml`, while `librarian generate` is responsible for all code generation, whether it's the first time for a new library or a subsequent regeneration. This design supports more efficient automation by allowing multiple libraries to be added before a single, potentially long-running generation process. Although it introduces an extra step for the user during initial setup, the improved clarity, predictability, and flexibility for automation were deemed more beneficial for the long-term maintainability and usability of the tool.
