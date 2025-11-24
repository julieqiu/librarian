// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package conventionalcommit provides functionality for parsing and analyzing
// conventional commit messages according to the Conventional Commits specification.
//
// This package wraps github.com/leodido/go-conventionalcommits internally but does
// not expose it. The third-party library should never be imported outside this package.
package conventionalcommit

import (
	"fmt"
	"strings"

	"github.com/leodido/go-conventionalcommits"
	"github.com/leodido/go-conventionalcommits/parser"
)

// Commit represents a parsed conventional commit message.
type Commit struct {
	// Type is the type of change (e.g., "feat", "fix", "docs").
	Type string

	// Scope is the optional scope of the change (e.g., "api", "cli").
	Scope string

	// Description is the short summary of the change.
	Description string

	// Body is the long-form description of the change.
	Body string

	// Footers contain metadata (e.g., "BREAKING CHANGE", "Reviewed-by").
	// Multiple values for the same key are joined with newlines.
	Footers map[string]string

	// IsBreaking indicates if the commit introduces a breaking change.
	IsBreaking bool
}

// Parser parses conventional commit messages.
type Parser struct {
	machine conventionalcommits.Machine
}

// NewParser creates a new conventional commit parser.
func NewParser() *Parser {
	return &Parser{
		machine: parser.NewMachine(),
	}
}

func (p *Parser) Parse(message string) (*Commit, error) {
	parsed, err := p.machine.Parse([]byte(message))
	if err != nil {
		return nil, fmt.Errorf("failed to parse commit message: %w", err)
	}

	ccCommit, ok := parsed.(*conventionalcommits.ConventionalCommit)
	if !ok || !ccCommit.Ok() {
		return nil, fmt.Errorf("invalid conventional commit message")
	}

	commit := &Commit{
		Type:        ccCommit.Type,
		Description: ccCommit.Description,
		IsBreaking:  ccCommit.IsBreakingChange(),
		Footers:     make(map[string]string),
	}

	if ccCommit.Scope != nil {
		commit.Scope = *ccCommit.Scope
	}

	if ccCommit.Body != nil {
		commit.Body = *ccCommit.Body
	}

	for key, values := range ccCommit.Footers {
		if len(values) > 0 {
			commit.Footers[key] = strings.Join(values, "\n")
		}
	}

	return commit, nil
}

// IsFeat returns true if the commit is a feature commit.
func (c *Commit) IsFeat() bool {
	return c.Type == "feat"
}

// IsFix returns true if the commit is a fix commit.
func (c *Commit) IsFix() bool {
	return c.Type == "fix"
}

// HasFooter returns true if the commit has at least one footer.
func (c *Commit) HasFooter() bool {
	return len(c.Footers) > 0
}
