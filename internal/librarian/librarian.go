// Copyright 2024 Google LLC
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

// Package librarian provides the core implementation for the Librarian CLI tool.
package librarian

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/googleapis/librarian/internal/cli"
)

// Run executes the Librarian CLI with the given command line arguments.
func Run(ctx context.Context, arg ...string) error {
	cmd := newLibrarianCommand()
	return cmd.Run(ctx, arg)
}

func setupLogger(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}
	handler := slog.NewTextHandler(os.Stderr, opts)
	slog.SetDefault(slog.New(handler))
}

func newLibrarianCommand() *cli.Command {
	cmdVersion := &cli.Command{
		Short:     "version prints the version information",
		UsageLine: "librarian version",
		Long:      versionLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println(cli.Version())
			return nil
		},
	}
	cmdVersion.Init()

	cmdConfig := &cli.Command{
		Short:     "config manages repository configuration",
		UsageLine: "librarian config <command> [arguments]",
		Long:      configLongHelp,
		Commands: []*cli.Command{
			newCmdConfigGet(),
			newCmdConfigSet(),
			newCmdConfigUpdate(),
		},
	}
	cmdConfig.Init()

	cmd := &cli.Command{
		Short:     "librarian automates the maintenance and release of versioned directories",
		UsageLine: "librarian <command> [arguments]",
		Long:      librarianLongHelp,
		Commands: []*cli.Command{
			newCmdInit(),
			newCmdAdd(),
			newCmdEdit(),
			newCmdRemove(),
			newCmdGenerate(),
			newCmdPrepare(),
			newCmdRelease(),
			cmdConfig,
			cmdVersion,
		},
	}
	cmd.Init()
	return cmd
}

func newCmdGenerate() *cli.Command {
	var verbose bool
	var all bool
	var commit bool
	cmdGenerate := &cli.Command{
		Short:     "generate generates or regenerates code for tracked directories",
		UsageLine: "librarian generate <path> | librarian generate --all",
		Long:      generateLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			setupLogger(verbose)
			slog.Debug("generate command verbose logging")
			return fmt.Errorf("generate command not yet implemented")
		},
	}
	cmdGenerate.Init()
	cmdGenerate.Flags.BoolVar(&all, "all", false, "generate all artifacts with a generate section")
	cmdGenerate.Flags.BoolVar(&commit, "commit", false, "create a git commit with the changes")
	addFlagVerbose(cmdGenerate.Flags, &verbose)
	return cmdGenerate
}

func newCmdRelease() *cli.Command {
	var verbose bool
	var all bool
	cmdRelease := &cli.Command{
		Short:     "release tags the prepared version and updates recorded release state",
		UsageLine: "librarian release <path> | librarian release --all",
		Long:      releaseLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			setupLogger(verbose)
			slog.Debug("release command verbose logging")
			return fmt.Errorf("release command not yet implemented")
		},
	}
	cmdRelease.Init()
	cmdRelease.Flags.BoolVar(&all, "all", false, "release all artifacts with a prepared release")
	addFlagVerbose(cmdRelease.Flags, &verbose)
	return cmdRelease
}

func newCmdPrepare() *cli.Command {
	var verbose bool
	var all bool
	var commit bool
	cmdPrepare := &cli.Command{
		Short:     "prepare determines next version, updates metadata, and prepares release notes",
		UsageLine: "librarian prepare <path> | librarian prepare --all",
		Long:      prepareLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			setupLogger(verbose)
			slog.Debug("prepare command verbose logging")
			return fmt.Errorf("prepare command not yet implemented")
		},
	}
	cmdPrepare.Init()
	cmdPrepare.Flags.BoolVar(&all, "all", false, "prepare all artifacts with a release section")
	cmdPrepare.Flags.BoolVar(&commit, "commit", false, "create a git commit with the changes")
	addFlagVerbose(cmdPrepare.Flags, &verbose)
	return cmdPrepare
}

func newCmdInit() *cli.Command {
	var verbose bool
	cmdInit := &cli.Command{
		Short:     "init initializes a repository for library management",
		UsageLine: "librarian init [language]",
		Long:      initLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			setupLogger(verbose)
			slog.Debug("init command verbose logging")
			return fmt.Errorf("init command not yet implemented")
		},
	}
	cmdInit.Init()
	addFlagVerbose(cmdInit.Flags, &verbose)
	return cmdInit
}

