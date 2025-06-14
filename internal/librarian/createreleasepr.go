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
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/githubrepo"

	"github.com/Masterminds/semver/v3"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const prNumberEnvVarName = "_PR_NUMBER"
const baselineCommitEnvVarName = "_BASELINE_COMMIT"

var CmdCreateReleasePR = &cli.Command{
	Name:  "create-release-pr",
	Short: "Generate a release PR.",
	Usage: "TODO(https://github.com/googleapis/librarian/issues/237): add documentation",
	Long:  "TODO(https://github.com/googleapis/librarian/issues/237): add documentation",
	Run:   runCreateReleasePR,
}

func init() {
	CmdCreateReleasePR.SetFlags([]func(fs *flag.FlagSet){
		addFlagImage,
		addFlagSecretsProject,
		addFlagWorkRoot,
		addFlagLanguage,
		addFlagLibraryID,
		addFlagLibraryVersion,
		addFlagPush,
		addFlagGitUserEmail,
		addFlagGitUserName,
		addFlagRepoRoot,
		addFlagSkipIntegrationTests,
		addFlagEnvFile,
		addFlagRepoUrl,
	})
}

func runCreateReleasePR(ctx context.Context) error {
	state, err := createCommandStateForLanguage(ctx)
	if err != nil {
		return err
	}
	return createReleasePR(state)
}

func createReleasePR(state *commandState) error {
	if err := validateSkipIntegrationTests(); err != nil {
		return err
	}
	if err := validatePush(); err != nil {
		return err
	}

	if flagLibraryVersion != "" && flagLibraryID == "" {
		return fmt.Errorf("flag -library-version is not valid without -library-id")
	}

	if flagLibraryID != "" && findLibraryByID(state.pipelineState, flagLibraryID) == nil {
		return fmt.Errorf("no such library: %s", flagLibraryID)
	}

	inputDirectory := filepath.Join(state.workRoot, "inputs")
	if err := os.Mkdir(inputDirectory, 0755); err != nil {
		slog.Error("Failed to create input directory")
		return err
	}

	// Find the head of the language repo before we start creating any release commits.
	// This will be validated later to check that libraries haven't changed since the release PR was created.
	baselineCommit, err := gitrepo.HeadHash(state.languageRepo)
	if err != nil {
		return err
	}
	if err := appendResultEnvironmentVariable(state.workRoot, baselineCommitEnvVarName, baselineCommit); err != nil {
		return err
	}

	releaseID := fmt.Sprintf("release-%s", formatTimestamp(state.startTime))
	if err := appendResultEnvironmentVariable(state.workRoot, releaseIDEnvVarName, releaseID); err != nil {
		return err
	}

	prContent, err := generateReleaseCommitForEachLibrary(state, inputDirectory, releaseID)
	if err != nil {
		return err
	}

	prMetadata, err := createPullRequest(state, prContent, "chore: Library release", fmt.Sprintf("Librarian-Release-ID: %s", releaseID), "release")
	if err != nil {
		return err
	}

	if prMetadata == nil {
		// We haven't created a release PR, and there are no errors. This could be because:
		// - There are no changes to release
		// - The -push flag wasn't specified.
		// Either way, complete successfully at this point.
		return nil
	}

	// Final steps if we've actually created a release PR.
	// - We always add the do-not-merge label so that Librarian can merge later.
	// - Add a result environment variable with the PR number, for the next stage of the process.
	err = githubrepo.AddLabelToPullRequest(state.ctx, *prMetadata, DoNotMergeLabel)
	if err != nil {
		slog.Warn(fmt.Sprintf("Received error trying to add label to PR: '%s'", err))
		return err
	}
	if err := appendResultEnvironmentVariable(state.workRoot, prNumberEnvVarName, strconv.Itoa(prMetadata.Number)); err != nil {
		return err
	}
	return nil
}

