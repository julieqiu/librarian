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
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/googleapis/librarian/internal/githubrepo"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

// The environment variable expected to hold an auth token which can be used
// when communicating with the repo which syncs with the language repo,
// as specified via flagSyncUrlPrefix.
const syncAuthTokenEnvironmentVariable string = "LIBRARIAN_SYNC_AUTH_TOKEN"

// The label used to avoid users merging the PR themselves.
const DoNotMergeLabel = "do-not-merge"
const DoNotMergeAppId = 91138
const ConventionalCommitsAppId = 37172

// The label used to indicate "I've noticed a problem with this PR; I won't check it again
// until you've done something".
const MergeBlockedLabel = "merge-blocked-see-comments"

var CmdMergeReleasePR = &Command{
	Name:  "merge-release-pr",
	Short: "Merge a validated release PR.",
	flagFunctions: []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagSecretsProject,
		addFlagWorkRoot,
		addFlagBaselineCommit,
		addFlagReleaseID,
		addFlagReleasePRUrl,
		addFlagSyncUrlPrefix,
	},
	maybeGetLanguageRepo: func(workRoot string) (*gitrepo.Repo, error) {
		return nil, nil
	},
	maybeLoadStateAndConfig: func(languageRepo *gitrepo.Repo) (*statepb.PipelineState, *statepb.PipelineConfig, error) {
		return nil, nil, nil
	},
	execute: mergeReleasePRImpl,
}

type SuspectRelease struct {
	LibraryID string
	Reason    string
}

const mergedReleaseCommitEnvVarName = "_MERGED_RELEASE_COMMIT"

func mergeReleasePRImpl(ctx *CommandContext) error {
	if flagSyncUrlPrefix != "" && os.Getenv(syncAuthTokenEnvironmentVariable) == "" {
		return errors.New("-sync-url-prefix specified, but no sync auth token present")
	}
	if githubrepo.GetAccessToken() == "" {
		return errors.New("no GitHub access token specified")
	}
	// We'll assume the PR URL is in the format https://github.com/{owner}/{name}/pulls/{pull-number}
	prRepo, err := githubrepo.ParseUrl(flagReleasePRUrl)
	if err != nil {
		return err
	}

	prNumber, err := parsePrNumberFromUrl(flagReleasePRUrl)
	if err != nil {
		return err
	}

	prMetadata := githubrepo.PullRequestMetadata{Repo: prRepo, Number: prNumber}

	if err := waitForPullRequestReadiness(ctx, prMetadata); err != nil {
		return err
	}

	mergeCommit, err := mergePullRequest(ctx, prMetadata)
	if err != nil {
		return err
	}

	if err := waitForSync(mergeCommit); err != nil {
		return err
	}
	return nil
}

func waitForPullRequestReadiness(ctx *CommandContext, prMetadata githubrepo.PullRequestMetadata) error {
	// TODO: time out here, or let Kokoro do so?
	// TODO: make polling frequency configurable?

	const pollDelaySeconds = 60
	for {
		ready, err := waitForPullRequestReadinessSingleIteration(ctx, prMetadata)
		if ready || err != nil {
			return err
		}
		slog.Info("Sleeping before next iteration")
		time.Sleep(time.Duration(pollDelaySeconds) * time.Second)
	}
}

