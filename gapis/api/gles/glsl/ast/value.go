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

import (
	"strconv"
	"strings"
)

// A langauge value. Each value has a type.
type Value interface {
	Type() Type
}

// FloatValue represents a floating point value, as a float64.
type FloatValue float64

func (v FloatValue) Type() Type { return &BuiltinType{Type: TFloat} }

func (v FloatValue) String() string {
	s := strconv.FormatFloat(float64(v), 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += "."
	}
	return s
}

// IntValue represents an integer value, as a int32.
type IntValue int32

func (v IntValue) Type() Type { return &BuiltinType{Type: TInt} }

func (v IntValue) String() string { return strconv.FormatInt(int64(v), 10) }

// UintValue represents an unsigned integer value, as a uint32.
type UintValue uint32

func (v UintValue) Type() Type { return &BuiltinType{Type: TUint} }

func (v UintValue) String() string { return strconv.FormatUint(uint64(v), 10) + "u" }

// BoolValue represents a boolean value.
type BoolValue bool

func (v BoolValue) Type() Type { return &BuiltinType{Type: TBool} }

// VectorValue represents vector values.
type VectorValue struct {
	Members []Value
	ValType BuiltinType
}

func (v VectorValue) Type() Type { return &v.ValType }

// NewVectorValue constructs a new vector value, given its type and a member callback function.
// The number of elements is determined from the type. The member function should return the
// member value, given its id. It is called in the natural order.
func NewVectorValue(t BareType, member func(i uint8) Value) VectorValue {
	_, row := TypeDimensions(t)
	v := VectorValue{
		Members: make([]Value, row),
		ValType: BuiltinType{Type: t},
	}
	for i := range v.Members {
		v.Members[i] = member(uint8(i))
	}
	return v
}

// MatrixValue represents matrix. Its elements are stored column-major.
type MatrixValue struct {
	Members [][]Value
	ValType BuiltinType
}

func (v MatrixValue) Type() Type { return &v.ValType }

// NewMatrixValue constructs a new matrix value. The number of colums and rows is determined
// using the type argument. The member callback function should return a value for each matrix
// member, given its column and row. The function is called in column-major order. The type of
// the created value will always be in a canonical (TMat2x2 instead of TMat2) form.
func NewMatrixValue(t BareType, member func(col, row uint8) Value) MatrixValue {
	col, row := TypeDimensions(t)
	return NewMatrixValueCR(col, row, member)
}

// NewMatrixValueCR constructs a new matrix value, given the number of its columns and rows. It
// is possible (and occasionally useful) to create matrix values with invalid sizes (number of
// columns of rows equal to 1 or greater than 4), but then the ValType field will not be
// populated correctly. The function is called in column-major order. The type of the created
// value will always be in a canonical (TMat2x2 instead of TMat2) form.
func NewMatrixValueCR(col, row uint8, member func(col, row uint8) Value) MatrixValue {
	m := MatrixValue{Members: make([][]Value, col)}
	if col >= 2 && col <= 4 && row >= 2 && row <= 4 {
		m.ValType.Type = matrixTypes[col][row]
	}
	for i := range m.Members {
		m.Members[i] = make([]Value, row)
		for j := range m.Members[i] {
			m.Members[i][j] = member(uint8(i), uint8(j))
		}
	}
	return m
}

// ArrayValue represents values of array type. Arrays of size zero are illegal.
type ArrayValue []Value

// Type returns the type of the value. The returned type will always be an *ArrayType and it will
// have correctly populated Base, Size and ComputedSize fields.
func (v ArrayValue) Type() Type {
	return &ArrayType{
		Base:         v[0].Type(),
		Size:         &ConstantExpr{Value: UintValue(len(v))},
		ComputedSize: UintValue(len(v)),
	}
}

// StructValue represents values of structure types.
type StructValue struct {
	Members map[string]Value
	ValType *StructType
}

func (v StructValue) Type() Type { return v.ValType }

// NewStructValue construct a new struct value, given its type and a callback function. The
// function should return the value for a member, given its name.
func NewStructValue(t *StructType, member func(name string) Value) StructValue {
	s := StructValue{
		Members: make(map[string]Value),
		ValType: t,
	}
	for _, mvd := range t.Sym.Vars {
		for _, v := range mvd.Vars {
			s.Members[v.Name()] = member(v.Name())
		}
	}
	return s
}

type FunctionValue struct {
	Func    func(v []Value) Value
	ValType *FunctionType
}

func (v FunctionValue) Type() Type { return v.ValType }

// ValueEquals compares two values for equality. The syntax of the GLES Shading
// Language does not permit comparing function values and this function will
// always return false for them.
func ValueEquals(left, right Value) bool {
	switch left := left.(type) {
	case FloatValue, IntValue, UintValue, BoolValue:
		return left == right

	case VectorValue:
		right, ok := right.(VectorValue)
		if !ok || len(left.Members) != len(right.Members) {
			return false
		}
		for i := range left.Members {
			if !ValueEquals(left.Members[i], right.Members[i]) {
				return false
			}
		}
		return true

	case MatrixValue:
		right, ok := right.(MatrixValue)
		if !ok || len(left.Members) != len(right.Members) {
			return false
		}
		for i := range left.Members {
			if len(left.Members[i]) != len(right.Members[i]) {
				return false
			}
			for j := range left.Members[i] {
				if !ValueEquals(left.Members[i][j], right.Members[i][j]) {
					return false
				}
			}
		}
		return true

	case ArrayValue:
		right, ok := right.(ArrayValue)
		if !ok || len(left) != len(right) {
			return false
		}
		for i := range left {
			if !ValueEquals(left[i], right[i]) {
				return false
			}
		}
		return true

	case StructValue:
		right, ok := right.(StructValue)
		if !ok || len(left.Members) != len(right.Members) {
			return false
		}
		for k, v := range left.Members {
			if !ValueEquals(v, right.Members[k]) {
				return false
			}
		}
		return true

	default:
		return false
	}
}
