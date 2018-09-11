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

package codegen

import (
	"fmt"
	"reflect"

	"llvm/bindings/go/llvm"
)

// Zero returns a zero-value of the specified type.
func (b *Builder) Zero(ty Type) *Value {
	if ty == b.m.Types.Void {
		return nil
	}
	return b.val(ty, llvm.ConstNull(ty.llvmTy())).SetName("zero")
}

// One returns a one-value of the specified bool, int or float type.
func (b *Builder) One(ty Type) *Value {
	var value *Value
	switch {
	case ty == b.m.Types.Bool:
		value = b.val(ty, llvm.ConstInt(ty.llvmTy(), 1, false))
	case IsInteger(ty):
		value = b.val(ty, llvm.ConstInt(ty.llvmTy(), 1, false))
	case IsFloat(ty):
		value = b.val(ty, llvm.ConstFloat(ty.llvmTy(), 1))
	default:
		fail("One does not support type %T", ty)
	}
	value.SetName("1")
	return value
}

// Not returns !x. The type of x must be Bool.
func (b *Builder) Not(x *Value) *Value {
	assertTypesEqual(x.ty, b.m.Types.Bool)
	return b.val(b.m.Types.Bool, b.llvm.CreateNot(x.llvm, "!"+x.Name()))
}

// Invert returns ~x.
func (b *Builder) Invert(x *Value) *Value {
	return b.val(x.ty, b.llvm.CreateNot(x.llvm, ""))
}

// Negate returns -x. The type of x must be a signed integer or float.
func (b *Builder) Negate(x *Value) *Value {
	ty := Scalar(x.Type())
	name := "-" + x.Name()
	switch {
	case IsSignedInteger(ty):
		return b.val(ty, b.llvm.CreateNeg(x.llvm, name))
	case IsFloat(ty):
		return b.val(ty, b.llvm.CreateFNeg(x.llvm, name))
	default:
		panic(fmt.Errorf("Cannot divide values of type %v", ty))
	}
}

func (b *Builder) cmp(x *Value, op string, y *Value, ucmp, scmp llvm.IntPredicate, fcmp llvm.FloatPredicate) *Value {
	assertTypesEqual(x.Type(), y.Type())
	ty := x.Type()
	if vec, isVec := ty.(Vector); isVec {
		ty = vec.Element
	}
	var v *Value
	switch {
	case IsSignedInteger(ty):
		v = b.val(b.m.Types.Bool, b.llvm.CreateICmp(scmp, x.llvm, y.llvm, ""))
	case IsUnsignedInteger(ty), IsPointer(ty), IsBool(ty):
		v = b.val(b.m.Types.Bool, b.llvm.CreateICmp(ucmp, x.llvm, y.llvm, ""))
	case IsFloat(ty):
		v = b.val(b.m.Types.Bool, b.llvm.CreateFCmp(fcmp, x.llvm, y.llvm, ""))
	default:
		panic(fmt.Errorf("Cannot compare %v types with %v", ty.TypeName(), op))
	}
	v.SetName(x.Name() + op + y.Name())
	return v
}

// Equal returns x == y. The types of the two values must be equal.
func (b *Builder) Equal(x, y *Value) *Value {
	ty := x.Type()
	if ty, ok := ty.(*Struct); ok {
		assertTypesEqual(x.Type(), y.Type())
		var eq *Value
		for i, f := range ty.Fields() {
			x, y := x.Extract(f.Name), y.Extract(f.Name)
			if i == 0 {
				eq = b.Equal(x, y)
			} else {
				eq = b.And(eq, b.Equal(x, y))
			}
		}
		return eq
	}
	return b.cmp(x, "==", y, llvm.IntEQ, llvm.IntEQ, llvm.FloatOEQ)
}

// NotEqual returns x != y. The types of the two values must be equal.
func (b *Builder) NotEqual(x, y *Value) *Value {
	ty := x.Type()
	if ty, ok := ty.(*Struct); ok {
		assertTypesEqual(x.Type(), y.Type())
		var neq *Value
		for i, f := range ty.Fields() {
			x, y := x.Extract(f.Name), y.Extract(f.Name)
			if i == 0 {
				neq = b.NotEqual(x, y)
			} else {
				neq = b.Or(neq, b.NotEqual(x, y))
			}
		}
		return neq
	}
	return b.cmp(x, "!=", y, llvm.IntNE, llvm.IntNE, llvm.FloatONE)
}

// GreaterThan returns x > y. The types of the two values must be equal.
func (b *Builder) GreaterThan(x, y *Value) *Value {
	return b.cmp(x, ">", y, llvm.IntUGT, llvm.IntSGT, llvm.FloatOGT)
}

// LessThan returns x < y. The types of the two values must be equal.
func (b *Builder) LessThan(x, y *Value) *Value {
	return b.cmp(x, "<", y, llvm.IntULT, llvm.IntSLT, llvm.FloatOLT)
}

