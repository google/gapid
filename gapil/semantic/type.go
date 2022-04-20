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

import (
	"fmt"

	"github.com/google/gapid/gapil/ast"
)

// Type is the interface to any object that can act as a type to the api
// language.
type Type interface {
	isType() // A placeholder function that's implemented by all semantic types.
	Owner
}

// Expression represents anything that can act as an expression in the api
// language, it must be able to correctly report the type of value it would
// return if executed.
type Expression interface {
	isExpression() // A placeholder function that's implemented by all semantic types.
	Node
	ExpressionType() Type // returns the expression value type.
}

// Class represents an api class construct.
type Class struct {
	owned
	members
	AST         *ast.Class    // the underlying syntax node this was built from
	Annotations               // the annotations applied to this class
	Named                     // implement Child
	Docs        Documentation // the documentation for the class
	Fields      []*Field      // the set of fields the class declares
	Methods     []*Function   // the set of functions associated with the class
}

func (c Class) String() string { return c.Name() }

func (*Class) isNode() {}
func (*Class) isType() {}

func (t *Class) ASTNode() ast.Node { return t.AST }

// FieldByName returns the field of t with the name n, or nil.
func (t *Class) FieldByName(n string) *Field {
	for _, f := range t.Fields {
		if f.Name() == n {
			return f
		}
	}
	return nil
}

// Field represents a field entry in a class.
type Field struct {
	owned
	AST         *ast.Field    // the underlying syntax node this was built from
	Annotations               // the annotations applied to this field
	Type        Type          // the type the field stores
	Named                     // the name of the field
	Docs        Documentation // the documentation for the field
	Default     Expression    // the default value of the field
}

func (f Field) String() string { return f.Name() }

func (*Field) isNode()       {}
func (*Field) isExpression() {}

// ExpressionType implements Expression to return the type stored in the field.
func (f *Field) ExpressionType() Type { return f.Type }

// ClassInitializer represents an expression that can assign values to multiple
// fields of a class.
type ClassInitializer struct {
	AST    *ast.Call           // the underlying syntax node this was built from
	Class  *Class              // the class to initialize
	Fields []*FieldInitializer // the set of field assignments
}

func (*ClassInitializer) isNode()       {}
func (*ClassInitializer) isExpression() {}

// IsCopyConstructor returns whether this ClassInitializer represents the
// calling of the copy constructor.
func (c *ClassInitializer) IsCopyConstructor() bool {
	return len(c.Fields) == 1 && c.Fields[0].Field == nil
}

// InitialValues returns the full set of initial values for each field in
// the class. If there is not an initialized or default value for a field, then
// nil is returned for that field.
func (c *ClassInitializer) InitialValues() []Expression {
	if c.IsCopyConstructor() {
		// Create expressions for each field using the copied value.
		out := make([]Expression, len(c.Class.Fields))
		for i, f := range c.Class.Fields {
			out[i] = &Member{
				Object: c.Fields[0].Value,
				Field:  f,
			}
		}
		return out
	}

	m := make(map[*Field]*FieldInitializer, len(c.Class.Fields))
	for _, f := range c.Fields {
		m[f.Field] = f
	}
	out := make([]Expression, len(c.Class.Fields))
	for i, f := range c.Class.Fields {
		if v, ok := m[f]; ok {
			out[i] = v.Value
		} else {
			out[i] = f.Default
		}
	}
	return out
}

// ExpressionType implements Expression returning the class type being initialized.
func (c *ClassInitializer) ExpressionType() Type {
	if c.Class != nil {
		return c.Class
	}
	return nil
}

// FieldInitializer represents the value initialization of a class field.
type FieldInitializer struct {
	AST   ast.Node   // the underlying syntax node this was built from
	Field *Field     // the field to assign to
	Value Expression // the value to assign
}

func (*FieldInitializer) isNode() {}

// Definition represents a named literal definition.
type Definition struct {
	noMembers
	Named
	AST         *ast.Definition // the underlying syntax node this was built from
	Annotations                 // the annotations applied to this definition
	Docs        Documentation   // the documentation for this definition
	Expression  Expression      // the value of this definition, type-inferred without context
}

