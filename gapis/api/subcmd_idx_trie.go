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

import "sort"

// SubCmdIdxTrie is a map-based trie using SubCmdIdx for indexing the data
// stored inside.
type SubCmdIdxTrie struct {
	value    interface{}
	children map[uint64]*SubCmdIdxTrie
}

// Value returns the value stored in the trie indexed by the given SubCmdIdx.
// if no value is found by the given SubCmdIdx, returns nil.
func (t *SubCmdIdxTrie) Value(indices SubCmdIdx) interface{} {
	for t != nil && len(indices) > 0 && len(t.children) > 0 {
		t, indices = t.children[indices[0]], indices[1:]
	}
	if t == nil || len(indices) > 0 {
		return nil // invalid index
	}
	return t.value
}

// Values returns the values stored in the trie indexed by all prefixes of the
// SubCmdIdx, in increasing order of length; if no value is found for a prefix,
// the result contains `nil` for that prefix.
func (t *SubCmdIdxTrie) Values(indices SubCmdIdx) []interface{} {
	values := make([]interface{}, len(indices)+1)

	for i, ix := range indices {
		values[i] = t.value
		if len(t.children) == 0 {
			t = nil
			break
		}
		t = t.children[ix]
		if t == nil {
			break
		}
	}

	if t != nil {
		values[len(indices)] = t.value
	}
	return values
}

// SetValue sets a value to the trie with the given SubCmdIdx as index.
func (t *SubCmdIdxTrie) SetValue(indices SubCmdIdx, v interface{}) {
	if v == nil { // nil value behaves the same as no value
		t.RemoveValue(indices)
		return
	}
	for _, i := range indices {
		next, ok := t.children[i]
		if !ok { // child does not exist - create it
			if t.children == nil {
				t.children = map[uint64]*SubCmdIdxTrie{}
			}
			next = &SubCmdIdxTrie{}
			t.children[i] = next
		}
		t = next
	}
	t.value = v
}

// RemoveValue tries to remove a value indexed by the given SubCmdIdx in the
// trie. If a value is found, removes it and returns true. If a value with that
// SubCmdIdx is not found, returns false.
func (t *SubCmdIdxTrie) RemoveValue(indices SubCmdIdx) bool {
	n := len(indices)
	stack := make([]*SubCmdIdxTrie, n+1)
	stack[0] = t
	for i, j := range indices { // build a stack of nodes starting with t
		t = t.children[j]
		if t == nil {
			return false // invalid index
		}
		stack[i+1] = t
	}
	if t.value == nil {
		return false // cannot remove a value it doesn't have
	}
	t.value = nil             // clear the value
	for i := n; i >= 0; i-- { // remove nodes that have no value or children
		if t.value != nil || len(t.children) > 0 {
			break // node cannot be deleted
		}
		delete(stack[i].children, indices[i-1])
		t = stack[i]
	}
	return true
}

// PostOrderSortedKeys returns the keys of the value stored in the trie, the
// keys will be sorted in the post traversal order and lesser to greater. e.g.:
// [0, 1, 2], [0, 2], [1], [1, 2, 3], [0, 1] will be sorted as:
// [0, 1, 2], [0, 1], [0, 2], [1, 2, 3], [1]
func (t *SubCmdIdxTrie) PostOrderSortedKeys() []SubCmdIdx {
	ks := make([]uint64, len(t.children))
	i := 0
	for k, _ := range t.children {
		ks[i] = k
		i++
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })

	keys := []SubCmdIdx{}
	for _, k := range ks {
		c := t.children[k]
		cks := c.PostOrderSortedKeys()
		for _, ck := range cks {
			keys = append(keys, SubCmdIdx(append([]uint64{k}, ck...)))
		}
	}
	if t.value != nil {
		keys = append(keys, SubCmdIdx{})
	}
	return keys
}

func (t *SubCmdIdxTrie) GetChildren(index uint64) *SubCmdIdxTrie {
	return t.children[index]
}
