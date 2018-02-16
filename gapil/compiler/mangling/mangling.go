// Copyright (C) 2018 Google Inc.
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

// Package mangling exposes a minimal collection of types used to build mangled
// strings using any of the sub-packages.
package mangling

// Mangler is a function that mangles the entity into a string.
type Mangler func(Entity) string

// Entity is implemented by all types in this package.
type Entity interface {
	isEntity()
}

// Scoped is the interface implemented by entities that can belong to a scope.
type Scoped interface {
	Entity
	Scope() Scope
}

// Named is the interface for entities that have a source name.
type Named interface {
	Entity
	GetName() string
}

// Scope is a namespace or class.
type Scope interface {
	Entity
	isScope()
}

// Type is a POD or Class type.
type Type interface {
	Entity
	isType()
}

// Templated is type that can has template arguments.
type Templated interface {
	Entity
	TemplateArguments() []Type
}

// Builtin is a builtin type.
type Builtin int

const (
	Void = Builtin(iota)
	WChar
	Bool
	Char
	SChar
	UChar
	Short
	UShort
	Int
	UInt
	Long
	ULong
	S64
	U64
	Float
	Double
	Ellipsis
)

func (Builtin) isType()     {}
func (b Builtin) isEntity() {}

// Pointer is a pointer type.
type Pointer struct {
	To Type
}

func (Pointer) isType()     {}
func (p Pointer) isEntity() {}

// TemplateParameter is a template parameter type index.
type TemplateParameter int

func (TemplateParameter) isType()     {}
func (t TemplateParameter) isEntity() {}

// Namespace represents a C++ namespace.
type Namespace struct {
	Parent Scope
	Name   string
}

func (*Namespace) isScope()          {}
func (n *Namespace) Scope() Scope    { return n.Parent }
func (n *Namespace) GetName() string { return n.Name }
func (n *Namespace) isEntity()       {}

// Class represents a C++ struct or class.
type Class struct {
	Parent       Scope
	Name         string
	TemplateArgs []Type
}

func (*Class) isScope()                    {}
func (*Class) isEntity()                   {}
func (*Class) isType()                     {}
func (c *Class) Scope() Scope              { return c.Parent }
func (c *Class) GetName() string           { return c.Name }
func (c *Class) TemplateArguments() []Type { return c.TemplateArgs }

// Function is a function declaration.
type Function struct {
	Parent       Scope
	Name         string
	Return       Type
	Parameters   []Type
	TemplateArgs []Type
	Const        bool
	Static       bool
}

func (*Function) isScope()                    {}
func (*Function) isEntity()                   {}
func (*Function) isType()                     {}
func (f *Function) Scope() Scope              { return f.Parent }
func (f *Function) GetName() string           { return f.Name }
func (f *Function) TemplateArguments() []Type { return f.TemplateArgs }
