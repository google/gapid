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

package parse_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/core/text/parse/test"
)

var (
	B     = test.Branch
	N     = test.Node
	S     = test.Separator
	F     = test.Fragment
	List  = test.List
	Array = test.Array
	Call  = test.Call
)

func TestEmpty(t *testing.T) {
	ctx := log.Testing(t)
	testParse(ctx, ``, B(), List())
}

func TestErrorInvalid(t *testing.T) {
	ctx := log.Testing(t)
	testFail(ctx, `@`)
}

func TestErrorExpected(t *testing.T) {
	ctx := log.Testing(t)
	testFail(ctx, `[,]`)
	testFail(ctx, `[ 0`)
}

func TestUnconsumed(t *testing.T) {
	ctx := log.Testing(t)
	testCustomFail(ctx, "a", func(p *parse.Parser, n *cst.Branch) {})
}

func TestUnconsumedByBranch(t *testing.T) {
	ctx := log.Testing(t)
	testCustomFail(ctx, "a", func(p *parse.Parser, n *cst.Branch) {
		p.ParseBranch(n, func(n *cst.Branch) {
			p.NotSpace()
		})
	})
}

func TestUnconsumedOnBranch(t *testing.T) {
	ctx := log.Testing(t)
	testCustomFail(ctx, "a", func(p *parse.Parser, n *cst.Branch) {
		p.NotSpace()
		p.ParseBranch(n, func(n *cst.Branch) {})
	})
}

func TestSpace(t *testing.T) {
	ctx := log.Testing(t)
	testParse(ctx, ` `, N(" ", B(), nil), List())
	testParse(ctx, `  `, N("  ", B(), nil), List())
}

func TestLineComment(t *testing.T) {
	ctx := log.Testing(t)
	testParse(ctx, `//note`, N(S(F("//note")), B(), nil), List())
	testParse(ctx, `//note
`, N(S(F("//note"), "\n"), B(), nil), List())
}

func TestBlockComment(t *testing.T) {
	ctx := log.Testing(t)
	testParse(ctx, `/*note*/`, N(S(F("/*note*/")), B(), nil), List())
}

func TestCommentUnclosed(t *testing.T) {
	ctx := log.Testing(t)
	testFail(ctx, `/*a`)
}

func TestComplexComment(t *testing.T) {
	ctx := log.Testing(t)
	testParse(ctx, `
///*a*/
/*b//c*
//*/ //d
d //e
`, N(
		S("\n", F("///*a*/"), "\n", F("/*b//c*\n//*/"), " ", F("//d"), "\n"),
		B("d"),
		S(" ", F("//e"), "\n")),
		List("d"))
}

func TestValue(t *testing.T) {
	ctx := log.Testing(t)
	testParse(ctx, `a`, B("a"), List("a"))
	testParse(ctx, `0xA4`, B("0xA4"), List(164))
	testParse(ctx, ` a`, N(" ", B("a"), nil), nil)
}

func TestArray(t *testing.T) {
	ctx := log.Testing(t)
	testParse(ctx, `[]`, B(B("[", B(), "]")), List(Array()))
	testParse(ctx, `[a]`, B(B("[", B("a"), "]")), List(Array("a")))
	testParse(ctx, `[a,b]`, B(B("[", B("a", ",", "b"), "]")), List(Array("a", "b")))
}

func TestCall1(t *testing.T) {
	ctx := log.Testing(t)
	testParse(ctx, `a(b)`, B("a", B("(", B("b"), ")")), List(Call("a", "b")))
	testParse(ctx, `a(b,c)`, B("a", B("(", B("b", ",", "c"), ")")), List(Call("a", "b", "c")))
}

func TestComplex(t *testing.T) {
	ctx := log.Testing(t)
	testParse(ctx,
		` a  (b   ,c)`,
		N(" ", B(
			N(nil, "a", "  "),
			B("(", B(N(nil, "b", "   "), ",", "c"), ")"),
		), nil),
		List(Call("a", "b", "c")))
}

func TestErrorLimit(t *testing.T) {
	ctx := log.Testing(t)
	errs := parse.Parse("parser_test.api", "", parse.NewSkip("//", "/*", "*/"), func(p *parse.Parser, n *cst.Branch) {
		for i := 0; true; i++ {
			p.ErrorAt(n, "failure")
			if i >= parse.ParseErrorLimit {
				log.F(ctx, true, "Parsing not terminated. %d errors", i)
			}
		}
	})
	assert.For(ctx, "errs").ThatSlice(errs).IsLength(parse.ParseErrorLimit)
}

func TestCursor(t *testing.T) {
	ctx := log.Testing(t)
	root := List()
	filename, line, column := "parser_test.api", 3, 5
	content := ""
	for i := 1; i < line; i++ {
		content += "\n"
	}
	for i := 1; i < column; i++ {
		content += " "
	}
	content += "@  \n  "
	errs := parse.Parse(filename, content, parse.NewSkip("//", "/*", "*/"), func(p *parse.Parser, b *cst.Branch) {
		root.Parser(p)(b)
	})
	if len(errs) == 0 {
		log.E(ctx, "Expected errors")
	}
	l, c := errs[0].At.Tok().Cursor()
	assert.For(ctx, "Line").That(l).Equals(line)
	assert.For(ctx, "Column").That(c).Equals(column)
	assert.For(ctx, "Error").ThatString(errs[0]).HasPrefix(fmt.Sprintf("%s:%v:%v: Unexpected", filename, line, column))
}

func TestCustomPanic(t *testing.T) {
	ctx := log.Testing(t)
	const custom = fault.Const("custom")
	defer func() {
		assert.For(ctx, "recover").That(recover()).Equals(custom)
	}()
	parse.Parse("parser_test.api", "", parse.NewSkip("//", "/*", "*/"), func(p *parse.Parser, _ *cst.Branch) {
		panic(custom)
	})
}

func testParse(ctx context.Context, content string, n cst.Node, ast *test.ListNode) {
	ctx = log.V{"content": content}.Bind(ctx)
	root := List()
	var gotCst *cst.Branch
	rootParse := func(p *parse.Parser, n *cst.Branch) {
		gotCst = n
		root.Parser(p)(n)
	}
	errs := parse.Parse("parser_test.api", content, parse.NewSkip("//", "/*", "*/"), rootParse)
	assert.For(ctx, "errors").ThatSlice(errs).IsEmpty()
	out := &bytes.Buffer{}
	gotCst.Write(out)
	assert.For(ctx, "content").ThatString(out).Equals(content)
	test.VerifyTokens(ctx, gotCst)
	if n != nil {
		assert.For(ctx, "CST").That(gotCst).DeepEquals(n)
	}
	if ast != nil {
		assert.For(ctx, "AST").That(root).DeepEquals(ast)
	}
}

func testCustomFail(ctx context.Context, content string, do parse.RootParser) {
	errs := parse.Parse("parser_test.api", content, parse.NewSkip("//", "/*", "*/"), do)
	if len(errs) == 0 {
		log.E(ctx, "Expected errors")
	} else {
		for _, e := range errs {
			line, column := e.At.Tok().Cursor()
			log.I(ctx, "%v:%v: %s\n", line, column, e.Message)
		}
	}
}

func testFail(ctx context.Context, content string) {
	root := List()
	testCustomFail(ctx, content, func(p *parse.Parser, b *cst.Branch) {
		root.Parser(p)(b)
	})
}
