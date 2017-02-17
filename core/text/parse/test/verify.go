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
	"unicode/utf8"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/parse"
)

func VerifyTokens(ctx log.Context, got parse.Node) {
	next := 0
	VerifyNodeTokens(ctx, got, &next)
}

func VerifyNodeTokens(ctx log.Context, n parse.Node, next *int) {
	// walk the prefix
	for _, f := range n.Prefix() {
		VerifyFragmentTokens(ctx, f, next)
	}
	start := *next
	if b, ok := n.(*parse.Branch); ok {
		for _, c := range b.Children {
			VerifyNodeTokens(ctx, c, next)
		}
	} else {
		VerifyFragmentTokens(ctx, n, next)
	}
	end := *next
	// walk the suffix
	for _, f := range n.Suffix() {
		VerifyFragmentTokens(ctx, f, next)
	}
	tok := n.Token()
	if start != end {
		assert.For(ctx, "branch start").That(tok.Start).Equals(start)
		assert.For(ctx, "branch end").That(tok.End).Equals(end)
	}
}

func VerifyFragmentTokens(ctx log.Context, f parse.Fragment, next *int) {
	tok := f.Token()
	str := tok.String()
	ctx = ctx.S("token", str)
	length := utf8.RuneCountInString(str)
	assert.For(ctx, "start").That(tok.Start).Equals(*next)
	assert.For(ctx, "start").That(tok.End).Equals(*next + length)
	*next = tok.End
}
