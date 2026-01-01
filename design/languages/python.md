Python Implementation Design
============================

This document details the architectural design of the Python support in `librarian`. It covers the transition from legacy container-based generation to a native Go implementation.

Overview
--------

The Python implementation aims to replicate the legacy `release-please` and Docker-based generator workflow within the native `librarian` binary. It orchestrates `protoc` for generation and `synthtool` for post-processing.

Configuration Architecture
--------------------------

### `librarian.yaml`

Python configuration is centered around the `python` block in `librarian.yaml`:

-	**`opt_args`**: A list of key-value pairs passed directly to the Python GAPIC generator (e.g., `warehouse-package-name`).
-	**`opt_args_by_channel`**: Allows channel-specific overrides for generation options.

Generation Pipeline
-------------------

1.	**Staging**:
	-	Code is generated into a temporary directory: `owl-bot-staging/{library_name}/{version}`.
	-	This isolation allows `protoc` to run cleanly without clobbering existing files immediately.
2.	**Protoc Invocation**:
	-	`librarian` constructs the `protoc` command line, injecting `transport`, `rest-numeric-enums`, and the configured `opt_args`.
	-	It auto-discovers `*_grpc_service_config.json` files to configure retry policies.
3.	**Synthesis (Post-Processing)**:
	-	`librarian` invokes `python3 -m synthtool`.
	-	Specifically, it calls `python_mono_repo.owlbot_main`, which moves files from the staging directory to their final destination, applying standard Python templates (e.g., `setup.py`, `noxfile.py`).
4.	**Metadata**:
	-	`.repo-metadata.json` is generated directly by `librarian` (Go logic), removing the dependency on external tools for this file.

Release Pipeline (Target Design)
--------------------------------

The Python release pipeline is currently being ported to Go. The target architecture is:

### 1. Preparation (`librarian release`\)

-	**Discovery**: Scans for `setup.py` or `pyproject.toml` files.
-	**Change Detection**: `git diff` logic similar to Rust.
-	**Versioning**:
	-	Extracts current version from `setup.py` (via Regex) or `pyproject.toml`.
	-	Calculates the next version.
-	**Manifest Update**:
	-	Performs in-place updates of `setup.py` and `src/__init__.py` using regex replacement to preserve formatting.
	-	Updates `librarian.yaml` version.

### 2. Publication (`librarian publish`\)

-	**Build**: Invokes `python3 -m build` to create `sdist` and `wheel` artifacts.
-	**Upload**: Uses `twine upload` to push to PyPI.

Constraints
-----------

-	**Runtime**: Requires Python 3.10+ and `protoc` in the environment.
-	**Synthtool**: Heavily relies on the `gcp-synthtool` library for the final mile of code structure.
