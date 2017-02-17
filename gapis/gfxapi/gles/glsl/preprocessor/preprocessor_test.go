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

package preprocessor

import (
	"regexp"
	"testing"

	"github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"
)

var (
	A = Identifier("A")
	B = Identifier("B")
	C = Identifier("C")
	a = Identifier("a")
	b = Identifier("b")
	c = Identifier("c")
	d = Identifier("d")
)

type tokenOnlyPrinter []TokenInfo

func (top tokenOnlyPrinter) String() string {
	ret := "["
	for i, t := range top {
		if i > 0 {
			ret += " "
		}
		ret += t.Token.String()
	}
	return ret + "]"
}

var preprocessorTests = []struct {
	Input           string
	ExpectedTokens  []Token
	ExpectedVersion string
	ExpectedErrors  []string
}{
	{"const", []Token{KwConst}, "", nil},
	{"A", []Token{A}, "", nil},
	{"47", []Token{ast.IntValue(47)}, "", nil},
	{"47u", []Token{ast.UintValue(47)}, "", nil},
	{"47.5", []Token{ast.FloatValue(47.5)}, "", nil},
	{"47.5e1", []Token{ast.FloatValue(475)}, "", nil},
	{"47.5e+1", []Token{ast.FloatValue(475)}, "", nil},
	{".75", []Token{ast.FloatValue(0.75)}, "", nil},
	{"4.75", []Token{ast.FloatValue(4.75)}, "", nil},
	{"<<=", []Token{ast.BoShlAssign}, "", nil},
	{"<=", []Token{ast.BoLessEq}, "", nil},
	{"<<", []Token{ast.BoShl}, "", nil},
	{"<", []Token{ast.BoLess}, "", nil},
	{"=", []Token{ast.BoAssign}, "", nil},
	{"'", nil, "", []string{"Unknown token"}},

	{"true; 1u;",
		[]Token{KwTrue, OpSemicolon, ast.UintValue(1), OpSemicolon}, "", nil},
	{"\n\nvoid\r\rA  (\t\t)\v\v;\n\r\v",
		[]Token{ast.TVoid, A, OpLParen, OpRParen, OpSemicolon}, "", nil},

	{`  void // int
        A /* [
        *///]
        (/**/)
        ;// `, []Token{ast.TVoid, A, OpLParen, OpRParen, OpSemicolon}, "", nil},

	{"/*...", nil, "", []string{"Unterminated block comment"}},
	{"/*...*", nil, "", []string{"Unterminated block comment"}},

	{`#define A B
	A`, []Token{B}, "", nil},

	{`# define A B
	A`, []Token{B}, "", nil},

	{`#define int void
	int`, []Token{ast.TVoid}, "", nil},

	{`#define
	a`, []Token{a}, "", []string{"#define needs an argument"}},

	{`#define A a b c
	#define b c a
	A`, []Token{a, c, a, c}, "", nil},

	{`#define B A B C
	B`, []Token{A, B, C}, "", nil},

	{`#define A a B
	#define B b A
	A B`, []Token{a, b, A, b, a, B}, "", nil},

	{`#define A
	#ifdef A
	a
	#else
	b
	#endif`, []Token{a}, "", nil},

	{`#define A
	#ifndef A
	a
	#else
	b
	#endif`, []Token{b}, "", nil},

	{`#define A
	#ifdef B
	a
	#else
	b
	#endif`, []Token{b}, "", nil},

	{"#ifdef AAA", nil, "", []string{"Unterminated #if"}},

	{`#ifdef A
	#else
	#else
	#endif`, nil, "", []string{"multiple #else"}},

	{"#else", nil, "", []string{"Unmatched #else"}},

	{"#endif", nil, "", []string{"Unmatched #endif"}},

	{"#ifdef\n#endif", nil, "", []string{"#ifdef needs an argument"}},
	{"#define A B\n#define A C", nil, "", []string{"'A' already defined"}},
	{"#undef A", nil, "", []string{"'A' not defined"}},
	{"#error Hello world", nil, "", []string{"Hello world"}},

	{"#define A B\n#undef A\nA", []Token{A}, "", nil},

	{`#define A(x) b
	A(a)`, []Token{b}, "", nil},

	{`#define A(x) x
	A(a)`, []Token{a}, "", nil},

	{`#define A(x) a x c
	A(b)`, []Token{a, b, c}, "", nil},

	{`#define A(x) a x
	#define B(x, y) A(x y)
	B(b, c)`, []Token{a, b, c}, "", nil},

	{`#define A(x) a x
	#define B(x) x(b)
	B(A)`, []Token{a, b}, "", nil},

	{`#define A(x) a B(x)
	#define B(x) A(x) b
	B(c)
	A(c)`, []Token{a, B, OpLParen, c, OpRParen, b, a, A, OpLParen, c, OpRParen, b}, "", nil},

	{`#define A(x) x b
	#define B(x) a A(x
	B(c))`, []Token{a, c, b}, "", nil},

	{`#define C(x, y) x y
	C(a)`, []Token{a}, "", []string{"Incorrect number of arguments to macro 'C': expected 2, got 1."}},

	{`#define A(x) x x
	A(a
	#undef A
	#define A b
	A)`, []Token{a, b, a, b}, "", nil},

	{`#define A(x) x
	A(a`, []Token{A}, "", []string{"Unexpected end of file while processing a macro."}},

	{`#define A(x, y) x y
	A((a,(b)),c)`, []Token{OpLParen, a, ast.BoComma, OpLParen, b, OpRParen, OpRParen, c}, "", nil},

	{`#define A a B
	#define B b A
	#define C(x) c x
	C(A)`, []Token{c, a, b, A}, "", nil},

	{"#define f(x, x) x", nil, "", []string{"Macro 'f' contains two arguments named 'x'."}},

	{`#define A(x) x
	#define B(x) A(C)(x)
	#define C(x) A(x)
	B(a)`, []Token{a}, "", nil},

	{`#define A (a) a
	A(b)`, []Token{OpLParen, a, OpRParen, a, OpLParen, b, OpRParen}, "", nil},

	{"+\\\n+", []Token{ast.UoPreinc}, "", nil},

	{`#define A(int) int
	A(b)`, []Token{b}, "", nil},

	{"#define A(", nil, "", []string{"Macro definition ended unexpectedly."}},
	{"#define A(*,b)", nil, "", []string{"Expected an identifier, got '\\*'."}},
	{"#define A(a*b)", nil, "", []string{"Expected ',', '\\)', got '\\*'."}},

	{`#ifdef A
	#undef Ignored
	#endif`, nil, "", nil},

	{`#ifdef A
	#error Ignored
	#endif`, nil, "", nil},

	{`#define A B(a
	#define B(x) b
	#define C(x) c x
	C(A)`, []Token{c, B}, "", []string{"Unexpected end of file while processing a macro."}},

	{`#define A(x) a A(x
	A(b))`, []Token{a, A, OpLParen, b, OpRParen}, "", nil},

	{`#define A1(x) A2(x
	#define A2(x) A3(x
	#define A3(x) x
	#define B(x) B(A1(x))
	B(a))`, []Token{B, OpLParen, a}, "", nil},

	{`#define JOIN(x,y) x ## _ ## y
	JOIN(a,b)`, []Token{Identifier("a_b")}, "", nil},

	{`#define JOIN(x,y) a x ## y d
	JOIN(b,c)`, []Token{a, Identifier("bc"), d}, "", nil},

	{`__LINE__ a __LINE__
	b __LINE__ b
	__LINE__`, []Token{ast.IntValue(1), a, ast.IntValue(1), b, ast.IntValue(2), b, ast.IntValue(3)}, "", nil},

	{`__FILE__`, []Token{ast.IntValue(0)}, "", nil},
	{`__VERSION__`, []Token{ast.IntValue(300)}, "", nil},
	{`GL_ES`, []Token{ast.IntValue(1)}, "", nil},

	// #if tests... see note on fakeExpressionEvaluator.
	{`#if 1
	A
	#else
	B
	#endif`, []Token{A}, "", nil},

	{`#if 1 2
	A
	#else
	B
	#endif`, []Token{B}, "", nil},

	{`#if 1
	A
	#elif 1
	B
	#endif`, []Token{A}, "", nil},

	{`#if 1
	A
	#elif 1
	B
	#elif 1
	C
	#endif`, []Token{A}, "", nil},

	{`#if 1 2
		#if 1
		A
		#else
		B
		#endif
	#else
		#if 1
		C
		#else
		D
		#endif
	#endif`, []Token{C}, "", nil},

	{`#ifdef A
		#ifndef A
		a
		#else
		b
		#endif
	#else
		#ifndef A
		c
		#else
		d
		#endif
	#endif`, []Token{c}, "", nil},

	{`#if 1 2
	A
	#elif 1
	B
	#endif`, []Token{B}, "", nil},

	{`#if 1 2
	A
	#elif 1 2
	B
	#else
	C
	#endif`, []Token{C}, "", nil},

	{`#if 1 2
	A
	#else
	C
	#elif 1 2
	B
	#endif`, []Token{C}, "", []string{"#elif after #else"}},

	{`#elif 1`, nil, "", []string{"Unmatched #elif"}},

	{`#define A 1 2
	#if A
	a
	#else
	b
	#endif`, []Token{b}, "", nil},

	{`#define A
	#if defined(A)
	a
	#else
	b
	#endif`, []Token{a}, "", nil},

	{`#define A
	#if defined A
	a
	#else
	b
	#endif`, []Token{a}, "", nil},

	{`#if defined A
	a
	#else
	b
	#endif`, []Token{a}, "", nil},

	{`#line 123`, []Token{}, "", nil},
	{`#line`, []Token{}, "", []string{"expected line/file number after #line"}},
	{`#line 1 2 3`, []Token{}, "", []string{"expected line/file number after #line"}},

	{`#if defined
	a
	#endif`, []Token{}, "", []string{"Operator 'defined' used incorrectly."}},

	{`#version 100
	a
	`, []Token{a}, "100", nil},
}

