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

package api

import "fmt"

// CmdID is the index of a command in a command stream.
type CmdID uint64

// CmdNoID is used when you have to pass an ID, but don't have one to use.
const CmdNoID = CmdID(1<<63 - 1) // use max int64 for the benefit of java

// Derived is used to create an ID which is used for generated extra commands.
// It is used purely for debugging (to print the related original command ID).
func (id CmdID) Derived() CmdID {
	return id | derivedBit
}

// Real create a real CmdID from a Derived CmdID.
func (id CmdID) Real() CmdID {
	return id & ^derivedBit
}

// IsReal returns true if the id is not derived nor CmdNoID.
func (id CmdID) IsReal() bool {
	return id != CmdNoID && (id&derivedBit) == 0
}

const derivedBit = CmdID(1 << 62)

func (id CmdID) String() string {
	if id == CmdNoID {
		return "(NoID)"
	} else if (id & derivedBit) != 0 {
		return fmt.Sprintf("%v*", uint64(id & ^derivedBit))
	} else {
		return fmt.Sprintf("%v", uint64(id))
	}
}
