// Copyright (C) 2019 Google Inc.
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

package types

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/memory"
)

type typeData struct {
	tp *Type
	rt reflect.Type
}

var type_map = make(map[uint64]*typeData)

func AddType(i uint64, t *Type, rt reflect.Type) {
	if _, ok := type_map[i]; ok {
		panic(fmt.Errorf("Error identical types exist %+v", i))
	}
	type_map[i] = &typeData{t, rt}
}

func GetType(i uint64) (*Type, error) {
	t, ok := type_map[i]
	if !ok {
		return nil, fmt.Errorf("Could not find type %v", i)
	}
	return t.tp, nil
}

func TryGetType(i uint64) (*Type, bool) {
	t, ok := type_map[i]
	if !ok {
		return nil, false
	}
	return t.tp, ok
}

func GetReflectedType(i uint64) (reflect.Type, error) {
	t, ok := type_map[i]
	if !ok {
		return nil, fmt.Errorf("Could not find type %v", i)
	}
	return t.rt, nil
}

const BoolType = uint64(1)
const IntType = uint64(2)
const UintType = uint64(3)
const SizeType = uint64(4)
const CharType = uint64(5)
const Uint8Type = uint64(6)
const Int8Type = uint64(7)
const Uint16Type = uint64(8)
const Int16Type = uint64(9)
const Float32Type = uint64(10)
const Uint32Type = uint64(11)
const Int32Type = uint64(12)
const Float64Type = uint64(13)
const Uint64Type = uint64(14)
const Int64Type = uint64(15)
const StringType = uint64(16)

// GetTypeIndex returns the index of the type.
func GetTypeIndex(i interface{}) (uint64, error) {
	switch i := i.(type) {
	case TypeProvider:
		return i.GetTypeIndex(), nil
	case bool:
		return 1, nil
	case memory.IntTy:
		return 2, nil
	case memory.UintTy:
		return 3, nil
	case memory.SizeTy:
		return 4, nil
	case memory.CharTy:
		return 5, nil
	case uint8:
		return 6, nil
	case int8:
		return 7, nil
	case uint16:
		return 8, nil
	case int16:
		return 9, nil
	case float32:
		return 10, nil
	case uint32:
		return 11, nil
	case int32:
		return 12, nil
	case float64:
		return 13, nil
	case uint64:
		return 14, nil
	case int64:
		return 15, nil
	case string:
		return 16, nil
	}
	return 0, fmt.Errorf("Unknown type %T", i)
}

// TypeProvider is an interface for any type that
// has a registered type.
type TypeProvider interface {
	GetTypeIndex() uint64
}

