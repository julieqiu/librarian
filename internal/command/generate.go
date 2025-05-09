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
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/githubrepo"
	"github.com/googleapis/librarian/internal/gitrepo"
)

var CmdGenerate = &Command{
	Name:  "generate",
	Short: "Generate client library code for an API",
	flagFunctions: []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagAPIPath,
		addFlagAPIRoot,
		addFlagLanguage,
		addFlagBuild,
		addFlagRepoUrl,
		addFlagSecretsProject,
	},
	// By default don't clone a language repo, we will clone later only if library exists in language repo.
	maybeGetLanguageRepo: func(workRoot string) (*gitrepo.Repo, error) {
		return nil, nil
	},
	execute: func(ctx *CommandContext) error {
		if err := validateRequiredFlag("api-path", flagAPIPath); err != nil {
			return err
		}
		if err := validateRequiredFlag("api-root", flagAPIRoot); err != nil {
			return err
		}

		outputDir := filepath.Join(ctx.workRoot, "output")
		if err := os.Mkdir(outputDir, 0755); err != nil {
			return err
		}
		slog.Info(fmt.Sprintf("Code will be generated in %s", outputDir))

		if err := runGenerateCommand(ctx, outputDir); err != nil {
			return err
		}

		if flagBuild {
			if err := container.BuildRaw(ctx.containerConfig, outputDir, flagAPIPath); err != nil {
				return err
			}
		}
		return nil
	},
}

// Checks if the library exists in the remote pipeline state, if so use GenerateLibrary command
// otherwise use GenerateRaw command
// In case of non fatal error when looking up library, we will fallback to GenerateRaw command
// and log the error
func runGenerateCommand(ctx *CommandContext, outputDir string) error {
	libraryID, err := checkIfLibraryExistsInLanguageRepo(ctx)
	if err != nil {
		return err
	}
	apiRoot, err := filepath.Abs(flagAPIRoot)
	if err != nil {
		return err
	}
	if libraryID != "" {
		ctx.languageRepo, err = cloneOrOpenLanguageRepo(ctx.workRoot)
		if err != nil {
			slog.Warn("Unable to checkout language repo ", "error", err)
			return err
		}
		generatorInput := filepath.Join(ctx.languageRepo.Dir, "generator-input")
		slog.Info("Performing refined generation for library ID", "libraryID", libraryID)
		args1:= []string{
			"run",
			"--rm",
			"-v",
			generatorInput+":/generator-input",
			"alpine",
			"ls -l /generator-input",

			
		docker run --rm -v /tmp/librarian-20250509T181304/google-cloud-dotnet/generator-input:/generator-input alpine ls -l /generator-input
		
		return container.GenerateLibrary(ctx.containerConfig, apiRoot, outputDir, generatorInput, libraryID)
	} else {
		slog.Info("No matching library found performing raw generation", "flagAPIPath", flagAPIPath)
		return container.GenerateRaw(ctx.containerConfig, apiRoot, outputDir, flagAPIPath)
	}
}

// Checks if the library with the given API path exists in the remote pipeline state
// If library exists, returns the library ID, otherwise returns an empty string
func checkIfLibraryExistsInLanguageRepo(ctx *CommandContext) (string, error) {
	if flagRepoUrl == "" {
		slog.Warn("repo url is not specified, cannot check if library exists")
		return "", nil
	}
	languageRepoMetadata, err := githubrepo.ParseUrl(flagRepoUrl)
	if err != nil {
		slog.Warn("failed to parse", "repo url:", flagRepoUrl, "error", err)
		return "", err
	}
	pipelineState, err := fetchRemotePipelineState(ctx.ctx, languageRepoMetadata, "HEAD")
	if err != nil {
		slog.Warn("failed to get pipeline state file", "error", err)
		return "", err
	}
	if pipelineState != nil {
		return findLibraryIDByApiPath(pipelineState, flagAPIPath), nil
	} else {
		slog.Warn("Pipeline state file is null")
	}
	return "", nil
}
