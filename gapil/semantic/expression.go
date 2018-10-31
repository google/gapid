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

var (
	// BuiltinThreadGlobal represents the $Thread global variable.
	BuiltinThreadGlobal = &Global{
		Type:    Uint64Type,
		Named:   "$Thread",
		Default: Uint64Value(0),
	}

	// BuiltinGlobals is the list of all builtin globals.
	BuiltinGlobals = []*Global{
		BuiltinThreadGlobal,
	}
)

// Select is the expression form of a switch.
type Select struct {
	AST     *ast.Switch // the underlying syntax node this was built from
	Type    Type        // the return type of the select if valid
	Value   Expression  // the value to match the cases against
	Choices []*Choice   // The set of possible choices to match
	Default Expression  // the expression to use if no condition matches
}

func (*Select) isNode()       {}
func (*Select) isExpression() {}

// ExpressionType implements Expression with the unified type of the choices
func (s *Select) ExpressionType() Type { return s.Type }

// Choice represents a possible choice in a select
type Choice struct {
	AST         *ast.Case    // the underlying syntax node this was built from
	Annotations              // the annotations applied to the choice
	Conditions  []Expression // the set of expressions to match the select value against
	Expression  Expression   // the expression to use if a condition matches
}

func (*Choice) isNode() {}

// Local represents an immutable local storage slot, created by a DeclareLocal,
// and referred to by identifiers that resolve to that slot.
type Local struct {
	Declaration *DeclareLocal // the statement that created the local
	Type        Type          // the type of the storage
	Named                     // the identifier that will resolve to this local
	Value       Expression    // the expression the local was assigned on creation
}

func (*Local) isNode()       {}
func (*Local) isExpression() {}

// ExpressionType implements Expression
func (l *Local) ExpressionType() Type { return l.Type }

// UnaryOp represents an operator applied to a single expression.
// It's type depends on the operator and the type of the expression it is
// being applied to.
type UnaryOp struct {
	AST        *ast.UnaryOp // the underlying syntax node this was built from
	Type       Type         // the resolved type of the operation
	Operator   string       // the operator being applied
	Expression Expression   // the expression to apply the operator to
}

func (*UnaryOp) isNode()       {}
func (*UnaryOp) isExpression() {}

// ExpressionType implements Expression
func (b *UnaryOp) ExpressionType() Type { return b.Type }

// BinaryOp represents any operator applied to two arguments.
// The resolved type of the expression depends on the types of the two
// arguments and which operator it represents.
type BinaryOp struct {
	AST      *ast.BinaryOp // the underlying syntax node this was built from
	Type     Type          // the resolved type of this binary expression
	LHS      Expression    // the expression that appears on the left of the operator
	Operator string        // the operator being applied
	RHS      Expression    // the expression that appears on the right of the operator
}

func (*BinaryOp) isNode()       {}
func (*BinaryOp) isExpression() {}

// ExpressionType implements Expression
func (b *BinaryOp) ExpressionType() Type { return b.Type }

// BitTest is the "in" operator applied to a bitfield.
type BitTest struct {
	AST      *ast.BinaryOp // the underlying syntax node this was built from
	Bitfield Expression    // the bitfield being tested
	Bits     Expression    // the bits to test for
}

func (*BitTest) isNode()       {}
func (*BitTest) isExpression() {}

// ExpressionType implements Expression
func (*BitTest) ExpressionType() Type { return BoolType }

// MapContains is the "in" operator applied to a map.
type MapContains struct {
	AST  *ast.BinaryOp // the underlying syntax node this was built from
	Type *Map          // the value type of the map being indexed
	Map  Expression    // the map being tested
	Key  Expression    // the key to test for
}

func (*MapContains) isNode()       {}
func (*MapContains) isExpression() {}

// ExpressionType implements Expression
func (*MapContains) ExpressionType() Type { return BoolType }

// SliceContains is the "in" operator applied to a slice.
type SliceContains struct {
	AST   *ast.BinaryOp // the underlying syntax node this was built from
	Type  *Slice        // the value type of the map being sliced
	Slice Expression    // the slice being tested
	Value Expression    // the value to test for
}

func (*SliceContains) isNode()       {}
func (*SliceContains) isExpression() {}

// ExpressionType implements Expression
func (*SliceContains) ExpressionType() Type { return BoolType }

