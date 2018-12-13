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

package interval

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
)

func TestUpdate(t *testing.T) {
	ctx := log.Testing(t)
	add1 := func(x interface{}) interface{} {
		if x == nil {
			return 0
		}
		return x.(int) + 1
	}
	const1 := func(x interface{}) interface{} {
		return 1
	}
	for _, test := range []struct {
		name     string
		list     ValueSpanList
		span     U64Span
		f        func(interface{}) interface{}
		expected ValueSpanList
	}{
		{"Empty",
			ValueSpanList{},
			U64Span{0, 10},
			add1,
			ValueSpanList{ValueSpan{U64Span{0, 10}, 0}},
		},
		{"match",
			ValueSpanList{ValueSpan{U64Span{0, 10}, 1}},
			U64Span{0, 10},
			add1,
			ValueSpanList{ValueSpan{U64Span{0, 10}, 2}},
		},
		{"split",
			ValueSpanList{ValueSpan{U64Span{5, 25}, 1}},
			U64Span{15, 20},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{5, 15}, 1},
				ValueSpan{U64Span{15, 20}, 2},
				ValueSpan{U64Span{20, 25}, 1},
			},
		},
		{"split match front",
			ValueSpanList{ValueSpan{U64Span{5, 25}, 1}},
			U64Span{5, 20},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{5, 20}, 2},
				ValueSpan{U64Span{20, 25}, 1},
			},
		},
		{"split match end",
			ValueSpanList{ValueSpan{U64Span{5, 25}, 1}},
			U64Span{15, 25},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{5, 15}, 1},
				ValueSpan{U64Span{15, 25}, 2},
			},
		},
		{"split front",
			ValueSpanList{ValueSpan{U64Span{15, 25}, 1}},
			U64Span{10, 20},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{10, 15}, 0},
				ValueSpan{U64Span{15, 20}, 2},
				ValueSpan{U64Span{20, 25}, 1},
			},
		},
		{"split front match front",
			ValueSpanList{ValueSpan{U64Span{15, 25}, 1}},
			U64Span{10, 15},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{10, 15}, 0},
				ValueSpan{U64Span{15, 25}, 1},
			},
		},
		{"split front match end",
			ValueSpanList{ValueSpan{U64Span{15, 25}, 1}},
			U64Span{10, 25},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{10, 15}, 0},
				ValueSpan{U64Span{15, 25}, 2},
			},
		},
		{"split end",
			ValueSpanList{ValueSpan{U64Span{5, 15}, 1}},
			U64Span{10, 20},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{5, 10}, 1},
				ValueSpan{U64Span{10, 15}, 2},
				ValueSpan{U64Span{15, 20}, 0},
			},
		},
		{"split end match front",
			ValueSpanList{ValueSpan{U64Span{5, 15}, 1}},
			U64Span{15, 20},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{5, 15}, 1},
				ValueSpan{U64Span{15, 20}, 0},
			},
		},
		{"split end match end",
			ValueSpanList{ValueSpan{U64Span{5, 15}, 1}},
			U64Span{5, 20},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{5, 15}, 2},
				ValueSpan{U64Span{15, 20}, 0},
			},
		},
		{"between",
			ValueSpanList{
				ValueSpan{U64Span{5, 10}, 1},
				ValueSpan{U64Span{25, 30}, 2},
			},
			U64Span{15, 20},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{5, 10}, 1},
				ValueSpan{U64Span{15, 20}, 0},
				ValueSpan{U64Span{25, 30}, 2},
			},
		},
		{"between match",
			ValueSpanList{
				ValueSpan{U64Span{5, 10}, 1},
				ValueSpan{U64Span{25, 30}, 2},
			},
			U64Span{10, 25},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{5, 10}, 1},
				ValueSpan{U64Span{10, 25}, 0},
				ValueSpan{U64Span{25, 30}, 2},
			},
		},
		{"merge intersection",
			ValueSpanList{
				ValueSpan{U64Span{5, 15}, 2},
				ValueSpan{U64Span{20, 30}, 2},
			},
			U64Span{10, 25},
			const1,
			ValueSpanList{
				ValueSpan{U64Span{5, 10}, 2},
				ValueSpan{U64Span{10, 25}, 1},
				ValueSpan{U64Span{25, 30}, 2},
			},
		},
		{"merge front",
			ValueSpanList{
				ValueSpan{U64Span{5, 15}, 1},
				ValueSpan{U64Span{20, 30}, 2},
			},
			U64Span{10, 25},
			const1,
			ValueSpanList{
				ValueSpan{U64Span{5, 25}, 1},
				ValueSpan{U64Span{25, 30}, 2},
			},
		},
		{"merge front match front",
			ValueSpanList{ValueSpan{U64Span{15, 25}, 0}},
			U64Span{10, 15},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{10, 25}, 0},
			},
		},
		{"merge end",
			ValueSpanList{
				ValueSpan{U64Span{5, 15}, 2},
				ValueSpan{U64Span{20, 30}, 1},
			},
			U64Span{10, 25},
			const1,
			ValueSpanList{
				ValueSpan{U64Span{5, 10}, 2},
				ValueSpan{U64Span{10, 30}, 1},
			},
		},
		{"merge end match front",
			ValueSpanList{ValueSpan{U64Span{5, 15}, 0}},
			U64Span{15, 20},
			add1,
			ValueSpanList{
				ValueSpan{U64Span{5, 20}, 0},
			},
		},
		{"merge union",
			ValueSpanList{
				ValueSpan{U64Span{5, 15}, 1},
				ValueSpan{U64Span{20, 30}, 1},
			},
			U64Span{10, 25},
			const1,
			ValueSpanList{
				ValueSpan{U64Span{5, 30}, 1},
			},
		},
		{"merge union match front",
			ValueSpanList{
				ValueSpan{U64Span{5, 15}, 1},
				ValueSpan{U64Span{20, 30}, 1},
			},
			U64Span{15, 25},
			const1,
			ValueSpanList{
				ValueSpan{U64Span{5, 30}, 1},
			},
		},
		{"merge union match end",
			ValueSpanList{
				ValueSpan{U64Span{5, 15}, 1},
				ValueSpan{U64Span{20, 30}, 1},
			},
			U64Span{10, 20},
			const1,
			ValueSpanList{
				ValueSpan{U64Span{5, 30}, 1},
			},
		},
		{"merge union match both",
			ValueSpanList{
				ValueSpan{U64Span{5, 15}, 1},
				ValueSpan{U64Span{20, 30}, 1},
			},
			U64Span{15, 20},
			const1,
			ValueSpanList{
				ValueSpan{U64Span{5, 30}, 1},
			},
		},
	} {
		ctx := log.Enter(ctx, test.name)
		Update(&test.list, test.span, test.f)
		assert.For(ctx, "list").ThatSlice(test.list).Equals(test.expected)
	}
}
