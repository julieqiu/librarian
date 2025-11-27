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

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDiscover(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Discover("testdata/googleapis"); err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, lib := range cfg.Libraries {
		got = append(got, lib.Channel)
	}
	slices.Sort(got)

	want := []string{
		"google/cloud/speech/v1",
		"google/cloud/speech/v1p1beta1",
		"google/cloud/speech/v2",
		"grafeas/v1",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestServiceConfig(t *testing.T) {
	got, err := ServiceConfig("testdata/googleapis", "google/cloud/speech/v1")
	if err != nil {
		t.Fatal(err)
	}
	want := "google/cloud/speech/v1/speech_v1.yaml"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
