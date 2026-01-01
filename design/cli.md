# Librarian CLI Specification

Librarian is a tool for managing Google Cloud client libraries.

It provides a unified interface for onboarding, generating, releasing, and publishing client libraries across multiple languages.

Usage:

```
librarian <command> [arguments]
```

The commands are:

*   **[create](#create)**: create a new client library
*   **[generate](#generate)**: generate client library code
*   **[update](#update)**: update sources to the latest version
*   **[tidy](#tidy)**: format and validate librarian.yaml
*   **[release](#release)**: prepare libraries for release
*   **[publish](#publish)**: publish client libraries
*   **[version](#version)**: print the librarian version

## Create

`librarian create <library> [flags]`

Create onboards a new client library by adding it to `librarian.yaml` and performing the initial code generation. The library argument is required.

### Options

```
      --output <path>
          The directory where the library should be generated. If omitted, it is derived from defaults.

      --service-config <path>
          The path to the upstream service configuration YAML.

      --specification-format <format>
          The format of the API source (default: protobuf).

      --specification-source <path>
          The path to the API definition in googleapis (e.g., google/cloud/secretmanager/v1).
```

### Examples

```
# Create a library for Secret Manager
$ librarian create google-cloud-secretmanager --specification-source google/cloud/secretmanager/v1

# Create a library with a custom output directory
$ librarian create google-cloud-secretmanager --output src/secretmanager
```

## Generate

`librarian generate <library> | --all [flags]`

Generate regenerates the code for managed libraries using the current configuration and sources. Either the library argument or the --all flag is required.

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

## Update

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

## Tidy

`librarian tidy`

Tidy formats and validates the `librarian.yaml` configuration file. It simplifies entries by removing fields that can be derived from defaults.

### Examples

```
# Format and validate librarian.yaml
$ librarian tidy
```

## Release

`librarian release <library> | --all [flags]`

Release updates versions and prepares release artifacts. It calculates the next semantic version based on changes, updates manifest files, and generates changelog entries. Either the library argument or the --all flag is required.

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

## Publish

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

## Version

`librarian version`

Version prints the version of the librarian binary.

### Examples

```
$ librarian version
librarian version 0.7.0
```