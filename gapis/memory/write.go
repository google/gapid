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

	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/core/os/device"
)

func getAlignment(memoryLayout *device.MemoryLayout, v interface{}) (uint64, error) {
	if pt := reflect.TypeOf(Pointer{}); reflect.TypeOf(v).ConvertibleTo(pt) {
		return uint64(memoryLayout.GetPointer().GetAlignment()), nil
	}
	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		return 1, nil
	case reflect.Int16, reflect.Uint16:
		return 2, nil
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 4, nil
	case reflect.Float64:
		return 8, nil
	case reflect.Int64, reflect.Uint64:
		return uint64(memoryLayout.GetI64().GetAlignment()), nil
	case reflect.Int, reflect.Uint:
		return uint64(memoryLayout.GetInteger().GetSize()), nil
	case reflect.Array, reflect.Slice:
		return getAlignment(memoryLayout, reflect.ValueOf(v).Index(0).Interface())
	case reflect.String:
		return 1, nil
	case reflect.Struct:
		value := reflect.ValueOf(v)
		alignment := uint64(1)
		for i := 0; i < value.NumField(); i++ {
			a, err := getAlignment(memoryLayout, value.Field(i).Interface())
			if err != nil {
				return 0, err
			}
			if alignment < a {
				alignment = a
			}
		}
		return alignment, nil
	default:
		return 0, fmt.Errorf("alignment calculation for type %v (%v) unimplemented", t, t.Kind())
	}
}

// Write writes the value v to the writer w using C alignment rules.
// If v is an array or slice, then each of the elements will be written,
// sequentially. Zeros are used as for paddings. On success, returns
// the number of bytes written and nil, Otherwise, returns 0 and an error.
func Write(w pod.Writer, memoryLayout *device.MemoryLayout, v interface{}) (uint64, error) {
	// <type>ᵖ types are aliases to Pointer. And alias types are different from
	// the underlying type in Go. We cannot directly use type assertion/switch
	// here to test whether v is essentially of Pointer type.
	if pt := reflect.TypeOf(Pointer{}); reflect.TypeOf(v).ConvertibleTo(pt) {
		v = reflect.ValueOf(v).Convert(pt).Interface()
		pod.WriteUint(w, memoryLayout.GetPointer().GetSize()*8, v.(Pointer).Address)
		return uint64(memoryLayout.GetPointer().GetSize()), w.Error()
	}

	r := reflect.ValueOf(v)
	t := r.Type()
	switch t.Kind() {
	case reflect.Float32:
		w.Float32(float32(r.Float()))
		return 4, nil

	case reflect.Float64:
		w.Float64(r.Float())
		return 8, nil

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		pod.WriteInt(w, int32(t.Bits()), r.Int())
		return uint64(t.Bits() / 8), nil

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		pod.WriteUint(w, int32(t.Bits()), r.Uint())
		return uint64(t.Bits() / 8), nil

	case reflect.Int:
		pod.WriteInt(w, memoryLayout.GetInteger().GetSize()*8, r.Int())
		return uint64(memoryLayout.GetInteger().GetSize()), nil

	case reflect.Uint:
		pod.WriteUint(w, memoryLayout.GetInteger().GetSize()*8, r.Uint())
		return uint64(memoryLayout.GetInteger().GetSize()), nil

	case reflect.Array, reflect.Slice:
		size := uint64(0)
		for i := 0; i < r.Len(); i++ {
			element := r.Index(i).Interface()
			alignment, err := getAlignment(memoryLayout, element)
			if err != nil {
				return 0, err
			}
			newSize := u64.AlignUp(size, alignment)
			pod.WriteBytes(w, 0, int32(newSize-size))
			size = newSize
			s, err := Write(w, memoryLayout, element)

			if err != nil {
				return 0, err
			}
			size += s
		}
		return size, nil

	case reflect.Struct:
		size := uint64(0)
		for i := 0; i < r.NumField(); i++ {
			field := r.Field(i).Interface()
			alignment, err := getAlignment(memoryLayout, field)
			if err != nil {
				return 0, err
			}
			newSize := u64.AlignUp(size, alignment)
			pod.WriteBytes(w, 0, int32(newSize-size))
			size = newSize
			s, err := Write(w, memoryLayout, field)
			if err != nil {
				return 0, err
			}
			size += s
		}
		return size, nil

	case reflect.String:
		s := r.String()
		w.String(s)
		// Since an unsized string cannot be the sub-element of some composite
		// type, the size returned here doesn't really matter regarding as for
		// calculating alignments.
		// However, the number of bytes written still depends on the writer.
		// We assume that the writer null-terminates the string here.
		return uint64(len(s)) + 1, nil

	case reflect.Bool:
		w.Bool(r.Bool())
		return 1, nil

	default:
		return 0, fmt.Errorf("Cannot write type: %s", t.Name())
	}
}
