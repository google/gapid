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
	"context"
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/analysis"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/parser"
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
)

func u32Rng(s, e int) analysis.Value {
	return &analysis.UintValue{
		Ty:     semantic.Uint32Type,
		Ranges: interval.U64SpanList{{Start: uint64(s), End: uint64(e)}},
	}
}

func u32(vals ...uint32) *analysis.UintValue {
	out := &analysis.UintValue{
		Ty:     semantic.Uint32Type,
		Ranges: interval.U64SpanList{},
	}
	for _, v := range vals {
		interval.Merge(&out.Ranges, interval.U64Span{Start: uint64(v), End: uint64(v) + 1}, true)
	}
	return out
}

func u64Rng(s, e uint64) *analysis.UintValue {
	return &analysis.UintValue{
		Ty:     semantic.Uint64Type,
		Ranges: interval.U64SpanList{{Start: uint64(s), End: uint64(e)}},
	}
}

func u64(vals ...uint64) *analysis.UintValue {
	out := &analysis.UintValue{
		Ty: semantic.Uint64Type,
	}
	for _, v := range vals {
		interval.Merge(&out.Ranges, interval.U64Span{Start: v, End: v + 1}, true)
	}
	return out
}

var anyU64 = &analysis.UintValue{
	Ty:     semantic.Uint64Type,
	Ranges: interval.U64SpanList{{End: 0xffffffffffffffff}},
}

func toValue(v interface{}) analysis.Value {
	switch v := v.(type) {
	case int:
		return u32(uint32(v))
	case analysis.Value:
		return v
	}
	panic(fmt.Errorf("toValue does not support type %T", v))
}

func compile(ctx context.Context, source string) (*semantic.API, *semantic.Mappings, error) {
	const maxErrors = 10
	m := &semantic.Mappings{}
	parsed, errs := parser.Parse("analysis_test.api", source, &m.AST)
	if err := gapil.CheckErrors(source, errs, maxErrors); err != nil {
		return nil, nil, err
	}
	compiled, errs := resolver.Resolve([]*ast.API{parsed}, m, resolver.Options{})
	if err := gapil.CheckErrors(source, errs, maxErrors); err != nil {
		return nil, nil, err
	}
	return compiled, m, nil
}

