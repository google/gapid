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
	"sort"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
)

type readFbTask struct {
	at   api.CmdID
	work func(ctx context.Context, w transform.Writer)
}

type readFramebuffer struct {
	tasks       []readFbTask
	tasksSorted bool
}

func newReadFramebuffer(ctx context.Context) *readFramebuffer {
	return &readFramebuffer{}
}

func (t *readFramebuffer) addTask(at api.CmdID, work func(context.Context, transform.Writer)) {
	t.tasks = append(t.tasks, readFbTask{at, work})
	t.tasksSorted = false
}

func (t *readFramebuffer) sortTasks() {
	if !t.tasksSorted {
		sort.Slice(t.tasks, func(i, j int) bool { return t.tasks[i].at < t.tasks[j].at })
		t.tasksSorted = true
	}
}

func (t *readFramebuffer) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	if id.IsReal() {
		t.sortTasks()
		for len(t.tasks) > 0 && t.tasks[0].at < id {
			t.tasks[0].work(ctx, out)
			t.tasks = t.tasks[1:]
		}
	}
	out.MutateAndWrite(ctx, id, cmd)
}

func (t *readFramebuffer) Flush(ctx context.Context, out transform.Writer) {
	t.sortTasks()
	for _, task := range t.tasks {
		task.work(ctx, out)
	}
	t.tasks = nil
}

func (t *readFramebuffer) depth(
	id api.CmdID,
	thread uint64,
	fb FramebufferId,
	res replay.Result) {

	t.addTask(id, func(ctx context.Context, out transform.Writer) {
		s := out.State()
		c := GetContext(s, thread)

		if fb == 0 {
			fb = c.Bound.DrawFramebuffer.GetID()
		}

		width, height, format, err := GetState(s).getFramebufferAttachmentInfo(thread, fb, api.FramebufferAttachment_Depth)
		if err != nil {
			log.W(ctx, "Failed to read framebuffer after cmd %v: %v", id, err)
			res(nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()})
			return
		}

		t := newTweaker(out, id.Derived(), CommandBuilder{Thread: thread})
		defer t.revert(ctx)
		t.glBindFramebuffer_Read(ctx, fb)

		postColorData(ctx, s, int32(width), int32(height), format, out, id, thread, res)
	})
}

func (t *readFramebuffer) color(
	id api.CmdID,
	thread uint64,
	width, height uint32,
	fb FramebufferId,
	bufferIdx uint32,
	res replay.Result) {

	t.addTask(id, func(ctx context.Context, out transform.Writer) {
		s := out.State()
		c := GetContext(s, thread)

		if fb == 0 {
			fb = c.Bound.DrawFramebuffer.GetID()
		}

		attachment := api.FramebufferAttachment_Color0 + api.FramebufferAttachment(bufferIdx)
		w, h, fmt, err := GetState(s).getFramebufferAttachmentInfo(thread, fb, attachment)
		if err != nil {
			log.W(ctx, "Failed to read framebuffer after cmd %v: %v", id, err)
			res(nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()})
			return
		}
		if fmt == 0 {
			log.W(ctx, "Failed to read framebuffer after cmd %v: no image format", id)
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
		cb := CommandBuilder{Thread: thread}
		t := newTweaker(out, dID, cb)
		defer t.revert(ctx)
		t.glBindFramebuffer_Read(ctx, fb)

		// TODO: These glReadBuffer calls need to be changed for on-device
		//       replay. Note that glReadBuffer was only introduced in
		//       OpenGL ES 3.0, and that GL_FRONT is not a legal enum value.
		if c.Bound.DrawFramebuffer == c.Objects.Default.Framebuffer {
			out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.State, b *builder.Builder) error {
				// TODO: We assume here that the default framebuffer is
				//       single-buffered. Once we support double-buffering we
				//       need to decide whether to read from GL_FRONT or GL_BACK.
				cb.GlReadBuffer(GLenum_GL_FRONT).Call(ctx, s, b)
				return nil
			}))
		} else {
			t.glReadBuffer(ctx, GLenum_GL_COLOR_ATTACHMENT0+GLenum(bufferIdx))
		}

		if inW == outW && inH == outH {
			postColorData(ctx, s, outW, outH, fmt, out, id, thread, res)
		} else {
			t.glScissor(ctx, 0, 0, GLsizei(inW), GLsizei(inH))
			framebufferID := t.glGenFramebuffer(ctx)
			t.glBindFramebuffer_Draw(ctx, framebufferID)
			renderbufferID := t.glGenRenderbuffer(ctx)
			t.glBindRenderbuffer(ctx, renderbufferID)

			mutateAndWriteEach(ctx, out, dID,
				cb.GlRenderbufferStorage(GLenum_GL_RENDERBUFFER, fmt, GLsizei(outW), GLsizei(outH)),
				cb.GlFramebufferRenderbuffer(GLenum_GL_DRAW_FRAMEBUFFER, GLenum_GL_COLOR_ATTACHMENT0, GLenum_GL_RENDERBUFFER, renderbufferID),
				cb.GlBlitFramebuffer(0, 0, GLint(inW), GLint(inH), 0, 0, GLint(outW), GLint(outH), GLbitfield_GL_COLOR_BUFFER_BIT, GLenum_GL_LINEAR),
			)
			t.glBindFramebuffer_Read(ctx, framebufferID)

			postColorData(ctx, s, outW, outH, fmt, out, id, thread, res)
		}

	})
}

func postColorData(ctx context.Context,
	s *api.State,
	width, height int32,
	sizedFormat GLenum,
	out transform.Writer,
	id api.CmdID,
	thread uint64,
	res replay.Result) {

	unsizedFormat, ty := getUnsizedFormatAndType(sizedFormat)
	imgFmt, err := getImageFormat(unsizedFormat, ty)
	if err != nil {
		res(nil, err)
		return
	}

	dID := id.Derived()
	cb := CommandBuilder{Thread: thread}
	t := newTweaker(out, dID, cb)
	t.setPackStorage(ctx, PixelStorageState{Alignment: 1}, 0)

	imageSize := imgFmt.Size(int(width), int(height), 1)
	tmp := s.AllocOrPanic(ctx, uint64(imageSize))
	out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.State, b *builder.Builder) error {
		// TODO: We use Call() directly here because we are calling glReadPixels
		// with depth formats which are not legal for GLES. Once we're replaying
		// on-device again, we need to take a look at methods for reading the
		// depth buffer.

		b.ReserveMemory(tmp.Range())
		cb.GlReadPixels(0, 0, GLsizei(width), GLsizei(height), unsizedFormat, ty, tmp.Ptr()).
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

	out.MutateAndWrite(ctx, dID, cb.GlGetError(0)) // Check for errors.

	t.revert(ctx)
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
