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
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/text/note"
)

func TestCollectAll(t *testing.T) {
	assert := assert.To(t)
	pad := note.Pad{}
	handler := note.CollectAll(&pad)
	for _, test := range testPages {
		handler(test.page)
	}
	assert.For("page count").ThatSlice(pad).IsLength(len(testPages))
	for i, page := range pad {
		got := note.Canonical.Print(page)
		assert.For("page %d %s", i, testPages[i].name).That(got).Equals(testPages[i].unsorted)
	}
}

func TestCollect(t *testing.T) {
	assert := assert.To(t)
	pad := note.Pad{}
	limit := 3
	handler := note.Collect(&pad, limit)
	for i, test := range testPages {
		err := handler(test.page)
		if i >= limit {
			assert.For(test.name).That(err).Equals(note.ErrNoteLimit)
			assert.For(test.name).ThatString(err.Error()).Contains("limit reached")
		} else {
			assert.For(test.name).ThatError(err).Succeeded()
		}
	}
	for i, page := range pad {
		got := note.Canonical.Print(page)
		assert.For(testPages[i].name).That(got).Equals(testPages[i].unsorted)
	}

}
