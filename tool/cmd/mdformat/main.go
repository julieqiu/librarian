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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("mdformat", flag.ContinueOnError)
	write := fs.Bool("w", false, "write result to (source) file instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: mdformat [-w] <filename>")
	}
	filename := fs.Arg(0)

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var processed, currentParagraph []string

	flushParagraph := func() {
		if len(currentParagraph) == 0 {
			return
		}
		text := strings.Join(currentParagraph, " ")
		processed = append(processed, wrap(text))
		currentParagraph = nil
	}

	for _, line := range lines {
		// Normalize bullets: change "* " to "- "
		if strings.HasPrefix(strings.TrimSpace(line), "* ") {
			line = strings.Replace(line, "* ", "- ", 1)
		}

		if isWrappable(line) {
			currentParagraph = append(currentParagraph, strings.TrimSpace(line))
		} else {
			flushParagraph()
			processed = append(processed, line)
		}
	}
	flushParagraph()

	// Post-processing: Enforce Vertical Density
	var denseLines []string
	for i, line := range processed {
		trimmed := strings.TrimSpace(line)
		if i > 0 {
			prevTrimmed := strings.TrimSpace(processed[i-1])
			// 1. Remove double blank lines.
			if trimmed == "" && prevTrimmed == "" {
				continue
			}
			// 2. Ensure blank line after headers (level 2 or more).
			if strings.HasPrefix(prevTrimmed, "##") && trimmed != "" {
				denseLines = append(denseLines, "")
			}
		}
		denseLines = append(denseLines, line)
	}

	output := strings.Join(denseLines, "\n")
	if *write {
		if err := os.WriteFile(filename, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	} else {
		fmt.Print(output)
	}
	return nil
}

func isWrappable(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	// Do not wrap headers, list items, tables, or indented code.
	if strings.HasPrefix(line, "#") ||
		strings.HasPrefix(trimmed, "- ") ||
		strings.HasPrefix(trimmed, "|") ||
		strings.HasPrefix(line, "  ") ||
		strings.HasPrefix(line, "\t") {
		return false
	}
	return true
}

func wrap(text string) string {
	var words []string
	if strings.HasPrefix(text, "- ") {
		words = strings.Fields(text[2:])
		return "- " + wrapWords(words, 78, "  ")
	}
	words = strings.Fields(text)
	return wrapWords(words, 80, "")
}

func wrapWords(words []string, width int, indent string) string {
	if len(words) == 0 {
		return ""
	}
	var out strings.Builder
	currLineLen := 0
	for _, w := range words {
		if currLineLen > 0 && currLineLen+1+len(w) > width {
			out.WriteString("\n")
			out.WriteString(indent)
			currLineLen = len(indent)
		} else if currLineLen > 0 {
			out.WriteString(" ")
			currLineLen++
		}
		out.WriteString(w)
		currLineLen += len(w)
	}
	return out.String()
}