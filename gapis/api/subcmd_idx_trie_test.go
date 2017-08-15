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

package api

import (
	"testing"

	"github.com/google/gapid/core/assert"
)

func TestSubCmdIdxTrie(t *testing.T) {
	trie := &SubCmdIdxTrie{}

	expectValue := func(indices SubCmdIdx, expectedValue int) {
		v := trie.Value(indices)
		assert.To(t).For("Expect Value(%v) value: %v", indices, expectedValue).That(v).Equals(expectedValue)
	}
	expectNonValue := func(indices SubCmdIdx) {
		v := trie.Value(indices)
		assert.To(t).For("Expect Value(%v) value: %v", indices, v).That(v).Equals(nil)
	}
	expectRemoveValue := func(indices SubCmdIdx, expectedReturn bool) {
		assert.To(t).For("Expect RemoveValue(%v) returns: %v", indices, expectedReturn).That(trie.RemoveValue(indices)).Equals(expectedReturn)
	}

	expectNonValue(SubCmdIdx{})
	expectNonValue(SubCmdIdx{0})
	expectNonValue(SubCmdIdx{1})
	expectNonValue(SubCmdIdx{1, 2, 3})

	trie.SetValue(SubCmdIdx{}, 100)
	expectValue(SubCmdIdx{}, 100)
	expectNonValue(SubCmdIdx{0})
	expectNonValue(SubCmdIdx{1})
	expectNonValue(SubCmdIdx{1, 2, 3})

	trie.SetValue(SubCmdIdx{1}, 101)
	expectValue(SubCmdIdx{}, 100)
	expectValue(SubCmdIdx{1}, 101)
	expectNonValue(SubCmdIdx{0})
	expectNonValue(SubCmdIdx{1, 2, 3})

	trie.SetValue(SubCmdIdx{1, 2, 3, 4, 5, 6}, 102)
	trie.SetValue(SubCmdIdx{100, 99, 98, 97}, 103)
	expectValue(SubCmdIdx{1, 2, 3, 4, 5, 6}, 102)
	expectValue(SubCmdIdx{100, 99, 98, 97}, 103)
	expectValue(SubCmdIdx{}, 100)
	expectValue(SubCmdIdx{1}, 101)
	expectNonValue(SubCmdIdx{0})
	expectNonValue(SubCmdIdx{1, 2, 3})

	expectRemoveValue(SubCmdIdx{1}, true)
	expectNonValue(SubCmdIdx{1})
	expectValue(SubCmdIdx{1, 2, 3, 4, 5, 6}, 102)

	expectRemoveValue(SubCmdIdx{}, true)
	expectNonValue(SubCmdIdx{})
	expectValue(SubCmdIdx{1, 2, 3, 4, 5, 6}, 102)
	expectValue(SubCmdIdx{100, 99, 98, 97}, 103)

	expectRemoveValue(SubCmdIdx{100, 99}, false)
	expectValue(SubCmdIdx{100, 99, 98, 97}, 103)
	expectNonValue(SubCmdIdx{100, 99})

	expectRemoveValue(SubCmdIdx{100, 99, 98, 97}, true)
	expectNonValue(SubCmdIdx{100, 99, 98, 97})
	expectNonValue(SubCmdIdx{100, 99, 98})
	expectNonValue(SubCmdIdx{100, 99})
	expectNonValue(SubCmdIdx{100})
}
