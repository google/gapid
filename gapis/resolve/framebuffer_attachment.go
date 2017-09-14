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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
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
	attachment api.FramebufferAttachment,
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

	// Check the command is valid. If we don't do it here, we'll likely get an
	// error deep in the bowels of the framebuffer data resolve.
	if _, err := Cmd(ctx, after); err != nil {
		return nil, err
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

// framebufferAttachmentInfo returns the framebuffer dimensions and format
// after a given command in the given capture, command and attachment.
// The first call to getFramebufferInfo for a given capture/context
// will trigger a computation for all commands of this capture, which will be
// cached to the database for subsequent calls, regardless of the given command.
func getFramebufferAttachmentInfo(ctx context.Context, after *path.Command, att api.FramebufferAttachment) (framebufferAttachmentInfo, error) {
	changes, err := FramebufferChanges(ctx, after.Capture)
	if err != nil {
		return framebufferAttachmentInfo{}, err
	}
	info, err := changes.attachments[att].after(ctx, api.SubCmdIdx(after.Indices))
	if err != nil {
		return framebufferAttachmentInfo{}, err
	}
	if info.err != nil {
		log.W(ctx, "Framebuffer error after %v: %v", after, info.err)
		return framebufferAttachmentInfo{}, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}
	}
	return info, nil
}

// Resolve implements the database.Resolver interface.
func (r *FramebufferAttachmentResolvable) Resolve(ctx context.Context) (interface{}, error) {
	fbInfo, err := getFramebufferAttachmentInfo(ctx, r.After, r.Attachment)
	if err != nil {
		return nil, err
	}
	width, height := uniformScale(fbInfo.width, fbInfo.height, r.Settings.MaxWidth, r.Settings.MaxHeight)

	id, err := database.Store(ctx, &FramebufferAttachmentBytesResolvable{
		Device:           r.Device,
		After:            r.After,
		Width:            width,
		Height:           height,
		Attachment:       r.Attachment,
		FramebufferIndex: fbInfo.index,
		WireframeMode:    r.Settings.WireframeMode,
		Hints:            r.Hints,
		ImageFormat:      fbInfo.format,
	})
	if err != nil {
		return nil, err
	}

	return &image.Info{
		Width:  width,
		Height: height,
		Depth:  1,
		Format: fbInfo.format,
		Bytes:  image.NewID(id),
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
	after  api.SubCmdIdx // index of the last command to change the attachment.
	width  uint32
	height uint32
	index  uint32 // The api-specific attachment index
	format *image.Format
	err    error
}

func (f framebufferAttachmentInfo) equal(o framebufferAttachmentInfo) bool {
	fe := (f.format == nil && o.format == nil) || (f.format != nil && o.format != nil && f.format.Name == o.format.Name)
	if (f.err == nil) != (o.err == nil) {
		return false
	}
	if f.err == nil {
		return fe && f.width == o.width && f.height == o.height && f.index == o.index
	}
	return f.err.Error() == o.err.Error()
}

func (c framebufferAttachmentChanges) after(ctx context.Context, i api.SubCmdIdx) (framebufferAttachmentInfo, error) {
	idx := sort.Search(len(c.changes), func(x int) bool { return i.LessThan(c.changes[x].after) }) - 1

	if idx < 0 {
		log.W(ctx, "No dimension records found after command %d. FB dimension records = %d", i, len(c.changes))
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

var allFramebufferAttachments = []api.FramebufferAttachment{
	api.FramebufferAttachment_Depth,
	api.FramebufferAttachment_Stencil,
	api.FramebufferAttachment_Color0,
	api.FramebufferAttachment_Color1,
	api.FramebufferAttachment_Color2,
	api.FramebufferAttachment_Color3,
}
