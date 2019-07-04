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
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/stream"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/protocol"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
)

type readFramebuffer struct {
	transform.Tasks
	targetVersion *Version
}

func newReadFramebuffer(ctx context.Context, device *device.Instance) *readFramebuffer {
	targetVersion, _ := ParseVersion(device.Configuration.Drivers.Opengl.Version)
	return &readFramebuffer{targetVersion: targetVersion}
}

func getBoundFramebufferID(thread uint64, s *api.GlobalState) (FramebufferId, error) {
	c := GetContext(s, thread)
	if c.IsNil() {
		return 0, fmt.Errorf("No OpenGL ES context")
	}
	if c.Bound().DrawFramebuffer().IsNil() {
		return 0, fmt.Errorf("No framebuffer bound")
	}
	return c.Bound().DrawFramebuffer().GetID(), nil
}

func (t *readFramebuffer) depth(
	id api.CmdID,
	thread uint64,
	fb FramebufferId,
	res replay.Result) {

	t.Add(id, func(ctx context.Context, out transform.Writer) {
		postFBData(ctx, id, thread, 0, 0, fb, GLenum_GL_DEPTH_ATTACHMENT, t.targetVersion, out, res)
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
		postFBData(ctx, id, thread, width, height, fb, attachment, t.targetVersion, out, res)
	})
}

