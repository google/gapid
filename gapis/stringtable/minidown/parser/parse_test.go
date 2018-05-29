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
	"github.com/google/gapid/gapis/stringtable/minidown/node"
	"github.com/google/gapid/gapis/stringtable/minidown/parser"
)

func TestParser(t *testing.T) {
	assert := assert.To(t)

	B := func(c ...node.Node) node.Node { return &node.Block{Children: c} }
	T := func(s string) node.Node { return &node.Text{Text: s} }
	E := func(s string, c node.Node) node.Node { return &node.Emphasis{Style: s, Body: c} }
	L := func(b, t node.Node) node.Node { return &node.Link{Body: b, Target: t} }
	TAG := func(id string) node.Node { return &node.Tag{Identifier: id, Type: "string"} }
	TAGT := func(id string, ty string) node.Node {
		return &node.Tag{Identifier: id, Type: ty}
	}
	H1 := func(n node.Node) node.Node { return &node.Heading{Scale: 1, Body: n} }
	H2 := func(n node.Node) node.Node { return &node.Heading{Scale: 2, Body: n} }
	H3 := func(n node.Node) node.Node { return &node.Heading{Scale: 3, Body: n} }
	WS := func() node.Node { return &node.Whitespace{} }
	NL := func() node.Node { return &node.NewLine{} }

	for _, test := range []struct {
		source   string
		expected node.Node
	}{
		{`Here is some text`, T("Here is some text")},

		{`This is a
			single line`, T("This is a single line")},

		{`This is also a` + "\r\n" + `single line`,
			T("This is also a single line")},

		{`These are

			lines with

` + "\r\n" + `

			breaks`, B(T("These are"), NL(), T("lines with"), NL(), T("breaks"))},

		{`This ends on a newline
			`, T("This ends on a newline")},

		{`_I_ __really__
			*love* **emphasising**`, B(
			E("_", T("I")), WS(), E("__", T("really")), WS(),
			E("*", T("love")), WS(), E("**", T("emphasising")))},

		{`_these *__are nested__* emphasises_`,
			E("_", B(T("these"), WS(), E("*", E("__", T("are nested"))), WS(), T("emphasises")))},

		{`this_is_not_emphasised`, T("this_is_not_emphasised")},

		{`_these are__ *unclosed emphasis**`,
			T("_these are__ *unclosed emphasis**")},

		{`# Heading 1
		  ## Heading 2
			### Heading 3
			## Heading 2
			# Heading 1`,
			B(
				H1(T("Heading 1")),
				H2(T("Heading 2")),
				H3(T("Heading 3")),
				H2(T("Heading 2")),
				H1(T("Heading 1")),
			),
		},

		{`# Heading 1`,
			H1(T("Heading 1"))},

		{"# Heading 1 ending with LF\n",
			H1(T("Heading 1 ending with LF"))},

		{"# Heading 1 ending with CRLF\r\n",
			H1(T("Heading 1 ending with CRLF"))},

		{`* Bullets are treated as text for now`,
			T("* Bullets are treated as text for now")},

		{`Let me [google](http://www.google.com) that for you.`,
			B(T("Let me"), WS(), L(T("google"), T("http://www.google.com")), WS(), T("that for you."))},

		{`\[{{Tag_in_brackets}}\]`,
			B(
				T("["),
				TAG("Tag_in_brackets"),
				T("]"),
			)},

		{`\[{{Tag_with_type_in_brackets:s64}}\]`,
			B(
				T("["),
				TAGT("Tag_with_type_in_brackets", "s64"),
				T("]"),
			)},
		{`[{{person}}]({{link}}) likes to use [google](http://www.google.com).`,
			B(
				L(TAG("person"), TAG("link")),
				WS(), T("likes to use"), WS(),
				L(T("google"), T("http://www.google.com")),
				T("."),
			)},

		{`This is an [unclosed link body.`,
			T("This is an [unclosed link body.")},

		{`This is an [unclosed link](target.`,
			B(T("This is an"), WS(), L(T("unclosed link"), nil), T("(target."))},
	} {
		got, errs := parser.Parse("test", test.source)
		assert.For(test.source).ThatSlice(errs).IsEmpty()
		assert.For(test.source).That(got).DeepEquals(test.expected)
	}
}
