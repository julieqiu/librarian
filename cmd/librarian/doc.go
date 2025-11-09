// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate go run -tags docgen ../doc_generate.go -cmd .

/*
Librarian manages Google API client libraries by automating onboarding,
regeneration, and release. It runs language-agnostic workflows while
delegating language-specific tasks—such as code generation, building, and
testing—to Docker images.

Usage:

	librarian <command> [arguments]

The commands are:

# init

Initializes a repository for library management.

	librarian init [language]

Creates .librarian.yaml at repository root. If language is provided (go, python,
rust, dart), adds librarian.language and generate section with defaults. Always
adds release section with default tag_format.

Examples:

	# Release-only repository
	librarian init

	# Repository with code generation and releases
	librarian init python

Usage:

	librarian init [language]

Flags:

	-v	enables verbose logging

# add

Tracks a directory for management.

	librarian add <path> [api...]

Creates <path>/.librarian.yaml. If APIs are provided AND repository has a
generate section, parses BUILD.bazel files and creates generate section with
API configurations. If repository has a release section, adds release.version: null.

The --commit flag writes a standard commit message for the change.

Examples:

	# Add handwritten code (no APIs)
	librarian add packages/my-tool

	# Add generated code (with APIs)
	librarian add packages/google-cloud-secret-manager secretmanager/v1 secretmanager/v1beta2 --commit

Usage:

	librarian add <path> [api...]

Flags:

	-commit
	  	create a git commit with the changes
	-v	enables verbose logging

# edit

Edits artifact configuration.

	librarian edit <path> [flags]

Configure artifact-specific settings like metadata, keep/remove/exclude lists,
and language-specific metadata. Running edit without flags displays current
configuration.

Examples:

	# Set metadata fields
	librarian edit packages/google-cloud-secret-manager \
	  --metadata name_pretty="Secret Manager" \
	  --metadata release_level=stable

	# Set language-specific metadata
	librarian edit packages/my-package --language python:package=my-package

	# Configure file handling
	librarian edit packages/my-tool --keep README.md --remove temp.txt --exclude tests

Usage:

	librarian edit <path> [flags]

Flags:

	-exclude value
	  	add file/directory to exclude list (can be repeated)
	-keep value
	  	add file/directory to keep list (can be repeated)
	-language string
	  	set language-specific metadata (LANG:KEY=VALUE)
	-metadata value
	  	set metadata field (KEY=VALUE, can be repeated)
	-remove value
	  	add file/directory to remove list (can be repeated)
	-v	enables verbose logging

# remove

Stops tracking a directory.

	librarian remove <path>

Removes <path>/.librarian.yaml. Source code is not modified.

Example:

	librarian remove packages/my-tool

Usage:

	librarian remove <path>

Flags:

	-v	enables verbose logging

# generate

Generates or regenerates code for tracked directories.

For artifacts with a generate section in their .librarian.yaml:

	librarian generate <path>

Generates or regenerates code using the container and configuration from
.librarian.yaml. Librarian updates the artifact's .librarian.yaml automatically.

Generate all artifacts that have a generate section:

	librarian generate --all

The --commit flag writes a standard commit message for the change.

Example:

	librarian generate packages/google-cloud-secret-manager --commit

Usage:

	librarian generate <path> | librarian generate --all

Flags:

	-all
	  	generate all artifacts with a generate section
	-commit
	  	create a git commit with the changes
	-v	enables verbose logging

# prepare

Prepares a release for tracked directories.

For artifacts with a release section in their .librarian.yaml:

	librarian prepare <path>

Determines the next version, updates metadata, and prepares release notes.
Does not tag or publish.

Prepare all artifacts that have a release section:

	librarian prepare --all

The --commit flag writes a standard commit message for the change.

Example:

	librarian prepare packages/google-cloud-secret-manager --commit

Usage:

	librarian prepare <path> | librarian prepare --all

Flags:

	-all
	  	prepare all artifacts with a release section
	-commit
	  	create a git commit with the changes
	-v	enables verbose logging

# release

Publishes a prepared release for tracked directories.

For artifacts with a release section and a prepared release:

	librarian release <path>

Tags the prepared version and updates recorded release state. If no prepared
release exists, the command does nothing.

Release all prepared artifacts:

	librarian release --all

Example:

	librarian release packages/google-cloud-secret-manager

Usage:

	librarian release <path> | librarian release --all

Flags:

	-all
	  	release all artifacts with a prepared release
	-v	enables verbose logging

# config

# Config manages repository configuration stored in .librarian.yaml

Usage:

	librarian config <command> [arguments]

Commands:

	get                        reads a configuration value
	set                        sets a configuration value
	update                     updates toolchain versions to latest

# config get

Reads a configuration value from .librarian.yaml.

	librarian config get <key>

Supported keys include librarian.language, generate.container.image, release.tag_format, etc.

Example:

	librarian config get generate.container.image

Usage:

	librarian config get <key>

Flags:

	-v	enables verbose logging

# config set

Sets a configuration value in .librarian.yaml.

	librarian config set <key> <value>

Supported keys include:
- librarian.language
- generate.dir
- generate.container.image
- generate.container.tag
- generate.container (syntactic sugar for image:tag)
- generate.googleapis.repo
- generate.googleapis.ref
- generate.discovery.repo
- generate.discovery.ref
- release.tag_format

Examples:

	# Set global generation directory
	librarian config set generate.dir packages

	# Set container image and tag
	librarian config set generate.container python-gen:v1.2.0

Usage:

	librarian config set <key> <value>

Flags:

	-v	enables verbose logging

# config update

Updates toolchain versions to latest.

	librarian config update [key]
	librarian config update --all

Supported keys:
- generate.container - Update container image to latest
- generate.googleapis - Update googleapis to latest commit
- generate.discovery - Update discovery-artifact-manager to latest commit

Examples:

	# Update container to latest
	librarian config update generate.container

	# Update all toolchain versions
	librarian config update --all

Usage:

	librarian config update [key] | librarian config update --all

Flags:

	-all
	  	update all toolchain versions
	-v	enables verbose logging

# version

Version prints version information for the librarian binary.

Usage:

	librarian version
*/
package main
