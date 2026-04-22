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

package gcloud

import (
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
)

func TestGenerate_RequiresSingleAPI(t *testing.T) {
	library := &config.Library{
		Gcloud: &config.GcloudCommand{},
	}
	if err := Generate(t.Context(), library, &sources.Sources{}); err == nil {
		t.Error("Generate() error = nil, want error for zero APIs")
	}
}

func TestGenerate_InvalidModel(t *testing.T) {
	library := &config.Library{
		APIs:   []*config.API{{Path: "google/cloud/parallelstore/v1"}},
		Gcloud: &config.GcloudCommand{},
	}
	src := &sources.Sources{Googleapis: "nonexistent_googleapis_dir"}
	if err := Generate(t.Context(), library, src); err == nil {
		t.Error("Generate() error = nil, want error for nonexistent googleapis dir")
	}
}
