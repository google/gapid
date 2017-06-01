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

package text_test

import (
	"fmt"

	"github.com/google/gapid/core/text"
)

func ExampleLineNumber() {
	fmt.Println(text.LineNumber(
		`This is some text.
That will have line numbers
added to it.

You may notice that the
line numbers will be of
a fixed width
so that text doesn't move
off to the right once
you get to the 10th line.`))
	// Output:
	//  1: This is some text.
	//  2: That will have line numbers
	//  3: added to it.
	//  4:
	//  5: You may notice that the
	//  6: line numbers will be of
	//  7: a fixed width
	//  8: so that text doesn't move
	//  9: off to the right once
	// 10: you get to the 10th line.
}
