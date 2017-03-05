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
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/gapil/analysis"
)

func TestU32ValueEquals(t *testing.T) {
	assert := assert.To(t)

	for i, test := range []struct {
		a, b     analysis.Value
		expected analysis.Possibility
	}{
		{u32(0), u32(0), T},
		{u32(1), u32(1), T},
		{u32(1), u32(2), F},
		{u32(1, 2), u32(2), M},
		{u32(1, 2), u32(1, 2), M},
		{u32(1, 2), u32(3, 4), F},
	} {
		got := test.a.Equals(test.b)
		assert.For("test %d", i).That(got).Equals(test.expected)
	}
}

func TestU32SetRelational(t *testing.T) {
	assert := assert.To(t)

	ops := map[string]func(a analysis.SetRelational, b analysis.Value) analysis.Value{
		"<":  analysis.SetRelational.SetLessThan,
		"<=": analysis.SetRelational.SetLessEqual,
		">":  analysis.SetRelational.SetGreaterThan,
		">=": analysis.SetRelational.SetGreaterEqual,
	}
	for _, test := range []struct {
		a        analysis.Value
		op       string
		b        analysis.Value
		expected analysis.Value
	}{
		{u32(0, 1, 2), "<", u32(0, 1, 2), u32(0, 1)},
		{u32(1, 2, 3), "<", u32(1, 2, 3), u32(1, 2)},
		{u32(1, 2, 3), "<", u32(7, 8, 9), u32(1, 2, 3)},
		{u32(7, 8, 9), "<", u32(1, 2, 3), u32()},

		{u32(0, 1, 2), "<=", u32(0, 1, 2), u32(0, 1, 2)},
		{u32(1, 2, 3), "<=", u32(1, 2, 3), u32(1, 2, 3)},
		{u32(1, 2, 3), "<=", u32(7, 8, 9), u32(1, 2, 3)},
		{u32(7, 8, 9), "<=", u32(1, 2, 3), u32()},

		{u32(0, 1, 2), ">", u32(0, 1, 2), u32(1, 2)},
		{u32(1, 2, 3), ">", u32(1, 2, 3), u32(2, 3)},
		{u32(1, 2, 3), ">", u32(7, 8, 9), u32()},
		{u32(7, 8, 9), ">", u32(1, 2, 3), u32(7, 8, 9)},

		{u32(0, 1, 2), ">=", u32(0, 1, 2), u32(0, 1, 2)},
		{u32(1, 2, 3), ">=", u32(1, 2, 3), u32(1, 2, 3)},
		{u32(1, 2, 3), ">=", u32(7, 8, 9), u32()},
		{u32(7, 8, 9), ">=", u32(1, 2, 3), u32(7, 8, 9)},
	} {
		got := ops[test.op](test.a.(analysis.SetRelational), test.b)
		assert.For("%v %s %v", test.a, test.op, test.b).That(fmt.Sprint(got)).Equals(fmt.Sprint(test.expected))
	}
}
