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
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/validate"
)

func init() {
	app.AddVerb(&app.Verb{
		Name:      "validate",
		ShortHelp: "Validates an api file for correctness",
		Action:    &validateVerb{},
	})
}

type validateVerb struct{}

func (v *validateVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	args := flags.Args()
	if len(args) < 1 {
		app.Usage(ctx, "Missing api file")
		return nil
	}
	for _, apiName := range args {
		processor := gapil.NewProcessor()
		compiled, errs := processor.Resolve(apiName)
		if err := gapil.CheckErrors(apiName, errs, maxErrors); err != nil {
			return err
		}
		log.I(ctx, "Validating %v", apiName)
		issues := validate.Validate(compiled, processor.Mappings, nil)
		fmt.Fprintf(os.Stderr, "%v\n", issues)
		if c := len(issues); c > 0 {
			return fmt.Errorf("%d issues found", c)
		}
	}
	return nil
}
