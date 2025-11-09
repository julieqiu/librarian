# Sidekick Merge Strategy

This document outlines the strategy for merging sidekick functionality into librarian.

## Current State

### Librarian Commands

- `librarian init [language]` - Initialize repository for library management
- `librarian add <path> [api...]` - Track a directory for management
- `librarian edit <path>` - Edit artifact configuration
- `librarian remove <path>` - Remove artifact from tracking
- `librarian generate <path>` - Generate/regenerate code
- `librarian prepare <path>` - Prepare release (not yet implemented)
- `librarian release <path>` - Tag and release (not yet implemented)
- `librarian config get <key>` - Get configuration value
- `librarian config set <key> <value>` - Set configuration value
- `librarian config update [--all]` - Update configuration to latest
- `librarian version` - Print version information

### Sidekick Commands

- `sidekick generate` - First-time generation for a client library
- `sidekick refresh` - Regenerate single client library
- `sidekick refresh-all` - Regenerate all client libraries
- `sidekick update` - Update .sidekick.toml with latest googleapis and refresh all
- `sidekick rust-generate` - First-time generation for Rust monorepo
- `sidekick rust-bump-versions` - Increment version numbers for changed crates
- `sidekick rust-publish` - Publish crates changed since last release

## End State: Unified Command Set

All sidekick functionality merged into librarian's existing commands:

| Sidekick Command | Librarian Equivalent | Notes |
|------------------|---------------------|-------|
| sidekick generate | `librarian generate` | Merge functionality into librarian generate |
| sidekick refresh | `librarian generate` | Same command, different name |
| sidekick refresh-all | `librarian generate --all` | Same command, different name |
| sidekick update | `librarian config update --all` | Already implemented in librarian |
| sidekick rust-generate | `librarian generate` | Container handles Rust-specific logic |
| sidekick rust-bump-versions | `librarian prepare` | **Direct 1:1 mapping** |
| sidekick rust-publish | `librarian release` | Integrate into release command |

**Result**: Only the core librarian commands remain. No `sidekick` subcommands needed.

## Key Insights

### 1. `rust-bump-versions` IS `prepare`

The `sidekick rust-bump-versions` command is **exactly** what `librarian prepare` is designed to do:

**What `rust-bump-versions` does**:
1. Find last release tag
2. Determine which files changed since last release
3. Find all affected package manifests
4. Bump versions for changed packages
5. Update manifest files
6. Run validation (e.g., `cargo semver-checks` for Rust)

**What `librarian prepare` should do**:
> Determines the next version, updates metadata, and prepares release notes.
> Does not tag or publish.

**They are the same operation!**

The only difference is language-specific implementation:
- **Rust**: Uses `Cargo.toml`, runs `cargo semver-checks`
- **Python**: Uses `pyproject.toml` / `setup.py`, runs Python validators
- **Go**: Uses `go.mod`, runs Go validators
- **Dart**: Uses `pubspec.yaml`, runs Dart validators

**Implementation**: `librarian prepare` dispatches to language-specific prepare logic from `internal/sidekick/rust_release/`.

### 2. `rust-generate` IS Just `generate` with Container Intelligence

The `sidekick rust-generate` command can be eliminated by making the Rust generator container smarter.

**What `rust-generate` does**:
1. Auto-derive paths from service config (e.g., `google/cloud/storage/v1` → `src/generated/cloud/storage/v1`)
2. Validate required tools (cargo, taplo, typos, git)
3. Scaffold package with `cargo new`
4. Generate code via regular `generate()`
5. Run full validation suite (fmt, test, doc, clippy, typos)
6. Stage files in git

**Solution**: Move all Rust-specific logic into the Rust generator container.

The container receives:
```json
{
  "id": "cloud-storage-v1",
  "apis": [{"path": "google/cloud/storage/v1"}]
}
```

The container:
1. Derives output path using Rust monorepo conventions
2. Checks for required tools at startup
3. Scaffolds package structure
4. Generates code
5. Runs full validation suite
6. Returns success/failure

**User workflow becomes**:
```bash
# Instead of:
sidekick rust-generate --service-config=google/cloud/storage/v1/storage.yaml

# Simply:
librarian add google/cloud/storage/v1
librarian generate src/generated/cloud/storage/v1
```

