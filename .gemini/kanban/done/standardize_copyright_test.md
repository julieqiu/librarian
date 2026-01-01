## Standardize `copyright_test.go`

**Objective:** Refactor `internal/librarian/copyright_test.go` to align with project testing guidelines.

**Description:**
Update `internal/librarian/copyright_test.go` to:
- Use `github.com/google/go-cmp/cmp.Diff` for all assertion comparisons.
- Standardize error output messages to the format: `t.Errorf("mismatch (-want +got):
%s", diff)`.
This will improve test readability and debugging clarity.
