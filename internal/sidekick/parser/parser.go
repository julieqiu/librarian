// Copyright 2024 Google LLC
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

package parser

import (
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/config"
)

// DocumentationOverride describes overrides for the documentation of a single element.
//
// This should be used sparingly. Generally we should prefer updating the
// comments upstream, and then getting a new version of the services
// specification. The exception may be when the fixes take a long time, or are
// specific to one language.
type DocumentationOverride struct {
	ID      string
	Match   string
	Replace string
}

// PaginationOverride describes overrides for pagination config of a method.
type PaginationOverride struct {
	// The method ID.
	ID string
	// The name of the field used for `items`.
	ItemField string
}

// Discovery defines the configuration for discovery docs.
//
// It is too complex to just use key/value pairs in the `Config.Source` field.
type Discovery struct {
	// The ID of the LRO operation type.
	//
	// For example: ".google.cloud.compute.v1.Operation".
	OperationID string

	// Possible prefixes to match the LRO polling RPCs.
	//
	// In discovery-based services there may be multiple resources and RPCs that
	// service as LRO pollers. The order is important, sidekick picks the first
	// match, so the configuration should list preferred matches first.
	Pollers []*Poller
}

// Poller defines how to find a suitable poller RPC.
//
// For operations that may be LROs sidekick will match the URL path of the
// RPC against the prefixes.
type Poller struct {
	// An acceptable prefix for the URL path, for example:
	//     `compute/v1/projects/{project}/zones/{zone}`
	Prefix string

	// The corresponding method ID.
	MethodID string
}

// LroServices returns the set of Discovery LRO services.
//
// The discovery doc parser avoids generating LRO annotations for methods in
// this set. These functions return the LRO operation, but are inserted to that
// list, poll, wait for, and cancel LROs. They do not need the annotations and
// generated helpers.
func (d *Discovery) LroServices() map[string]bool {
	found := map[string]bool{}
	for _, poller := range d.Pollers {
		found[poller.serviceID()] = true
	}
	return found
}

// PathParameters returns the list of path parameters associated with a LRO
// poller.
//
// In discovery-based APIs different LRO functions use different polling
// methods. Each one of those methods uses a *subset* of the LRO functions to
// poll the operation. This method returns that subset.
func (p *Poller) PathParameters() []string {
	var parameters []string
	for _, segment := range strings.Split(p.Prefix, "/") {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			parameters = append(parameters, segment[1:len(segment)-1])
		}
	}
	return parameters
}

func (p *Poller) serviceID() string {
	idx := strings.LastIndex(p.MethodID, ".")
	if idx == -1 {
		return p.MethodID
	}
	return p.MethodID[:idx]
}

// Config contains the parser configuration.
type Config struct {
	SpecificationFormat string
	SpecificationSource string
	ServiceConfig       string
	Source              map[string]string
	PaginationOverrides []PaginationOverride
	CommentOverrides    []DocumentationOverride
	Discovery           *Discovery
}

// NewConfig creates a parser.Config from a config.Config.
func NewConfig(cfg *config.Config) *Config {
	var paginationOverrides []PaginationOverride
	for _, override := range cfg.PaginationOverrides {
		paginationOverrides = append(paginationOverrides, PaginationOverride{
			ID:        override.ID,
			ItemField: override.ItemField,
		})
	}

	var commentOverrides []DocumentationOverride
	for _, override := range cfg.CommentOverrides {
		commentOverrides = append(commentOverrides, DocumentationOverride{
			ID:      override.ID,
			Match:   override.Match,
			Replace: override.Replace,
		})
	}

	var discovery *Discovery
	if cfg.Discovery != nil {
		var pollers []*Poller
		for _, poller := range cfg.Discovery.Pollers {
			pollers = append(pollers, &Poller{
				Prefix:   poller.Prefix,
				MethodID: poller.MethodID,
			})
		}
		discovery = &Discovery{
			OperationID: cfg.Discovery.OperationID,
			Pollers:     pollers,
		}
	}

	return &Config{
		SpecificationFormat: cfg.General.SpecificationFormat,
		SpecificationSource: cfg.General.SpecificationSource,
		ServiceConfig:       cfg.General.ServiceConfig,
		Source:              cfg.Source,
		PaginationOverrides: paginationOverrides,
		CommentOverrides:    commentOverrides,
		Discovery:           discovery,
	}
}

// CreateModel parses the service specification referenced in `config`,
// cross-references the model, and applies any transformations or overrides
// required by the configuration.
func CreateModel(cfg *Config) (*api.API, error) {
	var err error
	var model *api.API
	switch cfg.SpecificationFormat {
	case "disco":
		model, err = ParseDisco(cfg)
	case "openapi":
		model, err = ParseOpenAPI(cfg)
	case "protobuf":
		model, err = ParseProtobuf(cfg)
	case "none":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown parser %q", cfg.SpecificationFormat)
	}
	if err != nil {
		return nil, err
	}
	updateMethodPagination(cfg.PaginationOverrides, model)
	api.LabelRecursiveFields(model)
	if err := api.CrossReference(model); err != nil {
		return nil, err
	}
	if err := api.SkipModelElements(model, cfg.Source); err != nil {
		return nil, err
	}
	var apiCommentOverrides []api.DocumentationOverride
	for _, override := range cfg.CommentOverrides {
		apiCommentOverrides = append(apiCommentOverrides, api.DocumentationOverride{
			ID:      override.ID,
			Match:   override.Match,
			Replace: override.Replace,
		})
	}
	if err := api.PatchDocumentation(model, apiCommentOverrides); err != nil {
		return nil, err
	}
	// Verify all the services, messages and enums are in the same package.
	if err := api.Validate(model); err != nil {
		return nil, err
	}
	if name, ok := cfg.Source["name-override"]; ok {
		model.Name = name
	}
	if title, ok := cfg.Source["title-override"]; ok {
		model.Title = title
	}
	if description, ok := cfg.Source["description-override"]; ok {
		model.Description = description
	}
	return model, nil
}
