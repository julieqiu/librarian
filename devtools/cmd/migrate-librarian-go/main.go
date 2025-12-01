// Copyright 2025 Google LLC
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

// Command migrate-librarian-go converts google-cloud-go's .librarian configuration
// files (state.yaml, config.yaml, repo-config.yaml) to a librarian.yaml file.
//
// Usage:
//
//	go run ./devtools/cmd/migrate-librarian-go
//
// By default, this tool fetches the latest commit from
// github.com/googleapis/google-cloud-go using the GitHub API, downloads
// the tarball to $LIBRARIAN_CACHE (or $HOME/.cache/librarian if not set),
// reads the .librarian directory, and generates a librarian.yaml file.
//
// Flags:
//
//	-repo      Override the repository path (skips downloading)
//	-output    Output file path (default: librarian.yaml)
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/config/bazel"
	"github.com/googleapis/librarian/internal/fetch"
	"gopkg.in/yaml.v3"
)

const (
	defaultRepoOrg  = "googleapis"
	defaultRepoName = "google-cloud-go"
	defaultBranch   = "main"
)

// Input structures from .librarian files

// StateFile represents the .librarian/state.yaml file.
type StateFile struct {
	Image     string          `yaml:"image"`
	Libraries []*StateLibrary `yaml:"libraries"`
}

// StateLibrary represents a library in state.yaml.
type StateLibrary struct {
	ID                  string      `yaml:"id"`
	Version             string      `yaml:"version"`
	LastGeneratedCommit string      `yaml:"last_generated_commit"`
	APIs                []*StateAPI `yaml:"apis"`
	SourceRoots         []string    `yaml:"source_roots"`
	PreserveRegex       []string    `yaml:"preserve_regex"`
	RemoveRegex         []string    `yaml:"remove_regex"`
	ReleaseExcludePaths []string    `yaml:"release_exclude_paths"`
	TagFormat           string      `yaml:"tag_format"`
}

// StateAPI represents an API in state.yaml.
type StateAPI struct {
	Path          string `yaml:"path"`
	ServiceConfig string `yaml:"service_config"`
}

// ConfigFile represents the .librarian/config.yaml file.
type ConfigFile struct {
	GlobalFilesAllowlist []AllowlistEntry `yaml:"global_files_allowlist"`
	Libraries            []*ConfigLibrary `yaml:"libraries"`
}

// AllowlistEntry represents an entry in global_files_allowlist.
type AllowlistEntry struct {
	Path        string `yaml:"path"`
	Permissions string `yaml:"permissions"`
}

// ConfigLibrary represents a library in config.yaml.
type ConfigLibrary struct {
	ID             string `yaml:"id"`
	ReleaseBlocked bool   `yaml:"release_blocked"`
}

// RepoConfigFile represents the .librarian/generator-input/repo-config.yaml file.
type RepoConfigFile struct {
	Modules []*RepoConfigModule `yaml:"modules"`
}

// RepoConfigModule represents a module in repo-config.yaml.
type RepoConfigModule struct {
	Name                        string           `yaml:"name"`
	ModulePathVersion           string           `yaml:"module_path_version,omitempty"`
	DeleteGenerationOutputPaths []string         `yaml:"delete_generation_output_paths,omitempty"`
	APIs                        []*RepoConfigAPI `yaml:"apis,omitempty"`
}

// RepoConfigAPI represents an API in repo-config.yaml.
type RepoConfigAPI struct {
	Path         string   `yaml:"path"`
	DisableGAPIC bool     `yaml:"disable_gapic,omitempty"`
	NestedProtos []string `yaml:"nested_protos,omitempty"`
	ProtoPackage string   `yaml:"proto_package,omitempty"`
}


func main() {
	repoPath := flag.String("repo", "", "path to the google-cloud-go repository (if empty, downloads from GitHub)")
	output := flag.String("output", "librarian.yaml", "output file path")
	flag.Parse()

	// If no repo path provided, download from GitHub.
	path := *repoPath
	if path == "" {
		var err error
		path, err = fetchRepo(defaultRepoOrg, defaultRepoName, defaultBranch)
		if err != nil {
			log.Fatalf("failed to fetch repository: %v", err)
		}
		fmt.Printf("using repository at %s\n", path)
	}

	cfg, err := migrate(path)
	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Write output
	f, err := os.Create(*output)
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		log.Fatalf("failed to encode YAML: %v", err)
	}

	fmt.Printf("wrote %s\n", *output)
}

