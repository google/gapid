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

import "github.com/google/gapid/core/assert"

// An example of testing float values
func ExampleFloat() {
	assert := assert.To(nil)
	assert.For("1.1 IsAtLeast 2.1").ThatFloat(1.1).IsAtLeast(2.1)
	assert.For("3.1 IsAtMost 4.1").ThatFloat(3.1).IsAtMost(4.1)
	assert.For("6.1 IsAtLeast 5.1").ThatFloat(6.1).IsAtLeast(5.1)
	assert.For("8.1 IsAtMost 7.1").ThatFloat(8.1).IsAtMost(7.1)
	// Output:
	// Error:1.1 IsAtLeast 2.1
	//     Got       1.1
	//     Expect >= 2.1
	// Error:8.1 IsAtMost 7.1
	//     Got       8.1
	//     Expect <= 7.1
}
