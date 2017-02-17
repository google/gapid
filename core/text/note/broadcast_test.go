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

func TestBroadcast(t *testing.T) {
	assert := assert.To(t)
	aBuf := &bytes.Buffer{}
	bBuf := &bytes.Buffer{}
	cBuf := &bytes.Buffer{}
	handle := note.Broadcast(
		note.Raw.Scribe(aBuf),
		note.Detailed.Scribe(bBuf),
		note.Normal.Scribe(cBuf))
	for _, test := range testPages {
		aBuf.Reset()
		bBuf.Reset()
		cBuf.Reset()
		handle(test.page)
		assert.For("%s A", test.name).That(aBuf.String()).Equals(test.raw)
		assert.For("%s B", test.name).That(bBuf.String()).Equals(test.detailed)
		assert.For("%s C", test.name).That(cBuf.String()).Equals(test.normal)
	}
}