// Iterate over all configured libraries, and check for new commits since the previous release tag for that library.
// The error handling here takes one of two forms:
//   - Library-level errors do not halt the process, but are reported in the resulting PR (if any).
//     This can include tags being missing, release preparation failing, or the build failing.
//   - More fundamental errors (e.g. a failure to commit, or to save pipeline state) abort the whole process immediately.
func generateReleaseCommitForEachLibrary(state *commandState, inputDirectory string, releaseID string) (*PullRequestContent, error) {
	containerConfig := state.containerConfig
	libraries := state.pipelineState.Libraries
	languageRepo := state.languageRepo

	pr := new(PullRequestContent)

	for _, library := range libraries {
		// If we've specified a single library to release, skip all the others.
		if flagLibraryID != "" && library.Id != flagLibraryID {
			continue
		}
		if library.ReleaseAutomationLevel == statepb.AutomationLevel_AUTOMATION_LEVEL_BLOCKED {
			slog.Info(fmt.Sprintf("Skipping release-blocked library: '%s'", library.Id))
			continue
		}
		var commitMessages []*CommitMessage
		var previousReleaseTag string
		if library.CurrentVersion == "" {
			previousReleaseTag = ""
		} else {
			previousReleaseTag = formatReleaseTag(library.Id, library.CurrentVersion)
		}
		allSourcePaths := append(state.pipelineState.CommonLibrarySourcePaths, library.SourcePaths...)
		commits, err := gitrepo.GetCommitsForPathsSinceTag(languageRepo, allSourcePaths, previousReleaseTag)
		if err != nil {
			addErrorToPullRequest(pr, library.Id, err, "retrieving commits since last release")
			continue
		}

		for _, commit := range commits {
			commitMessages = append(commitMessages, ParseCommit(commit))
		}

		// If nothing release-worthy has happened, just continue to the next library.
		// (But if we've been asked to release a specific library, we force-release it anyway.)
		if flagLibraryID == "" && (len(commitMessages) == 0 || !isReleaseWorthy(commitMessages, library.Id)) {
			continue
		}

		releaseVersion, err := calculateNextVersion(library)
		if err != nil {
			return nil, err
		}

		releaseNotes := formatReleaseNotes(commitMessages)
		if err = createReleaseNotesFile(inputDirectory, library.Id, releaseVersion, releaseNotes); err != nil {
			return nil, err
		}

		// Update the pipeline state to record what we're releasing and when, and to clear the next version field.
		// Performing this before anything else means that container code can use the pipeline state for the steps
		// below, if it doesn't want/need to store the version separately.
		library.CurrentVersion = releaseVersion
		library.NextVersion = ""
		library.LastReleasedCommit = library.LastGeneratedCommit
		library.ReleaseTimestamp = timestamppb.Now()
		if err = savePipelineState(state); err != nil {
			return nil, err
		}

		if err := container.PrepareLibraryRelease(containerConfig, languageRepo.Dir, inputDirectory, library.Id, releaseVersion); err != nil {
			addErrorToPullRequest(pr, library.Id, err, "preparing library release")
			// Clean up any changes before starting the next iteration.
			if err := gitrepo.CleanWorkingTree(languageRepo); err != nil {
				return nil, err
			}
			continue
		}
		if err := container.BuildLibrary(containerConfig, languageRepo.Dir, library.Id); err != nil {
			addErrorToPullRequest(pr, library.Id, err, "building/testing library")
			// Clean up any changes before starting the next iteration.
			if err := gitrepo.CleanWorkingTree(languageRepo); err != nil {
				return nil, err
			}
			continue
		}
		if flagSkipIntegrationTests != "" {
			slog.Info(fmt.Sprintf("Skipping integration tests: %s", flagSkipIntegrationTests))
		} else if err := container.IntegrationTestLibrary(containerConfig, languageRepo.Dir, library.Id); err != nil {
			addErrorToPullRequest(pr, library.Id, err, "integration testing library")
			if err := gitrepo.CleanWorkingTree(languageRepo); err != nil {
				return nil, err
			}
			continue
		}

		releaseDescription := fmt.Sprintf("chore: Release library %s version %s", library.Id, releaseVersion)
		addSuccessToPullRequest(pr, releaseDescription)
		// Metadata for easy extraction later.
		metadata := fmt.Sprintf("Librarian-Release-Library: %s\nLibrarian-Release-Version: %s\nLibrarian-Release-ID: %s", library.Id, releaseVersion, releaseID)
		// Note that releaseDescription will already end with two line breaks, so we don't need any more before the metadata.
		err = commitAll(languageRepo, fmt.Sprintf("%s\n\n%s%s", releaseDescription, releaseNotes, metadata))
		if err != nil {
			return nil, err
		}
	}
	return pr, nil
}

