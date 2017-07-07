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

package evaluator

import (
	"strings"
	"testing"

	"github.com/google/gapid/gapis/api/gles/glsl/ast"
	"github.com/google/gapid/gapis/api/gles/glsl/parser"
)

// At this place in the dependency graph, we have all we need to fully test preprocessor #if
// expressions, so we run a few tests of that here.

var hashIfTests = []string{
	"1",
	"1 < 2",
	"1 < 2u",
	"A < 2",
	"!(2*3-6)",
	"4*5 == 60/3",
	"8-7 == 10%9",
	"defined(A)==1",
	"!defined(B)",
}

func TestHashIf(t *testing.T) {
	for _, test := range hashIfTests {
		test = `#define A 1
		#if ` + test + `
		// ok
		#else
		#error Test should have succeeded.
		#endif`

		t.Logf("Program: %s", test)
		_, _, _, err := parser.Parse(test, ast.LangVertexShader, EvaluatePreprocessorExpression)
		if len(err) > 0 {
			t.Errorf("Unexpected error parsing input: %s", err[0])
			continue
		}
	}
}

// Make sure that the tests above actually do report errors.
func TestFalsePreprocessorExpressionError(t *testing.T) {
	test := `#if 0
	// nothing
	#else
	#error This should be an error.
	#endif`

	t.Logf("Program: %s", test)
	_, _, _, err := parser.Parse(test, ast.LangVertexShader, EvaluatePreprocessorExpression)
	if len(err) == 0 {
		t.Errorf("Parsing unexpectedly succeeded.")
		return
	}
	if !strings.Contains(err[0].Error(), "This should be an error.") {
		t.Errorf("Unexpected error parsing input: %s", err[0])
	}
}

func TestHashIfUnknownSymbol(t *testing.T) {
	test := "#if foo bar baz\n#endif"
	_, _, _, err := parser.Parse(test, ast.LangVertexShader, EvaluatePreprocessorExpression)
	if len(err) == 0 {
		t.Errorf("Parsing unexpectedly succeeded.")
		return
	}
	expected := "Unknown symbol: 'foo'."
	if err[0].Error() != expected {
		t.Errorf("Unexpected error received: expected '%s', got '%s'.", expected, err[0])
	}
}

func TestHashIfUnknown(t *testing.T) {
	test := "#if 1 +\n#endif"
	_, _, _, err := parser.Parse(test, ast.LangVertexShader, EvaluatePreprocessorExpression)
	if len(err) == 0 {
		t.Errorf("Parsing unexpectedly succeeded.")
		return
	}
	expected := "1:8: Unexpected token (<nil>), was expecting one of: [primary expression]"
	if err[0].Error() != expected {
		t.Errorf("Unexpected error received: expected '%s', got '%s'.", expected, err[0])
	}
}
