// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build gofuzz

// Package fuzz is a fuzzing test for the api parser and compiler.
//
// See: https://github.com/dvyukov/go-fuzz
package fuzz

import (
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/semantic"
)

func compile(data []byte) bool {
	// Build a processor that will 'load' from data.
	processor := gapil.Processor{
		Mappings:            &semantic.Mappings{},
		Loader:              gapil.NewDataLoader(data),
		Parsed:              map[string]gapil.ParseResult{},
		Resolved:            map[string]gapil.ResolveResult{},
		ResolveOnParseError: true,
	}
	_, errs := processor.Resolve("fuzz")
	return len(errs) == 0
}

// Fuzz compiles the input API data.
func Fuzz(data []byte) int {
	if compile(data) {
		return 1 // the fuzzer should increase priority of the given input during subsequent fuzzing
	}
	return 0
}
