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

package config

// GcloudCommand contains gcloud-specific library configuration.
type GcloudCommand struct {
	// IncludeList is a comma-separated list of protobuf files to include
	// when generating gcloud commands.
	IncludeList string `yaml:"include_list,omitempty"`

	// DescriptorFiles is a comma-separated list of paths to binary
	// FileDescriptorSet files.
	DescriptorFiles string `yaml:"descriptor_files,omitempty"`

	// DescriptorFilesToGenerate is a comma-separated list of files to
	// generate from the descriptors.
	DescriptorFilesToGenerate string `yaml:"descriptor_files_to_generate,omitempty"`

	// BaseModule is the base Python module path for surface command groups.
	// Defaults to "googlecloudsdk".
	BaseModule string `yaml:"base_module,omitempty"`
}
