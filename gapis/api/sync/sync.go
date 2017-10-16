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
// of commands.
type SynchronizedAPI interface {
	// GetTerminator returns a transform that will allow the given capture to be terminated
	// after a command.
	GetTerminator(ctx context.Context, c *path.Capture) (transform.Terminator, error)

	// ResolveSynchronization resolve all of the synchronization information for
	// the given API.
	ResolveSynchronization(ctx context.Context, d *Data, c *path.Capture) error

	// MutateSubcommands mutates the given Cmd and calls callbacks for subcommands
	// attached to that Cmd. preSubCmdCallback and postSubCmdCallback will be
	// called before and after executing each subcommand callback.
	MutateSubcommands(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.GlobalState,
		preSubCmdCallback func(*api.GlobalState, api.SubCmdIdx, api.Cmd),
		postSubCmdCallback func(*api.GlobalState, api.SubCmdIdx, api.Cmd)) error

	// FlattenSubcommandIdx returns the flatten command id for the subcommand
	// specified by the given SubCmdIdx. If flattening succeeded, the flatten
	// command id and true will be returned, otherwise, zero and false will be
	// returned.
	FlattenSubcommandIdx(idx api.SubCmdIdx, d *Data, initialCall bool) (api.CmdID, bool)
}

type writer struct {
	state *api.GlobalState
	cmds  []api.Cmd
}

func (s *writer) State() *api.GlobalState { return s.state }

func (s *writer) MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) {
	cmd.Mutate(ctx, id, s.state, nil)
	s.cmds = append(s.cmds, cmd)
}

// MutationCmdsFor returns a list of command that represent the correct
// mutations to have the state for all commands before and including the given
// index.
func MutationCmdsFor(ctx context.Context, c *path.Capture, data *Data, cmds []api.Cmd, id api.CmdID, subindex api.SubCmdIdx, initialCall bool) ([]api.Cmd, error) {
	// This is where we want to handle sub-states
	// This involves transforming the tree for the given Indices, and
	//   then mutating that.
	rc, err := capture.ResolveFromPath(ctx, c)
	if err != nil {
		return nil, err
	}

	fullCommand := api.SubCmdIdx{uint64(id)}
	fullCommand = append(fullCommand, subindex...)

	lastCmd := cmds[len(cmds)-1]
	if api.CmdID(len(cmds)) > id {
		lastCmd = cmds[id]
	}

	if sync, ok := lastCmd.API().(SynchronizedAPI); ok {
		// For Vulkan, when preparing the mutation for memory view, we need to get
		// the initial call ID for the requesting subcommand.
		if flattenIdx, ok := sync.FlattenSubcommandIdx(fullCommand, data, initialCall); ok {
			id = flattenIdx
			subindex = api.SubCmdIdx{}
		}
	}

	terminators := make([]transform.Terminator, 0)
	transforms := transform.Transforms{}

	for _, api := range rc.APIs {
		if sync, ok := api.(SynchronizedAPI); ok {
			term, err := sync.GetTerminator(ctx, c)
			if err != nil {
				return nil, err
			}
			if term != nil {
				terminators = append(terminators, term)
				continue
			}
		}
		terminators = append(terminators, transform.NewEarlyTerminator(api.ID()))
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

// MutateWithSubcommands mutates a list of commands. And after mutating each
// Cmd, the given post-Cmd callback will be called. And the given
// pre-subcommand callback and the post-subcommand callback will be called
// before and after calling each subcommand callback function.
func MutateWithSubcommands(ctx context.Context, c *path.Capture, cmds []api.Cmd,
	postCmdCb func(*api.GlobalState, api.SubCmdIdx, api.Cmd),
	preSubCmdCb func(*api.GlobalState, api.SubCmdIdx, api.Cmd),
	postSubCmdCb func(*api.GlobalState, api.SubCmdIdx, api.Cmd)) error {
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
			sync.MutateSubcommands(ctx, id, cmd, s, preSubCmdCb, postSubCmdCb)
		} else {
			cmd.Mutate(ctx, id, s, nil)
		}
		postCmdCb(s, api.SubCmdIdx{uint64(id)}, cmd)
		return nil
	})
}
