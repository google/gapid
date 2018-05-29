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
	"time"

	"github.com/google/gapid/core/assert"
)

// An example of testing time durations
func ExampleTime() {
	assert := assert.To(nil)
	assert.For("1s IsAtLeast 2s").ThatDuration(time.Second * 1).IsAtLeast(time.Second * 2)
	assert.For("3s IsAtMost 4s").ThatDuration(time.Second * 3).IsAtMost(time.Second * 4)
	assert.For("6s IsAtLeast 5s").ThatDuration(time.Second * 6).IsAtLeast(time.Second * 5)
	assert.For("8s IsAtMost 7s").ThatDuration(time.Second * 8).IsAtMost(time.Second * 7)
	// Output:
	// Error:1s IsAtLeast 2s
	//     Got       1s
	//     Expect >= 2s
	// Error:8s IsAtMost 7s
	//     Got       8s
	//     Expect <= 7s
}
