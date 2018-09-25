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
	"llvm/bindings/go/llvm"
)

// Cast casts the value v to the type ty.
func (v *Value) Cast(ty Type) *Value {
	srcTy, dstTy := v.Type(), ty
	if srcTy == dstTy {
		return v // No-op
	}
	if IsVector(srcTy) != IsVector(dstTy) {
		fail("Cannot cast between vector and non-vector types. (%v -> %v)", srcTy, dstTy)
	}
	if IsPointer(srcTy) && IsPointer(dstTy) {
		name := srcTy.TypeName() + "->" + dstTy.TypeName()
		return v.b.val(ty, v.b.llvm.CreatePointerCast(v.llvm, ty.llvmTy(), name))
	}
	assertVectorsSameLength(srcTy, dstTy)
	srcElTy, dstElTy := Scalar(srcTy), Scalar(dstTy)
	grow := srcElTy.SizeInBits() < dstElTy.SizeInBits()
	shink := srcElTy.SizeInBits() > dstElTy.SizeInBits()
	switch {
	case IsInteger(srcElTy) && IsPointer(dstElTy):
		return v.b.val(ty, v.b.llvm.CreateIntToPtr(v.llvm, ty.llvmTy(), ""))
	case IsPointer(srcElTy) && IsInteger(dstElTy):
		return v.b.val(ty, v.b.llvm.CreatePtrToInt(v.llvm, ty.llvmTy(), ""))
	case IsSignedInteger(srcElTy) && IsFloat(dstElTy):
		return v.b.val(ty, v.b.llvm.CreateSIToFP(v.llvm, ty.llvmTy(), ""))
	case IsUnsignedInteger(srcElTy) && IsFloat(dstElTy):
		return v.b.val(ty, v.b.llvm.CreateUIToFP(v.llvm, ty.llvmTy(), ""))
	case IsFloat(srcElTy) && IsSignedInteger(dstElTy):
		return v.b.val(ty, v.b.llvm.CreateFPToSI(v.llvm, ty.llvmTy(), ""))
	case IsFloat(srcElTy) && IsUnsignedInteger(dstElTy):
		return v.b.val(ty, v.b.llvm.CreateFPToUI(v.llvm, ty.llvmTy(), ""))
	case IsBool(srcElTy) && IsInteger(dstElTy):
		return v.b.val(ty, v.b.llvm.CreateZExt(v.llvm, ty.llvmTy(), ""))
	case IsInteger(srcElTy) && IsBool(dstElTy):
		return v.b.NotEqual(v, v.b.Zero(srcElTy))
	case IsUnsignedInteger(srcElTy) && IsIntegerOrEnum(dstElTy) && grow:
		return v.b.val(ty, v.b.llvm.CreateZExt(v.llvm, ty.llvmTy(), ""))
	case IsSignedIntegerOrEnum(srcElTy) && IsIntegerOrEnum(dstElTy) && grow:
		return v.b.val(ty, v.b.llvm.CreateSExt(v.llvm, ty.llvmTy(), ""))
	case IsIntegerOrEnum(srcElTy) && IsIntegerOrEnum(dstElTy) && shink:
		return v.b.val(ty, v.b.llvm.CreateTrunc(v.llvm, ty.llvmTy(), ""))
	case IsIntegerOrEnum(srcElTy) && IsIntegerOrEnum(dstElTy):
		return v.b.val(ty, v.llvm) // signed conversion
	case IsFloat(srcElTy) && IsFloat(dstElTy) && shink:
		return v.b.val(ty, v.b.llvm.CreateFPTrunc(v.llvm, ty.llvmTy(), ""))
	case IsFloat(srcElTy) && IsFloat(dstElTy) && grow:
		return v.b.val(ty, v.b.llvm.CreateFPExt(v.llvm, ty.llvmTy(), ""))
	default:
		fail("Cannot cast from %v -> %v", srcTy.TypeName(), dstTy.TypeName())
		return nil
	}
}

// Bitcast performs a bitwise cast of the value v to the type ty.
func (v *Value) Bitcast(ty Type) *Value {
	srcTy, dstTy := v.Type(), ty
	if srcTy == dstTy {
		return v // No-op
	}
	if srcSize, dstSize := srcTy.SizeInBits(), dstTy.SizeInBits(); srcSize != dstSize {
		fail("Bitcast cannot change sizes. (%v -> %v)", srcTy, dstTy)
	}
	return v.b.val(ty, v.b.llvm.CreateBitCast(v.llvm, ty.llvmTy(), "bitcast"))
}

// Cast casts the constant v to the type ty.
func (v Const) Cast(ty Type) Const {
	srcTy, dstTy := v.Type, ty
	if srcTy == dstTy {
		return v // No-op
	}
	if IsVector(srcTy) != IsVector(dstTy) {
		fail("Cannot cast between vector and non-vector types. (%v -> %v)", srcTy, dstTy)
	}
	if IsPointer(srcTy) && IsPointer(dstTy) {
		return Const{ty, llvm.ConstPointerCast(v.llvm, ty.llvmTy())}
	}
	assertVectorsSameLength(srcTy, dstTy)
	srcElTy, dstElTy := Scalar(srcTy), Scalar(dstTy)
	grow := srcElTy.SizeInBits() < dstElTy.SizeInBits()
	shink := srcElTy.SizeInBits() > dstElTy.SizeInBits()
	switch {
	case IsInteger(srcElTy) && IsPointer(dstElTy):
		return Const{ty, llvm.ConstIntToPtr(v.llvm, ty.llvmTy())}
	case IsSignedInteger(srcElTy) && IsFloat(dstElTy):
		return Const{ty, llvm.ConstSIToFP(v.llvm, ty.llvmTy())}
	case IsUnsignedInteger(srcElTy) && IsFloat(dstElTy):
		return Const{ty, llvm.ConstUIToFP(v.llvm, ty.llvmTy())}
	case IsFloat(srcElTy) && IsSignedInteger(dstElTy):
		return Const{ty, llvm.ConstFPToSI(v.llvm, ty.llvmTy())}
	case IsFloat(srcElTy) && IsUnsignedInteger(dstElTy):
		return Const{ty, llvm.ConstFPToUI(v.llvm, ty.llvmTy())}
	case IsBool(srcElTy) && IsInteger(dstElTy):
		return Const{ty, llvm.ConstZExt(v.llvm, ty.llvmTy())}
	case IsInteger(srcElTy) && IsUnsignedInteger(dstElTy) && grow:
		return Const{ty, llvm.ConstZExt(v.llvm, ty.llvmTy())}
	case IsInteger(srcElTy) && IsSignedInteger(dstElTy) && grow:
		return Const{ty, llvm.ConstSExt(v.llvm, ty.llvmTy())}
	case IsInteger(srcElTy) && IsInteger(dstElTy) && shink:
		return Const{ty, llvm.ConstTrunc(v.llvm, ty.llvmTy())}
	case IsInteger(srcElTy) && IsInteger(dstElTy):
		return Const{ty, v.llvm} // signed conversion
	default:
		fail("Cannot cast from %v -> %v", srcTy.TypeName(), dstTy.TypeName())
		return Const{}
	}
}

// Cast casts the global value to the given type.
func (v Global) Cast(ty Type) Global {
	return Global(Const(v).Cast(ty))
}
