### Objective

This document proposes a container-based dependency management system for the `librarian` Go CLI that supports building distinct, language-specific toolkits for reproducible development environments.

### Background

The `librarian` CLI is a Go binary that orchestrates workflows involving various external, non-Go tools, including `protoc`, the Python toolchain (`pip`, `venv`, `black`), and the Rust toolchain (`cargo`).

Currently, there is no standardized way to manage these external dependencies. This leads to several critical problems:
*   **Developer Environment Drift:** Developers may have different, incompatible versions of these tools installed on their local machines, leading to inconsistent behavior and "it works on my machine" bugs.
*   **Complex Onboarding:** New contributors face a complex and error-prone setup process, requiring them to manually install and configure multiple toolchains correctly.
*   **CI/CD Brittleness:** Continuous integration builds must be carefully configured to replicate a developer's environment and are prone to breaking when system-level packages are updated.
*   **Inefficiency:** A single, monolithic toolkit containing all possible language dependencies is bloated and inefficient for teams that only require a subset of the tools (e.g., a Python team does not need the Rust toolchain).

This document proposes a container-based toolkit system to provide a reliable, reproducible, and flexible development environment for `librarian` users.

### Overview

The core principle of this design is to **separate the tool from its environment**. The `librarian` Go binary will be simplified to focus solely on its application logic, while its complex, multi-language environment will be provided by a versioned, self-contained Docker container.

This system is composed of three main components:

1.  **The Toolkit Container**: A versioned Docker image, built from a single multi-stage `Dockerfile`, that contains all necessary dependencies for a specific language. This container is the single source of truth for the development environment.
2.  **The Go CLI Binary (`librarian`)**: The compiled Go application, which is copied into the container. It is simplified to assume all its dependencies are available in the `PATH` provided by the container.
3.  **The User-Facing Wrapper Script**: A simple shell script that users install on their host machine. This script seamlessly executes `librarian` commands inside the appropriate toolkit container, making the experience feel native.

The design uses a multi-stage `Dockerfile` with the `--target` pattern to build different "flavors" of the toolkit. For example, a `python-toolkit` can be built for Python developers and a `rust-toolkit` for Rust developers, ensuring each image is minimal and contains only the necessary tools.

```
+---------------------------------+
|       Developer's Machine       |
|                                 |
|  +---------------------------+  |
|  |   librarian (Wrapper sh)  |  |
|  +---------------------------+  |
|               |               |
|      docker run ...           |
|               v               |
|  +---------------------------+  |
|  | librarian-python:v1.2.3   |  |
|  |---------------------------|
|  | (Go binary + Python env)  |
|  +---------------------------+
+---------------------------------+
+---------------------------------+
|    Docker Build Environment    |
|                                 |
|  +--------------------------+  |
|  | Multi-Stage Dockerfile   |  |
|  +--------------------------+  |
|        |          ^          |
|  build --target   |          |
|    python-toolkit |          |
|                   |
|  +--------------------------+  |
|  | your-org/librarian-rust  |
|  +--------------------------+
+---------------------------------+
```

### Detailed Design

#### 1. The Multi-Stage Toolkit Dockerfile

The environment and its dependencies are defined in a single, well-commented `Dockerfile`. This file contains several intermediate build stages and final, buildable "flavor" targets.

