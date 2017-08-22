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
)

func TestEnumGlobalAnalysis(t *testing.T) {
	ctx := log.Testing(t)

	common := `
    enum E{ A = 0x1  B = 0x2  C = 0x3 }
    E G
    `

	for _, test := range []struct {
		source   string
		expected *analysis.EnumValue
	}{
		{``, &analysis.EnumValue{
			Numbers: u64(0),
			Labels:  map[uint64]string{},
		}},
		{`cmd void c(E e) { G = e }`, &analysis.EnumValue{
			Numbers: u64Rng(0, 0xffffffffffffffff),
			Labels:  map[uint64]string{},
		}},
		{`cmd void c(E e) {
				switch e {
					case A, B, C:
						G = e
				}
			}`, &analysis.EnumValue{
			Numbers: u64(0, 1, 2, 3),
			Labels:  map[uint64]string{1: "A", 2: "B", 3: "C"},
		}},
		{`cmd void c(E e) {
				switch e {
					case A, B, C:
						G = e
				}
			}`, &analysis.EnumValue{
			Numbers: u64(0, 1, 2, 3),
			Labels:  map[uint64]string{1: "A", 2: "B", 3: "C"},
		}},
	} {
		ctx := log.V{"source": test.source}.Bind(ctx)
		api, mappings, err := compile(ctx, common+" "+test.source)
		assert.With(ctx).ThatError(err).Succeeded()
		res := analysis.Analyze(api, mappings)
		values := res.Globals[api.Globals[0]].(*analysis.EnumValue)
		assert.For(ctx, "numbers").That(values.Numbers).DeepEquals(test.expected.Numbers)
		assert.For(ctx, "labels").That(values.Labels).DeepEquals(test.expected.Labels)
	}
}

func TestEnumParameterAnalysis(t *testing.T) {
	ctx := log.Testing(t)

	common := `
    enum E{ A = 0x1  B = 0x2  C = 0x3 }
    `

	for _, test := range []struct {
		source   string
		expected *analysis.EnumValue
	}{
		{`cmd void c(E e) {
				switch e {
					case A, B, C: {}
				}
			}`, &analysis.EnumValue{
			Numbers: anyU64,
			Labels:  map[uint64]string{1: "A", 2: "B", 3: "C"},
		}}, {`cmd void c(E e) {
						if e == A {}
						if e == B {}
						if e == C {}
					}`, &analysis.EnumValue{
			Numbers: anyU64,
			Labels:  map[uint64]string{1: "A", 2: "B", 3: "C"},
		}}, {`cmd s32 c(E e) {
						return switch (e) {
							case A, B: 10
							case C: 30
						}
					}`, &analysis.EnumValue{
			Numbers: u64(1, 2, 3),
			Labels:  map[uint64]string{1: "A", 2: "B", 3: "C"},
		}}, {`sub s32 S(E e) {
						return switch (e) {
							case A, B: 10
							case C: 30
						}
					}
					cmd void c(E e) { x := S(e) }`, &analysis.EnumValue{
			Numbers: u64(1, 2, 3),
			Labels:  map[uint64]string{1: "A", 2: "B", 3: "C"},
		}},
	} {
		ctx := log.V{"source": test.source}.Bind(ctx)
		api, mappings, err := compile(ctx, common+" "+test.source)
		assert.With(ctx).ThatError(err).Succeeded()
		res := analysis.Analyze(api, mappings)
		values := res.Parameters[api.Functions[0].FullParameters[0]].(*analysis.EnumValue)
		assert.For(ctx, "numbers").That(values.Numbers).DeepEquals(test.expected.Numbers)
		assert.For(ctx, "labels").That(values.Labels).DeepEquals(test.expected.Labels)
	}
}

func TestBitfieldGlobalAnalysis(t *testing.T) {
	ctx := log.Testing(t)

	common := `
    bitfield E{ A = 0x1  B = 0x2  C = 0x4 }
    E G
    `

	for _, test := range []struct {
		source   string
		expected *analysis.EnumValue
	}{
		{``, &analysis.EnumValue{
			Numbers: u64(0),
			Labels:  map[uint64]string{},
		}},
		{`cmd void c(E e) { G = e }`, &analysis.EnumValue{
			Numbers: u64Rng(0, 0xffffffffffffffff),
			Labels:  map[uint64]string{},
		}},
		{`cmd void c(E e) {
				switch e {
					case A, B, C:
						G = e
				}
			}`, &analysis.EnumValue{
			Numbers: u64(0, 1, 2, 4),
			Labels:  map[uint64]string{1: "A", 2: "B", 4: "C"},
		}},
		{`cmd void c(E e) {
				if (A in e) { G = A }
				if (B in e) { G = B }
				if (C in e) { G = C }
			}`, &analysis.EnumValue{
			Numbers: u64(0, 1, 2, 4),
			Labels:  map[uint64]string{1: "A", 2: "B", 4: "C"},
		}},
		{`sub void s(E e) {
				G = e
			}
			cmd void c(E e) {
				s(A | B | C)
			}`, &analysis.EnumValue{
			Numbers: u64(0, 1, 2, 4),
			Labels:  map[uint64]string{1: "A", 2: "B", 4: "C"},
		}},
	} {
		ctx := log.V{"source": test.source}.Bind(ctx)
		api, mappings, err := compile(ctx, common+" "+test.source)
		assert.With(ctx).ThatError(err).Succeeded()
		res := analysis.Analyze(api, mappings)
		values := res.Globals[api.Globals[0]].(*analysis.EnumValue)
		assert.For(ctx, "numbers").That(values.Numbers).DeepEquals(test.expected.Numbers)
		assert.For(ctx, "labels").That(values.Labels).DeepEquals(test.expected.Labels)
	}
}

func TestBitfieldParameterAnalysis(t *testing.T) {
	ctx := log.Testing(t)

	common := `
    bitfield E{ A = 0x1  B = 0x2  C = 0x4 }
    `

	for _, test := range []struct {
		source   string
		expected *analysis.EnumValue
	}{
		{`cmd void c(E e) {
				switch e {
					case A, B, C: {}
				}
			}`, &analysis.EnumValue{
			Numbers: anyU64,
			Labels:  map[uint64]string{1: "A", 2: "B", 4: "C"},
		}},
		{`cmd void c(E e) {
				if (A in e) { }
				if (B in e) { }
				if (C in e) { }
			}`, &analysis.EnumValue{
			Numbers: anyU64,
			Labels:  map[uint64]string{1: "A", 2: "B", 4: "C"},
		}},
	} {
		ctx := log.V{"source": test.source}.Bind(ctx)
		api, mappings, err := compile(ctx, common+" "+test.source)
		assert.With(ctx).ThatError(err).Succeeded()
		res := analysis.Analyze(api, mappings)
		values := res.Parameters[api.Functions[0].FullParameters[0]].(*analysis.EnumValue)
		assert.For(ctx, "numbers").That(values.Numbers).DeepEquals(test.expected.Numbers)
		assert.For(ctx, "labels").That(values.Labels).DeepEquals(test.expected.Labels)
	}
}
