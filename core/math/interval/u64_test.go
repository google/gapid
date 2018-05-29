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
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
)

func str(l U64SpanList) string {
	s := make([]string, len(l))
	for i, v := range l {
		s[i] = fmt.Sprintf("%d:%d", v.Start, v.End)
	}
	return "[" + strings.Join(s, ",") + "]"
}

func TestMerge(t *testing.T) {
	ctx := log.Testing(t)
	var (
		always           = 0x0
		whenJoinAdjTrue  = 0x1
		whenJoinAdjFalse = 0x2
	)

	for _, test := range []struct {
		name     string
		list     U64SpanList
		with     U64Span
		expected U64SpanList
		when     int
	}{
		{"Empty",
			U64SpanList{},
			U64Span{0, 0},
			U64SpanList{U64Span{0, 0}},
			always,
		},
		{"Duplicate",
			U64SpanList{U64Span{10, 10}},
			U64Span{10, 10},
			U64SpanList{U64Span{10, 10}},
			always,
		},
		{"Zero length duplicate",
			U64SpanList{U64Span{10, 0}},
			U64Span{10, 0},
			U64SpanList{U64Span{10, 0}},
			always,
		},
		{"between",
			U64SpanList{U64Span{0, 10}, U64Span{40, 50}},
			U64Span{20, 30},
			U64SpanList{U64Span{0, 10}, U64Span{20, 30}, U64Span{40, 50}},
			always,
		},
		{"before",
			U64SpanList{U64Span{10, 20}},
			U64Span{0, 5},
			U64SpanList{U64Span{0, 5}, U64Span{10, 20}},
			always,
		},
		{"after",
			U64SpanList{U64Span{0, 5}},
			U64Span{10, 20},
			U64SpanList{U64Span{0, 5}, U64Span{10, 20}},
			always,
		},
		{"touch before (joinAdj == false)",
			U64SpanList{U64Span{3, 5}},
			U64Span{0, 3},
			U64SpanList{U64Span{0, 3}, U64Span{3, 5}},
			whenJoinAdjFalse,
		},
		{"touch after (joinAdj == false)",
			U64SpanList{U64Span{3, 5}},
			U64Span{5, 7},
			U64SpanList{U64Span{3, 5}, U64Span{5, 7}},
			whenJoinAdjFalse,
		},
		{"touch before (joinAdj == true)",
			U64SpanList{U64Span{3, 5}},
			U64Span{0, 3},
			U64SpanList{U64Span{0, 5}},
			whenJoinAdjTrue,
		},
		{"touch after (joinAdj == true)",
			U64SpanList{U64Span{3, 5}},
			U64Span{5, 7},
			U64SpanList{U64Span{3, 7}},
			whenJoinAdjTrue,
		},
		{"extend before",
			U64SpanList{U64Span{3, 5}},
			U64Span{0, 4},
			U64SpanList{U64Span{0, 5}},
			always,
		},
		{"extend after",
			U64SpanList{U64Span{3, 5}},
			U64Span{4, 7},
			U64SpanList{U64Span{3, 7}},
			always,
		},
		{"extend middle before",
			U64SpanList{U64Span{0, 2}, U64Span{4, 6}, U64Span{8, 10}},
			U64Span{3, 5},
			U64SpanList{U64Span{0, 2}, U64Span{3, 6}, U64Span{8, 10}},
			always,
		},
		{"extend middle after",
			U64SpanList{U64Span{0, 2}, U64Span{4, 6}, U64Span{8, 10}},
			U64Span{5, 7},
			U64SpanList{U64Span{0, 2}, U64Span{4, 7}, U64Span{8, 10}},
			always,
		},
		{"inside start",
			U64SpanList{U64Span{10, 20}},
			U64Span{10, 11},
			U64SpanList{U64Span{10, 20}},
			always,
		},
		{"inside end",
			U64SpanList{U64Span{10, 20}},
			U64Span{19, 20},
			U64SpanList{U64Span{10, 20}},
			always,
		},
		{"merge first two",
			U64SpanList{U64Span{0, 10}, U64Span{20, 30}, U64Span{40, 50}},
			U64Span{5, 25},
			U64SpanList{U64Span{0, 30}, U64Span{40, 50}},
			always,
		},
		{"merge last two",
			U64SpanList{U64Span{0, 10}, U64Span{20, 30}, U64Span{40, 50}},
			U64Span{25, 45},
			U64SpanList{U64Span{0, 10}, U64Span{20, 50}},
			always,
		},
		{"merge overlap",
			U64SpanList{U64Span{0, 10}, U64Span{20, 30}, U64Span{40, 50}},
			U64Span{5, 45},
			U64SpanList{U64Span{0, 50}},
			always,
		},
		{"merge encompass",
			U64SpanList{U64Span{5, 10}, U64Span{20, 30}, U64Span{40, 45}},
			U64Span{0, 50},
			U64SpanList{U64Span{0, 50}},
			always,
		},
	} {
		ctx := log.Enter(ctx, test.name)
		if test.when == always || test.when == whenJoinAdjFalse {
			Merge(&test.list, test.with, false)
			assert.For(ctx, "true").ThatSlice(test.list).Equals(test.expected)
		}
		if test.when == always || test.when == whenJoinAdjTrue {
			Merge(&test.list, test.with, true)
			assert.For(ctx, "false").ThatSlice(test.list).Equals(test.expected)
		}
	}
}