The only configuration needed:
```yaml
# .librarian.yaml
librarian:
    language: rust
generate:
    container:
        image: us-central1-docker.pkg.dev/.../rust-librarian-generator
        tag: latest
```

Everything else (path derivation, scaffolding, validation) is handled by the intelligent Rust container.

### 3. `rust-publish` Becomes Part of `release`

The `sidekick rust-publish` command publishes crates to crates.io. This is exactly what `librarian release` should do for Rust.

**Implementation**: `librarian release` dispatches to language-specific release logic from `internal/sidekick/rust_release/`.

## Unified Workflow

### Before (Sidekick)

```bash
# Rust workflow
sidekick rust-generate --service-config=google/cloud/storage/v1/storage.yaml
# ... make changes ...
sidekick rust-bump-versions
sidekick rust-publish

# Python workflow
sidekick generate --language=python --specification-source=... --output=...
# ... make changes ...
# (no prepare/release commands)
```

### After (Librarian)

```bash
# Rust workflow (same as all other languages!)
librarian add google/cloud/storage/v1
librarian generate src/generated/cloud/storage/v1
# ... make changes ...
librarian prepare src/generated/cloud/storage/v1
librarian release src/generated/cloud/storage/v1

# Python workflow (identical commands!)
librarian add google/cloud/storage/v1
librarian generate packages/google-cloud-storage
# ... make changes ...
librarian prepare packages/google-cloud-storage
librarian release packages/google-cloud-storage
```

**All languages use the same commands. Language-specific logic lives in containers and language-specific modules.**

## Code Reuse

Sidekick contains valuable, battle-tested code that will be reused:

### `internal/sidekick/rust_release/` → `internal/librarian/rust/`

Reuse directly for Rust-specific prepare/release logic:
- `bump_versions.go` - Version bumping logic (used by `librarian prepare`)
- `publish.go` - Publishing to crates.io (used by `librarian release`)
- `changes.go` - Detecting changed files
- `preflight.go` - Pre-release validation

### `internal/sidekick/parser/` → Move to Containers

The sophisticated API parsing logic should move into language-specific generator containers:
- Protobuf parsing
- OpenAPI parsing
- Service config parsing

This keeps librarian simple and language-agnostic while preserving parsing capabilities.

### `internal/sidekick/language/` → Move to Containers

Code generation templates and logic should move into language-specific generator containers.

## Configuration Migration

Both `.librarian.yaml` and `.sidekick.toml` will be supported during transition:

### `.sidekick.toml` (Legacy)
```toml
[root]
googleapis-root = "https://github.com/googleapis/googleapis"
googleapis-sha256 = "abc123..."

[release]
branch = "main"
remote = "origin"
```

### `.librarian.yaml` (Unified)
```yaml
librarian:
    language: rust
generate:
    container:
        image: us-central1-docker.pkg.dev/.../rust-librarian-generator
        tag: latest
    googleapis:
        path: https://github.com/googleapis/googleapis/archive/abc123.tar.gz
        sha256: abc123...
release:
    tag_format: '{name}-v{version}'
```

Migration: `librarian config migrate` converts `.sidekick.toml` to `.librarian.yaml`.

## Benefits of Unification

1. **Consistent commands across all languages** - No need to learn different commands for Rust vs Python vs Go
2. **Simpler mental model** - One tool, one config format, one workflow
3. **Language-specific logic in containers** - Easy to update without changing librarian
4. **Reuse battle-tested code** - Keep proven Rust release logic from sidekick
5. **No special-casing** - Rust is just another language, not a special case requiring separate commands
6. **Better discoverability** - All commands in `librarian --help`, no hidden `sidekick` binary

## Summary

**No sidekick commands remain.** Everything merges into librarian's existing command structure:

- `sidekick generate/refresh` → `librarian generate` (already exists)
- `sidekick update` → `librarian config update` (already exists)
- `sidekick rust-generate` → `librarian generate` (container handles Rust logic)
- `sidekick rust-bump-versions` → `librarian prepare` (implement with existing code)
- `sidekick rust-publish` → `librarian release` (implement with existing code)

The sidekick binary and subcommands are deprecated. All functionality lives in librarian.
