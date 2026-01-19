// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package librarianops orchestrates librarian operations across multiple
// repositories, including cloning, updating, generating, and creating pull
// requests.
package librarianops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/urfave/cli/v3"
)

const (
	repoRust = "google-cloud-rust"
	repoFake = "fake-repo" // used for testing

	branchPrefix = "librarianops-generateall-"
	commitTitle  = "chore: run librarian update and generate --all"
)

var supportedRepositories = map[string]struct{}{
	repoRust: {},
	repoFake: {},
}

type repoConfig struct {
	updateDiscovery bool
	runCargoUpdate  bool
}

func getRepoConfig(name string) repoConfig {
	switch name {
	case repoRust:
		return repoConfig{updateDiscovery: true, runCargoUpdate: true}
	default:
		return repoConfig{}
	}
}

// Run executes the librarianops command with the given arguments.
func Run(ctx context.Context, args ...string) error {
	cmd := &cli.Command{
		Name:      "librarianops",
		Usage:     "orchestrate librarian operations across multiple repositories",
		UsageText: "librarianops [command] [flags]",
		Commands: []*cli.Command{
			generateCommand(),
		},
	}
	return cmd.Run(ctx, args)
}

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate libraries across repositories",
		UsageText: "librarianops generate [<repo> | --all | -C <directory>]",
		Description: `Examples:
  librarianops generate google-cloud-rust
  librarianops generate --all
  librarianops generate -C ~/workspace/google-cloud-rust

Specify a repository name (e.g., google-cloud-rust) to process a single repository,
or use --all to process all repositories, or use -C to work in a specific directory
(repository name is inferred from directory name).

For each repository, librarianops will:
  1. Clone the repository to a temporary directory
  2. Create a branch: librarianops-generateall-YYYY-MM-DD
  3. Run librarian update discovery (google-cloud-rust only)
  4. Run librarian update googleapis
  5. Run librarian generate --all
  6. Run cargo update --workspace (google-cloud-rust only)
  7. Commit changes
  8. Create a pull request (pushes branch automatically)`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "process all repositories",
			},
			&cli.StringFlag{
				Name:  "C",
				Usage: "work in `directory` (assumes repo exists)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			all := cmd.Bool("all")
			workDir := cmd.String("C")
			repoName := ""
			if cmd.Args().Len() > 0 {
				repoName = cmd.Args().Get(0)
			}
			if all && repoName != "" {
				return fmt.Errorf("cannot specify both repository and --all flag")
			}
			if all && workDir != "" {
				return fmt.Errorf("cannot use -C flag with --all flag")
			}
			if workDir != "" && repoName == "" {
				absPath, err := filepath.Abs(workDir)
				if err != nil {
					return fmt.Errorf("failed to resolve directory path: %w", err)
				}
				repoName = filepath.Base(absPath)
			}
			if !all && repoName == "" {
				return fmt.Errorf("must specify either repository, --all flag, or -C flag")
			}
			return runGenerate(ctx, all, repoName, workDir)
		},
	}
}

func runGenerate(ctx context.Context, all bool, repoName, repoDir string) error {
	if all {
		for name := range supportedRepositories {
			if err := processRepo(ctx, name, ""); err != nil {
				return err
			}
		}
		return nil
	}

	if _, ok := supportedRepositories[repoName]; !ok {
		return fmt.Errorf("unsupported repository %q", repoName)
	}
	return processRepo(ctx, repoName, repoDir)
}

func processRepo(ctx context.Context, repoName, repoDir string) (err error) {
	if repoDir == "" {
		repoDir, err = os.MkdirTemp("", "librarianops-"+repoName+"-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer func() {
			cerr := os.RemoveAll(repoDir)
			if err == nil {
				err = cerr
			}
		}()
		if err := cloneRepo(ctx, repoDir, repoName); err != nil {
			return err
		}
	}
	originalWD, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	if err := os.Chdir(repoDir); err != nil {
		return fmt.Errorf("failed to change directory to %s: %w", repoDir, err)
	}
	defer func() {
		if cerr := os.Chdir(originalWD); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err := createBranch(ctx, time.Now()); err != nil {
		return err
	}

	cfg := getRepoConfig(repoName)
	if cfg.updateDiscovery {
		if err := runLibrarian(ctx, "librarian", "update", "discovery"); err != nil {
			return err
		}
	}
	if err := runLibrarian(ctx, "librarian", "update", "googleapis"); err != nil {
		return err
	}
	if err := runLibrarian(ctx, "librarian", "generate", "--all"); err != nil {
		return err
	}
	if cfg.runCargoUpdate {
		if err := runCargoUpdate(ctx); err != nil {
			return err
		}
	}

	if err := commitChanges(ctx); err != nil {
		return err
	}
	if repoName != repoFake {
		if err := createPR(ctx, repoName); err != nil {
			return err
		}
	}
	return nil
}

func cloneRepo(ctx context.Context, repoDir, repoName string) error {
	return runCommand(ctx, "gh", "repo", "clone", fmt.Sprintf("googleapis/%s", repoName), repoDir)
}

func createBranch(ctx context.Context, now time.Time) error {
	branchName := fmt.Sprintf("%s%s", branchPrefix, now.Format("2006-01-02"))
	return runCommand(ctx, "git", "checkout", "-b", branchName)
}

func commitChanges(ctx context.Context) error {
	if err := runCommand(ctx, "git", "add", "."); err != nil {
		return err
	}
	return runCommand(ctx, "git", "commit", "-m", commitTitle)
}

func createPR(ctx context.Context, repoName string) error {
	sources := "googleapis/googleapis"
	if repoName == repoRust {
		sources += " and googleapis/discovery-artifact-manager"
	}
	body := fmt.Sprintf("Update %s to the latest commit and regenerate all client libraries.", sources)
	return runCommand(ctx, "gh", "pr", "create", "--title", commitTitle, "--body", body)
}

func runCargoUpdate(ctx context.Context) error {
	return runCommand(ctx, "cargo", "update", "--workspace")
}

func runCommand(ctx context.Context, name string, args ...string) error {
	fmt.Printf("Running: %s %s\n", name, strings.Join(args, " "))
	return command.Run(ctx, name, args...)
}

func runLibrarian(ctx context.Context, args ...string) error {
	fmt.Printf("Running: %s\n", strings.Join(args, " "))
	if err := librarian.Run(ctx, args...); err != nil {
		return fmt.Errorf("librarian: %w", err)
	}
	return nil
}