func formatReleaseNotes(commitMessages []*CommitMessage) string {
	features := []string{}
	docs := []string{}
	fixes := []string{}

	// TODO: Deduping (same message across multiple commits)
	// TODO: Breaking changes
	// TODO: Use the source links etc
	for _, commitMessage := range commitMessages {
		features = append(features, commitMessage.Features...)
		docs = append(docs, commitMessage.Docs...)
		fixes = append(fixes, commitMessage.Fixes...)
	}

	var builder strings.Builder

	maybeAppendReleaseNotesSection(&builder, "New features", features)
	maybeAppendReleaseNotesSection(&builder, "Bug fixes", fixes)
	maybeAppendReleaseNotesSection(&builder, "Documentation improvements", docs)

	if builder.Len() == 0 {
		builder.WriteString("FIXME: Forced release with no commit messages; please write release notes.\n\n")
	}
	return builder.String()
}

func createReleaseNotesFile(inputDirectory, libraryId, releaseVersion, releaseNotes string) error {
	path := filepath.Join(inputDirectory, fmt.Sprintf("%s-%s-release-notes.txt", libraryId, releaseVersion))
	return createAndWriteToFile(path, releaseNotes)
}

func maybeAppendReleaseNotesSection(builder *strings.Builder, description string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(builder, "### %s\n\n", description)
	for _, line := range lines {
		if len(line) > 1 {
			// This assumes the first character is ASCII, but that's reasonable for all our
			// actual use cases.
			line = strings.ToUpper(line[0:1]) + line[1:]
		}
		fmt.Fprintf(builder, "- %s\n", line)
	}
	builder.WriteString("\n")
}

func calculateNextVersion(library *statepb.LibraryState) (string, error) {
	if flagLibraryVersion != "" {
		return flagLibraryVersion, nil
	}
	if library.NextVersion != "" {
		return library.NextVersion, nil
	}
	if library.CurrentVersion == "" {
		return "", fmt.Errorf("cannot determine new version for %s; no current version", library.Id)
	}
	current, err := semver.StrictNewVersion(library.CurrentVersion)
	if err != nil {
		return "", err
	}
	var next *semver.Version
	prerelease := current.Prerelease()
	if prerelease != "" {
		nextPrerelease, err := calculateNextPrerelease(prerelease)
		if err != nil {
			return "", err
		}
		next = semver.New(current.Major(), current.Minor(), current.Patch(), nextPrerelease, "")
	} else {
		next = semver.New(current.Major(), current.Minor()+1, current.Patch(), "", "")
	}
	return next.String(), nil
}

// Match trailing digits in the prerelease part, then parse those digits as an integer.
// Increment the integer, then format it again - keeping as much of the existing prerelease as is
// required to end up with a string longer-than-or-equal to the original.
// If there are no trailing digits, fail.
// Note: this assumes the prerelease is purely ASCII.
func calculateNextPrerelease(prerelease string) (string, error) {
	digits := 0
	for i := len(prerelease) - 1; i >= 0; i-- {
		c := prerelease[i]
		if c < '0' || c > '9' {
			break
		} else {
			digits++
		}
	}
	if digits == 0 {
		return "", fmt.Errorf("unable to create next prerelease from '%s'", prerelease)
	}
	currentSuffix := prerelease[len(prerelease)-digits:]
	currentNumber, err := strconv.Atoi(currentSuffix)
	if err != nil {
		return "", err
	}
	nextSuffix := strconv.Itoa(currentNumber + 1)
	if len(nextSuffix) < len(currentSuffix) {
		nextSuffix = strings.Repeat("0", len(currentSuffix)-len(nextSuffix)) + nextSuffix
	}
	return prerelease[:(len(prerelease)-digits)] + nextSuffix, nil
}

func isReleaseWorthy(messages []*CommitMessage, libraryId string) bool {
	for _, message := range messages {
		// TODO: Work out why we can't call message.IsReleaseWorthy(libraryId)
		if IsReleaseWorthy(message, libraryId) {
			return true
		}
	}
	return false
}
