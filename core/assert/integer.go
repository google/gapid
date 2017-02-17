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

// OnInteger is the result of calling ThatInt on an Assertion.
// It provides numeric assertion tests.
type OnInteger struct {
	Assertion
	value int
}

// ThatInteger returns an OnInteger for integer based assertions.
func (a Assertion) ThatInteger(value int) OnInteger {
	return OnInteger{Assertion: a, value: value}
}

// Equals asserts that the supplied integer is equal to the expected integer.
func (o OnInteger) Equals(expect int) bool {
	return o.Compare(o.value, "==", expect).Test(o.value == expect)
}

// NotEquals asserts that the supplied integer is not equal to the test integer.
func (o OnInteger) NotEquals(test int) bool {
	return o.Compare(o.value, "!=", test).Test(o.value != test)
}

// IsAtLeast asserts that the integer is at least the supplied minimum.
func (o OnInteger) IsAtLeast(min int) bool {
	return o.Compare(o.value, ">=", min).Test(o.value >= min)
}

// IsAtMost asserts that the integer is at most the supplied maximum.
func (o OnInteger) IsAtMost(max int) bool {
	return o.Compare(o.value, "<=", max).Test(o.value <= max)
}

// IsBetween asserts that the integer lies within the given range (inclusive).
func (o OnInteger) IsBetween(min, max int) bool {
	return o.CompareRaw(o.value, "in", min, "to", max).Test(o.value >= min && o.value <= max)
}
