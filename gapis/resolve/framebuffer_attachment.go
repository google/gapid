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

func FramebufferAttachments(ctx context.Context, p *path.FramebufferAttachments, r *path.ResolveConfig) (interface{}, error) {
	obj, err := database.Build(ctx, &FramebufferAttachmentsResolvable{Path: p, Config: r})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (r *FramebufferAttachmentsResolvable) Resolve(ctx context.Context) (interface{}, error) {
	changes, err := FramebufferChanges(ctx, r.Path.After.Capture, r.Config)
	if err != nil {
		return []*service.FramebufferAttachment{}, err
	}

	out := []*service.FramebufferAttachment{}
	for _, att := range changes.attachments {
		info, err := att.after(ctx, api.SubCmdIdx(r.Path.After.Indices))
		if err != nil {
			return []*service.FramebufferAttachment{}, err
		}

		if info.Err == nil {
			out = append(out, &service.FramebufferAttachment{
				Index: info.Index,
				Type:  info.Type,
				Label: fmt.Sprintf("%d: %s", info.Index, typeToString(info.Type)),
			})
		}
	}
	return &service.FramebufferAttachments{Attachments: out}, nil
}

func typeToString(t api.FramebufferAttachmentType) string {
	switch t {
	case api.FramebufferAttachmentType_OutputColor:
		return "Color"
	case api.FramebufferAttachmentType_OutputDepth:
		return "Depth"
	case api.FramebufferAttachmentType_InputColor:
		return "Input Color"
	case api.FramebufferAttachmentType_InputDepth:
		return "Input Depth"
	default:
		return "Unknown"
	}
}

// FramebufferAttachment resolves the specified framebuffer attachment at the
// specified point in a capture.
func FramebufferAttachment(ctx context.Context, p *path.FramebufferAttachment, r *path.ResolveConfig) (interface{}, error) {
	if r.ReplayDevice == nil {
		devices, err := devices.ForReplay(ctx, p.After.Capture)
		if err != nil {
			return nil, err
		}
		if len(devices) == 0 {
			return nil, fmt.Errorf("No compatible devices found")
		}
		r.ReplayDevice = devices[0]
	}

	// Check the command is valid. If we don't do it here, we'll likely get an
	// error deep in the bowels of the framebuffer data resolve.
	if _, err := Cmd(ctx, p.After, r); err != nil {
		return nil, err
	}

	id, err := database.Store(ctx, &FramebufferAttachmentResolvable{
		After:      p.After,
		Attachment: p.Index,
		Settings:   p.RenderSettings,
		Hints:      p.Hints,
		Config:     r,
	})

	if err != nil {
		return nil, err
	}
	return &service.FramebufferAttachment{
		Index:     p.Index,
		Type:      api.FramebufferAttachmentType_OutputColor,
		ImageInfo: path.NewImageInfo(id),
		Label:     fmt.Sprintf("Attachment %d", p.Index),
	}, nil
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
	if r.Settings.DrawMode == path.DrawMode_OVERDRAW {
		format = image.NewUncompressed("Count_U8", fmts.Count_U8)
	}

	id, err := database.Store(ctx, &FramebufferAttachmentBytesResolvable{
		After:            r.After,
		Width:            width,
		Height:           height,
		Attachment:       fbInfo.Type,
		FramebufferIndex: fbInfo.Index,
		Settings:         r.Settings,
		Hints:            r.Hints,
		ImageFormat:      format,
		Config:           r.Config,
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

	// Type of the attachment.
	Type api.FramebufferAttachmentType

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
		return fe && f.Width == o.Width && f.Height == o.Height && f.Index == o.Index && f.CanResize == o.CanResize && f.Type == o.Type
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
