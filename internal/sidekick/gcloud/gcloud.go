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

package gcloud

import (
	"context"
	"fmt"

	"github.com/googleapis/librarian/internal/cli"
)

// Run executes the Librarian CLI with the given command line arguments.
func Run(ctx context.Context, arg ...string) error {
	cmd := newGcloudCommand()
	return cmd.Run(ctx, arg)
}

func newGcloudCommand() *cli.Command {
	cmdVersion := &cli.Command{
		Short:     "version prints the version information",
		UsageLine: "gcloud version",
		Long:      "version prints the version information",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println(cli.Version())
			return nil
		},
	}
	cmdVersion.Init()

	cmd := &cli.Command{
		Short:     "gcloud autogenerates gcloud command YAML files",
		UsageLine: "gcloud <command> [arguments]",
		Long:      "gcloud autogenerates gcloud command YAML files",
		Commands: []*cli.Command{
			cmdVersion,
		},
	}
	cmd.Init()
	return cmd
}
