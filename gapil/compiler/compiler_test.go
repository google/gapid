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

package compiler_test

import (
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/semantic"
)

func TestIR(t *testing.T) {
	ctx := assert.Context(t)

	processor := gapil.NewProcessor()

	api, errs := processor.Resolve("compiler_test.api")
	assert.For(ctx, "resolve errors").That(errs).IsNil()

	settings := compiler.Settings{}

	prog, err := compiler.Compile(api, processor.Mappings, settings)
	assert.For(ctx, "compile errors").ThatError(err).Succeeded()

	funcIR := prog.IR()
	for _, f := range api.Subroutines {
		got, expected := funcIR[f.Name()], expectedIR(f, "ir")
		assert.For(ctx, "%v.IR", f.Name()).ThatString(got).Equals(expected)
	}

	for _, f := range api.Functions {
		got, expected := funcIR[f.Name()], expectedIR(f, "ir")
		assert.For(ctx, "%v.IR", f.Name()).ThatString(got).Equals(expected)
	}
}

func expectedIR(f *semantic.Function, tag string) string {
	a := f.Annotations.GetAnnotation(tag)
	if a == nil {
		return "<missing @" + tag + " annotation>"
	}
	return strings.TrimSpace(string(a.Arguments[0].(semantic.StringValue)))
}
