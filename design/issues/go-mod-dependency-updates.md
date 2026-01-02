# When are go.mod dependencies updated?

## Question
This document addresses a core design question for the Go implementation: What is the defined process and trigger for updating the dependencies listed in a library's `go.mod` file?

## Background
The `librarian generate` command is designed to be idempotent, meaning it regenerates all necessary files, including `go.mod`, from templates on every run.

However, it is unclear when the versions of the dependencies *within* that `go.mod` file should be updated. If `librarian generate` always regenerates the `go.mod` file with pinned, templated versions, the dependencies could become stale over time. Conversely, if `generate` always tries to fetch the latest dependencies, it could be slow and introduce unexpected updates.

This issue needs to be resolved to ensure a clear, predictable workflow for Go library maintainers.

## Design Considerations

There are a few potential approaches:

1.  **Dependencies are only updated by `librarian update`:** In this model, `librarian generate` would be responsible for code generation only. A separate `librarian update` command would be responsible for running `go get -u` or a similar command to refresh the dependencies in `go.mod` and `go.sum`. This provides a clear separation of concerns.

2.  **Dependencies are updated on every `librarian generate`:** This model would ensure that every generation run produces a library with the latest possible dependencies. However, this could significantly slow down the `generate` command and might lead to unintended dependency bumps during routine code regeneration.

3.  **Dependencies are managed manually:** In this model, `librarian` would generate the `go.mod` file, but developers would be expected to run `go get -u` themselves. This gives developers control but undermines the goal of `librarian` as a comprehensive management tool.

A decision needs to be made on which model provides the best balance of automation, predictability, and developer control.