// A single iteration of the loop in waitForPullRequestReadiness,
// in a separate function to make it easy to indicate an "early out".
// Returns true for "PR is ready to merge", false for "keep polling".
// If this function returns false (with no error) the reason will have been logged.
// Checks performed:
// - The PR must not be merged
// - The PR must not have the label with the name specified in MergeBlockedLabel
// - The PR must have the label with the name specified in DoNotMergeLabel
// - The PR must be mergeable
// - All status checks must pass, other than conventional commits and the do-not-merge check
// - All commits in the PR must contain Librarian-Release-Id (for this release), Librarian-Release-Library and Librarian-Release-Version metadata
// - No commit in the PR must start its release notes with "FIXME"
// - There must be no commits in the head of the repo which affect libraries released by the PR
// - There must be at least one approving reviews from a member/owner of the repo, and no reviews from members/owners requesting changes
func waitForPullRequestReadinessSingleIteration(ctx *CommandContext, prMetadata githubrepo.PullRequestMetadata) (bool, error) {
	slog.Info("Checking pull request for readiness")
	pr, err := githubrepo.GetPullRequest(ctx.ctx, prMetadata.Repo, prMetadata.Number)
	if err != nil {
		return false, err
	}

	// If the PR has been merged by someone else, abort this command. (We can skip this step in the flow, if we still want to release.)
	if pr.GetMerged() {
		return false, errors.New("pull request already merged")
	}

	// If the PR is closed, wait a minute and check if it's *still* closed (to allow for deliberate "close/reopen" workflows),
	// and if it is, abort the job.
	if pr.ClosedAt != nil {
		slog.Info("PR is closed; sleeping for a minute before checking again.")
		time.Sleep(time.Duration(60) * time.Second)
		pr, err = githubrepo.GetPullRequest(ctx.ctx, prMetadata.Repo, prMetadata.Number)
		if err != nil {
			return false, err
		}
		if pr.ClosedAt != nil {
			slog.Info("PR is still closed; aborting.")
			return false, errors.New("pull request closed")
		}
		slog.Info("PR has been reopened. Continuing.")
	}

	// If we've already blocked this PR, and the user hasn't cleared the label yet, don't check anything else.
	gotDoNotMergeLabel := false
	for _, label := range pr.Labels {
		if label.GetName() == MergeBlockedLabel {
			slog.Info(fmt.Sprintf("PR still has '%s' label; skipping other checks", MergeBlockedLabel))
			return false, nil
		}
		if label.GetName() == DoNotMergeLabel {
			gotDoNotMergeLabel = true
		}
	}

	// We expect to remove the do-not-merge label ourselves (and we'll fail otherwise).
	if !gotDoNotMergeLabel {
		return false, reportBlockingReason(ctx, prMetadata, fmt.Sprintf("Label '%s' has been removed already", DoNotMergeLabel))
	}

	// If the PR isn't mergeable, that requires user action.
	if !pr.GetMergeable() {
		// This will log the reason.
		return false, reportBlockingReason(ctx, prMetadata, "PR is not mergeable (e.g. there are conflicting commit)")
	}

	// Check the commits in the pull request. If this returns false,
	// the reason will already be logged (so we don't need to log it again).
	commitStatus, err := checkPullRequestCommits(ctx, prMetadata, pr)
	if err != nil {
		return false, err
	}
	if !commitStatus {
		return false, err
	}

	// Check for approval
	approved, err := checkPullRequestApproval(ctx, prMetadata)
	if err != nil {
		return false, err
	}
	if !approved {
		slog.Info("PR not yet approved")
		return false, nil
	}

	slog.Info("All checks passed, ready to merge.")
	return true, nil
}

func mergePullRequest(ctx *CommandContext, prMetadata githubrepo.PullRequestMetadata) (string, error) {
	slog.Info("Merging release PR")
	if err := githubrepo.RemoveLabelFromPullRequest(ctx.ctx, prMetadata.Repo, prMetadata.Number, "do-not-merge"); err != nil {
		return "", err
	}
	mergeResult, err := githubrepo.MergePullRequest(ctx.ctx, prMetadata.Repo, prMetadata.Number, github.MergeMethodRebase)
	if err != nil {
		return "", err
	}

	if err := appendResultEnvironmentVariable(ctx, mergedReleaseCommitEnvVarName, *mergeResult.SHA); err != nil {
		return "", err
	}
	slog.Info("Release PR merged")
	return *mergeResult.SHA, nil
}

// If flagSyncUrlPrefix is empty, this returns immediately.
// Otherwise, polls for up to 10 minutes (once every 30 seconds) for the
// given merge commit to be available at the repo specified via flagSyncUrlPrefix.
func waitForSync(mergeCommit string) error {
	if flagSyncUrlPrefix == "" {
		return nil
	}
	req, err := http.NewRequest("GET", flagSyncUrlPrefix+mergeCommit, nil)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}
	authToken := os.Getenv(syncAuthTokenEnvironmentVariable)
	req.Header.Add("Authorization", "Bearer "+authToken)
	client := &http.Client{}

	end := time.Now().Add(time.Duration(10) * time.Minute)

	for time.Now().Before(end) {
		slog.Info("Checking if merge commit has synchronized")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		// A status of OK means the commit has synced; we're done.
		// A status of NotFound means the commit hasn't *yet* synced; sleep and keep trying.
		// Any other status is unexpected, and we abort.
		if resp.StatusCode == http.StatusOK {
			slog.Info("Merge commit has synchronized")
			return nil
		} else if resp.StatusCode == http.StatusNotFound {
			slog.Info("Merge commit has not yet synchronized; sleeping before next attempt")
			time.Sleep(time.Duration(30) * time.Second)
			continue
		} else {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("unexpected status fetching commit: %d - %s", resp.StatusCode, string(bodyBytes))
		}
	}
	return fmt.Errorf("timed out waiting for commit to sync")
}