// We cannot test #if expressions here, as the expression evaluator is located in the ast package
// (which depends on this package) and this would create a circular dependency. Instead, we
// provide a fake evaluator, which returns the token count (modulo 2) as its result and allows us
// to test the #if logic. Tests for the correct expression evaluation are in the ast package.
func fakeExpressionEvaluator(tp *Preprocessor) (ast.IntValue, []error) {
	count := 0
	for tp.Next().Token != nil {
		count++
	}
	return ast.IntValue(count % 2), tp.Errors()
}

func TestPreprocessor(t *testing.T) {
	for _, test := range preprocessorTests {
		bad := false
		gotTokens, gotVersion, gotErrors := Preprocess(test.Input, fakeExpressionEvaluator, 0)

		for i := 0; i < len(test.ExpectedTokens) && i < len(gotTokens); i++ {
			if test.ExpectedVersion != gotVersion {
				t.Errorf("Version mismatch. Expected '%v', got '%v'.", test.ExpectedVersion, gotVersion)
				bad = true
				break
			}
			if test.ExpectedTokens[i].String() != gotTokens[i].Token.String() {
				t.Errorf("Token mismatch at position %d. "+
					"Expected '%v', got '%v'.",
					i, test.ExpectedTokens[i], gotTokens[i].Token)
				bad = true
				break
			}
		}

		if len(test.ExpectedTokens) > len(gotTokens) {
			t.Errorf("Missing tokens from position %d. Expected '%[2]v' (%[2]T).",
				len(gotTokens), test.ExpectedTokens[len(gotTokens)])
			bad = true
		}

		if len(test.ExpectedTokens) < len(gotTokens) {
			t.Errorf("Superfluous tokens from position %d. Got '%[2]v' (%[2]T).",
				len(test.ExpectedTokens), gotTokens[len(test.ExpectedTokens)].Token)
			bad = true
		}

		for i := 0; i < len(test.ExpectedErrors) && i < len(gotErrors); i++ {
			matched, err := regexp.MatchString(test.ExpectedErrors[i], gotErrors[i].Error())
			if err != nil {
				t.Errorf("Error matching regexp failed: %s", err)
				bad = true
			}
			if !matched {
				t.Errorf("Error mismatch at position %d. Expected '%[2]v', got '%[3]v'.",
					i, test.ExpectedErrors[i], gotErrors[i])
				bad = true
				break
			}
		}

		if len(test.ExpectedErrors) > len(gotErrors) {
			t.Errorf("Missing errors from position %d. Expected '%[2]v'.",
				len(gotErrors), test.ExpectedErrors[len(gotErrors)])
			bad = true
		}

		if len(test.ExpectedErrors) < len(gotErrors) {
			t.Errorf("Superfluous errors from position %d. Got '%[2]v'.",
				len(test.ExpectedErrors), gotErrors[len(test.ExpectedErrors)])
			bad = true
		}

		if bad {
			t.Logf("Program: %s", test.Input)
			t.Logf("Expected Tokens: %v", test.ExpectedTokens)
			t.Logf("Obtained Tokens: %v", tokenOnlyPrinter(gotTokens))
			t.Logf("Expected Errors: %v", test.ExpectedErrors)
			t.Logf("Obtained Errors: %v", gotErrors)
		}
	}
}

func TestPeek(t *testing.T) {
	pp := PreprocessStream("int a;", nil, 0)
	t.Logf("Lookahead: %v %v %v %v\n", pp.PeekN(0).Token, pp.PeekN(1).Token, pp.PeekN(2).Token,
		pp.PeekN(3).Token)
	if pp.PeekN(0).Token != ast.TInt {
		t.Errorf("First lookahead should be: %v", ast.TInt)
	}
	if pp.PeekN(1).Token != a {
		t.Errorf("Second lookahead should be: %v", a)
	}
	if pp.PeekN(2).Token != OpSemicolon {
		t.Errorf("Third lookahead should be: %v", OpSemicolon)
	}
	if pp.PeekN(3).Token != nil {
		t.Errorf("Fourth lookahead should be: %v", nil)
	}
}
