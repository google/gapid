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

package template

import (
	"hash/fnv"
	"reflect"
	"sort"
)

// Join returns the concatenation of all the passed array or slices into a
// single, sequential list.
func (*Functions) Join(slices ...interface{}) []interface{} {
	out := []interface{}{}
	for _, slice := range slices {
		v := reflect.ValueOf(slice)
		for i, c := 0, v.Len(); i < c; i++ {
			out = append(out, v.Index(i).Interface())
		}
	}
	return out
}

// HasMore returns true if the i'th indexed item in l is not the last.
func (*Functions) HasMore(i int, l interface{}) bool {
	return i < reflect.ValueOf(l).Len()-1
}

// Reverse returns a new list with all the elements of in reversed.
func (*Functions) Reverse(in interface{}) interface{} {
	v := reflect.ValueOf(in)
	c := v.Len()
	out := reflect.MakeSlice(v.Type(), c, c)
	for i := 0; i < c; i++ {
		out.Index(i).Set(v.Index(c - i - 1))
	}
	return out.Interface()
}

// IndexOf returns the index of value in the list array. If value is not found
// in array then IndexOf returns -1.
func (*Functions) IndexOf(array interface{}, value interface{}) int {
	a := reflect.ValueOf(array)
	for i, c := 0, a.Len(); i < c; i++ {
		if a.Index(i).Interface() == value {
			return i
		}
	}
	return -1
}

// Tail returns a slice of the list from start to len(array).
func (*Functions) Tail(start int, array interface{}) interface{} {
	v := reflect.ValueOf(array)
	c := v.Len() - start
	out := reflect.MakeSlice(v.Type(), c, c)
	for i := 0; i < c; i++ {
		out.Index(i).Set(v.Index(start + i))
	}
	return out.Interface()
}

// ForEach returns a string list containing the strings emitted by calling the
// macro m for each sequential item in the array arr. 0 length strings will
// be ommitted from the returned list.
func (f *Functions) ForEach(arr interface{}, m string) (stringList, error) {
	v := reflect.ValueOf(arr)
	c := v.Len()
	l := make(stringList, 0, c)
	for i := 0; i < c; i++ {
		e := v.Index(i).Interface()
		str, err := f.Macro(m, e)
		if err != nil {
			return nil, err
		}
		if len(str) > 0 {
			l = append(l, str)
		}
	}
	return l, nil
}

type sortElem struct {
	key string
	idx int
}

type sortElemList []sortElem

func (l sortElemList) Len() int           { return len(l) }
func (l sortElemList) Less(a, b int) bool { return l[a].key < l[b].key }
func (l sortElemList) Swap(a, b int)      { l[a], l[b] = l[b], l[a] }

// SortBy returns a new list containing the sorted elements of arr. The list is
// sorted by the string values returned by calling the template function
// keyMacro with each element in the list.
func (f *Functions) SortBy(arr interface{}, keyMacro string) (interface{}, error) {
	v := reflect.ValueOf(arr)
	c := v.Len()

	elems := make(sortElemList, c)
	for i := 0; i < c; i++ {
		e := v.Index(i).Interface()
		key, err := f.Macro(keyMacro, e)
		if err != nil {
			return nil, err
		}
		elems[i] = sortElem{key, i}
	}

	sort.Sort(elems)

	out := reflect.MakeSlice(v.Type(), c, c)
	for i, e := range elems {
		out.Index(i).Set(v.Index(e.idx))
	}
	return out.Interface(), nil
}

// Divides a slice into num buckets and returns a slice of new slices.
// keyMacro is the name of a macro which should generate a stable key for
// the object in the slice.
func (f *Functions) Partition(arr interface{}, keyMacro string, num int) (interface{}, error) {
	arrV := reflect.ValueOf(arr)
	n := arrV.Len()

	// Make the slice of slices
	sliceType := reflect.SliceOf(arrV.Type())
	slice := reflect.MakeSlice(sliceType, num, num)

	for i := 0; i < n; i++ {
		v := arrV.Index(i)
		key, err := f.Macro(keyMacro, v.Interface())
		if err != nil {
			return nil, err
		}
		hasher := fnv.New32()
		hasher.Write([]byte(key))
		hash := hasher.Sum32()
		bucketNum := int(hash % uint32(num))
		bucket := slice.Index(bucketNum)
		bucket.Set(reflect.Append(bucket, v))
	}

	return slice.Interface(), nil
}

// PartitionByKey Divides a slice into one bucket for each different keyMacro returned.
// keyMacro is the name of a macro which should generate a stable key for
// the object in the slice.
func (f *Functions) PartitionByKey(arr interface{}, keyMacro string) (interface{}, error) {
	arrV := reflect.ValueOf(arr)
	// Bucket is a slice of the arr element type. ([]el)
	bucketType := reflect.SliceOf(arrV.Type().Elem())
	// Slices is a slice of buckets. ([][]el)
	slicesType := reflect.SliceOf(bucketType)
	slices := reflect.MakeSlice(slicesType, 0, 8)
	bucketIndices := make(map[string]int)
	for i, c := 0, arrV.Len(); i < c; i++ {
		v := arrV.Index(i)
		key, err := f.Macro(keyMacro, v.Interface())
		if err != nil {
			return nil, err
		}
		bucketIdx, ok := bucketIndices[key]
		if !ok {
			bucketIdx = len(bucketIndices)
			bucketIndices[key] = bucketIdx
			slices = reflect.Append(slices, reflect.MakeSlice(bucketType, 0, 8))
		}
		bucket := slices.Index(bucketIdx)
		bucket.Set(reflect.Append(bucket, v))

	}
	return slices.Interface(), nil
}

func (f *Functions) Length(arr interface{}) (int, error) {
	v := reflect.ValueOf(arr)
	return v.Len(), nil
}
