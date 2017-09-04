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

package api

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/image"
)

func (l *CubemapLevel) faces() [6]*image.Info {
	return [6]*image.Info{
		l.NegativeX,
		l.PositiveX,
		l.NegativeY,
		l.PositiveY,
		l.NegativeZ,
		l.PositiveZ,
	}
}

func (l *CubemapLevel) setFaces(faces [6]*image.Info) {
	l.NegativeX,
		l.PositiveX,
		l.NegativeY,
		l.PositiveY,
		l.NegativeZ,
		l.PositiveZ = faces[0], faces[1], faces[2], faces[3], faces[4], faces[5]
}

type imageMatcher struct {
	best   *image.Info
	score  uint32
	width  uint32
	height uint32
	depth  uint32
}

func (m *imageMatcher) consider(i *image.Info) {
	if i == nil {
		return
	}

	if m.best == nil {
		m.score = 0xffffffff
	}
	dw, dh, dd := i.Width-m.width, i.Height-m.height, i.Depth-m.depth
	score := dw*dw + dh*dh + dd*dd
	if m.score > score {
		m.score = score
		m.best = i
	}
}

// Interface compliance check
var _ = image.Convertable((*Texture1D)(nil))
var _ = image.Thumbnailer((*Texture1D)(nil))

// ConvertTo returns this Texture1D with each mip-level converted to the requested format.
func (t *Texture1D) ConvertTo(ctx context.Context, f *image.Format) (interface{}, error) {
	out := &Texture1D{
		Levels: make([]*image.Info, len(t.Levels)),
	}
	for i, m := range t.Levels {
		obj, err := m.Convert(ctx, f)
		if err != nil {
			return nil, err
		}
		out.Levels[i] = obj
	}
	return out, nil
}

// Thumbnail returns the image that most closely matches the desired size.
func (t *Texture1D) Thumbnail(ctx context.Context, w, h, d uint32) (*image.Info, error) {
	m := imageMatcher{width: w, height: 1, depth: 1}
	for _, l := range t.Levels {
		m.consider(l)
	}

	return m.best, nil
}

// Interface compliance check
var _ = image.Convertable((*Texture2D)(nil))
var _ = image.Thumbnailer((*Texture2D)(nil))

// ConvertTo returns this Texture2D with each mip-level converted to the requested format.
func (t *Texture2D) ConvertTo(ctx context.Context, f *image.Format) (interface{}, error) {
	out := &Texture2D{
		Levels: make([]*image.Info, len(t.Levels)),
	}
	for i, m := range t.Levels {
		obj, err := m.Convert(ctx, f)
		if err != nil {
			return nil, err
		}
		out.Levels[i] = obj
	}
	return out, nil
}

// Thumbnail returns the image that most closely matches the desired size.
func (t *Texture2D) Thumbnail(ctx context.Context, w, h, d uint32) (*image.Info, error) {
	m := imageMatcher{width: w, height: h, depth: 1}
	for _, l := range t.Levels {
		m.consider(l)
	}

	return m.best, nil
}

// Interface compliance check
var _ = image.Convertable((*Texture2DArray)(nil))
var _ = image.Thumbnailer((*Texture2DArray)(nil))

// ConvertTo returns this Texture2DArray with each layer and  mip-level
// converted to the requested format.
func (t *Texture2DArray) ConvertTo(ctx context.Context, f *image.Format) (interface{}, error) {
	out := &Texture2DArray{
		Layers: make([]*Texture2D, len(t.Layers)),
	}
	for i, l := range t.Layers {
		l, err := l.ConvertTo(ctx, f)
		if err != nil {
			return nil, err
		}
		out.Layers[i] = l.(*Texture2D)
	}
	return out, nil
}

// Thumbnail returns the image that most closely matches the desired size.
func (t *Texture2DArray) Thumbnail(ctx context.Context, w, h, d uint32) (*image.Info, error) {
	m := imageMatcher{width: w, height: h, depth: 1}
	for _, layer := range t.Layers {
		for _, level := range layer.Levels {
			m.consider(level)
		}
	}
	return m.best, nil
}