// For each commit in the pull request, check:
// - We still have the Librarian metadata (release ID, library, version)
// - None of the paths which affect the library have been modified since the base of the PR
//
// Returns true if all the commits are fine, or false if a problem was detected, in which
// case it will have been reported on the PR, and the merge-blocking label applied.
func checkPullRequestCommits(ctx *CommandContext, prMetadata githubrepo.PullRequestMetadata, pr *github.PullRequest) (bool, error) {
	baseRepo := githubrepo.CreateGitHubRepoFromRepository(pr.Base.Repo)
	baseHeadState, err := fetchRemotePipelineState(ctx.ctx, baseRepo, *pr.Base.Ref)
	if err != nil {
		return false, err
	}
	baselineState, err := fetchRemotePipelineState(ctx.ctx, baseRepo, flagBaselineCommit)
	if err != nil {
		return false, err
	}

	// Fetch the commits which are in the PR, compared with the base (the target of the merge).
	// In most cases pr.Base.SHA will be the same as flagBaselineCommit, but the PR may have been rebased -
	// and we always only want the commits in the PR, not any that it's been rebased on top of.
	prCommits, err := githubrepo.GetDiffCommits(ctx.ctx, prMetadata.Repo, *pr.Base.SHA, *pr.Head.SHA)
	if err != nil {
		return false, err
	}

	releases, err := parseRemoteCommitsForReleases(prCommits, flagReleaseID)
	if err != nil {
		// This indicates that at least one commit is invalid - either it has missing
		// metadata, or it's for the wrong release. Report that reason, then return
		// a non-error from this function (we don't want to abort the process here).
		if err := reportBlockingReason(ctx, prMetadata, err.Error()); err != nil {
			return false, err
		}

		return false, nil
	}

	for _, release := range releases {
		if strings.HasPrefix(release.ReleaseNotes, "FIXME") {
			return false, reportBlockingReason(ctx, prMetadata, fmt.Sprintf("Release notes for '%s' need fixing", release.LibraryID))
		}
	}

	// Fetch the commits in the base repo since the baseline commit, but then fetch each individually
	// so we can tell which files were affected.
	baseCommits, err := githubrepo.GetDiffCommits(ctx.ctx, baseRepo, flagBaselineCommit, *pr.Base.Ref)
	if err != nil {
		return false, err
	}
	fullBaseCommits := []*github.RepositoryCommit{}
	for _, baseCommit := range baseCommits {
		fullCommit, err := githubrepo.GetCommit(ctx.ctx, baseRepo, *baseCommit.SHA)
		if err != nil {
			return false, err
		}
		fullBaseCommits = append(fullBaseCommits, fullCommit)
	}

	suspectReleases := []SuspectRelease{}

	slog.Info(fmt.Sprintf("Checking %d commits against %d libraries for intervening changes", len(fullBaseCommits), len(releases)))
	for _, release := range releases {
		suspectRelease := checkRelease(release, baseHeadState, baselineState, fullBaseCommits)
		if suspectRelease != nil {
			suspectReleases = append(suspectReleases, *suspectRelease)
		}
	}

	if len(suspectReleases) == 0 {
		return true, nil
	}

	var builder strings.Builder
	builder.WriteString("At least one library being released may have changed since release PR creation:\n\n")
	for _, suspectRelease := range suspectReleases {
		builder.WriteString(fmt.Sprintf("%s: %s\n", suspectRelease.LibraryID, suspectRelease.Reason))
	}
	return false, reportBlockingReason(ctx, prMetadata, builder.String())
}

