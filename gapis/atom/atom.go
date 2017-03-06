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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/schema"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay/builder"
)

// Atom is the interface implemented by all objects that describe an single
// event in a capture stream. Typical implementations of Atom describe an
// application's call to a graphics API function or provide meta-data describing
// observed memory or state at the time of capture.
//
// Each implementation of Atom should have a unique and stable Signature to ensure
// binary compatibility with old capture formats. Any change to the Atom's
// binary format should also result in a new Signature.
type Atom interface {
	binary.Object

	// All atoms belong to an API
	gfxapi.APIObject

	// AtomFlags returns the flags of the atom.
	AtomFlags() Flags

	// Extras returns all the Extras associated with the dynamic atom.
	Extras() *Extras

	// Mutate mutates the State using the atom. If the builder argument is
	// not nil then it will call the replay function on the builder.
	Mutate(log.Context, *gfxapi.State, *builder.Builder) error
}

// ID is the index of an atom in an atom stream.
type ID uint64

// NoID is used when you have to pass an ID, but don't have one to use.
const NoID = ID(1<<63 - 1) // use max int64 for the benefit of java

// AtomCast is automatically called by the generated decoders.
func AtomCast(obj binary.Object) Atom {
	if o, found := obj.(*schema.Object); found {
		a, err := Wrap(o)
		if err != nil {
			panic(err)
		}
		return a
	}
	if obj == nil {
		return nil
	}
	return obj.(Atom)
}
