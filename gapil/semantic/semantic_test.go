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

package semantic_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/gapil/semantic"
)

type replaceTest struct {
	name     string
	from     int
	count    int
	with     semantic.Statements
	expected semantic.Statements
}

func S(x int) semantic.Statement {
	return &semantic.Return{Value: semantic.Int64Value(x)}
}

func L(x ...int) semantic.Statements {
	statements := make(semantic.Statements, len(x))
	for i, v := range x {
		statements[i] = S(v)
	}
	return semantic.Statements(statements)
}

func TestStatementsReplace(t *testing.T) {
	assert := assert.To(t)

	tests := []replaceTest{
		{"none", 4, 0, L(), L(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)},
		{"single", 4, 1, L(99), L(0, 1, 2, 3, 99, 5, 6, 7, 8, 9)},
		{"remove", 4, 1, L(), L(0, 1, 2, 3, 5, 6, 7, 8, 9)},
		{"insert", 4, 0, L(99), L(0, 1, 2, 3, 99, 4, 5, 6, 7, 8, 9)},
		{"many", 4, 2, L(70, 80, 90), L(0, 1, 2, 3, 70, 80, 90, 6, 7, 8, 9)},
	}

	assert.For("With cap == len").Log()
	for _, test := range tests {
		s := L(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
		s.Replace(test.from, test.count, test.with...)
		assert.For("%s statements", test.name).That(s).DeepEquals(test.expected)
	}

	assert.For("With cap > len").Log()
	for _, test := range tests {
		s := make(semantic.Statements, 50)[:10]
		copy(s, L(0, 1, 2, 3, 4, 5, 6, 7, 8, 9))
		s.Replace(test.from, test.count, test.with...)
		assert.For("%s statements", test.name).That(s).DeepEquals(test.expected)
	}
}

func TestStatementsRemove(t *testing.T) {
	assert := assert.To(t)

	all := L(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)

	for _, test := range []struct {
		n        semantic.Statement
		expected semantic.Statements
	}{
		{nil, L(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)},
		{all[5], L(0, 1, 2, 3, 4, 6, 7, 8, 9)},
		{all[9], L(0, 1, 2, 3, 4, 5, 6, 7, 8)},
	} {
		s := append(semantic.Statements{}, all...)
		s.Remove(test.n)
		assert.For("statements %v", test.n).That(s).DeepEquals(test.expected)
	}
}
