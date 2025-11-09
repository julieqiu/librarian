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

package librarian

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/gitrepo"
)

const (
	generateCmdName = "generate"
)

type generateRunner struct {
	api               string
	branch            string
	build             bool
	commit            bool
	generateUnchanged bool
	containerClient   ContainerClient
	hostMount         string
	image             string
	library           string
	push              bool
	repo              gitrepo.Repository
	sourceRepo        gitrepo.Repository
	state             *config.LibrarianState
	workRoot          string
}

func newGenerateRunner(cfg *config.Config) (*generateRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, err
	}
	return &generateRunner{
		api:               cfg.API,
		branch:            cfg.Branch,
		build:             cfg.Build,
		commit:            cfg.Commit,
		containerClient:   runner.containerClient,
		generateUnchanged: cfg.GenerateUnchanged,
		hostMount:         cfg.HostMount,
		image:             runner.image,
		library:           cfg.Library,
		push:              cfg.Push,
		repo:              runner.repo,
		sourceRepo:        runner.sourceRepo,
		state:             runner.state,
		workRoot:          runner.workRoot,
	}, nil
}

// run executes the library generation process.
//
// It determines whether to generate a single library or all configured libraries based on the
// command-line flags. If an API or library is specified, it generates a single library. Otherwise,
// it iterates through all libraries defined in the state and generates them.
func (r *generateRunner) run(ctx context.Context) error {
	_ = ctx
	outputDir := filepath.Join(r.workRoot, "output")
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to make output directory, %s: %w", outputDir, err)
	}
	// The last generated commit is changed after library generation,
	var failedLibraries []string
	if r.api != "" || r.library != "" {
		libraryID := r.library
		if libraryID == "" {
			libraryID = findLibraryIDByAPIPath(r.state, r.api)
		}
		_, err := r.generateSingleLibrary(ctx, libraryID, outputDir)
		if err != nil {
			return err
		}
	} else {
		var succeededGenerations int
		var skippedGenerations int
		for _, library := range r.state.Libraries {
			shouldGenerate, err := r.shouldGenerate(library)
			if err != nil {
				slog.Error("failed to determine whether or not to generate library", "id", library.ID, "err", err)
				// While this isn't strictly a failed generation, it's a library for which
				// the generate command failed, so it's close enough.
				failedLibraries = append(failedLibraries, library.ID)
				continue
			}
			if !shouldGenerate {
				// We assume that the cause will have been logged in shouldGenerateLibrary.
				skippedGenerations++
				continue
			}
			_, err := r.generateSingleLibrary(ctx, library.ID, outputDir)
			if err != nil {
				slog.Error("failed to generate library", "id", library.ID, "err", err)
				failedLibraries = append(failedLibraries, library.ID)
			} else {
				succeededGenerations++
			}
		}

		slog.Info(
			"generation statistics",
			"all", len(r.state.Libraries),
			"successes", succeededGenerations,
			"skipped", skippedGenerations,
			"failures", len(failedLibraries))
		if len(failedLibraries) > 0 && len(failedLibraries)+skippedGenerations == len(r.state.Libraries) {
			return fmt.Errorf("all %d libraries failed to generate (skipped: %d)",
				len(failedLibraries), skippedGenerations)
		}
	}

	if err := saveLibrarianState(r.repo.GetDir(), r.state); err != nil {
		return err
	}

	return nil
}

// generateSingleLibrary manages the generation of a single client library.
//
// The single library generation executes as follows:
//
// 1. Configure the library, if the library is not configured in the state.yaml.
//
// 2. Generate the library.
//
// 3. Build the library.
//
// 4. Update the last generated commit or initial piper id if the library needs configure.
func (r *generateRunner) generateSingleLibrary(ctx context.Context, libraryID, outputDir string) error {
	safeLibraryDirectory := getSafeDirectoryName(libraryID)

	// At this point, we should have a library in the state.
	libraryState := r.state.LibraryByID(libraryID)
	if libraryState == nil {
		return fmt.Errorf("library %q not configured yet, generation stopped", libraryID)
	}

	if len(libraryState.APIs) == 0 {
		slog.Info("library has no APIs; skipping generation", "library", libraryID)
		return nil
	}

	if err := generateSingleLibrary(ctx, r.containerClient, r.state, libraryState, r.repo, r.sourceRepo, outputDir); err != nil {
		return err
	}

	if r.build {
		if err := buildSingleLibrary(ctx, r.containerClient, r.state, libraryState, r.repo); err != nil {
			return err
		}
	}

	if err := r.updateLastGeneratedCommitState(libraryID); err != nil {
		return err
	}

	return nil
}

