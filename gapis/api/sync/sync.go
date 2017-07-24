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
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/service/path"
)

// SynchronizedAPI defines an API that explicitly has multiple threads of
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
	MutateSubcommands(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.State, callback func(*api.State, api.SubCmdIdx, api.Cmd)) error
}

type writer struct {
	state *api.State
	cmds  []api.Cmd
}

func (s *writer) State() *api.State { return s.state }

func (s *writer) MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) {
	cmd.Mutate(ctx, s.state, nil)
	s.cmds = append(s.cmds, cmd)
}

// MutationCmdsFor returns a list of command that represent the correct
// mutations to have the state for all commands before and including the given
// index.
func MutationCmdsFor(ctx context.Context, c *path.Capture, cmds []api.Cmd, id api.CmdID, subindex []uint64) ([]api.Cmd, error) {
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
			if err != nil {
				return nil, err
			}
			terminators = append(terminators, term)
		} else {
			terminators = append(terminators, transform.NewEarlyTerminator(api.ID()))
		}
	}
	for _, t := range terminators {
		if err := t.Add(ctx, id, subindex); err != nil {
			return nil, err
		}
		transforms.Add(t)
	}

	w := &writer{rc.NewState(), nil}
	transforms.Transform(ctx, cmds, w)
	return w.cmds, nil
}

// MutateWithSubcommands returns a list of commands that represent the correct
// mutations to have the state for all commands before and including the given
// index.
func MutateWithSubcommands(ctx context.Context, c *path.Capture, cmds []api.Cmd, callback func(*api.State, api.SubCmdIdx, api.Cmd)) error {
	// This is where we want to handle sub-states
	// This involves transforming the tree for the given Indices, and
	//   then mutating that.
	rc, err := capture.ResolveFromPath(ctx, c)
	if err != nil {
		return err
	}
	s := rc.NewState()

	return api.ForeachCmd(ctx, cmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		if sync, ok := cmd.API().(SynchronizedAPI); ok {
			sync.MutateSubcommands(ctx, id, cmd, s, callback)
		} else {
			cmd.Mutate(ctx, s, nil)
		}
		callback(s, api.SubCmdIdx{uint64(id)}, cmd)
		return nil
	})
}
