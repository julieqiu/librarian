# Release Process Design

This document describes the release process for Librarian-managed client libraries.

## Overview

Librarian uses a **git-based release workflow** where:
- **Git history** is the source of truth for what needs to be released
- **Version files** in each library indicate pending releases
- **Conventional commits** determine the type of version bump
- **No GitHub dependency** - works with any git hosting

The release process has three phases:
1. **Prepare** - Bump versions and update changelogs
2. **Tag** - Create git tags and GitHub releases
3. **Publish** - Upload packages to registries (PyPI, pkg.go.dev, crates.io)

## librarian release prepare

Detects libraries with pending releases and updates their version files and changelogs.

### What it does

1. **Detects pending releases** by comparing version files between the last git tag and HEAD
2. **Determines version bump type** (major, minor, patch) from conventional commits
3. **Updates version files** with the new version
4. **Updates changelog files** with commit history since last release
5. **Creates a commit** with all version and changelog updates

### Usage

```bash
# Prepare all libraries with pending releases
librarian release prepare

# Prepare specific library
librarian release prepare --library=secretmanager

# Dry run to see what would be prepared
librarian release prepare --dry-run
```

### Version Detection

The command compares version files between the last tag and HEAD:

```bash
# Find last tag for library
git describe --abbrev=0 --match "secretmanager/v*"
# secretmanager/v1.11.0

# Read version at tag
git show secretmanager/v1.11.0:secretmanager/internal/version.go
# const Version = "1.11.0"

# Read version at HEAD
cat secretmanager/internal/version.go
# const Version = "1.12.0"

# Version changed → pending release!
```

### Version Bump Type

The command analyzes conventional commits since the last tag:

- **Major bump** (X.0.0) - Any commit with `!` (breaking change): `feat!:`, `fix!:`, `BREAKING CHANGE:`
- **Minor bump** (0.X.0) - `feat:` commits
- **Patch bump** (0.0.X) - `fix:` commits, `chore:`, `docs:`, etc.

Example:
```bash
# Since secretmanager/v1.11.0
git log secretmanager/v1.11.0..HEAD --oneline -- secretmanager/

feat(secretmanager): add Secret rotation support    → minor bump
fix(secretmanager): handle nil pointers correctly   → patch bump
```

Result: 1.11.0 → 1.12.0 (minor wins over patch)

### Changelog Format

The command generates changelogs in Keep a Changelog format:

```markdown
# Changelog

## [1.12.0] - 2025-01-15

### Added
- Add Secret rotation support

### Fixed
- Handle nil pointers correctly in GetSecret

## [1.11.0] - 2025-01-01
...
```

### Output

After running `librarian release prepare`:

```
Detected 3 libraries with pending releases:
  - secretmanager: 1.11.0 → 1.12.0 (minor)
  - pubsub: 2.5.0 → 2.5.1 (patch)
  - spanner: 3.2.1 → 4.0.0 (major - breaking change)

Updated files:
  secretmanager/internal/version.go
  secretmanager/CHANGELOG.md
  pubsub/internal/version.go
  pubsub/CHANGELOG.md
  spanner/internal/version.go
  spanner/CHANGELOG.md

Created commit: chore(release): prepare secretmanager v1.12.0, pubsub v2.5.1, spanner v4.0.0
```

### Validation

The command runs language-specific validation:

- **Go**: `go build ./...` and `go test ./... -short`
- **Python**: `nox -s unit`
- **Rust**: `cargo semver-checks` (validates no breaking changes in patch/minor releases)

If validation fails, the command exits with an error and does not create a commit.

### Error Handling

**No version file found**:
```
Error: No version file found for library 'secretmanager'
Expected one of:
  - secretmanager/internal/version.go
  - secretmanager/pyproject.toml
  - secretmanager/Cargo.toml
```

**No commits since last release**:
```
Warning: Library 'pubsub' has no commits since v2.5.0
Skipping release preparation.
```

**Version not bumped correctly**:
```
Error: Library 'spanner' has breaking changes but version is not a major bump
Last version: 3.2.1
Current version: 3.3.0
Expected version: 4.0.0

Breaking commits:
  - feat!: remove deprecated SpannerClient.Query method
```

## librarian release tag

Creates git tags and GitHub releases for libraries that have been prepared.

### What it does

1. **Finds prepared releases** by comparing version files at HEAD vs last tag
2. **Creates git tags** for each library at HEAD
3. **Pushes tags** to remote repository
4. **Creates GitHub releases** (optional) with changelog content

### Usage

