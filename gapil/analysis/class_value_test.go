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

package analysis_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil/analysis"
	"github.com/google/gapid/gapil/semantic"
)

type field struct {
	name  string
	value analysis.Value
}

func class(name string, fields ...field) *analysis.ClassValue {
	out := &analysis.ClassValue{
		Class:  &semantic.Class{Named: semantic.Named(name)},
		Fields: make(map[string]analysis.Value, len(fields)),
	}
	for _, f := range fields {
		out.Fields[f.name] = f.value
	}
	return out
}

func TestClassGlobalAnalysis(t *testing.T) {
	ctx := log.Testing(t)

	common := `
    class X {
        u32 a
        Y   b
    }
    class Y {
        u32 c
    }
    X G
    `

	for _, test := range []struct {
		source   string
		expected *analysis.ClassValue
	}{
		{``, class("X", field{"a", u32(0)}, field{"b", class("Y", field{"c", u32(0)})})},
		{`cmd void c() { G.a = 1 }`, class("X", field{"a", u32(0, 1)}, field{"b", class("Y", field{"c", u32(0)})})},
		{`cmd void c() { G.b.c = 1 }`, class("X", field{"a", u32(0)}, field{"b", class("Y", field{"c", u32(0, 1)})})},
		{`cmd void c() { G.b = Y(1) }`, class("X", field{"a", u32(0)}, field{"b", class("Y", field{"c", u32(0, 1)})})},
		{`cmd void c() { x := X(1, Y(2))  G = x }`, class("X", field{"a", u32(0, 1)}, field{"b", class("Y", field{"c", u32(0, 2)})})},
	} {
		ctx := log.V{"source": test.source}.Bind(ctx)
		api, mappings, err := compile(ctx, common+" "+test.source)
		assert.For(ctx, "err").ThatError(err).Succeeded()
		res := analysis.Analyze(api, mappings)
		got := res.Globals[api.Globals[0]].(*analysis.ClassValue)
		assert.For(ctx, "res").ThatString(got.Print(res)).Equals(test.expected.Print(res))
	}
}
