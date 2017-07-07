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
	"strings"
	"testing"

	"github.com/google/gapid/gapis/api/gles/glsl/ast"
)

func fmtFun(input string) string {
	input = "\n" + input
	input = strings.Replace(input, "\n", "\n\t", -1)
	return "int foo(int a) {" + input + "\n}\n"
}

var fmtTests = []string{
	"int a;\n",
	"int a;\nint b;\n",
	"int, a;\n",
	"int;\n",
	"int[] a = int[](1, 2);\n",

	"precision highp int;\n",
	"lowp int[2] a;\n",
	"mediump int foo(int a);\n",

	"int foo(int a) {\n    a;\n}\n",
	fmtFun("{\n}"),
	fmtFun("{\n\tint a;\n}"),

	fmtFun("if(a) {\n}"),
	fmtFun("if(a) {\n} else {\n}"),
	fmtFun("if(a)\n\ta;\nelse {\n}"),

	fmtFun("while(a) ;"),
	fmtFun("do {\n} while(a);"),

	fmtFun("for(a; a; a) ;"),
	fmtFun("for(int a = 0; a; a) ;"),
	fmtFun("for(a; bool a = false; a) ;"),

	fmtFun("switch(a) {\n\tcase a:\n\tdefault:\n}"),

	fmtFun("while(a) {\n\tcontinue;\n}"),
	fmtFun("while(a) {\n\tbreak;\n}"),

	fmtFun("discard;"),
	fmtFun("return;"),
	fmtFun("return a;"),

	fmtFun("int b;"),
	fmtFun("precision lowp int;"),

	fmtFun("foo();\nfoo(a);\nfoo(a, a);"),
	"void foo() {\n\tfoo(void);\n}\n",

	fmtFun("a = a;"),
	fmtFun("a = 1 + 2;"),
	fmtFun("1 + 2 = a = 3 + 4;"),
	fmtFun("a -= 1;"),
	fmtFun("a ? a : a;"),
	fmtFun("a = a ? a : a;"),
	fmtFun("a ? a : a = a;"),
	fmtFun("a ? a = a : a;"),
	fmtFun("a + a;"),
	fmtFun("a / a % a;"),
	fmtFun("a ^ a || a != a;"),
	fmtFun("a != a * a == a;"),
	fmtFun(" ++a;"),
	fmtFun(" !a;"),
	fmtFun(" + -- ++a;"),
	fmtFun("a++ ;"),
	fmtFun("a++ -- ;"),
	fmtFun(" ++a-- ++ ;"),
	fmtFun("a++ [0]++ ;"),
	fmtFun("a, a, a;"),
	fmtFun("int(a);"),
	fmtFun("highp int(a);"),

	"struct A {\n\tint a;\n};\nA a = A(1);\n",

	"int a, b[], c[3], d;\n",
	"int a = 0, b[2] = (1, 2);\n",

	"int foo(in int a, out int b, inout int c, int d, const int e, const inout int f);\n",

	"struct A {\n\tint a;\n};\n",
	"struct A {\n\tint a;\n} a, b;\n",
	"struct A {\n\tint a;\n};\nA a, b[2];\n",
	"struct A {\n\tconst int a;\n};\n",

	fmtFun("a.bar();"),

	"layout(location = 2) int a;\n",
	"layout(location = 2u, col_major) uniform;\n",

	"int a, b;\ninvariant a, b;\n",
	"invariant int a, b;\n",

	"uniform A {\n\tint a;\n};\n",
	"layout(row_major) uniform A {\n\tint a;\n};\n",
	"uniform A {\n\tint a;\n} a[47];\n",

	"void foo() {\n}\n",
	"void foo(int a, float b) {\n}\n",
	"void foo(void) {\n}\n",
	"void foo(int, float);\n",
	"void foo(int[7] a, float b[4]);\n",

	"smooth sampler2D a;\nflat isampler3D b;\n",
	"centroid in int a;\ncentroid out float b;\n",
	"const int a;\nin float b;\nout int c;\n",

	"#version 200\nint foo();\n",
	"#version 130\nvoid f(bool a) {\n\tif(a) ;\n}\n",
}

// Note to reader:
//
// These tests check whether the formatted output is character-for-character identical to the
// input. The verbatim printer should produce identical output (that is its purpose). On the
// other hand we don't really care about placement of each space in the case of the pretty
// printer, we only want it to be readable. If you are working on the formatter and you encounter
// a failure here, feel free to reformat the input if the result is still readable. In the tests
// we use \t for indentation. When comparing pretty printer output, the tabs are replaced with
// whatever makeIdent(1) produces.
func compare(t *testing.T, expected string, ast interface{}, version string, verbatim bool) {
	f := "%v"
	if verbatim {
		f = "%#v"
	}

	formatted := fmt.Sprintf(f, Formatter(ast))

	if version != "" {
		if verbatim {
			formatted = fmt.Sprintf("#version %s%s", version, formatted) // new-line is prefix of next line
		} else {
			formatted = fmt.Sprintf("#version %s\n%s", version, formatted)
		}
	}

	if !verbatim {
		expected = strings.Replace(expected, "\t", makeIndent(1), -1)
	}
	if expected != formatted {
		name := "pretty"
		if verbatim {
			name = "verbatim"
		}
		t.Errorf("Expected and obtained strings differ for %s printer.", name)
		t.Logf("Expected:\n%s", expected)
		t.Logf("Obtained:\n%s", formatted)
		for i := 0; i < len(expected) && i < len(formatted); i++ {
			f, s := expected[i], formatted[i]
			if f != s {
				t.Logf("First difference at character %d: '%c' vs '%c'", i, f, s)
				break
			}
		}
		if len(expected) > len(formatted) {
			t.Logf("Expected string longer.")
		}
		if len(expected) < len(formatted) {
			t.Logf("Obtained string longer.")
		}
	}
}

func TestFormat(t *testing.T) {
	for _, input := range fmtTests {
		ast, version, _, err := Parse(input, ast.LangVertexShader, nil)
		if len(err) > 0 {
			t.Errorf("Error parsing input: %s", err[0])
			return
		}

		compare(t, input, ast, version, false)
		compare(t, input, ast, version, true)
	}
}