func (*Definition) isNode()       {}
func (*Definition) isExpression() {}

func (d *Definition) ExpressionType() Type { return d.Expression.ExpressionType() }

// DefinitionUsage represents a named literal usage.
type DefinitionUsage struct {
	noMembers
	Definition *Definition // the definition of this definition usage
	Expression Expression  // the value of this definition, type-inferred by its usage context
}

func (*DefinitionUsage) isNode()       {}
func (*DefinitionUsage) isExpression() {}

func (d *DefinitionUsage) ExpressionType() Type { return d.Expression.ExpressionType() }

// Enum represents the api enum construct.
type Enum struct {
	owned
	members
	AST         *ast.Enum     // the underlying syntax node this was built from
	Annotations               // the annotations applied to this enum
	Named                     // the type name of the enum
	Docs        Documentation // the documentation for the enum
	IsBitfield  bool          // whether this enum is actually a bitfield
	NumberType  Type          // the numerical type of each entry
	Entries     []*EnumEntry  // the entries of this enum
}

func (e Enum) String() string { return e.Name() }

func (*Enum) isNode() {}
func (*Enum) isType() {}

func (t *Enum) ASTNode() ast.Node { return t.AST }

// EnumEntry represents a single entry in an Enum.
type EnumEntry struct {
	owned
	AST   *ast.EnumEntry // the underlying syntax node this was built from
	Named                // the name of this entry
	Docs  Documentation  // the documentation for the enum entry
	Value Expression     // the value this entry represents
}

func (e EnumEntry) String() string { return e.Name() }

func (*EnumEntry) isNode()       {}
func (*EnumEntry) isExpression() {}

// ExpressionType implements Expression returning the enum type.
func (e *EnumEntry) ExpressionType() Type {
	t, _ := e.Owner().(Type)
	return t
}

// Pseudonym represents the type construct.
// It acts as a type in it's own right that can carry methods, but is defined
// in terms of another type.
type Pseudonym struct {
	owned
	members     Symbols
	AST         *ast.Pseudonym // the underlying syntax node this was built from
	Annotations                // the annotations applied to this pseudonym
	Named                      // the type name
	Docs        Documentation  // the documentation for the pseudonym
	To          Type           // the underlying type
	Methods     []*Function    // the methods added directly to the pseudonym
}

func (p Pseudonym) String() string { return p.Name() }

func (*Pseudonym) isNode() {}
func (*Pseudonym) isType() {}

func (t *Pseudonym) ASTNode() ast.Node { return t.AST }

// Member implements Type returning the direct member if it has it, otherwise
// delegating the lookup to the underlying type.
func (t *Pseudonym) Member(name string) Owned {
	n, err := t.members.Find(name)
	if err != nil {
		// TODO: propagate errors from this function
		return nil
	}
	if n != nil {
		return n.(Owned)
	}
	return t.To.Member(name)
}

func (t *Pseudonym) addMember(child Owned) {
	t.members.AddNamed(child)
}

func (t *Pseudonym) VisitMembers(visitor func(Owned)) {
	t.members.sort()
	for _, e := range t.members.entries {
		visitor(e.node.(Owned))
	}
	t.To.VisitMembers(visitor)
}

func (t *Pseudonym) SortMembers() { t.members.sort() }

// StaticArray represents a one-dimension fixed size array type, of the form T[8]
type StaticArray struct {
	owned
	noMembers
	Named                // the full type name
	ValueType Type       // the storage type of the elements
	Size      uint32     // the array size
	SizeExpr  Expression // the expression representing the array size
}

func (a StaticArray) String() string { return fmt.Sprintf("%v[%v]", a.ValueType, a.Size) }

func (*StaticArray) isNode() {}
func (*StaticArray) isType() {}

// ArrayInitializer represents an expression that creates a new StaticArray
// instance using a value list, of the form T(v0, v1, v2)
type ArrayInitializer struct {
	AST    *ast.Call    // the underlying syntax node this was built from
	Array  Type         // the array type to initialize (may be aliased)
	Values []Expression // the list of element values
}

func (*ArrayInitializer) isNode()       {}
func (*ArrayInitializer) isExpression() {}

