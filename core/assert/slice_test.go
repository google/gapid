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

// An example of testing for slice lengths
func ExampleSliceLength() {
	assert := assert.To(nil)
	var nilSlice []string
	emptySlice := []string{}
	nonEmptySlice := []string{"one"}
	assert.For("nil is empty").ThatSlice(nilSlice).IsEmpty()
	assert.For("nil is not empty").ThatSlice(nilSlice).IsNotEmpty()
	assert.For("nil is length 0").ThatSlice(nilSlice).IsLength(0)
	assert.For("nil is length 1").ThatSlice(nilSlice).IsLength(1)
	assert.For("{} is empty").ThatSlice(emptySlice).IsEmpty()
	assert.For("{} is not empty").ThatSlice(emptySlice).IsNotEmpty()
	assert.For("{} is length 0").ThatSlice(emptySlice).IsLength(0)
	assert.For("{} is length 1").ThatSlice(emptySlice).IsLength(1)
	assert.For("values is empty").ThatSlice(nonEmptySlice).IsEmpty()
	assert.For("values is not empty").ThatSlice(nonEmptySlice).IsNotEmpty()
	assert.For("values is length 0").ThatSlice(nonEmptySlice).IsLength(0)
	assert.For("values is length 1").ThatSlice(nonEmptySlice).IsLength(1)
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

// An example of testing for slice equality
func ExampleSliceEquals() {
	assert := assert.To(nil)
	slice := []string{"a1", "a2"}
	theSame := []string{"a1", "a2"}
	longer := []string{"a1", "a2", "a3"}
	different := []string{"a1", "a3"}
	assert.For("the same").ThatSlice(slice).Equals(theSame)
	assert.For("longer").ThatSlice(slice).Equals(longer)
	assert.For("shorter").ThatSlice(longer).Equals(slice)
	assert.For("different").ThatSlice(slice).Equals(different)
	// Output:
	// Error:longer
	//       0 string `a1` ;
	//       1 string `a2` ;
	//     - 2             ==> string `a3`
	// Error:shorter
	//       0 string `a1` ;
	//       1 string `a2` ;
	//     + 2 string `a3` ;
	// Error:different
	//       0 string `a1` ;
	//     * 1 string `a2` ==> string `a3`
}

// An example of testing for slice equality
func ExampleSliceDeepEquals() {
	// TODO: slices of a more complext type
	assert := assert.To(nil)
	slice := []string{"a1", "a2"}
	theSame := []string{"a1", "a2"}
	longer := []string{"a1", "a2", "a3"}
	different := []string{"a1", "a3"}
	assert.For("the same").ThatSlice(slice).DeepEquals(theSame)
	assert.For("longer").ThatSlice(slice).DeepEquals(longer)
	assert.For("different").ThatSlice(slice).DeepEquals(different)
	// Output:
	// Error:longer
	//       0 string `a1` ;
	//       1 string `a2` ;
	//     - 2             ==> string `a3`
	// Error:different
	//       0 string `a1` ;
	//     * 1 string `a2` ==> string `a3`
}
