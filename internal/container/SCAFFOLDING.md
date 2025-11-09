# Scaffolding Integration Guide

This document describes how `librarian generate` should call the scaffolding generation functions before invoking containers.

## Overview

Each language package (`internal/container/{language}`) provides a `GenerateScaffolding` function that creates initial scaffolding files on first-time generation only.

## Integration in `librarian generate`

The `librarian generate` command should:

1. Read `.librarian.yaml` to get library configuration
2. **Call `GenerateScaffolding` for the language** (before container)
3. Call the generate container
4. Commit scaffolding + generated code

## Example Integration

```go
package main

import (
	"fmt"

	golang "github.com/googleapis/librarian/internal/container/go"
	"github.com/googleapis/librarian/internal/container/python"
	"github.com/googleapis/librarian/internal/container/rust"
)

func Generate(cfg *Config, libraryName string) error {
	// 1. Read library config from .librarian.yaml
	library, err := readLibraryConfig(cfg.RepoRoot, libraryName)
	if err != nil {
		return err
	}

	// 2. Generate scaffolding (first-time only)
	switch cfg.Language {
	case "go":
		// Extract APIs from library config
		apis := make([]golang.API, len(library.Generate.APIs))
		for i, api := range library.Generate.APIs {
			apis[i] = golang.API{
				Path:          api.Path,
				ServiceConfig: api.ServiceConfig,
			}
		}

		if err := golang.GenerateScaffolding(
			cfg.RepoRoot,
			cfg.GoogleAPIsRoot,
			library.Name,
			apis,
		); err != nil {
			return fmt.Errorf("failed to generate Go scaffolding: %w", err)
		}

	case "python":
		if err := python.GenerateScaffolding(
			cfg.RepoRoot,
			library.Name,
		); err != nil {
			return fmt.Errorf("failed to generate Python scaffolding: %w", err)
		}

	case "rust":
		// No-op for Rust - sidekick generate handles everything
		if err := rust.GenerateScaffolding(
			cfg.RepoRoot,
			library.Name,
		); err != nil {
			return fmt.Errorf("failed to generate Rust scaffolding: %w", err)
		}
	}

	// 3. Call generate container
	if err := callGenerateContainer(cfg, library); err != nil {
		return fmt.Errorf("container generation failed: %w", err)
	}

	// 4. Commit changes
	if err := commitGenerate(cfg.RepoRoot, library); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}
```

## Function Signatures

### Go
```go
func GenerateScaffolding(
	repoRoot string,        // Path to language repository
	googleapisRoot string,  // Path to googleapis repository
	libraryID string,       // Library name (e.g., "secretmanager")
	apis []API,             // List of APIs to generate
) error
```

**Creates (first time only):**
- `{libraryID}/README.md`
- `{libraryID}/CHANGES.md`
- `{libraryID}/internal/version.go`
- `{libraryID}/{clientDir}/version.go` (for each API)
- Updates `internal/generated/snippets/go.mod`

### Python
```go
func GenerateScaffolding(
	repoRoot string,   // Path to language repository
	libraryID string,  // Library name (e.g., "google-cloud-storage")
) error
```

**Creates (first time only):**
- `packages/{libraryID}/CHANGELOG.md`
- `packages/{libraryID}/docs/CHANGELOG.md`
- Updates `CHANGELOG.md` (global)

### Rust
```go
func GenerateScaffolding(
	repoRoot string,   // Path to language repository
	libraryID string,  // Library name (e.g., "cloud-storage-v1")
) error
```

**Creates:** Nothing (no-op). Sidekick generate creates all files.

## First-Time Detection

Each `GenerateScaffolding` function automatically detects if this is first-time generation by checking if the library directory exists:

**Go:**
```go
libraryPath := filepath.Join(repoRoot, libraryID)
if pathExists(libraryPath) {
	return nil // Skip scaffolding
}
```

**Python:**
```go
libraryPath := filepath.Join(repoRoot, "packages", libraryID)
if pathExists(libraryPath) {
	return nil // Skip scaffolding
}
```

**Rust:**
```go
return nil // Always skip - sidekick handles it
```

## Error Handling

All functions return `error` and should fail fast if:
- Service YAML cannot be read (Go)
- Global CHANGELOG.md cannot be found (Python)
- Directory creation fails
- File writing fails

## Testing

Test scaffolding generation with:

```bash
# Create test directory
mkdir -p /tmp/test-repo

# Call scaffolding
golang.GenerateScaffolding(
	"/tmp/test-repo",
	"/path/to/googleapis",
	"secretmanager",
	[]golang.API{{
		Path: "google/cloud/secretmanager/v1",
		ServiceConfig: "secretmanager_v1.yaml",
	}},
)

# Verify files created
ls /tmp/test-repo/secretmanager/
# Should show: README.md, CHANGES.md, internal/, apiv1/
```

## Benefits

1. **No container changes needed** - Scaffolding lives in librarian, easy to refactor
2. **Language-specific** - Each language's logic is isolated
3. **First-time detection** - Automatically skips scaffolding on regeneration
4. **Testable** - Pure Go functions, no Docker required
5. **Fast** - No container startup overhead for scaffolding