func postFBData(ctx context.Context,
	id api.CmdID,
	thread uint64,
	width, height uint32,
	fb FramebufferId,
	attachment GLenum,
	version *Version,
	out transform.Writer,
	res replay.Result) {

	s := out.State()
	c := GetContext(s, thread)

	if fb == 0 {
		var err error
		if fb, err = getBoundFramebufferID(thread, s); err != nil {
			log.W(ctx, "Could not read framebuffer after cmd %v: %v", id, err)
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
	if fbai.sizedFormat == 0 {
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
	cb := CommandBuilder{Thread: thread, Arena: s.Arena}
	t := newTweaker(out, dID, cb)
	defer t.revert(ctx)
	t.glBindFramebuffer_Read(ctx, fb)

	unsizedFormat, ty := getUnsizedFormatAndType(fbai.sizedFormat)

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

	// Error out early if trying to read depth data on GLES 2.
	if hasDepth && version.IsES && version.Major == 2 {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrDepthBufferNotSupported()})
		return
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
		if c.Bound().DrawFramebuffer() == c.Objects().Default().Framebuffer() {
			out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
				// TODO: We assume here that the default framebuffer is
				//       single-buffered. Once we support double-buffering we
				//       need to decide whether to read from GL_FRONT or GL_BACK.
				buf := GLenum_GL_BACK
				if !version.IsES {
					// OpenGL expects GL_FRONT for single-buffered
					// configurations. Note this is not a legal value for GLES.
					buf = GLenum_GL_FRONT
				}
				cb.GlReadBuffer(buf).Call(ctx, s, b)
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

		if hasColor {
			// Blit defaults to color attachment 0. attaching there avoids
			// having to setup the read/draw buffer mappings.
			attachment = GLenum_GL_COLOR_ATTACHMENT0
		}

		mutateAndWriteEach(ctx, out, dID,
			cb.GlRenderbufferStorage(GLenum_GL_RENDERBUFFER, fbai.sizedFormat, GLsizei(inW), GLsizei(inH)),
			cb.GlFramebufferRenderbuffer(GLenum_GL_DRAW_FRAMEBUFFER, attachment, GLenum_GL_RENDERBUFFER, renderbufferID),
			cb.GlBlitFramebuffer(0, 0, GLint(inW), GLint(inH), 0, 0, GLint(inW), GLint(inH), bufferBits, GLenum_GL_NEAREST),
		)

		t.glBindFramebuffer_Read(ctx, framebufferID)
		t.glReadBuffer(ctx, attachment)
	}

	// These are formats that are required to be supported by GLES glReadPixels.
	// See section 4.3.2 of the GLES 3.0 spec.
	needFBQuery := version.IsES && !((unsizedFormat == GLenum_GL_RGBA && ty == GLenum_GL_UNSIGNED_BYTE) ||
		(unsizedFormat == GLenum_GL_RGBA && ty == GLenum_GL_UNSIGNED_INT_2_10_10_10_REV) ||
		(unsizedFormat == GLenum_GL_RGBA_INTEGER && ty == GLenum_GL_INT) ||
		(unsizedFormat == GLenum_GL_RGBA_INTEGER && ty == GLenum_GL_UNSIGNED_INT))

	needResize := hasColor && (inW != outW || inH != outH)
	// If we need to ask the driver what format to use to read the pixels, due
	// to driver issues, it needs to be COLOR_ATTACHMENT0. See b/115574126.
	needColorAtt0 := needFBQuery && hasColor && attachment != GLenum_GL_COLOR_ATTACHMENT0

	if needResize || needColorAtt0 {
		t.glScissor(ctx, 0, 0, GLsizei(inW), GLsizei(inH))
		framebufferID := t.glGenFramebuffer(ctx)
		t.glBindFramebuffer_Draw(ctx, framebufferID)
		renderbufferID := t.glGenRenderbuffer(ctx)
		t.glBindRenderbuffer(ctx, renderbufferID)

		// Blit defaults to color attachment 0. Attaching there avoids having to
		// setup the read/draw buffer mappings.
		attachment = GLenum_GL_COLOR_ATTACHMENT0
		sampling := GLenum_GL_NEAREST
		if needResize {
			sampling = GLenum_GL_LINEAR
		}

		mutateAndWriteEach(ctx, out, dID,
			cb.GlRenderbufferStorage(GLenum_GL_RENDERBUFFER, fbai.sizedFormat, GLsizei(outW), GLsizei(outH)),
			cb.GlFramebufferRenderbuffer(GLenum_GL_DRAW_FRAMEBUFFER, attachment, GLenum_GL_RENDERBUFFER, renderbufferID),
			cb.GlBlitFramebuffer(0, 0, GLint(inW), GLint(inH), 0, 0, GLint(outW), GLint(outH), bufferBits, sampling),
		)
		t.glBindFramebuffer_Read(ctx, framebufferID)
	}

	if hasDepth && version.IsES {
		copyDepthToColorGLES(ctx, dID, thread, s, out, t, fbai, inW, inH)
	}

	if needFBQuery {
		// Can only query the FB if bound to GL_FRAMEBUFFER. See b/115574126.
		t.glBindFramebuffer_ReadToBoth(ctx)
	}

	t.setPackStorage(ctx, NewPixelStorageState(s.Arena,
		0, // ImageHeight
		0, // SkipImages
		0, // RowLength
		0, // SkipRows
		0, // SkipPixels
		1, // Alignment
	), 0)

	imageSize := imgFmt.Size(int(outW), int(outH), 1)
	bufferSize := imageSize
	if needFBQuery {
		// Since we have no idea what format the data will be in, we need to be
		// as pessimistic as possible. Assume 4 channels at 4 bytes each, which
		// is the maximum possible by GLES. We also need space to store the
		// format and type (2 4byte enums) as well as the GL error (4 byte enum).
		bufferSize = 16*int(outW)*int(outH) + 8 + 4
	}

	tmp := s.AllocOrPanic(ctx, uint64(bufferSize))
	defer tmp.Free()

	out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		b.ReserveMemory(tmp.Range())

		if needFBQuery {
			ptr := value.ObservedPointer(tmp.Address())
			// Clear the GL error
			b.Call(funcInfoGlGetError)
			b.Pop(1)
			// Query the driver for the format and type that glReadPixels supports.
			b.Push(value.U32(GLenum_GL_IMPLEMENTATION_COLOR_READ_FORMAT))
			b.Push(ptr.Offset(0))
			b.Call(funcInfoGlGetIntegerv)

			b.Push(value.U32(GLenum_GL_IMPLEMENTATION_COLOR_READ_TYPE))
			b.Push(ptr.Offset(4))
			b.Call(funcInfoGlGetIntegerv)

			// Call glReadPixels with the above returned format and type.
			b.Push(value.S32(0))                        // x
			b.Push(value.S32(0))                        // y
			b.Push(value.S32(outW))                     // width
			b.Push(value.S32(outH))                     // height
			b.Load(protocol.Type_Uint32, ptr.Offset(0)) // format
			b.Load(protocol.Type_Uint32, ptr.Offset(4)) // type
			b.Push(ptr.Offset(12))                      // pixels
			b.Call(funcInfoGlReadPixels)
			// Get the GL error.
			b.Call(funcInfoGlGetError)
			b.Store(ptr.Offset(8))
		} else {
			// We use Call() directly here because on host replay, we are calling
			// glReadPixels with depth formats which are not legal for GLES.
			cb.GlReadPixels(0, 0, GLsizei(outW), GLsizei(outH), unsizedFormat, ty, tmp.Ptr()).
				Call(ctx, s, b)
		}

		b.Post(value.ObservedPointer(tmp.Address()), uint64(bufferSize), func(r binary.Reader, err error) {
			res.Do(func() (interface{}, error) {
				if err != nil {
					return nil, err
				}

				u, t, e := unsizedFormat, ty, GLenum_GL_NO_ERROR
				if needFBQuery {
					u, t, e = GLenum(r.Int32()), GLenum(r.Int32()), GLenum(r.Int32())
					bufferSize -= 12
				}
				if e != GLenum_GL_NO_ERROR {
					return nil, fmt.Errorf("GL error reading pixels from framebuffer: %s", e)
				}

				data := make([]byte, bufferSize)
				r.Data(data)
				if err := r.Error(); err != nil {
					return nil, fmt.Errorf("Could not read framebuffer data (expected length %d bytes): %v", bufferSize, err)
				}

				if u == unsizedFormat && t == ty {
					return &image.Data{
						Bytes:  data[:imageSize],
						Width:  uint32(outW),
						Height: uint32(outH),
						Depth:  1,
						Format: imgFmt,
					}, nil
				}

				// We had to query the driver for the format and it is different
				// than what was requested, So, we need to convert it. Also
				// handles the case where we read depth as color.

				f, err := getImageFormat(u, t)
				if err != nil {
					return nil, err
				}

				img := &image.Data{
					Bytes:  data[:f.Size(int(outW), int(outH), 1)],
					Width:  uint32(outW),
					Height: uint32(outH),
					Depth:  1,
					Format: f,
				}

				if hasDepth {
					// Filter out the red channel and then pretend it's just depth.
					redFmt := filterUncompressedImageFormat(f, func(c stream.Channel) bool {
						return c == stream.Channel_Red
					})
					var err error
					if img, err = img.Convert(redFmt); err != nil {
						return nil, err
					}
					depthFmt, err := getImageFormat(GLenum_GL_DEPTH_COMPONENT, t)
					if err != nil {
						return nil, err
					}
					img.Format = depthFmt
					return img, nil
				}

				return img.Convert(imgFmt)
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
