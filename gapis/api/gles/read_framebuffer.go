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
	"github.com/google/gapid/core/stream"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
)

type readFramebuffer struct{ transform.Tasks }

func newReadFramebuffer(ctx context.Context) *readFramebuffer {
	return &readFramebuffer{}
}

func getBoundFramebufferID(thread uint64, s *api.GlobalState) (FramebufferId, error) {
	c := GetContext(s, thread)
	if c == nil {
		return 0, fmt.Errorf("No OpenGL ES context")
	}
	if c.Bound.DrawFramebuffer == nil {
		return 0, fmt.Errorf("No framebuffer bound")
	}
	return c.Bound.DrawFramebuffer.GetID(), nil
}

func (t *readFramebuffer) depth(
	id api.CmdID,
	thread uint64,
	fb FramebufferId,
	res replay.Result) {

	t.Add(id, func(ctx context.Context, out transform.Writer) {
		postFBData(ctx, id, thread, 0, 0, fb, GLenum_GL_DEPTH_ATTACHMENT, out, res)
	})
}

func (t *readFramebuffer) color(
	id api.CmdID,
	thread uint64,
	width, height uint32,
	fb FramebufferId,
	bufferIdx uint32,
	res replay.Result) {

	t.Add(id, func(ctx context.Context, out transform.Writer) {
		attachment := GLenum_GL_COLOR_ATTACHMENT0 + GLenum(bufferIdx)
		postFBData(ctx, id, thread, width, height, fb, attachment, out, res)
	})
}

