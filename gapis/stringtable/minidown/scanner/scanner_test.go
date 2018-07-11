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

package scanner_test

import (
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/gapis/stringtable/minidown/scanner"
	"github.com/google/gapid/gapis/stringtable/minidown/token"
)

var source = `# This is a H1 heading`

type tok struct{ Text, Kind string }

func TestScanner(t *testing.T) {
	assert := assert.To(t)

	B := func() tok { return tok{"*", "token.Bullet"} }
	H := func(t string) tok { return tok{t, "token.Heading"} }
	E := func(t string) tok { return tok{t, "token.Emphasis"} }
	T := func(t string) tok { return tok{t, "token.Text"} }
	TAG := func(t string) tok { return tok{"{{" + t + "}}", "token.Tag"} }
	OB := func(r rune) tok { return tok{string([]rune{r}), "token.OpenBracket"} }
	CB := func(r rune) tok { return tok{string([]rune{r}), "token.CloseBracket"} }
	NL := func() tok { return tok{"\n", "token.NewLine"} }

	for _, test := range []struct {
		source   string
		expected []tok
	}{
		{`Some basic-text ¢ 12 34`, []tok{T("Some"), T("basic-text"), T("¢"), T("12"), T("34")}},
		{`# A H1 heading`, []tok{H("#"), T("A"), T("H1"), T("heading")}},
		{`## A H2 heading`, []tok{H("##"), T("A"), T("H2"), T("heading")}},
		{`A ### H3 heading`, []tok{T("A"), H("###"), T("H3"), T("heading")}},
		{`   Multi   whitespace   `, []tok{T("Multi"), T("whitespace")}},
		{`An *emphasised* _string_`, []tok{T("An"), E("*"), T("emphasised"), E("*"), E("_"), T("string"), E("_")}},
		{`Not an * emphasised * _ string _`, []tok{
			T("Not"), T("an"), B(), T("emphasised"), B(), T("_"), T("string"), T("_"),
		}},
		{`this_is__not_emphasised`, []tok{T("this_is__not_emphasised")}},
		{`this*bit is*emphasised`, []tok{T("this"), E("*"), T("bit"), T("is"), E("*"), T("emphasised")}},
		{`_*__**nested emphasis**__*_`, []tok{
			E("_"), E("*"), E("__"), E("**"), T("nested"), T("emphasis"), E("**"), E("__"), E("*"), E("_"),
		}},
		{`Some text
    split over several
` + "\r\nlines", []tok{T("Some"), T("text"), NL(), T("split"), T("over"), T("several"), NL(), NL(), T("lines")}},
		{`{here's [some] (brackets)}`, []tok{
			OB('{'), T("here's"), OB('['), T("some"), CB(']'), OB('('), T("brackets"), CB(')'), CB('}'),
		}},
		{`{{these}} {{are}} {{t_a_g_s_111}}`, []tok{
			TAG("these"), TAG("are"), TAG("t_a_g_s_111"),
		}},
		{`[{{person}}]({{link}}) likes to use [google](http://www.google.com).`, []tok{
			OB('['), TAG("person"), CB(']'), OB('('), TAG("link"), CB(')'),
			T("likes"), T("to"), T("use"),
			OB('['), T("google"), CB(']'), OB('('), T("http://www.google.com"), CB(')'), T("."),
		}},
		{`\[This\] \\ \*has \*escaped\{\} characters`, []tok{
			T("["), T("This"), T("]"), T("\\"), T("*"), T("has"), T("*"),
			T("escaped"), T("{"), T("}"), T("characters"),
		}},
		{`This is a slash \\. This is a slash \. and so is this \`, []tok{
			T("This"), T("is"), T("a"), T("slash"), T("\\"),
			T("."), T("This"), T("is"), T("a"), T("slash"), T("\\"), T("."),
			T("and"), T("so"), T("is"), T("this"), T("\\"),
		}},
		{`\[{{Tag_in_brackets}}\]`, []tok{
			T("["), TAG("Tag_in_brackets"), T("]"),
		}},
	} {
		tokens, errs := scanner.Scan("test", test.source)
		assert.For(test.source).ThatSlice(errs).IsEmpty()

		got := make([]tok, len(tokens))
		for i, t := range tokens {
			switch t := t.(type) {
			case token.Text:
				got[i] = tok{t.String(), fmt.Sprintf("%T", t)}
			default:
				got[i] = tok{t.CST().Tok().String(), fmt.Sprintf("%T", t)}
			}
		}
		assert.For(test.source).That(got).DeepEquals(test.expected)
	}
}
