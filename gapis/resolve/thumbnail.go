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
func Thumbnail(ctx context.Context, p *path.Thumbnail, r *path.ResolveConfig) (*image.Info, error) {
	switch parent := p.Parent().(type) {
	case *path.Command:
		return CommandThumbnail(ctx, p.DesiredMaxWidth, p.DesiredMaxHeight, p.DesiredFormat, p.DisableOptimization, parent, r)
	case *path.CommandTreeNode:
		return CommandTreeNodeThumbnail(ctx, p.DesiredMaxWidth, p.DesiredMaxHeight, p.DesiredFormat, p.DisableOptimization, parent, r)
	case *path.ResourceData:
		return ResourceDataThumbnail(ctx, p.DesiredMaxWidth, p.DesiredMaxHeight, p.DesiredFormat, parent, r)
	default:
		return nil, fmt.Errorf("Unexpected Thumbnail parent %T", parent)
	}
}

// CommandThumbnail resolves and returns the thumbnail for the framebuffer at p.
func CommandThumbnail(
	ctx context.Context,
	w, h uint32,
	f *image.Format,
	noOpt bool,
	p *path.Command,
	r *path.ResolveConfig) (*image.Info, error) {

	fbaList, err := FramebufferAttachments(ctx, &path.FramebufferAttachments{After: p}, r)
	if err != nil {
		return nil, err
	}

	var fbaInfo *service.FramebufferAttachment = nil
	for _, fba := range fbaList.(*service.FramebufferAttachments).GetAttachments() {
		if fba.GetType() == api.FramebufferAttachmentType_OutputColor {
			fbaInfo = fba
			break
		}
	}

	if fbaInfo == nil {
		for _, fba := range fbaList.(*service.FramebufferAttachments).GetAttachments() {
			if fba.GetType() == api.FramebufferAttachmentType_OutputDepth {
				fbaInfo = fba
				break
			}
		}
	}

	if fbaInfo == nil {
		return nil, fmt.Errorf("No viable attachment exists for thumbnails")
	}

	fbaPath := &path.FramebufferAttachment{
		After: p,
		Index: fbaInfo.GetIndex(),
		RenderSettings: &path.RenderSettings{
			MaxWidth:                  w,
			MaxHeight:                 h,
			DrawMode:                  path.DrawMode_NORMAL,
			DisableReplayOptimization: noOpt,
		},
		Hints: &path.UsageHints{
			Preview: true,
		},
	}

	imageInfoPath, err := FramebufferAttachment(ctx, fbaPath, r)
	if err != nil {
		return nil, err
	}

	var boxedImageInfo interface{}
	if f != nil {
		boxedImageInfo, err = Get(ctx, imageInfoPath.(*service.FramebufferAttachment).GetImageInfo().As(f).Path(), r)
	} else {
		boxedImageInfo, err = Get(ctx, imageInfoPath.(*service.FramebufferAttachment).GetImageInfo().Path(), r)
	}
	if err != nil {
		return nil, err
	}

	return boxedImageInfo.(*image.Info), nil
}

// CommandTreeNodeThumbnail resolves and returns the thumbnail for the framebuffer at p.
func CommandTreeNodeThumbnail(
	ctx context.Context,
	w, h uint32,
	f *image.Format,
	noOpt bool,
	p *path.CommandTreeNode,
	r *path.ResolveConfig) (*image.Info, error) {

	boxedCmdTree, err := database.Resolve(ctx, p.Tree.ID())
	if err != nil {
		return nil, err
	}

	cmdTree := boxedCmdTree.(*commandTree)

	item, absID := cmdTree.index(p.Indices)
	switch item := item.(type) {
	case api.CmdIDGroup:
		thumbnail := []uint64{uint64(item.Range.Last())}
		if userData, ok := item.UserData.(*CmdGroupData); ok {
			thumbnail = []uint64{uint64(userData.Representation)}
		} else if len(absID) > 0 {
			thumbnail = append(absID, uint64(item.Range.Last()))
		}
		return CommandThumbnail(ctx, w, h, f, noOpt, cmdTree.path.Capture.Command(thumbnail[0], thumbnail[1:]...), r)
	case api.SubCmdIdx:
		return CommandThumbnail(ctx, w, h, f, noOpt, cmdTree.path.Capture.Command(uint64(item[0]), item[1:]...), r)
	case api.SubCmdRoot:
		return CommandThumbnail(ctx, w, h, f, noOpt, cmdTree.path.Capture.Command(uint64(item.Id[0]), item.Id[1:]...), r)
	default:
		panic(fmt.Errorf("Unexpected type: %T", item))
	}
}

// ResourceDataThumbnail resolves and returns the thumbnail for the resource at p.
func ResourceDataThumbnail(ctx context.Context, w, h uint32, f *image.Format, p *path.ResourceData, r *path.ResolveConfig) (*image.Info, error) {
	obj, err := ResolveInternal(ctx, p, r)
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
