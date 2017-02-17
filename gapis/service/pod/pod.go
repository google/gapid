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

// NewValue attempts to box and return v into a Value.
// If v cannot be boxed into a Value then nil is returned.
func NewValue(v interface{}) *Value {
	switch v := v.(type) {
	case float32:
		return &Value{&Value_Float{v}}
	case float64:
		return &Value{&Value_Double{v}}
	case uint:
		return &Value{&Value_Uint{uint64(v)}}
	case int:
		return &Value{&Value_Sint{int64(v)}}
	case uint8:
		return &Value{&Value_Uint8{uint32(v)}}
	case int8:
		return &Value{&Value_Sint8{int32(v)}}
	case uint16:
		return &Value{&Value_Uint16{uint32(v)}}
	case int16:
		return &Value{&Value_Sint16{int32(v)}}
	case uint32:
		return &Value{&Value_Uint32{v}}
	case int32:
		return &Value{&Value_Sint32{v}}
	case uint64:
		return &Value{&Value_Uint64{v}}
	case int64:
		return &Value{&Value_Sint64{v}}
	case bool:
		return &Value{&Value_Bool{v}}
	case string:
		return &Value{&Value_String_{v}}
	case []float32:
		return &Value{&Value_FloatArray{&FloatArray{v}}}
	case []float64:
		return &Value{&Value_DoubleArray{&DoubleArray{v}}}
	case []uint:
		q := make([]uint64, len(v))
		for i, v := range v {
			q[i] = uint64(v)
		}
		return &Value{&Value_Uint64Array{&Uint64Array{q}}}
	case []int:
		q := make([]int64, len(v))
		for i, v := range v {
			q[i] = int64(v)
		}
		return &Value{&Value_Sint64Array{&Sint64Array{q}}}
	case []byte:
		return &Value{&Value_Uint8Array{v}}
	case []int8:
		q := make([]int32, len(v))
		for i, v := range v {
			q[i] = int32(v)
		}
		return &Value{&Value_Sint32Array{&Sint32Array{q}}}
	case []uint16:
		q := make([]uint32, len(v))
		for i, v := range v {
			q[i] = uint32(v)
		}
		return &Value{&Value_Uint32Array{&Uint32Array{q}}}
	case []int16:
		q := make([]int32, len(v))
		for i, v := range v {
			q[i] = int32(v)
		}
		return &Value{&Value_Sint32Array{&Sint32Array{q}}}
	case []uint32:
		return &Value{&Value_Uint32Array{&Uint32Array{v}}}
	case []int32:
		return &Value{&Value_Sint32Array{&Sint32Array{v}}}
	case []uint64:
		return &Value{&Value_Uint64Array{&Uint64Array{v}}}
	case []int64:
		return &Value{&Value_Sint64Array{&Sint64Array{v}}}
	case []bool:
		return &Value{&Value_BoolArray{&BoolArray{v}}}
	case []string:
		return &Value{&Value_StringArray{&StringArray{v}}}
	}
	return nil
}

// Get returns the boxed error.
func (v *Value) Get() interface{} {
	switch v := v.Val.(type) {
	case *Value_Float:
		return v.Float
	case *Value_Double:
		return v.Double
	case *Value_Uint:
		return v.Uint
	case *Value_Sint:
		return v.Sint
	case *Value_Uint8:
		return v.Uint8
	case *Value_Sint8:
		return v.Sint8
	case *Value_Uint16:
		return v.Uint16
	case *Value_Sint16:
		return v.Sint16
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
	case *Value_FloatArray:
		return v.FloatArray.Val
	case *Value_DoubleArray:
		return v.DoubleArray.Val
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
