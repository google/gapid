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

import "github.com/pkg/errors"

// OnError is the result of calling ThatError on an Assertion.
// It provides assertion tests that are specific to error types.
type OnError struct {
	Assertion
	err error
}

// ThatError returns an OnError for error type assertions.
func (a Assertion) ThatError(err error) OnError {
	return OnError{Assertion: a, err: err}
}

// Succeeded asserts that the error value was nil.
func (o OnError) Succeeded() bool {
	return o.CompareRaw(o.err, "", "success").Test(o.err == nil)
}

// Failed asserts that the error value was not nil.
func (o OnError) Failed() bool {
	return o.ExpectRaw("", "failure").Test(o.err != nil)
}

// Equals asserts that the error value matches the expected error.
func (o OnError) Equals(expect error) bool {
	return o.Compare(o.err, "==", expect).Test(o.err == expect)
}

// DeepEquals asserts that the error value matches the expected error using compare.DeepEqual.
func (o OnError) DeepEquals(expect error) bool {
	return o.TestDeepEqual(o.err, expect)
}

// HasMessage asserts that the error string matches the expected message.
func (o OnError) HasMessage(expect string) bool {
	return o.Compare(o.err.Error(), "has message", expect).Test(o.err != nil && o.err.Error() == expect)
}

// HasCause asserts that the error cause matches expected error.
func (o OnError) HasCause(expect error) bool {
	cause := errors.Cause(o.err)
	return o.Got(o.err).Add("Cause", cause).Expect("==", expect).Test(cause == expect)
}
