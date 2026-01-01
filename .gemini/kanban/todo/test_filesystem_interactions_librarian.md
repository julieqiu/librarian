## Test Filesystem Interactions in `internal/librarian`

**Objective:** Establish robust and isolated tests for functions interacting with the filesystem in `internal/librarian`.

**Description:**
Create new or enhance existing tests for the following files/functions:
- `generate.go`: `cleanOutput` and overall file generation logic.
- `release.go`: Logic that modifies `librarian.yaml` or other configuration files.
- `create.go`: `addLibraryToLibrarianConfig`.

**Strategy:**
- Utilize `t.TempDir()` for each test to ensure a clean, isolated filesystem environment.
- Develop structured test helper functions to set up and tear down mock filesystem states (e.g., creating mock `librarian.yaml` files, fake source files).

**Note to Scrum Master:** This task is critical for ensuring the reliability of file manipulation operations and should be assigned accordingly.