func init() {
	AddType(BoolType, &Type{
		TypeId: BoolType,
		Name:   "bool",
		Ty: &Type_Pod{
			pod.Type_bool,
		},
	}, reflect.TypeOf(bool(false)))
	AddType(Uint8Type, &Type{
		TypeId: Uint8Type,
		Name:   "uint8_t",
		Ty: &Type_Pod{
			pod.Type_uint8,
		},
	}, reflect.TypeOf(uint8(0)))
	AddType(Int8Type, &Type{
		TypeId: Int8Type,
		Name:   "int8_t",
		Ty: &Type_Pod{
			pod.Type_sint8,
		},
	}, reflect.TypeOf(int8(0)))
	AddType(Uint16Type, &Type{
		TypeId: Uint16Type,
		Name:   "uint16_t",
		Ty: &Type_Pod{
			pod.Type_uint16,
		},
	}, reflect.TypeOf(uint16(0)))
	AddType(Int16Type, &Type{
		TypeId: Int16Type,
		Name:   "int16_t",
		Ty: &Type_Pod{
			pod.Type_sint16,
		},
	}, reflect.TypeOf(int16(0)))
	AddType(Float32Type, &Type{
		TypeId: Float32Type,
		Name:   "float32",
		Ty: &Type_Pod{
			pod.Type_float32,
		},
	}, reflect.TypeOf(float32(0)))
	AddType(Uint32Type, &Type{
		TypeId: Uint32Type,
		Name:   "uint32_t",
		Ty: &Type_Pod{
			pod.Type_uint32,
		},
	}, reflect.TypeOf(uint32(0)))
	AddType(Int32Type, &Type{
		TypeId: Int32Type,
		Name:   "int32_t",
		Ty: &Type_Pod{
			pod.Type_sint32,
		},
	}, reflect.TypeOf(int32(0)))
	AddType(Float64Type, &Type{
		TypeId: Float64Type,
		Name:   "float64",
		Ty: &Type_Pod{
			pod.Type_float64,
		},
	}, reflect.TypeOf(float64(0)))
	AddType(Uint64Type, &Type{
		TypeId: Uint64Type,
		Name:   "uint64_t",
		Ty: &Type_Pod{
			pod.Type_uint64,
		},
	}, reflect.TypeOf(uint64(0)))
	AddType(Int64Type, &Type{
		TypeId: Int64Type,
		Name:   "int64_t",
		Ty: &Type_Pod{
			pod.Type_sint64,
		},
	}, reflect.TypeOf(int64(0)))
	AddType(StringType, &Type{
		TypeId: StringType,
		Name:   "string",
		Ty: &Type_Pod{
			pod.Type_string,
		},
	}, reflect.TypeOf(string("")))

	AddType(IntType, &Type{
		TypeId: IntType,
		Name:   "int",
		Ty: &Type_Sized{
			SizedType_sized_int,
		},
	}, reflect.TypeOf(memory.Int(0)))
	AddType(UintType, &Type{
		TypeId: UintType,
		Name:   "uint_t",
		Ty: &Type_Sized{
			SizedType_sized_uint,
		},
	}, reflect.TypeOf(memory.Uint(0)))
	AddType(SizeType, &Type{
		TypeId: SizeType,
		Name:   "size_t",
		Ty: &Type_Sized{
			SizedType_sized_size,
		},
	}, reflect.TypeOf(memory.Size(0)))
	AddType(CharType, &Type{
		TypeId: CharType,
		Name:   "char_t",
		Ty: &Type_Sized{
			SizedType_sized_char,
		},
	}, reflect.TypeOf(memory.Char(0)))
}

func (t *Type) Alignment(ctx context.Context, d *device.MemoryLayout) (int, error) {
	switch t := t.Ty.(type) {
	case *Type_Pod:
		switch t.Pod {
		case pod.Type_uint:
			return int(d.Integer.Alignment), nil
		case pod.Type_sint:
			return int(d.Integer.Alignment), nil
		case pod.Type_uint8:
			return int(d.I8.Alignment), nil
		case pod.Type_sint8:
			return int(d.I8.Alignment), nil
		case pod.Type_uint16:
			return int(d.I16.Alignment), nil
		case pod.Type_sint16:
			return int(d.I16.Alignment), nil
		case pod.Type_uint32:
			return int(d.I32.Alignment), nil
		case pod.Type_sint32:
			return int(d.I32.Alignment), nil
		case pod.Type_uint64:
			return int(d.I64.Alignment), nil
		case pod.Type_sint64:
			return int(d.I64.Alignment), nil
		case pod.Type_float32:
			return int(d.F32.Alignment), nil
		case pod.Type_float64:
			return int(d.F64.Alignment), nil
		case pod.Type_bool:
			return int(d.I8.Alignment), nil
		case pod.Type_string:
			return int(d.Pointer.Alignment), nil // String is a char* in memory
		}
	case *Type_Pointer:
		return int(d.Pointer.Alignment), nil
	case *Type_Struct:
		maxAlign := 1
		for _, f := range t.Struct.Fields {
			elem, ok := TryGetType(f.Type)
			if !ok {
				return 0, log.Err(ctx, nil, "Incomplete type in struct alignment")
			}
			m, err := elem.Alignment(ctx, d)
			if err != nil {
				return 0, err
			}
			if m > maxAlign {
				maxAlign = m
			}
		}
		return maxAlign, nil
	case *Type_Sized:
		switch t.Sized {
		case SizedType_sized_int:
			return int(d.Integer.Alignment), nil
		case SizedType_sized_uint:
			return int(d.Integer.Alignment), nil
		case SizedType_sized_size:
			return int(d.Size.Alignment), nil
		case SizedType_sized_char:
			return int(d.Char.Alignment), nil
		}
	case *Type_Pseudonym:
		if elem, ok := TryGetType(t.Pseudonym.Underlying); ok {
			return elem.Alignment(ctx, d)
		}
	case *Type_Array:
		if elem, ok := TryGetType(t.Array.ElementType); ok {
			return elem.Alignment(ctx, d)
		}
	case *Type_Enum:
		if elem, ok := TryGetType(t.Enum.Underlying); ok {
			return elem.Alignment(ctx, d)
		}
	case *Type_Map:
		return 0, log.Err(ctx, nil, "Cannot decode map from memory")
	case *Type_Reference:
		return 0, log.Err(ctx, nil, "Cannot decode ref from memory")
	case *Type_Slice:
		return 0, log.Err(ctx, nil, "Cannot decode slice from memory")
	}
	return 0, log.Err(ctx, nil, fmt.Sprintf("Unhandled type alignment %T", t.Ty))
}

