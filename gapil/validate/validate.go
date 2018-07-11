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

// Package validate registers and implements the "validate" apic command.
//
// The validate command analyses the specified API for correctness, reporting errors if any problems
// are found.
package validate

import (
	"sort"

	"github.com/google/gapid/gapil/analysis"
	"github.com/google/gapid/gapil/semantic"
)

// Options controls the validation that's performed.
type Options struct {
	CheckUnused bool // Should unused types, fields, etc be reported?
}

// Validate performs a number of checks on the api file for correctness.
// If any problems are found then they are returned as errors.
// If options is nil then full validation is performed.
func Validate(api *semantic.API, mappings *semantic.Mappings, options *Options) Issues {
	res := analysis.Analyze(api, mappings)
	return WithAnalysis(api, mappings, options, res)
}

// WithAnalysis performs a number of checks on the api file for
// correctness using pre-built analysis results.
// If any problems are found then they are returned as errors.
// If options is nil then full validation is performed.
func WithAnalysis(api *semantic.API, mappings *semantic.Mappings, options *Options, analysis *analysis.Results) Issues {
	if options == nil {
		options = &Options{true}
	}
	issues := Issues{}
	if options.CheckUnused {
		issues = append(issues, noUnused(api, mappings)...)
	}
	issues = append(issues, inspect(api, mappings, analysis)...)
	sort.Sort(issues)
	return issues
}
