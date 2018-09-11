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
)

// Write writes the value v to the encoder e using C alignment rules.
// If v is an array or slice, then each of the elements will be written,
// sequentially. Zeros are used as for paddinge.
func Write(e *Encoder, v interface{}) {
	encode(e, reflect.ValueOf(v))
}

func encode(e *Encoder, v reflect.Value) {
	t := v.Type()

	if t.Implements(tyEncodable) {
		v.Interface().(Encodable).Encode(e)
		return
	}

	switch t.Kind() {
	case reflect.Float32:
		e.F32(float32(v.Float()))
	case reflect.Float64:
		e.F64(v.Float())
	case reflect.Int8:
		e.I8(int8(v.Int()))
	case reflect.Int16:
		e.I16(int16(v.Int()))
	case reflect.Int32:
		e.I32(int32(v.Int()))
	case reflect.Int64:
		if t.Implements(tyIntTy) {
			e.Int(Int(v.Int()))
		} else {
			e.I64(v.Int())
		}
	case reflect.Uint8:
		if t.Implements(tyCharTy) {
			e.Char(Char(v.Uint()))
		} else {
			e.U8(uint8(v.Uint()))
		}
	case reflect.Uint16:
		e.U16(uint16(v.Uint()))
	case reflect.Uint32:
		e.U32(uint32(v.Uint()))
	case reflect.Uint64:
		switch {
		case t.Implements(tyPointer):
			e.Pointer(v.Uint())
		case t.Implements(tySizeTy):
			e.Size(Size(v.Uint()))
		case t.Implements(tyUintTy):
			e.Uint(Uint(v.Uint()))
		default:
			e.U64(v.Uint())
		}
	case reflect.Int:
		e.Int(Int(v.Int()))
	case reflect.Uint:
		e.Uint(Uint(v.Uint()))
	case reflect.Array:
		for i, c := 0, v.Len(); i < c; i++ {
			encode(e, v.Index(i))
		}
	case reflect.Slice:
		if t.Elem() == tyUint8Ty {
			e.Data(v.Interface().([]uint8))
		} else {
			for i, c := 0, v.Len(); i < c; i++ {
				encode(e, v.Index(i))
			}
		}
	case reflect.Struct:
		e.Align(AlignOf(v.Type(), e.m))
		base := e.o
		for i, c := 0, v.NumField(); i < c; i++ {
			encode(e, v.Field(i))
		}
		written := e.o - base
		padding := SizeOf(v.Type(), e.m) - written
		e.Pad(padding)
	case reflect.String:
		e.String(v.String())
	case reflect.Bool:
		e.Bool(v.Bool())
	case reflect.Interface, reflect.Ptr:
		encode(e, v.Elem())
	default:
		switch {
		case t.Implements(tyPointer):
			e.Pointer(v.Interface().(Pointer).Address())
		default:
			panic(fmt.Errorf("Cannot write type: %v", t))
		}
	}

}
