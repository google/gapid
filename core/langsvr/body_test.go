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

package langsvr_test

import (
	"testing"

	"github.com/google/gapid/core/assert"

	ls "github.com/google/gapid/core/langsvr"
)

func TestBody(t *testing.T) {
	assert := assert.To(t)
	body := ls.NewBody("The quick\n" +
		/* 10 */ "brown fox jumps over\n" +
		/* 31 */ "\n" +
		/* 32 */ "the lazy dog")
	for _, test := range []struct {
		offset   int
		position ls.Position
	}{
		{offset: 0, position: ls.Position{Line: 1, Column: 1}},
		{offset: 1, position: ls.Position{Line: 1, Column: 2}},
		{offset: 2, position: ls.Position{Line: 1, Column: 3}},

		{offset: 9, position: ls.Position{Line: 1, Column: 10}},
		{offset: 10, position: ls.Position{Line: 2, Column: 1}},
		{offset: 11, position: ls.Position{Line: 2, Column: 2}},

		{offset: 30, position: ls.Position{Line: 2, Column: 21}},
		{offset: 31, position: ls.Position{Line: 3, Column: 1}},
		{offset: 32, position: ls.Position{Line: 4, Column: 1}},
		{offset: 33, position: ls.Position{Line: 4, Column: 2}},

		{offset: 44, position: ls.Position{Line: 4, Column: 13}},
	} {
		assert.For("position %d", test.offset).That(body.Position(test.offset)).Equals(test.position)
		assert.For("offset %v", test.position).That(body.Offset(test.position)).Equals(test.offset)
	}

	assert.For("offset").That(body.Position(1000)).Equals(ls.Position{Line: 4, Column: 13})
	assert.For("position").That(body.Offset(ls.Position{Line: 1, Column: 500})).Equals(9)
	assert.For("position").That(body.Offset(ls.Position{Line: 2, Column: 500})).Equals(30)
	assert.For("position").That(body.Offset(ls.Position{Line: 3, Column: 500})).Equals(31)
	assert.For("position").That(body.Offset(ls.Position{Line: 4, Column: 500})).Equals(44)
	assert.For("position").That(body.Offset(ls.Position{Line: 500, Column: 1})).Equals(44)
	assert.For("position").That(body.Offset(ls.Position{Line: 500, Column: 500})).Equals(44)
}