func (t *Type) Size(ctx context.Context, d *device.MemoryLayout) (int, error) {
	switch t := t.Ty.(type) {
	case *Type_Pod:
		switch t.Pod {
		case pod.Type_uint:
			return int(d.Integer.Size), nil
		case pod.Type_sint:
			return int(d.Integer.Size), nil
		case pod.Type_uint8:
			return int(d.I8.Size), nil
		case pod.Type_sint8:
			return int(d.I8.Size), nil
		case pod.Type_uint16:
			return int(d.I16.Size), nil
		case pod.Type_sint16:
			return int(d.I16.Size), nil
		case pod.Type_uint32:
			return int(d.I32.Size), nil
		case pod.Type_sint32:
			return int(d.I32.Size), nil
		case pod.Type_uint64:
			return int(d.I64.Size), nil
		case pod.Type_sint64:
			return int(d.I64.Size), nil
		case pod.Type_float32:
			return int(d.F32.Size), nil
		case pod.Type_float64:
			return int(d.F64.Size), nil
		case pod.Type_bool:
			return int(d.I8.Size), nil
		case pod.Type_string:
			return int(d.Pointer.Size), nil // String is a char* in memory
		}
	case *Type_Pointer:
		return int(d.Pointer.Size), nil // String is a char* in memory
	case *Type_Struct:
		size := 0
		maxAlign := 0
		for _, f := range t.Struct.Fields {
			elem, ok := TryGetType(f.Type)
			if !ok {
				return 0, log.Err(ctx, nil, "Incomplete type in struct size")
			}
			a, err := elem.Alignment(ctx, d)
			if err != nil {
				return 0, err
			}
			if a > maxAlign {
				maxAlign = a
			}

			size = sint.AlignUp(size, a)
			m, err := elem.Size(ctx, d)
			if err != nil {
				return 0, err
			}
			size += m
		}
		size = sint.AlignUp(size, maxAlign)
		return size, nil
	case *Type_Sized:
		switch t.Sized {
		case SizedType_sized_int:
			return int(d.Integer.Size), nil
		case SizedType_sized_uint:
			return int(d.Integer.Size), nil
		case SizedType_sized_size:
			return int(d.Size.Size), nil
		case SizedType_sized_char:
			return int(d.Char.Size), nil
		}
	case *Type_Pseudonym:
		if elem, ok := TryGetType(t.Pseudonym.Underlying); ok {
			return elem.Size(ctx, d)
		}
	case *Type_Array:
		if elem, ok := TryGetType(t.Array.ElementType); ok {
			sz, err := elem.Size(ctx, d)
			if err != nil {
				return 0, err
			}
			return sz * int(t.Array.Size), nil
		}
	case *Type_Enum:
		if elem, ok := TryGetType(t.Enum.Underlying); ok {
			return elem.Size(ctx, d)
		}
	case *Type_Map:
		return 0, log.Err(ctx, nil, "Cannot decode map from memory")
	case *Type_Reference:
		return 0, log.Err(ctx, nil, "Cannot decode refs from memory")
	case *Type_Slice:
		return 0, log.Err(ctx, nil, "Cannot decode slices from memory")
	}
	return 0, log.Err(ctx, nil, "Incomplete type size")
}
