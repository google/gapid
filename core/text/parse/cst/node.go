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

import "github.com/google/gapid/core/data/compare"

// Node is a Fragment in a cst that represents unskipped tokens.
type Node interface {
	Fragment
	// Parent returns the Branch that this node is under.
	Parent() *Branch
	// SetParent replaces the Branch that this node is under.
	SetParent(*Branch)
	// Prefix returns the set of skippable fragments associated with this Node
	// that precede it in the stream. Association is defined by the Skip function
	// in use.
	Prefix() Separator
	// AddPrefix adds more fragments to the Prefix list.
	AddPrefix(Separator)
	// Suffix returns the set of skippable fragments associated with this Node
	// that follow it in the stream. Association is defined by the Skip function
	// in use, the default is until the end of the line.
	Suffix() Separator
	// AddSuffix adds more fragments to the Suffix list.
	AddSuffix(Separator)
}

// NodeBase implements the non-fragment parts of the Node interface.
// NodeBase is intended to be used as an anonymous field of types that implement
// the Node interface.
type NodeBase struct {
	Branch *Branch
	Pre    Separator
	Post   Separator
}

// Parent returns the Branch that this node is under.
func (n *NodeBase) Parent() *Branch {
	return n.Branch
}

// SetParent replaces the Branch that this node is under.
func (n *NodeBase) SetParent(parent *Branch) {
	n.Branch = parent
}

// Prefix returns the set of skippable fragments associated with this Node
// that precede it in the stream. Association is defined by the Skip function
// in use.
func (n *NodeBase) Prefix() Separator {
	return n.Pre
}

// AddPrefix adds more fragments to the Prefix list.
func (n *NodeBase) AddPrefix(s Separator) {
	n.Pre = append(n.Pre, s...)
}

// Suffix returns the set of skippable fragments associated with this Node
// that follow it in the stream. Association is defined by the Skip function
// in use, the default is until the end of the line.
func (n *NodeBase) Suffix() Separator {
	return n.Post
}

// AddSuffix adds more fragments to the Suffix list.
func (n *NodeBase) AddSuffix(s Separator) {
	n.Post = append(n.Post, s...)
}

func compareNodes(c compare.Comparator, reference, value NodeBase) {
	c.With(c.Path.Member("Prefix", reference, value)).Compare(reference.Pre, value.Pre)
	c.With(c.Path.Member("Suffix", reference, value)).Compare(reference.Post, value.Post)
}

func init() {
	compare.Register(compareNodes)
}
