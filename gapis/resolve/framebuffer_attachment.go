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
	"github.com/google/gapid/core/stream/fmts"
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
	replaySettings *service.ReplaySettings,
	after *path.Command,
	attachment api.FramebufferAttachment,
	settings *service.RenderSettings,
	hints *service.UsageHints,
	config *path.ResolveConfig,
) (*path.ImageInfo, error) {

	if replaySettings.Device == nil {
		devices, err := devices.ForReplay(ctx, after.Capture)
		if err != nil {
			return nil, err
		}
		if len(devices) == 0 {
			return nil, fmt.Errorf("No compatible replay devices found")
		}
		replaySettings.Device = devices[0]
	}

	// Check the command is valid. If we don't do it here, we'll likely get an
	// error deep in the bowels of the framebuffer data resolve.
	if _, err := Cmd(ctx, after, config); err != nil {
		return nil, err
	}

	id, err := database.Store(ctx, &FramebufferAttachmentResolvable{
		ReplaySettings: replaySettings,
		After:          after,
		Attachment:     attachment,
		Settings:       settings,
		Hints:          hints,
		Config:         config,
	})

	if err != nil {
		return nil, err
	}
	return path.NewImageInfo(id), nil
}

// Resolve implements the database.Resolver interface.
func (r *FramebufferAttachmentResolvable) Resolve(ctx context.Context) (interface{}, error) {
	changes, err := FramebufferChanges(ctx, r.After.Capture, r.Config)
	if err != nil {
		return FramebufferAttachmentInfo{}, err
	}

	fbInfo, err := changes.Get(ctx, r.After, r.Attachment)
	if err != nil {
		return nil, err
	}

	width, height := uniformScale(fbInfo.Width, fbInfo.Height, r.Settings.MaxWidth, r.Settings.MaxHeight)
	if !fbInfo.CanResize {
		width, height = fbInfo.Width, fbInfo.Height
	}

	format := fbInfo.Format
	if r.Settings.DrawMode == service.DrawMode_OVERDRAW {
		format = image.NewUncompressed("Count_U8", fmts.Count_U8)
	}

	id, err := database.Store(ctx, &FramebufferAttachmentBytesResolvable{
		ReplaySettings:   r.ReplaySettings,
		After:            r.After,
		Width:            width,
		Height:           height,
		Attachment:       r.Attachment,
		FramebufferIndex: fbInfo.Index,
		DrawMode:         r.Settings.DrawMode,
		Hints:            r.Hints,
		ImageFormat:      format,
	})
	if err != nil {
		return nil, err
	}

	return &image.Info{
		Width:  width,
		Height: height,
		Depth:  1,
		Format: format,
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
	changes []FramebufferAttachmentInfo
}

// FramebufferAttachmentInfo describes the dimensions and format of a
// framebuffer attachment.
type FramebufferAttachmentInfo struct {
	// After is the index of the last command to change the attachment.
	After api.SubCmdIdx

	// Width of the framebuffer attachment in pixels.
	Width uint32

	// Height of the framebuffer attachment in pixels.
	Height uint32

	// Can this image be resized in the server
	CanResize bool

	// Index of the api-specific attachment.
	Index uint32

	// Format of the attachment.
	Format *image.Format

	// The error returned by the API. If this is non-null then all other fields
	// may contain undefined values.
	Err error
}

func (f FramebufferAttachmentInfo) equal(o FramebufferAttachmentInfo) bool {
	fe := (f.Format == nil && o.Format == nil) || (f.Format != nil && o.Format != nil && f.Format.Name == o.Format.Name)
	if (f.Err == nil) != (o.Err == nil) {
		return false
	}
	if f.Err == nil {
		return fe && f.Width == o.Width && f.Height == o.Height && f.Index == o.Index && f.CanResize == o.CanResize
	}
	return f.Err.Error() == o.Err.Error()
}

func (c framebufferAttachmentChanges) after(ctx context.Context, i api.SubCmdIdx) (FramebufferAttachmentInfo, error) {
	idx := sort.Search(len(c.changes), func(x int) bool { return i.LessThan(c.changes[x].After) }) - 1

	if idx < 0 {
		log.W(ctx, "No dimension records found after command %d. FB dimension records = %d", i, len(c.changes))
		return FramebufferAttachmentInfo{}, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}
	}

	return c.changes[idx], nil
}

func (c framebufferAttachmentChanges) last() FramebufferAttachmentInfo {
	if count := len(c.changes); count > 0 {
		return c.changes[count-1]
	}
	return FramebufferAttachmentInfo{}
}

var allFramebufferAttachments = []api.FramebufferAttachment{
	api.FramebufferAttachment_Depth,
	api.FramebufferAttachment_Stencil,
	api.FramebufferAttachment_Color0,
	api.FramebufferAttachment_Color1,
	api.FramebufferAttachment_Color2,
	api.FramebufferAttachment_Color3,
}