func newCmdAdd() *cli.Command {
	var verbose bool
	var commit bool
	cmdAdd := &cli.Command{
		Short:     "add tracks a directory for management",
		UsageLine: "librarian add <path> [api...]",
		Long:      addLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			setupLogger(verbose)
			slog.Debug("add command verbose logging")
			return fmt.Errorf("add command not yet implemented")
		},
	}
	cmdAdd.Init()
	cmdAdd.Flags.BoolVar(&commit, "commit", false, "create a git commit with the changes")
	addFlagVerbose(cmdAdd.Flags, &verbose)
	return cmdAdd
}

func newCmdEdit() *cli.Command {
	var verbose bool
	var metadata []string
	var language string
	var keep []string
	var remove []string
	var exclude []string
	cmdEdit := &cli.Command{
		Short:     "edit edits artifact configuration",
		UsageLine: "librarian edit <path> [flags]",
		Long:      editLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			setupLogger(verbose)
			slog.Debug("edit command verbose logging")
			return fmt.Errorf("edit command not yet implemented")
		},
	}
	cmdEdit.Init()
	cmdEdit.Flags.Var((*stringSliceFlag)(&metadata), "metadata", "set metadata field (KEY=VALUE, can be repeated)")
	cmdEdit.Flags.StringVar(&language, "language", "", "set language-specific metadata (LANG:KEY=VALUE)")
	cmdEdit.Flags.Var((*stringSliceFlag)(&keep), "keep", "add file/directory to keep list (can be repeated)")
	cmdEdit.Flags.Var((*stringSliceFlag)(&remove), "remove", "add file/directory to remove list (can be repeated)")
	cmdEdit.Flags.Var((*stringSliceFlag)(&exclude), "exclude", "add file/directory to exclude list (can be repeated)")
	addFlagVerbose(cmdEdit.Flags, &verbose)
	return cmdEdit
}

func newCmdRemove() *cli.Command {
	var verbose bool
	cmdRemove := &cli.Command{
		Short:     "remove stops tracking a directory",
		UsageLine: "librarian remove <path>",
		Long:      removeLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			setupLogger(verbose)
			slog.Debug("remove command verbose logging")
			return fmt.Errorf("remove command not yet implemented")
		},
	}
	cmdRemove.Init()
	addFlagVerbose(cmdRemove.Flags, &verbose)
	return cmdRemove
}

func newCmdConfigGet() *cli.Command {
	var verbose bool
	cmdConfigGet := &cli.Command{
		Short:     "get reads a configuration value",
		UsageLine: "librarian config get <key>",
		Long:      configGetLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			setupLogger(verbose)
			slog.Debug("config get command verbose logging")
			return fmt.Errorf("config get command not yet implemented")
		},
	}
	cmdConfigGet.Init()
	addFlagVerbose(cmdConfigGet.Flags, &verbose)
	return cmdConfigGet
}

func newCmdConfigSet() *cli.Command {
	var verbose bool
	cmdConfigSet := &cli.Command{
		Short:     "set sets a configuration value",
		UsageLine: "librarian config set <key> <value>",
		Long:      configSetLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			setupLogger(verbose)
			slog.Debug("config set command verbose logging")
			return fmt.Errorf("config set command not yet implemented")
		},
	}
	cmdConfigSet.Init()
	addFlagVerbose(cmdConfigSet.Flags, &verbose)
	return cmdConfigSet
}

func newCmdConfigUpdate() *cli.Command {
	var verbose bool
	var all bool
	cmdConfigUpdate := &cli.Command{
		Short:     "update updates toolchain versions to latest",
		UsageLine: "librarian config update [key] | librarian config update --all",
		Long:      configUpdateLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			setupLogger(verbose)
			slog.Debug("config update command verbose logging")
			return fmt.Errorf("config update command not yet implemented")
		},
	}
	cmdConfigUpdate.Init()
	cmdConfigUpdate.Flags.BoolVar(&all, "all", false, "update all toolchain versions")
	addFlagVerbose(cmdConfigUpdate.Flags, &verbose)
	return cmdConfigUpdate
}

// stringSliceFlag implements flag.Value for string slices.
type stringSliceFlag []string

// String returns a string representation of the stringSliceFlag.
func (s *stringSliceFlag) String() string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("%v", *s)
}

// Set appends a value to the stringSliceFlag.
func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}
