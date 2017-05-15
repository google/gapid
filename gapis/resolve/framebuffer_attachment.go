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
	"fmt"
	"sort"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay/devices"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// FramebufferAttachment resolves the specified framebuffer attachment at the
// specified point in a capture.
func FramebufferAttachment(
	ctx context.Context,
	device *path.Device,
	after *path.Command,
	attachment gfxapi.FramebufferAttachment,
	settings *service.RenderSettings,
	hints *service.UsageHints,
) (*path.ImageInfo, error) {
	if device == nil {
		devices, err := devices.ForReplay(ctx, after.Capture)
		if err != nil {
			return nil, err
		}
		if len(devices) == 0 {
			return nil, fmt.Errorf("No compatible replay devices found")
		}
		device = devices[0]
	}

	id, err := database.Store(ctx, &FramebufferAttachmentResolvable{
		device,
		after,
		attachment,
		settings,
		hints,
	})
	if err != nil {
		return nil, err
	}
	return path.NewImageInfo(id), nil
}

// FramebufferAttachmentInfo returns the framebuffer dimensions and format
// after a given atom in the given capture, atom and attachment.
// The first call to getFramebufferInfo for a given capture/context
// will trigger a computation for all atoms of this capture, which will be
// cached to the database for subsequent calls, regardless of the given atom.
func FramebufferAttachmentInfo(ctx context.Context, after *path.Command, att gfxapi.FramebufferAttachment) (framebufferAttachmentInfo, error) {
	changes, err := FramebufferChanges(ctx, path.FindCapture(after))
	if err != nil {
		return framebufferAttachmentInfo{}, err
	}
	atomIdx := after.Index[0]
	if len(after.Index) > 1 {
		return framebufferAttachmentInfo{}, fmt.Errorf("Subcommands currently not supported") // TODO: Subcommands
	}
	info, err := changes.attachments[att].after(ctx, atomIdx)
	if err != nil {
		return framebufferAttachmentInfo{}, err
	}
	if !info.valid {
		return framebufferAttachmentInfo{}, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}
	}
	return info, nil
}

// Resolve implements the database.Resolver interface.
func (r *FramebufferAttachmentResolvable) Resolve(ctx context.Context) (interface{}, error) {
	fbInfo, err := FramebufferAttachmentInfo(ctx, r.After, r.Attachment)
	if err != nil {
		return nil, err
	}
	width, height := uniformScale(fbInfo.width, fbInfo.height, r.Settings.MaxWidth, r.Settings.MaxHeight)

	data, err := database.Store(ctx, &FramebufferAttachmentDataResolvable{
		Device:        r.Device,
		After:         r.After,
		Width:         width,
		Height:        height,
		Attachment:    r.Attachment,
		WireframeMode: r.Settings.WireframeMode,
		Hints:         r.Hints,
		ImageFormat:   fbInfo.format,
	})
	if err != nil {
		return nil, err
	}

	return &image.Info2D{
		Width:  width,
		Height: height,
		Format: fbInfo.format,
		Data:   image.NewID(data),
	}, nil
}

func uniformScale(width, height, maxWidth, maxHeight uint32) (w, h uint32) {
	w, h = width, height
	scaleX, scaleY := float32(w)/float32(maxWidth), float32(h)/float32(maxHeight)
	if scaleX > 1.0 || scaleY > 1.0 {
		if scaleX > scaleY {
			w, h = uint32(float32(w)/scaleX), uint32(float32(h)/scaleX)
		} else {
			w, h = uint32(float32(w)/scaleY), uint32(float32(h)/scaleY)
		}
	}
	return w, h
}

// framebufferAttachmentChanges describes the list of changes to a single
// attachment over the span of the entire capture.
type framebufferAttachmentChanges struct {
	changes []framebufferAttachmentInfo
}

// framebufferAttachmentInfo describes the dimensions and format of a
// framebuffer attachment.
type framebufferAttachmentInfo struct {
	after  uint64 // index of the last atom to change the attachment.
	width  uint32
	height uint32
	format *image.Format
	valid  bool
}

func (c framebufferAttachmentChanges) after(ctx context.Context, i uint64) (framebufferAttachmentInfo, error) {
	idx := sort.Search(len(c.changes), func(x int) bool { return c.changes[x].after > i }) - 1

	if idx < 0 {
		log.W(ctx, "No dimension records found after atom %d. FB dimension records = %d", i, len(c.changes))
		return framebufferAttachmentInfo{}, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}
	}

	return c.changes[idx], nil
}

func (c framebufferAttachmentChanges) last() framebufferAttachmentInfo {
	if count := len(c.changes); count > 0 {
		return c.changes[count-1]
	}
	return framebufferAttachmentInfo{}
}

var allFramebufferAttachments = []gfxapi.FramebufferAttachment{
	gfxapi.FramebufferAttachment_Depth,
	gfxapi.FramebufferAttachment_Stencil,
	gfxapi.FramebufferAttachment_Color0,
	gfxapi.FramebufferAttachment_Color1,
	gfxapi.FramebufferAttachment_Color2,
	gfxapi.FramebufferAttachment_Color3,
}
