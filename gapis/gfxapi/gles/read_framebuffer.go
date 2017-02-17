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
	"fmt"

	"github.com/google/gapid/core/data/pod"
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
	injections map[atom.ID][]func(ctx log.Context, out transform.Writer)
}

func newReadFramebuffer(ctx log.Context) *readFramebuffer {
	return &readFramebuffer{
		injections: make(map[atom.ID][]func(ctx log.Context, out transform.Writer)),
	}
}

func (t *readFramebuffer) Transform(ctx log.Context, id atom.ID, a atom.Atom, out transform.Writer) {
	out.MutateAndWrite(ctx, id, a)
	if r, ok := t.injections[id]; ok {
		for _, injection := range r {
			injection(ctx, out)
		}
		delete(t.injections, id)
	}
}

func (t *readFramebuffer) Flush(ctx log.Context, out transform.Writer) {}

func (t *readFramebuffer) Depth(id atom.ID, res chan<- imgRes) {
	t.injections[id] = append(t.injections[id], func(ctx log.Context, out transform.Writer) {
		s := out.State()
		width, height, format, err := GetState(s).getFramebufferAttachmentInfo(gfxapi.FramebufferAttachment_Depth)
		if err != nil {
			res <- imgRes{err: &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}}
			return
		}

		postColorData(ctx, s, int32(width), int32(height), format, out, func(i imgRes) { res <- i })
	})
}

func (t *readFramebuffer) Color(id atom.ID, width, height, bufferIdx uint32, res chan<- imgRes) {
	t.injections[id] = append(t.injections[id], func(ctx log.Context, out transform.Writer) {
		s := out.State()
		c := GetContext(s)

		attachment := gfxapi.FramebufferAttachment_Color0 + gfxapi.FramebufferAttachment(bufferIdx)
		w, h, fmt, err := GetState(s).getFramebufferAttachmentInfo(attachment)
		if err != nil {
			res <- imgRes{err: &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}}
			return
		}

		var (
			origRenderbufferID            = c.BoundRenderbuffer
			origReadFramebufferID         = c.BoundReadFramebuffer
			origDrawFramebufferID         = c.BoundDrawFramebuffer
			origDrawFramebufferReadBuffer = c.Instances.Framebuffers[origDrawFramebufferID].ReadBuffer

			inW  = int32(w)
			inH  = int32(h)
			outW = int32(width)
			outH = int32(height)
		)

		out.MutateAndWrite(ctx, atom.NoID, NewGlBindFramebuffer(GLenum_GL_READ_FRAMEBUFFER, c.BoundDrawFramebuffer))

		// TODO: These glReadBuffer calls need to be changed for on-device
		//       replay. Note that glReadBuffer was only introduced in
		//       OpenGL ES 3.0, and that GL_FRONT is not a legal enum value.
		if origDrawFramebufferID == 0 {
			out.MutateAndWrite(ctx, atom.NoID, replay.Custom(func(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
				// TODO: We assume here that the default framebuffer is
				//       single-buffered. Once we support double-buffering we
				//       need to decide whether to read from GL_FRONT or GL_BACK.
				NewGlReadBuffer(GLenum_GL_FRONT).Call(ctx, s, b)
				return nil
			}))
		} else {
			out.MutateAndWrite(ctx, atom.NoID, NewGlReadBuffer(GLenum_GL_COLOR_ATTACHMENT0+GLenum(bufferIdx)))
		}

		if inW == outW && inH == outH {
			postColorData(ctx, s, outW, outH, fmt, out, func(i imgRes) { res <- i })
		} else {
			// Generate new unused object IDs.
			renderbufferID := RenderbufferId(newUnusedID('R', func(x uint32) bool { _, ok := c.Instances.Renderbuffers[RenderbufferId(x)]; return ok }))
			framebufferID := FramebufferId(newUnusedID('F', func(x uint32) bool { _, ok := c.Instances.Framebuffers[FramebufferId(x)]; return ok }))

			c := GetContext(s)
			origScissor := c.FragmentOperations.Scissor.Box

			tmpF := atom.Must(atom.AllocData(ctx, s, framebufferID))
			tmpR := atom.Must(atom.AllocData(ctx, s, renderbufferID))
			mutateAndWriteEach(ctx, out,
				NewGlScissor(0, 0, GLsizei(inW), GLsizei(inH)),
				NewGlGenFramebuffers(1, tmpF.Ptr()).AddRead(tmpF.Data()),
				NewGlBindFramebuffer(GLenum_GL_DRAW_FRAMEBUFFER, framebufferID),
				NewGlGenRenderbuffers(1, tmpR.Ptr()).AddRead(tmpR.Data()),
				NewGlBindRenderbuffer(GLenum_GL_RENDERBUFFER, renderbufferID),
				NewGlRenderbufferStorage(GLenum_GL_RENDERBUFFER, fmt.sif, GLsizei(outW), GLsizei(outH)),
				NewGlFramebufferRenderbuffer(GLenum_GL_DRAW_FRAMEBUFFER, GLenum_GL_COLOR_ATTACHMENT0, GLenum_GL_RENDERBUFFER, renderbufferID),
				NewGlBlitFramebuffer(0, 0, GLint(inW), GLint(inH), 0, 0, GLint(outW), GLint(outH), GLbitfield_GL_COLOR_BUFFER_BIT, GLenum_GL_LINEAR),
				NewGlBindFramebuffer(GLenum_GL_READ_FRAMEBUFFER, framebufferID),
			)

			postColorData(ctx, s, outW, outH, fmt, out, func(i imgRes) { res <- i })

			mutateAndWriteEach(ctx, out,
				NewGlBindRenderbuffer(GLenum_GL_RENDERBUFFER, origRenderbufferID),
				NewGlBindFramebuffer(GLenum_GL_READ_FRAMEBUFFER, origReadFramebufferID),
				NewGlBindFramebuffer(GLenum_GL_DRAW_FRAMEBUFFER, origDrawFramebufferID),
				NewGlDeleteRenderbuffers(1, tmpR.Ptr()).AddRead(tmpR.Data()),
				NewGlDeleteFramebuffers(1, tmpF.Ptr()).AddRead(tmpF.Data()),
				NewGlScissor(origScissor.X, origScissor.Y, origScissor.Width, origScissor.Height),
			)
		}

		if origDrawFramebufferID == 0 {
			// The original read buffer is likely GL_BACK, which is invalid on the replay device.
		} else {
			out.MutateAndWrite(ctx, atom.NoID, NewGlBindFramebuffer(GLenum_GL_READ_FRAMEBUFFER, origDrawFramebufferID))
			out.MutateAndWrite(ctx, atom.NoID, NewGlReadBuffer(origDrawFramebufferReadBuffer))
		}
		out.MutateAndWrite(ctx, atom.NoID, NewGlBindFramebuffer(GLenum_GL_READ_FRAMEBUFFER, origReadFramebufferID))
	})
}

