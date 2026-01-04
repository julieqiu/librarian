# Librarian

Librarian is a unified command line tool for Google Cloud SDK client library
configuration, generation, and releasing.

This guide defines the Go development standards for this repository. Apply
these instructions for all code generation, refactors, and review tasks across
the SDLC.

## 1. Plan and Design

- **Adhere to Local Standards:** Align all architectural decisions with
  [`doc/howwewritego.md`](../doc/howwewritego.md).
- **Codebase Map:**
  - `cmd/`: Main entrypoint to CLI commands.
  - `doc/`: All documentation for the project.
  - `tool/`: Internal tools.
  - `internal/`: Private application code.
  - `internal/command/`: **ALWAYS** use this package to execute shell
    commands. Do not use `os/exec` directly.
  - `internal/yaml/`: **ALWAYS** use this package for YAML operations. Do
    not use external YAML libraries directly.
- **Ignore:**
  - `doc/legacylibrarian/`: **DO NOT READ/EDIT** this directory unless
    explicitly asked; this is legacy documentation.
  - `internal/legacylibrarian/`: **DO NOT READ/EDIT** this directory unless
    explicitly asked; this is legacy code.
- **Dependencies:** **DO NOT** add new external dependencies to the project.
  Use what is already available in the codebase.
- **Before you write:** Explain your plan first. Identify which files need to
  change and why.

## 2. Write (Implementation)

- **Follow [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).**
- **Command Execution:** Use the `internal/command` package for all external
  process execution to ensure consistent logging and error handling.
  - Use `command.Run(ctx, ...)` when you only care about the error.
  - Use `command.Output(ctx, ...)` when you need the stdout/stderr.
- **Vertical Density:** Use line breaks only to signal a shift in logic.
  Avoid unnecessary vertical padding.
- **Naming:** Use singular form for package/folder names (e.g., `image/`, not
  `images/`).

## 3. Test (Verification)

- **Adhere to Local Standards:** Align all testing decisions with
  [`doc/howwewritego.md` → Writing Tests](doc/howwewritego.md#writing-tests).
- **Follow [Go Test Comments](https://go.dev/wiki/TestComments).**
- **Requirement:** Every new feature or bug fix must include a test.
- **Quality Gates:** Changes must not break `TestGolangCILint` and
  `TestGoImports` in `all_test.go`.
- **Test Context:** **ALWAYS** use `t.Context()` instead of
  `context.Background()`.
- **Temp Dirs:** **ALWAYS** use `t.TempDir()` for file system tests.
- **Comparisons:** **ALWAYS** use
  [`cmp.Diff`](https://pkg.go.dev/github.com/google/go-cmp/cmp) for assertions.
  Do not use `reflect.DeepEqual`.
  - Format: `t.Errorf("mismatch (-want +got):\n%s", diff)`
- **Table-Driven Tests:** Use table-driven designs with `t.Run` for
  logic-heavy functions.
  ```go
  for _, test := range []struct {
      name string
      arg  string
      want string
  }{
      {"success", "input", "output"},
  } {
      t.Run(test.name, func(t *testing.T) {
          got := Do(test.arg)
          if diff := cmp.Diff(test.want, got); diff != "" {
              t.Errorf("mismatch (-want +got):\n%s", diff)
          }
      })
  }
  ```

## 4. Review & Refactor (The Audit)

- **Scope:** Keep changes focused and small. Do not perform massive
  refactorings unless explicitly requested.
- **Self-Correction:** Perform a secondary pass to ensure code follows
  [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
- **DRY Check:** Reuse existing project utilities (especially
  `internal/command` and `internal/testhelper`).
- **Simplification:** Refactor complex logic for readability. Prioritize
  being clear over clever.

## 5. Commit (Documentation)

- **Atomic Commits:** Keep commits small and atomic. Each commit should
  represent a single logical change.
- **Formatting:** Follow
  [`CONTRIBUTING.md` → Commit Messages](../CONTRIBUTING.md#commit-messages).
- **Structure:** `<type>(<scope>): <description>` (e.g., `feat(internal/git):
  add diff helper`).
- **Imperative Mood:** Use the imperative mood for the summary line (e.g.,
  "fix bug", not "fixed bug").
- **Referencing Issues:** Use "Fixes #123" to close or "For #123" for partials.
  Never "Fix #123".

## 6. Tool Usage & Formatting

- **Auto-Format:** AFTER using `replace` or `write_file` on source code, you
  **MUST** immediately run the appropriate formatter (e.g., `gofmt -w` and
  `goimports -w`). Do not leave this for later.
- **Vertical Density:** Group related lines of code tightly. Use blank lines
  sparingly to separate logical steps, like paragraphs in writing.
  **Formatters like `gofmt` do not fix this for you; you must author dense
  code.**
  - Group related variable declarations of the same type on one line (e.g.,
    `var a, b []string`) or use a `var (...)` block for multiple declarations.
- **Whitespace Hygiene:** When constructing `replace` blocks, ensure
  `new_string` has the exact same surrounding vertical whitespace as the code
  it replaces unless you specifically intend to add/remove lines.
- **YAML Files:** Run `yamlfmt` for all YAML files.
- **Markdown Files:** Run `go run ./tool/cmd/mdformat -w` for all Markdown
  files.
