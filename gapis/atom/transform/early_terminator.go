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

package transform

import (
	"context"

	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
)

// EarlyTerminator is an implementation of Transformer that will consume all
// atoms (except for the EOS atom) once all the atoms passed to Add have passed
// through the transformer. It will only remove atoms of the given API type
type EarlyTerminator struct {
	lastIndex atom.ID
	done      bool
	ApiIdx    gfxapi.ID
}

// Interface check
var _ Terminator = &EarlyTerminator{}

// Adds the given atom as the last atom that must be passed through the
// pass. Once the atom with the given id is found all atoms from this API
// will be silenced.
// This takes advantage of the fact that in practice IDs are sequential, and
// atom.NoID is used for new atoms.
func (t *EarlyTerminator) Add(ctx context.Context, id atom.ID, idx []uint64) error {
	if id > t.lastIndex {
		t.lastIndex = id
	}
	return nil
}

func (t *EarlyTerminator) Transform(ctx context.Context, id atom.ID, a atom.Atom, out Writer) {
	if t.done && (a.API() == nil || a.API().ID() == t.ApiIdx) {
		return
	}

	out.MutateAndWrite(ctx, id, a)
	// Keep a.API() == nil so that we can test without an API
	if t.lastIndex == id {
		t.done = true
		return
	}
}

func (t *EarlyTerminator) Flush(ctx context.Context, out Writer) {}
