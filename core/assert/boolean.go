// Copyright (C) 2019 Google Inc.
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

// OnBoolean is the result of calling ThatBoolean on an Assertion.
// It provides boolean assertion tests.
type OnBoolean struct {
	Assertion
	value bool
}

// ThatBoolean returns an OnBoolean for boolean based assertions.
func (a Assertion) ThatBoolean(value bool) OnBoolean {
	return OnBoolean{Assertion: a, value: value}
}

// Equals asserts that the supplied boolean is equal to the expected boolean.
func (o OnBoolean) Equals(expect bool) bool {
	return o.Compare(o.value, "==", expect).Test(o.value == expect)
}

// NotEquals asserts that the supplied boolean is not equal to the test boolean.
func (o OnBoolean) NotEquals(test bool) bool {
	return o.Compare(o.value, "!=", test).Test(o.value != test)
}

// IsTrue asserts that the supplied boolean is true
func (o OnBoolean) IsTrue() bool {
	return o.Equals(true)
}

// IsFalse asserts that the supplied boolean is false
func (o OnBoolean) IsFalse() bool {
	return o.Equals(false)
}
