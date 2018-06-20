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

package pod

import "reflect"

// NewValue attempts to box and return val into a Value.
// If val cannot be boxed into a Value then nil is returned.
func NewValue(val interface{}) *Value {
	if val == nil {
		return nil
	}
	v := reflect.ValueOf(val)
	t := v.Type()
	switch t.Kind() {
	case reflect.Bool:
		return &Value{Val: &Value_Bool{v.Bool()}}
	case reflect.String:
		return &Value{Val: &Value_String_{v.String()}}
	case reflect.Float32:
		return &Value{Val: &Value_Float32{float32(v.Float())}}
	case reflect.Float64:
		return &Value{Val: &Value_Float64{v.Float()}}
	case reflect.Int:
		return &Value{Val: &Value_Sint{v.Int()}}
	case reflect.Int8:
		return &Value{Val: &Value_Sint8{(int32)(v.Int())}}
	case reflect.Int16:
		return &Value{Val: &Value_Sint16{(int32)(v.Int())}}
	case reflect.Int32:
		return &Value{Val: &Value_Sint32{(int32)(v.Int())}}
	case reflect.Int64:
		return &Value{Val: &Value_Sint64{v.Int()}}
	case reflect.Uint:
		return &Value{Val: &Value_Uint{v.Uint()}}
	case reflect.Uint8:
		return &Value{Val: &Value_Uint8{(uint32)(v.Uint())}}
	case reflect.Uint16:
		return &Value{Val: &Value_Uint16{(uint32)(v.Uint())}}
	case reflect.Uint32:
		return &Value{Val: &Value_Uint32{(uint32)(v.Uint())}}
	case reflect.Uint64:
		return &Value{Val: &Value_Uint64{v.Uint()}}

	case reflect.Slice, reflect.Array:
		switch t.Elem().Kind() {
		case reflect.Bool:
			arr := make([]bool, v.Len())
			for i := range arr {
				arr[i] = v.Index(i).Bool()
			}
			return &Value{Val: &Value_BoolArray{&BoolArray{Val: arr}}}
		case reflect.String:
			arr := make([]string, v.Len())
			for i := range arr {
				arr[i] = v.Index(i).String()
			}
			return &Value{Val: &Value_StringArray{&StringArray{Val: arr}}}
		case reflect.Float32:
			arr := make([]float32, v.Len())
			for i := range arr {
				arr[i] = (float32)(v.Index(i).Float())
			}
			return &Value{Val: &Value_Float32Array{&Float32Array{Val: arr}}}
		case reflect.Float64:
			arr := make([]float64, v.Len())
			for i := range arr {
				arr[i] = v.Index(i).Float()
			}
			return &Value{Val: &Value_Float64Array{&Float64Array{Val: arr}}}
		case reflect.Int:
			arr := make([]int64, v.Len())
			for i := range arr {
				arr[i] = v.Index(i).Int()
			}
			return &Value{Val: &Value_SintArray{&Sint64Array{Val: arr}}}
		case reflect.Int8:
			arr := make([]int32, v.Len())
			for i := range arr {
				arr[i] = (int32)(v.Index(i).Int())
			}
			return &Value{Val: &Value_Sint8Array{&Sint32Array{Val: arr}}}
		case reflect.Int16:
			arr := make([]int32, v.Len())
			for i := range arr {
				arr[i] = (int32)(v.Index(i).Int())
			}
			return &Value{Val: &Value_Sint16Array{&Sint32Array{Val: arr}}}
		case reflect.Int32:
			arr := make([]int32, v.Len())
			for i := range arr {
				arr[i] = (int32)(v.Index(i).Int())
			}
			return &Value{Val: &Value_Sint32Array{&Sint32Array{Val: arr}}}
		case reflect.Int64:
			arr := make([]int64, v.Len())
			for i := range arr {
				arr[i] = v.Index(i).Int()
			}
			return &Value{Val: &Value_Sint64Array{&Sint64Array{Val: arr}}}
		case reflect.Uint:
			arr := make([]uint64, v.Len())
			for i := range arr {
				arr[i] = v.Index(i).Uint()
			}
			return &Value{Val: &Value_UintArray{&Uint64Array{Val: arr}}}
		case reflect.Uint8:
			arr := make([]byte, v.Len())
			for i := range arr {
				arr[i] = (byte)(v.Index(i).Uint())
			}
			return &Value{Val: &Value_Uint8Array{Uint8Array: arr}}
		case reflect.Uint16:
			arr := make([]uint32, v.Len())
			for i := range arr {
				arr[i] = (uint32)(v.Index(i).Uint())
			}
			return &Value{Val: &Value_Uint16Array{&Uint32Array{Val: arr}}}
		case reflect.Uint32:
			arr := make([]uint32, v.Len())
			for i := range arr {
				arr[i] = (uint32)(v.Index(i).Uint())
			}
			return &Value{Val: &Value_Uint32Array{&Uint32Array{Val: arr}}}
		case reflect.Uint64:
			arr := make([]uint64, v.Len())
			for i := range arr {
				arr[i] = v.Index(i).Uint()
			}
			return &Value{Val: &Value_Uint64Array{&Uint64Array{Val: arr}}}
		}
	}
	return nil
}

