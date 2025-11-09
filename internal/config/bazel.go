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

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// ParseBUILDFile reads a BUILD.bazel file and extracts API configuration for the given language.
// Supported languages: go, python, rust, dart, java.
func ParseBUILDFile(googleapisRoot, apiPath, language string) (*APIConfig, error) {
	buildPath := filepath.Join(googleapisRoot, apiPath, "BUILD.bazel")
	data, err := os.ReadFile(buildPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read BUILD.bazel at %s: %w", buildPath, err)
	}

	content := string(data)

	// Determine the gapic rule name based on language
	ruleName := getGapicRuleName(language)
	if ruleName == "" {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// Find the gapic library block
	blockRE := regexp.MustCompile(fmt.Sprintf(`%s\((?s:.)*?\)`, regexp.QuoteMeta(ruleName)))
	block := blockRE.FindString(content)
	if block == "" {
		return nil, fmt.Errorf("no %s rule found in BUILD.bazel", ruleName)
	}

	// Extract configuration from the block
	config := &APIConfig{
		Path: apiPath,
	}

	config.GRPCServiceConfig = findStringAttr(block, "grpc_service_config")
	config.ServiceYAML = strings.TrimPrefix(findStringAttr(block, "service_yaml"), ":")
	config.Transport = findStringAttr(block, "transport")

	restNumericEnums, err := findBoolAttr(block, "rest_numeric_enums")
	if err != nil {
		return nil, fmt.Errorf("failed to parse rest_numeric_enums: %w", err)
	}
	config.RestNumericEnums = restNumericEnums

	// Extract opt_args (language-specific options)
	config.OptArgs = findListAttr(block, "opt_args")

	return config, nil
}

// getGapicRuleName returns the Bazel rule name for the given language's gapic library.
func getGapicRuleName(language string) string {
	rules := map[string]string{
		"go":     "go_gapic_library",
		"python": "py_gapic_library",
		"rust":   "rust_gapic_library",
		"dart":   "dart_gapic_library",
		"java":   "java_gapic_library",
	}
	return rules[language]
}

var reCache = &sync.Map{}

func getRegexp(key, pattern string) *regexp.Regexp {
	val, ok := reCache.Load(key)
	if !ok {
		val = regexp.MustCompile(pattern)
		reCache.Store(key, val)
	}
	return val.(*regexp.Regexp)
}

// findStringAttr extracts a string attribute from a Bazel rule block.
func findStringAttr(content, name string) string {
	re := getRegexp("findString_"+name, fmt.Sprintf(`%s\s*=\s*(?:"([^"]+)"|'([^']+)')`, regexp.QuoteMeta(name)))
	match := re.FindStringSubmatch(content)
	if len(match) > 2 {
		if match[1] != "" {
			return match[1] // Double-quoted
		}
		return match[2] // Single-quoted
	}
	return ""
}

// findBoolAttr extracts a boolean attribute from a Bazel rule block.
func findBoolAttr(content, name string) (bool, error) {
	re := getRegexp("findBool_"+name, fmt.Sprintf(`%s\s*=\s*(\w+)`, regexp.QuoteMeta(name)))
	match := re.FindStringSubmatch(content)
	if len(match) > 1 {
		b, err := strconv.ParseBool(match[1])
		if err != nil {
			return false, fmt.Errorf("invalid boolean value %q for %s", match[1], name)
		}
		return b, nil
	}
	return false, nil // Default to false if not found
}

// findListAttr extracts a list attribute from a Bazel rule block.
func findListAttr(content, name string) []string {
	// Match list like: opt_args = ["arg1", "arg2"]
	re := getRegexp("findList_"+name, fmt.Sprintf(`%s\s*=\s*\[((?:\s*"[^"]*"\s*,?)*)\s*\]`, regexp.QuoteMeta(name)))
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return nil
	}

	listContent := match[1]
	if listContent == "" {
		return nil
	}

	// Extract individual quoted strings
	itemRE := regexp.MustCompile(`"([^"]*)"`)
	matches := itemRE.FindAllStringSubmatch(listContent, -1)

	var result []string
	for _, m := range matches {
		if len(m) > 1 {
			result = append(result, m[1])
		}
	}

	return result
}
