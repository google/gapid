// Copyright (C) 2018 Google Inc.
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

// An example of testing for map lengths
func ExampleMapLength() {
	assert := assert.To(nil)
	var nilMap map[string]string
	emptyMap := map[string]string{}
	nonEmptyMap := map[string]string{
		"one": "two",
	}
	assert.For("nil is empty").ThatMap(nilMap).IsEmpty()
	assert.For("nil is not empty").ThatMap(nilMap).IsNotEmpty()
	assert.For("nil is length 0").ThatMap(nilMap).IsLength(0)
	assert.For("nil is length 1").ThatMap(nilMap).IsLength(1)
	assert.For("{} is empty").ThatMap(emptyMap).IsEmpty()
	assert.For("{} is not empty").ThatMap(emptyMap).IsNotEmpty()
	assert.For("{} is length 0").ThatMap(emptyMap).IsLength(0)
	assert.For("{} is length 1").ThatMap(emptyMap).IsLength(1)
	assert.For("values is empty").ThatMap(nonEmptyMap).IsEmpty()
	assert.For("values is not empty").ThatMap(nonEmptyMap).IsNotEmpty()
	assert.For("values is length 0").ThatMap(nonEmptyMap).IsLength(0)
	assert.For("values is length 1").ThatMap(nonEmptyMap).IsLength(1)
	// Output:
	// Error:nil is not empty
	//     Got             0
	//     Expect length > 0
	// Error:nil is length 1
	//     Got              0
	//     Expect length == 1
	// Error:{} is not empty
	//     Got             0
	//     Expect length > 0
	// Error:{} is length 1
	//     Got              0
	//     Expect length == 1
	// Error:values is empty
	//     Got       1
	//     Expect is empty
	// Error:values is length 0
	//     Got              1
	//     Expect length == 0
}

// An example of testing for map equality
func ExampleMapEquals() {
	assert := assert.To(nil)
	mp := map[int]string{0: "zero", 1: "one"}
	theSame := map[int]string{0: "zero", 1: "one"}
	longer := map[int]string{0: "zero", 1: "one", 2: "two"}
	// Note: differ by at most one key otherwise it won't print the errors
	// in a consistent order
	different := map[int]string{0: "zero", 1: "two"}
	assert.For("the same").ThatMap(mp).Equals(theSame)
	assert.For("shorter").ThatMap(mp).Equals(longer)
	assert.For("longer").ThatMap(longer).Equals(mp)
	assert.For("different").ThatMap(mp).Equals(different)
	// Output:
	// Error:shorter
	//      Key missing: 2
	// Error:longer
	//      Extra key: 2
	// Error:different
	//      Key: 1, "one" differs from expected: "two"
}

// An example of testing for map equality
func ExampleMapDeepEquals() {
	assert := assert.To(nil)
	mp := map[int][]int{0: []int{0, 0}, 1: []int{1, 1}}
	theSame := map[int][]int{0: []int{0, 0}, 1: []int{1, 1}}
	longer := map[int][]int{0: []int{0, 0}, 1: []int{1, 1}, 2: []int{2, 2}}
	// Note: differ by at most one key otherwise it won't print the errors
	// in a consistent order
	different := map[int][]int{0: []int{0, 0}, 1: []int{1, 2}}
	assert.For("the same").ThatMap(mp).DeepEquals(theSame)
	assert.For("shorter").ThatMap(mp).DeepEquals(longer)
	assert.For("longer").ThatMap(longer).DeepEquals(mp)
	assert.For("different").ThatMap(mp).DeepEquals(different)
	// Output:
	// Error:shorter
	//      Key missing: 2
	// Error:longer
	//      Extra key: 2
	// Error:different
	//      Key: 1, []int{1, 1} differs from expected: []int{1, 2}
}