// Get returns the boxed Basic value.
func (v *Value) Get() interface{} {
	switch v := v.Val.(type) {
	case *Value_Float32:
		return v.Float32
	case *Value_Float64:
		return v.Float64
	case *Value_Uint:
		return (uint)(v.Uint)
	case *Value_Sint:
		return (int)(v.Sint)
	case *Value_Uint8:
		return (uint8)(v.Uint8)
	case *Value_Sint8:
		return (int8)(v.Sint8)
	case *Value_Uint16:
		return (uint16)(v.Uint16)
	case *Value_Sint16:
		return (int16)(v.Sint16)
	case *Value_Uint32:
		return v.Uint32
	case *Value_Sint32:
		return v.Sint32
	case *Value_Uint64:
		return v.Uint64
	case *Value_Sint64:
		return v.Sint64
	case *Value_Bool:
		return v.Bool
	case *Value_String_:
		return v.String_
	case *Value_Float32Array:
		return v.Float32Array.Val
	case *Value_Float64Array:
		return v.Float64Array.Val
	case *Value_UintArray:
		o := make([]uint, len(v.UintArray.Val))
		for i, v := range v.UintArray.Val {
			o[i] = uint(v)
		}
		return o
	case *Value_SintArray:
		o := make([]int, len(v.SintArray.Val))
		for i, v := range v.SintArray.Val {
			o[i] = int(v)
		}
		return o
	case *Value_Uint8Array:
		return v.Uint8Array
	case *Value_Sint8Array:
		o := make([]int8, len(v.Sint8Array.Val))
		for i, v := range v.Sint8Array.Val {
			o[i] = int8(v)
		}
		return o
	case *Value_Uint16Array:
		o := make([]uint16, len(v.Uint16Array.Val))
		for i, v := range v.Uint16Array.Val {
			o[i] = uint16(v)
		}
		return o
	case *Value_Sint16Array:
		o := make([]int16, len(v.Sint16Array.Val))
		for i, v := range v.Sint16Array.Val {
			o[i] = int16(v)
		}
		return o
	case *Value_Uint32Array:
		return v.Uint32Array.Val
	case *Value_Sint32Array:
		return v.Sint32Array.Val
	case *Value_Uint64Array:
		return v.Uint64Array.Val
	case *Value_Sint64Array:
		return v.Sint64Array.Val
	case *Value_BoolArray:
		return v.BoolArray.Val
	case *Value_StringArray:
		return v.StringArray.Val
	}
	return nil
}