// Member is an expression that looks up a field by name from an object.
type Member struct {
	AST    *ast.Member // the underlying syntax node this was built from
	Object Expression  // the object to look up a field of
	Field  *Field      // the field to look up
}

func (*Member) isNode()       {}
func (*Member) isExpression() {}

// ExpressionType implements Expression returning the type of the field.
func (m *Member) ExpressionType() Type { return m.Field.Type }

// GetAnnotation returns the annotation with the matching name, if present.
func (m *Member) GetAnnotation(name string) *Annotation {
	return m.Field.GetAnnotation(name)
}

// MessageValue is an expression that produces a localized message
type MessageValue struct {
	AST       *ast.Class          // the underlying syntax node this was built from
	Arguments []*FieldInitializer // the list of message arguments
}

func (*MessageValue) isNode()       {}
func (*MessageValue) isExpression() {}

// ExpressionType implements Expression with a type of MessageType
func (*MessageValue) ExpressionType() Type { return MessageType }

// PointerRange represents using the indexing operator on a pointer type with a
// range expression.
type PointerRange struct {
	AST     *ast.Index // the underlying syntax node this was built from
	Type    *Slice     // the slice type returned.
	Pointer Expression // the expression that returns the pointer to be indexed
	Range   *BinaryOp  // the range to use on the slice
}

func (*PointerRange) isNode()       {}
func (*PointerRange) isExpression() {}

// ExpressionType implements Expression.
// It returns the same slice type being sliced.
func (i *PointerRange) ExpressionType() Type { return i.Type }

// SliceRange represents using the indexing operator on a slice type with a
// range expression.
type SliceRange struct {
	AST   *ast.Index // the underlying syntax node this was built from
	Type  *Slice     // the slice type
	Slice Expression // the expression that returns the slice to be indexed
	Range *BinaryOp  // the range to use on the slice
}

func (*SliceRange) isNode()       {}
func (*SliceRange) isExpression() {}

// ExpressionType implements Expression.
// It returns the same slice type being sliced.
func (i *SliceRange) ExpressionType() Type { return i.Type }

// ArrayIndex represents using the indexing operator on a static-array type.
type ArrayIndex struct {
	AST   *ast.Index   // the underlying syntax node this was built from
	Type  *StaticArray // the array type
	Array Expression   // the expression that returns the array to be indexed
	Index Expression   // the index to use on the array
}

func (*ArrayIndex) isNode()       {}
func (*ArrayIndex) isExpression() {}

// ExpressionType implements Expression.
// It returns the element type of the array.
func (i *ArrayIndex) ExpressionType() Type { return i.Type.ValueType }

// SliceIndex represents using the indexing operator on a slice type.
type SliceIndex struct {
	AST   *ast.Index // the underlying syntax node this was built from
	Type  *Slice     // the slice type
	Slice Expression // the expression that returns the slice to be indexed
	Index Expression // the index to use on the slice
}

func (*SliceIndex) isNode()       {}
func (*SliceIndex) isExpression() {}

// ExpressionType implements Expression.
// It returns the value type of the slice.
func (i *SliceIndex) ExpressionType() Type { return i.Type.To }

// MapIndex represents using the indexing operator on a map type.
type MapIndex struct {
	AST   *ast.Index // the underlying syntax node this was built from
	Type  *Map       // the value type of the map being indexed
	Map   Expression // the expression that returns the map to be indexed
	Index Expression // the index to use on the map
}

func (*MapIndex) isNode()       {}
func (*MapIndex) isExpression() {}

// ExpressionType implements Expression returning the value type of the map.
func (i *MapIndex) ExpressionType() Type { return i.Type.ValueType }

// Unknown represents a value that cannot be predicted.
// These values are non-deterministic with regard to the API specification and
// may vary between implementations of the API.
type Unknown struct {
	AST      *ast.Unknown // the underlying syntax node this was built from
	Inferred Expression   // the inferred expression to derive the unknown from the outputs
}

func (*Unknown) isNode()       {}
func (*Unknown) isExpression() {}

// ExpressionType implements Expression with the inferred type of the unknown.
// If the unknown could not be inferred, it will be of type "any" so allow
// expressions using it to resolve anyway.
func (u *Unknown) ExpressionType() Type {
	if u.Inferred == nil {
		return AnyType
	}
	return u.Inferred.ExpressionType()
}

