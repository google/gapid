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

package jdbg

import (
	"fmt"
	"reflect"
)

// assignable returns true if the type of val can be assigned to dst.
func (j *JDbg) assignable(dst Type, val interface{}) bool {
	if v, ok := val.(Value); ok {
		if v.ty.CastableTo(dst) {
			return true
		}
		if dst.Signature() == v.ty.Signature() {
			// Sanity check.
			panic(fmt.Sprint("Two different types found with identical signatures! ", dst.Signature()))
		}
		return j.assignable(dst, v.val)
	}
	return j.assignableT(dst, reflect.TypeOf(val))
}

func (j *JDbg) assignableT(dst Type, src reflect.Type) bool {
	if dst == j.cache.objTy {
		return true // Anything goes in an object!
	}
	if src == nil {
		_, isClass := dst.(*Class)
		return isClass
	}
	switch src.Kind() {
	case reflect.Ptr, reflect.Interface:
		return j.assignableT(dst, src.Elem())
	case reflect.Bool:
		return dst == j.cache.boolTy
	case reflect.Int8, reflect.Uint8:
		return dst == j.cache.byteTy
	case reflect.Int16:
		return dst == j.cache.charTy || dst == j.cache.shortTy
	case reflect.Int32, reflect.Int:
		return dst == j.cache.intTy
	case reflect.Int64:
		return dst == j.cache.longTy
	case reflect.Float32:
		return dst == j.cache.floatTy
	case reflect.Float64:
		return dst == j.cache.doubleTy
	case reflect.String:
		return dst == j.cache.stringTy
	case reflect.Array, reflect.Slice:
		if dstArray, ok := dst.(*Array); ok {
			return j.assignableT(dstArray.el, src.Elem())
		}
	}

	return false
}