func (r *generateRunner) updateLastGeneratedCommitState(libraryID string) error {
	hash, err := r.sourceRepo.HeadHash()
	if err != nil {
		return err
	}
	for _, l := range r.state.Libraries {
		if l.ID == libraryID {
			l.LastGeneratedCommit = hash
			break
		}
	}
	return nil
}

// getExistingSrc returns source roots as-is of a given library ID, if the source roots exist in the language repo.
func (r *generateRunner) getExistingSrc(libraryID string) []string {
	library := r.state.LibraryByID(libraryID)
	if library == nil {
		return nil
	}

	var existingSrc []string
	for _, src := range library.SourceRoots {
		relPath := filepath.Join(r.repo.GetDir(), src)
		if _, err := os.Stat(relPath); err == nil {
			existingSrc = append(existingSrc, src)
		}
	}

	return existingSrc
}

func setAllAPIStatus(state *config.LibrarianState, status string) {
	for _, library := range state.Libraries {
		for _, api := range library.APIs {
			api.Status = status
		}
	}
}

// shouldGenerate determines whether a library should be generated by the generate
// command. It does *not* observe the -library or -api flag, as those are handled
// higher up in run. If this function returns false (with a nil error), it always
// logs why the library was skipped.
//
// The decision of whether or not a library should be generated is relatively complex,
// and should be kept centrally in this function, with a comment for each path in the flow
// for clarity.
func (r *generateRunner) shouldGenerate(library *config.LibraryState) (bool, error) {
	// If the library has no APIs, it is skipped.
	if len(library.APIs) == 0 {
		slog.Info("library has no APIs, skipping", "id", library.ID)
		return false, nil
	}

	// If we've been asked to generate libraries even with unchanged APIs,
	// we don't need to check whether any have changed: we should definitely generate.
	if r.generateUnchanged {
		return true, nil
	}

	// If we don't know the last commit at which the library was generated,
	// we can't tell whether or not it's changed, so we always generate.
	if library.LastGeneratedCommit == "" {
		return true, nil
	}

	// Most common case: a non-generation-blocked library with APIs, and without the
	// -generate-unchanged flag. Check each API to see whether anything under API.Path
	// has changed between the last_generated_commit and the HEAD commit of r.sourceRepo.
	// If any API has changed, the library is generated - otherwise it's skipped.
	headHash, err := r.sourceRepo.HeadHash()
	if err != nil {
		return false, fmt.Errorf("failed to get head hash for source repo: %v", err)
	}
	for _, api := range library.APIs {
		oldHash, err := r.sourceRepo.GetHashForPath(library.LastGeneratedCommit, api.Path)
		if err != nil {
			return false, fmt.Errorf("failed to get hash for path %v at commit %v: %v", api.Path, library.LastGeneratedCommit, err)
		}
		newHash, err := r.sourceRepo.GetHashForPath(headHash, api.Path)
		if err != nil {
			return false, fmt.Errorf("failed to get hash for path %v at commit %v: %v", api.Path, headHash, err)
		}
		if oldHash != newHash {
			return true, nil
		}
	}
	slog.Info("no APIs have changed; skipping", "library", library.ID)
	return false, nil
}

// addAPIToLibrary adds a new API to a library in the state.
// If the library does not exist, it creates a new one.
// If the API already exists in the library, do nothing.
func addAPIToLibrary(state *config.LibrarianState, libraryID, apiPath string) {
	lib := state.LibraryByID(libraryID)
	if lib == nil {
		// If the library is not found, create a new one.
		state.Libraries = append(state.Libraries, &config.LibraryState{
			ID:   libraryID,
			APIs: []*config.API{{Path: apiPath, Status: config.StatusNew}},
		})
		return
	}

	// If the library is found, check if the API already exists.
	for _, api := range lib.APIs {
		if api.Path == apiPath {
			return
		}
	}

	// For new API paths, set the status to "new".
	lib.APIs = append(lib.APIs, &config.API{Path: apiPath, Status: config.StatusNew})
}
