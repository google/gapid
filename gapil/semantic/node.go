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

package semantic

import "github.com/google/gapid/gapil/ast"

// Node represents any semantic-tree node type.
type Node interface {
	isNode() // A placeholder function that's implemented by all semantic node types.
}

// NamedNode represents any semantic-tree node that carries a name.
type NamedNode interface {
	Node
	Name() string // Returns the partial name of the object.
}

// Owned is the interface to an object with a unique name and an owner.
type Owned interface {
	NamedNode
	// Owner returns the owner of this node.
	Owner() Owner
	// setOwner sets the owner of this node.
	setOwner(Owner)
}

// Owner is the interface for an object that has named Members.
type Owner interface {
	NamedNode
	// Member looks up a member by name from an owner.
	Member(string) Owned
	// VisitMembers invokes the supplied function once for each member.
	VisitMembers(func(Owned))
	// SortMembers sorts the members alphabetically.
	SortMembers()
	// addMember adds a child to an owner.
	addMember(Owned)
}

type ASTBacked interface {
	ASTNode() ast.Node
}

// Add connects an Owned to its Owner.
func Add(p Owner, c Owned) {
	p.addMember(c)
	c.setOwner(p)
}

// Named is mixed in to implement the Name method of NamedNode.
type Named string

func (n Named) Name() string { return string(n) }

type owned struct {
	owner Owner
}

func (o *owned) Owner() Owner         { return o.owner }
func (o *owned) setOwner(owner Owner) { o.owner = owner }

type members Symbols

func (m *members) Member(name string) Owned {
	n, err := (*Symbols)(m).Find(name)
	if err != nil {
		// TODO: propagate errors from this function
		return nil
	}
	o, _ := n.(Owned)
	return o
}

func (m *members) addMember(child Owned) {
	(*Symbols)(m).AddNamed(child)
}

// VisitMembers invokes the supplied function once for each member.
func (m *members) VisitMembers(visitor func(Owned)) {
	m.SortMembers()
	for _, e := range (*Symbols)(m).entries {
		visitor(e.node.(Owned))
	}
}

// SortMembers sorts the members alphabetically.
func (m *members) SortMembers() { (*Symbols)(m).sort() }

type noAddMembers struct{}

func (noAddMembers) addMember(Owned) { panic("Not allowed members") }

type noMembers struct{ noAddMembers }

func (noMembers) Member(string) Owned      { return nil }
func (noMembers) VisitMembers(func(Owned)) {}
func (noMembers) SortMembers()             {}
