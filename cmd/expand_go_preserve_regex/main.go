package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	testdataDir = "internal/container/go/testdata"
	goRepoDir   = "/Users/julieqiu/code/googleapis/google-cloud-go"
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
	Keep                        []string `yaml:"keep,omitempty"`
	RemoveRegex                 []string `yaml:"remove_regex,omitempty"`
	ReleaseExcludePaths         []string `yaml:"release_exclude_paths,omitempty"`
	TagFormat                   string   `yaml:"tag_format,omitempty"`
	ModulePathVersion           string   `yaml:"module_path_version,omitempty"`
	DeleteGenerationOutputPaths []string `yaml:"delete_generation_output_paths,omitempty"`
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

	if librarian.Go == nil {
		return nil
	}

	// Read the old preserve_regex field
	var oldConfig struct {
		Go struct {
			PreserveRegex []string `yaml:"preserve_regex"`
		} `yaml:"go"`
	}
	if err := yaml.Unmarshal(data, &oldConfig); err != nil {
		return fmt.Errorf("failed to unmarshal old config: %w", err)
	}

	if len(oldConfig.Go.PreserveRegex) == 0 {
		return nil
	}

	// Find all matching files in the Go repo
	var matchedFiles []string
	for _, pattern := range oldConfig.Go.PreserveRegex {
		files, err := findMatchingFiles(goRepoDir, pattern)
		if err != nil {
			log.Printf("Error finding files for pattern %s: %v", pattern, err)
			continue
		}
		matchedFiles = append(matchedFiles, files...)
	}

	// Update the Go config with explicit file list
	librarian.Go.Keep = matchedFiles

	// Write back
	out, err := yaml.Marshal(&librarian)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml: %w", err)
	}

	if err := os.WriteFile(librarianPath, out, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func findMatchingFiles(repoDir string, pattern string) ([]string, error) {
	// Compile the regex pattern
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex %s: %w", pattern, err)
	}

	var matches []string

	err = filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip hidden directories and common non-source directories
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path from repo root
		relPath, err := filepath.Rel(repoDir, path)
		if err != nil {
			return err
		}

		// Check if path matches the regex
		if re.MatchString(relPath) {
			matches = append(matches, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matches, nil
}