func TestU32GlobalAnalysis(t *testing.T) {
	ctx := log.Testing(t)

	common := `u32 G = 10`

	for _, test := range []struct {
		source   string
		expected analysis.Value
	}{
		{``, u32(10)},
		{`cmd void c() { G = 20 } cmd void bar() { G = 30 }`, u32(10, 20, 30)},
		{`cmd void c() { x := as!u32(20)  G = x }`, u32(10, 20)},
		{`cmd void c() { if true { G = 20 } else { G = 30 } }`, u32(10, 20)},
		{`cmd void c() { if false { G = 20 } else { G = 30 } }`, u32(10, 30)},
		{`cmd void c() { switch true { case true:  G = 20  default: G = 30 } }`, u32(10, 20)},
		{`cmd void c() { switch true { case false: G = 20  default: G = 30 } }`, u32(10, 30)},
		{`cmd void c() { G = switch true { case true:  20  default: 30 } }`, u32(10, 20)},
		{`cmd void c() { G = switch true { case false: 20  default: 30 } }`, u32(10, 30)},
		{`cmd void c(u32 a) { if a == 5 { G = 20 } else { G = 30 } }`, u32(10, 20, 30)},
		{`cmd void c(u32 a) { if (a == 1) { G = a } }`, u32(10, 1)},
		{`cmd void c(u32 a) { if (a == 1) || (a == 2) { G = a } }`, u32(10, 1, 2)},
		{`cmd void c(u32 a) { if a == 1 { G = a } }`, u32(10, 1)},
		{`cmd void c(u32 a) { if a < 1 { G = a } }`, u32(10, 0)},
		{`cmd void c(u32 a) { if (a > 1) && (a < 3) { G = a } }`, u32(10, 2)},
		{`cmd void c(u32 a) { if (a >= 1) && (a <= 3) { G = a } }`, u32(10, 1, 2, 3)},
		{`cmd void c(u32 a) { G = switch a { case 0: 20  case 1: 30  case 2: 40 } }`, u32(10, 20, 30, 40)},
		{`cmd void c(u32 a) { if a == 1 { G = switch a { case 0: 20  case 1: 30  case 2: 40 } } }`, u32(10, 30)},
		{`cmd void c(u32 a) { if a < 1 { G = switch a { case 0: 20  case 1: 30  case 2: 40 } } }`, u32(10, 20)},
		{`cmd void c(u32 a) { if a > 1 { G = switch a { case 0: 20  case 1: 30  case 2: 40 } } }`, u32(10, 40)},
		{`cmd void c(u32 a) { switch true { case a >= 5: {}  default: G = a } }`, u32(10, 0, 1, 2, 3, 4)},
		{`cmd void c(u32 a) { switch true { case a == 0, a == 2: {}  case a < 5: G = a } }`, u32(10, 1, 3, 4)},
		{`cmd void c(u32 a) { switch a { case 0: {} default: abort } G = a }`, u32(10, 0)},
		{`cmd void c(u32 a) { switch a { case 0, 2: {} default: abort } G = a }`, u32(10, 0, 2)},
		{`cmd void c(u32 a) { switch a { case 0, 2, 4: {} default: abort } G = a }`, u32(10, 0, 2, 4)},
		{`cmd void c(u32 a) { switch a { case 0: { switch a { case 0: G = a } } } }`, u32(10, 0)},

		{`cmd void c(u32 a) { if a >  2 { abort } G = a }`, u32(10, 0, 1, 2)},
		{`cmd void c(u32 a) { if a >= 2 { abort } G = a }`, u32(10, 0, 1)},
		{`cmd void c(u32 a) { if a <  2 { abort } G = a }`, u32Rng(2, 0x100000000)},
		{`cmd void c(u32 a) { if a <= 2 { abort } G = a }`, u32Rng(3, 0x100000000)},
		{`cmd void c(u32 a) { if a >  2 { } else { abort } G = a }`, u32Rng(3, 0x100000000)},
		{`cmd void c(u32 a) { if a >= 2 { } else { abort } G = a }`, u32Rng(2, 0x100000000)},
		{`cmd void c(u32 a) { if a <  2 { } else { abort } G = a }`, u32(10, 0, 1)},
		{`cmd void c(u32 a) { if a <= 2 { } else { abort } G = a }`, u32(10, 0, 1, 2)},

		{`cmd void c(u32 a) { if !(a >  2) { abort } G = a }`, u32Rng(3, 0x100000000)},
		{`cmd void c(u32 a) { if !(a >= 2) { abort } G = a }`, u32Rng(2, 0x100000000)},
		{`cmd void c(u32 a) { if !(a <  2) { abort } G = a }`, u32(10, 0, 1)},
		{`cmd void c(u32 a) { if !(a <= 2) { abort } G = a }`, u32(10, 0, 1, 2)},
		{`cmd void c(u32 a) { if !(a >  2) { } else { abort } G = a }`, u32(10, 0, 1, 2)},
		{`cmd void c(u32 a) { if !(a >= 2) { } else { abort } G = a }`, u32(10, 0, 1)},
		{`cmd void c(u32 a) { if !(a <  2) { } else { abort } G = a }`, u32Rng(2, 0x100000000)},
		{`cmd void c(u32 a) { if !(a <= 2) { } else { abort } G = a }`, u32Rng(3, 0x100000000)},

		{`sub void s(u32 a) { if a > 1 { abort } }  cmd void c(u32 x) { s(x)  G = x }`, u32(10, 0, 1)},
		{`cmd void c(u32* a) { G = switch (a != null) { case true: 20  case false: 30 } }`, u32(10, 20, 30)},
		{`cmd void c(u32* a) { switch (a != null) { case true: G = 20  case false: G = 30 } }`, u32(10, 20, 30)},
		{`sub void s(u32 a) { G = a }  cmd void c() { s(20) s(30) }`, u32(10, 30)},
		{`sub u32 s() { return 1 }  cmd void c(u32 x) { G = s() }`, u32(10, 1)},

		{`type u32 X  type u32 Y  cmd void c(Y a) { if (a == 1) { G = as!u32(a) } }`, u32(10, 1)},

		{`cmd void c(u32 a, u32 b) { if a > b { G = a } }`, u32Rng(1, 0x100000000)},
		{`cmd void c(u32 a, u32 b) { if a < b { G = a } }`, u32Rng(0, 0xffffffff)},
		{`cmd void c(u32 a, u32 b) { if a >= b { G = a } }`, u32Rng(0, 0x100000000)},
		{`cmd void c(u32 a, u32 b) { if a <= b { G = a } }`, u32Rng(0, 0x100000000)},
		{`cmd void c(u32 a, u32 b) { if a > b { G = b } }`, u32Rng(0, 0xffffffff)},
		{`cmd void c(u32 a, u32 b) { if a < b { G = b } }`, u32Rng(1, 0x100000000)},
		{`cmd void c(u32 a, u32 b) { if a >= b { G = b } }`, u32Rng(0, 0x100000000)},
		{`cmd void c(u32 a, u32 b) { if a <= b { G = b } }`, u32Rng(0, 0x100000000)},
	} {
		ctx := log.V{"source": test.source}.Bind(ctx)
		api, mappings, err := compile(ctx, common+" "+test.source)
		assert.For(ctx, "err").ThatError(err).Succeeded()
		res := analysis.Analyze(api, mappings)
		values := res.Globals[api.Globals[0]]
		assert.For(ctx, "vals").That(values).DeepEquals(test.expected)
	}
}
