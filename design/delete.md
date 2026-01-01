Library Deletion Workflow
=========================

This document outlines the process for safely deleting a client library managed by `librarian` from a repository.

Goal
----

To remove a library's configuration from `librarian.yaml`, clean up its generated code, and ensure it is no longer managed by `librarian` for generation or release.

Prerequisites
-------------

1.	**Repository Configuration**: The `librarian.yaml` is up-to-date.
2.	**Tooling Installed**: The `librarian` CLI tool is installed.
3.	**Clean State**: It is recommended to start from a clean Git state on a fresh branch.

Deletion Workflow
-----------------

### 1. Create a Feature Branch

Start by creating a new Git branch for your deletion work. This ensures your changes are isolated and can be reviewed via a Pull Request.

```bash
git checkout -b feat/delete-<library-name>
```

### 2. Run `librarian delete`

Use the `librarian delete` command to remove the library's entry from your repository's `librarian.yaml` and clean up its associated generated code and files.

```bash
librarian delete <library-name>
# Example:
librarian delete google-cloud-oldservice
```

-	**Actions by `librarian delete`:**
	1.	**`librarian.yaml` Update**: The entry for `<library-name>` is removed from the `libraries` list in your `librarian.yaml`.
	2.	**Code Cleanup**: All generated files and directories associated with `<library-name>` (as determined by `generation.output` and any library-specific output overrides) are removed from the repository.
	3.	**Local Repository Integration**: The command stages the changes (removal from `librarian.yaml`, deleted files) for Git.

### 3. Review and Commit Changes

Review the changes made by `librarian delete`.

```bash
git status
git diff
```

Commit the changes, including the updated `librarian.yaml` and the deleted generated code.

```bash
git commit -m "feat(<library-name>): delete client library"
```

### 4. Open a Pull Request

Push your branch and open a Pull Request against the `main` branch of your repository. The PR will be reviewed by maintainers to ensure:

-	The `librarian.yaml` entry is correctly removed.
-	All generated files are cleaned up.
-	No unintended files were deleted.

### 5. Final Steps

Once the PR is merged, the library is successfully deleted and no longer managed by `librarian`.

Automation Integration (LibrarianOps)
-------------------------------------

In the future `librarianops` world, deletion might be triggered by:

-	An API being moved to a `deprecated` or `removed` status in `sdk.yaml`.
-	A manual trigger by the Librarian team to initiate a deletion PR across relevant language repositories.
