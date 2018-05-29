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
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapil/analysis"
)

func TestU32ToU32MapGlobalAnalysis(t *testing.T) {
	ctx := log.Testing(t)

	common := `map!(u32, u32) M`

	for _, test := range []struct {
		source   string
		expected string
	}{
		{``, `{ }`},
		{`cmd void f() { M[0x1] = 2 }`, `{ <[0x1]: [0x2]> }`},
		{`cmd void f() { M[0x1] = 2 M[0x2] = 3 }`, `{ <[0x1]: [0x2]>, <[0x2]: [0x3]> }`},
		{`cmd void f() { M[0x1] = 2 M[0x1] = 3 }`, `{ <[0x1]: [0x3]> }`},
		{`cmd void f() { M[0x1] = 2 } cmd void bar() { M[0x1] = 3 }`, `{ <[0x1]: [0x2-0x3]> }`},
		{`cmd void f(u32 a, u32 b) { M[a] = 1  M[b] = 2 }`, `{ <[0x0-0xffffffff]: [0x1-0x2]> }`},
		{`cmd void f(u32 a) { if a < 2 { M[a] = 1 } }`, `{ <[0x0-0x1]: [0x1]> }`},
		{`cmd void f(u32 a) {
          switch a {
            case 0: M[0] = 0
            case 1: M[0x1] = 1
            case 2: M[0x2] = 2
          }
        }`, `{ <[0x0]: [0x0]>, <[0x1]: [0x1]>, <[0x2]: [0x2]> }`,
		}, {
			`cmd void f(u32 a) { switch a { case 0, 1, 2: M[a] = a } }
             cmd void g(u32 a) { switch a { case 1, 2, 3: M[a] = a } }`,
			`{ <[0x0]: [0x0]>, <[0x1]: [0x1]>, <[0x2]: [0x2]>, <[0x3]: [0x3]> }`,
		},
	} {
		ctx := log.V{"source": test.source}.Bind(ctx)
		api, mappings, err := compile(ctx, common+" "+test.source)
		assert.For(ctx, "err").ThatError(err).Succeeded()
		res := analysis.Analyze(api, mappings)
		got := res.Globals[api.Globals[0]].(*analysis.MapValue)
		s := strings.Join(strings.Fields(got.Print(res)), " ")
		assert.For(ctx, "s").ThatString(s).Equals(test.expected)
	}
}

func TestU32ToU32MapGlobalAnalysis2(t *testing.T) {
	ctx := log.Testing(t)

	common := `
    map!(u32, u32) M
    u32 G`

	for _, test := range []struct {
		source   string
		expected string
	}{
		{``, `[0x0]`},
		{`cmd void f() { G = M[0x1] }`, `[0x0]`},
	} {
		ctx := log.V{"source": test.source}.Bind(ctx)
		api, mappings, err := compile(ctx, common+" "+test.source)
		assert.For(ctx, "err").ThatError(err).Succeeded()
		res := analysis.Analyze(api, mappings)
		got := res.Globals[api.Globals[1]].(*analysis.UintValue)
		s := strings.Join(strings.Fields(got.Print(res)), " ")
		assert.For(ctx, "s").ThatString(s).Equals(test.expected)
	}
}
func TestPointerToRefStructMapGlobalAnalysis(t *testing.T) {
	ctx := log.Testing(t)

	common := `class C { u32 a  u32 b  u32 c}
               map!(void*, ref!C) M`

	for _, test := range []struct {
		source   string
		expected string
	}{
		{``, `{ }`},
		{
			`cmd void f(void* k) { M[k] = new!C(1, 2, 3) }`,
			`{ <<untracked>: ref!C{ a: [0x1] b: [0x2] c: [0x3] }> }`,
		}, {
			`cmd void f(void* k) { M[k] = new!C(1, 2, 3) }
             cmd void g(void* k) { M[k] = new!C(4, 5, 6) }`,
			`{ <<untracked>: ref!C{ a: [0x1] [0x4] b: [0x2] [0x5] c: [0x3] [0x6] }> }`,
		}, {
			`cmd void f(void* k) { M[k] = new!C(1, 2, 3) }
             cmd void g(void* k) { if M[k] != null { M[k] = new!C(4, 5, 6) } }`,
			`{ <<untracked>: ref!C{ a: [0x1] [0x4] b: [0x2] [0x5] c: [0x3] [0x6] }> }`,
		}, {
			`cmd void f(void* k) { M[k] = new!C(1, 2, 3) }
             cmd void g(void* k) { if M[k] == null { abort } M[k] = new!C(4, 5, 6) }`,
			`{ <<untracked>: ref!C{ a: [0x1] [0x4] b: [0x2] [0x5] c: [0x3] [0x6] }> }`,
		},
	} {
		ctx := log.V{"source": test.source}.Bind(ctx)
		api, mappings, err := compile(ctx, common+" "+test.source)
		assert.For(ctx, "err").ThatError(err).Succeeded()
		res := analysis.Analyze(api, mappings)
		got := res.Globals[api.Globals[0]].(*analysis.MapValue)
		s := strings.Join(strings.Fields(got.Print(res)), " ")
		assert.For(ctx, "s").ThatString(s).Equals(test.expected)
	}
}
