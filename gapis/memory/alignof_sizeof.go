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

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/core/os/device"
)

// AlignOf returns the byte alignment of the type t.
func AlignOf(t reflect.Type, m *device.MemoryLayout) uint64 {
	switch t.Kind() {
	case reflect.Uint8:
		if t.Implements(tyCharTy) {
			return uint64(m.GetChar().GetAlignment())
		}
		return uint64(m.GetI8().GetAlignment())
	case reflect.Bool, reflect.Int8:
		return uint64(m.GetI8().GetAlignment())
	case reflect.Int16, reflect.Uint16:
		return uint64(m.GetI16().GetAlignment())
	case reflect.Int32, reflect.Uint32:
		return uint64(m.GetI32().GetAlignment())
	case reflect.Float32:
		return uint64(m.GetF32().GetAlignment())
	case reflect.Float64:
		return uint64(m.GetF64().GetAlignment())
	case reflect.Int64, reflect.Uint64:
		switch {
		case t.Implements(tyPointer):
			return uint64(m.GetPointer().GetAlignment())
		case t.Implements(tyIntTy) || t.Implements(tyUintTy):
			return uint64(m.GetInteger().GetAlignment())
		case t.Implements(tySizeTy):
			return uint64(m.GetSize().GetAlignment())
		default:
			return uint64(m.GetI64().GetAlignment())
		}
	case reflect.Int, reflect.Uint:
		return uint64(m.GetInteger().GetAlignment())
	case reflect.Array, reflect.Slice:
		return AlignOf(t.Elem(), m)
	case reflect.String:
		return 1
	}

	switch {
	case t.Implements(tyPointer):
		return uint64(m.GetPointer().GetAlignment())
	case t.Implements(tyAlignedTy):
		return reflect.New(t).Interface().(AlignedTy).TypeAlignment(m)
	}

	panic(fmt.Errorf("MemoryLayout.AlignOf not implemented for type %v (%v)", t, t.Kind()))
}

// SizeOf returns the byte size of the type t.
func SizeOf(t reflect.Type, m *device.MemoryLayout) uint64 {
	switch t.Kind() {
	case reflect.Uint8:
		if t.Implements(tyCharTy) {
			return uint64(m.GetChar().GetSize())
		}
		return uint64(m.GetI8().GetSize())
	case reflect.Bool, reflect.Int8:
		return uint64(m.GetI8().GetSize())
	case reflect.Int16, reflect.Uint16:
		return uint64(m.GetI16().GetSize())
	case reflect.Int32, reflect.Uint32:
		return uint64(m.GetI32().GetSize())
	case reflect.Float32:
		return uint64(m.GetF32().GetSize())
	case reflect.Float64:
		return uint64(m.GetF64().GetSize())
	case reflect.Int64, reflect.Uint64:
		switch {
		case t.Implements(tyPointer):
			return uint64(m.GetPointer().GetSize())
		case t.Implements(tyIntTy) || t.Implements(tyUintTy):
			return uint64(m.GetInteger().GetSize())
		case t.Implements(tySizeTy):
			return uint64(m.GetSize().GetSize())
		default:
			return uint64(m.GetI64().GetSize())
		}
	case reflect.Int, reflect.Uint:
		return uint64(m.GetInteger().GetSize())
	case reflect.Array:
		return SizeOf(t.Elem(), m) * uint64(t.Len())
	case reflect.String:
		return 1
	}

	switch {
	case t.Implements(tyPointer):
		return uint64(m.GetPointer().GetSize())
	case t.Implements(tySizedTy):
		return reflect.New(t).Interface().(SizedTy).TypeSize(m)
	}

	panic(fmt.Errorf("MemoryLayout.SizeOf not implemented for type %v (%v)", t, t.Kind()))
}
