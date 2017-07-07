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

package atom

import "fmt"

// ID is the index of an atom in an atom stream.
type ID uint64

// NoID is used when you have to pass an ID, but don't have one to use.
const NoID = ID(1<<63 - 1) // use max int64 for the benefit of java

// Derived is used to create an ID which is used for generated extra atoms.
// It is used purely for debugging (to print the related original atom ID).
func (id ID) Derived() ID {
	return id | derivedBit
}

const derivedBit = ID(1 << 62)

func (id ID) String() string {
	if id == NoID {
		return "(NoID)"
	} else if (id & derivedBit) != 0 {
		return fmt.Sprintf("%v*", uint64(id & ^derivedBit))
	} else {
		return fmt.Sprintf("%v", uint64(id))
	}
}
