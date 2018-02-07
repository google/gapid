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

package codegen

import "fmt"

func assertTypesEqual(x, y Type) {
	if x != y {
		panic(fmt.Errorf("Arguments have differing types: %v and %v", x.TypeName(), y.TypeName()))
	}
}

func assertVectorsSameLength(x, y Type) {
	vecX, ok := x.(Vector)
	if !ok {
		return
	}
	vecY, ok := y.(Vector)
	if !ok {
		return
	}
	if vecX.Count != vecY.Count {
		panic(fmt.Errorf("Expected vectors of same length, got %v and %v", x.TypeName(), y.TypeName()))
	}
}
