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

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
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

		f, err := getImageFormat(GLenum(r.DataFormat), GLenum(r.DataType))
		if err != nil {
			res(nil, err)
			return
		}

		tex, ok := c.Objects.Shared.Textures[TextureId(r.Texture)]
		if !ok {
			panic(fmt.Errorf("Attempting to read from a texture that does not exist.\nResolvable: %+v\nTexture: %+v", r, tex))
		}
		layer := tex.Levels[GLint(r.Level)].Layers[GLint(r.Layer)]
		if layer == nil {
			panic(fmt.Errorf("Attempting to read from a texture layer that does not exist.\nResolvable: %+v\nTexture: %+v", r, tex))
		}

		size := uint64(f.Size(int(layer.Width), int(layer.Height), 1))
		tmp := s.AllocOrPanic(ctx, size)
		defer tmp.Free()

		t := newTweaker(out, dID, cb)
		defer t.revert(ctx)

		t.setPackStorage(ctx, PixelStorageState{Alignment: 1}, 0)
		t.glBindTexture(ctx, tex)

		target := tex.Kind
		if tex.Kind == GLenum_GL_TEXTURE_CUBE_MAP {
			target = GLenum_GL_TEXTURE_CUBE_MAP_POSITIVE_X + GLenum(r.Layer)
		}
		out.MutateAndWrite(ctx, dID, cb.GlGetTexImage(target, GLint(r.Level), GLenum(r.DataFormat), GLenum(r.DataType), tmp.Ptr()))

		out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.State, b *builder.Builder) error {
			b.Post(value.ObservedPointer(tmp.Address()), size, func(r binary.Reader, err error) error {
				data := make([]byte, size)
				if err == nil {
					r.Data(data)
					err = r.Error()
				}
				if err == nil {
					res(data, nil)
				} else {
					res(nil, err)
				}
				return err
			})
			return nil
		}))
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
	return res.([]byte), nil
}
