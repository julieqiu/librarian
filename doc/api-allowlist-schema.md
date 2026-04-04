# API Allowlist Schema

This document describes the schema for the API Allowlist.

## API Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `path` | string | Is the proto directory path in github.com/googleapis/googleapis. If ServiceConfig is empty, the service config is assumed to live at this path. |
| `description` | string | Provides the information for describing an API. |
| `discovery` | string | Is the file path to a discovery document in github.com/googleapis/discovery-artifact-manager. Used by sidekick languages (Rust, Dart) as an alternative to proto files. |
| `documentation_uri` | string | Overrides the product documentation URI from the service config's publishing section. |
| `languages` | list of string | Restricts which languages can generate client libraries for this API. Use "all" to indicate all languages can use this API.<br><br>Restrictions exist for several reasons:<br>- Newer languages (Rust, Dart) skip older beta versions when stable versions exist<br>- Python has historical legacy APIs not available to other languages<br>- Some APIs (like DIREGAPIC protos) are only used by specific languages |
| `new_issue_uri` | string | Overrides the new issue URI from the service config's publishing section. |
| `no_rest_numeric_enums` | map[string]bool | Determines whether to use numeric enums in REST requests. The "No" prefix is used because the default behavior (when this field is `false` or omitted) is to generate numeric enums. Map key is the language name (e.g., "python", "rust"). Optional. If omitted, the generator default is used. |
| `open_api` | string | Is the file path to an OpenAPI spec, currently in internal/testdata. This is not an official spec yet and exists only for Rust to validate OpenAPI support. |
| `release_level` | map[string]string | Is the release level per language. Map key is the language name (e.g., "python", "rust"). Optional. If omitted, the generator default is used.<br><br>TODO(https://github.com/googleapis/librarian/issues/4834): Go uses "alpha", "beta", and "ga" instead of "preview" and "stable". We should standardize release level vocabulary across lanaguages. |
| `short_name` | string | Overrides the API short name from the service config's publishing section. |
| `service_config` | string | Is the service config file path override. If empty, the service config is discovered in the directory specified by Path. |
| `service_name` | string | Is a DNS-like logical identifier for the service, such as `calendar.googleapis.com`. |
| `title` | string | Overrides the API title from the service config. |
| `transports` | map[string]Transport | Defines the supported transports per language. Map key is the language name (e.g., "python", "rust"). Optional. If omitted, all languages use GRPCRest by default. |
| `grpc_service_config` | [GRPCServiceConfig](#grpcserviceconfig-configuration) (optional) | Contains inline gRPC service config data (retry/timeout settings). |

## GRPCServiceConfig Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `method_config` | list of [MethodConfig](#methodconfig-configuration) |  |

## MethodConfig Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | list of [MethodName](#methodname-configuration) | Identifies which service methods this config applies to. An empty Method field means all methods on the named service. |
| `timeout` | string | Is the maximum duration for an RPC, including all retry attempts. Uses proto3 Duration string format (e.g., "60s"). |
| `retry_policy` | [RetryPolicy](#retrypolicy-configuration) (optional) | Defines the retry behavior for matching methods. |

## MethodName Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `service` | string | Is the fully qualified gRPC service name (e.g., "google.cloud.secretmanager.v1.SecretManagerService"). |
| `method` | string | Is the method name (e.g., "GetSecret"). If empty, the config applies to all methods on the service. |

## RetryPolicy Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `max_attempts` | int | Is the maximum number of RPC attempts including the original request. Values above 5 are treated as 5. |
| `initial_backoff` | string | Is the delay before the first retry, in proto3 Duration format (e.g., "1s"). |
| `max_backoff` | string | Caps the retry delay, in proto3 Duration format (e.g., "10s"). |
| `backoff_multiplier` | float64 | Controls exponential growth of the delay between successive retry attempts. |
| `retryable_status_codes` | list of string | Lists the gRPC status codes that trigger a retry (e.g., "UNAVAILABLE"). |
