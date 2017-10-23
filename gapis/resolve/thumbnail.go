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

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Thumbnail resolves and returns the thumbnail from the path p.
func Thumbnail(ctx context.Context, p *path.Thumbnail) (*image.Info, error) {
	switch parent := p.Parent().(type) {
	case *path.Command:
		return CommandThumbnail(ctx, p.DesiredMaxWidth, p.DesiredMaxHeight, p.DesiredFormat, parent)
	case *path.CommandTreeNode:
		return CommandTreeNodeThumbnail(ctx, p.DesiredMaxWidth, p.DesiredMaxHeight, p.DesiredFormat, parent)
	case *path.ResourceData:
		return ResourceDataThumbnail(ctx, p.DesiredMaxWidth, p.DesiredMaxHeight, p.DesiredFormat, parent)
	default:
		return nil, fmt.Errorf("Unexpected Thumbnail parent %T", parent)
	}
}

// CommandThumbnail resolves and returns the thumbnail for the framebuffer at p.
func CommandThumbnail(ctx context.Context, w, h uint32, f *image.Format, p *path.Command) (*image.Info, error) {
	imageInfoPath, err := FramebufferAttachment(ctx,
		nil, // device
		p,
		api.FramebufferAttachment_Color0,
		&service.RenderSettings{
			MaxWidth:      w,
			MaxHeight:     h,
			WireframeMode: service.WireframeMode_None,
		},
		&service.UsageHints{
			Preview: true,
		},
	)
	if err != nil {
		return nil, err
	}

	var boxedImageInfo interface{}
	if f != nil {
		boxedImageInfo, err = Get(ctx, imageInfoPath.As(f).Path())
	} else {
		boxedImageInfo, err = Get(ctx, imageInfoPath.Path())
	}
	if err != nil {
		return nil, err
	}

	return boxedImageInfo.(*image.Info), nil
}

// CommandTreeNodeThumbnail resolves and returns the thumbnail for the framebuffer at p.
func CommandTreeNodeThumbnail(ctx context.Context, w, h uint32, f *image.Format, p *path.CommandTreeNode) (*image.Info, error) {
	boxedCmdTree, err := database.Resolve(ctx, p.Tree.ID())
	if err != nil {
		return nil, err
	}

	cmdTree := boxedCmdTree.(*commandTree)

	item, _ := cmdTree.index(p.Indices)
	switch item := item.(type) {
	case api.CmdIDGroup:
		thumbnail := item.Range.Last()
		if userData, ok := item.UserData.(*CmdGroupData); ok {
			thumbnail = userData.Representation
		}
		return CommandThumbnail(ctx, w, h, f, cmdTree.path.Capture.Command(uint64(thumbnail)))
	case api.SubCmdIdx:
		return CommandThumbnail(ctx, w, h, f, cmdTree.path.Capture.Command(uint64(item[0]), item[1:]...))
	case api.SubCmdRoot:
		return CommandThumbnail(ctx, w, h, f, cmdTree.path.Capture.Command(uint64(item.Id[0]), item.Id[1:]...))
	default:
		panic(fmt.Errorf("Unexpected type: %T", item))
	}
}

// ResourceDataThumbnail resolves and returns the thumbnail for the resource at p.
func ResourceDataThumbnail(ctx context.Context, w, h uint32, f *image.Format, p *path.ResourceData) (*image.Info, error) {
	obj, err := ResolveInternal(ctx, p)
	if err != nil {
		return nil, err
	}

	t, ok := obj.(image.Thumbnailer)
	if !ok {
		return nil, fmt.Errorf("Type %T does not support thumbnailing", obj)
	}

	img, err := t.Thumbnail(ctx, w, h, 1)
	if err != nil {
		return nil, err
	}

	if img == nil || img.Format == nil || img.Bytes == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData("")}
	}

	if f != nil {
		// Convert the image to the desired format.
		if img.Format.Key() != f.Key() {
			img, err = img.Convert(ctx, f)
			if err != nil {
				return nil, err
			}
		}
	}

	// Image format supports resizing. See if the image should be.
	scaleX, scaleY := float32(1), float32(1)
	if w > 0 && img.Width > w {
		scaleX = float32(w) / float32(img.Width)
	}
	if h > 0 && img.Height > h {
		scaleY = float32(h) / float32(img.Height)
	}
	scale := scaleX // scale := min(scaleX, scaleY)
	if scale > scaleY {
		scale = scaleY
	}

	targetWidth := uint32(float32(img.Width) * scale)
	targetHeight := uint32(float32(img.Height) * scale)

	// Prevent scaling to zero size.
	if targetWidth == 0 {
		targetWidth = 1
	}
	if targetHeight == 0 {
		targetHeight = 1
	}

	if targetWidth == img.Width && targetHeight == img.Height {
		// Image is already at requested target size.
		return img, err
	}

	return img.Resize(ctx, targetWidth, targetHeight, 1)
}
