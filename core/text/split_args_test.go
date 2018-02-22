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

package text_test

import (
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text"
)

func ExampleSplitArgs() {
	for i, s := range text.SplitArgs(`--cat=meow -dog woof "fish"="\"blub blub\""`) {
		fmt.Printf("%v: '%v'\n", i, s)
	}
	// Output:
	// 0: '--cat=meow'
	// 1: '-dog'
	// 2: 'woof'
	// 3: 'fish="blub blub"'
}

func TestSplitArgs(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []struct {
		str      string
		expected []string
	}{
		{`a b c`, []string{`a`, `b`, `c`}},
		{`"a b c"`, []string{`a b c`}},
		{`\"a b c\"`, []string{`"a`, `b`, `c"`}},
		{`\\a b c\\"`, []string{`\a`, `b`, `c\`}},
		{`\\ abc \\"`, []string{`\`, `abc`, `\`}},
		{`\x abc \x"`, []string{`x`, `abc`, `x`}},
		{`"meow \" woof"`, []string{`meow " woof`}},
	} {
		got := text.SplitArgs(test.str)
		assert.For(ctx, "text.SplitArgs(%v)", test.str).ThatSlice(got).Equals(test.expected)
	}
}

func TestQuote(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []struct {
		str      []string
		expected []string
	}{
		{[]string{`a`, `b`, `c`}, []string{`a`, `b`, `c`}},
		{[]string{`a`, `a b c`}, []string{`a`, `"a b c"`}},
		{[]string{`\"a`, `b`, `c\"`}, []string{`\\\\"a`, `b`, `c\\\\"`}},
		{[]string{`meow \" woof`}, []string{`"meow \\\\" woof"`}},
	} {
		got := text.Quote(test.str)
		assert.For(ctx, "test.Quote(%v)", test.str).ThatSlice(got).Equals(test.expected)
	}
}
