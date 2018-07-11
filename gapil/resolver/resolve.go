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

// Package resolver implements a semantic resolving for the api language.
// It is responsible for converting from an abstract syntax tree to a typed
// semantic graph ready for code generation.
package resolver

import (
	"fmt"

	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/semantic"
)

// Resolve takes valid asts as produced by the parser and converts them to the
// semantic graph form.
// If the asts are not fully valid (ie there were parse errors) then the results
// are undefined.
// If there are semantic problems with the ast, Resolve will return the set of
// errors it finds, and the returned graph may be incomplete/invalid.
func Resolve(includes []*ast.API, mappings *semantic.Mappings, options Options) (*semantic.API, parse.ErrorList) {
	rv := &resolver{
		api: &semantic.API{},
		scope: &scope{
			types: map[string]semantic.Type{},
		},
		mappings:           mappings,
		genericSubroutines: map[string]genericSubroutine{},
		options:            options,
	}
	func() {
		defer func() {
			err := recover()
			if err != nil && err != parse.AbortParse {
				if len(rv.errors) != 0 {
					panic(fmt.Errorf("Panic: %v\nErrors: %v", err, rv.errors))
				} else {
					panic(err)
				}
			}
		}()
		// Register all the built in symbols
		for _, t := range semantic.BuiltinTypes {
			rv.addType(t)
		}
		rv.with(semantic.VoidType, func() {
			for _, api := range includes {
				apiNames(rv, api)
				rv.mappings.Add(api, rv.api)
			}
			resolve(rv)
		})
	}()

	return rv.api, rv.errors
}
