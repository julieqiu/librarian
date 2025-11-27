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

package cli

// ServiceConfigs maps channel paths to their service config file paths.
// This map only contains exceptional cases where the service config is NOT
// in the same directory as the channel. Most service configs are auto-discovered
// by scanning the channel directory for YAML files containing "type: google.api.Service".
var ServiceConfigs = map[string]string{
	// AI Platform schema libraries have service config in parent directory.
	"google/cloud/aiplatform/v1/schema/predict/instance":       "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/v1/schema/predict/params":         "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/v1/schema/predict/prediction":     "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/v1/schema/trainingjob/definition": "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
}