```bash
# Tag all prepared releases
librarian release tag

# Tag specific library
librarian release tag --library=secretmanager

# Create tags locally but don't push
librarian release tag --no-push

# Skip GitHub release creation
librarian release tag --no-github-release
```

### Tag Format

Tags follow the format: `{library}/v{version}`

Examples:
- `secretmanager/v1.12.0`
- `pubsub/v2.5.1`
- `spanner/v4.0.0`

### Output

After running `librarian release tag`:

```
Creating tags for 3 libraries:

secretmanager/v1.12.0
  Tag created: secretmanager/v1.12.0
  Pushed to origin
  GitHub release created: https://github.com/googleapis/google-cloud-go/releases/tag/secretmanager%2Fv1.12.0

pubsub/v2.5.1
  Tag created: pubsub/v2.5.1
  Pushed to origin
  GitHub release created: https://github.com/googleapis/google-cloud-go/releases/tag/pubsub%2Fv2.5.1

spanner/v4.0.0
  Tag created: spanner/v4.0.0
  Pushed to origin
  GitHub release created: https://github.com/googleapis/google-cloud-go/releases/tag/spanner%2Fv4.0.0

All releases tagged successfully.
```

### GitHub Release

If `--no-github-release` is not set, the command creates a GitHub release for each tag using the `gh` CLI:

```bash
gh release create secretmanager/v1.12.0 \
  --title "secretmanager v1.12.0" \
  --notes-file secretmanager/CHANGELOG.md
```

The release notes are extracted from the library's CHANGELOG.md file.

### Error Handling

**No prepared releases found**:
```
Error: No prepared releases found.
Run 'librarian release prepare' first.
```

**Tag already exists**:
```
Error: Tag 'secretmanager/v1.12.0' already exists
If you want to re-release, delete the tag first:
  git tag -d secretmanager/v1.12.0
  git push origin :secretmanager/v1.12.0
```

**Working directory not clean**:
```
Error: Working directory is not clean.
Commit or stash your changes before tagging releases.
```

## librarian release publish

Publishes tagged releases to package registries.

### What it does

1. **Finds tagged releases** that haven't been published yet
2. **Validates the release** using language-specific tools
3. **Publishes to registry**:
   - **Go**: No-op (pkg.go.dev indexes automatically from git tags)
   - **Python**: `twine upload` to PyPI
   - **Rust**: `cargo publish` to crates.io

### Usage

```bash
# Publish all tagged releases
librarian release publish

# Publish specific library
librarian release publish --library=secretmanager

# Dry run to see what would be published
librarian release publish --dry-run
```

### Go Publishing

For Go libraries, no action is needed. The command verifies the tag exists and pkg.go.dev will automatically index it:

```
secretmanager/v1.12.0
  Tag verified: secretmanager/v1.12.0
  No action needed - pkg.go.dev will automatically index this release
  Track indexing status: https://pkg.go.dev/cloud.google.com/go/secretmanager/apiv1
```

### Python Publishing

For Python libraries, the command:
1. Checks out the tag
2. Builds the distribution with `python -m build`
3. Uploads to PyPI with `twine upload`

```
google-cloud-secret-manager v1.12.0
  Checked out tag: secretmanager/v1.12.0
  Building distribution...
  Built: dist/google-cloud-secret-manager-1.12.0.tar.gz
  Built: dist/google_cloud_secret_manager-1.12.0-py3-none-any.whl
  Uploading to PyPI...
  Published: https://pypi.org/project/google-cloud-secret-manager/1.12.0/
```

### Rust Publishing

For Rust libraries, the command:
1. Checks out the tag
2. Runs `cargo semver-checks` to validate no breaking changes
3. Runs `cargo publish` to upload to crates.io

```
google-cloud-bigtable-admin-v2 v4.0.0
  Checked out tag: google-cloud-bigtable-admin-v2/v4.0.0
  Running semver-checks...
  Validation passed
  Publishing to crates.io...
  Published: https://crates.io/crates/google-cloud-bigtable-admin-v2/4.0.0
```

### Validation

Before publishing, the command validates:

- **Go**: Tag exists and is pushed to remote
- **Python**: Distribution builds successfully, version matches tag
- **Rust**: `cargo semver-checks` passes (validates API compatibility)

### Credentials

Publishing requires credentials:

- **Python**: `~/.pypirc` or `TWINE_USERNAME` + `TWINE_PASSWORD` environment variables
- **Rust**: `~/.cargo/credentials` or `CARGO_REGISTRY_TOKEN` environment variable

The command will prompt for credentials if not found.

### Error Handling

**Tag not found**:
```
Error: Tag 'secretmanager/v1.12.0' not found
Run 'librarian release tag' first.
```

