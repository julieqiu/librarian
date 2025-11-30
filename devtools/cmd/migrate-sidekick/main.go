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

// Command migrate-sidekick converts .sidekick.toml files to a librarian.yaml
// configuration file.
//
// Usage:
//
//	go run ./devtools/cmd/migrate-sidekick
//
// By default, this tool fetches the latest commit from
// github.com/googleapis/google-cloud-rust using the GitHub API, downloads
// the tarball to $LIBRARIAN_CACHE (or $HOME/.cache/librarian if not set),
// scans for .sidekick.toml files, and generates a librarian.yaml file.
//
// Flags:
//
//	-repo      Override the repository path (skips downloading)
//	-output    Output file path (default: librarian.yaml)
//	-language  Language for the repository (default: rust)
package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/librarian"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	defaultRepoOrg  = "googleapis"
	defaultRepoName = "google-cloud-rust"
	defaultBranch   = "main"
)

// knownServiceConfigs maps API paths to their service config files for cases
// where the service config is not in the same directory as the API.
var knownServiceConfigs = map[string]string{
	"google/cloud/aiplatform/v1/schema/predict/instance":       "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/v1/schema/predict/params":         "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/v1/schema/predict/prediction":     "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/v1/schema/trainingjob/definition": "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
}

func main() {
	repoPath := flag.String("repo", "", "path to the repository containing .sidekick.toml files (if empty, downloads from GitHub)")
	output := flag.String("output", "librarian.yaml", "output file path")
	language := flag.String("language", "rust", "language for the repository")
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

	cfg, err := migrate(path, *language)
	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	if err := yaml.Write(*output, cfg); err != nil {
		log.Fatalf("failed to write config: %v", err)
	}
	fmt.Printf("wrote %s\n", *output)

	// Format the output file to remove auto-discoverable fields and normalize.
	if err := librarian.Run(context.Background(), "librarian", "tidy"); err != nil {
		log.Fatalf("failed to format config: %v", err)
	}
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

// migrate scans the repository for .sidekick.toml files and converts them to
// a librarian.yaml configuration.
func migrate(repoPath, language string) (*config.Config, error) {
	rootConfigPath := filepath.Join(repoPath, ".sidekick.toml")
	rootConfig, err := sidekickconfig.LoadRootConfig(rootConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load root config: %w", err)
	}

	cfg := &config.Config{
		Language: language,
		Sources:  convertSources(rootConfig.Source),
		Default:  convertDefaultConfig(rootConfig),
	}

	// Scan for .sidekick.toml files in subdirectories.
	var sidekickFiles []string
	err = filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == ".sidekick.toml" && path != rootConfigPath {
			sidekickFiles = append(sidekickFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan repository: %w", err)
	}

	for _, sidekickPath := range sidekickFiles {
		// Only migrate files under src/generated, excluding test/validation directories.
		relPath, err := filepath.Rel(repoPath, sidekickPath)
		if err != nil {
			log.Printf("warning: failed to get relative path for %s: %v", sidekickPath, err)
			continue
		}
		if !strings.HasPrefix(relPath, "src/generated/") {
			continue
		}
		if strings.Contains(relPath, "openapi-validation") || strings.Contains(relPath, "testdata") || strings.Contains(relPath, "showcase") {
			continue
		}

		// Load the local config without merging, so we only get library-specific settings.
		localConfig, err := sidekickconfig.LoadRootConfig(sidekickPath)
		if err != nil {
			log.Printf("warning: failed to load %s: %v", sidekickPath, err)
			continue
		}

		lib, err := convertLibrary(repoPath, sidekickPath, localConfig, rootConfig)
		if err != nil {
			log.Printf("warning: failed to convert %s: %v", sidekickPath, err)
			continue
		}

		cfg.Libraries = append(cfg.Libraries, lib)
	}

	return cfg, nil
}

// convertSources converts sidekick source map to librarian Sources config.
func convertSources(source map[string]string) *config.Sources {
	if len(source) == 0 {
		return nil
	}

	sources := &config.Sources{}

	// The sidekick source map uses "googleapis-root" with a tarball URL.
	// Extract the commit SHA from the URL.
	if url, ok := source["googleapis-root"]; ok {
		if commit := extractCommitFromURL(url); commit != "" {
			sources.Googleapis = &config.Source{
				Commit: commit,
			}
			if sha256, ok := source["googleapis-sha256"]; ok {
				sources.Googleapis.SHA256 = sha256
			}
		}
	}

	if url, ok := source["discovery-root"]; ok {
		if commit := extractCommitFromURL(url); commit != "" {
			sources.Discovery = &config.Source{
				Commit: commit,
			}
			if sha256, ok := source["discovery-sha256"]; ok {
				sources.Discovery.SHA256 = sha256
			}
		}
	}

	return sources
}

// extractCommitFromURL extracts the commit SHA from a GitHub archive URL.
// Example: https://github.com/googleapis/googleapis/archive/abc123.tar.gz -> abc123.
func extractCommitFromURL(url string) string {
	// URL format: https://github.com/{org}/{repo}/archive/{commit}.tar.gz
	url = strings.TrimSuffix(url, ".tar.gz")
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// convertDefaultConfig converts the root sidekick config to default librarian config.
func convertDefaultConfig(sc *sidekickconfig.Config) *config.Default {
	d := &config.Default{
		Output: "src/generated",
		Branch: "main",
	}

	if sc.Release != nil {
		d.Remote = sc.Release.Remote
		d.Branch = sc.Release.Branch
	}

	// Convert codec options to Rust defaults.
	if len(sc.Codec) > 0 {
		d.Rust = convertRustDefaults(sc.Codec)
	}

	return d
}

// convertRustDefaults converts codec options to Rust default configuration.
func convertRustDefaults(codec map[string]string) *config.RustDefault {
	rd := &config.RustDefault{}

	// Convert package:* options to package dependencies.
	for key, value := range codec {
		if strings.HasPrefix(key, "package:") {
			dep := parsePackageDependency(key, value)
			if dep != nil {
				rd.PackageDependencies = append(rd.PackageDependencies, dep)
			}
		}
	}

	if warnings, ok := codec["disabled-rustdoc-warnings"]; ok && warnings != "" {
		rd.DisabledRustdocWarnings = strings.Split(warnings, ",")
	}

	return rd
}

// convertLibrary converts a single .sidekick.toml to a Library config.
// rootConfig is used to filter out settings that match the defaults.
func convertLibrary(repoPath, sidekickPath string, sc *sidekickconfig.Config, rootConfig *sidekickconfig.Config) (*config.Library, error) {
	// Determine output path relative to repo root.
	dir := filepath.Dir(sidekickPath)
	relPath, err := filepath.Rel(repoPath, dir)
	if err != nil {
		return nil, err
	}

	lib := &config.Library{
		APIs: []*config.API{},
	}

	// Read library name and publish from Cargo.toml.
	// Note: version is read from Cargo.toml during generation, not migrated here.
	cargoPath := filepath.Join(dir, "Cargo.toml")
	cargoPkg := readCargoPackage(cargoPath)
	lib.Name = cargoPkg.Name
	if cargoPkg.Publish != nil && !*cargoPkg.Publish {
		lib.SkipPublish = true
	}

	// Set specification format and source.
	if sc.General.SpecificationFormat == "disco" {
		lib.APIs = append(lib.APIs, &config.API{
			Path:   sc.General.SpecificationSource,
			Format: "discovery",
		})
		// Discovery APIs always need explicit output since they don't have a
		// channel.
		lib.Output = relPath
	} else {
		// Default to protobuf format.
		lib.APIs = append(lib.APIs, &config.API{
			Format: sc.General.SpecificationFormat,
		})
	}

	// For protobuf APIs, only set output if it can't be derived from the channel.
	// The Fill() method derives output as: default.output + channel (without "google/" prefix)
	// e.g., "src/generated" + "spanner/admin/instance/v1" -> "src/generated/spanner/admin/instance/v1"
	if lib.APIs[0].Format == "" {
		derivedChannel := strings.ReplaceAll(lib.Name, "-", "/")
		derivedOutput := filepath.Join("src/generated", strings.TrimPrefix(derivedChannel, "google/"))
		if relPath != derivedOutput {
			lib.Output = relPath
		}
	}

	// For protobuf APIs, only set channel if it differs from what can be derived from name.
	// The Fill() method will derive channel from name by replacing "-" with "/".
	if lib.APIs[0].Format == "" && sc.General.SpecificationSource != "" {
		derivedChannel := strings.ReplaceAll(lib.Name, "-", "/")
		if sc.General.SpecificationSource != derivedChannel {
			lib.APIs[0].Path = sc.General.SpecificationSource
		}
	}

	// Set service config from sidekick config or known mappings.
	if sc.General.ServiceConfig != "" {
		lib.APIs[0].ServiceConfig = sc.General.ServiceConfig
	} else if apiPath := lib.APIs[0].Path; apiPath != "" {
		if knownSC, ok := knownServiceConfigs[apiPath]; ok {
			lib.APIs[0].ServiceConfig = knownSC
		}
	}

	// Convert codec and source options to Rust config, filtering out defaults.
	if len(sc.Codec) > 0 || len(sc.Source) > 0 {
		lib.Rust = convertRustCrate(sc.Codec, sc.Source, rootConfig.Codec)
	}

	// Convert documentation overrides.
	for _, override := range sc.CommentOverrides {
		if lib.Rust == nil {
			lib.Rust = &config.RustCrate{}
		}
		lib.Rust.DocumentationOverrides = append(lib.Rust.DocumentationOverrides, config.RustDocumentationOverride{
			ID:      override.ID,
			Match:   override.Match,
			Replace: override.Replace,
		})
	}

	// Convert pagination overrides.
	if len(sc.PaginationOverrides) > 0 && lib.Rust == nil {
		lib.Rust = &config.RustCrate{}
	}
	if lib.Rust != nil {
		for _, override := range sc.PaginationOverrides {
			lib.Rust.PaginationOverrides = append(lib.Rust.PaginationOverrides, config.RustPaginationOverride{
				ID:        override.ID,
				ItemField: override.ItemField,
			})
		}
	}

	// Convert discovery config for LRO polling.
	if sc.Discovery != nil && (sc.Discovery.OperationID != "" || len(sc.Discovery.Pollers) > 0) {
		if lib.Rust == nil {
			lib.Rust = &config.RustCrate{}
		}
		lib.Rust.Discovery = &config.RustDiscovery{
			OperationID: sc.Discovery.OperationID,
		}
		for _, poller := range sc.Discovery.Pollers {
			lib.Rust.Discovery.Pollers = append(lib.Rust.Discovery.Pollers, config.RustPoller{
				Prefix:   poller.Prefix,
				MethodID: poller.MethodID,
			})
		}
	}

	// Note: version and copyright-year are not migrated because the generate
	// command reads them directly from the existing Cargo.toml file, avoiding
	// duplication in config.

	// Set release level on the API (Rust has one API per library).
	if level, ok := sc.Codec["release-level"]; ok && len(lib.APIs) > 0 {
		lib.APIs[0].ReleaseLevel = level
	}

	// Convert extra-modules to both keep (preserve files) and extra_modules
	// (include module declarations in lib.rs).
	if v, ok := sc.Codec["extra-modules"]; ok && v != "" {
		modules := strings.Split(v, ",")
		for _, mod := range modules {
			lib.Keep = append(lib.Keep, "src/"+mod+".rs")
		}
		if lib.Rust == nil {
			lib.Rust = &config.RustCrate{}
		}
		lib.Rust.ExtraModules = modules
	}

	// Check for manually-added files that should be preserved.
	// main.rs is not generated but may exist for placeholder crates.
	mainPath := filepath.Join(dir, "src", "main.rs")
	if _, err := os.Stat(mainPath); err == nil {
		lib.Keep = append(lib.Keep, "src/main.rs")
	}

	return lib, nil
}

// convertRustCrate converts codec and source options to Rust crate configuration.
// rootCodec is used to filter out settings that match the defaults.
func convertRustCrate(codec, source, rootCodec map[string]string) *config.RustCrate {
	rc := &config.RustCrate{}
	hasContent := false

	// Note: package-name-override is used for the library name, not stored in Rust config.

	// Handle source options.
	if v, ok := source["title-override"]; ok && v != "" {
		rc.TitleOverride = v
		hasContent = true
	}
	if v, ok := source["description-override"]; ok && v != "" {
		rc.DescriptionOverride = v
		hasContent = true
	}
	if v, ok := source["skipped-ids"]; ok && v != "" {
		// skipped-ids is a comma-separated list; split into individual IDs.
		rc.SkippedIds = strings.Split(v, ",")
		hasContent = true
	}

	if v, ok := codec["module-path"]; ok && v != "" && v != "crate::model" {
		rc.ModulePath = v
		hasContent = true
	}

	if v, ok := codec["template-override"]; ok && v != "" {
		rc.TemplateOverride = v
		hasContent = true
	}

	if v, ok := codec["per-service-features"]; ok && v == "true" {
		rc.PerServiceFeatures = true
		hasContent = true
	}

	if v, ok := codec["has-veneer"]; ok && v == "true" {
		rc.HasVeneer = true
		hasContent = true
	}

	if v, ok := codec["routing-required"]; ok && v == "true" {
		rc.RoutingRequired = true
		hasContent = true
	}

	if v, ok := codec["include-grpc-only-methods"]; ok && v == "true" {
		rc.IncludeGrpcOnlyMethods = true
		hasContent = true
	}

	if v, ok := codec["generate-setter-samples"]; ok && v == "true" {
		rc.GenerateSetterSamples = true
		hasContent = true
	}

	if v, ok := codec["detailed-tracing-attributes"]; ok && v == "true" {
		rc.DetailedTracingAttributes = true
		hasContent = true
	}

	if v, ok := codec["default-features"]; ok && v != "" {
		rc.DefaultFeatures = strings.Split(v, ",")
		hasContent = true
	}

	if v, ok := codec["disabled-rustdoc-warnings"]; ok && v != "" {
		rc.DisabledRustdocWarnings = strings.Split(v, ",")
		hasContent = true
	}

	if v, ok := codec["disabled-clippy-warnings"]; ok && v != "" {
		rc.DisabledClippyWarnings = strings.Split(v, ",")
		hasContent = true
	}

	if v, ok := codec["name-overrides"]; ok && v != "" {
		rc.NameOverrides = v
		hasContent = true
	}

	// Convert package:* options to package dependencies, but only include
	// ones that differ from the root config defaults.
	for key, value := range codec {
		if strings.HasPrefix(key, "package:") {
			// Skip if this package dependency is the same as in root config.
			if rootValue, ok := rootCodec[key]; ok && rootValue == value {
				continue
			}
			dep := parsePackageDependency(key, value)
			if dep != nil {
				rc.PackageDependencies = append(rc.PackageDependencies, dep)
				hasContent = true
			}
		}
	}

	if !hasContent {
		return nil
	}
	return rc
}

// parsePackageDependency parses a package:* codec option into a RustPackageDependency.
func parsePackageDependency(key, value string) *config.RustPackageDependency {
	name := strings.TrimPrefix(key, "package:")
	dep := &config.RustPackageDependency{
		Name: name,
	}

	for _, part := range strings.Split(value, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := kv[0], kv[1]
		switch k {
		case "package":
			dep.Package = v
		case "source":
			dep.Source = v
		case "feature":
			dep.Feature = v
		case "force-used":
			dep.ForceUsed = v == "true"
		case "used-if":
			dep.UsedIf = v
		case "ignore":
			dep.Ignore = v == "true"
		}
	}

	return dep
}

// cargoPackage contains package info read from Cargo.toml.
type cargoPackage struct {
	Name    string
	Version string
	Publish *bool // nil means default (true), false means publish = false
}

// readCargoPackage reads the package name and version from a Cargo.toml file.
func readCargoPackage(cargoPath string) cargoPackage {
	data, err := os.ReadFile(cargoPath)
	if err != nil {
		return cargoPackage{}
	}
	var pkg cargoPackage
	// Simple parsing: look for name/version = "..." lines in [package] section.
	inPackage := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "[package]" {
			inPackage = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inPackage = false
			continue
		}
		if inPackage {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, `"`)
			switch key {
			case "name":
				pkg.Name = value
			case "version":
				pkg.Version = value
			case "publish":
				if value == "false" {
					f := false
					pkg.Publish = &f
				}
			}
		}
	}
	return pkg
}