// fetchRepo fetches the latest commit from GitHub and downloads the repository
// tarball to the cache directory. Returns the path to the extracted repository.
// If the commit already exists in the cache, skips downloading.
func fetchRepo(org, repo, branch string) (string, error) {
	// Get the latest commit SHA from GitHub API.
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", org, repo, branch)
	fmt.Printf("fetching latest commit from %s...\n", apiURL)

	sha, err := fetch.LatestSha(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to get latest commit SHA: %w", err)
	}
	fmt.Printf("latest commit: %s\n", sha)

	// Check if this commit already exists in the cache.
	repoPath := fmt.Sprintf("github.com/%s/%s", org, repo)
	cachedDir := fetch.CachedRepoDir(repoPath, sha)
	if cachedDir != "" {
		fmt.Printf("using cached repository at %s\n", cachedDir)
		return cachedDir, nil
	}

	// Get the SHA256 of the tarball.
	tarballURL := fmt.Sprintf("https://github.com/%s/%s/archive/%s.tar.gz", org, repo, sha)
	fmt.Printf("computing SHA256 for %s...\n", tarballURL)

	checksum, err := fetch.Sha256(tarballURL)
	if err != nil {
		return "", fmt.Errorf("failed to compute SHA256: %w", err)
	}
	fmt.Printf("SHA256: %s\n", checksum)

	// Download and extract the tarball using the standard cache mechanism.
	dir, err := fetch.RepoDir(repoPath, sha, checksum)
	if err != nil {
		return "", fmt.Errorf("failed to download repository: %w", err)
	}

	return dir, nil
}

// fetchGoogleapis fetches the googleapis commit used by most google-cloud-go libraries,
// computes its SHA256, and returns the source config and the local directory path.
func fetchGoogleapis() (*config.Source, string, error) {
	// Use the googleapis commit from google-cloud-go's .librarian/state.yaml.
	sha := "7c0dcbba70fc5dd64655a77e74dbbf8aaf04c1bf"
	fmt.Printf("using googleapis commit: %s\n", sha)

	tarballURL := fmt.Sprintf("https://github.com/googleapis/googleapis/archive/%s.tar.gz", sha)
	fmt.Printf("computing SHA256 for %s...\n", tarballURL)

	checksum, err := fetch.Sha256(tarballURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to compute SHA256: %w", err)
	}
	fmt.Printf("googleapis SHA256: %s\n", checksum)

	// Download and extract googleapis to get the directory.
	dir, err := fetch.RepoDir("github.com/googleapis/googleapis", sha, checksum)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download googleapis: %w", err)
	}
	fmt.Printf("googleapis directory: %s\n", dir)

	return &config.Source{
		Commit: sha,
		SHA256: checksum,
	}, dir, nil
}

// migrate reads the .librarian configuration files and converts them to
// a librarian.yaml configuration.
func migrate(repoPath string) (*config.Config, error) {
	stateFile, err := readStateFile(filepath.Join(repoPath, ".librarian", "state.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read state.yaml: %w", err)
	}

	configFile, err := readConfigFile(filepath.Join(repoPath, ".librarian", "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read config.yaml: %w", err)
	}

	repoConfig, err := readRepoConfigFile(filepath.Join(repoPath, ".librarian", "generator-input", "repo-config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read repo-config.yaml: %w", err)
	}

	// Fetch googleapis commit, sha256, and directory.
	googleapisSrc, googleapisDir, err := fetchGoogleapis()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch googleapis: %w", err)
	}

	// Build lookup maps
	releaseBlocked := make(map[string]bool)
	for _, lib := range configFile.Libraries {
		releaseBlocked[lib.ID] = lib.ReleaseBlocked
	}

	moduleConfigs := make(map[string]*RepoConfigModule)
	for _, mod := range repoConfig.Modules {
		moduleConfigs[mod.Name] = mod
	}

	// Build output config
	output := &config.Config{
		Language: "go",
		Repo:     "googleapis/google-cloud-go",
		Sources: &config.Sources{
			Googleapis: googleapisSrc,
		},
		Default: &config.Default{
			TagFormat: "{name}/v{version}",
			Remote:    "origin",
			Branch:    "main",
		},
	}

	// Convert libraries
	for _, stateLib := range stateFile.Libraries {
		lib := convertLibrary(stateLib, releaseBlocked[stateLib.ID], moduleConfigs[stateLib.ID], googleapisDir)
		output.Libraries = append(output.Libraries, lib)
	}

	// Filter out auto-discoverable libraries (they will be discovered from googleapis).
	output.Libraries = filterAutoDiscoverable(output.Libraries)

	// Sort libraries by name
	slices.SortFunc(output.Libraries, func(a, b *config.Library) int {
		return strings.Compare(a.Name, b.Name)
	})

	return output, nil
}