// GreaterOrEqualTo returns x >= y. The types of the two values must be equal.
func (b *Builder) GreaterOrEqualTo(x, y *Value) *Value {
	return b.cmp(x, ">=", y, llvm.IntUGE, llvm.IntSGE, llvm.FloatOGE)
}

// LessOrEqualTo returns x <= y. The types of the two values must be equal.
func (b *Builder) LessOrEqualTo(x, y *Value) *Value {
	return b.cmp(x, "<=", y, llvm.IntULE, llvm.IntSLE, llvm.FloatOLE)
}

// Add returns x + y. The types of the two values must be equal.
func (b *Builder) Add(x, y *Value) *Value {
	assertTypesEqual(x.Type(), y.Type())
	ty := Scalar(x.Type())
	switch {
	case IsInteger(ty):
		return b.val(x.Type(), b.llvm.CreateAdd(x.llvm, y.llvm, x.Name()+"+"+y.Name()))
	case IsFloat(ty):
		return b.val(x.Type(), b.llvm.CreateFAdd(x.llvm, y.llvm, x.Name()+"+"+y.Name()))
	default:
		panic(fmt.Errorf("Cannot add values of type %v", ty))
	}
}

// AddS returns x + y. The types of the two values must be equal.
func (b *Builder) AddS(x *Value, y interface{}) *Value {
	return b.Add(x, b.Scalar(y))
}

// Sub returns x - y. The types of the two values must be equal.
func (b *Builder) Sub(x, y *Value) *Value {
	assertTypesEqual(x.Type(), y.Type())
	ty := Scalar(x.Type())
	switch {
	case IsInteger(ty):
		return b.val(x.Type(), b.llvm.CreateSub(x.llvm, y.llvm, x.Name()+"-"+y.Name()))
	case IsFloat(ty):
		return b.val(x.Type(), b.llvm.CreateFSub(x.llvm, y.llvm, x.Name()+"-"+y.Name()))
	default:
		panic(fmt.Errorf("Cannot subtract values of type %v", ty))
	}
}

// SubS returns x - y. The types of the two values must be equal.
func (b *Builder) SubS(x *Value, y interface{}) *Value {
	return b.Sub(x, b.Scalar(y))
}

// Mul returns x * y. The types of the two values must be equal.
func (b *Builder) Mul(x, y *Value) *Value {
	assertTypesEqual(x.Type(), y.Type())
	ty := Scalar(x.Type())
	switch {
	case IsInteger(ty):
		return b.val(x.Type(), b.llvm.CreateMul(x.llvm, y.llvm, x.Name()+"*"+y.Name()))
	case IsFloat(ty):
		return b.val(x.Type(), b.llvm.CreateFMul(x.llvm, y.llvm, x.Name()+"*"+y.Name()))
	default:
		panic(fmt.Errorf("Cannot multiply values of type %v", ty))
	}
}

// MulS returns x * y. The types of the two values must be equal.
func (b *Builder) MulS(x *Value, y interface{}) *Value {
	return b.Mul(x, b.Scalar(y))
}

// Div returns x / y. The types of the two values must be equal.
func (b *Builder) Div(x, y *Value) *Value {
	assertTypesEqual(x.Type(), y.Type())
	ty := Scalar(x.Type())
	name := x.Name() + "/" + y.Name()
	switch {
	case IsSignedInteger(ty):
		return b.val(ty, b.llvm.CreateSDiv(x.llvm, y.llvm, name))
	case IsUnsignedInteger(ty):
		return b.val(ty, b.llvm.CreateUDiv(x.llvm, y.llvm, name))
	case IsFloat(ty):
		return b.val(ty, b.llvm.CreateFDiv(x.llvm, y.llvm, name))
	default:
		panic(fmt.Errorf("Cannot divide values of type %v", ty))
	}
}

// Rem returns x % y. The types of the two values must be equal.
func (b *Builder) Rem(x, y *Value) *Value {
	assertTypesEqual(x.Type(), y.Type())
	ty := Scalar(x.Type())
	name := x.Name() + "%" + y.Name()
	switch {
	case IsSignedInteger(ty):
		return b.val(ty, b.llvm.CreateSRem(x.llvm, y.llvm, name))
	case IsUnsignedInteger(ty):
		return b.val(ty, b.llvm.CreateURem(x.llvm, y.llvm, name))
	case IsFloat(ty):
		return b.val(ty, b.llvm.CreateFRem(x.llvm, y.llvm, name))
	default:
		panic(fmt.Errorf("Cannot divide values of type %v", ty))
	}
}

// DivS returns x / y. The types of the two values must be equal.
func (b *Builder) DivS(x *Value, y interface{}) *Value {
	return b.Div(x, b.Scalar(y))
}

// And performs a bitwise-and of the two integers.
// The types of the two values must be equal.
func (b *Builder) And(x, y *Value) *Value {
	assertTypesEqual(x.Type(), y.Type())
	return b.val(x.Type(), b.llvm.CreateAnd(x.llvm, y.llvm, x.Name()+"&"+y.Name()))
}

