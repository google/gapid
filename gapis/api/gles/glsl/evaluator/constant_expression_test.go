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
	"regexp"
	"strings"
	"testing"

	"github.com/google/gapid/gapis/api/gles/glsl/ast"
	"github.com/google/gapid/gapis/api/gles/glsl/parser"
	"github.com/google/gapid/gapis/api/gles/glsl/sema"
)

var constantExpressions = []string{
	"1 <<2u== 4",
	"1u<<2u== 4u",
	"5 >>2u== 1",
	"5u>>2u== 1u",

	"1 < 2",
	"1 <= 1",
	"2u > 1u",
	"1u >= 1u",
	"1. < 2.",
	"-1. >= -2.",
	"1 != 2",
	"1u != 2u",
	"1. != 2.",

	"bool(1) == true",
	"bvec4(1, 0, -1, false) == bvec4(true, false, true, false)",
	"bvec4(1u, 0u, -1u, true) == bvec4(true, false, true, true)",
	"bvec4(1., 0., -1., 1e-7) == bvec4(true, false, true, true)",
	"uvec2(true, false) == uvec2(1u, 0u)",
	"uvec4(1, 0, -1, 47) == uvec4(1u, 0u, -1u, 47u)",
	"uvec4(1., 0., -1., 1e-7) == uvec4(1u, 0u, -1u, 0u)",
	"ivec2(true, false) == ivec2(1, 0)",
	"ivec4(1u, 0u, -1u, 47u) == ivec4(1, 0, -1, 47)",
	"ivec4(1., 0., -1., 1e-7) == ivec4(1, 0, -1, 0)",
	"vec2(true, false) == vec2(1., 0.)",
	"vec4(1u, 0u, -1u, 47u) == vec4(1., 0., 4294967295., 47.)",
	"vec4(1, 0, -1, 47) == vec4(1., 0., -1., 47.)",
	"-vec3(1) == vec3(-1)",
	"-mat4(1) == mat4(-1)",

	"vec4(vec3(1,2,3), 4) == vec4(1,2,3,4)",
	"vec4(mat2(1)) == vec4(1,0,0,1)",
	"vec4(mat3(1)) == vec4(1,0,0,0)",
	"mat2(vec4(1,2,3,4)) == mat2(1,2,3,4)",
	"mat3(vec4(1,2,3,4), vec3(5,6,7), vec4(8,9,10,11)) == mat3(1,2,3,4,5,6,7,8,9)",
	"mat4(mat2(1,2,3,4)) == mat4(1,2,0,0 , 3,4,0,0 , 0,0,1,0 , 0,0,0,1)",
	"mat2(mat3(1,2,3,4,5,6,7,8,9)) == mat2(1,2,4,5)",
	"float(vec2(1,2)) == 1.",

	// struct ST { int i; uint u; float f; bool b; };
	"ST(1, 1u, 1., true) == ST(1, 1u, 1., true)",
	"ST(1, 1u, 1., true).i == 1",
	"ST(1, 1u, 1., true).u == 1u",
	"ST(1, 1u, 1., true).f == 1.",
	"ST(1, 1u, 1., true).b == true",

	"mat4(1)[1] == vec4(0,1,0,0)",
	"vec4(1,2,3,4)[1] == 2.",
	"mat2(1,2,3,4)[1][1] == 4.",
	"mat2(1,2,3,4)[1u][1u] == 4.",
	"int[2](1,2)[1] == 2",
	"int[](1,2,3,4)[3] == 4",
	"int[](1,2,3,4).length() == 4u",

	"ivec2(1,2).x == 1",
	"ivec2(1,2).r == 1",
	"ivec2(1,2).s == 1",
	"ivec2(1,2).xx == ivec2(1)",
	"ivec2(1,2).xxyy == ivec4(1,1,2,2)",
	"ivec4(1,2,3,4).xw == ivec2(1,4)",
	"ivec4(1,2,3,4).wzyx == ivec4(4,3,2,1)",
	"ivec4(1,2,3,4).xz.yyxx == ivec4(3,3,1,1)",
	"ivec3(1,2,3).z == 3",

	"mat2(1.)+1. == mat2(2.,1.,1.,2.)",
	"ivec2(1)+1 == ivec2(2)",
	"1u+uvec2(1) == uvec2(2)",

	"(6&3) == 2",
	"(6u&3u) == 2u",
	"(6|3) == 7",
	"(6u|3u) == 7u",
	"(6^3) == 5",
	"(6u^3u) == 5u",
	"7/2 == 3",
	"7u/2u == 3u",
	"7./2. == 3.5",
	"8%3 == 2",
	"8u%3u == 2u",
	"4*7==28",
	"4u*7u==28u",
	"4.*7.==28.",
	"4-7==-3",
	"4u-7u==-3u",
	"4.-7.==-3.",
	"+1 == 1",
	"~0==0xffffffff",
	"~0u==0xffffffffu",
	"vec3(1) * mat2x3(1,1,1,1,1,1) == vec2(3)",
	"mat2x3(1,1,1,1,1,1) * vec2(1) == vec3(2)",
	"mat2x3(1,1,1,1,1,1) * mat3x2(1,1,1,1,1,1) == mat3(2,2,2,2,2,2,2,2,2)",
	"true || false",
	"false || true",
	"true && true",
	"!(false && true)",
	"!(true && false)",
	"false ^^ true",
	"!(true ^^ true)",
	"false ? false : true",

	// const int VAR = 1;
	"VAR == 1",
}