// filterAutoDiscoverable removes libraries that can be auto-discovered from
// googleapis and don't need explicit configuration.
func filterAutoDiscoverable(libs []*config.Library) []*config.Library {
	var result []*config.Library
	for _, lib := range libs {
		if !isAutoDiscoverable(lib) {
			result = append(result, lib)
		}
	}
	return result
}

// isAutoDiscoverable returns true if a library can be auto-discovered
// from googleapis and doesn't need to be in the config.
func isAutoDiscoverable(lib *config.Library) bool {
	// Must have a name to be discoverable.
	if lib.Name == "" {
		return false
	}
	// Libraries with special flags must be kept.
	if lib.SkipGenerate || lib.SkipRelease || lib.SkipPublish {
		return false
	}
	// Libraries with keep files must be kept.
	if len(lib.Keep) > 0 {
		return false
	}
	// Libraries with custom output must be kept.
	if lib.Output != "" {
		return false
	}
	// Libraries with custom tag format must be kept.
	if lib.TagFormat != "" {
		return false
	}
	// Libraries with Go module config must be kept.
	if lib.Go != nil {
		return false
	}
	// Check each API for special config.
	for _, api := range lib.APIs {
		if !isAutoDiscoverableAPI(api) {
			return false
		}
	}
	return true
}

// isAutoDiscoverableAPI returns true if an API can be auto-discovered
// and doesn't need special configuration.
func isAutoDiscoverableAPI(api *config.API) bool {
	// APIs with special format (e.g., discovery) must be kept.
	if api.Format != "" {
		return false
	}
	// APIs with disable_gapic must be kept.
	if api.DisableGAPIC {
		return false
	}
	// APIs with Go config must be kept.
	if api.Go != nil {
		return false
	}
	return true
}

// deriveGoGapicPackage derives the go-gapic-package value from the API path.
// The format is: "cloud.google.com/go/{path}/api{version};{packagename}"
func deriveGoGapicPackage(apiPath string) string {
	if apiPath == "" {
		return ""
	}

	// Strip "google/" prefix.
	path := strings.TrimPrefix(apiPath, "google/")
	// Strip "cloud/" prefix (for google/cloud/... paths).
	path = strings.TrimPrefix(path, "cloud/")

	// Split into components.
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}

	// Find the version component (starts with "v" followed by digit).
	versionIdx := -1
	for i, part := range parts {
		if len(part) > 1 && part[0] == 'v' && part[1] >= '0' && part[1] <= '9' {
			versionIdx = i
			break
		}
	}
	if versionIdx == -1 || versionIdx == 0 {
		return ""
	}

	// Path components before version.
	pathParts := parts[:versionIdx]
	version := parts[versionIdx]

	// Package name is the last component before version.
	packageName := pathParts[len(pathParts)-1]
	packageName = strings.ReplaceAll(packageName, "-", "")

	// Build import path.
	importPath := strings.Join(pathParts, "/")
	return fmt.Sprintf("cloud.google.com/go/%s/api%s;%s", importPath, version, packageName)
}

