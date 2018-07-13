// Copyright (C) 2018 Google Inc.
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

// OnMap is the result of calling ThatMap on an Assertion.
// It provides assertion tests that are specific to map types.
type OnMap struct {
	Assertion
	mp interface{}
}

// ThatMap returns an OnMap for assertions on map type objects.
// Calling this with a non map type will result in panics.
func (a Assertion) ThatMap(mp interface{}) OnMap {
	return OnMap{Assertion: a, mp: mp}
}

// IsEmpty asserts that the map was of length 0
func (o OnMap) IsEmpty() bool {
	value := reflect.ValueOf(o.mp)
	return o.CompareRaw(value.Len(), "is", "empty").Test(value.Len() == 0)
}

// IsNotEmpty asserts that the map has elements
func (o OnMap) IsNotEmpty() bool {
	value := reflect.ValueOf(o.mp)
	return o.Compare(value.Len(), "length >", 0).Test(value.Len() > 0)
}

// IsLength asserts that the map has exactly the specified number of elements
func (o OnMap) IsLength(length int) bool {
	value := reflect.ValueOf(o.mp)
	return o.Compare(value.Len(), "length ==", length).Test(value.Len() == length)
}

// Equals asserts the array or map matches expected.
func (o OnMap) Equals(expected interface{}) bool {
	return o.mapsEqual(expected, func(a, b interface{}) bool { return a == b })
}

// EqualsWithComparator asserts the array or map matches expected using a comparator function
func (o OnMap) EqualsWithComparator(expected interface{}, same func(a, b interface{}) bool) bool {
	return o.mapsEqual(expected, same)
}

// DeepEquals asserts the array or map matches expected using a deep-equal comparison.
func (o OnMap) DeepEquals(expected interface{}) bool {
	return o.mapsEqual(expected, compare.DeepEqual)
}

func (o OnMap) mapsEqual(expected interface{}, same func(a, b interface{}) bool) bool {
	return o.Test(func() bool {
		gs := reflect.ValueOf(o.mp)
		es := reflect.ValueOf(expected)
		equal := true
		for _, k := range gs.MapKeys() {
			gv := gs.MapIndex(k)
			ev := es.MapIndex(k)
			if !ev.IsValid() {
				o.Printf("\tExtra key: %#v\n", k.Interface())
				equal = false
				continue
			}
			if !same(gv.Interface(), ev.Interface()) {
				o.Printf("\tKey: %#v, %#v differs from expected: %#v\n", k.Interface(), gv.Interface(), ev.Interface())
				equal = false
			}
		}
		for _, k := range es.MapKeys() {
			if !gs.MapIndex(k).IsValid() {
				o.Printf("\tKey missing: %#v\n", k.Interface())
				equal = false
			}
		}
		return equal
	}())
}
