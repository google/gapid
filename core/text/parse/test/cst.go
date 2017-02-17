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

package test

import (
	"bytes"

	"github.com/google/gapid/core/text/parse"
)

func Peek(r *parse.Reader, value string) bool {
	result := r.String(value)
	r.Rollback()
	return result
}

func Token(value string) parse.Token {
	source := &parse.Source{Filename: "cst_test.api", Runes: bytes.Runes([]byte(value))}
	return parse.Token{Source: source, Start: 0, End: len(source.Runes)}
}

func Fragment(tok string) parse.Fragment {
	return parse.NewFragment(Token(tok))
}

func Separator(list ...interface{}) parse.Separator {
	var sep parse.Separator
	for _, e := range list {
		if f, ok := e.(parse.Fragment); ok {
			sep = append(sep, f)
		} else {
			sep = append(sep, Fragment(e.(string)))
		}
	}
	return sep
}

func Leaf(v string) parse.Node {
	l := &parse.Leaf{}
	l.SetToken(Token(v))
	return l
}

func asNode(v interface{}) parse.Node {
	if n, ok := v.(parse.Node); ok {
		return n
	}
	return Leaf(v.(string))
}

func Branch(nodes ...interface{}) *parse.Branch {
	n := &parse.Branch{}
	for _, v := range nodes {
		c := asNode(v)
		n.Children = append(n.Children, c)
	}
	return n
}

func Node(prefix interface{}, v interface{}, suffix interface{}) parse.Node {
	n := asNode(v)
	if prefix != nil {
		if sep, ok := prefix.(parse.Separator); ok {
			n.AddPrefix(sep)
		} else {
			n.AddPrefix(Separator(prefix))
		}
	}
	if suffix != nil {
		if sep, ok := suffix.(parse.Separator); ok {
			n.AddSuffix(sep)
		} else {
			n.AddSuffix(Separator(suffix))
		}
	}
	return n
}
