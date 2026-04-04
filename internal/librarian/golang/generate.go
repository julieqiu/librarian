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

// Package golang provides functionality for generating Go client libraries.
package golang

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/snippetmetadata"
	"github.com/googleapis/librarian/internal/sources"
)

var (
	//go:embed template/_README.md.txt
	readmeTmpl       string
	readmeTmplParsed = template.Must(template.New("readme").Parse(readmeTmpl))
)

// Generate generates a Go client library.
func Generate(ctx context.Context, library *config.Library, srcs *sources.Sources) error {
	googleapisDir := srcs.Googleapis
	outDir, err := filepath.Abs(library.Output)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// Generate protoc output into a temp directory so it does not pollute the
	// library output directory with intermediate artifacts.
	tmpDir, err := os.MkdirTemp("", "librarian-generate-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	for _, api := range library.APIs {
		goAPI := findGoAPI(library, api.Path)
		if goAPI == nil {
			return fmt.Errorf("error finding goAPI associated with API %s: %w", api.Path, errGoAPINotFound)
		}
		sc, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageGo)
		if err != nil {
			return err
		}
		if err := generateAPI(ctx, goAPI, sc, googleapisDir, tmpDir); err != nil {
			return fmt.Errorf("api %q: %w", api.Path, err)
		}
		if err := moveGeneratedFiles(library, goAPI, tmpDir, outDir); err != nil {
			return err
		}
		if err := generateClientVersionFile(library, goAPI); err != nil {
			return err
		}
		if err := generateRepoMetadata(sc, library, goAPI); err != nil {
			return err
		}
	}

	// Delete paths configured for removal after generation.
	if library.Go != nil {
		for _, p := range library.Go.DeleteGenerationOutputPaths {
			if err := os.RemoveAll(filepath.Join(outDir, p)); err != nil {
				return err
			}
		}
	}
	if err := generateInternalVersionFile(outDir, library.CopyrightYear, library.Version); err != nil {
		return err
	}
	// Generate README using the first API's service config.
	if len(library.APIs) > 0 {
		sc, err := serviceconfig.Find(googleapisDir, library.APIs[0].Path, config.LanguageGo)
		if err != nil {
			return err
		}
		if err := generateREADME(library, sc, outDir); err != nil {
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(outDir, "go.mod")); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// New client, init the module.
			return initModule(ctx, outDir, modulePath(library))
		}
		return err
	}
	return nil
}

func generateAPI(ctx context.Context, goAPI *config.GoAPI, sc *serviceconfig.API, googleapisDir, tmpDir string) error {
	nestedProtos := goAPI.NestedProtos
	args := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
		"--go_out=" + tmpDir,
		"-I=" + googleapisDir,
		"--go-grpc_out=" + tmpDir,
		"--go-grpc_opt=require_unimplemented_servers=false",
	}
	if !goAPI.ProtoOnly {
		gapicOpts, err := buildGAPICOpts(goAPI, sc, googleapisDir)
		if err != nil {
			return err
		}
		args = append(args, "--go_gapic_out="+tmpDir)
		for _, opt := range gapicOpts {
			args = append(args, "--go_gapic_opt="+opt)
		}
	}

	protoFiles, err := collectProtoFiles(googleapisDir, goAPI.Path, nestedProtos)
	if err != nil {
		return err
	}
	args = append(args, protoFiles...)
	return command.Run(ctx, args[0], args[1:]...)
}

func buildGAPICOpts(goAPI *config.GoAPI, sc *serviceconfig.API, googleapisDir string) ([]string, error) {
	gc, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, goAPI.Path)
	if err != nil {
		return nil, err
	}

	opts := []string{"go-gapic-package=" + buildGAPICImportPath(goAPI)}
	if !goAPI.NoMetadata {
		opts = append(opts, "metadata")
	}
	if goAPI.NoSnippets {
		opts = append(opts, "omit-snippets")
	}
	if sc != nil && sc.HasRESTNumericEnums(config.LanguageGo) {
		opts = append(opts, "rest-numeric-enums")
	}
	if goAPI.DIREGAPIC {
		opts = append(opts, "diregapic")
	}
	if goAPI.EnabledGeneratorFeatures != nil {
		opts = append(opts, goAPI.EnabledGeneratorFeatures...)
	}
	if sc != nil {
		opts = append(opts, "api-service-config="+filepath.Join(googleapisDir, sc.ServiceConfig))
	}
	if gc != "" {
		opts = append(opts, "grpc-service-config="+filepath.Join(googleapisDir, gc))
	}
	// TODO(https://github.com/googleapis/librarian/issues/3775): assuming
	// transport is library-wide for now, until we have figured out the config
	// for transports.
	if trans := transport(sc); trans != "" {
		opts = append(opts, fmt.Sprintf("transport=%s", trans))
	}
	opts = append(opts, "release-level="+sc.ReleaseLevel(config.LanguageGo))
	return opts, nil
}

