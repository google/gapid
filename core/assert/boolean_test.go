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

package assert_test

import "github.com/google/gapid/core/assert"

// An example of testing integer values
func ExampleBoolean() {
	assert := assert.To(nil)
	assert.For("true Equals true").ThatBoolean(true).Equals(true)
	assert.For("true NotEquals false").ThatBoolean(true).NotEquals(false)
	assert.For("true IsTrue").ThatBoolean(true).IsTrue()
	assert.For("false IsFalse").ThatBoolean(false).IsFalse()
}
