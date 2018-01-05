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

// Package amd64 represents the AMD64 System V Application Binary Interface.
//
// See: https://github.com/hjl-tools/x86-psABI/wiki/x86-64-psABI-r252.pdf
//
// This package is incomplete and may be removed.
package amd64

import (
	"github.com/google/gapid/core/codegen"
)

type regClass int

// Argument class types
const (
	regPointer = regClass(iota)
	regInteger
	regSSE
	regSSEUP
	regX87
	regX87UP
	regComplexX87
	regNoClass
	regMemory
)

// See: 3.2.3 Parameter Passing
func classifyParameter(ty codegen.Type) {
	switch {
	case codegen.IsPointer(ty):
		// Pointers are in the POINTER class.
		return regPointer
	case codegen.IsBool(ty), codegen.IsInteger(ty):
		// Arguments of types (signed and unsigned) _Bool, char, short, int,
		// long and long long are in the INTEGER class.
		return regInteger
	case codegen.IsFloat(ty):
		// Arguments of types float, double, _Decimal32, _Decimal64 and __m64
		// are in class SSE.
		return regSSE
	case codegen.IsStruct(ty):
		return classifyStruct(ty.(*codegen.Struct))
	case codegen.IsArray(ty):
		// TODO
	}
}

func classifyStruct(ty *codegen.Struct) {
	// If the size of an object is larger than eight eightbytes, or it contains
	// unaligned fields, it has class MEMORY.
	if sizeof(ty) > 8*8 {
		return regMemory
	}
	if ty.Packed {
		// Unaligned fields are currently unsupported.
		panic("Packed structs are not supported")
	}

}
