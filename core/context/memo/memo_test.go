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

package memo_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/context/memo"
	"github.com/google/gapid/core/text/note"
)

func TestTag(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	ctx = memo.Tag(ctx, "Message")
	check(assert, ctx, memo.TagKey, "Message")
	ctx = memo.Tag(ctx, "Second")
	check(assert, ctx, memo.TagKey, "Second")
	ctx = memo.Tagf(ctx, "I am the %dnd %s", 2, "message")
	check(assert, ctx, memo.TagKey, "I am the 2nd message")
}

func TestEnter(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	ctx = memo.Enter(ctx, "First")
	check(assert, ctx, memo.TraceKey, "First")
	ctx = memo.Enter(ctx, "Second")
	check(assert, ctx, memo.TraceKey, "First->Second")
}

func TestPrint(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	ctx = memo.Print(ctx, "Message")
	check(assert, ctx, memo.PrintKey, "Message")
	ctx = memo.Print(ctx, "Second")
	check(assert, ctx, memo.PrintKey, "Second")
	ctx = memo.Printf(ctx, "I am the %dnd %s", 2, "message")
	check(assert, ctx, memo.PrintKey, "I am the 2nd message")
}

func TestStatus(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	ctx = memo.Status(ctx, "Failed")
	check(assert, ctx, memo.StatusKey, "Failed")
	ctx = memo.Status(ctx, "Succeded")
	check(assert, ctx, memo.StatusKey, "Succeded")
	ctx = memo.Statusf(ctx, "Died %d times", 9)
	check(assert, ctx, memo.StatusKey, "Died 9 times")
}

func TestComplexNote(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	page := note.Page{}
	memo.PrintKey.Transcribe(ctx, &page, "A Message")
	memo.TagKey.Transcribe(ctx, &page, "Tests")
	page.Append(memo.ExtraSection, "Silly1", "Clown")
	page.Append(memo.DetailSection, "Child", "A Value")
	page.Append(memo.DetailSection, "Descendant", "42")
	child := note.Page{}
	memo.PrintKey.Transcribe(ctx, &child, "A Message")
	child.Append(memo.DetailSection, 10, "Cheese")
	child.Append(memo.DetailSection, "Apple", true)
	page.Append(memo.DetailSection, "Nested", child)
	memo.PrefixPair("A1").Transcribe(ctx, &page, 10)
	memo.PrefixValue("A2").Transcribe(ctx, &page, 20)
	memo.PrefixPair("A3").Transcribe(ctx, &page, 30)
	memo.DetailPair("C1").Transcribe(ctx, &page, 5)
	memo.SuffixPair("B1").Transcribe(ctx, &page, 30)
	memo.SuffixValue("B2").Transcribe(ctx, &page, 40)
	memo.SuffixPair("B3").Transcribe(ctx, &page, 50)
	page.Sort()
	for _, test := range []struct {
		style  note.Style
		expect string
	}{
		{note.Raw, `Tests:A Message`},
		{note.Brief, `Tests:A Message`},
		{note.Normal, `Tests:A1=10,20,A3=30:A Message:B1=30,40,B3=50:C1=5,Child=A Value,Descendant=42,Nested={A Message:10=Cheese,Apple=true}`},
		{note.Detailed, `
Tests:A1=10,20,A3=30:A Message:B1=30,40,B3=50
    C1         = 5
    Child      = A Value
    Descendant = 42
    Nested     = A Message
        10    = Cheese
        Apple = true
    Silly1 = Clown`},
	} {
		got := test.style.Print(page)
		expect := strings.TrimSpace(test.expect)
		assert.For("Style %s", test.style).That(got).Equals(expect)
	}
}

func check(assert assert.Manager, ctx context.Context, key interface{}, expect string) {
	type transcriber interface {
		Transcribe(ctx context.Context, page *note.Page, value interface{})
	}
	value := ctx.Value(key)
	page := note.Page{}
	key.(transcriber).Transcribe(ctx, &page, value)
	got := note.Detailed.Print(page)
	assert.For("transcription").That(got).Equals(expect)
}
