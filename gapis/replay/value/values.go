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

package value

import (
	"math"

	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/replay/protocol"
)

// Bool is a Value of type TypeBool.
type Bool bool

// Get returns TypeBool and 1 if the Bool is true, otherwise 0.
func (v Bool) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	if v {
		return protocol.Type_Bool, 1, false
	} else {
		return protocol.Type_Bool, 0, false
	}
}

// U8 is a Value of type TypeUint8.
type U8 uint8

// Get returns TypeUint8 and the value zero-extended to a uint64.
func (v U8) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_Uint8, uint64(v), false
}

// S8 is a Value of type TypeInt8.
type S8 int8

// Get returns TypeInt8 and the value sign-extended to a uint64.
func (v S8) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_Int8, uint64(v), false
}

// U16 is a Value of type TypeUint16.
type U16 uint16

// Get returns TypeUint16 and the value zero-extended to a uint64.
func (v U16) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_Uint16, uint64(v), false
}

// S16 is a Value of type TypeInt16.
type S16 int16

// Get returns TypeInt16 and the value sign-extended to a uint64.
func (v S16) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_Int16, uint64(v), false
}

// F32 is a Value of type TypeFloat.
type F32 float32

// Get returns TypeFloat and the IEEE 754 representation of the value packed
// into the low part of a uint64.
func (v F32) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_Float, uint64(math.Float32bits(float32(v))), false
}

// U32 is a Value of type TypeUint32.
type U32 uint32

// Get returns TypeUint32 and the value zero-extended to a uint64.
func (v U32) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_Uint32, uint64(v), false
}

// S32 is a Value of type TypeInt32.
type S32 int32

// Get returns TypeInt32 and the value sign-extended to a uint64.
func (v S32) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_Int32, uint64(v), false
}

// F64 is a Value of type TypeDouble.
type F64 float64

// Get returns TypeDouble and the IEEE 754 representation of the value packed
// into a uint64.
func (v F64) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_Double, math.Float64bits(float64(v)), false
}

// U64 is a Value of type TypeUint64.
type U64 uint64

// Get returns TypeUint64 the value zero-extended to a uint64.
func (v U64) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_Uint64, uint64(v), false
}

// S64 is a Value of type TypeInt64.
type S64 int64

// Get returns TypeInt64 and the value reinterpreted as a uint64.
func (v S64) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_Int64, uint64(v), false
}

// AbsoluteStackPointer represents a pointer on the top of the stack in the
// absolute address-space that will not be altered before being passed to the
// protocol.
type AbsoluteStackPointer struct{}

// Get returns TypeAbsolutePointer and the uint64 value of the absolute pointer.
func (p AbsoluteStackPointer) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_AbsolutePointer, 0, true
}

// Offset returns the sum of the pointer with offset.
func (p AbsoluteStackPointer) Offset(offset uint64) Pointer {
	panic("AbsoluteStackPointer.Offset is not implemented")
}

// IsValid returns true for all absolute pointers.
func (p AbsoluteStackPointer) IsValid() bool { return true }

// AbsolutePointer is a pointer in the absolute address-space that will not be
// altered before being passed to the protocol.
type AbsolutePointer uint64

// Get returns TypeAbsolutePointer and the uint64 value of the absolute pointer.
func (p AbsolutePointer) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_AbsolutePointer, uint64(p), false
}

// Offset returns the sum of the pointer with offset.
func (p AbsolutePointer) Offset(offset uint64) Pointer {
	return p + AbsolutePointer(offset)
}

// IsValid returns true for all absolute pointers.
func (p AbsolutePointer) IsValid() bool { return true }

// ObservedPointer is a pointer that was observed at capture time.
// Pointers of this type are remapped to an equivalent volatile address-space
// pointer, or absolute address-space pointer before being passed to the
// protocol.
type ObservedPointer uint64

// Get returns the pointer type and the pointer translated to either an
// equivalent volatile address-space pointer or absolute pointer.
func (p ObservedPointer) Get(r PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	ty, val = r.ResolveObservedPointer(p)
	return ty, val, false
}

// Offset returns the sum of the pointer with offset.
func (p ObservedPointer) Offset(offset uint64) Pointer {
	return p + ObservedPointer(offset)
}

// Anything very low in application address-space is extremely
// unlikely to be a valid pointer.
const FirstValidAddress = 0x1001

var ValidMemoryRanges = interval.U64RangeList{
	interval.U64Range{First: FirstValidAddress, Count: math.MaxUint64 - FirstValidAddress},
}

// IsValid returns true if the pointer considered valid. Currently this is a
// test for the pointer being greater than 0x1000 as low addresses are likely
// to be a wrong interpretation of the value. This may change in the future.
func (p ObservedPointer) IsValid() bool {
	return p >= FirstValidAddress
}

// PointerIndex is an index to a pointer in the pointer table.
type PointerIndex uint64

// Get returns TypeVolatilePointer and the volatile address of the pointer.
func (p PointerIndex) Get(r PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	ty, val = r.ResolvePointerIndex(p)
	return ty, val, false
}

// Offset returns the sum of the pointer index with offset.
func (p PointerIndex) Offset(offset uint64) Pointer {
	return p + PointerIndex(offset)
}

// IsValid returns true.
func (p PointerIndex) IsValid() bool {
	return true
}

// VolatilePointer is a pointer to the volatile address-space.
// Unlike ObservedPointer, there is no remapping.
type VolatilePointer uint64

// Get returns TypeVolatilePointer and the uint64 value of the pointer in
// volatile address-space.
func (p VolatilePointer) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_VolatilePointer, uint64(p), false
}

// Offset returns the sum of the pointer with offset.
func (p VolatilePointer) Offset(offset uint64) Pointer {
	return p + VolatilePointer(offset)
}

// IsValid returns true.
func (p VolatilePointer) IsValid() bool { return true }

// TemporaryPointer is a pointer to in temporary address-space.
// The temporary address-space sits within a reserved area of the the volatile
// address space and its offset is calculated dynamically.
// TODO: REMOVE
type TemporaryPointer uint64

// Get returns TypeVolatilePointer and the dynamically calculated offset of the
// temporary pointer within volatile address-space.
func (p TemporaryPointer) Get(r PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_VolatilePointer, uint64(r.ResolveTemporaryPointer(p)), false
}

// Offset returns the sum of the pointer with offset.
func (p TemporaryPointer) Offset(offset uint64) Pointer {
	return p + TemporaryPointer(offset)
}

// IsValid returns true.
func (p TemporaryPointer) IsValid() bool { return true }

// ConstantPointer is a pointer in the constant address-space that will not be
// altered before being passed to the protocol.
type ConstantPointer uint64

// Get returns TypeConstantPointer and the uint64 value of the pointer in constant address-space.
func (p ConstantPointer) Get(PointerResolver) (ty protocol.Type, val uint64, onStack bool) {
	return protocol.Type_ConstantPointer, uint64(p), false
}

// Offset returns the sum of the pointer with offset.
func (p ConstantPointer) Offset(offset uint64) Pointer {
	return p + ConstantPointer(offset)
}

// IsValid returns true.
func (p ConstantPointer) IsValid() bool { return true }
