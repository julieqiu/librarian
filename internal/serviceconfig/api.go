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

package serviceconfig

import (
	"path/filepath"
	"strings"
)

// API holds configuration for a specific API that was previously stored
// in librarian.yaml but can be derived or looked up from overrides.
type API struct {
	// RESTNumericEnums controls whether the REST client supports numeric enums.
	// Default: true
	RESTNumericEnums bool

	// Metadata controls whether metadata (e.g., gapic_metadata.json) is generated.
	// Default: true
	Metadata bool

	// Transport is the transport protocol for this API: "grpc", "rest", or "grpc+rest".
	// Default: "grpc+rest"
	Transport string

	// ReleaseLevel is the release level for this API: "alpha", "beta", or "ga".
	// Default: derived from API path version suffix
	ReleaseLevel string

	// DIREGAPIC enables DIREGAPIC (Discovery REST GAPICs) for compute/GCE.
	// Default: false
	DIREGAPIC bool

	// GRPCServiceConfig is the filename of the gRPC service config JSON file.
	// This is the filename only, not the full path.
	GRPCServiceConfig string

	// ServiceConfig is the full path to the service config YAML file.
	ServiceConfig string
}

// transport maps API paths to their transport when it differs from the
// default "grpc+rest".
var transport = map[string]string{
	// gRPC only
	"google/bigtable/admin/v2":              "grpc",
	"google/bigtable/v2":                    "grpc",
	"google/cloud/aiplatform/v1":            "grpc",
	"google/cloud/bigquery/migration/v2":    "grpc",
	"google/cloud/bigquery/storage/v1":      "grpc",
	"google/cloud/bigquery/storage/v1alpha": "grpc",
	"google/cloud/bigquery/storage/v1beta":  "grpc",
	"google/cloud/clouddms/v1":              "grpc",
	"google/cloud/gkemulticloud/v1":         "grpc",
	"google/cloud/managedidentities/v1":     "grpc",
	"google/cloud/mediatranslation/v1beta1": "grpc",
	"google/cloud/networkconnectivity/v1":   "grpc",
	"google/cloud/notebooks/v1":             "grpc",
	"google/cloud/pubsublite/v1":            "grpc",
	"google/cloud/recaptchaenterprise/v1":   "grpc",
	"google/cloud/tpu/v1":                   "grpc",
	"google/cloud/video/stitcher/v1":        "grpc",
	"google/cloud/workflows/executions/v1":  "grpc",
	"google/maps/fleetengine/v1":            "grpc",
	"google/monitoring/metricsscope/v1":     "grpc",
	"google/monitoring/v3":                  "grpc",
	"google/spanner/executor/v1":            "grpc",
	"google/storage/v2":                     "grpc",
	// REST only
	"google/cloud/apihub/v1":                  "rest",
	"google/cloud/compute/v1":                 "rest",
	"google/cloud/compute/v1beta":             "rest",
	"google/cloud/gkeconnect/gateway/v1":      "rest",
	"google/cloud/gkeconnect/gateway/v1beta1": "rest",
	"google/cloud/memorystore/v1":             "rest",
	"google/cloud/memorystore/v1beta":         "rest",
}

// diregapic maps API paths that use DIREGAPIC (Discovery REST GAPICs).
var diregapic = map[string]bool{
	"google/cloud/compute/v1":     true,
	"google/cloud/compute/v1beta": true,
}

// releaseLevel maps API paths to their release level when it differs from
// what would be derived from the path's version suffix.
var releaseLevel = map[string]string{
	"google/ai/generativelanguage/v1":                "beta",
	"google/ai/generativelanguage/v1alpha":           "beta",
	"google/cloud/alloydb/v1alpha":                   "beta",
	"google/cloud/apihub/v1":                         "beta",
	"google/cloud/apphub/v1":                         "beta",
	"google/apps/events/subscriptions/v1":            "beta",
	"google/apps/meet/v2":                            "beta",
	"google/cloud/bigquery/biglake/v1alpha1":         "beta",
	"google/cloud/bigquery/datapolicies/v2":          "beta",
	"google/cloud/bigquery/storage/v1alpha":          "beta",
	"google/chat/v1":                                 "beta",
	"google/cloud/chronicle/v1":                      "beta",
	"google/devtools/cloudprofiler/v2":               "beta",
	"google/cloud/confidentialcomputing/v1alpha1":    "beta",
	"google/cloud/configdelivery/v1":                 "beta",
	"google/cloud/dataform/v1":                       "beta",
	"google/cloud/developerconnect/v1":               "beta",
	"google/cloud/devicestreaming/v1":                "beta",
	"google/cloud/discoveryengine/v1alpha":           "beta",
	"google/cloud/financialservices/v1":              "beta",
	"google/cloud/gkeconnect/gateway/v1":             "beta",
	"google/cloud/gkerecommender/v1":                 "beta",
	"google/iam/v3":                                  "beta",
	"google/cloud/licensemanager/v1":                 "beta",
	"google/cloud/locationfinder/v1":                 "beta",
	"google/cloud/lustre/v1":                         "beta",
	"google/cloud/managedkafka/v1":                   "beta",
	"google/cloud/managedkafka/schemaregistry/v1":    "beta",
	"google/maps/areainsights/v1":                    "beta",
	"google/maps/places/v1":                          "beta",
	"google/maps/routeoptimization/v1":               "beta",
	"google/cloud/memorystore/v1":                    "beta",
	"google/cloud/modelarmor/v1":                     "beta",
	"google/monitoring/v3":                           "beta",
	"google/cloud/oracledatabase/v1":                 "beta",
	"google/cloud/parallelstore/v1":                  "beta",
	"google/cloud/parametermanager/v1":               "beta",
	"google/cloud/privilegedaccessmanager/v1":        "beta",
	"google/cloud/security/publicca/v1":              "beta",
	"google/cloud/securitycenter/v2":                 "beta",
	"google/cloud/securityposture/v1":                "beta",
	"google/shopping/css/v1":                         "beta",
	"google/shopping/merchant/productstudio/v1alpha": "beta",
	"google/spanner/adapter/v1":                      "beta",
	"google/spanner/executor/v1":                     "beta",
	"google/cloud/storagebatchoperations/v1":         "beta",
	"google/streetview/publish/v1":                   "beta",
	"google/devtools/cloudtrace/v2":                  "beta",
	"google/cloud/compute/v1beta":                    "ga",
	"google/cloud/bigquery/v2":                       "alpha",
}

