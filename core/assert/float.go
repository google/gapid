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

// OnFloat is the result of calling ThatFloat on an Assertion.
// It provides numeric assertion tests.
type OnFloat struct {
	Assertion
	value float64
}

// ThatFloat returns an OnFloat for floating point based assertions.
func (a Assertion) ThatFloat(value float64) OnFloat {
	return OnFloat{Assertion: a, value: value}
}

// IsAtLeast asserts that the float is at least the supplied minimum.
func (o OnFloat) IsAtLeast(min float64) bool {
	return o.Compare(o.value, ">=", min).Test(o.value >= min)
}

// IsAtMost asserts that the float is at most the supplied maximum.
func (o OnFloat) IsAtMost(max float64) bool {
	return o.Compare(o.value, "<=", max).Test(o.value <= max)
}

// Equals asserts that the float equals v with ± tolerance.
func (o OnFloat) Equals(v, tolerance float64) bool {
	min, max := v-tolerance, v+tolerance
	return o.Compare(o.value, "in", min, "to", max).Test(o.value >= min && o.value <= max)
}
