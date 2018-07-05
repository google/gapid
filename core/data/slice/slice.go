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

import (
	"fmt"
	"reflect"
	"unsafe"
)

// New returns a new addressable slice value of type ty.
func New(ty reflect.Type, len, cap int) reflect.Value {
	ptr := reflect.New(ty)
	ptr.Elem().Set(reflect.MakeSlice(ty, len, cap))
	return ptr.Elem()
}

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
	out := New(slice.Type(), 0, slice.Len())
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

// RemoveAt removes n elements from s starting at i.
// s must be a pointer to a slice.
func RemoveAt(s interface{}, i, n int) {
	ptr, slice := getSlicePtr(s)
	out := New(slice.Type(), 0, slice.Len()-n)
	//  aaaaa XXXXXX bbbbbb
	// 0     i      j      e
	e := slice.Len()
	j := i + n
	out = reflect.AppendSlice(out, slice.Slice(0, i))
	out = reflect.AppendSlice(out, slice.Slice(j, e))
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

// Clone returns a shallow copy of the slice s.
func Clone(s interface{}) interface{} {
	in := getSlice(s)
	out := New(in.Type(), in.Len(), in.Len())
	reflect.Copy(out, in)
	return out.Interface()
}

// Reverse swaps the order of elements in the slice so that the first become the
// last, and so on.
func Reverse(s interface{}) {
	slice := getSlice(s)
	tmp := reflect.New(slice.Type().Elem()).Elem()
	for i, c, m := 0, slice.Len(), slice.Len()/2; i < m; i++ {
		j := c - i - 1
		a, b := slice.Index(i), slice.Index(j)
		tmp.Set(a)
		a.Set(b)
		b.Set(tmp)
	}
}

// Bytes returns a byte-slice from the given unsafe pointer of the given
// size in bytes.
func Bytes(ptr unsafe.Pointer, size uint64) []byte {
	return ((*[1 << 30]byte)(ptr))[:size]
}

// Cast performs an unsafe cast of the slice at sli to the given slice type.
// Cast uses internal casting which may completely break in future releases of
// the Golang language / compiler.
func Cast(sli interface{}, to reflect.Type) interface{} {
	s, t := reflect.ValueOf(sli), reflect.TypeOf(sli)
	in := reflect.New(t)
	in.Elem().Set(s)
	out := reflect.New(to)
	src := (*reflect.SliceHeader)((unsafe.Pointer)(in.Pointer()))
	dst := (*reflect.SliceHeader)((unsafe.Pointer)(out.Pointer()))
	lenBytes := uintptr(src.Len) * t.Elem().Size()
	capBytes := uintptr(src.Cap) * t.Elem().Size()
	dst.Data = src.Data
	dst.Len = int(lenBytes / to.Elem().Size())
	dst.Cap = int(capBytes / to.Elem().Size())
	return out.Elem().Interface()
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
		panic(fmt.Errorf("s must be a pointer to a slice, got: %T", s))
	}
	slice = ptr.Elem()
	if slice.Kind() != reflect.Slice {
		panic(fmt.Errorf("s must be a pointer to a slice, got: %T", s))
	}
	return ptr, slice
}

// getSlice checks s is a slice, returning the slice as a reflect.Value.
func getSlice(s interface{}) (slice reflect.Value) {
	slice = reflect.ValueOf(s)
	if slice.Kind() != reflect.Slice {
		panic(fmt.Errorf("s must be a slice, got: %T", s))
	}
	return slice
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
