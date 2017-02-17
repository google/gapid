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

package dom

import "github.com/gopherjs/gopherjs/js"

// Node is the interface implemented by DOM node types.
type Node interface {
	// Children returns all the child nodes of this node.
	Children() []Node

	// AppendNode adds the child node at the end of the list of children.
	AppendNode(Node)

	obj() *js.Object
}

type node struct{ *js.Object }

func (n node) Children() []Node {
	nodes := n.Object.Get("childNodes")
	out := make([]Node, nodes.Length())
	for i := range out {
		out[i] = node{Object: nodes.Index(i)}
	}
	return out
}

func (n node) AppendNode(child Node) {
	n.Object.Call("appendChild", child.obj())
}

func (n node) obj() *js.Object {
	return n.Object
}
