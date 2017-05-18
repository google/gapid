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
	tyInt     = reflect.TypeOf(Int(0))
	tyUint    = reflect.TypeOf(Uint(0))
	tySize    = reflect.TypeOf(Size(0))
)

// Int is a signed integer type.
type Int int64

// Uint is an unsigned integer type.
type Uint uint64

// Size is a size_t type.
type Size uint64

// Char is the possibly signed but maybe unsigned C/C++ char.
type Char uint8

// CharToBytes changes the Char values to their byte[] representation.
func CharToBytes(ϟchars []Char) []byte {
	bytes := make([]byte, len(ϟchars))
	for i := range ϟchars {
		bytes[i] = byte(ϟchars[i])
	}
	return bytes
}
