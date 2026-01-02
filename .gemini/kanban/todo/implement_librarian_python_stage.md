Implement `librarian stage` command for Python.

- **Create `stage.go`:** Create a new file `internal/librarian/python/stage.go`.
- **Implement Version Update Logic:** Create a `Stage` function that:
    1. Calculates the next semantic version based on Git commit history.
    2. Updates the `version` field in the `librarian.yaml` file on disk.
    3. Reads the `setup.py` and `__init__.py` files, updates the version fields within them, and writes the files back.
