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

package assert

import (
	"reflect"

	"github.com/google/gapid/core/data/compare"
)

// OnSlice is the result of calling ThatSlice on an Assertion.
// It provides assertion tests that are specific to slice types.
type OnSlice struct {
	Assertion
	slice interface{}
}

// ThatSlice returns an OnSlice for assertions on slice type objects.
// Calling this with a non slice type will result in panics.
func (a Assertion) ThatSlice(slice interface{}) OnSlice {
	return OnSlice{Assertion: a, slice: slice}
}

// IsEmpty asserts that the slice was of length 0
func (o OnSlice) IsEmpty() bool {
	value := reflect.ValueOf(o.slice)
	return o.CompareRaw(value.Len(), "is", "empty").Test(value.Len() == 0)
}

// IsNotEmpty asserts that the slice has elements
func (o OnSlice) IsNotEmpty() bool {
	value := reflect.ValueOf(o.slice)
	return o.Compare(value.Len(), "length >", 0).Test(value.Len() > 0)
}

// IsLength asserts that the slice has exactly the specified number of elements
func (o OnSlice) IsLength(length int) bool {
	value := reflect.ValueOf(o.slice)
	return o.Compare(value.Len(), "length ==", length).Test(value.Len() == length)
}

// Equals asserts the array or slice matches expected.
func (o OnSlice) Equals(expected interface{}) bool {
	return o.slicesEqual(expected, func(a, b interface{}) bool { return a == b })
}

// EqualsWithComparator asserts the array or slice matches expected using a comparator function
func (o OnSlice) EqualsWithComparator(expected interface{}, same func(a, b interface{}) bool) bool {
	return o.slicesEqual(expected, same)
}

// DeepEquals asserts the array or slice matches expected using a deep-equal comparison.
func (o OnSlice) DeepEquals(expected interface{}) bool {
	return o.slicesEqual(expected, compare.DeepEqual)
}

func (o OnSlice) slicesEqual(expected interface{}, same func(a, b interface{}) bool) bool {
	return o.Test(func() bool {
		gs := reflect.ValueOf(o.slice)
		glen := gs.Len()
		es := reflect.ValueOf(expected)
		elen := es.Len()
		max := glen
		if max < elen {
			max = elen
		}
		equal := true
		for i := 0; i < max; i++ {
			switch {
			case i >= glen:
				// expected but not present
				ev := es.Index(i)
				o.Printf("-\t%d\t\t", i)
				o.Printf("\t==>\t%T\t", ev.Interface())
				o.Println(ev.Interface())
				equal = false
			case i >= elen:
				// present but not expected
				gv := gs.Index(i)
				o.Printf("+\t%d\t%T\t", i, gv.Interface())
				o.Print(gv.Interface())
				o.Rawln("\t;")
				equal = false
			default:
				// in both
				ev := es.Index(i)
				gv := gs.Index(i)
				if same(gv.Interface(), ev.Interface()) {
					o.Printf("\t%d\t%T\t", i, gv.Interface())
					o.Print(gv.Interface())
					o.Rawln("\t;")
				} else {
					ev := es.Index(i)
					o.Printf("*\t%d\t%T\t", i, gv.Interface())
					o.Print(gv.Interface())
					o.Printf("\t==>\t%T\t", ev.Interface())
					o.Println(ev.Interface())
					equal = false
				}
			}
		}
		return equal
	}())
}