// Ignore represents an _ expression.
type Ignore struct {
	AST ast.Node // the underlying syntax node this was built from
}

func (*Ignore) isNode()       {}
func (*Ignore) isExpression() {}

// ExpressionType implements Expression.
func (i *Ignore) ExpressionType() Type {
	return AnyType
}

// Null represents a default value.
type Null struct {
	AST  *ast.Null // the underlying syntax node this was built from
	Type Type      // the resolved type of this null
}

func (Null) isNode()       {}
func (Null) isExpression() {}

// ExpressionType implements Expression with the inferred type of the null.
func (n Null) ExpressionType() Type {
	return n.Type
}

// Int8Value is an int8 that implements Expression so it can be in the semantic graph
type Int8Value int8

func (Int8Value) isNode()       {}
func (Int8Value) isExpression() {}

// ExpressionType implements Expression with a type of Int8Type
func (v Int8Value) ExpressionType() Type { return Int8Type }

// Uint8Value is a uint8 that implements Expression so it can be in the semantic graph
type Uint8Value uint8

func (Uint8Value) isNode()       {}
func (Uint8Value) isExpression() {}

// ExpressionType implements Expression with a type of Uint8Type
func (v Uint8Value) ExpressionType() Type { return Uint8Type }

// Int16Value is an int16 that implements Expression so it can be in the semantic graph
type Int16Value int16

func (Int16Value) isNode()       {}
func (Int16Value) isExpression() {}

// ExpressionType implements Expression with a type of Int16Type
func (v Int16Value) ExpressionType() Type { return Int16Type }

// Uint16Value is a uint16 that implements Expression so it can be in the semantic graph
type Uint16Value uint16

func (Uint16Value) isNode()       {}
func (Uint16Value) isExpression() {}

// ExpressionType implements Expression with a type of Uint16Type
func (v Uint16Value) ExpressionType() Type { return Uint16Type }

// Int32Value is an int32 that implements Expression so it can be in the semantic graph
type Int32Value int32

func (Int32Value) isNode()       {}
func (Int32Value) isExpression() {}

// ExpressionType implements Expression with a type of Int32Type
func (v Int32Value) ExpressionType() Type { return Int32Type }

// Uint32Value is a uint32 that implements Expression so it can be in the semantic graph
type Uint32Value uint32

func (Uint32Value) isNode()       {}
func (Uint32Value) isExpression() {}

// ExpressionType implements Expression with a type of Uint32Type
func (v Uint32Value) ExpressionType() Type { return Uint32Type }

// Int64Value is an int64 that implements Expression so it can be in the semantic graph
type Int64Value int64

func (Int64Value) isNode()       {}
func (Int64Value) isExpression() {}

// ExpressionType implements Expression with a type of Int64Type
func (v Int64Value) ExpressionType() Type { return Int64Type }

// Uint64Value is a uint64 that implements Expression so it can be in the semantic graph
type Uint64Value uint64

func (Uint64Value) isNode()       {}
func (Uint64Value) isExpression() {}

// ExpressionType implements Expression with a type of Uint64Type
func (v Uint64Value) ExpressionType() Type { return Uint64Type }

// BoolValue is a bool that implements Expression so it can be in the semantic graph
type BoolValue bool

func (BoolValue) isNode()       {}
func (BoolValue) isExpression() {}

// ExpressionType implements Expression with a type of BoolType
func (v BoolValue) ExpressionType() Type { return BoolType }

// StringValue is a string that implements Expression so it can be in the semantic graph
type StringValue string

func (StringValue) isNode()       {}
func (StringValue) isExpression() {}

// ExpressionType implements Expression with a type of StringType
func (v StringValue) ExpressionType() Type { return StringType }

// Float64Value is a float64 that implements Expression so it can be in the semantic graph
type Float64Value float64

func (Float64Value) isNode()       {}
func (Float64Value) isExpression() {}

// ExpressionType implements Expression with a type of Float64Type
func (v Float64Value) ExpressionType() Type { return Float64Type }

// Float32Value is a float32 that implements Expression so it can be in the semantic graph
type Float32Value float32

func (Float32Value) isNode()       {}
func (Float32Value) isExpression() {}

// ExpressionType implements Expression with a type of Float32Type
func (v Float32Value) ExpressionType() Type { return Float32Type }
