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

// ExcludedAPIs lists API paths that should not be generated.
// The "All" list applies to all languages; language-specific lists
// contain additional exclusions for that language only.
// Paths are matched as prefixes (e.g., "google/cloud/foo" matches "google/cloud/foo/v1").
var ExcludedAPIs = struct {
	All  []string
	Go   []string
	Rust []string
}{
	All: []string{
		// Consumer Apps & Services
		"google/actions",   // Google Actions SDK
		"google/assistant", // Google Assistant
		"google/chromeos",  // ChromeOS

		// Internal/Infrastructure APIs
		"gapic",             // GAPIC metadata (internal)
		"google/bytestream", // ByteStream API (internal)
		"google/gapic",      // GAPIC metadata (internal)

		// Nested types that shouldn't be separate libraries
		"google/compute/logging",
		"google/iam/v1/logging",
		"google/iam/v2/logging",

		// AI & Machine Learning (beta/alpha versions and specialized APIs)
		"google/cloud/aiplatform/logging", // Logging types only
		"google/cloud/aiplatform/v1beta1",

		// API Infrastructure (internal/specialized)
		"google/api/expr",
		"google/api/serviceusage/v1beta1",

		// Bigtable (legacy versions)
		"google/bigtable/v1",
		"google/bigtable/v2",

		// Beta/Alpha versions and specialized Cloud services
		"google/cloud/abuseevent",
		"google/cloud/audit", // Audit logging types only
		"google/cloud/asset/v1p7beta1",
		"google/cloud/assuredworkloads/regulatoryintercept",
		"google/cloud/backupdr/logging",
		"google/cloud/bigquery/connection/v1beta1",
		"google/cloud/blockchainnodeengine",
		"google/cloud/certificatemanager/logging",
		"google/cloud/clouddms/logging",
		"google/cloud/cloudsetup",
		"google/cloud/confidentialcomputing/v1alpha1",
		"google/cloud/databasecenter", // Internal database center
		"google/cloud/dataform/logging",
		"google/cloud/datafusion/v1beta1",
		"google/cloud/datapipelines",
		"google/cloud/dataproc/logging", // Logging types only
		"google/cloud/datastream/logging",
		"google/cloud/discoveryengine/logging", // Logging types only
		"google/cloud/domains/v1alpha2",
		"google/cloud/eventarc/logging",
		"google/cloud/filer",
		"google/cloud/filestore/v1beta1",
		"google/cloud/functions/v2alpha",
		"google/cloud/functions/v2beta",
		"google/cloud/gkebackup/logging",
		"google/cloud/gkehub/policycontroller",
		"google/cloud/gkehub/servicemesh",
		"google/cloud/gkehub/v1/configmanagement",    // Nested module of gkehub/v1
		"google/cloud/gkehub/v1/multiclusteringress", // Nested module of gkehub/v1
		"google/cloud/gkehub/v1alpha",
		"google/cloud/gkehub/v1beta",
		"google/cloud/gsuiteaddons/logging",
		"google/cloud/healthcare", // Healthcare API (specialized)
		"google/cloud/iap/v1beta1",
		"google/cloud/identitytoolkit",
		"google/cloud/ids/logging",
		"google/cloud/integrations",
		"google/cloud/iot",
		"google/cloud/kms/logging",
		"google/cloud/kubernetes", // Kubernetes types (internal)
		"google/cloud/language/v1beta1",
		"google/cloud/managedidentities/v1beta1",
		"google/cloud/metastore/logging",
		"google/cloud/networkanalyzer",
		"google/cloud/networkmanagement/v1beta1",
		"google/cloud/networkservices/v1beta1",
		"google/cloud/notebooks/logging",
		"google/cloud/osconfig/agentendpoint",
		"google/cloud/osconfig/logging", // Logging types only
		"google/cloud/osconfig/v1beta",
		"google/cloud/oslogin/v1alpha",
		"google/cloud/oslogin/v1beta",
		"google/cloud/paymentgateway",
		"google/cloud/policytroubleshooter/iam/v3beta",
		"google/cloud/pubsublite",
		"google/cloud/recaptchaenterprise/v1beta1",
		"google/cloud/recommender/logging/v1beta1",
		"google/cloud/resourcemanager/v2",
		"google/cloud/retail/logging", // Logging types only
		"google/cloud/runtimeconfig",
		"google/cloud/saasaccelerator",
		"google/cloud/secretmanager/logging",
		"google/cloud/security/publicca/v1alpha1",
		"google/cloud/securitycenter/settings",
		"google/cloud/sensitiveaction",
		"google/cloud/servicehealth/logging",
		"google/cloud/sql/v1beta4",
		"google/cloud/stream",
		"google/cloud/universalledger",
		"google/cloud/vectorsearch",
		"google/cloud/video/livestream/logging",
		"google/cloud/workflows/type", // Workflow types (internal)
		"google/cloud/workstations/logging",

		// Other Google APIs (non-cloud)
		"google/compute",
		"google/container/v1alpha1",
		"google/datastore/admin/v1beta1",
		"google/datastore/v1",
		"google/datastore/v1beta3",
		"google/devtools/build",
		"google/devtools/clouderrorreporting",
		"google/devtools/containeranalysis/v1beta1",
		"google/devtools/remoteworkers",
		"google/devtools/resultstore",
		"google/devtools/sourcerepo",
		"google/devtools/testing",
		"google/example",
		"google/firebase",
		"google/firestore/admin/v1beta1",
		"google/firestore/admin/v1beta2",
		"google/firestore/bundle", // Bundle format types (internal)
		"google/firestore/v1",
		"google/firestore/v1beta1",
		"google/genomics",
		"google/home",
		"google/iam/v1beta",
		"google/networking/trafficdirector/type",
		"google/partner",
		"google/pubsub",
		"google/search",
		"google/security",
		"google/spanner/adapter",
		"google/spanner/executor",
		"google/spanner/v1",
		"google/streetview",
		"google/watcher",
	},

	Go: []string{
		// Advertising & Marketing
		"google/ads/admanager/v1",
		"google/ads/admob/v1",
		"google/ads/datamanager/v1",
		"google/ads/googleads/v19/common",
		"google/ads/googleads/v19/enums",
		"google/ads/googleads/v19/errors",
		"google/ads/googleads/v19/resources",
		"google/ads/googleads/v19/services",
		"google/ads/googleads/v20/common",
		"google/ads/googleads/v20/enums",
		"google/ads/googleads/v20/errors",
		"google/ads/googleads/v20/resources",
		"google/ads/googleads/v20/services",
		"google/ads/googleads/v21/common",
		"google/ads/googleads/v21/enums",
		"google/ads/googleads/v21/errors",
		"google/ads/googleads/v21/resources",
		"google/ads/googleads/v21/services",
		"google/ads/googleads/v22/common",
		"google/ads/googleads/v22/enums",
		"google/ads/googleads/v22/errors",
		"google/ads/googleads/v22/resources",
		"google/ads/googleads/v22/services",
		"google/ads/searchads360/v0/common",
		"google/ads/searchads360/v0/enums",
		"google/ads/searchads360/v0/errors",
		"google/ads/searchads360/v0/resources",
		"google/ads/searchads360/v0/services",

		// AI & Machine Learning
		"google/ai/generativelanguage/v1beta3",

		// Analytics
		"google/analytics/admin/v1beta",
		"google/analytics/cloud",
		"google/analytics/data/v1alpha",
		"google/analytics/data/v1beta",

		// API Infrastructure
		"google/api",
		"google/api/servicecontrol/v2",

		// App Engine
		"google/appengine/legacy",
		"google/appengine/logging/v1",
		"google/appengine/v1beta",

		// Google Apps
		"google/apps/alertcenter/v1beta1",
		"google/apps/card/v1",
		"google/apps/drive/activity/v2",
		"google/apps/drive/labels/v2",
		"google/apps/drive/labels/v2beta",
		"google/apps/script/type",
		"google/apps/script/type/calendar",
		"google/apps/script/type/docs",
		"google/apps/script/type/drive",
		"google/apps/script/type/gmail",
		"google/apps/script/type/sheets",
		"google/apps/script/type/slides",

		// Chat
		"google/chat/logging/v1",

		// Cloud AI Platform
		"google/cloud/aiplatform/v1/schema/predict/instance",
		"google/cloud/aiplatform/v1/schema/predict/params",
		"google/cloud/aiplatform/v1/schema/predict/prediction",
		"google/cloud/aiplatform/v1/schema/trainingjob/definition",

		// Cloud APIs (beta/alpha/specialized)
		"google/cloud/asset/v1p1beta1",
		"google/cloud/batch/v1alpha",
		"google/cloud/biglake/v1",
		"google/cloud/bigquery/logging/v1",
		"google/cloud/cloudsecuritycompliance/v1",
		"google/cloud/commerce/consumer/procurement/v1alpha1",
		"google/cloud/common",
		"google/cloud/compute/v1small",
		"google/cloud/configdelivery/v1alpha",
		"google/cloud/connectors/v1",
		"google/cloud/contentwarehouse/v1",
		"google/cloud/domains/v1",
		"google/cloud/enterpriseknowledgegraph/v1",
		"google/cloud/gdchardwaremanagement/v1alpha",
		"google/cloud/geminidataanalytics/v1alpha",
		"google/cloud/gkehub/v1",
		"google/cloud/hypercomputecluster/v1alpha",
		"google/cloud/hypercomputecluster/v1beta",
		"google/cloud/location",
		"google/cloud/mediatranslation/v1alpha1",
		"google/cloud/networksecurity/v1",
		"google/cloud/networksecurity/v1alpha1",
		"google/cloud/orchestration/airflow/service/v1beta1",
		"google/cloud/recommender/logging/v1",
		"google/cloud/redis/cluster/v1beta1",
		"google/cloud/secrets/v1beta1",
		"google/cloud/security/privateca/v1beta1",
		"google/cloud/sql/v1",
		"google/cloud/telcoautomation/v1alpha1",
		"google/cloud/texttospeech/v1beta1",
		"google/cloud/timeseriesinsights/v1",
		"google/cloud/tpu/v2",
		"google/cloud/tpu/v2alpha1",
		"google/cloud/translate/v3beta1",
		"google/cloud/videointelligence/v1p1beta1",
		"google/cloud/videointelligence/v1p2beta1",
		"google/cloud/vision/v1p2beta1",
		"google/cloud/vision/v1p3beta1",
		"google/cloud/vision/v1p4beta1",
		"google/cloud/visionai/v1alpha1",
		"google/cloud/websecurityscanner/v1alpha",
		"google/cloud/websecurityscanner/v1beta",

		// Container
		"google/container/v1beta1",

		// DevTools
		"google/devtools/containeranalysis/v1",
		"google/devtools/source/v1",

		// Geo
		"google/geo/type",

		// IAM
		"google/iam/admin/v1",
		"google/iam/v2beta",
		"google/identity/accesscontextmanager/type",

		// Logging
		"google/logging/type",

		// Maps
		"google/maps/aerialview/v1",
		"google/maps/mapsplatformdatasets/v1",
		"google/maps/mobilitybilling/logs/v1",
		"google/maps/playablelocations/v3",
		"google/maps/playablelocations/v3/sample",
		"google/maps/regionlookup/v1alpha",
		"google/maps/roads/v1op",
		"google/maps/routes/v1",
		"google/maps/routes/v1alpha",
		"google/maps/unity",
		"google/maps/weather/v1",

		// Marketing Platform
		"google/marketingplatform/admin/v1alpha",

		// RPC
		"google/rpc",
		"google/rpc/context",

		// Shopping
		"google/shopping/merchant/accounts/v1alpha",
		"google/shopping/merchant/reports/v1alpha",

		// Storage
		"google/storage/platformlogs/v1",
		"google/storage/v1",
		"google/storagetransfer/logging",

		// Types
		"google/type",

		// Grafeas
		"grafeas/v1",
	},

	Rust: []string{
		// Advertising & Marketing
		"google/ads",       // Google Ads (AdManager, AdMob, GoogleAds, SearchAds360)
		"google/analytics", // Google Analytics

		// Consumer Apps & Services
		"google/apps",    // Google Workspace apps (Meet, Chat, Drive Activity, etc.)
		"google/area120", // Area 120 experimental products
		"google/chat",    // Google Chat

		// Platform Services (typically not needed for client libraries)
		"google/appengine", // App Engine APIs

		// AI & Machine Learning (beta/alpha versions and specialized APIs)
		"google/ai", // AI APIs (mostly beta/experimental)

		// API Infrastructure (internal/specialized)
		"google/api/cloudquotas/v1beta",

		// Beta/Alpha versions and specialized Cloud services
		"google/cloud/alloydb/connectors/v1alpha",
		"google/cloud/alloydb/connectors/v1beta",
		"google/cloud/alloydb/v1alpha",
		"google/cloud/alloydb/v1beta",
		"google/cloud/apigeeregistry",
		"google/cloud/asset/v1p1beta1",
		"google/cloud/asset/v1p2beta1",
		"google/cloud/asset/v1p5beta1",
		"google/cloud/assuredworkloads/v1beta1",
		"google/cloud/automl",
		"google/cloud/batch",
		"google/cloud/biglake",
		"google/cloud/bigquery/biglake",
		"google/cloud/bigquery/dataexchange",
		"google/cloud/bigquery/datapolicies/v1beta1",
		"google/cloud/bigquery/datapolicies/v2beta1",
		"google/cloud/bigquery/logging",
		"google/cloud/bigquery/migration/v2alpha",
		"google/cloud/bigquery/storage",
		"google/cloud/billing/budgets",
		"google/cloud/binaryauthorization/v1beta1",
		"google/cloud/capacityplanner",
		"google/cloud/channel",
		"google/cloud/cloudcontrolspartner/v1beta",
		"google/cloud/commerce/consumer/procurement/v1alpha1",
		"google/cloud/compute/v1",
		"google/cloud/compute/v1beta",
		"google/cloud/configdelivery/v1alpha",
		"google/cloud/configdelivery/v1beta",
		"google/cloud/contentwarehouse",
		"google/cloud/datacatalog/v1beta1",
		"google/cloud/dataform/v1beta1",
		"google/cloud/datalabeling",
		"google/cloud/dataqna",
		"google/cloud/datastream/v1alpha1",
		"google/cloud/dialogflow/cx/v3beta1",
		"google/cloud/dialogflow/v2beta1",
		"google/cloud/discoveryengine/v1alpha",
		"google/cloud/discoveryengine/v1beta",
		"google/cloud/documentai/v1beta3",
		"google/cloud/domains/v1beta1",
		"google/cloud/enterpriseknowledgegraph",
		"google/cloud/eventarc/publishing",
		"google/cloud/functions/v1",
		"google/cloud/gdchardwaremanagement",
		"google/cloud/geminidataanalytics",
		"google/cloud/gkeconnect/gateway/v1beta1",
		"google/cloud/gkehub/v1beta1",
		"google/cloud/hypercomputecluster",
		"google/cloud/language/v1",
		"google/cloud/language/v1beta2",
		"google/cloud/lifesciences",
		"google/cloud/maintenance",
		"google/cloud/mediatranslation",
		"google/cloud/memcache/v1beta2",
		"google/cloud/memorystore/v1beta",
		"google/cloud/metastore/v1alpha",
		"google/cloud/metastore/v1beta",
		"google/cloud/modelarmor/v1beta",
		"google/cloud/networkconnectivity/v1alpha1",
		"google/cloud/networksecurity/v1alpha1",
		"google/cloud/networksecurity/v1beta1",
		"google/cloud/notebooks/v1",
		"google/cloud/notebooks/v1beta1",
		"google/cloud/orchestration/airflow/service/v1beta1",
		"google/cloud/osconfig/v1alpha",
		"google/cloud/parallelstore/v1beta",
		"google/cloud/phishingprotection",
		"google/cloud/privatecatalog",
		"google/cloud/recommendationengine",
		"google/cloud/recommender/v1beta1",
		"google/cloud/redis/cluster/v1beta1",
		"google/cloud/redis/v1beta1",
		"google/cloud/retail/v2alpha",
		"google/cloud/retail/v2beta",
		"google/cloud/saasplatform",
		"google/cloud/scheduler/v1beta1",
		"google/cloud/secretmanager/v1beta2",
		"google/cloud/secrets",
		"google/cloud/security/privateca/v1beta1",
		"google/cloud/security/publicca/v1beta1",
		"google/cloud/securitycenter/v1",
		"google/cloud/securitycenter/v1beta1",
		"google/cloud/securitycenter/v1p1beta1",
		"google/cloud/securitycentermanagement",
		"google/cloud/servicedirectory/v1beta1",
		"google/cloud/speech/v1",
		"google/cloud/speech/v1p1beta1",
		"google/cloud/support/v2beta",
		"google/cloud/talent/v4beta1",
		"google/cloud/tasks/v2beta2",
		"google/cloud/tasks/v2beta3",
		"google/cloud/telcoautomation/v1alpha1",
		"google/cloud/texttospeech/v1beta1",
		"google/cloud/tpu/v1",
		"google/cloud/tpu/v2alpha1",
		"google/cloud/translate/v3beta1",
		"google/cloud/videointelligence/v1",
		"google/cloud/videointelligence/v1beta2",
		"google/cloud/videointelligence/v1p1beta1",
		"google/cloud/videointelligence/v1p2beta1",
		"google/cloud/videointelligence/v1p3beta1",
		"google/cloud/vision/v1",
		"google/cloud/vision/v1p1beta1",
		"google/cloud/vision/v1p2beta1",
		"google/cloud/vision/v1p3beta1",
		"google/cloud/vision/v1p4beta1",
		"google/cloud/visionai/v1",
		"google/cloud/visionai/v1alpha1",
		"google/cloud/webrisk/v1beta1",
		"google/cloud/websecurityscanner/v1alpha",
		"google/cloud/websecurityscanner/v1beta",
		"google/cloud/workflows/executions/v1beta",
		"google/cloud/workflows/v1beta",
		"google/cloud/workstations/v1beta",

		// Other Google APIs (non-cloud)
		"google/container/v1beta1",
		"google/dataflow",
		"google/devtools/artifactregistry/v1beta2",
		"google/devtools/cloudtrace/v1",
		"google/devtools/source",
		"google/geo/type",
		"google/iam/v2beta",
		"google/iam/v3beta",
		"google/maps",
		"google/marketingplatform",
		"google/shopping",
		"google/storage",
	},
}

// ExactExcludedAPIs lists API paths that should be excluded only when they match exactly.
// Unlike ExcludedAPIs, these are not prefix matches - the channel path must match exactly.
var ExactExcludedAPIs = struct {
	All  []string
	Go   []string
	Rust []string
}{
	All: []string{
		"google/cloud", // Root cloud directory (common_resources.proto, extended_operations.proto)
	},
	Go:   []string{},
	Rust: []string{},
}
