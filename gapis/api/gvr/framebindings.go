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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
)

func getFramebuffer(ctx context.Context, id api.CmdID) (gles.FramebufferId, error) {
	c := capture.Get(ctx)
	cmds, err := resolve.Cmds(ctx, c)
	if err != nil {
		return 0, err
	}
	switch cmds[id].(type) {
	case *Gvr_frame_submit:
		bindings, err := getFrameBindings(ctx, c)
		if err != nil {
			return 0, err
		}
		return bindings.submitBuffer[id], nil
	}
	return 0, nil
}

type frameBindings struct {
	submitBuffer map[api.CmdID]gles.FramebufferId
}

func getFrameBindings(ctx context.Context, c *path.Capture) (*frameBindings, error) {
	obj, err := database.Build(ctx, &FrameBindingsResolvable{c})
	if err != nil {
		return nil, err
	}
	return obj.(*frameBindings), nil
}

// Resolve implements the database.Resolver interface.
func (r *FrameBindingsResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Capture)

	cmds, err := resolve.Cmds(ctx, r.Capture)
	if err != nil {
		return nil, err
	}

	s, err := capture.NewState(ctx)
	if err != nil {
		return nil, err
	}

	out := &frameBindings{
		submitBuffer: map[api.CmdID]gles.FramebufferId{},
	}
	frameToBuffer := map[GvrFrameᵖ]gles.FramebufferId{}

	api.ForeachCmd(ctx, cmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		switch cmd := cmd.(type) {
		case *Gvr_frame_submit:
			// Annoyingly gvr_frame_submit takes a pointer to the frame pointer,
			// just so it can be nullified before returning. To avoid another
			// state mutation just to get the pointer, cache them here.
			cmd.extras.Observations().ApplyReads(s.Memory.ApplicationPool())
			frame, err := cmd.Frame.Read(ctx, cmd, s, nil)
			if err != nil {
				return err
			}
			out.submitBuffer[id] = frameToBuffer[frame]
		case *Gvr_frame_get_framebuffer_object:
			frameToBuffer[GvrFrameᵖ(cmd.Frame)] = gles.FramebufferId(cmd.Result)
		case *gles.GlBindFramebuffer:
			if callerID := cmd.Caller(); callerID != api.CmdNoID {
				switch caller := cmds[callerID].(type) {
				case *Gvr_frame_bind_buffer:
					if caller.Index == 0 { // Only consider the 0'th frame index.
						frameToBuffer[caller.Frame] = cmd.Framebuffer
					}
				}
			}
		}
		cmd.Mutate(ctx, id, s, nil)
		return nil
	})

	return out, nil
}
