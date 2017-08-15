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
	value     interface{}
	nextLevel map[uint64]*SubCmdIdxTrie
}

func (t *SubCmdIdxTrie) index(indices SubCmdIdx, indexCb func(*SubCmdIdxTrie),
	preOrderCb, postOrderCb func(*SubCmdIdxTrie, SubCmdIdx)) bool {
	if len(indices) == 0 {
		indexCb(t)
		return true
	}
	var ret bool
	if preOrderCb != nil {
		preOrderCb(t, indices)
	}
	if next, ok := t.nextLevel[indices[0]]; ok {
		ret = next.index(indices[1:], indexCb, preOrderCb, postOrderCb)
	} else {
		return false
	}
	if postOrderCb != nil {
		postOrderCb(t, indices)
	}
	return ret
}

// Value returnes the value stored in the trie indexed by the given SubCmdIdx.
// if no value is found by the given SubCmdIdx, returns nil.
func (t *SubCmdIdxTrie) Value(indices SubCmdIdx) interface{} {
	var n *SubCmdIdxTrie
	t.index(indices, func(t *SubCmdIdxTrie) { n = t }, nil, nil)
	if n != nil {
		return n.value
	}
	return nil
}

// SetValue sets a value to the trie with the given SubCmdIdx as index.
func (t *SubCmdIdxTrie) SetValue(indices SubCmdIdx, v interface{}) {
	t.index(indices, func(t *SubCmdIdxTrie) { t.value = v },
		func(t *SubCmdIdxTrie, indices SubCmdIdx) {
			if t.nextLevel == nil {
				t.nextLevel = map[uint64]*SubCmdIdxTrie{}
			}
			if _, ok := t.nextLevel[indices[0]]; !ok {
				t.nextLevel[indices[0]] = &SubCmdIdxTrie{}
			}
		}, nil)
}

// RemoveValue tries to remove a value indexed by the given SubCmdIdx in the
// trie. If a value is found, removes it and returns true. If a value with that
// SubCmdIdx is not found, returns false.
func (t *SubCmdIdxTrie) RemoveValue(indices SubCmdIdx) bool {
	hadValue := false
	r := t.index(indices, func(t *SubCmdIdxTrie) {
		hadValue = t.value != nil
		t.value = nil
	}, nil, func(t *SubCmdIdxTrie, indices SubCmdIdx) {
		if n, ok := t.nextLevel[indices[0]]; ok {
			if n.value == nil && n.nextLevel == nil {
				delete(t.nextLevel, indices[0])
			}
		}
		if len(t.nextLevel) == 0 {
			t.nextLevel = nil
		}
	})
	return r && hadValue
}
