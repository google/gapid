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

package note_test

import (
	"bytes"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/text/note"
)

func TestSort(t *testing.T) {
	assert := assert.To(t)
	buf := &bytes.Buffer{}
	handler := note.Cloner(note.Sorter(note.Canonical.Scribe(buf)))
	for _, test := range testPages {
		got := note.Canonical.Print(test.page)
		assert.For(test.name).That(got).Equals(test.unsorted)
		buf.Reset()
		handler(test.page)
		got = buf.String()
		assert.For(test.name).That(got).Equals(test.sorted)
	}
}

func TestAppend(t *testing.T) {
	assert := assert.To(t)
	page := note.Page{}
	got := note.Canonical.Print(page)
	expect := ""
	assert.For("empty page").That(got).Equals(expect)
	tests := note.SectionInfo{Key: "Tests"}
	extras := note.SectionInfo{Key: "Extras"}
	page = append(page, note.Section{SectionInfo: tests})
	got = note.Canonical.Print(page)
	assert.For("empty section").That(got).Equals(expect)
	for _, test := range []struct {
		name    string
		section note.SectionInfo
		key     string
		value   string
		expect  string
	}{
		{"first item", tests, "A", "a", `Tests{A="a"}`},
		{"second item", tests, "B", "b", `Tests{A="a",B="b"}`},
		{"second section", extras, "C", "c", `Tests{A="a",B="b"},Extras{C="c"}`},
		{"append second section", extras, "D", "d", `Tests{A="a",B="b"},Extras{C="c",D="d"}`},
		{"append first section", tests, "E", "e", `Tests{A="a",B="b",E="e"},Extras{C="c",D="d"}`},
	} {
		page.Append(test.section, test.key, test.value)
		got := note.Canonical.Print(page)
		assert.For(test.name).That(got).Equals(test.expect)
	}
}