// TypeOf returns the POD type of ty.
func TypeOf(ty reflect.Type) (Type, bool) {
	switch ty.Kind() {
	case reflect.Bool:
		return Type_bool, true
	case reflect.String:
		return Type_string, true
	case reflect.Float32:
		return Type_float32, true
	case reflect.Float64:
		return Type_float64, true
	case reflect.Int:
		return Type_sint, true
	case reflect.Int8:
		return Type_sint8, true
	case reflect.Int16:
		return Type_sint16, true
	case reflect.Int32:
		return Type_sint32, true
	case reflect.Int64:
		return Type_sint64, true
	case reflect.Uint:
		return Type_uint, true
	case reflect.Uint8:
		return Type_uint8, true
	case reflect.Uint16:
		return Type_uint16, true
	case reflect.Uint32:
		return Type_uint32, true
	case reflect.Uint64:
		return Type_uint64, true

	case reflect.Slice, reflect.Array:
		switch ty.Elem().Kind() {
		case reflect.Bool:
			return Type_bool_array, true
		case reflect.String:
			return Type_string_array, true
		case reflect.Float32:
			return Type_float_array, true
		case reflect.Float64:
			return Type_double_array, true
		case reflect.Int:
			return Type_sint_array, true
		case reflect.Int8:
			return Type_sint8_array, true
		case reflect.Int16:
			return Type_sint16_array, true
		case reflect.Int32:
			return Type_sint32_array, true
		case reflect.Int64:
			return Type_sint64_array, true
		case reflect.Uint:
			return Type_uint_array, true
		case reflect.Uint8:
			return Type_uint8_array, true
		case reflect.Uint16:
			return Type_uint16_array, true
		case reflect.Uint32:
			return Type_uint32_array, true
		case reflect.Uint64:
			return Type_uint64_array, true
		}
	}
	return 0, false
}

// Get returns the POD type.
func (t Type) Get() reflect.Type {
	switch t {
	case Type_float32:
		return reflect.TypeOf(float32(0))
	case Type_float64:
		return reflect.TypeOf(float64(0))
	case Type_uint:
		return reflect.TypeOf(uint(0))
	case Type_sint:
		return reflect.TypeOf(int(0))
	case Type_uint8:
		return reflect.TypeOf(uint8(0))
	case Type_sint8:
		return reflect.TypeOf(int8(0))
	case Type_uint16:
		return reflect.TypeOf(uint16(0))
	case Type_sint16:
		return reflect.TypeOf(int16(0))
	case Type_uint32:
		return reflect.TypeOf(uint32(0))
	case Type_sint32:
		return reflect.TypeOf(int32(0))
	case Type_uint64:
		return reflect.TypeOf(uint64(0))
	case Type_sint64:
		return reflect.TypeOf(int64(0))
	case Type_bool:
		return reflect.TypeOf(false)
	case Type_string:
		return reflect.TypeOf("")
	case Type_float_array:
		return reflect.TypeOf([]float32{})
	case Type_double_array:
		return reflect.TypeOf([]float64{})
	case Type_uint_array:
		return reflect.TypeOf([]uint{})
	case Type_sint_array:
		return reflect.TypeOf([]int{})
	case Type_uint8_array:
		return reflect.TypeOf([]uint8{})
	case Type_sint8_array:
		return reflect.TypeOf([]int8{})
	case Type_uint16_array:
		return reflect.TypeOf([]uint16{})
	case Type_sint16_array:
		return reflect.TypeOf([]int16{})
	case Type_uint32_array:
		return reflect.TypeOf([]uint32{})
	case Type_sint32_array:
		return reflect.TypeOf([]int32{})
	case Type_uint64_array:
		return reflect.TypeOf([]uint64{})
	case Type_sint64_array:
		return reflect.TypeOf([]int64{})
	case Type_bool_array:
		return reflect.TypeOf([]bool{})
	case Type_string_array:
		return reflect.TypeOf([]string{})
	}
	return nil
}
