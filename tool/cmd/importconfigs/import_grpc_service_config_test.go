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

package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestRunImportGRPCServiceConfig(t *testing.T) {
	for _, test := range []struct {
		name     string
		original []*serviceconfig.API
		configs  map[string]*serviceconfig.GRPCServiceConfig
		want     []*serviceconfig.API
	}{
		{
			name: "with retry policy",
			original: []*serviceconfig.API{
				{
					Path: "google/cloud/secretmanager/v1",
				},
			},
			configs: map[string]*serviceconfig.GRPCServiceConfig{
				"google/cloud/secretmanager/v1": {
					MethodConfig: []serviceconfig.MethodConfig{
						{
							Name: []serviceconfig.MethodName{
								{
									Service: "google.cloud.secretmanager.v1.SecretManagerService",
								},
							},
							RetryPolicy: &serviceconfig.RetryPolicy{
								MaxAttempts:          5,
								InitialBackoff:       "0.100s",
								MaxBackoff:           "60s",
								BackoffMultiplier:    1.3,
								RetryableStatusCodes: []string{"UNAVAILABLE"},
							},
						},
					},
				},
			},
			want: []*serviceconfig.API{
				{
					Path: "google/cloud/secretmanager/v1",
					GRPCServiceConfig: &serviceconfig.GRPCServiceConfig{
						MethodConfig: []serviceconfig.MethodConfig{
							{
								Name: []serviceconfig.MethodName{
									{
										Service: "google.cloud.secretmanager.v1.SecretManagerService",
									},
								},
								RetryPolicy: &serviceconfig.RetryPolicy{
									MaxAttempts:          5,
									InitialBackoff:       "0.100s",
									MaxBackoff:           "60s",
									BackoffMultiplier:    1.3,
									RetryableStatusCodes: []string{"UNAVAILABLE"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "with timeout",
			original: []*serviceconfig.API{
				{
					Path: "google/cloud/tasks/v2",
				},
			},
			configs: map[string]*serviceconfig.GRPCServiceConfig{
				"google/cloud/tasks/v2": {
					MethodConfig: []serviceconfig.MethodConfig{
						{
							Name: []serviceconfig.MethodName{
								{
									Service: "google.cloud.tasks.v2.CloudTasks",
								},
							},
							Timeout: "20s",
						},
					},
				},
			},
			want: []*serviceconfig.API{
				{
					Path: "google/cloud/tasks/v2",
					GRPCServiceConfig: &serviceconfig.GRPCServiceConfig{
						MethodConfig: []serviceconfig.MethodConfig{
							{
								Name: []serviceconfig.MethodName{
									{
										Service: "google.cloud.tasks.v2.CloudTasks",
									},
								},
								Timeout: "20s",
							},
						},
					},
				},
			},
		},
		{
			name: "no config file",
			original: []*serviceconfig.API{
				{
					Path: "google/cloud/compute/v1",
				},
			},
			want: []*serviceconfig.API{
				{
					Path: "google/cloud/compute/v1",
				},
			},
		},
		{
			name: "empty method config",
			original: []*serviceconfig.API{
				{
					Path: "google/cloud/bigquery/v2",
				},
			},
			configs: map[string]*serviceconfig.GRPCServiceConfig{
				"google/cloud/bigquery/v2": {
					MethodConfig: []serviceconfig.MethodConfig{},
				},
			},
			want: []*serviceconfig.API{
				{
					Path: "google/cloud/bigquery/v2",
				},
			},
		},
		{
			name: "sorts by path",
			original: []*serviceconfig.API{
				{
					Path: "google/cloud/tasks/v2",
				},
				{
					Path: "google/cloud/bigquery/v2",
				},
			},
			want: []*serviceconfig.API{
				{
					Path: "google/cloud/bigquery/v2",
				},
				{
					Path: "google/cloud/tasks/v2",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sdkYaml := filepath.Join(tmpDir, "sdk.yaml")
			if err := yaml.Write(sdkYaml, test.original); err != nil {
				t.Fatal(err)
			}

			googleapisDir := filepath.Join(tmpDir, "googleapis")
			for apiPath, cfg := range test.configs {
				dir := filepath.Join(googleapisDir, apiPath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				data, err := json.Marshal(cfg)
				if err != nil {
					t.Fatal(err)
				}
				// Use a realistic file name matching the *_grpc_service_config.json glob.
				filename := filepath.Base(apiPath) + "_grpc_service_config.json"
				if err := os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
					t.Fatal(err)
				}
			}

			if err := runImportGRPCServiceConfig(sdkYaml, googleapisDir); err != nil {
				t.Fatal(err)
			}
			got, err := yaml.Read[[]*serviceconfig.API](sdkYaml)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, *got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunImportGRPCServiceConfig_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		sdkYaml string
		setup   func(*testing.T) string
		wantErr error
	}{
		{
			name:    "invalid yaml file",
			sdkYaml: "non-existent.yaml",
			wantErr: os.ErrNotExist,
		},
		{
			name: "multiple config files",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				apiPath := "google/cloud/example/v1"

				// Write sdk.yaml with one API.
				sdkYaml := filepath.Join(tmpDir, "sdk.yaml")
				if err := yaml.Write(sdkYaml, []*serviceconfig.API{
					{
						Path: apiPath,
					},
				}); err != nil {
					t.Fatal(err)
				}

				// Create two grpc_service_config.json files in the same directory.
				dir := filepath.Join(tmpDir, "googleapis", apiPath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{
					"foo_grpc_service_config.json",
					"bar_grpc_service_config.json",
				} {
					if err := os.WriteFile(filepath.Join(dir, name), []byte(`{}`), 0644); err != nil {
						t.Fatal(err)
					}
				}
				return tmpDir
			},
			wantErr: errMultipleGRPCServiceConfigs,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sdkYaml := test.sdkYaml
			googleapisDir := ""
			if test.setup != nil {
				tmpDir := test.setup(t)
				sdkYaml = filepath.Join(tmpDir, "sdk.yaml")
				googleapisDir = filepath.Join(tmpDir, "googleapis")
			}
			err := runImportGRPCServiceConfig(sdkYaml, googleapisDir)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("got error %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestImportGRPCServiceConfigCommand(t *testing.T) {
	cmd := importGRPCServiceConfigCommand()
	ctx := t.Context()
	if err := cmd.Run(ctx, []string{"import-grpc-service-config", "--help"}); err != nil {
		t.Fatalf("cmd.Run() error = %v", err)
	}
}
