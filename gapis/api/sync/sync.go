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
	"fmt"

	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/commandGenerator"
	"github.com/google/gapid/gapis/api/controlFlowGenerator"
	"github.com/google/gapid/gapis/api/terminator"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/service/path"
)

// NoMECSubcommandsError is used to notify the caller that this API does not
// support MEC subcommands.
type NoMECSubcommandsError struct{}

func (e NoMECSubcommandsError) Error() string {
	return "This api does not have MEC subcommands"
}

// SynchronizedAPI defines an API that explicitly has multiple threads of
// execution. This means that replays are not necessarily linear in terms
// of commands.
type SynchronizedAPI interface {
	// GetTerminator returns a transform that will allow the given capture to be terminated
	// after a command.
	GetTerminator(ctx context.Context, c *path.Capture) (terminator.Terminator, error)

	// ResolveSynchronization resolve all of the synchronization information for
	// the given API.
	ResolveSynchronization(ctx context.Context, d *Data, c *path.Capture) error

	// MutateSubcommands mutates the given Cmd and calls callbacks for subcommands
	// attached to that Cmd. preSubCmdCallback and postSubCmdCallback will be
	// called before and after executing each subcommand callback.
	// Both preSubCmdCallback() and postSubCmdCallback() receive the parent command
	// as an api.Cmd, the complete subcommand index, and an interface that allows
	// to retrieve the actual subcommand via API-specific primitives. In Vulkan,
	// this interface{} is a CommandReferenceʳ that can be used in GetCommandArgs().
	MutateSubcommands(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.GlobalState,
		preSubCmdCallback func(s *api.GlobalState, idx api.SubCmdIdx, cmd api.Cmd, subCmdRef interface{}),
		postSubCmdCallback func(s *api.GlobalState, idx api.SubCmdIdx, cmd api.Cmd, subCmdRef interface{})) error

	// FlattenSubcommandIdx returns the flatten command id for the subcommand
	// specified by the given SubCmdIdx. If flattening succeeded, the flatten
	// command id and true will be returned, otherwise, zero and false will be
	// returned.
	FlattenSubcommandIdx(idx api.SubCmdIdx, d *Data, initialCall bool) (api.CmdID, bool)

	// RecoverMidExecutionCommand returns a virtual command, used to describe the
	// a subcommand that was created before the start of the trace.
	// If the api does not have mid-execution commands, NoMECSubcommandsError should be returned.
	RecoverMidExecutionCommand(ctx context.Context, c *path.Capture, data interface{}) (api.Cmd, error)

	// IsTrivialTerminator returns true if stopping at the given command is trivial.
	IsTrivialTerminator(ctx context.Context, c *path.Capture, cmd api.SubCmdIdx) (bool, error)
}

type writer struct {
	state *api.GlobalState
	cmds  []api.Cmd
}

func (s *writer) State() *api.GlobalState { return s.state }

func (s *writer) MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
	if err := cmd.Mutate(ctx, id, s.state, nil, nil); err != nil {
		return err
	}
	s.cmds = append(s.cmds, cmd)
	return nil
}

// MutationCmdsFor returns a list of command that represent the correct
// mutations to have the state for all commands before and including the given
// index.
func MutationCmdsFor(ctx context.Context, c *path.Capture, data *Data, cmds []api.Cmd, id api.CmdID, subindex api.SubCmdIdx, initialCall bool) ([]api.Cmd, error) {
	// This is where we want to handle sub-states
	// This involves transforming the tree for the given Indices, and
	//   then mutating that.
	rc, err := capture.ResolveGraphicsFromPath(ctx, c)
	if err != nil {
		return nil, err
	}

	fullCommand := api.SubCmdIdx{uint64(id)}
	fullCommand = append(fullCommand, subindex...)

	lastCmd := cmds[len(cmds)-1]
	if api.CmdID(len(cmds)) > id {
		lastCmd = cmds[id]
	} else {
		return nil, log.Errf(ctx, nil, "Requested CmdID %v exceeds range of commands", id)
	}

	if sync, ok := lastCmd.API().(SynchronizedAPI); ok {
		// For Vulkan, when preparing the mutation for memory view, we need to get
		// the initial call ID for the requesting subcommand.
		if flattenIdx, ok := sync.FlattenSubcommandIdx(fullCommand, data, initialCall); ok {
			id = flattenIdx
			subindex = api.SubCmdIdx{}
		}
	}

	terminators := make([]terminator.Terminator, 0)
	transforms := make([]transform.Transform, 0)
	isTrivial := true

	for _, api := range rc.APIs {
		if sync, ok := api.(SynchronizedAPI); ok {
			term, err := sync.GetTerminator(ctx, c)
			if err != nil {
				return nil, err
			}

			t, err := sync.IsTrivialTerminator(ctx, c, fullCommand)
			if err != nil {
				return nil, err
			}
			isTrivial = t && isTrivial
			if term != nil {
				terminators = append(terminators, term)
				continue
			}
		}
		terminators = append(terminators, terminator.NewEarlyTerminator())
	}
	for _, t := range terminators {
		if err := t.Add(ctx, id, subindex); err != nil {
			return nil, err
		}
		transforms = append(transforms, t)
	}
	if isTrivial {
		return cmds[0 : id+1], nil
	}
	w := &writer{rc.NewState(ctx), nil}

	cmdGenerator := commandGenerator.NewLinearCommandGenerator(nil, cmds)
	chain := transform.CreateTransformChain(ctx, cmdGenerator, transforms, w)
	controlFlow := controlFlowGenerator.NewLinearControlFlowGenerator(chain)
	if err := controlFlow.TransformAll(ctx); err != nil {
		log.E(ctx, "Sync Error: %v", err)
		return nil, err
	}

	return w.cmds, nil
}

// MutateWithSubcommands mutates a list of commands. And after mutating each
// Cmd, the given post-Cmd callback will be called. And the given
// pre-subcommand callback and the post-subcommand callback will be called
// before and after calling each subcommand callback function.
// If cmds is nil, all commands from the capture are used instead.
func MutateWithSubcommands(ctx context.Context, c *path.Capture, cmds []api.Cmd,
	postCmdCb func(*api.GlobalState, api.SubCmdIdx, api.Cmd),
	preSubCmdCb func(s *api.GlobalState, idx api.SubCmdIdx, cmd api.Cmd, subCmdRef interface{}),
	postSubCmdCb func(s *api.GlobalState, idx api.SubCmdIdx, cmd api.Cmd, subCmdRef interface{})) error {

	ctx = status.Start(ctx, "Sync.MutateWithSubcommands")
	defer status.Finish(ctx)

	// This is where we want to handle sub-states
	// This involves transforming the tree for the given Indices, and
	//   then mutating that.
	rc, err := capture.ResolveGraphicsFromPath(ctx, c)
	if err != nil {
		return err
	}
	s := rc.NewState(ctx)

	if cmds == nil {
		cmds = rc.Commands
	}

	return api.ForeachCmd(ctx, cmds, false, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		if sync, ok := cmd.API().(SynchronizedAPI); ok {
			if err := sync.MutateSubcommands(ctx, id, cmd, s, preSubCmdCb, postSubCmdCb); err != nil {
				return err
			}
		} else {
			if err := cmd.Mutate(ctx, id, s, nil, nil); err != nil {
				return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
			}
		}
		if postCmdCb != nil {
			postCmdCb(s, api.SubCmdIdx{uint64(id)}, cmd)
		}
		return nil
	})
}
