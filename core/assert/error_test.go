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

package assert_test

import (
	"errors"

	"github.com/google/gapid/core/assert"
)

// An example of testing errors
func ExampleErrors() {
	assert := assert.To(nil)
	err := errors.New("failure")
	otherErr := errors.New("other failure")
	assert.For("nil succeeded").ThatError(nil).Succeeded()
	assert.For("nil failed").ThatError(nil).Failed()
	assert.For("err succeeded").ThatError(err).Succeeded()
	assert.For("err failed").ThatError(err).Failed()
	assert.For("err equals").ThatError(err).Equals(err)
	assert.For("err not equals").ThatError(err).Equals(otherErr)
	assert.For("err deep equals").ThatError(err).DeepEquals(err)
	assert.For("err deep not equals").ThatError(err).DeepEquals(otherErr)
	assert.For("message").ThatError(err).HasMessage(err.Error())
	assert.For("wrong message").ThatError(err).HasMessage(otherErr.Error())
	// Output:
	// Error:nil failed
	//     Expect  failure
	// Error:err succeeded
	//     Got     `failure`
	//     Expect  success
	// Error:err not equals
	//     Got       `failure`
	//     Expect == `other failure`
	// Error:err deep not equals
	//     Got            `failure`
	//     Expect deep == `other failure`
	// Error:wrong message
	//     Got                `failure`
	//     Expect has message `other failure`
}