// configFilename maps API paths to their service config filename when it
// doesn't follow the standard {service}_{version}.yaml pattern.
var configFilename = map[string]string{
	"google/cloud/billing/budgets/v1":              "billingbudgets.yaml",
	"google/cloud/billing/budgets/v1beta1":         "billingbudgets.yaml",
	"google/iam/v1":                                "iam_meta_api.yaml",
	"google/longrunning":                           "longrunning.yaml",
	"google/monitoring/dashboard/v1":               "monitoring.yaml",
	"google/monitoring/metricsscope/v1":            "monitoring.yaml",
	"google/monitoring/v3":                         "monitoring.yaml",
	"google/cloud/securitycenter/settings/v1beta1": "securitycenter_settings.yaml",
	"google/api/servicecontrol/v1":                 "servicecontrol.yaml",
	"google/spanner/adapter/v1":                    "spanner.yaml",
	"google/spanner/admin/database/v1":             "spanner.yaml",
	"google/spanner/admin/instance/v1":             "spanner.yaml",
	"google/spanner/v1":                            "spanner.yaml",
	"google/spanner/executor/v1":                   "spanner_cloud_executor.yaml",
	"google/streetview/publish/v1":                 "streetviewpublish.yaml",
}

// externalPath maps API paths to their service config path when the
// config is in a different directory than the API.
var externalPath = map[string]string{
	"google/cloud/aiplatform/v1/schema/predict/instance":       "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/v1/schema/predict/params":         "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/v1/schema/predict/prediction":     "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
	"google/cloud/aiplatform/v1/schema/trainingjob/definition": "google/cloud/aiplatform/v1/schema/aiplatform_v1.yaml",
}

// NewAPI returns the configuration for the given API path.
// It uses defaults and applies overrides where necessary.
func NewAPI(apiPath string) *API {
	cfg := &API{
		RESTNumericEnums: true,
		Metadata:         true,
		Transport:        "grpc+rest",
		ReleaseLevel:     deriveReleaseLevel(apiPath),
		DIREGAPIC:        diregapic[apiPath],
	}
	if t, ok := transport[apiPath]; ok {
		cfg.Transport = t
	}
	if rl, ok := releaseLevel[apiPath]; ok {
		cfg.ReleaseLevel = rl
	}
	return cfg
}

// deriveReleaseLevel determines the release level from the API path's version suffix.
// - Paths ending with alpha or alpha{N} return "alpha"
// - Paths ending with beta or beta{N} return "beta"
// - All other paths return "ga"
func deriveReleaseLevel(apiPath string) string {
	version := filepath.Base(apiPath)
	if strings.Contains(version, "alpha") {
		return "alpha"
	}
	if strings.Contains(version, "beta") {
		return "beta"
	}
	return "ga"
}

// PathOverride returns the service config path for APIs whose config
// is in a different directory and cannot be derived.
func PathOverride(apiPath string) (string, bool) {
	path, ok := externalPath[apiPath]
	return path, ok
}

// DerivePath returns the expected service config path for an API.
// It checks overrides first, then falls back to the standard pattern.
func DerivePath(apiPath string) string {
	// Check external path first (for APIs with service config in different directory).
	if p, ok := externalPath[apiPath]; ok {
		return p
	}

	// Check filename.
	if f, ok := configFilename[apiPath]; ok {
		return filepath.Join(apiPath, f)
	}

	// Standard pattern: {api_path}/{service_name}_{version}.yaml
	version := filepath.Base(apiPath)
	parentDir := filepath.Dir(apiPath)
	serviceName := filepath.Base(parentDir)

	// Handle cases where the parent is also a version-like component
	// e.g., google/cloud/bigquery/storage/v1 -> bigquerystorage
	if strings.HasPrefix(serviceName, "v") {
		serviceName = filepath.Base(filepath.Dir(parentDir))
	}

	filename := serviceName + "_" + version + ".yaml"
	return filepath.Join(apiPath, filename)
}
