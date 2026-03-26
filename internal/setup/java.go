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

package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/fetch"
)

const (
	jarGAPICJava  = "gapic-generator-java"
	jarGRPCJava   = "protoc-gen-grpc-java"
	jarJavaFormat = "google-java-format"

	hostMavenCentral = "https://repo1.maven.org/maven2"

	grpcJavaURL       = hostMavenCentral + "/io/grpc/protoc-gen-grpc-java/%s/protoc-gen-grpc-java-%s-linux-x86_64.exe"
	javaFormatURL     = hostMavenCentral + "/com/google/googlejavaformat/google-java-format/%s/%s"
	javaFormatJAR     = "google-java-format-%s-all-deps.jar"
	gapicJavaURL = hostMavenCentral + "/com/google/api/gapic-generator-java/%s/%s"
	gapicGeneratorJavaJAR = "gapic-generator-java-%s.jar"

	javaJARScript  = "#!/bin/bash\nexec java -jar %q \"$@\"\n"
	javaMainScript = "#!/bin/bash\nexec java -cp %q com.google.api.generator.Main \"$@\"\n"
)

func installJavaTool(ctx context.Context, tool ToolVersion) error {
	switch tool.Name {
	case jarGAPICJava:
		return installGapicGeneratorJava(ctx, tool)
	case jarJavaFormat:
		return installGoogleJavaFormat(ctx, tool)
	case jarGRPCJava:
		return installProtocGenGRPCJava(ctx, tool)
	default:
		return fmt.Errorf("unknown java tool: %s", tool.Name)
	}
}

func installProtocGenGRPCJava(ctx context.Context, tool ToolVersion) error {
	binDir, err := javaToolsBinDir()
	if err != nil {
		return err
	}
	url := fmt.Sprintf(grpcJavaURL, tool.Version, tool.Version)
	dest := filepath.Join(binDir, jarGRPCJava)
	if err := fetch.Download(ctx, dest, url, ""); err != nil {
		return err
	}
	return os.Chmod(dest, 0o755)
}

func installGoogleJavaFormat(ctx context.Context, tool ToolVersion) error {
	binDir, err := javaToolsBinDir()
	if err != nil {
		return err
	}
	jarName := fmt.Sprintf(javaFormatJAR, tool.Version)
	url := fmt.Sprintf(javaFormatURL, tool.Version, jarName)
	jarPath := filepath.Join(binDir, jarName)
	if err := fetch.Download(ctx, jarPath, url, ""); err != nil {
		return err
	}

	wrapper := filepath.Join(binDir, jarJavaFormat)
	script := fmt.Sprintf(javaJARScript, jarPath)
	return os.WriteFile(wrapper, []byte(script), 0o755)
}

func installGapicGeneratorJava(ctx context.Context, tool ToolVersion) error {
	binDir, err := javaToolsBinDir()
	if err != nil {
		return err
	}
	jarName := fmt.Sprintf(gapicGeneratorJavaJAR, tool.Version)
	url := fmt.Sprintf(gapicJavaURL, tool.Version, jarName)
	jarPath := filepath.Join(binDir, jarName)
	if err := fetch.Download(ctx, jarPath, url, ""); err != nil {
		return err
	}

	wrapper := filepath.Join(binDir, "gapic-generator-java")
	script := fmt.Sprintf(javaMainScript, jarPath)
	return os.WriteFile(wrapper, []byte(script), 0o755)
}

func javaToolsBinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	dir := filepath.Join(home, "java_tools", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating java tools directory: %w", err)
	}
	return dir, nil
}
