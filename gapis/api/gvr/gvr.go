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
	"fmt"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
)

func (c *State) preMutate(ctx context.Context, s *api.State, cmd api.Cmd) error {
	return nil
}

type CustomState struct{}

// GetFramebufferAttachmentInfo returns the width, height and format of the specified framebuffer attachment.
func (API) GetFramebufferAttachmentInfo(state *api.State, thread uint64, attachment api.FramebufferAttachment) (width, height, index uint32, format *image.Format, err error) {
	return 0, 0, 0, nil, fmt.Errorf("GVR does not support framebuffers")
}

// Context returns the active context for the given state and thread.
func (API) Context(s *api.State, thread uint64) api.Context {
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
	cmds, err := resolve.Cmds(ctx, c)
	if err != nil {
		return err
	}
	for i, c := range cmds {
		if caller := c.Caller(); caller != api.CmdNoID {
			id := api.CmdID(i)
			d.Hidden.Add(id)
			if d.Hidden.Contains(caller) {
				continue // Most likely a sub-sub-command, which we don't currently support.
			}
			l := d.SubcommandReferences[caller]
			idx := api.SubCmdIdx{uint64(len(l))}
			l = append(l, sync.SubcommandReference{
				Index:         idx,
				GeneratingCmd: id,
			})
			d.SubcommandReferences[caller] = l
			d.SubcommandGroups[caller] = []api.SubCmdIdx{idx}
		}
	}
	return nil
}

// MutateSubcommands mutates the given Cmd and calls callbacks for subcommands
// called before and after executing each subcommand callback.
func (API) MutateSubcommands(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.State,
	preSubCmdCallback func(*api.State, api.SubCmdIdx, api.Cmd),
	postSubCmdCallback func(*api.State, api.SubCmdIdx, api.Cmd)) error {

	return nil
}
