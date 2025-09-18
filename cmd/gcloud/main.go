// Copyright 2024 Google LLC
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

// Librarian manages Google API client libraries by automating onboarding,
// regeneration, and release. It runs language‑agnostic workflows while
// delegating language‑specific tasks—such as code generation, building, and
// testing—to Docker images.
package main

import (
	"context"
	"log"
	"os"

	"github.com/googleapis/librarian/internal/librarian"
)

func main() {
	ctx := context.Background()
	if err := librarian.Run(ctx, os.Args[1:]...); err != nil {
		log.Fatal(err)
	}
}
