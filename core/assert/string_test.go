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

// An example of testing string equality
func ExampleStringEquals() {
	assert := assert.To(nil)
	assert.For(`"a" Equals "a"`).ThatString("a").Equals("a")
	assert.For(`"a" Equals "b"`).ThatString("a").Equals("b")
	assert.For(`"a" NotEquals "a"`).ThatString("a").NotEquals("a")
	assert.For(`"a" NotEquals "b"`).ThatString("a").NotEquals("b")
	// Output:
	// Error:"a" Equals "b"
	//     Got          `a`
	//     Expect  ==   `b`
	//     Differs from `a`
	// Error:"a" NotEquals "a"
	//     Got       `a`
	//     Expect != `a`
}

// An example of testing partial string equality
func ExampleStringFragments() {
	assert := assert.To(nil)
	assert.For(`"abc" Contains "b"`).ThatString("abc").Contains("b")
	assert.For(`"abc" Contains "d"`).ThatString("abc").Contains("d")
	assert.For(`"abc" HasPrefix "a"`).ThatString("abc").HasPrefix("a")
	assert.For(`"abc" HasPrefix "b"`).ThatString("abc").HasPrefix("b")
	assert.For(`"abc" HasSuffix "c"`).ThatString("abc").HasSuffix("c")
	assert.For(`"abc" HasSuffix "b"`).ThatString("abc").HasSuffix("b")
	// Output:
	// Error:"abc" Contains "d"
	//     Got             `abc`
	//     Expect contains `d`
	// Error:"abc" HasPrefix "b"
	//     Got                `abc`
	//     Expect starts with `b`
	// Error:"abc" HasSuffix "b"
	//     Got              `abc`
	//     Expect ends with `b`
}

// An example of testing non strings as strings
func ExampleStringTypes() {
	assert := assert.To(nil)
	assert.For(`"abc" Equals "a"`).ThatString([]byte{'a', 'b', 'c'}).Equals("a")
	assert.For(`10 Equals "10"`).ThatString(10).Equals("10")
	// Output:
	// Error:"abc" Equals "a"
	//     Got       `abc`
	//     Expect == `a`
	//     Longer by `bc`
}
