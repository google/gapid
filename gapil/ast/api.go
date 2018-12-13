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

// Package ast holds the set of types used in the abstract syntax tree
// representation of the api language.
package ast

// API is the root of the AST tree, and constitutes one entire parsed file.
// It holds the set of top level AST nodes, grouped by type.
type API struct {
	Imports     []*Import     // api files imported with the "import" keyword
	Externs     []*Function   // functions declared with the "extern" keyword
	Commands    []*Function   // functions declared with the "cmd" keyword
	Subroutines []*Function   // functions declared with the "sub" keyword
	Pseudonyms  []*Pseudonym  // strong type aliases declared with the "type" keyword
	Enums       []*Enum       // enumerated types, declared with the "enum" keyword
	Classes     []*Class      // class types, declared with the "class" keyword
	Fields      []*Field      // variables declared at the global scope
	Definitions []*Definition // definitions declared with the "define" keyword
	Index       *Number       // the API index
}

func (API) isNode() {}

// Annotation is the AST node that represents «@name(arguments) constructs»
type Annotation struct {
	Name      *Identifier // the name part (between the @ and the brackets)
	Arguments []Node      // the list of arguments (the bit in brackets)
}

func (Annotation) isNode() {}

// Annotations represents the set of Annotation objects that apply to another
// AST node.
type Annotations []*Annotation

// GetAnnotation finds annotation with given name, or nil if not found.
func (a Annotations) GetAnnotation(name string) *Annotation {
	for _, entry := range a {
		if entry.Name != nil && entry.Name.Value == name {
			return entry
		}
	}
	return nil
}

// Import is the AST node that represents «import name "path"» constructs
type Import struct {
	Annotations Annotations // the annotations applied to the import
	Path        *String     // the relative path to the api file
}

func (Import) isNode() {}

// Abort is the AST node that represents «abort» statement
type Abort struct {
	ignore bool // filed added so the instances get a unique address
}

func (Abort) isNode() {}

// Fence is the AST node that represents «fence» statement
type Fence struct {
	ignore bool // filed added so the instances get a unique address
}

func (Fence) isNode() {}

// Delete is the AST node that represents «delete(map, key)» statement
type Delete struct {
	Map Node
	Key Node
}

func (Delete) isNode() {}

// Clear is the AST node that represents «clear(map)» statement
type Clear struct {
	Map Node
}

func (Clear) isNode() {}
