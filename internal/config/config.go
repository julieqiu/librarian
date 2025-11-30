// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

// Config represents a librarian.yaml configuration file.
type Config struct {
	// Language is one of "go", "python", or "rust".
	Language string `yaml:"language"`

	// Repo is the repository path, such as "googleapis/google-cloud-python".
	Repo string `yaml:"repo,omitempty"`

	// Sources references external source repositories.
	Sources *Sources `yaml:"sources,omitempty"`

	// Default contains default settings for all libraries.
	Default *Default `yaml:"defaults,omitempty"`

	// Libraries contains library configurations.
	Libraries []*Library `yaml:"libraries,omitempty"`
}

// Sources references external source repositories.
type Sources struct {
	// Discovery is the discovery-artifact-manager repository configuration.
	Discovery *Source `yaml:"discovery,omitempty"`

	// Googleapis is the googleapis repository configuration.
	Googleapis *Source `yaml:"googleapis,omitempty"`
}

// Source represents a source repository.
type Source struct {
	// Commit is the git commit hash or tag to use.
	Commit string `yaml:"commit"`

	// SHA256 is the expected hash of the tarball for this commit.
	SHA256 string `yaml:"sha256,omitempty"`

	// Dir is a local directory path to use instead of fetching. If set, Commit
	// and SHA256 are ignored.
	Dir string `yaml:"dir,omitempty"`
}

// Default contains default settings for all libraries.
type Default struct {
	// Output is the directory where generated code is written.
	Output string `yaml:"output,omitempty"`

	// TagFormat is the template for git tags, such as "{name}/v{version}".
	TagFormat string `yaml:"tag_format,omitempty"`

	// Remote is the git remote name, such as "upstream" or "origin".
	Remote string `yaml:"remote,omitempty"`

	// Branch is the release branch, such as "main" or "master".
	Branch string `yaml:"branch,omitempty"`

	// Rust contains Rust-specific default configuration.
	Rust *RustDefault `yaml:"rust,omitempty"`
}

// Library represents a library configuration.
type Library struct {
	// Name is the library name, such as "secretmanager" or "storage".
	Name string `yaml:"name"`

	// APIs lists the APIs to include in this library.
	APIs []*API `yaml:"apis,omitempty"`

	// Version is the library version.
	Version string `yaml:"version,omitempty"`

	// SkipGenerate disables code generation for this library.
	SkipGenerate bool `yaml:"skip_generate,omitempty"`

	// SkipRelease disables releasing for this library.
	SkipRelease bool `yaml:"skip_release,omitempty"`

	// SkipPublish disables publishing for this library.
	SkipPublish bool `yaml:"skip_publish,omitempty"`

	// Output is the directory where generated code is written.
	Output string `yaml:"output,omitempty"`

	// Keep lists files and directories to preserve during regeneration.
	Keep []string `yaml:"keep,omitempty"`

	// CopyrightYear is the copyright year for the library.
	CopyrightYear string `yaml:"copyright_year,omitempty"`

	// Rust contains Rust-specific library configuration.
	Rust *RustCrate `yaml:"rust,omitempty"`

	// Go contains Go-specific library configuration.
	Go *GoModule `yaml:"go,omitempty"`
}

// API describes an API to include in a library.
type API struct {
	// Path is the path to the API specification, such as
	// "google/cloud/secretmanager/v1".
	Path string `yaml:"path"`

	// DIREGAPIC enables DIREGAPIC (Discovery REST GAPICs) for compute/GCE.
	DIREGAPIC bool `yaml:"diregapic,omitempty"`

	// DisableGAPIC skips GAPIC generation for this API.
	DisableGAPIC bool `yaml:"disable_gapic,omitempty"`

	// Format is the API specification format, either "protobuf" (default) or
	// "discovery".
	Format string `yaml:"format,omitempty"`

	// GRPCServiceConfig is the name of the gRPC service config JSON file,
	// e.g., "cloudasset_grpc_service_config.json".
	GRPCServiceConfig string `yaml:"grpc_service_config,omitempty"`

	// Metadata controls whether metadata (e.g., gapic_metadata.json) is generated.
	Metadata *bool `yaml:"metadata,omitempty"`

	// ReleaseLevel is the release level for this API, e.g., "alpha", "beta", or "ga".
	// This is extracted from the BUILD.bazel file.
	ReleaseLevel string `yaml:"release_level,omitempty"`

	// RESTNumericEnums controls whether the REST client supports numeric enums.
	RESTNumericEnums *bool `yaml:"rest_numeric_enums,omitempty"`

	// ServiceConfig is the path to the service config file.
	ServiceConfig string `yaml:"service_config,omitempty"`

	// Transport is the transport protocol for this API, e.g., "grpc", "rest", or "grpc+rest".
	// This overrides the library-level transport setting.
	Transport string `yaml:"transport,omitempty"`

	// Go contains Go-specific API configuration.
	Go *GoPackage `yaml:"go,omitempty"`
}

// GoPackage contains Go-specific configuration for an API. These fields
// correspond to the go_gapic_library rule in BUILD.bazel files.
type GoPackage struct {
	// ImportPath is the Go import path for the generated GAPIC client,
	// e.g., "cloud.google.com/go/asset/apiv1;asset".
	ImportPath string `yaml:"import_path,omitempty"`

	// GoGRPC indicates whether to use go_grpc_library style (newer) vs
	// go_proto_library style.
	GoGRPC *bool `yaml:"go_grpc,omitempty"`

	// LegacyGRPC indicates whether go_proto_library uses the legacy
	// @io_bazel_rules_go//proto:go_grpc.
	LegacyGRPC bool `yaml:"legacy_grpc,omitempty"`
}

// GoModule contains Go-specific configuration for a library.
type GoModule struct {
	// ModulePath is the Go module path, e.g., "cloud.google.com/go/secretmanager".
	ModulePath string `yaml:"module_path,omitempty"`
}
