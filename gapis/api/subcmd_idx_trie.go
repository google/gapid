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

// SubCmdIdxTrie is a map-based trie using SubCmdIdx for indexing the data
// stored inside.
type SubCmdIdxTrie struct {
	hasValue  bool
	value     interface{}
	nextLevel map[uint64]*SubCmdIdxTrie
}

// Constructs a new SubCmdIdxTrie instance and returns a pointer to the instance.
func NewSubCmdIdxTrie() *SubCmdIdxTrie {
	return &SubCmdIdxTrie{
		hasValue:  false,
		nextLevel: map[uint64]*SubCmdIdxTrie{},
	}
}

// Tries to get the value stored in the trie by the given SubCmdIdx. If a value
// is stored in the trie for the given SubCmdIdx, returns the value and true,
// otherwise returns nil and false.
func (trie *SubCmdIdxTrie) GetValue(indices SubCmdIdx) (interface{}, bool) {
	t := trie
	for _, k := range indices {
		_, ok := t.nextLevel[k]
		if ok {
			t = t.nextLevel[k]
		} else {
			return nil, false
		}
	}
	if t.hasValue {
		return t.value, true
	}
	return nil, false
}

// Sets a value to the trie with the given SubCmdIdx as index
func (trie *SubCmdIdxTrie) SetValue(indices SubCmdIdx, v interface{}) {
	t := trie
	for _, k := range indices {
		_, ok := t.nextLevel[k]
		if ok {
		} else {
			t.nextLevel[k] = NewSubCmdIdxTrie()
		}
		t = t.nextLevel[k]
	}
	t.value = v
	t.hasValue = true
}

// Tries to remove a value indexed by the given SubCmdIdx in the trie. If a
// value is found, removes it and returns true. If a value with that SubCmdIdx
// is not found, returns false. Removing the value from a parent node won't
// remove the child nodes and their value, i.e.: Removing with [0,1] won't
// erase the value and the node like: [0, 1, 2].
func (trie *SubCmdIdxTrie) RemoveValue(indices SubCmdIdx) bool {
	t := trie
	for _, k := range indices {
		_, ok := t.nextLevel[k]
		if ok {
			t = t.nextLevel[k]
		} else {
			return false
		}
	}
	t.hasValue = false
	return true
}

// Tries to remove a trie node indexed by the given SubCmdIdx. If a node is
// is found, removes it can all its child nodes, and returns true. Otherwise,
// returns false.
func (trie *SubCmdIdxTrie) RemoveNode(indices SubCmdIdx) bool {
	if len(indices) == 0 {
		return false
	}
	t := trie
	for i, k := range indices {
		if i == len(indices)-1 {
			if _, ok := t.nextLevel[k]; ok {
				delete(t.nextLevel, k)
				return true
			}
		}
		_, ok := t.nextLevel[k]
		if ok {
			t = t.nextLevel[k]
		} else {
			return false
		}
	}
	return false
}
