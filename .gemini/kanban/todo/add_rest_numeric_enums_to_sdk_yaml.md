Migrate `rest_numeric_enums` configuration to `sdk.yaml`.

- **Migration:** Review the existing data for `rest_numeric_enums` in `google-cloud-python/.librarian/state.yaml` and migrate it to the appropriate locations within `sdk.yaml`.
- **Implementation:**
    - Define the final structure and location for `rest_numeric_enums` within `sdk.yaml`.
    - Update the configuration parsing logic to load this value.
    - Ensure all relevant generator invocations or logic use this setting from `sdk.yaml`.