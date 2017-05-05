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

// Package slice provides utilities for mutating generic slices.
package slice

import "reflect"

// Replace replaces count elements of s starting from first with with.
// s must be a pointer to a slice.
func Replace(s interface{}, first, count int, with interface{}) {
	ptr, old := getSlicePtr(s)
	replace(ptr, old, first, count, with)
}

// Remove removes all occurances of v from the slice s.
// s must be a pointer to a slice.
func Remove(s, v interface{}) {
	ptr, slice := getSlicePtr(s)
	out := newSlice(slice.Type(), 0, slice.Len())
	for i, c := 0, slice.Len(); i < c; i++ {
		el := slice.Index(i)
		if el.Interface() != v {
			l := out.Len()
			out.SetLen(l + 1)
			out.Index(l).Set(el)
		}
	}
	ptr.Elem().Set(out)
}

// InsertBefore inserts v before the i'th element in s.
// s must be a pointer to a slice.
// v can be a slice or a single element.
func InsertBefore(s interface{}, i int, v interface{}) {
	Replace(s, i, 0, v)
}

// Append appends s with v.
// s must be a pointer to a slice.
// v can be a slice or a single element.
func Append(s interface{}, v interface{}) {
	ptr, old := getSlicePtr(s)
	replace(ptr, old, old.Len(), 0, v)
}

func replace(ptr, old reflect.Value, first, count int, with interface{}) {
	d := toSlice(with, old.Type())
	newLen, oldLen := old.Len()-count+d.Len(), old.Len()
	new := old
	// ensure the slice is big enough to hold all the new data.
	if old.Cap() < newLen {
		// l isn't big enough to hold the new elements.
		// Create a new buffer, with room to grow.
		new = reflect.MakeSlice(old.Type(), newLen, newLen*2)
		reflect.Copy(new, old.Slice(0, first))
	} else {
		new.SetLen(newLen) // Extend the slice length.
	}
	// move the part after the insertion to the right by len(with).
	reflect.Copy(
		new.Slice(first+d.Len(), new.Len()),
		old.Slice(first+count, oldLen),
	)
	// copy in with.
	reflect.Copy(new.Slice(first, new.Len()), d)
	ptr.Elem().Set(new)
}

// getSlicePtr checks s is a pointer to a slice, returning both the pointer
// and slice as a reflect.Value.
func getSlicePtr(s interface{}) (ptr, slice reflect.Value) {
	ptr = reflect.ValueOf(s)
	if ptr.Kind() != reflect.Ptr {
		panic("s must be a pointer to a slice")
	}
	slice = ptr.Elem()
	if slice.Kind() != reflect.Slice {
		panic("s must be a pointer to a slice")
	}
	return ptr, slice
}

// newSlice returns a new addressable slice value.
func newSlice(ty reflect.Type, len, cap int) reflect.Value {
	ptr := reflect.New(ty)
	ptr.Elem().Set(reflect.MakeSlice(ty, len, cap))
	return ptr.Elem()
}

// toSlice returns the reflect.Value of v, turning it into a single element
// slice (of type sliceTy) if it isn't a slice already.
func toSlice(v interface{}, sliceTy reflect.Type) reflect.Value {
	e := reflect.ValueOf(v)
	if e.Kind() == reflect.Slice {
		return e
	}
	s := reflect.MakeSlice(sliceTy, 1, 1)
	s.Index(0).Set(e)
	return s
}