func convertLibrary(state *StateLibrary, releaseBlocked bool, moduleConfig *RepoConfigModule, googleapisDir string) *config.Library {
	lib := &config.Library{
		Name: state.ID,
	}

	// Skip generation for handwritten libraries (no APIs).
	if len(state.APIs) == 0 {
		lib.SkipGenerate = true
	}

	// Skip generation if any API path doesn't exist in the current googleapis commit.
	for _, api := range state.APIs {
		apiDir := filepath.Join(googleapisDir, api.Path)
		if _, err := os.Stat(apiDir); os.IsNotExist(err) {
			fmt.Printf("warning: skipping %s: API path %s not found in googleapis\n", state.ID, api.Path)
			lib.SkipGenerate = true
			break
		}
	}

	// Set output based on name (for non-standard cases)
	if state.ID == "root-module" {
		lib.Output = "."
	}

	// Handle release blocking
	if releaseBlocked {
		lib.SkipRelease = true
	}

	// Convert preserve_regex to keep (strip regex anchors and unescape)
	for _, pattern := range state.PreserveRegex {
		path := strings.TrimPrefix(pattern, "^")
		path = strings.TrimSuffix(path, "$")
		path = strings.ReplaceAll(path, `\.`, ".")
		lib.Keep = append(lib.Keep, path)
	}

	// Handle custom tag format (only if different from default)
	if state.TagFormat != "" && state.TagFormat != "{id}/v{version}" {
		lib.TagFormat = strings.ReplaceAll(state.TagFormat, "{id}", "{name}")
	}

	// Build API config map from repo-config.yaml
	apiConfigs := make(map[string]*RepoConfigAPI)
	if moduleConfig != nil {
		for _, api := range moduleConfig.APIs {
			apiConfigs[api.Path] = api
		}

		// Handle module_path_version (for v2+ modules like "storage/v2")
		if moduleConfig.ModulePathVersion != "" {
			lib.Go = &config.GoModule{
				ModulePath: fmt.Sprintf("cloud.google.com/go/%s/%s", state.ID, moduleConfig.ModulePathVersion),
			}
		}
	}

	// Set module path for libraries with version suffix in name (like "bigquery/v2")
	if lib.Go == nil && strings.Contains(state.ID, "/v") {
		lib.Go = &config.GoModule{
			ModulePath: fmt.Sprintf("cloud.google.com/go/%s", state.ID),
		}
	}

	// Convert APIs
	for _, stateAPI := range state.APIs {
		api := &config.API{
			Path: stateAPI.Path,
		}

		// Parse BUILD.bazel to get GAPIC configuration.
		buildFile := filepath.Join(googleapisDir, stateAPI.Path, "BUILD.bazel")
		bazelCfg, err := bazel.Parse(buildFile)
		if err != nil {
			fmt.Printf("warning: failed to parse %s: %v\n", buildFile, err)
		} else if bazelCfg.HasGAPIC {
			// Use import path from BUILD.bazel.
			if bazelCfg.GAPICImportPath != "" {
				if api.Go == nil {
					api.Go = &config.GoPackage{}
				}
				api.Go.ImportPath = bazelCfg.GAPICImportPath
			}

			if bazelCfg.HasLegacyGRPC {
				if api.Go == nil {
					api.Go = &config.GoPackage{}
				}
				api.Go.LegacyGRPC = true
			}
		} else if !bazelCfg.HasGAPIC {
			// No GAPIC rule means this API doesn't need GAPIC generation.
			api.DisableGAPIC = true
		}

		// Apply repo-config API settings (may override bazel settings).
		if apiConfig, ok := apiConfigs[stateAPI.Path]; ok {
			if apiConfig.DisableGAPIC {
				api.DisableGAPIC = true
			}
			if api.Go == nil {
				api.Go = &config.GoPackage{}
			}
			if len(apiConfig.NestedProtos) > 0 {
				api.Go.NestedProtos = apiConfig.NestedProtos
			}
			if apiConfig.ProtoPackage != "" {
				api.Go.ProtoPackage = apiConfig.ProtoPackage
			}
		}

		// Clean up Go config if it only has derivable import_path.
		if api.Go != nil {
			// Clear import_path if it matches what would be derived.
			if api.Go.ImportPath != "" && api.Go.ImportPath == deriveGoGapicPackage(api.Path) {
				api.Go.ImportPath = ""
			}
			// Clear Go config entirely if empty.
			if api.Go.ImportPath == "" && !api.Go.LegacyGRPC &&
				len(api.Go.NestedProtos) == 0 && api.Go.ProtoPackage == "" {
				api.Go = nil
			}
		}

		lib.APIs = append(lib.APIs, api)
	}

	return lib
}

func readStateFile(path string) (*StateFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state StateFile
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func readConfigFile(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func readRepoConfigFile(path string) (*RepoConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config RepoConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