**Already published**:
```
Error: Package 'google-cloud-secret-manager' version '1.12.0' already exists on PyPI
Skipping publish.
```

**Validation failed**:
```
Error: cargo semver-checks failed for google-cloud-bigtable-admin-v2

Breaking changes detected in patch release:
  - Removed public function: Client::query()
  - Changed signature: Client::new() now returns Result instead of Self

This is a major release (4.0.0) but contains breaking changes.
Publish aborted.
```

**Missing credentials**:
```
Error: PyPI credentials not found
Configure credentials in ~/.pypirc or set environment variables:
  export TWINE_USERNAME=__token__
  export TWINE_PASSWORD=pypi-...
```

## Workflows

### Standard Release Workflow

Developer makes changes and wants to release:

```bash
# 1. Make changes and commit with conventional commits
git add secretmanager/
git commit -m "feat(secretmanager): add Secret rotation support"
git push

# 2. Prepare the release (updates version files and changelogs)
librarian release prepare
# Creates commit: "chore(release): prepare secretmanager v1.12.0"

# 3. Create PR and get it merged
gh pr create --title "chore(release): prepare secretmanager v1.12.0"
# ... PR gets reviewed and merged ...

# 4. After merge, create tags
librarian release tag
# Creates and pushes: secretmanager/v1.12.0

# 5. Publish to registries
librarian release publish
# Uploads to PyPI/crates.io (Go auto-indexes)
```

### First Release Workflow

For a brand new library that has never been released:

```bash
# 1. Generate the library
librarian generate --library=secretmanager

# 2. Manually create first tag
git tag -a secretmanager/v0.1.0 -m "Initial release of secretmanager"
git push origin secretmanager/v0.1.0

# 3. Now librarian release commands work
# Make changes...
librarian release prepare   # Detects v0.1.0 as baseline
librarian release tag
librarian release publish
```

The first tag must be created manually to prevent accidental releases and ensure conscious versioning decisions.

## Design Principles

### 1. Git as Source of Truth

All release decisions are based on git history:
- Version files in git determine what needs to be released
- Git tags mark what has been released
- Conventional commits determine version bump type
- Git diff shows what changed since last release

**Benefits**:
- No external dependencies (works offline)
- Auditable (full history in git)
- Works with any git hosting (GitHub, GitLab, Bitbucket, self-hosted)

### 2. Language-Agnostic Core

The core release logic is the same for all languages:
- Version detection: compare version files
- Version bump: analyze conventional commits
- Changelog: generate from commit messages
- Tag format: `{library}/v{version}`

Only the language-specific details differ:
- Version file location (version.go, pyproject.toml, Cargo.toml)
- Publishing command (no-op for Go, twine for Python, cargo for Rust)
- Validation tools (go test, nox, cargo semver-checks)

**Benefits**:
- Consistent workflow across languages
- Easier to maintain (less duplication)
- Easier to add new languages

### 3. Idempotent Operations

All commands can be run multiple times safely:
- `librarian release prepare` - Only updates if versions changed
- `librarian release tag` - Skips tags that already exist
- `librarian release publish` - Skips packages already published

**Benefits**:
- Safe to re-run in CI/CD
- Recoverable from partial failures
- No side effects from accidental runs

### 4. Fail-Safe First Release

The first release of a library requires a manual tag:

```bash
git tag -a secretmanager/v0.1.0 -m "Initial release"
git push origin secretmanager/v0.1.0
```

Without a previous tag, `librarian release prepare` fails with:

```
Error: No previous tag found for library 'secretmanager'
This is the first release. Create the initial tag manually:
  git tag -a secretmanager/v0.1.0 -m "Initial release"
  git push origin secretmanager/v0.1.0
```

**Benefits**:
- Prevents accidental first releases
- Forces conscious versioning decision
- Establishes baseline for future releases

### 5. No GitHub Dependency

The release process doesn't require GitHub-specific features:
- No PRs with special labels
- No GitHub Actions required
- No GitHub API calls (except optional `gh release create`)

**Benefits**:
- Works with any git hosting
- Works in air-gapped environments
- Reduces points of failure

## Conventional Commits

Librarian uses conventional commits to determine version bumps:

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

- **feat**: New feature (minor bump)
- **fix**: Bug fix (patch bump)
- **feat!**: Breaking change (major bump)
- **fix!**: Breaking fix (major bump)
- **docs**: Documentation only
- **chore**: Maintenance tasks
- **test**: Adding tests
- **refactor**: Code refactoring

### Breaking Changes

Mark breaking changes with `!` or `BREAKING CHANGE:` in footer:

