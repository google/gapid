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

package snippets

import (
	"fmt"

	"github.com/google/gapid/framework/binary"
)

// pathway is way of finding a static component in a nested structure.
type Pathway interface {
	binary.Object
	isPath() // tag method

	// root returns the root of this pathway
	root() Pathway

	// getBase returns the base of this pathway or nil if it is the root.
	getBase() Pathway
}

// pathwayCast needed for generated code.
func PathwayCast(o binary.Object) Pathway {
	return o.(Pathway)
}

// namepath is a path representing a symbol name in a particular category
// (local, global, param). A namePath can be the root of a path.
type namePath struct {
	binary.Generate
	cat  SymbolCategory // category of this symbol (local, global, param)
	name string         // name of this symbol
}

func (*namePath) isPath() {}
func (n *namePath) root() Pathway {
	return n
}

func (n *namePath) getBase() Pathway {
	return nil
}

var _ Pathway = &namePath{}

// fieldPath is a path representing a field in an entity.
type fieldPath struct {
	binary.Generate
	base Pathway // Pathway to the entity.
	name string  // name of the field.
}

func (*fieldPath) isPath() {}
func (p *fieldPath) root() Pathway {
	return p.base.root()
}

func (p *fieldPath) getBase() Pathway {
	return p.base
}

var _ Pathway = &fieldPath{}

// partPath is a path representing a component of a container.
type partPath struct {
	binary.Generate
	base Pathway  // Pathway to the container.
	kind PartKind // kind of the component.
}

func (*partPath) isPath() {}
func (p *partPath) root() Pathway {
	return p.base.root()
}

func (p *partPath) getBase() Pathway {
	return p.base
}

var _ Pathway = &partPath{}

// relativePath is a path representing a symbol relative to a particular
// schema entity. A relativePath can be the root of a path.
type relativePath struct {
	binary.Generate

	// The name of the type as it is in the API file. The type of the global
	// state object is synthetic at determined by the tag "globals" in the
	// generated code.
	typeName string
}

func (*relativePath) isPath() {}
func (n *relativePath) root() Pathway {
	return n
}

func (n *relativePath) getBase() Pathway {
	return nil
}

var _ Pathway = &relativePath{}

func (n *namePath) String() string {
	return fmt.Sprintf("%s:%s", n.cat, n.name)
}

func (f *fieldPath) String() string {
	return fmt.Sprintf("%s.%s", f.base, f.name)
}

func (f *partPath) String() string {
	return fmt.Sprintf("%s.%s()", f.base, f.kind)
}

func (n *relativePath) String() string {
	return fmt.Sprintf("%s:", n.typeName)
}

// MakeRelative return a copy of p with the root replaced
// by a relative root for typeName.
func MakeRelative(p Pathway, typeName string) Pathway {
	switch p := p.(type) {
	case *partPath:
		return &partPath{base: MakeRelative(p.getBase(), typeName), kind: p.kind}
	case *fieldPath:
		return Field(MakeRelative(p.getBase(), typeName), p.name)
	case *namePath:
		return Field(Relative(typeName), p.name)
	case *relativePath:
		return Relative(typeName)
	default:
		panic(fmt.Errorf("Unexpected Pathway type %T in MakeRelative(%v, %s)", p, p, typeName))
	}
}

// Make a relative pathway for the API type named typeName
func Relative(typeName string) Pathway {
	return &relativePath{typeName: typeName}
}

// Variable returns a Pathway for a symbol named name in category cat.
func Variable(cat SymbolCategory, name string) Pathway {
	return &namePath{cat: cat, name: name}
}

// Elem returns a Pathway to the element of a collection.
func Elem(p Pathway) Pathway {
	return &partPath{base: p, kind: PartKind_Elem}
}

// Key returns a Pathway to the element of a collection.
func Key(p Pathway) Pathway {
	return &partPath{base: p, kind: PartKind_Key}
}

// Field returns a Pathway to a field of an entity.
func Field(p Pathway, name string) Pathway {
	return &fieldPath{base: p, name: name}
}

// Range returns a Pathway to a range of a slice or pointer.
func Range(p Pathway) Pathway {
	return &partPath{base: p, kind: PartKind_Range}
}

func Equal(left, right Pathway) bool {
	switch l := left.(type) {
	case *namePath:
		r, ok := right.(*namePath)
		return ok && *l == *r
	case *fieldPath:
		r, ok := right.(*fieldPath)
		return ok && l.name == r.name && Equal(l.base, r.base)
	case *partPath:
		r, ok := right.(*partPath)
		return ok && l.kind == r.kind && Equal(l.base, r.base)
	case *relativePath:
		r, ok := right.(*relativePath)
		return ok && l.typeName == r.typeName
	}
	return false
}

func IsGlobal(p Pathway) bool {
	switch root := p.root().(type) {
	case *namePath:
		return root.cat == SymbolCategory_Global
	case *relativePath:
		panic(fmt.Errorf("Relative path %v used with IsGlobal", p))
	default:
		panic(fmt.Errorf("Unexpected Pathway root type %T in IsGlobal(%v)", p, p))
	}
}
