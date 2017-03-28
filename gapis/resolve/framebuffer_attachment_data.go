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
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
)

// Resolve implements the database.Resolver interface.
func (r *FramebufferAttachmentDataResolvable) Resolve(ctx context.Context) (interface{}, error) {
	intent := replay.Intent{
		Device:  r.Device,
		Capture: r.After.Commands.Capture,
	}

	after, err := Command(ctx, r.After)
	if err != nil {
		return nil, err
	}

	api := after.API()
	if api == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}
	}

	query, ok := api.(replay.QueryFramebufferAttachment)
	if !ok {
		log.E(ctx, "API %s does not implement FramebufferAttachmentDataResolvable", api.Name())
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}
	}

	wireframeMode := replay.WireframeMode_None
	switch r.WireframeMode {
	case service.WireframeMode_None:
	case service.WireframeMode_All:
		wireframeMode = replay.WireframeMode_All
	case service.WireframeMode_Overlay:
		wireframeMode = replay.WireframeMode_Overlay
	default:
		return nil, &service.ErrInvalidArgument{Reason: messages.ErrInvalidEnumValue(wireframeMode, "WireframeMode")}
	}

	mgr := replay.GetManager(ctx)

	res, err := query.QueryFramebufferAttachment(
		ctx,
		intent,
		mgr,
		atom.ID(r.After.Index),
		r.Width,
		r.Height,
		r.Attachment,
		wireframeMode,
		r.Hints,
	)
	if err != nil {
		if _, ok := err.(*service.ErrDataUnavailable); ok {
			return nil, err
		}
		return nil, log.Err(ctx, err, "Couldn't get framebuffer attachment")
	}

	return res.Data, nil
}
