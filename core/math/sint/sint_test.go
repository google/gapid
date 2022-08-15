// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package sint_test

import (
	"fmt"

	"github.com/google/gapid/core/math/sint"
)

func ExampleLog10() {
	for _, n := range []int{0, 1, 2, 9, 10, 11, 99, 100, 101} {
		fmt.Printf("Log10(%v): %v\n", n, sint.Log10(n))
	}
	// Output:
	// Log10(0): 0
	// Log10(1): 0
	// Log10(2): 0
	// Log10(9): 0
	// Log10(10): 1
	// Log10(11): 1
	// Log10(99): 1
	// Log10(100): 2
	// Log10(101): 2
}
