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
	ctx := assert.Context(nil)
	assert.With(ctx).ThatInteger(1).Equals(2)
	assert.With(ctx).ThatInteger(1).NotEquals(2)
	assert.With(ctx).ThatInteger(1).IsAtLeast(2)
	assert.With(ctx).ThatInteger(3).IsAtMost(4)
	assert.With(ctx).ThatInteger(6).IsAtLeast(5)
	assert.With(ctx).ThatInteger(8).IsAtMost(7)
	assert.With(ctx).ThatInteger(3).IsBetween(2, 4)
	assert.With(ctx).ThatInteger(3).IsBetween(3, 4)
	assert.With(ctx).ThatInteger(3).IsBetween(2, 3)
	assert.With(ctx).ThatInteger(3).IsBetween(4, 5)
	assert.With(ctx).ThatInteger(3).IsBetween(1, 2)
	// Output:
	// Error:
	//     Got       1
	//     Expect == 2
	// Error:
	//     Got       1
	//     Expect >= 2
	// Error:
	//     Got       8
	//     Expect <= 7
	// Error:
	//     Got       3
	//     Expect in 4 to 5
	// Error:
	//     Got       3
	//     Expect in 1 to 2
}
