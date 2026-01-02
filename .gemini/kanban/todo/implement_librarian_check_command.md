Implement `librarian check` command for quality assurance.

**Objective:**
Provide a single, unified command for the librarian team and developers to run the full suite of language-specific validation checks, ensuring consistency and reducing cognitive load during releases.

**Requirements:**
- Create a new command: `librarian check <library-name>`.
- The command should be language-aware and execute a suite of validation tools based on the library's language.

**Language-Specific Implementations:**
- **For Rust:**
    - `cd` into the library directory.
    - Execute `cargo test`.
    - Execute `cargo doc`.
    - Execute `cargo clippy`.
- **For Python:**
    - `cd` into the library directory.
    - Execute `nox -s test` (or a default set of `nox` sessions).
- **For Go:**
    - `cd` into the library directory.
    - Execute `go test ./...`.
    - Execute `go vet ./...`.