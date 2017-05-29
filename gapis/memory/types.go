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

package memory

import "reflect"

var (
	tyPointer = reflect.TypeOf((*Pointer)(nil)).Elem()
	tyChar    = reflect.TypeOf((*CharTy)(nil)).Elem()
	tyInt     = reflect.TypeOf((*IntTy)(nil)).Elem()
	tyUint    = reflect.TypeOf((*UintTy)(nil)).Elem()
	tySize    = reflect.TypeOf((*SizeTy)(nil)).Elem()
)

// Int is a signed integer type.
type Int int64

// IntTy is the interface implemented by types that should be treated as int type.
type IntTy interface {
	IsInt()
}

// Dummy function to make Int implement IntTy interface
func (Int) IsInt() {}

// Uint is an unsigned integer type.
type Uint uint64

// UintTy is the interface implemented by types that should be treated as uint type.
type UintTy interface {
	IsUint()
}

// Dummy function to make Uint implement UintTy interface
func (Uint) IsUint() {}

// Char is the possibly signed but maybe unsigned C/C++ char.
type Char uint8

// UintTy is the interface implemented by types that should be treated as char type.
type CharTy interface {
	IsChar()
}

// Dummy function to make Uint implement CharTy interface
func (Char) IsChar() {}

// CharToBytes changes the Char values to their byte[] representation.
func CharToBytes(ϟchars []Char) []byte {
	bytes := make([]byte, len(ϟchars))
	for i := range ϟchars {
		bytes[i] = byte(ϟchars[i])
	}
	return bytes
}

// Size is a size_t type.
type Size uint64

// SizeTy is the interface implemented by types that should be treated as size_t type.
type SizeTy interface {
	IsMemorySize()
}

// Dummy function to make Size implement SizeTy interface
func (Size) IsMemorySize() {}

// IsSize returns true if v is a Size or alias to a Size.
func IsSize(v interface{}) bool {
	_, ok := v.(SizeTy)
	return ok
}
