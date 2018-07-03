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

package api_test

import (
	"reflect"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/gapis/api"
)

func TestSubCmdIdxTrie(t *testing.T) {
	assert := assert.To(t)
	trie := &api.SubCmdIdxTrie{}

	expectValue := func(indices api.SubCmdIdx, expectedValue int) {
		v := trie.Value(indices)
		assert.For("Expect Value(%v) value: %v", indices, expectedValue).That(v).Equals(expectedValue)
	}
	expectNonValue := func(indices api.SubCmdIdx) {
		v := trie.Value(indices)
		assert.For("Expect Value(%v) value: %v", indices, v).That(v).Equals(nil)
	}
	expectRemoveValue := func(indices api.SubCmdIdx, expectedReturn bool) {
		assert.For("Expect RemoveValue(%v) returns: %v", indices, expectedReturn).That(trie.RemoveValue(indices)).Equals(expectedReturn)
	}

	expectNonValue(api.SubCmdIdx{})
	expectNonValue(api.SubCmdIdx{0})
	expectNonValue(api.SubCmdIdx{1})
	expectNonValue(api.SubCmdIdx{1, 2, 3})

	trie.SetValue(api.SubCmdIdx{}, 100)
	expectValue(api.SubCmdIdx{}, 100)
	expectNonValue(api.SubCmdIdx{0})
	expectNonValue(api.SubCmdIdx{1})
	expectNonValue(api.SubCmdIdx{1, 2, 3})

	trie.SetValue(api.SubCmdIdx{1}, 101)
	expectValue(api.SubCmdIdx{}, 100)
	expectValue(api.SubCmdIdx{1}, 101)
	expectNonValue(api.SubCmdIdx{0})
	expectNonValue(api.SubCmdIdx{1, 2, 3})

	trie.SetValue(api.SubCmdIdx{1, 2, 3, 4, 5, 6}, 102)
	trie.SetValue(api.SubCmdIdx{100, 99, 98, 97}, 103)
	expectValue(api.SubCmdIdx{1, 2, 3, 4, 5, 6}, 102)
	expectValue(api.SubCmdIdx{100, 99, 98, 97}, 103)
	expectValue(api.SubCmdIdx{}, 100)
	expectValue(api.SubCmdIdx{1}, 101)
	expectNonValue(api.SubCmdIdx{0})
	expectNonValue(api.SubCmdIdx{1, 2, 3})

	expectRemoveValue(api.SubCmdIdx{1}, true)
	expectNonValue(api.SubCmdIdx{1})
	expectValue(api.SubCmdIdx{1, 2, 3, 4, 5, 6}, 102)

	expectRemoveValue(api.SubCmdIdx{}, true)
	expectNonValue(api.SubCmdIdx{})
	expectValue(api.SubCmdIdx{1, 2, 3, 4, 5, 6}, 102)
	expectValue(api.SubCmdIdx{100, 99, 98, 97}, 103)

	expectRemoveValue(api.SubCmdIdx{100, 99}, false)
	expectValue(api.SubCmdIdx{100, 99, 98, 97}, 103)
	expectNonValue(api.SubCmdIdx{100, 99})

	expectRemoveValue(api.SubCmdIdx{100, 99, 98, 97}, true)
	expectNonValue(api.SubCmdIdx{100, 99, 98, 97})
	expectNonValue(api.SubCmdIdx{100, 99, 98})
	expectNonValue(api.SubCmdIdx{100, 99})
	expectNonValue(api.SubCmdIdx{100})
}

func TestSubCmdIdxTriePostOrderSortedKeys(t *testing.T) {
	assert := assert.To(t)
	expectKeys := func(trie *api.SubCmdIdxTrie, expectedKeys []api.SubCmdIdx) {
		keys := trie.PostOrderSortedKeys()
		assert.For("Expected returned keys: %v", expectedKeys).That(
			reflect.DeepEqual(keys, expectedKeys)).Equals(true)
	}

	trie := &api.SubCmdIdxTrie{}
	expectKeys(trie, []api.SubCmdIdx{})

	trie.SetValue(api.SubCmdIdx{}, 100)
	expectKeys(trie, []api.SubCmdIdx{
		api.SubCmdIdx{},
	})

	trie.SetValue(api.SubCmdIdx{1}, 101)
	trie.SetValue(api.SubCmdIdx{1, 2, 3, 4, 5, 6}, 102)
	trie.SetValue(api.SubCmdIdx{100, 99, 98, 97}, 103)
	expectKeys(trie, []api.SubCmdIdx{
		api.SubCmdIdx{1, 2, 3, 4, 5, 6},
		api.SubCmdIdx{1},
		api.SubCmdIdx{100, 99, 98, 97},
		api.SubCmdIdx{},
	})

	trie.RemoveValue(api.SubCmdIdx{})
	trie.RemoveValue(api.SubCmdIdx{1})
	trie.RemoveValue(api.SubCmdIdx{1, 2, 3, 4, 5, 6})
	trie.RemoveValue(api.SubCmdIdx{100, 99, 98, 97})
	expectKeys(trie, []api.SubCmdIdx{})

	trie.SetValue(api.SubCmdIdx{0, 1, 2}, true)
	trie.SetValue(api.SubCmdIdx{0, 2}, true)
	trie.SetValue(api.SubCmdIdx{1}, true)
	trie.SetValue(api.SubCmdIdx{1, 2, 3}, true)
	trie.SetValue(api.SubCmdIdx{0, 1}, true)
	expectKeys(trie, []api.SubCmdIdx{
		api.SubCmdIdx{0, 1, 2},
		api.SubCmdIdx{0, 1},
		api.SubCmdIdx{0, 2},
		api.SubCmdIdx{1, 2, 3},
		api.SubCmdIdx{1},
	})
}
