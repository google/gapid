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

// An example of testing integer values
func ExampleInteger() {
	assert := assert.To(nil)
	assert.For("1 Equals 2").ThatInteger(1).Equals(2)
	assert.For("1 NotEquals 2").ThatInteger(1).NotEquals(2)
	assert.For("1 IsAtLeast 2").ThatInteger(1).IsAtLeast(2)
	assert.For("3 IsAtMost 4").ThatInteger(3).IsAtMost(4)
	assert.For("6 IsAtLeast 5").ThatInteger(6).IsAtLeast(5)
	assert.For("8 IsAtMost 7").ThatInteger(8).IsAtMost(7)
	assert.For("3 IsBetween [2, 4]").ThatInteger(3).IsBetween(2, 4)
	assert.For("3 IsBetween [3, 4]").ThatInteger(3).IsBetween(3, 4)
	assert.For("3 IsBetween [2, 3]").ThatInteger(3).IsBetween(2, 3)
	assert.For("3 IsBetween [4, 5]").ThatInteger(3).IsBetween(4, 5)
	assert.For("3 IsBetween [1, 2]").ThatInteger(3).IsBetween(1, 2)
	// Output:
	// Error:1 Equals 2
	//     Got       1
	//     Expect == 2
	// Error:1 IsAtLeast 2
	//     Got       1
	//     Expect >= 2
	// Error:8 IsAtMost 7
	//     Got       8
	//     Expect <= 7
	// Error:3 IsBetween [4, 5]
	//     Got       3
	//     Expect in 4 to 5
	// Error:3 IsBetween [1, 2]
	//     Got       3
	//     Expect in 1 to 2
}
