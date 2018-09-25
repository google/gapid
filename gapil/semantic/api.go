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

// Package semantic holds the set of types used in the abstract semantic graph
// representation of the api language.
package semantic

import "github.com/google/gapid/gapil/ast"

// API is the root of the ASG, and holds a fully resolved api.
type API struct {
	members
	Named
	Enums        []*Enum        // the set of enums
	Definitions  []*Definition  // the set of definitions
	Classes      []*Class       // the set of classes
	Pseudonyms   []*Pseudonym   // the set of pseudo types
	Externs      []*Function    // the external function references
	Subroutines  []*Function    // the global subroutines
	Functions    []*Function    // the global functions
	Methods      []*Function    // the method functions
	Globals      []*Global      // the global variables
	StaticArrays []*StaticArray // the fixed size array types used
	Maps         []*Map         // the map types used
	Pointers     []*Pointer     // the pointer types used
	Slices       []*Slice       // the pointer types used
	References   []*Reference   // the reference types used
	Signatures   []*Signature   // the function signature types used
	Index        Uint8Value     // the index of this api
}

func (*API) isNode() {}

// Documentation represents the documentation strings for a type or function.
type Documentation []string

func (Documentation) isNode() {}

// Import wraps an API with it's imported name.
type Import struct {
	owned
	noMembers
	Named      // the full type name
	API   *API // the API being imported
}

// Member implements the Owner interface delegating member lookup to the imported API
func (i Import) Member(name string) Owned {
	return i.API.Member(name)
}

// Annotation represents a single annotation on an Annotated.
type Annotation struct {
	AST       *ast.Annotation // the underlying syntax node this was built from
	Named                     // the name of the annotation
	Arguments []Expression    // the arguments to the annotation
}

func (*Annotation) isNode() {}

// Annotated is the common interface to objects that can carry annotations.
type Annotated interface {
	// GetAnnotation returns the annotation with the matching name, if present.
	GetAnnotation(name string) *Annotation
}

// Annotations is an array of Annotation objects that implements the Annotated
// interface. It is used as an anonymous field on objects that carry
// annotations.
type Annotations []*Annotation

// IsInternal returns true if the object is annotated with @internal.
// It is illegal to assign a non-external pointer or slice to an internal
// pointer or slice.
func (a *Annotations) IsInternal() bool { return a.GetAnnotation("internal") != nil }

// GetAnnotation implements the Annotated interface for the Annotations type.
func (a Annotations) GetAnnotation(name string) *Annotation {
	for _, entry := range a {
		if entry.Name() == name {
			return entry
		}
	}
	return nil
}

// Global represents a global variable.
type Global struct {
	owned
	AST         *ast.Field // the underlying syntax node this was built from
	Annotations            // the annotations applied to this global
	Type        Type       // the type the global stores
	Named                  // the name of the global
	Default     Expression // the initial value of the global
}

func (*Global) isNode()       {}
func (*Global) isExpression() {}

// ExpressionType returns the type stored in the global.
func (g *Global) ExpressionType() Type { return g.Type }

// CommandIndex returns the index of the given command, or -1 if the function
// is not a command of the API.
func (a *API) CommandIndex(cmd *Function) int {
	for i, f := range a.Functions {
		if f == cmd {
			return i
		}
	}
	return -1
}

// ClassIndex returns the index of the given class, or -1 if the class does not
// belong to the API.
func (a *API) ClassIndex(class *Class) int {
	for i, c := range a.Classes {
		if c == class {
			return i
		}
	}
	return -1
}

// EnumIndex returns the index of the given enum, or -1 if the enum does not
// belong to the API.
func (a *API) EnumIndex(enum *Enum) int {
	for i, e := range a.Enums {
		if e == enum {
			return i
		}
	}
	return -1
}

// MapIndex returns the index of the given map, or -1 if the map does not belong
// to the API.
func (a *API) MapIndex(t *Map) int {
	for i, m := range a.Maps {
		if m == t {
			return i
		}
	}
	return -1
}

// SliceIndex returns the index of the given slice, or -1 if the slice does not
// belong to the API.
func (a *API) SliceIndex(slice *Slice) int {
	for i, s := range a.Slices {
		if s == slice {
			return i
		}
	}
	return -1
}
