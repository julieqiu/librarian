package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	testdataDir   = "internal/container/go/testdata"
	googleapisDir = "/Users/julieqiu/code/googleapis/googleapis"
)

type Librarian struct {
	Name     string            `yaml:"name"`
	Version  string            `yaml:"version"`
	Generate LibrarianGenerate `yaml:"generate"`
	Go       *GoConfig         `yaml:"go,omitempty"`
}

type LibrarianGenerate struct {
	SpecificationFormat string `yaml:"specification_format,omitempty"`
	APIs                []API  `yaml:"apis,omitempty"`
}

type API struct {
	Path              string   `yaml:"path"`
	ServiceConfig     string   `yaml:"service_config,omitempty"`
	ClientDirectory   string   `yaml:"client_directory,omitempty"`
	DisableGapic      bool     `yaml:"disable_gapic,omitempty"`
	NestedProtos      []string `yaml:"nested_protos,omitempty"`
	ProtoPackage      string   `yaml:"proto_package,omitempty"`
	GRPCServiceConfig string   `yaml:"grpc_service_config,omitempty"`
	RestNumericEnums  *bool    `yaml:"rest_numeric_enums,omitempty"`
	Transport         string   `yaml:"transport,omitempty"`
	Importpath        string   `yaml:"importpath,omitempty"`
	ReleaseLevel      string   `yaml:"release_level,omitempty"`
}

type GoConfig struct {
	SourceRoots                 []string `yaml:"source_roots,omitempty"`
	PreserveRegex               []string `yaml:"preserve_regex,omitempty"`
	RemoveRegex                 []string `yaml:"remove_regex,omitempty"`
	ReleaseExcludePaths         []string `yaml:"release_exclude_paths,omitempty"`
	TagFormat                   string   `yaml:"tag_format,omitempty"`
	ModulePathVersion           string   `yaml:"module_path_version,omitempty"`
	DeleteGenerationOutputPaths []string `yaml:"delete_generation_output_paths,omitempty"`
}

type GoGapicLibrary struct {
	GRPCServiceConfig string `yaml:"grpc_service_config,omitempty"`
	RestNumericEnums  *bool  `yaml:"rest_numeric_enums,omitempty"`
	Transport         string `yaml:"transport,omitempty"`
	Importpath        string `yaml:"importpath,omitempty"`
	ReleaseLevel      string `yaml:"release_level,omitempty"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		return fmt.Errorf("failed to read testdata directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		librarianPath := filepath.Join(testdataDir, entry.Name(), ".librarian.yaml")
		if err := processLibrarian(librarianPath); err != nil {
			log.Printf("Error processing %s: %v", entry.Name(), err)
			continue
		}
		log.Printf("Processed %s", entry.Name())
	}

	return nil
}

func processLibrarian(librarianPath string) error {
	// Read the .librarian.yaml file
	data, err := os.ReadFile(librarianPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var librarian Librarian
	if err := yaml.Unmarshal(data, &librarian); err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// For each API, try to read BUILD.bazel and extract go_gapic_library data
	modified := false
	for i := range librarian.Generate.APIs {
		api := &librarian.Generate.APIs[i]

		// Skip if GAPIC is disabled
		if api.DisableGapic {
			continue
		}

		buildPath := filepath.Join(googleapisDir, api.Path, "BUILD.bazel")

		goGapicData, err := parseBuildBazel(buildPath)
		if err != nil {
			// If BUILD.bazel doesn't exist or can't be parsed, skip
			continue
		}

		// Add go_gapic_library data directly to the API
		if goGapicData != nil {
			api.GRPCServiceConfig = goGapicData.GRPCServiceConfig
			api.RestNumericEnums = goGapicData.RestNumericEnums
			api.Transport = goGapicData.Transport
			api.Importpath = goGapicData.Importpath
			api.ReleaseLevel = goGapicData.ReleaseLevel
			modified = true
		}
	}

	// Write back if modified
	if modified {
		out, err := yaml.Marshal(&librarian)
		if err != nil {
			return fmt.Errorf("failed to marshal yaml: %w", err)
		}

		if err := os.WriteFile(librarianPath, out, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}

func isEmpty(b *GoGapicLibrary) bool {
	return b.GRPCServiceConfig == "" &&
		b.RestNumericEnums == nil &&
		b.Transport == "" &&
		b.Importpath == "" &&
		b.ReleaseLevel == ""
}

// parseBuildBazel parses a BUILD.bazel file and extracts go_gapic_library configuration.
func parseBuildBazel(buildPath string) (*GoGapicLibrary, error) {
	content, err := os.ReadFile(buildPath)
	if err != nil {
		return nil, err
	}

	text := string(content)

	// Look for go_gapic_library rule (not go_gapic_assembly_pkg)
	if !strings.Contains(text, "_go_gapic") {
		return nil, nil // No go_gapic_library found
	}

	// Check if this is actually a go_gapic_library (not just assembly_pkg)
	lines := strings.Split(text, "\n")
	inGoGapic := false
	var goGapic GoGapicLibrary

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Start of go_gapic_library block
		if strings.HasPrefix(trimmed, "go_gapic_library(") ||
			(strings.Contains(trimmed, "_go_gapic") && strings.Contains(trimmed, "=") && !strings.Contains(trimmed, "go_gapic_assembly_pkg")) {
			inGoGapic = true
			continue
		}

		if !inGoGapic {
			continue
		}

		// End of rule
		if strings.HasPrefix(trimmed, ")") {
			break
		}

		// Parse attributes
		if strings.Contains(trimmed, "grpc_service_config") {
			val := extractValue(trimmed)
			goGapic.GRPCServiceConfig = val
		} else if strings.Contains(trimmed, "rest_numeric_enums") {
			val := extractValue(trimmed)
			boolVal := val == "True"
			goGapic.RestNumericEnums = &boolVal
		} else if strings.Contains(trimmed, "transport") && !strings.Contains(trimmed, "//") {
			val := extractValue(trimmed)
			goGapic.Transport = val
		} else if strings.Contains(trimmed, "importpath") && !strings.Contains(trimmed, "//") {
			val := extractValue(trimmed)
			goGapic.Importpath = val
		} else if strings.Contains(trimmed, "release_level") && !strings.Contains(trimmed, "//") {
			val := extractValue(trimmed)
			goGapic.ReleaseLevel = val
		}
	}

	if isEmpty(&goGapic) {
		return nil, nil // Found go_gapic but no attributes
	}

	return &goGapic, nil
}

// extractValue extracts the string value from a line like 'key = "value",' or 'key = value,'
func extractValue(line string) string {
	// Remove leading/trailing whitespace
	line = strings.TrimSpace(line)

	// Split by '='
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		// Handle simple quoted string case
		line = strings.Trim(line, ",")
		line = strings.Trim(line, "\"'")
		return line
	}

	val := strings.TrimSpace(parts[1])
	// Remove trailing comma
	val = strings.TrimSuffix(val, ",")
	// Remove quotes
	val = strings.Trim(val, "\"'")

	return val
}