func buildGAPICImportPath(goAPI *config.GoAPI) string {
	return fmt.Sprintf("cloud.google.com/go/%s;%s",
		goAPI.ImportPath, goAPI.ClientPackage)
}

// moveGeneratedFiles moves generated API and snippet files from the protoc
// output directory (srcDir) to their final destination derived from outDir.
func moveGeneratedFiles(library *config.Library, goAPI *config.GoAPI, srcDir, outDir string) error {
	if err := moveAPIDirectory(library, goAPI, srcDir, outDir); err != nil {
		return err
	}
	return moveAndUpdateSnippets(library, goAPI, srcDir, outDir)
}

// moveAPIDirectory moves the generated API directory from srcDir to its final
// destination in the repository.
func moveAPIDirectory(library *config.Library, goAPI *config.GoAPI, srcDir, outDir string) error {
	librarySrc := filepath.Join(srcDir, "cloud.google.com", "go", goAPI.ImportPath)
	libraryDest := filepath.Join(repoRootPath(outDir, library.Name), clientPathFromRepoRoot(library, goAPI))
	if err := os.MkdirAll(libraryDest, 0755); err != nil {
		return err
	}
	return filesystem.MoveAndMerge(librarySrc, libraryDest)
}

// moveAndUpdateSnippets moves the generated snippets from srcDir to their final
// destination and updates their library versions.
func moveAndUpdateSnippets(library *config.Library, goAPI *config.GoAPI, srcDir, outDir string) error {
	snippetDest := findSnippetDirectory(library, goAPI, outDir)
	if snippetDest == "" {
		return nil
	}
	if err := os.MkdirAll(snippetDest, 0755); err != nil {
		return err
	}
	snippetSrc := filepath.Join(srcDir, "cloud.google.com", "go", "internal", "generated", "snippets", goAPI.ImportPath)
	if err := filesystem.MoveAndMerge(snippetSrc, snippetDest); err != nil {
		return err
	}
	return snippetmetadata.UpdateAllLibraryVersions(snippetDest, library.Version)
}

func collectProtoFiles(googleapisDir, apiPath string, nestedProtos []string) ([]string, error) {
	apiDir := filepath.Join(googleapisDir, apiPath)
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read API directory %s: %w", apiDir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".proto" {
			files = append(files, filepath.Join(apiDir, entry.Name()))
		}
	}

	for _, nested := range nestedProtos {
		files = append(files, filepath.Join(apiDir, nested))
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no .proto files found in %s", apiDir)
	}
	return files, nil
}

func generateREADME(library *config.Library, api *serviceconfig.API, moduleRoot string) error {
	readmePath := filepath.Join(moduleRoot, "README.md")
	// Skip generating README if it's in the keep list.
	// Handwritten/veneer libraries should have the top-level README in the keep list.
	// TODO(https://github.com/googleapis/librarian/issues/4113): investigate the difference between
	// GAPIC and handwritten libraries.
	for _, k := range library.Keep {
		path := filepath.Join(moduleRoot, k)
		if path == readmePath {
			return nil
		}
	}
	f, err := os.Create(readmePath)
	if err != nil {
		return err
	}
	err = readmeTmplParsed.Execute(f, map[string]string{
		"Name":       api.Title,
		"ModulePath": modulePath(library),
	})
	cerr := f.Close()
	if err != nil {
		return err
	}
	return cerr
}

// transport get transport from serviceconfig.API for language Go.
//
// The default value is serviceconfig.GRPCRest.
func transport(sc *serviceconfig.API) serviceconfig.Transport {
	if sc != nil {
		return sc.Transport(config.LanguageGo)
	}
	return serviceconfig.GRPCRest
}
