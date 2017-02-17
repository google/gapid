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

package severity_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/text/note"
)

func TestFork(t *testing.T) {
	ctx := context.Background()
	page := note.Page{{Content: []note.Item{
		{Key: "A", Value: 1},
	}}, {SectionInfo: note.SectionInfo{Relevance: note.Relevant, Multiline: true}, Content: []note.Item{
		{Key: "B", Value: note.Page{{Content: []note.Item{
			{Key: "C", Value: 2},
		}}}},
	}}, {Content: []note.Item{
		{Key: "D", Value: 3},
	}}}

	buf := &bytes.Buffer{}
	normal := note.Normal.Scribe(buf)
	handle := severity.Split(
		note.Raw.Scribe(buf),
		severity.Splits{
			severity.Info:   normal,
			severity.Debug:  note.Brief.Scribe(buf),
			severity.Error:  note.Detailed.Scribe(buf),
			severity.Notice: normal,
		})
	for level := severity.Emergency; level <= severity.Debug; level++ {
		buf.Reset()
		expect := ""
		switch level {
		case severity.Error:
			expect = "A=1\n    B = C=2\nD=3:Error"
		case severity.Notice:
			expect = "A=1:B={C=2}:D=3:Notice"
		case severity.Info:
			expect = "A=1:B={C=2}:D=3:Info"
		case severity.Debug:
			expect = "A=1:D=3:Debug"
		default:
			expect = "1:3"
		}
		page := page.Clone()
		severity.LevelKey.Transcribe(ctx, &page, level)
		handle(page)
		if buf.String() != expect {
			t.Errorf("For %s got %q expected %q", level, buf.String(), expect)
		}
	}
}
