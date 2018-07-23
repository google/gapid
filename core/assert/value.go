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

// OnValue is the result of calling That on an Assertion.
// It provides generice assertion tests that work for any type.
type OnValue struct {
	Assertion
	value interface{}
}

// That returns an OnValue for the specified untyped value.
func (a Assertion) That(value interface{}) OnValue {
	return OnValue{Assertion: a, value: value}
}

func isNil(value interface{}) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Map,
		reflect.Ptr,
		reflect.Interface,
		reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// IsNil asserts that the supplied value was a nil.
// Typed nils are also be allowed.
func (o OnValue) IsNil() bool {
	return o.Compare(o.value, "==", "nil").Test(isNil(o.value))
}

// IsNotNil asserts that the supplied value was not a nil.
// Typed nils will also fail.
func (o OnValue) IsNotNil() bool {
	return o.Compare(o.value, "!=", "nil").Test(!isNil(o.value))
}

// Equals asserts that the supplied value is equal to the expected value.
func (o OnValue) Equals(expect interface{}) bool {
	return o.Compare(o.value, "==", expect).Test(o.value == expect)
}

// NotEquals asserts that the supplied value is not equal to the test value.
func (o OnValue) NotEquals(test interface{}) bool {
	return o.Compare(o.value, "!=", test).Test(o.value != test)
}

// CustomDeepEquals asserts that the supplied value is equal to the expected
// value using compare.Diff and the custom comparators.
func (o OnValue) CustomDeepEquals(expect interface{}, c compare.Custom) bool {
	return o.TestCustomDeepDiff(o.value, expect, c)
}

// DeepEquals asserts that the supplied value is equal to the expected value using compare.Diff.
func (o OnValue) DeepEquals(expect interface{}) bool {
	return o.TestDeepDiff(o.value, expect)
}

// DeepNotEquals asserts that the supplied value is not equal to the test value using a deep comparison.
func (o OnValue) DeepNotEquals(test interface{}) bool {
	return o.TestDeepNotEqual(o.value, test)
}

// Implements asserts that the supplied value implements the specified interface.
func (o OnValue) Implements(iface reflect.Type) bool {
	t := reflect.TypeOf(o.value)
	return o.Compare(t, "implements", iface).Test(t.Implements(iface))
}
