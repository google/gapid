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

// Package sync provides interfaces for managing externally synchronized APIs.
//
// The methods allow queries to be performed on an API to allow
// the determination of where blocking operations between threads
// of execution happen. These methods allow us to reason about
// execution in a non-linear way.
package sync

import (
	"context"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/service/path"
)

// SynchronizedApi defines an API that explicitly has multiple threads of
// execution. This means that replays are not necessarily linear in terms
// of atoms.
type SynchronizedAPI interface {
	// GetTerminator returns a transform that will allow the given capture to be terminated
	// after a atom
	GetTerminator(ctx context.Context, c *path.Capture) (transform.Terminator, error)

	// ResolveSynchronization resolve all of the synchronization information for
	// the given API
	ResolveSynchronization(ctx context.Context, d *Data, c *path.Capture) error

	// MutateSubcommands mutates the given Atom calling callback after each subcommand is executed.
	MutateSubcommands(ctx context.Context, id atom.ID, cmd api.Cmd, s *api.State, callback func(*api.State, SubcommandIndex, api.Cmd)) error
}

type writer struct {
	st    *api.State
	Atoms *atom.List
}

func (s writer) State() *api.State {
	return s.st
}

func (s writer) MutateAndWrite(ctx context.Context, id atom.ID, cmd api.Cmd) {
	cmd.Mutate(ctx, s.st, nil)
	s.Atoms.Atoms = append(s.Atoms.Atoms, cmd)
}

// MutationAtomsFor returns a list of atoms that represent the correct mutations to have the state for all
// atoms before and including the given index.
func MutationAtomsFor(ctx context.Context, c *path.Capture, atoms *atom.List, id atom.ID, subindex []uint64) (*atom.List, error) {
	// This is where we want to handle sub-states
	// This involves transforming the tree for the given Indices, and
	//   then mutating that.
	rc, err := capture.ResolveFromPath(ctx, c)
	if err != nil {
		return nil, err
	}
	terminators := make([]transform.Terminator, 0)
	transforms := transform.Transforms{}

	for _, api := range rc.APIs {
		if sync, ok := api.(SynchronizedAPI); ok {
			term, err := sync.GetTerminator(ctx, c)
			if err == nil {
				terminators = append(terminators, term)
			} else {
				return nil, err
			}

		} else {
			terminators = append(terminators, &transform.EarlyTerminator{APIIdx: api.ID()})
		}
	}
	for _, t := range terminators {
		if err := t.Add(ctx, id, subindex); err != nil {
			return nil, err
		}
		transforms.Add(t)
	}

	state := rc.NewState()
	a := atom.List{Atoms: make([]api.Cmd, 0)}
	w := writer{state, &a}
	transforms.Transform(ctx, *atoms, w)
	return &a, nil
}

// MutateWithSubcommands returns a list of atoms that represent the correct
// mutations to have the state for all atoms before and including the given index.
func MutateWithSubcommands(ctx context.Context, c *path.Capture, atoms atom.List, callback func(*api.State, SubcommandIndex, api.Cmd)) error {
	// This is where we want to handle sub-states
	// This involves transforming the tree for the given Indices, and
	//   then mutating that.
	rc, err := capture.ResolveFromPath(ctx, c)
	if err != nil {
		return err
	}
	s := rc.NewState()

	for id, cmd := range atoms.Atoms {
		if sync, ok := cmd.API().(SynchronizedAPI); ok {
			if err := sync.MutateSubcommands(ctx, atom.ID(id), cmd, s, callback); err != nil && err == context.Canceled {
				return err
			}
		} else {
			if err := cmd.Mutate(ctx, s, nil); err != nil && err == context.Canceled {
				return err
			}
		}
		callback(s, SubcommandIndex{uint64(id)}, cmd)
	}

	return nil
}
