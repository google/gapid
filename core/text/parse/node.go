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

package parse

import "github.com/google/gapid/core/data/compare"

// Node is a Fragment in a cst that represents unskipped tokens.
type Node interface {
	Fragment
	// Parent returns the Branch that this node is under.
	Parent() *Branch
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

type node struct {
	parent *Branch
	prefix Separator
	suffix Separator
}

func (n *node) Parent() *Branch {
	return n.parent
}

func (n *node) Prefix() Separator {
	return n.prefix
}

func (n *node) AddPrefix(s Separator) {
	n.prefix = append(n.prefix, s...)
}

func (n *node) Suffix() Separator {
	return n.suffix
}

func (n *node) AddSuffix(s Separator) {
	n.suffix = append(n.suffix, s...)
}

func compareNodes(c compare.Comparator, reference, value node) {
	c.With(c.Path.Member("prefix", reference, value)).Compare(reference.prefix, value.prefix)
	c.With(c.Path.Member("suffix", reference, value)).Compare(reference.suffix, value.suffix)
}

func init() {
	compare.Register(compareNodes)
}
