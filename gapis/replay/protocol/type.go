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

package protocol

import "fmt"

// Size returns the size in bytes of the type. pointerSize is the size in bytes
// of a pointer for the target architecture.
func (t Type) Size(pointerSize int32) int {
	switch t {
	case Type_Bool:
		return 1
	case Type_Int8:
		return 1
	case Type_Int16:
		return 2
	case Type_Int32:
		return 4
	case Type_Int64:
		return 8
	case Type_Uint8:
		return 1
	case Type_Uint16:
		return 2
	case Type_Uint32:
		return 4
	case Type_Uint64:
		return 8
	case Type_Float:
		return 4
	case Type_Double:
		return 8
	case Type_AbsolutePointer:
		return int(pointerSize)
	case Type_ConstantPointer:
		return int(pointerSize)
	case Type_VolatilePointer:
		return int(pointerSize)
	case Type_Void:
		return 0
	default:
		panic(fmt.Errorf("Unknown ValueType %v", t))
	}
}

// Returns the protocol_type needed for a Size member for the given
// pointer width
func SizeType(pointerSize int32) Type {
	if pointerSize == 4 {
		return Type_Uint32
	} else if pointerSize == 8 {
		return Type_Uint64
	}
	panic(fmt.Errorf("Unknown Pointer Width %v", pointerSize))
}
