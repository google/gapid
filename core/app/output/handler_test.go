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

package output_test

import (
	"context"
	"testing"

	"github.com/google/gapid/core/app/output"
	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/text/note"
)

func TestHandlerContext(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	id := 0
	ctx = output.NewContext(ctx, func(note.Page) error { id = 1; return nil })
	output.Send(ctx, note.Page{})
	assert.For("handler install").That(id).Equals(1)
	ctx = output.NewContext(ctx, func(note.Page) error { id = 2; return nil })
	output.Send(ctx, note.Page{})
	assert.For("handler override").That(id).Equals(2)
}

func TestHandlerDefault(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	id := 0
	output.Default = func(note.Page) error { id = 1; return nil }
	output.Send(ctx, note.Page{})
	assert.For("initial default install").That(id).Equals(1)
	output.Default = func(note.Page) error { id = 2; return nil }
	output.Send(ctx, note.Page{})
	assert.For("replaced default handler").That(id).Equals(2)
}

type handlerTest struct {
	severity severity.Level
	message  string
	expect   string
}

var tests = []handlerTest{
	{severity.Emergency, "A", "Emergency:A"},
	{severity.Alert, "B", "Alert:B"},
	{severity.Critical, "C", "Critical:C"},
	{severity.Error, "D", "Error:D"},
	{severity.Warning, "E", "Warning:E"},
	{severity.Notice, "F", "Notice:F"},
	{severity.Info, "G", "Info:G"},
	{severity.Debug, "H", "Debug:H"},
}

func testHandler(ctx context.Context, test handlerTest) {
	page := note.Page{}
	page.Append(note.SectionInfo{Order: 1}, nil, test.message)
	severity.LevelKey.Transcribe(ctx, &page, test.severity)
	page.Sort()
	output.Send(ctx, page)
}
