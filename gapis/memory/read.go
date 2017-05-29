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

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/os/device"
)

// Read reads the value pointed at p from the reader r using C alignment rules.
// If v is an array or slice, then each of the elements will be read,
// sequentially.
func Read(r binary.Reader, m *device.MemoryLayout, p interface{}) {
	v := reflect.ValueOf(p)
	if v.Kind() != reflect.Ptr {
		panic(fmt.Errorf("p must be pointer, got %T", p))
	}
	decode(NewDecoder(r, m), v)
}

func decode(d *Decoder, v reflect.Value) {
	t := v.Type()
	switch {
	case t.Implements(tyPointer):
		p := v.Interface().(Pointer).Set(d.Pointer(), ApplicationPool)
		v.Set(reflect.ValueOf(p))
	case t.Implements(tyChar):
		v.SetUint(uint64(d.Char()))
	case t.Implements(tyInt):
		v.SetInt(int64(d.Int()))
	case t.Implements(tyUint):
		v.SetUint(uint64(d.Uint()))
	case t.Implements(tySize):
		v.SetUint(uint64(d.Size()))
	default:

		switch t.Kind() {
		case reflect.Float32:
			v.SetFloat(float64(d.F32()))
		case reflect.Float64:
			v.SetFloat(d.F64())
		case reflect.Int8:
			v.SetInt(int64(d.I8()))
		case reflect.Int16:
			v.SetInt(int64(d.I16()))
		case reflect.Int32:
			v.SetInt(int64(d.I32()))
		case reflect.Int64:
			v.SetInt(d.I64())
		case reflect.Uint8:
			v.SetUint(uint64(d.U8()))
		case reflect.Uint16:
			v.SetUint(uint64(d.U16()))
		case reflect.Uint32:
			v.SetUint(uint64(d.U32()))
		case reflect.Uint64:
			v.SetUint(d.U64())
		case reflect.Int:
			v.SetInt(int64(d.Int()))
		case reflect.Uint:
			v.SetUint(uint64(d.Uint()))
		case reflect.Array, reflect.Slice:
			for i, c := 0, v.Len(); i < c; i++ {
				decode(d, v.Index(i))
			}
		case reflect.Struct:
			d.Align(AlignOf(v.Type(), d.m))
			base := d.o
			for i, c := 0, v.NumField(); i < c; i++ {
				decode(d, v.Field(i))
			}
			read := d.o - base
			padding := SizeOf(v.Type(), d.m) - read
			d.Skip(padding)
		case reflect.String:
			v.SetString(d.String())
		case reflect.Bool:
			v.SetBool(d.Bool())
		case reflect.Interface, reflect.Ptr:
			decode(d, v.Elem())
		default:
			panic(fmt.Errorf("Cannot write type: %v", t))
		}
	}
}
