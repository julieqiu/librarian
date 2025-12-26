# Multiple Python Runtimes in Librarian Image

## The Problem

The Python Librarian image currently supports two Python runtimes: 3.13 and 3.14. This dual-runtime support is a temporary measure necessitated by the fact that some Python client library repositories (e.g., `python-storage`) have not yet fully migrated and achieved compatibility with Python 3.14.

Supporting multiple runtimes in the Librarian image introduces:

*   **Increased Complexity:** The build process for the image is more complex, and there's a higher maintenance burden to ensure both runtimes are correctly configured and isolated.
*   **Larger Image Size:** Including an additional runtime increases the overall size of the Librarian Docker image.
*   **Testing Overhead:** Every generator update or tooling change potentially needs to be tested against both runtimes, doubling testing effort.
*   **Technical Debt:** The presence of the older runtime perpetuates the delay in migrating dependent repositories.

## Goal

To consolidate the Python Librarian image to support a single, modern Python runtime (3.14) exclusively. This will simplify maintenance, reduce image size, and align with Python's standard lifecycle (where older versions eventually reach end-of-life).

## Solution Strategy

The migration to a single Python 3.14 runtime will be a phased approach, tightly coupled with the migration status of dependent client library repositories.

1.  **Repository Migration:** Language teams (e.g., Python team) will be responsible for updating their client library repositories to achieve full compatibility with Python 3.14. This includes updating dependencies, fixing any compatibility issues, and updating their CI/CD to use Python 3.14 for builds and tests.
    *   **Tracking:** Progress will be tracked via internal issue b/375664027.

2.  **Librarian Image Update:** Once all critical Python repositories have demonstrated full compatibility with Python 3.14 and no longer require Python 3.13, the Librarian Engineering team will update the Python Librarian image to remove Python 3.13 support.
    *   This will involve modifying the Dockerfile and build scripts for the Python Librarian image.

3.  **Deprecation/Removal:** The older Python 3.13 runtime will be fully deprecated and removed from the Librarian image.

## Impact

*   **Language Teams:** Repositories not yet compatible with Python 3.14 will need to prioritize their migration efforts.
*   **Librarian Engineering Team:** Reduced maintenance burden, smaller image size, and simplified testing matrix.

## Related Issues

*   Internal Issue: b/375664027 (Tracking Python 3.14 compatibility across repositories)