// Checks that the pull request has at least one approved review, and no "changes requested" reviews.
func checkPullRequestApproval(ctx *CommandContext, prMetadata githubrepo.PullRequestMetadata) (bool, error) {
	reviews, err := githubrepo.GetPullRequestReviews(ctx.ctx, prMetadata)
	if err != nil {
		return false, err
	}

	slog.Info(fmt.Sprintf("Considering %d reviews (including history)", len(reviews)))
	// Collect all latest non-pending reviews from members/owners of the repository.
	latestReviews := make(map[int64]*github.PullRequestReview)
	for _, review := range reviews {
		association := review.GetAuthorAssociation()
		// TODO: Check the required approvals (b/417995406)
		if association != "MEMBER" && association != "OWNER" && association != "COLLABORATOR" && association != "CONTRIBUTOR" {
			slog.Info(fmt.Sprintf("Ignoring review with author association '%s'", association))
			continue
		}

		if review.GetState() == "PENDING" {
			slog.Info("Ignoring pending review")
			continue
		}

		userID := review.GetUser().GetID()
		// Need to ensure review is the latest for the user
		if current, exists := latestReviews[userID]; !exists || review.GetSubmittedAt().After(current.GetSubmittedAt().Time) {
			latestReviews[userID] = review
		}
	}

	approved := false
	for _, review := range latestReviews {
		slog.Info(fmt.Sprintf("Review at %s: %s", review.GetSubmittedAt().Format(time.RFC3339), review.GetState()))
		if review.GetState() == "APPROVED" {
			approved = true
		} else if review.GetState() == "CHANGES_REQUESTED" {
			slog.Info("Changes requested by at least one member/owner review; treating as unapproved.")
			return false, nil
		}
	}
	return approved, nil
}

func reportBlockingReason(ctx *CommandContext, prMetadata githubrepo.PullRequestMetadata, description string) error {
	slog.Warn(fmt.Sprintf("Adding '%s' label to PR and a comment with a description of '%s'", MergeBlockedLabel, description))
	comment := fmt.Sprintf("%s\n\nAfter resolving the issue, please remove the '%s' label.", description, MergeBlockedLabel)
	if err := githubrepo.AddCommentToPullRequest(ctx.ctx, prMetadata.Repo, prMetadata.Number, comment); err != nil {
		return err
	}
	if err := githubrepo.AddLabelToPullRequest(ctx.ctx, prMetadata, MergeBlockedLabel); err != nil {
		return err
	}
	return nil
}

func checkRelease(release LibraryRelease, baseHeadState, baselineState *statepb.PipelineState, baseCommits []*github.RepositoryCommit) *SuspectRelease {
	baseHeadLibrary := findLibraryByID(baseHeadState, release.LibraryID)
	if baseHeadLibrary == nil {
		return &SuspectRelease{LibraryID: release.LibraryID, Reason: "Library does not exist in head pipeline state"}
	}
	baselineLibrary := findLibraryByID(baselineState, release.LibraryID)
	if baselineLibrary == nil {
		return &SuspectRelease{LibraryID: release.LibraryID, Reason: "Library does not exist in baseline commit pipeline state"}
	}
	// TODO: Find a better way of comparing these.
	if baseHeadLibrary.String() != baselineLibrary.String() {
		return &SuspectRelease{LibraryID: release.LibraryID, Reason: "Pipeline state has changed between baseline and head"}
	}
	sourcePaths := append(baseHeadState.CommonLibrarySourcePaths, baseHeadLibrary.SourcePaths...)
	changeCommits := []string{}
	for _, commit := range baseCommits {
		if checkIfCommitAffectsAnySourcePath(commit, sourcePaths) {
			changeCommits = append(changeCommits, *commit.SHA)
		}
	}
	if len(changeCommits) > 0 {
		reason := fmt.Sprintf("Library source changed in intervening commits: %s", strings.Join(changeCommits, ", "))
		return &SuspectRelease{LibraryID: release.LibraryID, Reason: reason}
	}
	return nil
}

func checkIfCommitAffectsAnySourcePath(commit *github.RepositoryCommit, sourcePaths []string) bool {
	for _, commitFile := range commit.Files {
		changedPath := *commitFile.Filename
		for _, sourcePath := range sourcePaths {
			if changedPath == sourcePath || (strings.HasPrefix(changedPath, sourcePath) && strings.HasPrefix(changedPath, sourcePath+"/")) {
				return true
			}
		}
	}
	return false
}

func parseRemoteCommitsForReleases(commits []*github.RepositoryCommit, releaseID string) ([]LibraryRelease, error) {
	releases := []LibraryRelease{}
	for _, commit := range commits {
		release, err := parseCommitMessageForRelease(*commit.Commit.Message, *commit.SHA)
		if err != nil {
			return nil, err
		}
		if release.ReleaseID != releaseID {
			return nil, fmt.Errorf("while finding releases for release ID %s, found commit with release ID %s", releaseID, release.ReleaseID)
		}
		releases = append(releases, *release)
	}
	return releases, nil
}

func parsePrNumberFromUrl(url string) (int, error) {
	parts := strings.Split(url, "/")
	return strconv.Atoi(parts[len(parts)-1])
}