func TestReplace(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []struct {
		name     string
		list     U64SpanList
		with     U64Span
		expected U64SpanList
	}{
		{"Empty",
			U64SpanList{},
			U64Span{0, 10},
			U64SpanList{U64Span{0, 10}},
		},
		{"match",
			U64SpanList{U64Span{5, 10}},
			U64Span{5, 10},
			U64SpanList{U64Span{5, 10}},
		},
		{"split",
			U64SpanList{U64Span{5, 25}},
			U64Span{15, 20},
			U64SpanList{U64Span{5, 15}, U64Span{15, 20}, U64Span{20, 25}},
		},
		{"split front",
			U64SpanList{U64Span{15, 25}},
			U64Span{10, 20},
			U64SpanList{U64Span{10, 20}, U64Span{20, 25}},
		},
		{"split end",
			U64SpanList{U64Span{5, 15}},
			U64Span{10, 20},
			U64SpanList{U64Span{5, 10}, U64Span{10, 20}},
		},
		{"between",
			U64SpanList{U64Span{5, 10}, U64Span{25, 30}},
			U64Span{15, 20},
			U64SpanList{U64Span{5, 10}, U64Span{15, 20}, U64Span{25, 30}},
		},
	} {
		ctx := log.Enter(ctx, test.name)
		Replace(&test.list, test.with)
		assert.For(ctx, "list").ThatSlice(test.list).Equals(test.expected)
	}
}

func TestRemove(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []struct {
		name     string
		list     U64SpanList
		with     U64Span
		expected U64SpanList
	}{
		{"empty",
			U64SpanList{},
			U64Span{0, 0},
			U64SpanList{},
		},
		{"match",
			U64SpanList{U64Span{3, 5}},
			U64Span{3, 5},
			U64SpanList{},
		},
		{"before",
			U64SpanList{U64Span{3, 5}},
			U64Span{0, 3},
			U64SpanList{U64Span{3, 5}},
		},
		{"after",
			U64SpanList{U64Span{3, 6}},
			U64Span{6, 7},
			U64SpanList{U64Span{3, 6}},
		},
		{"trim front",
			U64SpanList{U64Span{10, 20}},
			U64Span{5, 15},
			U64SpanList{U64Span{15, 20}},
		},
		{"split",
			U64SpanList{U64Span{10, 20}, U64Span{30, 40}, U64Span{50, 60}},
			U64Span{32, 38},
			U64SpanList{U64Span{10, 20}, U64Span{30, 32}, U64Span{38, 40}, U64Span{50, 60}},
		},
		{"trim 2",
			U64SpanList{U64Span{10, 20}, U64Span{30, 40}, U64Span{50, 60}},
			U64Span{35, 55},
			U64SpanList{U64Span{10, 20}, U64Span{30, 35}, U64Span{55, 60}},
		},
		{"all",
			U64SpanList{U64Span{10, 20}, U64Span{30, 40}, U64Span{50, 60}},
			U64Span{0, 100},
			U64SpanList{},
		},
	} {
		ctx := log.Enter(ctx, test.name)
		Remove(&test.list, test.with)
		assert.For(ctx, "list").ThatSlice(test.list).Equals(test.expected)
	}
}

