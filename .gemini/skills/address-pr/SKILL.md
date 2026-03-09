---
name: address-pr
description: Use this skill if the user asks you to help them address GitHub PR comments for their current branch. Requires the `gh` CLI tool.
---
You are helping the user address comments on their Pull Request. These comments may have come from an automated review agent or a team member.

OBJECTIVE: Help the user review and address comments on their PR.

# Comment Review Procedure

1. Run `gh pr view --json reviews,comments,body` to get the latest PR info and state. Read the entire output.
2. Summarize the review status by analyzing the diff, commit log, and comments to see which still need to be addressed. Pay attention to the current user's comments.
   - For resolved threads, summarize as a single line with a ✅.
   - For open threads, provide a reference number e.g. [1] and the comment content.
3. Present your summary of the feedback and current state and allow the user to guide you as to what to fix/address/skip. DO NOT begin fixing issues automatically.

# Implementation Principles

When the user asks you to implement a fix for a specific comment:
- Apply the requested changes surgically.
- **Commit Strategy:** Create additional commits to address feedback instead of amending/force-pushing (as per `CONTRIBUTING.md`). Do not squash or amend unless specifically requested.
- **Style Consistency:** Ensure fixes follow [Effective Go](https://go.dev/doc/effective_go) and the project's [GEMINI.md](../../../GEMINI.md).
- Ensure all changes are validated by running tests: `go test -short ./...`.

# Communication Principles

When helping the user draft written replies to reviewers, adopt their specific communication style:
- **Explicit Blockquoting:** Use Markdown blockquotes (`>`) to quote the exact sentences from the reviewer you are responding to. Include nested blockquotes for previous context if the thread is deep.
- **Decisive Scope Management:** Actively prevent scope creep. If a comment raises a systemic issue outside the PR's original intent, suggest explicitly limiting the scope in the reply and offer to open a new, separate issue to track the systemic fix.
- **Bias Toward Action & Testing:** When system assumptions are unclear, propose adding test cases to capture edge cases and empirically prove out the system before making further changes.
