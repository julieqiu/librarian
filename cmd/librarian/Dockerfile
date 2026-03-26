# syntax=docker/dockerfile:1
# The BuildKit syntax directive enables BuildKit features such as cache mounts
# for improved build performance.

# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Prerequisites:
#   To build this image, you must have access to marketplace.gcr.io. Run the
#   following commands to authenticate and configure Docker to use your gcloud
#   credentials:
#
#   gcloud auth login
#   gcloud auth configure-docker marketplace.gcr.io
#
# Build:
#   docker build --target rust -t librarian:rust -f cmd/librarian/Dockerfile .
#   docker build --target python -t librarian:python -f cmd/librarian/Dockerfile .
#
# Usage:
#   docker run --rm -v "$(pwd)":/workspace -w /workspace librarian:rust generate
#   docker run --rm -v "$(pwd)":/workspace -w /workspace librarian:python generate
#

ARG TARGETARCH
ARG GO_VERSION=1.26.1
ARG RUST_VERSION=1.92

# ============== Build Stage ==============
# Use the MOSS-compliant base image.
FROM marketplace.gcr.io/google/debian12@sha256:326ccf35aa72f7cc8cbd25b69e819a81c5fd9ed675c6c9ffb51f3214a64f23cf AS build
ARG GO_VERSION
ARG TARGETARCH
ENV GO_VERSION=$GO_VERSION

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    curl \
    git \
    wget && \
    rm -rf /var/lib/apt/lists/*

RUN wget -q https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz -O go.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm go.tar.gz
ENV PATH=/usr/local/go/bin:$PATH

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /go/bin/librarian ./cmd/librarian

# ============== Base Stage ==============
FROM debian:bookworm-slim AS base

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    unzip curl git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /go/bin/librarian .
ENTRYPOINT ["/app/librarian"]

# ============== Rust Stage ==============
FROM base AS rust
ARG RUST_VERSION

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install Rust toolchain
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

RUN rustup install ${RUST_VERSION} && \
    rustup default ${RUST_VERSION} && \
    rustup component add clippy rustfmt

RUN /app/librarian setup rust

# Make sure the tools are executable by any user (not just root).
# librarianops executes the docker image as the current user.
RUN chmod a+wx /root && \
    chmod -R a+w /root/.cargo
ENV HOME=/root

# ============== Python Stage ==============
FROM base AS python

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip python3-venv \
    && rm -rf /var/lib/apt/lists/*

RUN /app/librarian setup python

# ============== Go Stage ==============
FROM base AS go
ARG GO_VERSION
ARG TARGETARCH

ENV GO_VERSION=$GO_VERSION

RUN apt-get update && apt-get install -y --no-install-recommends \
    wget \
    && rm -rf /var/lib/apt/lists/*

RUN wget -q https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz -O go.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm go.tar.gz
ENV PATH=/root/go/bin:/usr/local/go/bin:$PATH

RUN /app/librarian setup go

# Make sure the tools are executable by any user (not just root).
# librarianops executes the docker image as the current user.
RUN chmod a+rx /root/go/bin/* && \
    chmod a+x /root && \
    chmod a+x /root/go && \
    chmod a+x /root/go/bin