func postFBData(ctx context.Context,
	id api.CmdID,
	thread uint64,
	width, height uint32,
	fb FramebufferId,
	attachment GLenum,
	out transform.Writer,
	res replay.Result) {

	s := out.State()
	c := GetContext(s, thread)

	if fb == 0 {
		var err error
		if fb, err = getBoundFramebufferID(thread, s); err != nil {
			log.W(ctx, "Could not read framebuffer after cmd %v: err", err)
			res(nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()})
			return
		}
	}

	fbai, err := GetState(s).getFramebufferAttachmentInfo(thread, fb, attachment)
	if err != nil {
		log.W(ctx, "Failed to read framebuffer after cmd %v: %v", id, err)
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()})
		return
	}
	if fbai.format == 0 {
		log.W(ctx, "Failed to read framebuffer after cmd %v: no image format", id)
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()})
		return
	}

	var (
		inW  = int32(fbai.width)
		inH  = int32(fbai.height)
		outW = int32(width)
		outH = int32(height)
	)

	if outW == 0 {
		outW = inW
	}
	if outH == 0 {
		outH = inH
	}

	dID := id.Derived()
	cb := CommandBuilder{Thread: thread}
	t := newTweaker(out, dID, cb)
	defer t.revert(ctx)
	t.glBindFramebuffer_Read(ctx, fb)

	unsizedFormat, ty := getUnsizedFormatAndType(fbai.format)

	imgFmt, err := getImageFormat(unsizedFormat, ty)
	if err != nil {
		res(nil, err)
		return
	}

	channels := imgFmt.Channels()
	hasColor := channels.ContainsColor()
	hasDepth := channels.ContainsDepth()
	hasStencil := channels.ContainsStencil()

	if hasColor && (hasDepth || hasStencil) {
		// Sanity check.
		// If this fails, the logic of this function has to be rewritten.
		panic("Found framebuffer attachment with both color and depth/stencil components!")
	}

	bufferBits := GLbitfield(0)
	if hasColor {
		bufferBits |= GLbitfield_GL_COLOR_BUFFER_BIT
	}
	if hasDepth {
		bufferBits |= GLbitfield_GL_DEPTH_BUFFER_BIT
	}
	if hasStencil {
		bufferBits |= GLbitfield_GL_STENCIL_BUFFER_BIT
	}

	if (attachment == GLenum_GL_DEPTH_ATTACHMENT || attachment == GLenum_GL_STENCIL_ATTACHMENT) &&
		hasDepth && hasStencil {
		// The caller of this function has specified that they want either the
		// depth or the stencil buffer, but the FBO is actually depth and
		// stencil.
		//
		// To keep the replay logic sane, preserve both depth and stencil data
		// and post both back to GAPIS. We then can strip out the unwanted
		// component.
		var outputFormat *image.Format
		if attachment == GLenum_GL_DEPTH_ATTACHMENT {
			outputFormat = filterUncompressedImageFormat(imgFmt, stream.Channel.IsDepth)
		} else {
			outputFormat = filterUncompressedImageFormat(imgFmt, stream.Channel.IsStencil)
		}
		res = res.Transform(func(in interface{}) (interface{}, error) {
			return in.(*image.Data).Convert(outputFormat)
		})

		attachment = GLenum_GL_DEPTH_STENCIL_ATTACHMENT
	}

	if hasColor {
		// TODO: These glReadBuffer calls need to be changed for on-device
		//       replay. Note that glReadBuffer was only introduced in
		//       OpenGL ES 3.0, and that GL_FRONT is not a legal enum value.
		if c.Bound.DrawFramebuffer == c.Objects.Default.Framebuffer {
			out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
				// TODO: We assume here that the default framebuffer is
				//       single-buffered. Once we support double-buffering we
				//       need to decide whether to read from GL_FRONT or GL_BACK.
				cb.GlReadBuffer(GLenum_GL_FRONT).Call(ctx, s, b)
				return nil
			}))
		} else {
			t.glReadBuffer(ctx, attachment)
		}
	}

	if fbai.multisampled {
		// Resolve
		t.glScissor(ctx, 0, 0, GLsizei(inW), GLsizei(inH))
		framebufferID := t.glGenFramebuffer(ctx)
		t.glBindFramebuffer_Draw(ctx, framebufferID)
		renderbufferID := t.glGenRenderbuffer(ctx)
		t.glBindRenderbuffer(ctx, renderbufferID)

		mutateAndWriteEach(ctx, out, dID,
			cb.GlRenderbufferStorage(GLenum_GL_RENDERBUFFER, fbai.format, GLsizei(inW), GLsizei(inH)),
			cb.GlFramebufferRenderbuffer(GLenum_GL_DRAW_FRAMEBUFFER, attachment, GLenum_GL_RENDERBUFFER, renderbufferID),
			cb.GlBlitFramebuffer(0, 0, GLint(inW), GLint(inH), 0, 0, GLint(inW), GLint(inH), bufferBits, GLenum_GL_NEAREST),
		)

		t.glBindFramebuffer_Read(ctx, framebufferID)
	}

	if hasColor && (inW != outW || inH != outH) {
		// Resize
		t.glScissor(ctx, 0, 0, GLsizei(inW), GLsizei(inH))
		framebufferID := t.glGenFramebuffer(ctx)
		t.glBindFramebuffer_Draw(ctx, framebufferID)
		renderbufferID := t.glGenRenderbuffer(ctx)
		t.glBindRenderbuffer(ctx, renderbufferID)

		mutateAndWriteEach(ctx, out, dID,
			cb.GlRenderbufferStorage(GLenum_GL_RENDERBUFFER, fbai.format, GLsizei(outW), GLsizei(outH)),
			cb.GlFramebufferRenderbuffer(GLenum_GL_DRAW_FRAMEBUFFER, attachment, GLenum_GL_RENDERBUFFER, renderbufferID),
			cb.GlBlitFramebuffer(0, 0, GLint(inW), GLint(inH), 0, 0, GLint(outW), GLint(outH), bufferBits, GLenum_GL_LINEAR),
		)
		t.glBindFramebuffer_Read(ctx, framebufferID)
	}

	t.setPackStorage(ctx, PixelStorageState{Alignment: 1}, 0)

	imageSize := imgFmt.Size(int(outW), int(outH), 1)
	tmp := s.AllocOrPanic(ctx, uint64(imageSize))
	defer tmp.Free()

	out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		// TODO: We use Call() directly here because we are calling glReadPixels
		// with depth formats which are not legal for GLES. Once we're replaying
		// on-device again, we need to take a look at methods for reading the
		// depth buffer.

		b.ReserveMemory(tmp.Range())
		cb.GlReadPixels(0, 0, GLsizei(outW), GLsizei(outH), unsizedFormat, ty, tmp.Ptr()).
			Call(ctx, s, b)

		b.Post(value.ObservedPointer(tmp.Address()), uint64(imageSize), func(r binary.Reader, err error) error {
			return res.Do(func() (interface{}, error) {
				if err != nil {
					return nil, err
				}

				data := make([]byte, imageSize)
				r.Data(data)
				if err := r.Error(); err != nil {
					return nil, fmt.Errorf("Could not read framebuffer data (expected length %d bytes): %v", imageSize, err)
				}

				return &image.Data{
					Bytes:  data,
					Width:  uint32(outW),
					Height: uint32(outH),
					Depth:  1,
					Format: imgFmt,
				}, nil
			})
		})
		return nil
	}))

	out.MutateAndWrite(ctx, dID, cb.GlGetError(0)) // Check for errors.
}

func mutateAndWriteEach(ctx context.Context, out transform.Writer, id api.CmdID, cmds ...api.Cmd) {
	for _, cmd := range cmds {
		out.MutateAndWrite(ctx, id, cmd)
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
