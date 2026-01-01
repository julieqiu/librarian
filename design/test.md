# End-to-End Testing Plan

## Objective
This document outlines a comprehensive, scalable strategy for end-to-end (E2E) testing of the `librarian` CLI across multiple languages. The goal is to ensure the compiled binary functions correctly in a realistic user workflow, from generating a library to preparing a release.

## High-Level E2E Strategy
The core of our E2E testing will be to simulate real-world usage of the `librarian` binary in an isolated environment. The E2E tests will be structured as a matrix, combining various `librarian` scenarios with each supported language.

### E2E Test Matrix: Scenarios x Languages

The testing matrix will cover different `librarian` functionalities across each language:

1.  **Scenario (Librarian Type):** Represents a complete user workflow, often involving a sequence of `librarian` commands.
    *   **`Generate` Scenario:** Tests the creation of a new client library.
        *   **Commands:** `librarian generate ...`
        *   **Validation:** Verify the generated library's existence, correctness, and language-specific validity (e.g., compilation for Rust, linting for Python).
    *   **`Release` Scenario:** Tests the version updating and release artifact preparation.
        *   **Setup:** This will involve programmatic setup within the test function (e.g., initializing a Git repository, creating dummy generated files, and setting an initial `librarian.yaml` state).
        *   **Commands:** `librarian release ...`
        *   **Validation:** Check for correct `librarian.yaml` updates, proper Git tag creation, and other release-related artifacts.
    *   **`Tidy` Scenario:** Tests the formatting and validation of `librarian.yaml`.
        *   **Setup:** This will involve programmatic setup within the test function to create a `librarian.yaml` with known formatting or validation issues.
        *   **Commands:** `librarian tidy`
        *   **Validation:** Assert that `librarian.yaml` is correctly formatted and validated.
    *   *(Future scenarios like `publish` can be added here.)*

2.  **Language:** The target client library language (e.g., Python, Rust).

This matrix ensures systematic coverage, testing each scenario for every supported language (e.g., `Generate` for Python, `Generate` for Rust, `Release` for Python, etc.).

### E2E Test Fixtures (`internal/testdata`)
E2E test data will be organized under `internal/testdata` to maintain consistency with existing project conventions. Each language subdirectory will contain a baseline `librarian.yaml` and any other common initial files. Scenario-specific modifications will be handled programmatically within the Go tests.

```
internal/
├── testdata/
│   ├── googleapis/ # Existing directory
│   ├── google-cloud-python/
│   │   ├── librarian.yaml # Baseline config for Python E2E tests
│   │   └── ... (other common initial files for Python)
│   ├── google-cloud-rust/
│   │   ├── librarian.yaml # Baseline config for Rust E2E tests
│   │   └── ...
│   └── ... (other languages as they are added)
├── e2e/ # E2E test Go files will reside here
│   └── librarian_e2e_test.go
└── ... (other internal packages)
```

### Go E2E Test Structure (`internal/e2e/librarian_e2e_test.go`)

To support failure isolation, the Go E2E tests will be updated to include an explicit, version-controlled skip mechanism and will be driven by an environment variable to ensure CI jobs only run tests for their designated language.



```go

package e2e



import (

	"path/filepath"

	"testing"

	"os"

	"os/exec" // Required for running compiled librarian and validation commands

	"io/fs"

	"io"

	// ... other imports as needed

)



// define our language-specific configurations for the matrix

var languages = []struct {

	name             string

	testdataDir      string   // Subdirectory under internal/testdata, e.g., "google-cloud-python"

	validationCmd    []string // Command and args to validate generated artifacts

	SkipReason       string   // New field: If non-empty, all tests for this language will be skipped.

}{

	{"python", "google-cloud-python", []string{"python", "-m", "compileall", "."}, ""},

	{"rust", "google-cloud-rust", []string{"cargo", "check", "--all-targets"}, "Skipping until Rust toolchain v1.77 is available in CI."},

	// Add other languages here

}



// define our scenarios for the matrix

var scenarios = []struct {

	name    string

	runFunc func(t *testing.T, lang languageConfig, workDir, librarianPath string)

}{

	{"generate", testGenerateScenario},

	{"release", testReleaseScenario},

	{"tidy", testTidyScenario},

	// Add other scenarios here

}



// languageConfig is a helper struct to pass language details to scenario functions

type languageConfig struct {

	name             string

	testdataDir      string

	validationCmd    []string

	SkipReason       string

}



// TestLibrarianE2E is the main entry point for all end-to-end tests.

func TestLibrarianE2E(t *testing.T) {

	librarianPath := buildLibrarian(t)



	// In CI, run tests for only one language, determined by an environment variable.

	// For local runs, if the variable is not set, run tests for all languages.

	targetLang := os.Getenv("LIBRARIAN_E2E_LANGUAGE")



	for _, lang := range languages {

		if targetLang != "" && lang.name != targetLang {

			continue // Skip languages not targeted in this CI job.

		}



		currentLang := lang // Capture loop variable for t.Parallel

		t.Run(currentLang.name, func(t *testing.T) {

			if currentLang.SkipReason != "" {

				t.Skip(currentLang.SkipReason)

			}

			t.Parallel()



			for _, scenario := range scenarios {

				currentScenario := scenario

				t.Run(currentScenario.name, func(t *testing.T) {

					workDir := t.TempDir()

					copyBaselineFixtures(t, workDir, currentLang.testdataDir)

					currentScenario.runFunc(t, languageConfig(currentLang), workDir, librarianPath)

				})

			}

		})

	}

}



// buildLibrarian compiles the librarian binary once for all E2E tests.

// ... (implementation as before) ...



// copyBaselineFixtures copies test data from internal/testdata/<langTestdataDir>/ to destDir.

// ... (implementation as before) ...



// testGenerateScenario, testReleaseScenario, testTidyScenario

// ... (implementations as before) ...

```

