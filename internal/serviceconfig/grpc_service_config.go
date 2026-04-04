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

package serviceconfig

// GRPCServiceConfig represents the gRPC service config defined in AIP-4221.
// It enables API producers to specify client-side retry and timeout behavior
// per method without requiring users to configure retries manually.
//
// See https://google.aip.dev/client-libraries/4221 for the full specification
// and https://github.com/grpc/proposal/blob/master/A6-client-retries.md for
// the retry design.
type GRPCServiceConfig struct {
	MethodConfig []MethodConfig `yaml:"method_config,omitempty" json:"methodConfig,omitempty"`
}

// MethodConfig describes retry and timeout behavior for a set of methods.
// Each entry applies to the methods identified by its Name field.
type MethodConfig struct {
	// Name identifies which service methods this config applies to.
	// An empty Method field means all methods on the named service.
	Name []MethodName `yaml:"name,omitempty" json:"name,omitempty"`

	// Timeout is the maximum duration for an RPC, including all retry
	// attempts. Uses proto3 Duration string format (e.g., "60s").
	Timeout string `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// RetryPolicy defines the retry behavior for matching methods.
	RetryPolicy *RetryPolicy `yaml:"retry_policy,omitempty" json:"retryPolicy,omitempty"`
}

// MethodName identifies a method or all methods of a service.
// If Method is empty, the config applies to all methods on the service.
type MethodName struct {
	// Service is the fully qualified gRPC service name
	// (e.g., "google.cloud.secretmanager.v1.SecretManagerService").
	Service string `yaml:"service,omitempty" json:"service,omitempty"`

	// Method is the method name (e.g., "GetSecret"). If empty, the
	// config applies to all methods on the service.
	Method string `yaml:"method,omitempty" json:"method,omitempty"`
}

// RetryPolicy defines exponential backoff retry behavior. The delay before
// attempt n is random(0, min(InitialBackoff * BackoffMultiplier^(n-1), MaxBackoff)).
//
// Retries only occur for status codes listed in RetryableStatusCodes, and
// the overall RPC deadline applies across all attempts.
type RetryPolicy struct {
	// MaxAttempts is the maximum number of RPC attempts including the
	// original request. Values above 5 are treated as 5.
	MaxAttempts int `yaml:"max_attempts,omitempty" json:"maxAttempts,omitempty"`

	// InitialBackoff is the delay before the first retry, in proto3 Duration
	// format (e.g., "1s").
	InitialBackoff string `yaml:"initial_backoff,omitempty" json:"initialBackoff,omitempty"`

	// MaxBackoff caps the retry delay, in proto3 Duration format (e.g., "10s").
	MaxBackoff string `yaml:"max_backoff,omitempty" json:"maxBackoff,omitempty"`

	// BackoffMultiplier controls exponential growth of the delay between
	// successive retry attempts.
	BackoffMultiplier float64 `yaml:"backoff_multiplier,omitempty" json:"backoffMultiplier,omitempty"`

	// RetryableStatusCodes lists the gRPC status codes that trigger a retry
	// (e.g., "UNAVAILABLE").
	RetryableStatusCodes []string `yaml:"retryable_status_codes,omitempty" json:"retryableStatusCodes,omitempty"`
}
