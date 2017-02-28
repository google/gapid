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
	"testing"

	"github.com/google/gapid/core/assert"
	st "github.com/google/gapid/gapis/stringtable"
)

var source = `
# SIMPLE_STRING
This is a simple string

# SINGLE_LINE_STRING
This string is
displayed as a single
line

# SIMPLE_PARAMETERS
{{person}} loves to eat {{food}}.

# REPEATING_PARAMETERS
{{person}} loves to eat his(her) ({{person}}'s) {{food}}, because {{food}} is really tasty for {{person}}.

# REPEATING_PARAMETERS_DIFFERENT_TYPES
{{person:string}} loves to eat his(her) ({{person}}'s) {{food:u32}}, because {{food:bool}} is really tasty for {{person}}.

# LINKS
[{{person}}]({{link}}) likes to use [google](http://www.google.com).

` + "# SINGLE_LINE_STRING_CRLF\r\nThis is\r\n a simple\r\n string"

func TestParser(t *testing.T) {
	assert := assert.To(t)

	loc, _, errs := Parse("test", source)
	assert.For("errors").ThatSlice(errs).DeepEquals([]error{fmt.Errorf(
		"Entry REPEATING_PARAMETERS_DIFFERENT_TYPES contains duplicate parameters" +
			" food with different types: u32 != bool")})

	B := func(c ...*st.Node) *st.Node { return &st.Node{Node: &st.Node_Block{Block: &st.Block{Children: c}}} }
	T := func(s string) *st.Node { return &st.Node{Node: &st.Node_Text{Text: &st.Text{Text: s}}} }
	P := func(k string) *st.Node { return &st.Node{Node: &st.Node_Parameter{Parameter: &st.Parameter{Key: k}}} }
	L := func(b, t *st.Node) *st.Node { return &st.Node{Node: &st.Node_Link{Link: &st.Link{Body: b, Target: t}}} }
	WS := func() *st.Node { return &st.Node{Node: &st.Node_Whitespace{Whitespace: &st.Whitespace{}}} }

	for _, test := range []struct {
		key      string
		expected *st.Node
	}{
		{"SIMPLE_STRING", T("This is a simple string")},
		{"SINGLE_LINE_STRING", T("This string is displayed as a single line")},
		{"SIMPLE_PARAMETERS", B(
			P("person"), WS(), T("loves to eat"), WS(), P("food"), T("."))},
		{"REPEATING_PARAMETERS", B(
			P("person"), WS(), T("loves to eat his(her) ("), T("'s)"), WS(),
			P("food"), T(", because"), WS(), WS(), T("is really tasty for"), WS(),
			T("."))},
		{"REPEATING_PARAMETERS_DIFFERENT_TYPES", B(
			P("person"), WS(), T("loves to eat his(her) ("), T("'s)"), WS(),
			P("food"), T(", because"), WS(), WS(), T("is really tasty for"), WS(),
			T("."))},
		{"LINKS", B(
			L(P("person"), P("link")),
			WS(), T("likes to use"), WS(),
			L(T("google"), T("http://www.google.com")),
			T("."))},
		{"SINGLE_LINE_STRING_CRLF", T("This is a simple string")},
	} {
		got, found := loc.Entries[test.key]
		assert.For(test.key).That(found).Equals(true)
		assert.For(test.key).That(got).DeepEquals(test.expected)
	}
}
