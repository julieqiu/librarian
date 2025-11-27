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

// Package cli contains hardcoded configuration for the Librarian CLI.
package cli

// DocOverride specifies a documentation text replacement for a proto element.
type DocOverride struct {
	ID      string // fully qualified proto element ID
	Match   string // text to match
	Replace string // replacement text
}

// DocOverrides contains documentation fixes that apply across all languages.
var DocOverrides = []DocOverride{
	{
		ID:      ".google.cloud.developerconnect.v1.ArtifactConfig.google_artifact_registry",
		Match:   "regsitry",
		Replace: "registry",
	},
	{
		ID: ".google.cloud.dialogflow.v2.HumanAgentAssistantConfig.ConversationModelConfig.baseline_model_version",
		Match: `Version of current baseline model. It will be ignored if
[model][google.cloud.dialogflow.v2.HumanAgentAssistantConfig.ConversationModelConfig.model]
is set. Valid versions are:
  Article Suggestion baseline model:
    - 0.9
    - 1.0 (default)
  Summarization baseline model:
    - 1.0`,
		Replace: `Version of current baseline model. It will be ignored if
[model][google.cloud.dialogflow.v2.HumanAgentAssistantConfig.ConversationModelConfig.model]
is set. Valid versions are:
- Article Suggestion baseline model:
  - 0.9
  - 1.0 (default)
- Summarization baseline model:
  - 1.0`,
	},
	{
		ID:      ".google.cloud.networkservices.v1.WasmPlugin.LogConfig.min_log_level",
		Match:   "Specificies",
		Replace: "Specifies",
	},
	{
		ID:      ".google.cloud.retail.v2.SearchRequest.user_attributes",
		Match:   "Duplcate",
		Replace: "Duplicate",
	},
}