func postColorData(ctx log.Context,
	s *gfxapi.State,
	width, height int32,
	glfmt imgfmt,
	out transform.Writer,
	callback func(imgRes)) {

	imgFmt := glfmt.asImageOrPanic()

	c := GetContext(s)
	origPackAlignment := c.PixelStorage.PackAlignment
	if origPackAlignment != 1 {
		out.MutateAndWrite(ctx, atom.NoID, NewGlPixelStorei(GLenum_GL_PACK_ALIGNMENT, 1))
		defer out.MutateAndWrite(ctx, atom.NoID, NewGlPixelStorei(GLenum_GL_PACK_ALIGNMENT, origPackAlignment))
	}
	if origPackBuffer := c.BoundBuffers.PixelPackBuffer; origPackBuffer != 0 {
		out.MutateAndWrite(ctx, atom.NoID, NewGlBindBuffer(GLenum_GL_PIXEL_PACK_BUFFER, 0))
		defer out.MutateAndWrite(ctx, atom.NoID, NewGlBindBuffer(GLenum_GL_PIXEL_PACK_BUFFER, origPackBuffer))
	}

	imageSize := imgFmt.Size(int(width), int(height))
	tmp := atom.Must(atom.Alloc(ctx, s, uint64(imageSize)))
	out.MutateAndWrite(ctx, atom.NoID, replay.Custom(func(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
		// TODO: We use Call() directly here because we are calling glReadPixels
		// with depth formats which are not legal for GLES. Once we're replaying
		// on-device again, we need to take a look at methods for reading the
		// depth buffer.

		b.ReserveMemory(tmp.Range())
		NewGlReadPixels(0, 0, GLsizei(width), GLsizei(height), glfmt.base, glfmt.ty, tmp.Ptr()).
			Call(ctx, s, b)

		b.Post(value.ObservedPointer(tmp.Address()), uint64(imageSize), func(r pod.Reader, err error) error {
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
			img := &image.Image2D{
				Data:   data,
				Width:  uint32(width),
				Height: uint32(height),
				Format: imgFmt,
			}
			callback(imgRes{img: img, err: err})
			return err
		})
		return nil
	}))
	tmp.Free()
}

func mutateAndWriteEach(ctx log.Context, out transform.Writer, atoms ...atom.Atom) {
	for _, a := range atoms {
		out.MutateAndWrite(ctx, atom.NoID, a)
	}
}

var nextUnusedID = map[rune]uint32{}

// newUnusedID returns temporary object ID.
// The tag makes the IDs for given object type more deterministic.
func newUnusedID(tag rune, existenceTest func(uint32) bool) uint32 {
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
