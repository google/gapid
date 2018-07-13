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

import "reflect"

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
	return o.mapsEqual(expected, reflect.DeepEqual)
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func (o OnMap) mapsEqual(expected interface{}, same func(a, b interface{}) bool) bool {
	return o.Test(func() bool {
		gs := reflect.ValueOf(o.mp)
		es := reflect.ValueOf(expected)
		if gs.Len() < es.Len() {
			o.Printf("\tShorter\tby\t%v\tkeys\n", es.Len()-gs.Len())
			return false
		}
		equal := true
		for _, k := range gs.MapKeys() {
			gv := gs.MapIndex(k)
			ev := es.MapIndex(k)
			if ev == reflect.ValueOf(nil) {
				o.Printf("\tKey\tmissing:\t%#v\n", k.Interface())
				equal = false
				continue
			}
			if !same(gv.Interface(), ev.Interface()) {
				o.Printf("\tKey:\t%#v,\t%#v\tdiffers\tfrom\texpected:\t%#v\n", k.Interface(), gv.Interface(), ev.Interface())
				equal = false
			}
		}
		return equal
	}())
}
