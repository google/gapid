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
func (s *State) Root(ctx context.Context, p *path.State) (path.Node, error) {
	return nil, nil
}

func (c *State) preMutate(ctx context.Context, s *api.GlobalState, cmd api.Cmd) error {
	return nil
}

type CustomState struct{}

func (API) QueryFramebufferAttachment(
	ctx context.Context,
	intent replay.Intent,
	mgr *replay.Manager,
	after []uint64,
	width, height uint32,
	attachment api.FramebufferAttachment,
	framebufferIndex uint32,
	wireframeMode replay.WireframeMode,
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
		wireframeMode,
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
	attachment api.FramebufferAttachment) (width, height, index uint32, format *image.Format, err error) {

	fb, err := getFramebuffer(ctx, api.CmdID(after[0]))
	if err != nil {
		return 0, 0, 0, nil, err
	}
	return gles.GetFramebufferAttachmentInfoByID(state, thread, attachment, fb)
}

// Context returns the active context for the given state and thread.
func (API) Context(s *api.GlobalState, thread uint64) api.Context {
	return gles.API{}.Context(s, thread)
}

// Mesh implements the api.MeshProvider interface.
func (API) Mesh(ctx context.Context, o interface{}, p *path.Mesh) (*api.Mesh, error) {
	return nil, nil
}

var _ sync.SynchronizedAPI = API{}

// GetTerminator returns a transform that will allow the given capture to be terminated
// after a atom
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

// MutateSubcommands mutates the given Cmd and calls callbacks for subcommands
// called before and after executing each subcommand callback.
func (API) MutateSubcommands(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.GlobalState,
	preSubCmdCallback func(*api.GlobalState, api.SubCmdIdx, api.Cmd),
	postSubCmdCallback func(*api.GlobalState, api.SubCmdIdx, api.Cmd)) error {

	return nil
}