## Dependency Management & CI Environment
Managing dependencies for multiple languages is critical for reproducible E2E tests. The strategy is to create a dedicated, isolated environment for each language using Docker.

This approach builds on the dependency definitions already established in `design/tool.yaml`.

1.  **Language-Specific Docker Images:**
    *   For each supported language, we will create a dedicated `Dockerfile` (e.g., `infra/imagebuilders/e2e-python.Dockerfile`, `infra/imagebuilders/e2e-rust.Dockerfile`).
    *   Each Dockerfile will start from an official base image (e.g., `python:3.11-slim`, `rust:1.76`).
    *   It will then install the *exact* tool versions specified in `design/tool.yaml` using the native package manager (e.g., `pip`, `cargo install`).
2.  **Automated Image Builds:**
    *   A CI workflow (e.g., a GitHub Action) will be created to automatically build these Docker images and push them to a container registry (e.g., GitHub Container Registry).
    *   This build process will be triggered whenever a language's `Dockerfile` or the `design/tool.yaml` file is modified, ensuring test environments are always in sync with their definitions.

## CI Workflow Setup
With versioned Docker images available, we can configure our CI system to run E2E tests in a clean, parallel, and scalable manner. This workflow will align with the Go E2E test runner by using an environment variable to specify which language to test, enabling failure isolation.

A new GitHub Actions workflow will be created at `.github/workflows/e2e.yaml` with the following characteristics:
*   **Matrix Strategy:** The workflow will use a `matrix` strategy to create a separate job for each language (`python`, `rust`, etc.).
*   **Containerized Jobs:** Each job in the matrix will pull its corresponding Docker image (`librarian-e2e-python:latest`) and run the E2E tests inside that container.
*   **Environment Variable:** Each job will set the `LIBRARIAN_E2E_LANGUAGE` environment variable to match the language from the matrix. This tells the Go test runner to only execute tests for that specific language.

This setup provides:
*   **Isolation:** Python tests run in a Python environment, Rust tests in a Rust environment. There is no risk of toolchain conflicts or test pollution.
*   **Parallelism:** All language tests run simultaneously, providing fast feedback.
*   **Scalability:** Adding E2E tests for a new language becomes as simple as adding its `Dockerfile` and adding its name to the matrix in the `e2e.yaml` file.
*   **Resilience:** A failure in one language's job will not affect other languages' jobs.

### Conceptual `e2e.yaml`
```yaml
name: End-to-End Tests

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Build librarian binary
        run: go build -o librarian ./cmd/librarian
      - name: Upload librarian binary
        uses: actions/upload-artifact@v4
        with:
          name: librarian-binary
          path: librarian

  e2e:
    needs: build
    strategy:
      matrix:
        language: [python, rust] # New languages are added here
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Download librarian binary
        uses: actions/download-artifact@v4
        with:
          name: librarian-binary
          path: .

      - name: Make binary executable
        run: chmod +x ./librarian

      - name: Run ${{ matrix.language }} E2E Tests
        # Fetches the pre-built Docker image with all dependencies
        uses: docker://ghcr.io/your-org/librarian-e2e-${{ matrix.language }}:latest
        env:
          # This variable tells the Go runner to ONLY run tests for this language.
          LIBRARIAN_E2E_LANGUAGE: ${{ matrix.language }}
        run: |
          # The librarian binary is already built and downloaded as an artifact.
          # Run the E2E-tagged tests.
          go test -tags=e2e -v ./internal/e2e/...
```