// Or performs a bitwise-or of the two integers.
// The types of the two values must be equal.
func (b *Builder) Or(x, y *Value) *Value {
	assertTypesEqual(x.Type(), y.Type())
	return b.val(x.Type(), b.llvm.CreateOr(x.llvm, y.llvm, x.Name()+"|"+y.Name()))
}

// Xor performs a bitwise-xor of the two integers.
// The types of the two values must be equal.
func (b *Builder) Xor(x, y *Value) *Value {
	assertTypesEqual(x.Type(), y.Type())
	return b.val(x.Type(), b.llvm.CreateXor(x.llvm, y.llvm, x.Name()+"^"+y.Name()))
}

// Shuffle performs a vector-shuffle operation of the two vector values x and y
// with the specified indices.
func (b *Builder) Shuffle(x, y, indices *Value) *Value {
	assertTypesEqual(x.Type(), y.Type())
	assertVectorsSameLength(x.Type(), y.Type())
	dataTy, ok := x.Type().(Vector)
	if !ok {
		panic(fmt.Errorf("Shuffle must be passed vector types, got: %v", x.Type()))
	}
	indicesTy, ok := indices.Type().(Vector)
	if !ok || indicesTy.Element != b.m.Types.Int32 {
		panic(fmt.Errorf("Shuffle indices must be a vector of Int32, got: %v", indices.Type()))
	}
	ty := b.m.Types.Vector(dataTy.Element, indicesTy.Count)
	return b.val(ty, b.llvm.CreateShuffleVector(x.llvm, y.llvm, indices.llvm, "shuffle"))
}

// ShiftLeft performs a bit-shift left by shift bits.
func (b *Builder) ShiftLeft(x, y *Value) *Value {
	assertTypesEqual(x.Type(), y.Type())
	return b.val(x.Type(), b.llvm.CreateShl(x.llvm, y.llvm, x.Name()+"<<"+y.Name()))
}

// ShiftRight performs a bit-shift right by shift bits.
func (b *Builder) ShiftRight(x, y *Value) *Value {
	assertTypesEqual(x.Type(), y.Type())
	ty := x.Type()
	switch {
	case IsSignedInteger(ty):
		return b.val(ty, b.llvm.CreateAShr(x.llvm, y.llvm, x.Name()+">>"+y.Name()))
	case IsUnsignedInteger(ty):
		return b.val(ty, b.llvm.CreateLShr(x.llvm, y.llvm, x.Name()+">>"+y.Name()))
	default:
		panic(fmt.Errorf("Cannot divide values of type %v", ty))
	}
}

// Select returns (cond ? x : y). x and y must be of the same type.
func (b *Builder) Select(cond, x, y *Value) *Value {
	assertTypesEqual(x.Type(), y.Type())
	assertTypesEqual(cond.Type(), b.m.Types.Bool)
	return b.val(x.Type(), b.llvm.CreateSelect(cond.llvm, x.llvm, y.llvm,
		fmt.Sprintf("(%v?%v:%v)", cond.Name(), x.Name(), y.Name())))
}

// Scalar returns a constant scalar with the value v.
func (b *Builder) Scalar(v interface{}) *Value {
	ty := b.m.Types.TypeOf(v)
	switch {
	case IsStruct(ty):
		ty := ty.(*Struct)
		val := llvm.Undef(ty.llvm)
		r := reflect.ValueOf(v)
		for i, c := 0, r.NumField(); i < c; i++ {
			f := b.Scalar(r.Field(i).Interface()).llvm
			val = b.llvm.CreateInsertValue(val, f, i, "")
		}
		return b.val(ty, val)

	default:
		return b.m.Scalar(v).Value(b)
	}
}

// Vector returns a constant vector with the specified values.
func (b *Builder) Vector(el0 interface{}, els ...interface{}) *Value {
	types := make([]Type, len(els)+1)
	values := make([]llvm.Value, len(els)+1)
	v := b.Scalar(el0)
	types[0] = v.Type()
	values[0] = v.llvm
	allSameType := true
	for i, el := range els {
		v := b.Scalar(el)
		types[i+1] = v.Type()
		values[i+1] = v.llvm
		allSameType = types[0] == types[i+1]
	}
	if !allSameType {
		fail("Vector passed mix of element types: %v", types)
	}
	return b.val(b.m.Types.Vector(types[0], len(values)), llvm.ConstVector(values, false))
}

// VectorN returns a constant vector of n elements with the same value v in each
// element.
func (b *Builder) VectorN(n int, v interface{}) *Value {
	values := make([]llvm.Value, n)
	s := b.Scalar(v)
	for i := range values {
		values[i] = s.llvm
	}
	return b.val(b.m.Types.Vector(s.Type(), len(values)), llvm.ConstVector(values, false))
}
