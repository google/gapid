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
	"io"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/text/note"
)

func TestWriter(t *testing.T) {
	assert := assert.To(t)
	pad := note.Pad{}
	w := note.Writer(note.CollectAll(&pad))
	input := []string{
		"A short part",
		" of a long line\n",
		"And now",
		" a split line\nAnd another one too\n",
		"And finally",
		" fragments",
		" with no",
		" newlines",
	}
	expect := []string{
		"A short part of a long line",
		"And now a split line",
		"And another one too",
		"And finally fragments with no newlines",
	}
	for _, in := range input {
		io.WriteString(w, in)
	}
	w.Close()
	assert.For("captured lines").ThatSlice(pad).IsLength(len(expect))
	for i, expect := range expect {
		got := note.Normal.Print(pad[i])
		assert.For("line %d", i).That(got).Equals(expect)
	}
}