// Interface compliance check
var _ = image.Convertable((*Texture3D)(nil))
var _ = image.Thumbnailer((*Texture3D)(nil))

// ConvertTo returns this Texture3D with each mip-level converted to the requested format.
func (t *Texture3D) ConvertTo(ctx context.Context, f *image.Format) (interface{}, error) {
	out := &Texture3D{
		Levels: make([]*image.Info, len(t.Levels)),
	}
	for i, m := range t.Levels {
		obj, err := m.Convert(ctx, f)
		if err != nil {
			return nil, err
		}
		out.Levels[i] = obj
	}
	return out, nil
}

// Thumbnail returns the image that most closely matches the desired size.
func (t *Texture3D) Thumbnail(ctx context.Context, w, h, d uint32) (*image.Info, error) {
	m := imageMatcher{width: w, height: h, depth: d}
	for _, l := range t.Levels {
		m.consider(l)
	}

	return m.best, nil
}

// Interface compliance check
var _ = image.Convertable((*Cubemap)(nil))
var _ = image.Thumbnailer((*Cubemap)(nil))

// ConvertTo returns this Cubemap with each mip-level face converted to the requested format.
func (t *Cubemap) ConvertTo(ctx context.Context, f *image.Format) (interface{}, error) {
	out := &Cubemap{
		Levels: make([]*CubemapLevel, len(t.Levels)),
	}
	for i, m := range t.Levels {
		out.Levels[i] = &CubemapLevel{}
		dst, src := out.Levels[i].faces(), m.faces()
		for j, srcFace := range src {
			if srcFace == nil {
				continue
			}

			cnvFace, err := srcFace.Convert(ctx, f)
			if err != nil {
				return nil, err
			}
			dst[j] = cnvFace
		}
		out.Levels[i].setFaces(dst)
	}
	return out, nil
}

// Thumbnail returns the image that most closely matches the desired size.
func (t *Cubemap) Thumbnail(ctx context.Context, w, h, d uint32) (*image.Info, error) {
	m := imageMatcher{width: w, height: h, depth: 1}

	for _, l := range t.Levels {
		m.consider(l.NegativeX)
		m.consider(l.PositiveX)
		m.consider(l.NegativeY)
		m.consider(l.PositiveY)
		m.consider(l.NegativeZ)
		m.consider(l.PositiveZ)
	}

	return m.best, nil
}

// Interface compliance check
var _ = image.Convertable((*Texture)(nil))
var _ = image.Thumbnailer((*Texture)(nil))

// ConvertTo returns this Texture2D with each mip-level converted to the requested format.
func (t *Texture) ConvertTo(ctx context.Context, f *image.Format) (interface{}, error) {
	data := protoutil.OneOf(t.Type)
	if c, ok := data.(image.Convertable); ok {
		data, err := c.ConvertTo(ctx, f)
		if err != nil {
			return nil, err
		}
		return NewTexture(data), nil
	}
	return nil, nil
}

// Thumbnail returns the image that most closely matches the desired size.
func (t *Texture) Thumbnail(ctx context.Context, w, h, d uint32) (*image.Info, error) {
	data := protoutil.OneOf(t.Type)
	if t, ok := data.(image.Thumbnailer); ok {
		return t.Thumbnail(ctx, w, h, d)
	}
	return nil, nil
}

// NewTexture returns a new *ResourceData with the specified texture.
func NewTexture(t interface{}) *Texture {
	switch t := t.(type) {
	case *Texture1D:
		return &Texture{Type: &Texture_Texture_1D{t}}
	case *Texture1DArray:
		return &Texture{Type: &Texture_Texture_1DArray{t}}
	case *Texture2D:
		return &Texture{Type: &Texture_Texture_2D{t}}
	case *Texture2DArray:
		return &Texture{Type: &Texture_Texture_2DArray{t}}
	case *Texture3D:
		return &Texture{Type: &Texture_Texture_3D{t}}
	case *Cubemap:
		return &Texture{Type: &Texture_Cubemap{t}}
	case *CubemapArray:
		return &Texture{Type: &Texture_CubemapArray{t}}
	default:
		panic(fmt.Errorf("%T is not a Texture type", t))
	}
}
