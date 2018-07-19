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

package ast

import "fmt"

// Visit invokes visitor for all the children of the supplied node.
func Visit(node Node, visitor func(Node)) {
	switch n := node.(type) {
	case *Abort:

	case *Annotation:
		visitor(n.Name)
		for _, a := range n.Arguments {
			visitor(a)
		}

	case *API:
		for _, m := range n.Imports {
			visitor(m)
		}
		for _, m := range n.Externs {
			visitor(m)
		}
		for _, m := range n.Subroutines {
			visitor(m)
		}
		for _, m := range n.Commands {
			visitor(m)
		}
		for _, m := range n.Pseudonyms {
			visitor(m)
		}
		for _, m := range n.Enums {
			visitor(m)
		}
		for _, m := range n.Classes {
			visitor(m)
		}
		for _, m := range n.Fields {
			visitor(m)
		}
		for _, m := range n.Definitions {
			visitor(m)
		}

	case *Assign:
		visitor(n.LHS)
		visitor(n.RHS)

	case *BinaryOp:
		visitor(n.LHS)
		visitor(n.RHS)

	case *Block:
		for _, s := range n.Statements {
			visitor(s)
		}

	case *Bool:

	case *Branch:
		visitor(n.Condition)
		visitor(n.True)
		if n.False != nil {
			visitor(n.False)
		}

	case *Call:
		visitor(n.Target)
		for _, a := range n.Arguments {
			visitor(a)
		}

	case *Case:
		for _, c := range n.Conditions {
			visitor(c)
		}
		visitor(n.Block)

	case *Class:
		for _, a := range n.Annotations {
			visitor(a)
		}
		visitor(n.Name)
		for _, f := range n.Fields {
			visitor(f)
		}

	case *DeclareLocal:
		visitor(n.Name)
		visitor(n.RHS)

	case *Default:
		visitor(n.Block)

	case *Definition:
		for _, a := range n.Annotations {
			visitor(a)
		}
		visitor(n.Name)
		visitor(n.Expression)

	case *Delete:
		visitor(n.Map)
		visitor(n.Key)

	case *Enum:
		for _, a := range n.Annotations {
			visitor(a)
		}
		visitor(n.Name)
		for _, e := range n.Entries {
			visitor(e)
		}

	case *EnumEntry:
		visitor(n.Name)
		visitor(n.Value)

	case *Fence:

	case *Field:
		for _, a := range n.Annotations {
			visitor(a)
		}
		visitor(n.Name)
		visitor(n.Type)
		if n.Default != nil {
			visitor(n.Default)
		}

	case *Function:
		for _, a := range n.Annotations {
			visitor(a)
		}
		visitor(n.Generic)
		for _, p := range n.Parameters {
			visitor(p)
		}
		if n.Block != nil {
			visitor(n.Block)
		}

	case *Generic:
		visitor(n.Name)
		for _, a := range n.Arguments {
			visitor(a)
		}

	case *Group:
		visitor(n.Expression)

	case *Identifier:

	case *Import:
		visitor(n.Path)

	case *Imported:
		visitor(n.Name)
		visitor(n.From)

	case *Index:
		visitor(n.Object)
		visitor(n.Index)

	case *IndexedType:
		visitor(n.ValueType)
		if n.Index != nil {
			visitor(n.Index)
		}

	case *Invalid:

	case *Iteration:
		visitor(n.Variable)
		visitor(n.Iterable)
		visitor(n.Block)

	case *MapIteration:
		visitor(n.IndexVariable)
		visitor(n.KeyVariable)
		visitor(n.ValueVariable)
		visitor(n.Map)
		visitor(n.Block)

	case *Member:
		visitor(n.Object)
		visitor(n.Name)

	case *NamedArg:
		visitor(n.Name)
		visitor(n.Value)

	case *Null:

	case *Number:

	case *Parameter:
		for _, a := range n.Annotations {
			visitor(a)
		}
		if n.Name != nil {
			visitor(n.Name)
		}
		visitor(n.Type)

	case *PointerType:
		visitor(n.To)

	case *PreConst:
		visitor(n.Type)

	case *Pseudonym:
		for _, a := range n.Annotations {
			visitor(a)
		}
		visitor(n.Name)
		visitor(n.To)

	case *Return:
		visitor(n.Value)

	case *String:

	case *Switch:
		visitor(n.Value)
		for _, c := range n.Cases {
			visitor(c)
		}
		if n.Default != nil {
			visitor(n.Default)
		}

	case *UnaryOp:
		visitor(n.Expression)

	case *Unknown:

	default:
		panic(fmt.Errorf("Unsupported ast node type %T", n))
	}
}
