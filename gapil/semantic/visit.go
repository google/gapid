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

import "fmt"

// Visit invokes visitor for all the children of the supplied node.
func Visit(node Node, visitor func(Node)) {
	Replace(node, func(n Node) Node { visitor(n); return n })
}

// Replace invokes visitor for all the children of the supplied node, replacing
// the node with the returned value.
func Replace(node Node, visitor func(Node) Node) {
	switch n := node.(type) {
	case *Abort:
	case *API:
		for i, c := range n.Enums {
			n.Enums[i] = visitor(c).(*Enum)
		}
		for i, c := range n.Definitions {
			n.Definitions[i] = visitor(c).(*Definition)
		}
		for i, c := range n.Classes {
			n.Classes[i] = visitor(c).(*Class)
		}
		for i, c := range n.Pseudonyms {
			n.Pseudonyms[i] = visitor(c).(*Pseudonym)
		}
		for i, c := range n.Externs {
			n.Externs[i] = visitor(c).(*Function)
		}
		for i, c := range n.Subroutines {
			n.Subroutines[i] = visitor(c).(*Function)
		}
		for i, c := range n.Functions {
			n.Functions[i] = visitor(c).(*Function)
		}
		for i, c := range n.Methods {
			n.Methods[i] = visitor(c).(*Function)
		}
		for i, c := range n.Globals {
			n.Globals[i] = visitor(c).(*Global)
		}
		for i, c := range n.StaticArrays {
			n.StaticArrays[i] = visitor(c).(*StaticArray)
		}
		for i, c := range n.Maps {
			n.Maps[i] = visitor(c).(*Map)
		}
		for i, c := range n.Pointers {
			n.Pointers[i] = visitor(c).(*Pointer)
		}
		for i, c := range n.Slices {
			n.Slices[i] = visitor(c).(*Slice)
		}
		for i, c := range n.References {
			n.References[i] = visitor(c).(*Reference)
		}
		for i, c := range n.Signatures {
			n.Signatures[i] = visitor(c).(*Signature)
		}
	case *ArrayAssign:
		n.To = visitor(n.To).(*ArrayIndex)
		n.Value = visitor(n.Value).(Expression)
	case *ArrayIndex:
		n.Array = visitor(n.Array).(Expression)
		n.Index = visitor(n.Index).(Expression)
	case *ArrayInitializer:
		n.Array = visitor(n.Array).(Type)
		for i, c := range n.Values {
			n.Values[i] = visitor(c).(Expression)
		}
	case *Slice:
		n.To = visitor(n.To).(Type)
	case *SliceIndex:
		n.Slice = visitor(n.Slice).(Expression)
		n.Index = visitor(n.Index).(Expression)
	case *SliceAssign:
		n.To = visitor(n.To).(*SliceIndex)
		n.Value = visitor(n.Value).(Expression)
	case *Assert:
		n.Condition = visitor(n.Condition).(Expression)
	case *Assign:
		n.LHS = visitor(n.LHS).(Expression)
		n.RHS = visitor(n.RHS).(Expression)
	case *Annotation:
		for i, c := range n.Arguments {
			n.Arguments[i] = visitor(c).(Expression)
		}
	case *Block:
		for i, c := range n.Statements {
			n.Statements[i] = visitor(c).(Statement)
		}
	case BoolValue:
	case *BinaryOp:
		if n.LHS != nil {
			n.LHS = visitor(n.LHS).(Expression)
		}
		if n.RHS != nil {
			n.RHS = visitor(n.RHS).(Expression)
		}
	case *BitTest:
		n.Bitfield = visitor(n.Bitfield).(Expression)
		n.Bits = visitor(n.Bits).(Expression)
	case *UnaryOp:
		n.Expression = visitor(n.Expression).(Expression)
	case *Branch:
		n.Condition = visitor(n.Condition).(Expression)
		n.True = visitor(n.True).(*Block)
		if n.False != nil {
			n.False = visitor(n.False).(*Block)
		}
	case *Builtin:
	case *Reference:
		n.To = visitor(n.To).(Type)
	case *Call:
		n.Type = visitor(n.Type).(Type)
		n.Target = visitor(n.Target).(*Callable)
		for i, a := range n.Arguments {
			n.Arguments[i] = visitor(a).(Expression)
		}
	case *Callable:
		if n.Object != nil {
			n.Object = visitor(n.Object).(Expression)
		}
		n.Function = visitor(n.Function).(*Function)
	case *Case:
		for i, c := range n.Annotations {
			n.Annotations[i] = visitor(c).(*Annotation)
		}
		for i, c := range n.Conditions {
			n.Conditions[i] = visitor(c).(Expression)
		}
		n.Block = visitor(n.Block).(*Block)
	case *Cast:
		n.Object = visitor(n.Object).(Expression)
		n.Type = visitor(n.Type).(Type)
	case *Class:
		for i, c := range n.Annotations {
			n.Annotations[i] = visitor(c).(*Annotation)
		}
		for i, f := range n.Fields {
			n.Fields[i] = visitor(f).(*Field)
		}
		for i, m := range n.Methods {
			n.Methods[i] = visitor(m).(*Function)
		}
	case *ClassInitializer:
		for i, f := range n.Fields {
			n.Fields[i] = visitor(f).(*FieldInitializer)
		}
	case *Choice:
		for i, c := range n.Annotations {
			n.Annotations[i] = visitor(c).(*Annotation)
		}
		for i, c := range n.Conditions {
			n.Conditions[i] = visitor(c).(Expression)
		}
		n.Expression = visitor(n.Expression).(Expression)
	case *Definition:
		for i, c := range n.Annotations {
			n.Annotations[i] = visitor(c).(*Annotation)
		}
		n.Expression = visitor(n.Expression).(Expression)
	case *DefinitionUsage:
		n.Expression = visitor(n.Expression).(Expression)
		n.Definition = visitor(n.Definition).(*Definition)
	case *DeclareLocal:
		n.Local = visitor(n.Local).(*Local)
		if n.Local.Value != nil {
			n.Local.Value = visitor(n.Local.Value).(Expression)
		}
	case Documentation:
	case *Enum:
		for i, c := range n.Annotations {
			n.Annotations[i] = visitor(c).(*Annotation)
		}
		for i, e := range n.Entries {
			n.Entries[i] = visitor(e).(*EnumEntry)
		}
	case *EnumEntry:
	case *Fence:
		if n.Statement != nil {
			n.Statement = visitor(n.Statement).(Statement)
		}
	case *Field:
		for i, c := range n.Annotations {
			n.Annotations[i] = visitor(c).(*Annotation)
		}
		n.Type = visitor(n.Type).(Type)
		if n.Default != nil {
			n.Default = visitor(n.Default).(Expression)
		}
	case *FieldInitializer:
		n.Value = visitor(n.Value).(Expression)
	case Float32Value:
	case Float64Value:
	case *Function:
		for i, c := range n.Annotations {
			n.Annotations[i] = visitor(c).(*Annotation)
		}
		if n.Return != nil {
			n.Return = visitor(n.Return).(*Parameter)
		}
		for i, c := range n.FullParameters {
			n.FullParameters[i] = visitor(c).(*Parameter)
		}
		if n.Block != nil {
			n.Block = visitor(n.Block).(*Block)
		}
		n.Signature = visitor(n.Signature).(*Signature)
	case *Global:
		for i, c := range n.Annotations {
			n.Annotations[i] = visitor(c).(*Annotation)
		}
	case *StaticArray:
		n.ValueType = visitor(n.ValueType).(Type)
		n.SizeExpr = visitor(n.SizeExpr).(Expression)
	case *Signature:
	case Int8Value:
	case Int16Value:
	case Int32Value:
	case Int64Value:
	case *Iteration:
		n.Iterator = visitor(n.Iterator).(*Local)
		n.From = visitor(n.From).(Expression)
		n.To = visitor(n.To).(Expression)
		n.Block = visitor(n.Block).(*Block)
	case *MapIteration:
		n.IndexIterator = visitor(n.IndexIterator).(*Local)
		n.KeyIterator = visitor(n.KeyIterator).(*Local)
		n.ValueIterator = visitor(n.ValueIterator).(*Local)
		n.Map = visitor(n.Map).(Expression)
		n.Block = visitor(n.Block).(*Block)
	case Invalid:
	case *Length:
		n.Object = visitor(n.Object).(Expression)
	case *Local:
		n.Type = visitor(n.Type).(Type)
	case *Map:
		n.KeyType = visitor(n.KeyType).(Type)
		n.ValueType = visitor(n.ValueType).(Type)
	case *MapAssign:
		n.To = visitor(n.To).(*MapIndex)
		n.Value = visitor(n.Value).(Expression)
	case *MapContains:
		n.Key = visitor(n.Key).(Expression)
		n.Map = visitor(n.Map).(Expression)
	case *MapIndex:
		n.Map = visitor(n.Map).(Expression)
		n.Index = visitor(n.Index).(Expression)
	case *MapRemove:
		n.Map = visitor(n.Map).(Expression)
		n.Key = visitor(n.Key).(Expression)
	case *MapClear:
		n.Map = visitor(n.Map).(Expression)
	case *Member:
		n.Object = visitor(n.Object).(Expression)
		n.Field = visitor(n.Field).(*Field)
	case *MessageValue:
		for i, a := range n.Arguments {
			n.Arguments[i] = visitor(a).(*FieldInitializer)
		}
	case *New:
		n.Type = visitor(n.Type).(*Reference)
	case *Parameter:
		for i, c := range n.Annotations {
			n.Annotations[i] = visitor(c).(*Annotation)
		}
		n.Type = visitor(n.Type).(Type)
	case *Pointer:
		n.To = visitor(n.To).(Type)
	case *Pseudonym:
		for i, c := range n.Annotations {
			n.Annotations[i] = visitor(c).(*Annotation)
		}
		n.To = visitor(n.To).(Type)
		for i, m := range n.Methods {
			n.Methods[i] = visitor(m).(*Function)
		}
	case *Return:
		if n.Value != nil {
			n.Value = visitor(n.Value).(Expression)
		}
	case *Select:
		n.Value = visitor(n.Value).(Expression)
		for i, c := range n.Choices {
			n.Choices[i] = visitor(c).(*Choice)
		}
		if n.Default != nil {
			n.Default = visitor(n.Default).(Expression)
		}
	case StringValue:
	case *Switch:
		n.Value = visitor(n.Value).(Expression)
		for i, c := range n.Cases {
			n.Cases[i] = visitor(c).(*Case)
		}
		if n.Default != nil {
			n.Default = visitor(n.Default).(*Block)
		}
	case Uint8Value:
	case Uint16Value:
	case Uint32Value:
	case Uint64Value:
	case *Unknown:
	case *Clone:
		n.Slice = visitor(n.Slice).(Expression)
	case *Copy:
		n.Src = visitor(n.Src).(Expression)
		n.Dst = visitor(n.Dst).(Expression)
	case *Create:
		n.Type = visitor(n.Type).(*Reference)
		n.Initializer = visitor(n.Initializer).(*ClassInitializer)
	case *Ignore:
	case *Make:
		n.Type = visitor(n.Type).(*Slice)
		n.Size = visitor(n.Size).(Expression)
	case Null:
	case *Print:
		for i, a := range n.Arguments {
			n.Arguments[i] = visitor(a).(Expression)
		}
	case *PointerRange:
		n.Pointer = visitor(n.Pointer).(Expression)
		n.Range = visitor(n.Range).(*BinaryOp)
	case *Read:
		n.Slice = visitor(n.Slice).(Expression)
	case *SliceContains:
		n.Value = visitor(n.Value).(Expression)
		n.Slice = visitor(n.Slice).(Expression)
	case *SliceRange:
		n.Slice = visitor(n.Slice).(Expression)
		n.Range = visitor(n.Range).(*BinaryOp)
	case *Write:
		n.Slice = visitor(n.Slice).(Expression)
	default:
		panic(fmt.Errorf("Unsupported semantic node type %T", n))
	}
}