func TestConstantExpressions(t *testing.T) {
	for _, test := range constantExpressions {
		test = "bool x[(" + test + ")?1:-1];"
		if strings.Contains(test, "ST") {
			test = "struct ST { int i; uint u; float f; bool b; }; " + test
		}
		if strings.Contains(test, "VAR") {
			test = "const int VAR = 1; " + test
		}
		t.Logf("Program: %s", test)
		program, _, _, err := parser.Parse(test, ast.LangVertexShader, EvaluatePreprocessorExpression)
		if len(err) > 0 {
			t.Errorf("Unexpected error parsing input: %s", err[0])
			continue
		}

		err = sema.Analyze(program, Evaluate)

		if len(err) > 0 {
			t.Errorf("Unexpected sema error: %s", err[0])
		}
	}
}

// Make sure that the tests above actually do report errors.
func TestNegativeArraySizeError(t *testing.T) {
	test := "bool x[(1 == -1) ? 1 : -1];"
	t.Logf("Program: %s", test)

	program, _, _, err := parser.Parse(test, ast.LangVertexShader, EvaluatePreprocessorExpression)
	if len(err) > 0 {
		t.Errorf("Unexpected error parsing input: %s", err[0])
		return
	}

	err = sema.Analyze(program, Evaluate)
	if len(err) == 0 {
		t.Errorf("Semantic check unexpectedly succeeded.")
	}
	if err[0].Error() != "Array size must be positive." {
		t.Errorf("Unexpected kind of sema error: %s", err[0])
	}
}

var constantExpressionErrors = []struct {
	expr string
	err  string
}{
	{"mat4(1)[5]==vec4(1)", "Index 5 is out of range for a value of type 'mat4x4'"},
	{"mat4(1)[5u]==vec4(1)", "Index 5 is out of range for a value of type 'mat4x4'"},
}

func TestConstantExpressionErrors(t *testing.T) {
	for _, test := range constantExpressionErrors {
		expr := "bool x[(" + test.expr + ")?1:-1];"
		t.Logf("Program: %s", expr)
		program, _, _, err := parser.Parse(expr, ast.LangVertexShader, EvaluatePreprocessorExpression)
		if len(err) > 0 {
			t.Errorf("Unexpected error parsing input: %s", err[0])
			continue
		}

		err = sema.Analyze(program, Evaluate)
		if len(err) == 0 {
			t.Errorf("Sema unexpectedly succeeded.")
			continue
		}

		matched, regerr := regexp.MatchString(test.err, err[0].Error())
		if regerr != nil {
			t.Errorf("Error matching regexp failed: %s", regerr)
			continue
		}
		if !matched {
			t.Errorf("Unexpected sema error received. Expected '%v', got '%v'.", test.err, err[0])
			continue
		}
	}
}
