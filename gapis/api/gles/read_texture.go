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

package gles

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
)

type textureRequest struct {
	data *ReadGPUTextureDataResolveable
}

type readTexture struct{ transform.Tasks }

func (t *readTexture) add(ctx context.Context, r *ReadGPUTextureDataResolveable, res replay.Result) {
	id := api.CmdID(r.After)
	t.Add(id, func(ctx context.Context, out transform.Writer) {
		s := out.State()
		c := GetContext(s, r.Thread)
		dID := id.Derived()
		cb := CommandBuilder{Thread: r.Thread}

		tex, ok := c.Objects.Textures.Lookup(TextureId(r.Texture))
		if !ok {
			err := fmt.Errorf("Attempting to read from texture %v that does not exist.\n"+
				"Resolvable: %+v\nTexture: %+v", r.Texture, r, tex)
			log.W(ctx, "%v", err)
			res(nil, err)
			return
		}
		lvl := tex.Levels.Get(GLint(r.Level))
		layer := lvl.Layers.Get(GLint(r.Layer))
		if layer == nil {
			err := fmt.Errorf("Attempting to read from texture %v (Level: %v/%v, Layer: %v/%v) that does not exist.\n"+
				"Resolvable: %+v\n"+
				"Texture: %+v",
				r.Texture, r.Level, tex.Levels.Len(), r.Layer, lvl.Layers.Len(), r, tex)
			log.W(ctx, "%v", err)
			res(nil, err)
			return
		}

		t := newTweaker(out, dID, cb)
		defer t.revert(ctx)

		framebufferID := t.glGenFramebuffer(ctx)
		t.glBindFramebuffer_Draw(ctx, framebufferID)

		streamFmt, err := getUncompressedStreamFormat(getUnsizedFormatAndType(layer.SizedFormat))
		if err != nil {
			res(nil, err)
			return
		}

		var attachment GLenum
		channels := streamFmt.Channels()
		switch {
		case channels.ContainsColor():
			attachment = GLenum_GL_COLOR_ATTACHMENT0
		case channels.ContainsDepth() && channels.ContainsStencil():
			attachment = GLenum_GL_DEPTH_STENCIL_ATTACHMENT
		case channels.ContainsDepth():
			attachment = GLenum_GL_DEPTH_ATTACHMENT
		case channels.ContainsStencil():
			attachment = GLenum_GL_STENCIL_ATTACHMENT
		default:
			res(nil, fmt.Errorf("Unsupported texture format %v", streamFmt))
			return
		}

		if r.Layer == 0 {
			out.MutateAndWrite(ctx, dID, cb.GlFramebufferTexture(GLenum_GL_DRAW_FRAMEBUFFER, attachment, tex.ID, GLint(r.Level)))
		} else {
			out.MutateAndWrite(ctx, dID, cb.GlFramebufferTextureLayer(GLenum_GL_DRAW_FRAMEBUFFER, attachment, tex.ID, GLint(r.Level), GLint(r.Layer)))
		}

		// Compat may have altered the texture format.
		// The caller expects the data in the texture's authored format.
		// Convert.
		f, err := getImageFormat(GLenum(r.DataFormat), GLenum(r.DataType))
		if err != nil {
			res(nil, err)
			return
		}
		res = res.Transform(func(in interface{}) (interface{}, error) {
			return in.(*image.Data).Convert(f)
		})

		postFBData(ctx, dID, r.Thread, uint32(layer.Width), uint32(layer.Height), framebufferID, attachment, out, res)
	})
}

// Resolve implements the database.Resolver interface.
func (r *ReadGPUTextureDataResolveable) Resolve(ctx context.Context) (interface{}, error) {
	c := drawConfig{}
	mgr := replay.GetManager(ctx)
	intent := replay.Intent{
		Device:  r.Device,
		Capture: r.Capture,
	}
	hints := &service.UsageHints{}
	res, err := mgr.Replay(ctx, intent, c, textureRequest{r}, API{}, hints)
	if err != nil {
		return nil, err
	}
	return res.(*image.Data).Bytes, nil
}
