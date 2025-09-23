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
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
)

const (
	pullRequestSegments  = 7
	tagAndReleaseCmdName = "tag-and-release"
	releasePendingLabel  = "release:pending"
	releaseDoneLabel     = "release:done"
)

var (
	detailsRegex = regexp.MustCompile(`(?s)<details><summary>(.*?)</summary>(.*?)</details>`)
	summaryRegex = regexp.MustCompile(`(.*?): (v?\d+\.\d+\.\d+)`)
)

type tagAndReleaseRunner struct {
	ghClient    GitHubClient
	pullRequest string
}

func newTagAndReleaseRunner(cfg *config.Config) (*tagAndReleaseRunner, error) {
	languageRepo, err := cloneOrOpenRepo(cfg.WorkRoot, cfg.Repo, cfg.APISourceDepth, cfg.Branch, cfg.CI, cfg.GitHubToken)
	if err != nil {
		return nil, err
	}
	state, err := loadRepoState(languageRepo, "")
	if err != nil {
		return nil, err
	}
	ghClient, err := newGitHubClient(cfg.Repo, cfg.GitHubToken, languageRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}
	repo, err := github.ParseRemote(cfg.Repo)
	if err != nil {
		return nil, err
	}
	ghClient := github.NewClient(cfg.GitHubToken, repo)
	// If a custom GitHub API endpoint is provided (for testing),
	// parse it and set it as the BaseURL on the GitHub client.
	if cfg.GitHubAPIEndpoint != "" {
		endpoint, err := url.Parse(cfg.GitHubAPIEndpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to parse github-api-endpoint: %w", err)
		}
		ghClient.BaseURL = endpoint
	}
	return &tagAndReleaseRunner{
		ghClient:    ghClient,
		pullRequest: cfg.PullRequest,
		repo:        languageRepo,
		state:       state,
	}, nil
}

func (r *tagAndReleaseRunner) run(ctx context.Context) error {
	slog.Info("running tag-and-release command")
	prs, err := r.determinePullRequestsToProcess(ctx)
	if err != nil {
		return err
	}
	if len(prs) == 0 {
		slog.Info("no pull requests to process, exiting")
		return nil
	}

	var hadErrors bool
	for _, p := range prs {
		if err := r.processPullRequest(ctx, p); err != nil {
			slog.Error("failed to process pull request", "pr", p.GetNumber(), "error", err)
			hadErrors = true
			continue
		}
		slog.Info("processed pull request", "pr", p.GetNumber())
	}
	slog.Info("finished processing all pull requests")

	if hadErrors {
		return errors.New("failed to process some pull requests")
	}
	return nil
}

func (r *tagAndReleaseRunner) determinePullRequestsToProcess(ctx context.Context) ([]*github.PullRequest, error) {
	slog.Info("determining pull requests to process")
	if r.pullRequest != "" {
		slog.Info("processing a single pull request", "pr", r.pullRequest)
		ss := strings.Split(r.pullRequest, "/")
		if len(ss) != pullRequestSegments {
			return nil, fmt.Errorf("invalid pull request format: %s", r.pullRequest)
		}
		prNum, err := strconv.Atoi(ss[pullRequestSegments-1])
		if err != nil {
			return nil, fmt.Errorf("invalid pull request number: %s", ss[pullRequestSegments-1])
		}
		pr, err := r.ghClient.GetPullRequest(ctx, prNum)
		if err != nil {
			return nil, fmt.Errorf("failed to get pull request %d: %w", prNum, err)
		}
		return []*github.PullRequest{pr}, nil
	}

	slog.Info("searching for pull requests to tag and release")
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	query := fmt.Sprintf("label:%s merged:>=%s", releasePendingLabel, thirtyDaysAgo)
	prs, err := r.ghClient.SearchPullRequests(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search pull requests: %w", err)
	}
	return prs, nil
}