func TestIntersect(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []struct {
		name   string
		list   U64SpanList
		with   U64Span
		offset int
		count  int
	}{
		{"empty",
			U64SpanList{},
			U64Span{0, 10},
			0, 0,
		},
		{"match",
			U64SpanList{U64Span{10, 20}},
			U64Span{10, 20},
			0, 1,
		},
		{"start",
			U64SpanList{U64Span{10, 20}},
			U64Span{10, 15},
			0, 1,
		},
		{"end",
			U64SpanList{U64Span{10, 20}},
			U64Span{15, 20},
			0, 1,
		},
		{"middle",
			U64SpanList{U64Span{10, 20}},
			U64Span{12, 18},
			0, 1,
		},
		{"overlap 3",
			U64SpanList{U64Span{10, 20}, U64Span{30, 40}, U64Span{50, 60}},
			U64Span{15, 55},
			0, 3,
		},
		{"first 2",
			U64SpanList{U64Span{10, 20}, U64Span{30, 40}, U64Span{50, 60}},
			U64Span{10, 35},
			0, 2,
		},
		{"last 2",
			U64SpanList{U64Span{10, 20}, U64Span{30, 40}, U64Span{50, 60}},
			U64Span{35, 60},
			1, 2,
		},
	} {
		ctx := log.Enter(ctx, test.name)
		o, c := Intersect(&test.list, test.with)
		assert.For(ctx, "Offset").That(o).Equals(test.offset)
		assert.For(ctx, "Count").That(c).Equals(test.count)
	}
}

func TestU64SpanListIndexOf(t *testing.T) {
	l := U64SpanList{U64Span{10, 20}, U64Span{30, 40}, U64Span{50, 60}}
	ctx := log.Testing(t)
	ctx = log.V{"List": l}.Bind(ctx)
	for _, test := range []struct {
		value uint64
		index int
	}{
		{0, -1},
		{9, -1},
		{10, 0},
		{15, 0},
		{19, 0},
		{20, -1},
		{32, 1},
		{59, 2},
	} {
		ctx := log.V{"Value": test.value}.Bind(ctx)
		got := IndexOf(&l, test.value)
		assert.For(ctx, "got").That(got).Equals(test.index)
	}
}

const maxIntervalValue = 100000
const maxIntervalRange = 10000

type iteration struct{ merge, replace U64Span }

func buildRands(b *testing.B) []iteration {
	b.StopTimer()
	defer b.StartTimer()
	iterations := make([]iteration, b.N)
	rand.Seed(1)
	for i := range iterations {
		iterations[i].merge = U64Range{
			First: uint64(rand.Intn(maxIntervalValue)),
			Count: uint64(rand.Intn(maxIntervalValue))}.Span()
		iterations[i].replace = U64Range{
			First: uint64(rand.Intn(maxIntervalValue)),
			Count: uint64(rand.Intn(maxIntervalValue))}.Span()
	}
	return iterations
}

func BenchmarkGeneral(b *testing.B) {
	iterations := buildRands(b)
	l := U64SpanList{}
	for _, iter := range iterations {
		Merge(&l, iter.merge, false)
		Replace(&l, iter.replace)
	}
}
