# gcloud Declarative YAML Schema

This document describes the schema for the gcloud Declarative YAML.

## ArgSpec Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `api_field` | string | Is the name of the request field this spec entry maps to. |

## Argument Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `arg_name` | string | Is the name of the argument as it appears on the command line. |
| `api_field` | string | Is the name of the request field this argument maps to. |
| `help_text` | string | Is the help text shown for this argument. |
| `action` | string | Specifies a special argument action, such as "store_true". |
| `is_positional` | bool | Reports whether this argument is a positional argument. |
| `is_primary_resource` | bool | Reports whether this argument refers to the command's primary resource. |
| `request_id_field` | string | Names the request field used to carry this argument's identifier. |
| `resource_spec` | [ResourceSpec](#resourcespec-configuration) (optional) | Describes the resource referenced by this argument. |
| `required` | bool | Reports whether the argument must be supplied by the user. |
| `repeated` | bool | Reports whether the argument may be supplied multiple times. |
| `clearable` | bool | Reports whether the argument's value can be cleared. |
| `type` | string | Specifies the argument's type when not a plain string. |
| `default` | [Default](#default-configuration) | Holds the argument's default value, if any. |
| `choices` | list of [Choice](#choice-configuration) | Lists the allowed values for a choice-typed argument. |
| `spec` | list of [ArgSpec](#argspec-configuration) | Lists nested argument specifications. |
| `resource_method_params` | map[string]string | Maps method parameter names to request field paths. |

## Arguments Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `params` | list of [Argument](#argument-configuration) | Holds the list of argument definitions for the command. |

## Async Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `collection` | list of string | Is the resource collection path of the operation resource. |
| `extract_resource_result` | bool | Extracts the target resource from the operation result when true. |

## Attribute Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `parameter_name` | string | Is the name of the parameter in the API request. |
| `attribute_name` | string | Is the name of the attribute in the resource spec. |
| `help` | string | Is the help text shown for this attribute. |
| `property` | string | Names the gcloud property that supplies a default value for this attribute. |

## Choice Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `arg_value` | string | Is the value the user types on the command line. |
| `enum_value` | string | Is the corresponding API enum value. |
| `help_text` | string | Is the help text shown for this choice. |

## Command Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `release_tracks` | list of string | Lists the gcloud release tracks (GA, BETA, ALPHA) that this command definition applies to. |
| `auto_generated` | bool | Indicates that the YAML file was produced by a generator and should not be edited by hand. |
| `hidden` | bool | Hides the command from help listings when true. |
| `help_text` | [HelpText](#helptext-configuration) | Holds the command's help text. |
| `arguments` | [Arguments](#arguments-configuration) | Lists the command's positional and flag arguments. |
| `request` | [Request](#request-configuration) (optional) | Describes the API request issued by the command, if any. |
| `async` | [Async](#async-configuration) (optional) | Describes the long-running operation behavior of the command, if any. |
| `response` | [Response](#response-configuration) (optional) | Describes how to interpret the command's response, if any. |
| `update` | [Update](#update-configuration) (optional) | Describes how update commands handle field masks, if applicable. |
| `output` | [Output](#output-configuration) (optional) | Describes how the command's output should be formatted, if specified. |

## Default Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `Value` | any (optional) | Is the underlying default value emitted when present. |

## HelpText Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `brief` | string | Is a single-line summary of the command. |
| `description` | string | Is the long-form description of the command. |
| `examples` | string | Provides example invocations of the command. |

## Output Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `format` | string | Specifies the gcloud output format string for the command. |

## Request Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `api_version` | string | Is the API version used to issue the request. |
| `collection` | list of string | Is the resource collection path used to issue the request. |
| `method` | string | Is the API method invoked by the command. |
| `static_fields` | map[string]string | Sets request fields to fixed values. |

## ResourceSpec Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is the singular resource name. |
| `plural_name` | string | Is the plural resource name. |
| `collection` | string | Is the resource's collection identifier. |
| `attributes` | list of [Attribute](#attribute-configuration) | Lists the attributes that identify the resource. |
| `disable_auto_completers` | bool | Turns off auto-completion for this resource when true. |

## Response Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `id_field` | string | Names the field on the response that identifies the resource. |

## Update Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `read_modify_update` | bool | Causes the command to read the resource, apply changes, then write it back when true. |
| `disable_auto_field_mask` | bool | Disables automatic generation of the update field mask when true. |
