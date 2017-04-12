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
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Thumbnail resolves and returns the thumbnail from the path p.
func Thumbnail(ctx context.Context, p *path.Thumbnail) (*image.Info2D, error) {
	obj, err := ResolveInternal(ctx, p.Parent())
	if err != nil {
		return nil, err
	}

	t, ok := obj.(image.Thumbnailer)
	if !ok {
		return nil, fmt.Errorf("Type %T does not support thumbnailing", obj)
	}

	img, err := t.Thumbnail(ctx, p.DesiredMaxWidth, p.DesiredMaxHeight)
	if err != nil {
		return nil, err
	}

	if img == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData("")}
	}

	if p.DesiredFormat != nil {
		// Convert the image to the desired format.
		f := p.DesiredFormat
		if img.Format.Key() != f.Key() {
			img, err = img.ConvertTo(ctx, f)
			if err != nil {
				return nil, err
			}
		}
	}

	// Image format supports resizing. See if the image should be.
	scaleX, scaleY := float32(1), float32(1)
	if p.DesiredMaxWidth > 0 && img.Width > p.DesiredMaxWidth {
		scaleX = float32(p.DesiredMaxWidth) / float32(img.Width)
	}
	if p.DesiredMaxHeight > 0 && img.Height > p.DesiredMaxHeight {
		scaleY = float32(p.DesiredMaxHeight) / float32(img.Height)
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

	return img.Resize(ctx, targetWidth, targetHeight)
}
