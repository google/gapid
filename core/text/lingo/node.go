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

package lingo

import "fmt"

// Record is an entry in a Records list that holds a parse result along with its scan stream bounds.
type Record struct {
	Start  int
	End    int
	Object interface{}
}

// Records is a list of Record objects that represent the ordered sequence of parse results.
type Records []Record

// Node is the form used when Records are reconstituted into a
type Node struct {
	Start    int
	End      int
	Object   interface{}
	Parent   *Node
	Children []*Node
}

// ToCST converts from a record list to a node span tree.
func (r Records) ToCST() *Node {
	root := &Node{
		Start: 0,
		End:   len(r),
	}
	active := root
	for _, entry := range r {
		for entry.Start >= active.End && active.Parent != nil {
			active = active.Parent
		}
		child := &Node{
			Start:  entry.Start,
			End:    entry.End,
			Object: entry.Object,
			Parent: active,
		}
		active.Children = append(active.Children, child)
		active = child
	}
	return root
}

// Format implements fmt.Formatter to print the full record list.
func (r Records) Format(f fmt.State, _ rune) {
	for _, entry := range r {
		fmt.Fprint(f, entry.Object)
	}
}

// Format implements fmt.Formatter to print the leaf nodes in depth first order.
func (n *Node) Format(f fmt.State, _ rune) {
	if len(n.Children) > 0 {
		for _, child := range n.Children {
			fmt.Fprint(f, child)
		}
	} else if n.Object != nil {
		fmt.Fprint(f, n.Object)
	}
}
