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

package cst

import (
	"io"
)

// Branch is a CST node that can have children.
type Branch struct {
	NodeBase
	// Children is the slice of child nodes for this Branch.
	Children []Node
}

// First returns the first child of the branch, or nil if the branch has no
// children.
func (n *Branch) First() Node {
	if n == nil || len(n.Children) == 0 {
		return nil
	}
	return n.Children[0]
}

// Last returns the last child of the branch, or nil if the branch has no
// children.
func (n *Branch) Last() Node {
	if n == nil || len(n.Children) == 0 {
		return nil
	}
	return n.Children[len(n.Children)-1]
}

// Tok returns the underlying token of this node.
func (n *Branch) Tok() Token {
	tok := Token{}
	if len(n.Children) > 0 {
		first := n.Children[0].Tok()
		last := n.Children[len(n.Children)-1].Tok()
		tok = first
		tok.End = last.End
	}
	return tok
}

// Write writes the branch node to the writer w.
func (n *Branch) Write(w io.Writer) error {
	if err := n.Pre.Write(w); err != nil {
		return err
	}
	for _, c := range n.Children {
		if err := c.Write(w); err != nil {
			return err
		}
	}
	return n.Post.Write(w)
}
