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

package parser_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/core/text/parse/test"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/parser"
)

var (
	B = test.Branch
	L = test.Leaf
	N = test.Node
)

func TestParsedCST(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		name     string
		expected cst.Node
		source   string
		errors   parse.ErrorList
	}{
		{
			name:   "const int*",
			source: `const int* a`,
			expected: B(B(
				N(nil, B(
					N(nil, "const", " "),
					B(B(L("int")), L("*")),
				), " ",
				),
				L("a"),
			)),
		},
		{
			name:   "const char* const *",
			source: `const char* const * a`,
			expected: B(B(
				B(
					N(nil, B(
						N(nil, "const", " "),
						B(
							B(L("char")),
							L("*"),
						),
					), " "),
					N(nil, L("const"), " "),
					N(nil, L("*"), " "),
				),
				L("a"),
			)),
		},
	} {
		m := &ast.Mappings{}
		api, errs := parser.Parse("parser_test.api", test.source, m)
		assert.For(test.name).ThatSlice(errs).Equals(test.errors)
		assert.For(test.name).That(m.CST(api)).DeepEquals(test.expected)
	}
}
