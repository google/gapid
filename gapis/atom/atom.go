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

import (
	"context"
	"fmt"

	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay/builder"
)

// Atom is the interface implemented by all objects that describe an single
// event in a capture stream. Typical implementations of Atom describe an
// application's call to a graphics API function or provide meta-data describing
// observed memory or state at the time of capture.
type Atom interface {
	// All atoms belong to an API
	gfxapi.APIObject

	// Thread returns the thread index this atom was executed on.
	Thread() uint64

	// SetThread changes the thread index.
	SetThread(uint64)

	// AtomName returns the name of the atom.
	AtomName() string

	// AtomFlags returns the flags of the atom.
	AtomFlags() Flags

	// Extras returns all the Extras associated with the dynamic atom.
	Extras() *Extras

	// Mutate mutates the State using the atom. If the builder argument is
	// not nil then it will call the replay function on the builder.
	Mutate(context.Context, *gfxapi.State, *builder.Builder) error
}

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
