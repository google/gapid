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

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
)

type readFramebuffer struct {
	injections map[atom.ID][]func(ctx context.Context, out transform.Writer)
}

func newReadFramebuffer(ctx context.Context) *readFramebuffer {
	return &readFramebuffer{
		injections: make(map[atom.ID][]func(ctx context.Context, out transform.Writer)),
	}
}

func (t *readFramebuffer) Transform(ctx context.Context, id atom.ID, a atom.Atom, out transform.Writer) {
	out.MutateAndWrite(ctx, id, a)
	if r, ok := t.injections[id]; ok {
		for _, injection := range r {
			injection(ctx, out)
		}
		delete(t.injections, id)
	}
}

func (t *readFramebuffer) Flush(ctx context.Context, out transform.Writer) {}

func (t *readFramebuffer) Depth(id atom.ID, res replay.Result) {
	t.injections[id] = append(t.injections[id], func(ctx context.Context, out transform.Writer) {
		s := out.State()
		width, height, format, err := GetState(s).getFramebufferAttachmentInfo(gfxapi.FramebufferAttachment_Depth)
		if err != nil {
			res(nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()})
			return
		}

		postColorData(ctx, s, int32(width), int32(height), format, out, id, res)
	})
}

func (t *readFramebuffer) Color(id atom.ID, width, height, bufferIdx uint32, res replay.Result) {
	t.injections[id] = append(t.injections[id], func(ctx context.Context, out transform.Writer) {
		s := out.State()
		c := GetContext(s)

		attachment := gfxapi.FramebufferAttachment_Color0 + gfxapi.FramebufferAttachment(bufferIdx)
		w, h, fmt, err := GetState(s).getFramebufferAttachmentInfo(attachment)
		if err != nil {
			log.W(ctx, "Failed to read framebuffer after atom %v: %v", id, err)
			res(nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()})
			return
		}
		if fmt == 0 {
			log.W(ctx, "Failed to read framebuffer after atom %v: no image format", id)
			res(nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()})
			return
		}

		var (
			inW  = int32(w)
			inH  = int32(h)
			outW = int32(width)
			outH = int32(height)
		)

		dID := id.Derived()
		t := newTweaker(ctx, out, dID)

		t.glBindFramebuffer_Read(c.Bound.DrawFramebuffer.GetID())

		// TODO: These glReadBuffer calls need to be changed for on-device
		//       replay. Note that glReadBuffer was only introduced in
		//       OpenGL ES 3.0, and that GL_FRONT is not a legal enum value.
		if c.Bound.DrawFramebuffer == c.Objects.Default.Framebuffer {
			out.MutateAndWrite(ctx, dID, replay.Custom(func(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
				// TODO: We assume here that the default framebuffer is
				//       single-buffered. Once we support double-buffering we
				//       need to decide whether to read from GL_FRONT or GL_BACK.
				NewGlReadBuffer(GLenum_GL_FRONT).Call(ctx, s, b)
				return nil
			}))
		} else {
			t.glReadBuffer(GLenum_GL_COLOR_ATTACHMENT0 + GLenum(bufferIdx))
		}

		if inW == outW && inH == outH {
			postColorData(ctx, s, outW, outH, fmt, out, id, res)
		} else {
			t.glScissor(0, 0, GLsizei(inW), GLsizei(inH))
			framebufferID := t.glGenFramebuffer()
			t.glBindFramebuffer_Draw(framebufferID)
			renderbufferID := t.glGenRenderbuffer()
			t.glBindRenderbuffer(renderbufferID)

			mutateAndWriteEach(ctx, out, dID,
				NewGlRenderbufferStorage(GLenum_GL_RENDERBUFFER, fmt, GLsizei(outW), GLsizei(outH)),
				NewGlFramebufferRenderbuffer(GLenum_GL_DRAW_FRAMEBUFFER, GLenum_GL_COLOR_ATTACHMENT0, GLenum_GL_RENDERBUFFER, renderbufferID),
				NewGlBlitFramebuffer(0, 0, GLint(inW), GLint(inH), 0, 0, GLint(outW), GLint(outH), GLbitfield_GL_COLOR_BUFFER_BIT, GLenum_GL_LINEAR),
			)
			t.glBindFramebuffer_Read(framebufferID)

			postColorData(ctx, s, outW, outH, fmt, out, id, res)
		}

		t.revert()
	})
}

func postColorData(ctx context.Context,
	s *gfxapi.State,
	width, height int32,
	sizedFormat GLenum,
	out transform.Writer,
	id atom.ID,
	res replay.Result) {

	unsizedFormat, ty := getUnsizedFormatAndType(sizedFormat)
	imgFmt := getImageFormatOrPanic(unsizedFormat, ty)

	dID := id.Derived()
	t := newTweaker(ctx, out, dID)
	t.setPixelStorage(PixelStorageState{PackAlignment: 1, UnpackAlignment: 1}, 0, 0)

	imageSize := imgFmt.Size(int(width), int(height), 1)
	tmp := atom.Must(atom.Alloc(ctx, s, uint64(imageSize)))
	out.MutateAndWrite(ctx, dID, replay.Custom(func(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
		// TODO: We use Call() directly here because we are calling glReadPixels
		// with depth formats which are not legal for GLES. Once we're replaying
		// on-device again, we need to take a look at methods for reading the
		// depth buffer.

		b.ReserveMemory(tmp.Range())
		NewGlReadPixels(0, 0, GLsizei(width), GLsizei(height), unsizedFormat, ty, tmp.Ptr()).
			Call(ctx, s, b)

		b.Post(value.ObservedPointer(tmp.Address()), uint64(imageSize), func(r binary.Reader, err error) error {
			var data []byte
			if err == nil {
				data = make([]byte, imageSize)
				r.Data(data)
				err = r.Error()
			}
			if err != nil {
				err = fmt.Errorf("Could not read framebuffer data (expected length %d bytes): %v", imageSize, err)
				data = nil
			}
			img := &image.Data{
				Bytes:  data,
				Width:  uint32(width),
				Height: uint32(height),
				Depth:  1,
				Format: imgFmt,
			}
			res(img, err)
			return err
		})
		return nil
	}))
	tmp.Free()

	out.MutateAndWrite(ctx, dID, NewGlGetError(0)) // Check for errors.

	t.revert()
}

func mutateAndWriteEach(ctx context.Context, out transform.Writer, id atom.ID, atoms ...atom.Atom) {
	for _, a := range atoms {
		out.MutateAndWrite(ctx, id, a)
	}
}

type nextUnusedIDKeyTy string

const nextUnusedIDKey = nextUnusedIDKeyTy("nextUnusedID")

func PutUnusedIDMap(ctx context.Context) context.Context {
	return keys.WithValue(ctx, nextUnusedIDKey, map[rune]uint32{})
}

// newUnusedID returns temporary object ID.
// The tag makes the IDs for given object type more deterministic.
func newUnusedID(ctx context.Context, tag rune, existenceTest func(uint32) bool) uint32 {
	val := ctx.Value(nextUnusedIDKey)
	if val == nil {
		panic(nextUnusedIDKey + " missing from context")
	}
	nextUnusedID := val.(map[rune]uint32)

	// Use the tag to allocate from different ranges.
	prefix := uint32(tag)
	if prefix == 0 || prefix > 128 {
		panic(fmt.Errorf("Expected ASCII character"))
	}
	prefix = prefix * 10000000
	// Get the next ID and make sure it is free.
	for {
		nextUnusedID[tag]++
		x := prefix + nextUnusedID[tag]
		if !existenceTest(x) && x != 0 {
			return x
		}
	}
}
