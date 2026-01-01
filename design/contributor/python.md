# Python Contributor Guide

This guide is intended for contributors to the `google-cloud-python` repository. It details the workflows for generating and maintaining client libraries using the `librarian` tool.

## Prerequisites

Ensure you have the following installed:

1.  **Librarian CLI**: The latest version of the `librarian` binary.
2.  **Python 3.10+**: Required for `synthtool` and post-processing.
3.  **Protoc**: Protocol Buffer compiler (v23.0+).

```bash
librarian version
python3 --version
protoc --version
```

## Workflows

### Generate a New Library

To onboard a new API (e.g., `google-cloud-secret-manager`), use the `create` command.

```bash
# 1. Create a feature branch
git checkout -b feat-new-library

# 2. Run librarian create
# Usage: librarian create <library-id> [api-path]
librarian create google-cloud-secret-manager google/cloud/secretmanager/v1

# 3. Verify and Commit
git add librarian.yaml packages/google-cloud-secret-manager
git commit -m "feat: add google-cloud-secret-manager"
```

### Regenerate Libraries

When upstream protos change or the generator logic updates:

```bash
# Regenerate fleet-wide
librarian generate --all

# Regenerate a single library
librarian generate google-cloud-secret-manager
```

### Release (Preview)

**Note:** The native release pipeline is currently in preview.

```bash
# Prepare Release (Updates setup.py, librarian.yaml)
librarian release google-cloud-secret-manager
```

## Troubleshooting

*   **Protoc not found**: Ensure `protoc` is in your `$PATH`.
*   **Synthtool errors**: Ensure you have installed the necessary Python dependencies (`pip install gcp-synthtool`).
