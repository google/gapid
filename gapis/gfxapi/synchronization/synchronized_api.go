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

package synchronization

import (
	"context"

	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/service/path"
)

// SynchronizedApi defines an API that explicitly has multiple threads of
// execution. This means that replays are not necessarily linear in terms
// of atoms.
type SynchronizedApi interface {
	// GetTerminator returns a tranform that will allow the given capture to be terminated
	// after a atom
	GetTerminator(ctx context.Context, c *path.Capture) (transform.Terminator, error)

	// ResolveSynchronization resolve all of the synchronization information for
	// the given API
	ResolveSynchronization(ctx context.Context, d *SynchronizationData, c *path.Capture) error
}

type writer struct {
	st    *gfxapi.State
	Atoms *atom.List
}

func (s writer) State() *gfxapi.State {
	return s.st
}

func (s writer) MutateAndWrite(ctx context.Context, id atom.ID, atom atom.Atom) {
	atom.Mutate(ctx, s.st, nil)
	s.Atoms.Atoms = append(s.Atoms.Atoms, atom)
}

// Returns a list of atoms that represent the correct mutations to have the state for all
// atoms before and including the given index.
func GetMutationAtomsFor(ctx context.Context, cap *path.Capture, atoms *atom.List, id atom.ID, subindex []uint64) (*atom.List, error) {
	// This is where we want to handle sub-states
	// This involves transforming the tree for the given Indices, and
	//   then mutating that.
	c, err := capture.ResolveFromPath(ctx, cap)
	if err != nil {
		return nil, err
	}
	terminators := make([]transform.Terminator, 0)
	transforms := transform.Transforms{}

	for _, api := range c.APIs {
		if sync, ok := api.(SynchronizedApi); ok {
			term, err := sync.GetTerminator(ctx, cap)
			if err == nil {
				terminators = append(terminators, term)
			} else {
				return nil, err
			}

		} else {
			terminators = append(terminators, &transform.EarlyTerminator{ApiIdx: api.ID()})
		}
	}
	for _, t := range terminators {
		if err := t.Add(ctx, id, subindex); err != nil {
			return nil, err
		}
		transforms.Add(t)
	}

	state := c.NewState()
	a := atom.List{make([]atom.Atom, 0)}
	w := writer{state, &a}
	transforms.Transform(ctx, *atoms, w)
	return &a, nil
}