// ExpressionType implements Expression returning the class type being initialized.
func (c *ArrayInitializer) ExpressionType() Type {
	return c.Array
}

// Map represents an api map type declaration, of the form
// map!(KeyType, ValueType)
type Map struct {
	owned
	members
	Named          // the full type name
	KeyType   Type // the type used as an indexing key
	ValueType Type // the type stored in the map
	Dense     bool // Is this a dense map
}

func (m Map) String() string { return fmt.Sprintf("Map!(%v, %v)", m.KeyType, m.ValueType) }

func (*Map) isNode() {}
func (*Map) isType() {}

// Pointer represents an api pointer type declaration, of the form To*
type Pointer struct {
	owned
	noAddMembers
	Named        // the full type name
	To    Type   // the type this is a pointer to
	Const bool   // whether the pointer was declared with the const attribute
	Slice *Slice // The complementary slice type for this pointer.
}

func (p Pointer) String() string { return fmt.Sprintf("%v*", p.To) }

func (*Pointer) isNode() {}
func (*Pointer) isType() {}

func (t *Pointer) Member(name string) Owned {
	return t.To.Member(name)
}

func (t *Pointer) VisitMembers(visitor func(Owned)) {
	t.To.VisitMembers(visitor)
}

func (t *Pointer) SortMembers() { t.To.SortMembers() }

// Slice represents an api slice type declaration, of the form To[]
type Slice struct {
	owned
	noMembers
	Named            // the full type name
	To      Type     // The type this is a slice of
	Pointer *Pointer // The complementary pointer type for this slice.
}

func (s Slice) String() string { return fmt.Sprintf("%v[]", s.To) }

func (*Slice) isNode() {}
func (*Slice) isType() {}

// Reference represents an api reference type declaration, of the form
// ref!To
type Reference struct {
	owned
	noAddMembers
	Named      // the full type name
	To    Type // the type this is a reference to
}

func (r Reference) String() string { return fmt.Sprintf("ref!%v", r.To) }

func (*Reference) isNode() {}
func (*Reference) isType() {}

func (t *Reference) Member(name string) Owned {
	return t.To.Member(name)
}

func (t *Reference) VisitMembers(visitor func(Owned)) {
	t.To.VisitMembers(visitor)
}

func (t *Reference) SortMembers() { t.To.SortMembers() }

// Builtin represents one of the primitive types.
type Builtin struct {
	owned
	noMembers
	Named // the primitive type name
}

func (b Builtin) String() string { return b.Name() }

func (*Builtin) isNode() {}
func (*Builtin) isType() {}

func builtin(name string) *Builtin {
	b := &Builtin{Named: Named(name)}
	BuiltinTypes = append(BuiltinTypes, b)
	return b
}

var (
	// These are all the fundamental primitive types of the api language

	// Special types
	VoidType    = builtin("void")
	AnyType     = builtin("any")
	StringType  = builtin("string")
	MessageType = builtin("message")
	// Unsized primitives
	BoolType = builtin("bool")
	IntType  = builtin("int")
	UintType = builtin("uint")
	// Size is used to represent the size_t type in C/C++. It is transmitted
	// between various components as uint64. There's an up-conversion
	// on recording, and a down-conversion on replay.
	SizeType = builtin("size")
	// Char is supposed to be unsized type, but is treated by the APIC templates
	// as a synonym for u8 and the UI assumes ASCII in the memory view.
	CharType = builtin("char")
	// Fixed size integer forms
	Int8Type   = builtin("s8")
	Uint8Type  = builtin("u8")
	Int16Type  = builtin("s16")
	Uint16Type = builtin("u16")
	Int32Type  = builtin("s32")
	Uint32Type = builtin("u32")
	Int64Type  = builtin("s64")
	Uint64Type = builtin("u64")
	// Floating point forms
	Float32Type = builtin("f32")
	Float64Type = builtin("f64")
)

var BuiltinTypes []*Builtin

// IsReference returns true if ty is a reference type.
func IsReference(ty Type) bool {
	_, ok := ty.(*Reference)
	return ok
}

