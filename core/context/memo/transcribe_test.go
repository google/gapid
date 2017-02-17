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
	"testing"
	"time"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/context/memo"
	"github.com/google/gapid/core/text/note"
)

type (
	Filter string
)

func (k Filter) String() string { return string(k) }

func (k Filter) Filter(ctx context.Context) bool {
	if v, ok := ctx.Value(k).(bool); ok {
		return v
	}
	return false
}

func TestTranscribe(t *testing.T) {
	assert := assert.To(t)
	testTime, _ := time.Parse(time.Stamp, "Dec  4 14:19:11.691")
	ctx := context.Background()
	ctx = memo.Timestamp(ctx, func() time.Time { return testTime })
	ctx = memo.Tag(ctx, "tagged")
	ctx = keys.WithValue(ctx, "A", "a")
	ctx = keys.WithValue(ctx, "B", "b")
	ctx = memo.Print(ctx, "message")
	expect := `Time{Time="0000-12-04 14:19:11.691 +0000 UTC"},Tag{Tag="tagged"},Text{Text="message"},Extra{A="a",B="b"}`
	page := memo.Transcribe(ctx)
	page.Sort()
	assert.For("Canonical").That(note.Canonical.Print(page)).Equals(expect)
	expect = `Dec  4 14:19:11:tagged:message`
	assert.For("Normal").That(note.Normal.Print(page)).Equals(expect)
}

func TestNil(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	ctx = keys.WithValue(ctx, "A", "a")
	ctx = keys.WithValue(ctx, "B", nil)
	ctx = memo.Print(ctx, "message")
	expect := `Text{Text="message"},Extra{A="a"}`
	page := memo.Transcribe(ctx)
	page.Sort()
	assert.For("Canonical").That(note.Canonical.Print(page)).Equals(expect)
	expect = `message`
	assert.For("Normal").That(note.Normal.Print(page)).Equals(expect)
}

func TestFrom(t *testing.T) {
	assert := assert.To(t)
	testTime, _ := time.Parse(time.Stamp, "Jan  8 10:21:53.283")
	ctx := context.Background()
	ctx = keys.WithValue(ctx, "A", "a")
	ctx = keys.WithValue(ctx, "B", "b")
	ctx = keys.WithValue(ctx, memo.TimeKey, testTime)
	ctx = memo.Print(ctx, "message")
	expect := `Time{Time="0000-01-08 10:21:53.283 +0000 UTC"},Text{Text="message"},Extra{A="a",B="b"}`
	page, ok := memo.From(ctx)
	assert.For("unfiltered").That(ok).Equals(true)
	page.Sort()
	assert.For("Canonical").That(note.Canonical.Print(page)).Equals(expect)
	expect = "Jan  8 10:21:53:message"
	assert.For("Normal").That(note.Normal.Print(page)).Equals(expect)
	ctx = keys.WithValue(ctx, Filter("C"), true)
	expect = ""
	page, ok = memo.From(ctx)
	assert.For("filtered").That(ok).Equals(false)
	page.Sort()
	assert.For("Canonical").That(note.Canonical.Print(page)).Equals(expect)
}
