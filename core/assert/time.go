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

import "time"

// OnDuration is the result of calling ThatDuration on an Assertion.
// It provides assertion tests that are specific to the time duration.
type OnDuration struct {
	Assertion
	value time.Duration
}

// ThatDuration returns an OnDuration for time duration based assertions.
func (a Assertion) ThatDuration(value time.Duration) OnDuration {
	return OnDuration{Assertion: a, value: value}
}

// IsAtLeast asserts that the time duration is at least than the supplied minimum.
func (o OnDuration) IsAtLeast(min time.Duration) bool {
	return o.Compare(o.value, ">=", min).Test(o.value >= min)
}

// IsAtMost asserts that the time duration is at most than the supplied maximum.
func (o OnDuration) IsAtMost(max time.Duration) bool {
	return o.Compare(o.value, "<=", max).Test(o.value <= max)
}