```dockerfile
# === Stage 1: Base builder with common system tools ===
# This stage provides a consistent base with tools needed by multiple builders.
FROM ubuntu:22.04 AS builder
RUN apt-get update && apt-get install -y \
    build-essential \
    curl \
    git \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# === Stage 2: Go builder for the librarian CLI ===
# This stage builds the librarian Go binary statically, ensuring high portability.
FROM golang:1.21 AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build a static binary to minimize runtime dependencies in the final image.
RUN CGO_ENABLED=0 go build -o /librarian ./cmd/librarian

# === Stage 3: Reusable Language Environment Builders ===
# These stages create the self-contained toolchains for each language. They
# are cached independently, providing efficient rebuilds.

# --- Protoc Builder ---
FROM builder AS protoc-builder
ARG PROTOC_VERSION="25.3"
RUN curl -Lo protoc.zip "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip" \
    && unzip protoc.zip -d /opt/protoc && rm protoc.zip

# --- Python Builder ---
FROM builder AS python-builder
RUN apt-get update && apt-get install -y python3.11 python3.11-venv && rm -rf /var/lib/apt/lists/*
# Create a virtual environment to isolate Python dependencies.
RUN python3.11 -m venv /opt/venv
# Copy and install requirements into the venv for a self-contained toolset.
COPY requirements.txt .
RUN /opt/venv/bin/pip install -r requirements.txt

# --- Rust Builder ---
FROM builder AS rust-builder
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
# Add cargo to the path for subsequent RUN commands in this stage.
ENV PATH="/root/.cargo/bin:${PATH}"
RUN cargo install cargo-semver-checks --version 0.44.0
RUN cargo install cargo-workspaces --version 0.4.0


# === Stage 4: Common Toolkit Base ===
# This intermediate stage creates a minimal, common foundation for all final
# toolkit flavors. It contains only the assets needed by every flavor,
# such as the librarian binary and protoc. This reduces duplication.
FROM ubuntu:22.04 AS toolkit-base
# Install only essential runtime dependencies.
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*
# Copy the compiled Go binary from the go-builder stage.
COPY --from=go-builder /librarian /usr/local/bin/librarian
# Copy the protoc binary and includes from the protoc-builder stage.
COPY --from=protoc-builder /opt/protoc/bin/* /usr/local/bin/
COPY --from=protoc-builder /opt/protoc/include/* /usr/local/include/
WORKDIR /work
ENTRYPOINT ["librarian"]


# === Stage 5: Final, Buildable Toolkit "Flavors" ===
# Each of these is a separate, buildable target that starts from our common
# base and adds a single language's toolchain.

# --- The Python Toolkit Flavor ---
# To build: docker build --target python-toolkit -t <tag> .
FROM toolkit-base AS python-toolkit
# Copy the entire pre-built Python virtual environment from the python-builder.
COPY --from=python-builder /opt/venv /opt/venv
# Add the venv's bin directory to the PATH.
ENV PATH="/opt/venv/bin:${PATH}"

# --- The Rust Toolkit Flavor ---
# To build: docker build --target rust-toolkit -t <tag> .
FROM toolkit-base AS rust-toolkit
# Copy the entire pre-built cargo home directory from the rust-builder.
COPY --from=rust-builder /root/.cargo /root/.cargo
# Add cargo's bin directory to the PATH.
ENV PATH="/root/.cargo/bin:${PATH}"
```

#### 2. The Go CLI Binary (`librarian`)

The Go application logic is significantly simplified. It is no longer responsible for finding, versioning, or managing its dependencies.
*   **Dependency Logic Removed**: All logic related to parsing manifests, checking for tool existence (`getToolPath`), and managing a local toolkit cache is removed.
*   **Direct Execution**: The binary executes tools like `protoc` and `cargo` directly using `os/exec.Command()`. It relies on the `PATH` variable provided by the container to resolve these executables.
*   **Statelessness**: The Go binary is effectively stateless regarding its environment, making it simpler, more robust, and easier to test.

#### 3. The User-Facing Wrapper Script

To provide a seamless user experience, a simple wrapper script is provided to users. This script is placed in the user's `PATH` (e.g., in `/usr/local/bin`).

**`librarian.sh`:**
```bash
#!/bin/bash
# Wrapper script for the librarian toolkit.
set -e

# The specific, versioned toolkit image to use.
# This could be parameterized based on a project's configuration file.
TOOLKIT_IMAGE="your-org/librarian-python:v1.2.3"

# Ensure Docker is available.
if ! command -v docker &> /dev/null;
    echo "Error: 'docker' is not installed or not in your PATH. Please install Docker to use librarian." >&2
    exit 1
fi

# Execute the librarian command inside the container, mounting the current
# directory to /work and passing all arguments to the entrypoint.
docker run --rm -it \
  -v "$(pwd):/work" \
  "${TOOLKIT_IMAGE}" "$@"
```

### Alternatives Considered

1.  **Go Binary Manages All Dependencies**: The initial design involved the Go binary parsing a manifest and installing tools into a local cache. This was rejected because it made the Go binary a complex, custom package manager, introduced a difficult bootstrap problem (requiring `python`, `cargo`, etc., to be pre-installed), and was philosophically misaligned with Go's goal of simplicity.

2.  **A Local Bootstrap Script (No Docker)**: A standalone `bootstrap.sh` script could be used to install a local toolkit on the host filesystem. This is a strong alternative that avoids a Docker dependency. However, it was not chosen as the primary design because it is less hermetic than a container (it depends on the host's compilers and system libraries) and cannot guarantee the same level of reproducibility across different operating systems and CI environments as a container can.