```
feat(secretmanager)!: remove deprecated Query method

The Query method has been deprecated since v1.5.0.
Use ListSecrets instead.

BREAKING CHANGE: Query method removed
```

### Examples

**Minor release** (new feature):
```
feat(secretmanager): add Secret rotation support

Adds RotateSecret method to SecretManagerClient.
```

**Patch release** (bug fix):
```
fix(secretmanager): handle nil pointers in GetSecret

GetSecret now properly handles nil SecretVersion responses.
```

**Major release** (breaking change):
```
feat(secretmanager)!: change SecretManagerClient.New signature

SecretManagerClient.New now returns (client, error) instead of just client.
This allows proper error handling during client initialization.
```

## Future Enhancements

### Automated Dependency Updates

Track dependency updates and include in changelogs:

```markdown
### Dependencies
- Updated google.golang.org/api to v0.150.0
- Updated google.golang.org/grpc to v1.60.0
```

### Release Notes Templates

Support custom release note templates per library:

```markdown
# secretmanager v1.12.0

## What's New
- Secret rotation support

## Bug Fixes
- Nil pointer handling

## Installation
\`\`\`bash
go get cloud.google.com/go/secretmanager/apiv1@v1.12.0
\`\`\`
```

### Breaking Change Detection

Enhance validation with language-specific API diff tools:
- **Go**: `gorelease` or `apidiff`
- **Python**: `griffe`
- **Rust**: `cargo semver-checks` (already implemented)

Automatically detect breaking changes and enforce major version bumps.

### Multi-Language Library Support

For libraries that span multiple languages (e.g., Protocol Buffers), coordinate releases:

```bash
librarian release prepare --library=secretmanager --all-languages
# Prepares: google-cloud-secret-manager (Python), cloud.google.com/go/secretmanager (Go)
```

---

## Appendix: Current Implementation

This appendix documents the current release implementation for reference.

### Current Go/Python Implementation

Go and Python currently use a **PR + label workflow**:

1. Developer creates a PR with changes
2. PR is labeled `release:pending`
3. CI runs `release-init` (Python) or `release-stage` (Go) which:
   - Reads configuration to find libraries with `release_triggered: true`
   - Updates version files and changelogs
   - Creates a commit on the PR
4. PR is merged
5. CI creates git tags
6. CI publishes to registries

**Configuration Example** (Python):
```yaml
# .librarian.yaml
libraries:
  - name: google-cloud-secret-manager
    path: packages/google-cloud-secret-manager
    release_triggered: true  # Set by automation
    version: 1.12.0
```

**Commands**:
- Python: `python .generator/cli.py release-init`
- Go: `go run internal/librariangen/main.go release-stage`

**Limitations**:
- Requires GitHub (labels, PRs)
- Manual label management
- Configuration must be updated per release
- Easy to miss libraries or make errors

### Current Rust Implementation

Rust currently uses a **git-based workflow**:

1. Developer makes changes and commits
2. Developer runs `sidekick rust-bump-versions --filter-changed` which:
   - Detects changed Cargo.toml files since last tag
   - Bumps versions in Cargo.toml
   - Updates Cargo.lock
3. Developer creates PR and merges
4. Developer runs `sidekick rust-publish` which:
   - Validates with `cargo workspaces plan`
   - Runs `cargo semver-checks`
   - Creates git tags
   - Publishes to crates.io

**Commands**:
```bash
# Bump versions for changed crates
go run ./cmd/sidekick rust-bump-versions --filter-changed

# Publish to crates.io
go run ./cmd/sidekick rust-publish
```

**Advantages**:
- No GitHub dependency
- Git history is source of truth
- Validation with semver-checks
- Explicit and auditable

### Comparison

| Aspect | Go/Python (Current) | Rust (Current) | Proposed |
|--------|-------------------|----------------|----------|
| **GitHub Dependency** | Yes (labels, PRs) | No | No |
| **Configuration** | release_triggered flag | None | None |
| **Version Detection** | Manual flag | Git diff | Git diff |
| **Breaking Changes** | Manual | cargo semver-checks | Language-specific tools |
| **First Release** | Works automatically | Requires manual tag | Requires manual tag |
| **Commands** | release-init, release-stage | rust-bump-versions, rust-publish | release prepare, release tag, release publish |

### Why Implementations Differ

**Historical reasons**: Go and Python implementations predate Rust and were built around GitHub automation workflows.

**Language ecosystem**: Rust has excellent tooling (`cargo workspaces`, `cargo semver-checks`) that makes git-based workflows natural.

**Proposed design**: Unifies all languages with Rust's git-based approach while remaining language-agnostic.
