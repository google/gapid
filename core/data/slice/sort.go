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

package slice

import (
	"fmt"
	"reflect"
	"sort"
)

// Sort sorts the elements of the slice s into ascending order.
// Numbers are sorted numerically.
// Booleans are sorted with false before true.
// Everything else is sorted lexicographically by first converting to string
// with Sprintf.
func Sort(s interface{}) {
	v := getSlice(s)
	switch v.Type().Elem().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		sort.Slice(s, func(i, j int) bool {
			return v.Index(i).Int() < v.Index(j).Int()
		})
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		sort.Slice(s, func(i, j int) bool {
			return v.Index(i).Uint() < v.Index(j).Uint()
		})
	case reflect.Float32, reflect.Float64:
		sort.Slice(s, func(i, j int) bool {
			return v.Index(i).Float() < v.Index(j).Float()
		})
	case reflect.Bool:
		sort.Slice(s, func(i, j int) bool {
			return !v.Index(i).Bool() && v.Index(j).Bool()
		})
	case reflect.String:
		sort.Slice(s, func(i, j int) bool {
			return v.Index(i).String() < v.Index(j).String()
		})
	default:
		sort.Slice(s, func(i, j int) bool {
			a, b := v.Index(i).Interface(), v.Index(j).Interface()
			return fmt.Sprint(a) < fmt.Sprint(b)
		})
	}
}

// SortValues sorts the reflect.Values in the slice s into ascending order.
// All the elements of e need to be of the type elTy.
// Numbers are sorted numerically.
// Booleans are sorted with false before true.
// Everything else is sorted lexicographically by first converting to string
// with Sprintf.
func SortValues(s []reflect.Value, elTy reflect.Type) {
	switch elTy.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		sort.Slice(s, func(i, j int) bool {
			return s[i].Int() < s[j].Int()
		})
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		sort.Slice(s, func(i, j int) bool {
			return s[i].Uint() < s[j].Uint()
		})
	case reflect.Float32, reflect.Float64:
		sort.Slice(s, func(i, j int) bool {
			return s[i].Float() < s[j].Float()
		})
	case reflect.Bool:
		sort.Slice(s, func(i, j int) bool {
			return !s[i].Bool() && s[j].Bool()
		})
	case reflect.String:
		sort.Slice(s, func(i, j int) bool {
			return s[i].String() < s[j].String()
		})
	default:
		sort.Slice(s, func(i, j int) bool {
			a, b := s[i].Interface(), s[j].Interface()
			return fmt.Sprint(a) < fmt.Sprint(b)
		})
	}
}
