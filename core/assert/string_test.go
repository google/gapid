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
	ctx := assert.Context(nil)
	assert.With(ctx).ThatString("a").Equals("a")
	assert.With(ctx).ThatString("a").Equals("b")
	assert.With(ctx).ThatString("a").NotEquals("a")
	assert.With(ctx).ThatString("a").NotEquals("b")
	// Output:
	// Error:
	//     Got          `a`
	//     Expect  ==   `b`
	//     Differs from `a`
	// Error:
	//     Got       `a`
	//     Expect != `a`
}

// An example of testing partial string equality
func ExampleStringFragments() {
	ctx := assert.Context(nil)
	assert.With(ctx).ThatString("abc").Contains("b")
	assert.With(ctx).ThatString("abc").Contains("d")
	assert.With(ctx).ThatString("abc").HasPrefix("a")
	assert.With(ctx).ThatString("abc").HasPrefix("b")
	assert.With(ctx).ThatString("abc").HasSuffix("c")
	assert.With(ctx).ThatString("abc").HasSuffix("b")
	// Output:
	// Error:
	//     Got             `abc`
	//     Expect contains `d`
	// Error:
	//     Got                `abc`
	//     Expect starts with `b`
	// Error:
	//     Got              `abc`
	//     Expect ends with `b`
}

// An example of testing non strings as strings
func ExampleStringTypes() {
	ctx := assert.Context(nil)
	assert.With(ctx).ThatString([]byte{'a', 'b', 'c'}).Equals("a")
	assert.With(ctx).ThatString(10).Equals("10")
	// Output:
	// Error:
	//     Got       `abc`
	//     Expect == `a`
	//     Longer by `bc`
}
