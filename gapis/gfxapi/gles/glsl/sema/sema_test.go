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

package sema

import (
	"regexp"
	"testing"

	"github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl/evaluator"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl/parser"
)

var semaTests = []struct {
	program, cerr string
}{
	{"vec3 x = vec2(1.0) * mat3x2(1.0);", ""},
	{"vec2 x = mat3x2(1.0) * vec3(1.0);", ""},
	{"vec2 x = vec2(1.0) * float(1.0);", ""},
	{"ivec3 x = 1 * ivec3(1.0);", ""},
	{"uvec4 x = uvec4(1.0) + uvec4(1.0);", ""},
	{"mat2 x = mat2(1.0) - mat2(1.0);", ""},
	{"mat3x2 x; mat3x2 y = x *= mat3(1.0);", ""},
	{"int x = 1 % 1;", ""},
	{"bool x = 1 < 1;", ""},
	{"bool x = mat4(1.0) == mat4(1.0);", ""},
	{"bool x = true && false;", ""},
	{"int x = 1 << 1u;", ""},
	{"float x = (true, 1.0);", ""},
	{"int x = 1 & 1;", ""},
	{"int x; int y = x = 1;", ""},

	{"int x; int y = x++;", ""},
	{"bool x = !true;", ""},
	{"int x = ~0;", ""},
	{"int x = false?1:0;", ""},
	{"float[2] x; float y = x[1];", ""},
	{"mat2 x; vec2 y = x[1];", ""},
	{"vec2 x; float y = x[1];", ""},

	{"int[2] x; uint y = x.length();", ""},
	{"struct { int foo; } x; uint y = x.foo();", "Cannot call method 'foo' of type 'int'"},
	{"struct { int a; } x; int y = x.a;", ""},
	{"vec2 x; float y = x.x;", ""},

	{"vec2 x; mat2 y = mat2(1.0,x,1.0);", ""},
	{"struct A { int a, b; }; A x = A(1, 1);", ""},
	{"int[2u] x = int[2](1, 1);", ""},
	{"int[] x = int[2](1, 1);", ""},
	{"int[] x = int[](1, 1);", ""},
	{"mat3 y; mat2 x = mat2(y);", ""},

	{"const int x = 1;", ""},
	{"const int x = 1; int[x] y = int[1](1);", ""},

	{"struct Bar { int i; }; void Foo(Bar Bar) {}", ""},

	{"int[2] x[2];", "declared as an array of arrays"},
	{"mat2 x = mat3(1.0);", "initialized with an expression of incompatible type"},
	{"const int x;", "must be initialized"},
	{"int x; const int y = x;", "initialized with a non-constant expression"},

	{"int x = 1?1:1;", "Incompatible condition argument"},
	{"int x = true?1:1.0;", "Expression arguments of operator '\\?:' have different types"},
	{"int[2] x; int y = x[false];", "Cannot index with an expression of type"},
	{"int x = 2[2];", "Base argument '2' \\(type 'int'\\) is not indexable."},
	{"mat2 x = mat3(1)*mat4(1);", "Invalid operand sizes for matrix multiplication:"},
	{"int x = vec2(1)+vec3(1);", "Arguments of incompatible types in expression"},
	{"int x = 1 = 2;", "Expression '1' is not a lvalue."},
	{"int x = 1u % 2;", "Incompatible fundamental types for operation '%'"},
	{"struct { int x; } x; int y = x+1;", "Incompatible left argument 'x'"},
	{"struct { int x; } x; int y = 1+x;", "Incompatible right argument 'x'"},
	{"bool x = 1 < 1.0;", "Incompatible types for operation '<'"},
	{"mat2 x; mat2 y = x *= vec2(1);", "Assignment from an incompatible type: 'mat2' vs. 'vec2'"},
	{"ivec2 x = ivec2(1) % ivec3(1);", "Operation '%' cannot operate on vectors of different size."},
	{"bool x = mat2(1) < 1.0;", "Non-scalar left operand type 'mat2'"},
	{"bool x = 1.0 > mat2(1);", "Non-scalar right operand type 'mat2'"},
	{"bool x = 1 == 1.0;", "Arguments of incompatible types"},
	{"bool x = 1 && true;", "Incompatible non-boolean left argument '1' of type 'int'"},
	{"bool x = true || 1;", "Incompatible non-boolean right argument '1' of type 'int'"},
	{"int x = 1 << 1;", "Right operand type of operation '<<' is not unsigned integer"},
	{"int x = 1.0 << 1u;", "Non-integral left operand of operation '<<'"},
	{"ivec2 x = 1 << uvec2(1);", "Arguments of incompatible types"},
	{"ivec2 x = ivec2(1) & ivec3(1);", "Arguments of incompatible types"},

	{"int x = ++1;", "Expression '1' is not a lvalue."},
	{"struct A { int x; } x; bool y = x--;", "Incompatible argument 'x'"},
	{"bool x; bool y = x--;", "Incompatible argument 'x' of type 'bool' for operation '--'"},
	{"bool x = !1;", "Incompatible argument '1' of type 'int' for operation '!'"},
	{"bool x = ~true;", "Incompatible argument 'true' of type 'bool' for operation '~'"},

	{"int[2] x; int y = x.foo();", "Expression 'x' of type 'int\\[2\\]' has no member 'foo'."},
	{"struct A { int x; }; A x; int y = x.foo;", "Expression 'x' of type 'A' has no member 'foo'."},
	{"float x = mat2(1).x;", "Illegal type 'mat2' for the dot operator."},
	{"int x = ivec2(1).rx;", "Swizzle sequence 'rx' contains characters from multiple coordinate sets."},
	{"int x = ivec2(1).z;", "Cannot access component 'z' of a vector of size 2."},
	{"ivec2 x; ivec2 y = x.xx += 1;", "Expression 'x.xx' is not a lvalue."},
	{"int x = ivec2(1).c;", "Invalid character 'c' for vector swizzle."},
	{"int x = ivec2(1).xxxxx;", "Swizzle result must have between 1 and 4 elements."},

	{"int x; int y[x];", "Array size expression 'x' is not constant."},
	{"int x[false];", "Invalid type 'bool' of array size expression 'false'."},
	{"int x[0u];", "Cannot construct an array of size zero."},
	{"int x[-1];", "Array size must be positive."},

	{"sampler2D x = sampler2D(1);", "Cannot construct an object of type 'sampler2D'."},
	{"sampler2D x; int y = int(x);", "Cannot construct an object of type 'int' using 'x'"},
	{"int x = int(1,1);", "Too many components for constructing an object of type 'int'."},
	{"mat2 x = mat2(1, mat2(1));", "Constructor for a matrix type 'mat2' can only take one matrix"},
	{"mat2 x = mat2(1, 1);", "Not enough arguments for construction of an object"},

	{"struct A { int x; }; A x = A();", "Not enough arguments to construct struct 'A'."},
	{"struct A { int x; }; A x = A(1.0);", "Cannot initialize 'A.x' \\(type 'int'\\)"},
	{"struct A { int x; }; A x = A(1, 1);", "Too many arguments for construction of struct 'A'."},

	{"int[] x = int[1](1.0);", "Cannot construct an array of type 'int\\[1\\]' from a member"},
	{"int[] x = int[1](1, 1);", "Array of size 1 constructed with an incorrect number"},
	{"int[] x = int[]();", "Cannot construct an array of size zero."},

	{"const int[] x = int[2](1,1); const int y = x[3];", "evaluating constant expression.*out of range"},
	{"const int[] x = int[2](1,1); int y[x[3]];", "evaluating constant expression.*out of range"},
}

func TestSema(t *testing.T) {
	for _, test := range semaTests {
		t.Logf("Program: %s", test.program)
		ast, _, _, err := parser.Parse(test.program, ast.LangVertexShader,
			evaluator.EvaluatePreprocessorExpression)
		if len(err) > 0 {
			t.Errorf("Unexpected error parsing input: %s", err[0])
			continue
		}

		err = Analyze(ast, evaluator.Evaluate)

		if test.cerr == "" {
			if len(err) > 0 {
				t.Errorf("Unexpected sema error: %s", err[0])
			}
		} else {
			matched, regerr := regexp.MatchString(test.cerr, err[0].Error())
			if regerr != nil {
				t.Errorf("Error matching regexp failed: %s", regerr)
				continue
			}
			if !matched {
				t.Errorf("Incorrect sema error received. Expected '%v', got '%v'.",
					test.cerr, err[0])
			}
		}
	}
}
