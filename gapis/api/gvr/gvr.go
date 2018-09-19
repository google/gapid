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

package gvr

import (
	"context"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

var _ = replay.QueryFramebufferAttachment(API{})

// Root returns the path to the root of the state to display. It can vary based
// on filtering mode. Returning nil, nil indicates there is no state to show at
// this point in the capture.
func (s *State) Root(ctx context.Context, p *path.State, r *path.ResolveConfig) (path.Node, error) {
	return nil, nil
}

// SetupInitialState sanitizes deserialized state to make it valid.
// It can fill in any derived data which we choose not to serialize,
// or it can apply backward-compatibility fixes for older traces.
func (State) SetupInitialState(ctx context.Context) {}

func (s *State) preMutate(ctx context.Context, g *api.GlobalState, cmd api.Cmd) error {
	return nil
}

type customState struct{}

func (customState) init(*State) {}

// RebuildState is a no-op to conform to the api.API interface.
func (API) RebuildState(ctx context.Context, g *api.GlobalState) ([]api.Cmd, interval.U64RangeList) {
	return nil, nil
}

func (API) QueryFramebufferAttachment(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	after []uint64,
	width, height uint32,
	attachment api.FramebufferAttachment,
	framebufferIndex uint32,
	drawMode service.DrawMode,
	disableReplayOptimization bool,
	displayToSurface bool,
	hints *service.UsageHints) (*image.Data, error) {

	if framebufferIndex == 0 {
		fb, err := getFramebuffer(ctx, api.CmdID(after[0]))
		if err != nil {
			return nil, err
		}
		framebufferIndex = uint32(fb)
	}
	return gles.API{}.QueryFramebufferAttachment(
		ctx,
		intent,
		mgr,
		after,
		width, height,
		attachment,
		framebufferIndex,
		drawMode,
		disableReplayOptimization,
		displayToSurface,
		hints,
	)
}

// GetFramebufferAttachmentInfo returns the width, height and format of the
// specified framebuffer attachment.
func (API) GetFramebufferAttachmentInfo(
	ctx context.Context,
	after []uint64,
	state *api.GlobalState,
	thread uint64,
	attachment api.FramebufferAttachment) (inf api.FramebufferAttachmentInfo, err error) {

	fb, err := getFramebuffer(ctx, api.CmdID(after[0]))
	if err != nil {
		return api.FramebufferAttachmentInfo{}, err
	}
	return gles.GetFramebufferAttachmentInfoByID(state, thread, attachment, fb)
}

// Context returns the active context for the given state and thread.
func (API) Context(ctx context.Context, s *api.GlobalState, thread uint64) api.Context {
	return gles.API{}.Context(ctx, s, thread)
}

// Mesh implements the api.MeshProvider interface.
func (API) Mesh(ctx context.Context, o interface{}, p *path.Mesh, r *path.ResolveConfig) (*api.Mesh, error) {
	return nil, nil
}

var _ sync.SynchronizedAPI = API{}

// GetTerminator returns a transform that will allow the given capture to be terminated
// after a command
func (API) GetTerminator(ctx context.Context, c *path.Capture) (transform.Terminator, error) {
	return nil, nil
}

// ResolveSynchronization resolve all of the synchronization information for
// the given API
func (API) ResolveSynchronization(ctx context.Context, d *sync.Data, c *path.Capture) error {
	return nil
}

// FlattenSubcommandIdx flattens grouped ids to their flattened linear ids if possible.
func (API) FlattenSubcommandIdx(idx api.SubCmdIdx, data *sync.Data, unused bool) (api.CmdID, bool) {
	sg, ok := data.SubcommandReferences[api.CmdID(idx[0])]
	if !ok {
		return api.CmdID(0), false
	}
	for _, v := range sg {
		if v.Index.Equals(idx[1:]) {
			if v.IsCallerGroup {
				return v.GeneratingCmd, true
			}
			break
		}
	}
	return api.CmdID(0), false
}

// IsTrivialTerminator returns true if the terminator is just stopping at the given index
func (API) IsTrivialTerminator(ctx context.Context, p *path.Capture, command api.SubCmdIdx) (bool, error) {
	return true, nil
}

// RecoverMidExecutionCommand returns a virtual command, used to describe the
// a subcommand that was created before the start of the trace
// GVR has no subcommands of this type, so this should never be called
func (API) RecoverMidExecutionCommand(ctx context.Context, c *path.Capture, i interface{}) (api.Cmd, error) {
	return nil, sync.NoMECSubcommandsError{}
}

// MutateSubcommands mutates the given Cmd and calls callbacks for subcommands
// called before and after executing each subcommand callback.
func (API) MutateSubcommands(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.GlobalState,
	preSubCmdCallback func(*api.GlobalState, api.SubCmdIdx, api.Cmd),
	postSubCmdCallback func(*api.GlobalState, api.SubCmdIdx, api.Cmd)) error {

	return nil
}
