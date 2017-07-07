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

package parser

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/google/gapid/gapis/api/gles/glsl/ast"
	pp "github.com/google/gapid/gapis/api/gles/glsl/preprocessor"
)

func fun(body string) string { return "int fun(int a) { " + body + " }" }

type testInfo struct {
	Program, Error string
}

var shaderBuilderTests = []testInfo{
	{"int a;", ""},
	{"int ,a;", ""},
	{"int;", ""},
	{"int[] a = int[](1,2);", ""},

	{"int float;", `Unexpected token \(float\)`},
	{"int a[2;", `Unexpected token \(;\), was expecting one of: \[\]\]`},
	{"int a[2]{", `Unexpected token \({\), was expecting one of: \[, ;\]`},
	{"int a[2],int", `Unexpected token \(int\), was expecting one of: \[identifier\]`},

	{"precision highp int;", ""},
	{"lowp int[2] a;", ""},
	{"precision lowp void;", `Invalid type for a precision declaration.`},
	{"struct A { int a; }; lowp A a;", `Type A cannot be used with a precision specifier.`},

	{"mediump int foo(int a);", ""},
	{"int foo(int;", `Unexpected token \(;\), was expecting one of: \[, \)\]`},
	{"int foo(int a) { a; }", ""},

	{fun("{}"), ""},
	{fun("if(a) {}"), ""},
	{"void foo() { if,", `Unexpected token \(,\), was expecting one of: \[\(\]`},

	{fun("if(a) {} else {}"), ""},
	{fun("while(a) ;"), ""},
	{"void foo() { while,", `Unexpected token \(,\), was expecting one of: \[\(\]`},

	{fun("do { } while(a);"), ""},
	{"void foo() { do ; ,", `Unexpected token \(,\), was expecting one of: \[while\]`},
	{"void foo() { do ; while,", `Unexpected token \(,\), was expecting one of: \[\(\]`},

	{fun("for(a;a;a) ;"), ""},
	{fun("for(int a = 0;a;a) ;"), ""},
	{fun("for(a;bool a=false;a) ;"), ""},
	{"void foo() { for,", `Unexpected token \(,\), was expecting one of: \[\(\]`},
	{"void foo() { for(0;0]", `Unexpected token \(\]\), was expecting one of: \[;\]`},
	{"void foo() { for(0;int a;", `Unexpected token \(;\), was expecting one of: \[=\]`},

	{"void foo() { for(0;int float;",
		`Unexpected token \(float\), was expecting one of: \[identifier\]`},

	{fun("switch(a) { case a: default: }"), ""},
	{"void foo() { switch", `Unexpected token \(<nil>\), was expecting one of: \[\(\]`},

	{fun("while(a) { continue; }"), ""},
	{fun("while(a) { break; }"), ""},

	{fun("discard;"), ""},
	{fun("return;"), ""},
	{fun("return a;"), ""},
	{fun("int b;"), ""},

	{"precision lowp int;", ""},

	{fun("fun(); fun(a); fun(a,a);"), ""},
	{"void foo() { foo(void); }", ""},
	{"void foo() { foo(1}", `Unexpected token \(}\), was expecting one of: \[, \)\]`},
	{"void foo(int a) { a(); }", `'a' is not a function`},
	{"void foo(int a) { foo; }", `'foo' is a function`},

	{fun("a = a;"), ""},
	{fun("a = 1 + 2;"), ""},
	{fun("1 + 2 = a = 3 + 4;"), ""},
	{fun("a -= 1;"), ""},
	{fun("a ? a : a;"), ""},
	{fun("a = a?a:a;"), ""},
	{fun("a?a:a=a;"), ""},
	{fun("a?a=a:a;"), ""},
	{fun("a+a;"), ""},
	{fun("a/a%a;"), ""},
	{fun("a^a || a!=a;"), ""},
	{fun("a!=a * a==a;"), ""},
	{fun("++a;"), ""},
	{fun("!a;"), ""},
	{fun("+--++a;"), ""},
	{fun("a++;"), ""},
	{fun("a++--;"), ""},
	{fun("++a--++;"), ""},
	{fun("a++[0]++;"), ""},
	{fun("1; 1.0; false; true; 1u;"), ""},
	{fun("a,a,a;"), ""},
	{fun("int(a);"), ""},
	{fun("highp int(a);"), ""},
	{"void foo() { 1?2;", `Unexpected token \(;\), was expecting one of: \[:\]`},
	{"void foo() { a; }", `Undeclared identifier 'a'`},
	{fun("sqrt(1.0);"), ""},

	{"struct A { int a; }; A a = A(1);", ""},
	{"int a, b[], c[3], d;", ""},
	{"int a = 0, b[2] = (1,2);", ""},
	{"int foo(in int a, out int b, inout int c, int d, const int e, const inout int f);", ""},
	{"struct A { int a; };", ""},
	{"struct A { int a; } a, b;", ""},
	{"struct A { int a; }; A a,b[2];", ""},
	{"struct A { const int a; };", ""},
	{fun("a.bar();"), ""},

	{"struct A { int a; }; void foo(A a) { a.int",
		`Unexpected token \(int\), was expecting one of: \[identifier\]`},

	{"layout(location=2) int a;", ""},
	{"layout(location=2u, col_major) uniform;", ""},
	{"layout(location=~) uniform;",
		`Unexpected token \(~\), was expecting one of: \[intconstant uintconstant\]`},
	{"layout(location=2-) uniform;",
		`Unexpected token \(-\), was expecting one of: \[, \)\]`},

	{"int a,b; invariant a,b;", ""},
	{"int a,b; invariant a, float;", `Unexpected token \(float\)`},
	{"int b; invariant a;", `Undeclared identifier 'a'`},

	{"uniform A { int a; };", ""},
	{"layout(row_major) uniform A { int a; };", ""},
	{"uniform A { int a; } a[47];", ""},
	{"uniform A;", `Unexpected token \(;\), was expecting one of: \[{\]`},
	{"int a; uniform A { int b; } a;", `Declaration of 'a' already exists`},

	{"void foo() { }", ""},
	{"void foo(int a, float b) { }", ""},
	{"void foo(void) { }", ""},
	{"void foo(int, float);", ""},
	{"void foo(int[7] a, float b[4]);", ""},
	{"const void foo();", "Type qualifiers are not allowed on function return types."},

	{"struct { int a; } foo();",
		"Structure definitions are not allowed in function return values."},

	{"void foo(struct { int a; } a);",
		"Structure definitions are not allowed in function prototypes."},

	{"void foo(const in bar",
		`Unexpected token \(bar\), was expecting one of: \[type struct\]`},

	{"void foo(int) { }", "Formal parameter 1 of function 'foo' lacks a name."},
	{"void foo(int i) i;", `Unexpected token \(i\), was expecting one of: \[{\]`},
	{"void foo(int[] a);", `Array size must be specified for function parameters.`},
	{"void foo(int[2] a[2]);", `Declaring an array of arrays.`},

	{"smooth sampler2D a; flat isampler3D b;", ""},
	{"centroid in int a; centroid out float b;", ""},
	{"const int a; in float b; out int c;", ""},
	{"layout(column_major) const foo;", "Invalid combination of qualifiers:"},
	{"invariant invariant int a;", `Invariant qualifier specified twice.`},

	{"layout(column_major) const;",
		"Invalid combination of qualifiers for a layout qualifier declaration."},

	{"layout(row_major) invariant int a;",
		`'invariant' cannot be combined with layout qualifiers.`},

	{"smooth invariant int a;",
		`'invariant' must go before interpolation and storage qualifiers.`},

	{"layout(row_major) smooth int a;",
		`Interpolation qualifiers cannot be combined with layout qualifiers.`},

	{"smooth layout(row_major) int a;",
		`'layout' cannot be combined with interpolation and invariant qualifiers.`},

	{"const smooth int a;", `Interpolation qualifiers must go before storage qualifiers.`},
	{"smooth flat int a;", `Interpolation qualifier specified twice.`},
	{"layout(row_major) layout(column_major) int a;", `Layout qualifier specified twice.`},
	{"int a; float a;", "Declaration of 'a' already exists"},
	{"void foo(int); void foo(float);", ""},
}

