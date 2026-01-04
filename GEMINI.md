# Librarian

This guide defines the Go development standards for this repository. Apply
these instructions for all code generation, refactors, and review tasks across
the SDLC.

---

## 1. Write (Implementation)
* **Adhere to Local Standards:** Align all architectural decisions with
  [`doc/howwewritego.md`](../doc/howwewritego.md).
* **Idiomatic Go:** Follow
  [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
* **Vertical Density:** Use line breaks only to signal a shift in logic. Avoid
  unnecessary vertical padding or double blank lines.

## 2. Test (Verification)
* **Requirement:** Every new feature or bug fix must include a test.
* **Quality Gates:** Changes must not break the suite in `all_test.go`,
  specifically `TestGolangCILint` and `TestGoImports`.
* **Table-Driven Tests:** Use table-driven designs with `t.Run` for all
  logic-heavy functions.
* **Guidance:** Follow [Go Test Comments](https://go.dev/wiki/TestComments).

## 3. Review & Refactor (The Audit)
* **Self-Correction:** After generating code, perform a secondary pass to
  ensure it follows
  [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) and
  [Go Test Comments](https://go.dev/wiki/TestComments).
* **DRY Check:** Look for opportunities to reuse existing project utilities
  rather than creating new ones.
* **Simplification:** Refactor complex logic for readability. Prioritize being [clear over clever](https://www.youtube.com/watch?v=PAAkCSZUG1c&t=875s) and follow established [Go Proverbs](https://go-proverbs.github.io/) to ensure code remains simple and "boring."

## 4. Commit (Documentation)
* **Formatting:** Follow
  [`CONTRIBUTING.md` → Commit Messages](../CONTRIBUTING.md#commit-messages).
* **Imperative Mood:** Use the imperative mood (e.g., "Fix linter error," not
  "Fixed linter error").
* **Grammar:** Write commit messages in complete sentences with proper
  punctuation.
