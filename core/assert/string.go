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
	"fmt"
	"strings"
)

// OnString is the result of calling ThatString on an Assertion.
// It provides assertion tests that are specific to strings.
type OnString struct {
	Assertion
	value string
}

// ThatString returns an OnString for string based assertions.
// The untyped argument is converted to a string using fmt.Sprint, and the result supports string specific tests.
func (a Assertion) ThatString(value interface{}) OnString {
	s := ""
	switch v := value.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		s = fmt.Sprint(value)
	}
	return OnString{Assertion: a, value: s}
}

// Equals asserts that the supplied string is equal to the expected string.
func (o OnString) Equals(expect string) bool {
	return o.Compare(o.value, "==", expect).Test(func() bool {
		if o.value == expect {
			return true
		}
		re := ([]rune)(expect)
		for i, c := range o.value {
			if i >= len(re) {
				o.Printf("Longer\tby\t")
				o.Println(o.value[i:])
				return false
			}
			if c != re[i] {
				o.Printf("Differs\tfrom\t")
				o.Println(o.value[i:])
				return false
			}
		}
		o.Printf("Shorter\tby\t")
		o.Println(expect[len(o.value):])
		return false
	}())
}

// NotEquals asserts that the supplied string is not equal to the test string.
func (o OnString) NotEquals(test string) bool {
	return o.Compare(o.value, "!=", test).Test(o.value != test)
}

// Contains asserts that the supplied string contains substr.
func (o OnString) Contains(substr string) bool {
	return o.Compare(o.value, "contains", substr).Test(strings.Contains(o.value, substr))
}

// DoesNotContain asserts that the supplied string does not contain substr.
func (o OnString) DoesNotContain(substr string) bool {
	return o.Compare(o.value, "does not contain", substr).Test(!strings.Contains(o.value, substr))
}

// HasPrefix asserts that the supplied string start with substr.
func (o OnString) HasPrefix(substr string) bool {
	return o.Compare(o.value, "starts with", substr).Test(strings.HasPrefix(o.value, substr))
}

// HasSuffix asserts that the supplied string ends with with substr.
func (o OnString) HasSuffix(substr string) bool {
	return o.Compare(o.value, "ends with", substr).Test(strings.HasSuffix(o.value, substr))
}
