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

package resolve

import (
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Resolve implements the database.Resolver interface.
func (r *FramebufferAttachmentBytesResolvable) Resolve(ctx context.Context) (interface{}, error) {
	c := path.FindCapture(r.After)
	ctx = capture.Put(ctx, c)

	intent := replay.Intent{
		Device:  r.ReplaySettings.Device,
		Capture: c,
	}

	after, err := Cmd(ctx, r.After)
	if err != nil {
		return nil, err
	}

	api := after.API()
	if api == nil {
		log.W(ctx, "No API!")
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}
	}

	query, ok := api.(replay.QueryFramebufferAttachment)
	if !ok {
		log.E(ctx, "API %s does not implement QueryFramebufferAttachment", api.Name())
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}
	}

	drawMode := replay.DrawMode_NORMAL
	switch r.DrawMode {
	case service.DrawMode_NORMAL:
	case service.DrawMode_WIREFRAME_ALL:
		drawMode = replay.DrawMode_WIREFRAME_ALL
	case service.DrawMode_WIREFRAME_OVERLAY:
		drawMode = replay.DrawMode_WIREFRAME_OVERLAY
	case service.DrawMode_OVERDRAW:
		drawMode = replay.DrawMode_OVERDRAW
	default:
		return nil, &service.ErrInvalidArgument{Reason: messages.ErrInvalidEnum(drawMode)}
	}

	mgr := replay.GetManager(ctx)

	res, err := query.QueryFramebufferAttachment(
		ctx,
		intent,
		mgr,
		r.After.Indices,
		r.Width,
		r.Height,
		r.Attachment,
		r.FramebufferIndex,
		drawMode,
		r.ReplaySettings.DisableReplayOptimization,
		r.Hints,
	)
	if err != nil {
		if _, ok := err.(*service.ErrDataUnavailable); ok {
			return nil, err
		}
		return nil, log.Err(ctx, err, "Couldn't get framebuffer attachment")
	}

	res, err = res.Convert(r.ImageFormat)
	if err != nil {
		return nil, log.Err(ctx, err, "Couldn't get framebuffer attachment")
	}

	return res.Bytes, nil
}
