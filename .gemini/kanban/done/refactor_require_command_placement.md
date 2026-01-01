## Refactor `testhelper.RequireCommand` Placement

**Objective:** Standardize the placement of external command checks in tests to enforce consistent requirements for all test cases within a single test function.

**Description:**
In all test files that use `testhelper.RequireCommand`, move the call from inside table-driven test loops (`t.Run`) to be the first line of the main test function (e.g., `TestSomeFunction(t *testing.T)`).

**Reasoning:**
This change ensures that if an external command (like `git`) is required, it is a prerequisite for all test cases within that function. This makes the test's dependencies explicit and consistent, skipping the entire test function if the required command is not available, rather than skipping individual sub-tests.

**Example (Current):**
```go
func TestGitIntegration(t *testing.T) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testhelper.RequireCommand(t, "git")
			// ... test logic
		})
	}
}
```

**Example (Target):**
```go
func TestGitIntegration(t *testing.T) {
	testhelper.RequireCommand(t, "git") // Moved to the top
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// ... test logic
		})
	}
}
```

**Note to Scrum Master:** This is a refactoring task to improve test consistency and clarity of dependencies.
