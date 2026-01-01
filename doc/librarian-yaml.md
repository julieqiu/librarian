# librarian.yaml Specification

The `librarian.yaml` file is the central configuration file for the Librarian tool. It defines the workspace settings, managed libraries, and external dependencies.

## Structure

The configuration is structured as follows:

```yaml
# Language is the language for this workspace (e.g., "go", "python", "rust").
language: string

# Repo is the repository name, such as "googleapis/google-cloud-python".
repo: string

# Default contains default settings applied to all libraries.
default:
  # Output is the directory where code is written (e.g., "src/generated").
  output: string

  # ReleaseLevel is either "stable" or "preview".
  release_level: string

  # TagFormat is the template for git tags, such as "{name}/v{version}".
  tag_format: string

  # Transport is the transport protocol, such as "grpc+rest" or "grpc".
  transport: string

  # Rust contains Rust-specific default configuration.
  rust:
    # PackageDependencies is a list of default package dependencies.
    package_dependencies:
      - name: string
        package: string
        ignore: boolean
        source: string
        feature: string
        force_used: boolean
        used_if: string

    # DisabledRustdocWarnings is a list of rustdoc warnings to disable.
    disabled_rustdoc_warnings: [string]

    # GenerateSetterSamples indicates whether to generate setter samples.
    generate_setter_samples: string

# Sources references external source repositories.
sources:
  # Googleapis is the googleapis repository configuration.
  googleapis:
    commit: string
    sha256: string
    dir: string # Local directory override
    branch: string
    subpath: string

  # Discovery is the discovery-artifact-manager repository configuration.
  discovery:
    commit: string
    sha256: string
    dir: string
    branch: string
    subpath: string

  # Conformance is the path to the `conformance-tests` repository.
  conformance:
    commit: string
    sha256: string
    dir: string
    branch: string
    subpath: string

  # ProtobufSrc is the path to the `protobuf` repository.
  protobuf:
    commit: string
    sha256: string
    dir: string
    branch: string
    subpath: string

  # Showcase is the showcase repository configuration.
  showcase:
    commit: string
    sha256: string
    dir: string
    branch: string
    subpath: string

# Release holds configuration for release automation.
release:
  # Branch sets the name of the release branch, typically `main`.
  branch: string

  # Remote sets the name of the source-of-truth remote for releases, typically `upstream`.
  remote: string

  # IgnoredChanges defines globs that are ignored in change analysis.
  ignored_changes: [string]

  # Preinstalled defines the list of tools that must be pre-installed.
  preinstalled:
    <tool_name>: <path>

  # Tools defines the list of tools to install, indexed by installer.
  tools:
    <installer_name>:
      - name: string
        version: string

  # RootsPem is an alternative location for the `roots.pem` file.
  roots_pem: string

# Libraries contains configuration overrides for specific libraries.
libraries:
  - name: string
    version: string
    
    # ReleaseLevel overrides the default release level.
    release_level: string

    # Transport overrides the default transport.
    transport: string

    # Output overrides the default output directory.
    output: string

    # CopyrightYear is the copyright year for the library.
    copyright_year: string

    # DescriptionOverride overrides the library description.
    description_override: string

    # SpecificationFormat specifies the API specification format ("protobuf" or "discovery").
    specification_format: string

    # Veneer indicates if the library uses language-specific module configuration.
    veneer: boolean

    # SkipGenerate disables code generation for this library.
    skip_generate: boolean

    # SkipPublish disables publishing for this library.
    skip_publish: boolean

    # SkipRelease disables releasing for this library.
    skip_release: boolean

    # Keep lists files and directories to preserve during regeneration.
    keep: [string]

    # Roots specifies the source roots to use for generation.
    roots: [string]

    # Channel specifies which googleapis Channel to generate from.
    channels:
      - path: string
        service_config: string
        service_config_does_not_exist: boolean

    # Go contains Go-specific library configuration.
    go:
      module_path_version: string
      delete_generation_output_paths: [string]
      go_apis:
        - path: string
          client_directory: string
          disable_gapic: boolean
          nested_protos: [string]
          proto_package: string

    # Python contains Python-specific library configuration.
    python:
      opt_args: [string]
      opt_args_by_channel:
        <channel_path>: [string]

    # Rust contains Rust-specific library configuration.
    rust:
      # Inherits RustDefault fields
      package_dependencies: ...
      disabled_rustdoc_warnings: ...
      generate_setter_samples: ...

      # Modules specifies generation targets for veneer crates.
      modules:
        - source: string
          output: string
          template: string
          module_path: string
          include_list: string
          included_ids: [string]
          skipped_ids: [string]
          service_config: string
          routing_required: boolean
          has_veneer: boolean
          extend_grpc_transport: boolean
          include_grpc_only_methods: boolean
          generate_setter_samples: boolean
          post_process_protos: string
          name_overrides: string
          title_override: string
          module_roots:
            <key>: <value>
          disabled_rustdoc_warnings: [string]
          documentation_overrides:
            - id: string
              match: string
              replace: string

      per_service_features: boolean
      module_path: string
      template_override: string
      title_override: string
      package_name_override: string
      root_name: string
      default_features: [string]
      include_list: [string]
      included_ids: [string]
      skipped_ids: [string]
      disabled_clippy_warnings: [string]
      has_veneer: boolean
      routing_required: boolean
      include_grpc_only_methods: boolean
      generate_rpc_samples: boolean
      post_process_protos: string
      detailed_tracing_attributes: boolean
      name_overrides: string
      
      documentation_overrides:
        - id: string
          match: string
          replace: string

      pagination_overrides:
        - id: string
          item_field: string

      discovery:
        operation_id: string
        pollers:
          - prefix: string
            method_id: string
```