func (r *tagAndReleaseRunner) processPullRequest(ctx context.Context, p *github.PullRequest) error {
	slog.Info("processing pull request", "pr", p.GetNumber())
	releases := parsePullRequestBody(p.GetBody())
	if len(releases) == 0 {
		slog.Warn("no release details found in pull request body, skipping")
		return nil
	}

	// Load library state from remote repo
	libraryState, err := loadRepoStateFromGitHub(ctx, r.ghClient, *p.Base.Ref)
	if err != nil {
		return err
	}

	// Add a tag to the release commit to trigger louhi flow: "release-please-{pr number}"
	// TODO: remove this logic as part of https://github.com/googleapis/librarian/issues/2044
	commitSha := p.GetMergeCommitSHA()
	tagName := fmt.Sprintf("release-please-%d", p.GetNumber())
	if err := r.ghClient.CreateTag(ctx, tagName, commitSha); err != nil {
		return fmt.Errorf("failed to create tag %s: %w", tagName, err)
	}
	for _, release := range releases {
		slog.Info("creating release", "library", release.Library, "version", release.Version)

		tagFormat, err := determineTagFormat(release.Library, libraryState)
		if err != nil {
			slog.Warn("could not determine tag format", "library", release.Library)
			return err
		}

		// Create the release.
		tagName := formatTag(tagFormat, release.Library, release.Version)
		releaseName := fmt.Sprintf("%s %s", release.Library, release.Version)
		if _, err := r.ghClient.CreateRelease(ctx, tagName, releaseName, release.Body, commitSha); err != nil {
			return fmt.Errorf("failed to create release: %w", err)
		}

	}
	return r.replacePendingLabel(ctx, p)
}

func determineTagFormat(libraryID string, librarianState *config.LibrarianState) (string, error) {
	// TODO(#2177): read from LibrarianConfig
	libraryState := librarianState.LibraryByID(libraryID)
	if libraryState == nil {
		return "", fmt.Errorf("library %s not found", libraryID)
	}
	if libraryState.TagFormat != "" {
		return libraryState.TagFormat, nil
	}
	return "", fmt.Errorf("library %s did not configure tag_format", libraryID)
}

// libraryRelease holds the parsed information from a pull request body.
type libraryRelease struct {
	// Body contains the release notes.
	Body string
	// Library is the library id of the library being released
	Library string
	// Version is the version that is being released
	Version string
}

// parsePullRequestBody parses a string containing release notes and returns a slice of ParsedPullRequestBody.
func parsePullRequestBody(body string) []libraryRelease {
	slog.Info("parsing pull request body")
	var parsedBodies []libraryRelease
	matches := detailsRegex.FindAllStringSubmatch(body, -1)
	for _, match := range matches {
		summary := match[1]
		content := strings.TrimSpace(match[2])

		summaryMatches := summaryRegex.FindStringSubmatch(summary)
		if len(summaryMatches) == 3 {
			slog.Info("parsed pull request body", "library", summaryMatches[1], "version", summaryMatches[2])
			library := strings.TrimSpace(summaryMatches[1])
			version := strings.TrimSpace(summaryMatches[2])
			parsedBodies = append(parsedBodies, libraryRelease{
				Version: version,
				Library: library,
				Body:    content,
			})
		} else {
			slog.Warn("failed to parse pull request body", "match", strings.Join(match, "\n"))
		}
	}

	return parsedBodies
}

// replacePendingLabel is a helper function that replaces the `release:pending` label with `release:done`.
func (r *tagAndReleaseRunner) replacePendingLabel(ctx context.Context, p *github.PullRequest) error {
	var currentLabels []string
	for _, label := range p.Labels {
		currentLabels = append(currentLabels, label.GetName())
	}
	currentLabels = slices.DeleteFunc(currentLabels, func(s string) bool {
		return s == releasePendingLabel
	})
	currentLabels = append(currentLabels, releaseDoneLabel)
	if err := r.ghClient.ReplaceLabels(ctx, p.GetNumber(), currentLabels); err != nil {
		return fmt.Errorf("failed to replace labels: %w", err)
	}
	return nil
}