func testBuilder(t *testing.T, lang ast.Language, tests []testInfo) {
	for _, test := range tests {
		expectedInfos, _, err := pp.Preprocess(test.Program, nil, 0)
		if len(err) > 0 {
			t.Logf("Program: %s", test.Program)
			t.Errorf("Unexpected preprocessor error: %s", err[0])
			continue
		}
		ast, _, _, obtainedErrors := Parse(test.Program, lang, nil)
		if test.Error != "" {
			if len(obtainedErrors) == 0 {
				t.Logf("Program: %s", test.Program)
				t.Errorf("Parsing unexpectedly succeeded.")
				continue
			}

			matched, err := regexp.MatchString(test.Error, obtainedErrors[0].Error())
			if err != nil {
				t.Errorf("Error matching regexp failed: %s", err)
				continue
			}

			if !matched {
				t.Logf("Program: %s", test.Program)
				t.Logf("Expected error: %s", test.Error)
				t.Logf("Obtained error: %s", obtainedErrors[0])
				t.Error("Incorrect parsing error received.")
			}
		} else {
			if len(obtainedErrors) > 0 {
				t.Logf("Program: %s", test.Program)
				t.Errorf("Error parsing input: %s", obtainedErrors[0])
				continue
			}

			formatted := fmt.Sprint(Formatter(ast))
			obtainedInfos, _, err := pp.Preprocess(formatted, nil, 0)
			if len(err) > 0 {
				t.Logf("Program:        %s", test.Program)
				t.Logf("Builder output: %s", formatted)
				t.Errorf("Unexpected round-trip preprocessor error: %s", err[0])
				continue
			}

			bad := len(obtainedInfos) != len(expectedInfos)
			if !bad {
				for i := range expectedInfos {
					if obtainedInfos[i].Token != expectedInfos[i].Token {
						bad = true
						break
					}
				}
			}

			if bad {
				t.Logf("Program:        %s", test.Program)
				t.Logf("Builder output: %s", formatted)
				t.Error("Ast builder round-trip mismatch.")
			}
		}
	}
}

func TestBuilderShader(t *testing.T) {
	testBuilder(t, ast.LangVertexShader, shaderBuilderTests)
}

var preprocessorBuilderTests = []testInfo{
	{"1+2*3", ""},
	{"1+A", ""},
	{"(2-5)", ""},
	{"4.7", `Unexpected token \(4.7\)`},
	{"1+", `Unexpected token \(<nil>\), was expecting one of: \[primary expression\]`},
}

func TestBuilderPreprocessor(t *testing.T) {
	testBuilder(t, ast.LangPreprocessor, preprocessorBuilderTests)
}