// IsNumeric returns true if t is one of the primitive numeric types.
func IsNumeric(ty Type) bool {
	if IsInteger(ty) {
		return true
	}
	switch ty {
	case Float32Type, Float64Type:
		return true
	}
	return false
}

// IsInteger returns true if ty is an integer type
func IsInteger(ty Type) bool {
	switch ty {
	case IntType, UintType,
		Int8Type, Uint8Type,
		Int16Type, Uint16Type,
		Int32Type, Uint32Type,
		Int64Type, Uint64Type,
		SizeType,
		CharType:
		return true
	}
	return false
}

// IsUnsigned returns true if ty is an unsigned integer type
func IsUnsigned(ty Type) bool {
	switch ty {
	case UintType, Uint8Type, Uint16Type, Uint32Type, Uint64Type, SizeType:
		return true
	}
	return false
}

// IntegerSizeInBits returns the size in bits of the given integer type.
// If ty is not an integer, then IntegerSizeInBits returns 0.
func IntegerSizeInBits(ty Type) int {
	switch ty {
	case Int8Type, Uint8Type:
		return 8
	case Int16Type, Uint16Type:
		return 16
	case Int32Type, Uint32Type:
		return 32
	case Int64Type, Uint64Type:
		return 64
	}
	return 0
}

// Integer returns a signed integer type of the given size in bytes.
// size must be 1, 2, 4 or 8.
func Integer(size int32) Type {
	switch size {
	case 1:
		return Int8Type
	case 2:
		return Int16Type
	case 4:
		return Int32Type
	case 8:
		return Int64Type
	default:
		panic(fmt.Errorf("Unexpected target integer size %v", size))
	}
}

// UnsignedInteger returns an unsigned integer type of the given size in bytes.
// size must be 1, 2, 4 or 8.
func UnsignedInteger(size int32) Type {
	switch size {
	case 1:
		return Uint8Type
	case 2:
		return Uint16Type
	case 4:
		return Uint32Type
	case 8:
		return Uint64Type
	default:
		panic(fmt.Errorf("Unexpected target integer size %v", size))
	}
}

// AsUint64 converts a sized integer value to a uint64.
func AsUint64(val Expression) (uint64, bool) {
	switch v := val.(type) {
	case Int8Value:
		return uint64(v), true
	case Uint8Value:
		return uint64(v), true
	case Int16Value:
		return uint64(v), true
	case Uint16Value:
		return uint64(v), true
	case Int32Value:
		return uint64(v), true
	case Uint32Value:
		return uint64(v), true
	case Int64Value:
		return uint64(v), true
	case Uint64Value:
		return uint64(v), true
	default:
		return uint64(0), false
	}
}

// Underlying returns the underlying type for ty by recursively traversing the
// pseudonym chain until reaching and returning the first non-pseudoym type.
// If ty is not a pseudonym then it is simply returned.
func Underlying(ty Type) Type {
	for {
		if pseudo, ok := ty.(*Pseudonym); ok {
			ty = pseudo.To
		} else {
			return ty
		}
	}
}

// TypeOf returns the resolved semantic type of the Type, Field or Expression.
func TypeOf(v Node) (Type, error) {
	if v == nil {
		return VoidType, nil
	}
	switch e := v.(type) {
	case Type:
		return e, nil
	case *Field:
		return e.Type, nil
	case Expression:
		return e.ExpressionType(), nil
	default:
		return nil, fmt.Errorf("Type \"%T\" is not an expression", v)
	}
}

// IsStorageType returns true if ty can be used as a storage type.
func IsStorageType(ty Type) bool {
	switch ty := ty.(type) {
	case *Builtin:
		switch ty {
		case StringType,
			AnyType,
			MessageType:
			return false
		default:
			return true
		}
	case *Pseudonym:
		return IsStorageType(ty.To)
	case *Pointer:
		return IsStorageType(ty.To)
	case *Class:
		for _, f := range ty.Fields {
			if !IsStorageType(f.Type) {
				return false
			}
		}
		return true
	case *Enum:
		return true
	case *StaticArray:
		return IsStorageType(ty.ValueType)
	default:
		return false
	}
}
