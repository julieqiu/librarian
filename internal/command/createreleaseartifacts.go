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

package command

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/utils"
)

type LibraryRelease struct {
	LibraryID    string
	ReleaseID    string
	Version      string
	CommitHash   string
	ReleaseNotes string
}

var CmdCreateReleaseArtifacts = &Command{
	Name:  "create-release-artifacts",
	Short: "Create release artifacts from a merged release PR.",
	flagFunctions: []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagLanguage,
		addFlagRepoRoot,
		addFlagRepoUrl,
		addFlagReleaseID,
		addFlagSecretsProject,
		addFlagSkipIntegrationTests,
	},
	maybeGetLanguageRepo:    cloneOrOpenLanguageRepo,
	maybeLoadStateAndConfig: loadRepoStateAndConfig,
	execute:                 createReleaseArtifactsImpl,
}

func createReleaseArtifactsImpl(ctx *CommandContext) error {
	if err := validateSkipIntegrationTests(); err != nil {
		return err
	}
	if err := validateRequiredFlag("release-id", flagReleaseID); err != nil {
		return err
	}

	outputRoot := filepath.Join(ctx.workRoot, "output")
	if err := os.Mkdir(outputRoot, 0755); err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Packages will be created in %s", outputRoot))

	releases, err := parseCommitsForReleases(ctx.languageRepo, flagReleaseID)
	if err != nil {
		slog.Info(fmt.Sprintf("received error parsing commits for release %s", err.Error()))
		return err
	}

	slog.Info(fmt.Sprintf("Got here!"))

	for _, release := range releases {
		if err := buildTestPackageRelease(ctx, outputRoot, release); err != nil {
			slog.Info(fmt.Sprintf("received error building test package %s", err.Error()))
			return err
		}
	}

	if err := copyMetadataFiles(ctx, outputRoot, releases); err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Release artifact creation complete. Artifact root: %s", outputRoot))
	return nil
}

// The publish-release-artifacts stage will need bits of metadata:
// - The releases we're creating
// - The pipeline config
// - (Just in case) The pipeline state
// The pipeline config and state files are copied by checking out the commit of the last
// release, which should effectively be the tip of the release PR.
func copyMetadataFiles(ctx *CommandContext, outputRoot string, releases []LibraryRelease) error {
	releasesJson, err := json.Marshal(releases)
	if err != nil {
		return err
	}
	if err := utils.CreateAndWriteBytesToFile(filepath.Join(outputRoot, "releases.json"), releasesJson); err != nil {
		return err
	}

	languageRepo := ctx.languageRepo
	finalRelease := releases[len(releases)-1]
	if err := gitrepo.Checkout(languageRepo, finalRelease.CommitHash); err != nil {
		return err
	}
	sourceStateFile := filepath.Join(languageRepo.Dir, "generator-input", pipelineStateFile)
	destStateFile := filepath.Join(outputRoot, pipelineStateFile)
	if err := utils.CopyFile(sourceStateFile, destStateFile); err != nil {
		return err
	}

	sourceConfigFile := filepath.Join(languageRepo.Dir, "generator-input", pipelineConfigFile)
	destConfigFile := filepath.Join(outputRoot, pipelineConfigFile)
	if err := utils.CopyFile(sourceConfigFile, destConfigFile); err != nil {
		return err
	}
	return nil
}

func buildTestPackageRelease(ctx *CommandContext, outputRoot string, release LibraryRelease) error {
	containerConfig := ctx.containerConfig
	languageRepo := ctx.languageRepo

	if err := gitrepo.Checkout(languageRepo, release.CommitHash); err != nil {
		slog.Info(fmt.Sprintf("received error checking out commit %s: %s", release.CommitHash, err.Error()))
		return err
	}
	if err := container.BuildLibrary(containerConfig, languageRepo.Dir, release.LibraryID); err != nil {
		return err
	}
	if flagSkipIntegrationTests != "" {
		slog.Info(fmt.Sprintf("Skipping integration tests: %s", flagSkipIntegrationTests))
	} else if err := container.IntegrationTestLibrary(containerConfig, languageRepo.Dir, release.LibraryID); err != nil {
		return err
	}
	outputDir := filepath.Join(outputRoot, release.LibraryID)
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return err
	}
	if err := container.PackageLibrary(containerConfig, languageRepo.Dir, release.LibraryID, outputDir); err != nil {
		return err
	}
	return nil
}

func parseCommitsForReleases(repo *gitrepo.Repo, releaseID string) ([]LibraryRelease, error) {
	commits, err := gitrepo.GetCommitsForReleaseID(repo, releaseID)
	if err != nil {
		slog.Info("error 1 ")
		return nil, err
	}
	releases := []LibraryRelease{}
	for _, commit := range commits {
		release, err := parseCommitMessageForRelease(commit.Message, commit.Hash.String())
		slog.Info("error 1 ")
		if err != nil {
			return nil, err
		}
		releases = append(releases, *release)
	}
	return releases, nil
}

func parseCommitMessageForRelease(message, hash string) (*LibraryRelease, error) {
	messageLines := strings.Split(message, "\n")

	// Remove the expected "title and blank line" (as we'll have a release title).
	// We're fairly conservative about this - if the commit message has been manually
	// changed, we'll leave it as it is.
	if len(messageLines) > 0 && strings.HasPrefix(messageLines[0], "chore: Release library ") {
		messageLines = messageLines[1:]
		if len(messageLines) > 0 && messageLines[0] == "" {
			messageLines = messageLines[1:]
		}
	}

	libraryID, err := findMetadataValue("Librarian-Release-Library", messageLines, hash)
	if err != nil {
		return nil, err
	}
	version, err := findMetadataValue("Librarian-Release-Version", messageLines, hash)
	if err != nil {
		return nil, err
	}
	releaseID, err := findMetadataValue("Librarian-Release-ID", messageLines, hash)
	if err != nil {
		return nil, err
	}
	releaseNotesLines := []string{}
	for _, line := range messageLines {
		if !strings.HasPrefix(line, "Librarian-Release") {
			releaseNotesLines = append(releaseNotesLines, line)
		}
	}
	releaseNotes := strings.Join(releaseNotesLines, "\n")
	return &LibraryRelease{
		LibraryID:    libraryID,
		Version:      version,
		ReleaseNotes: releaseNotes,
		CommitHash:   hash,
		ReleaseID:    releaseID,
	}, nil
}

func findMetadataValue(key string, lines []string, hash string) (string, error) {
	prefix := key + ": "
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return line[len(prefix):], nil
		}
	}
	return "", fmt.Errorf("unable to find metadata value for key '%s' in commit %s", key, hash)
}
