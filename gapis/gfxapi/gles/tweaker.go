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

	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
)

// tweaker provides a set of methods for temporarily changing the GLES state.
type tweaker struct {
	ctx  context.Context // Needed so functions match gl signature
	out  transform.Writer
	cb   CommandBuilder
	dID  atom.ID // Derived ID to use for generated atoms. Can be NoID.
	s    *gfxapi.State
	c    *Context
	undo []func()
}

func newTweaker(ctx context.Context, out transform.Writer, id atom.ID, cb CommandBuilder) *tweaker {
	s := out.State()
	c := GetContext(s)
	dID := id.Derived()
	return &tweaker{ctx: ctx, out: out, cb: cb, dID: dID, s: s, c: c}
}

// revert undoes all the changes made by the tweaker.
func (t *tweaker) revert() {
	for i := len(t.undo) - 1; i >= 0; i-- {
		t.undo[i]()
	}
	t.undo = nil
}

func (t *tweaker) doAndUndo(do atom.Atom, undo atom.Atom) {
	t.out.MutateAndWrite(t.ctx, t.dID, do)
	t.undo = append(t.undo, func() {
		t.out.MutateAndWrite(t.ctx, t.dID, undo)
	})
}

func (t *tweaker) AllocData(v ...interface{}) atom.AllocResult {
	tmp := atom.Must(atom.AllocData(t.ctx, t.s, v...))
	t.undo = append(t.undo, func() {
		tmp.Free()
	})
	return tmp
}

func (t *tweaker) getCapability(name GLenum) bool {
	a := t.cb.GlIsEnabled(name, 0)
	o := a.Extras().Observations()
	s := t.out.State()
	i := GLuint(0) // capability index.
	res, err := subGetCapability(t.ctx, a, o, s, GetState(s), nil, name, i)
	if err != nil {
		panic(err)
	}
	return res != 0
}

func (t *tweaker) glEnable(name GLenum) {
	// TODO: This does not correctly handle indexed state.
	if o := t.getCapability(name); o != true {
		t.doAndUndo(
			t.cb.GlEnable(name),
			t.cb.GlDisable(name))
	}
}

func (t *tweaker) glDisable(name GLenum) {
	// TODO: This does not correctly handle indexed state.
	if o := t.getCapability(name); o != false {
		t.doAndUndo(
			t.cb.GlDisable(name),
			t.cb.GlEnable(name))
	}
}

func (t *tweaker) glDepthMask(v GLboolean) {
	if o := t.c.Pixel.DepthWritemask; o != v {
		t.doAndUndo(
			t.cb.GlDepthMask(v),
			t.cb.GlDepthMask(o))
	}
}

func (t *tweaker) glDepthFunc(v GLenum) {
	if o := t.c.Pixel.Depth.Func; o != v {
		t.doAndUndo(
			t.cb.GlDepthFunc(v),
			t.cb.GlDepthFunc(o))
	}
}

func (t *tweaker) glBlendColor(r, g, b, a GLfloat) {
	n := Color{Red: r, Green: g, Blue: b, Alpha: a}
	if o := t.c.Pixel.BlendColor; o != n {
		t.doAndUndo(
			t.cb.GlBlendColor(r, g, b, a),
			t.cb.GlBlendColor(o.Red, o.Green, o.Blue, o.Alpha))
	}
}

func (t *tweaker) glBlendFunc(src, dst GLenum) {
	t.glBlendFuncSeparate(src, dst, src, dst)
}

func (t *tweaker) glBlendFuncSeparate(srcRGB, dstRGB, srcA, dstA GLenum) {
	// TODO: This does not correctly handle indexed state.
	o := t.c.Pixel.Blend[0]
	n := o
	n.SrcRgb, n.DstRgb, n.SrcAlpha, n.DstAlpha = srcRGB, dstRGB, srcA, dstA
	if o != n {
		t.doAndUndo(
			t.cb.GlBlendFuncSeparate(srcRGB, dstRGB, srcA, dstA),
			t.cb.GlBlendFuncSeparate(o.SrcRgb, o.DstRgb, o.SrcAlpha, o.DstAlpha))
	}
}

// glPolygonOffset adjusts the offset depth factor and units. Unlike the original glPolygonOffset,
// this function adds the given values to the current values rather than setting them.
func (t *tweaker) glPolygonOffset(factor, units GLfloat) {
	origFactor, origUnits := t.c.Rasterization.PolygonOffsetFactor, t.c.Rasterization.PolygonOffsetUnits
	t.doAndUndo(
		t.cb.GlPolygonOffset(origFactor+factor, origUnits+units),
		t.cb.GlPolygonOffset(origFactor, origUnits))
}

func (t *tweaker) glLineWidth(width GLfloat) {
	if o := t.c.Rasterization.LineWidth; o != width {
		t.doAndUndo(
			t.cb.GlLineWidth(width),
			t.cb.GlLineWidth(o))
	}
}

// This will either bind new VAO (GLES 3.x) or save state of the default one (GLES 2.0).
func (t *tweaker) makeVertexArray(enabledLocations ...AttributeLocation) {
	ctx := t.ctx
	if t.c.Constants.MajorVersion >= 3 {
		// GLES 3.0 and 3.1 introduce a lot of new state which would be hard to restore.
		// It is much easier to just create a fresh Vertex Array Object to work with.
		vertexArrayID := t.glGenVertexArray()
		t.glBindVertexArray(vertexArrayID)
		for _, location := range enabledLocations {
			t.out.MutateAndWrite(ctx, t.dID, t.cb.GlEnableVertexAttribArray(location))
		}
	} else {
		// GLES 2.0 does not have Vertex Array Objects, but the state is fairly simple.
		vao := t.c.Bound.VertexArray
		// Disable all vertex attribute arrays
		for location, origVertexAttrib := range vao.VertexAttributeArrays {
			if origVertexAttrib.Enabled == GLboolean_GL_TRUE {
				t.doAndUndo(
					t.cb.GlDisableVertexAttribArray(location),
					t.cb.GlEnableVertexAttribArray(location))
			}
		}
		// Enable and save state for the attribute arrays that we will use
		origArrayBufferID := t.c.Bound.ArrayBuffer.GetID()
		for _, location := range enabledLocations {
			location := location
			t.doAndUndo(
				t.cb.GlEnableVertexAttribArray(location),
				t.cb.GlDisableVertexAttribArray(location))
			origVertexAttrib := *(vao.VertexAttributeArrays[location])
			origVertexBinding := *(vao.VertexBufferBindings[VertexBufferBindingIndex(location)])
			t.undo = append(t.undo, func() {
				t.out.MutateAndWrite(ctx, t.dID, t.cb.GlBindBuffer(GLenum_GL_ARRAY_BUFFER, origVertexBinding.Buffer))
				t.out.MutateAndWrite(ctx, t.dID, t.cb.GlVertexAttribPointer(location, origVertexAttrib.Size, origVertexAttrib.Type, origVertexAttrib.Normalized, origVertexAttrib.Stride, memory.Pointer(origVertexAttrib.Pointer)))
				t.out.MutateAndWrite(ctx, t.dID, t.cb.GlBindBuffer(GLenum_GL_ARRAY_BUFFER, origArrayBufferID))
			})
		}
	}
}

func (t *tweaker) glGenBuffer() BufferId {
	id := BufferId(newUnusedID(t.ctx, 'B', func(x uint32) bool { return t.c.Objects.Shared.Buffers[BufferId(x)] != nil }))
	tmp := t.AllocData(id)
	t.doAndUndo(
		t.cb.GlGenBuffers(1, tmp.Ptr()).AddWrite(tmp.Data()),
		t.cb.GlDeleteBuffers(1, tmp.Ptr()).AddRead(tmp.Data()))
	return id
}

func (t *tweaker) glGenRenderbuffer() RenderbufferId {
	id := RenderbufferId(newUnusedID(t.ctx, 'R', func(x uint32) bool { return t.c.Objects.Shared.Renderbuffers[RenderbufferId(x)] != nil }))
	tmp := t.AllocData(id)
	t.doAndUndo(
		t.cb.GlGenRenderbuffers(1, tmp.Ptr()).AddWrite(tmp.Data()),
		t.cb.GlDeleteRenderbuffers(1, tmp.Ptr()).AddRead(tmp.Data()))
	return id
}

func (t *tweaker) glGenFramebuffer() FramebufferId {
	id := FramebufferId(newUnusedID(t.ctx, 'F', func(x uint32) bool { return t.c.Objects.Framebuffers[FramebufferId(x)] != nil }))
	tmp := t.AllocData(id)
	t.doAndUndo(
		t.cb.GlGenFramebuffers(1, tmp.Ptr()).AddWrite(tmp.Data()),
		t.cb.GlDeleteFramebuffers(1, tmp.Ptr()).AddRead(tmp.Data()))
	return id
}

func (t *tweaker) glGenTexture() TextureId {
	id := TextureId(newUnusedID(t.ctx, 'T', func(x uint32) bool { return t.c.Objects.Shared.Textures[TextureId(x)] != nil }))
	tmp := t.AllocData(id)
	t.doAndUndo(
		t.cb.GlGenTextures(1, tmp.Ptr()).AddWrite(tmp.Data()),
		t.cb.GlDeleteTextures(1, tmp.Ptr()).AddRead(tmp.Data()))
	return id
}

func (t *tweaker) glGenVertexArray() VertexArrayId {
	id := VertexArrayId(newUnusedID(t.ctx, 'V', func(x uint32) bool { return t.c.Objects.VertexArrays[VertexArrayId(x)] != nil }))
	tmp := t.AllocData(id)
	t.doAndUndo(
		t.cb.GlGenVertexArrays(1, tmp.Ptr()).AddWrite(tmp.Data()),
		t.cb.GlDeleteVertexArrays(1, tmp.Ptr()).AddRead(tmp.Data()))
	return id
}

func (t *tweaker) glCreateProgram() ProgramId {
	id := ProgramId(newUnusedID(t.ctx, 'P', func(x uint32) bool {
		return t.c.Objects.Shared.Programs[ProgramId(x)] != nil || t.c.Objects.Shared.Shaders[ShaderId(x)] != nil
	}))
	t.doAndUndo(
		t.cb.GlCreateProgram(id),
		t.cb.GlDeleteProgram(id))
	return id
}

func (t *tweaker) makeProgram(vertexShaderSource, fragmentShaderSource string) ProgramId {
	programID := t.glCreateProgram()
	vertexShaderID := t.glCreateShader(GLenum_GL_VERTEX_SHADER)
	t.glShaderSource(vertexShaderID, vertexShaderSource)
	t.out.MutateAndWrite(t.ctx, t.dID, t.cb.GlCompileShader(vertexShaderID))
	fragmentShaderID := t.glCreateShader(GLenum_GL_FRAGMENT_SHADER)
	t.glShaderSource(fragmentShaderID, fragmentShaderSource)
	t.out.MutateAndWrite(t.ctx, t.dID, t.cb.GlCompileShader(fragmentShaderID))
	t.out.MutateAndWrite(t.ctx, t.dID, t.cb.GlAttachShader(programID, vertexShaderID))
	t.out.MutateAndWrite(t.ctx, t.dID, t.cb.GlAttachShader(programID, fragmentShaderID))
	return programID
}

func (t *tweaker) glCreateShader(shaderType GLenum) ShaderId {
	id := ShaderId(newUnusedID(t.ctx, 'S', func(x uint32) bool {
		return t.c.Objects.Shared.Programs[ProgramId(x)] != nil || t.c.Objects.Shared.Shaders[ShaderId(x)] != nil
	}))
	// We need to mutate the state, as otherwise two consecutive calls can return the same ShaderId.
	t.doAndUndo(
		t.cb.GlCreateShader(shaderType, id),
		t.cb.GlDeleteShader(id))
	return id
}

func (t *tweaker) glShaderSource(shaderID ShaderId, shaderSource string) {
	tmpSrc := t.AllocData(shaderSource)
	tmpSrcLen := t.AllocData(GLint(len(shaderSource)))
	tmpPtrToSrc := t.AllocData(tmpSrc.Ptr())
	t.out.MutateAndWrite(t.ctx, t.dID, t.cb.GlShaderSource(shaderID, 1, tmpPtrToSrc.Ptr(), tmpSrcLen.Ptr()).
		AddRead(tmpPtrToSrc.Data()).
		AddRead(tmpSrcLen.Data()).
		AddRead(tmpSrc.Data()))
	return
}

func (t *tweaker) glScissor(x, y GLint, w, h GLsizei) {
	v := Rect{X: x, Y: y, Width: w, Height: h}
	if o := t.c.Pixel.Scissor.Box; o != v {
		t.doAndUndo(
			t.cb.GlScissor(x, y, w, h),
			t.cb.GlScissor(o.X, o.Y, o.Width, o.Height))
	}
}

func (t *tweaker) GlBindBuffer_ArrayBuffer(id BufferId) {
	if o := t.c.Bound.ArrayBuffer.GetID(); o != id {
		t.doAndUndo(
			t.cb.GlBindBuffer(GLenum_GL_ARRAY_BUFFER, id),
			t.cb.GlBindBuffer(GLenum_GL_ARRAY_BUFFER, o))
	}
}

func (t *tweaker) GlBindBuffer_ElementArrayBuffer(id BufferId) {
	vao := t.c.Bound.VertexArray
	if o := vao.ElementArrayBuffer.GetID(); o != id {
		t.doAndUndo(
			t.cb.GlBindBuffer(GLenum_GL_ELEMENT_ARRAY_BUFFER, id),
			t.cb.GlBindBuffer(GLenum_GL_ELEMENT_ARRAY_BUFFER, o))
	}
}

func (t *tweaker) glBindFramebuffer_Draw(id FramebufferId) {
	if o := t.c.Bound.DrawFramebuffer.GetID(); o != id {
		t.doAndUndo(
			t.cb.GlBindFramebuffer(GLenum_GL_DRAW_FRAMEBUFFER, id),
			t.cb.GlBindFramebuffer(GLenum_GL_DRAW_FRAMEBUFFER, o))
	}
}

func (t *tweaker) glBindFramebuffer_Read(id FramebufferId) {
	if o := t.c.Bound.ReadFramebuffer.GetID(); o != id {
		t.doAndUndo(
			t.cb.GlBindFramebuffer(GLenum_GL_READ_FRAMEBUFFER, id),
			t.cb.GlBindFramebuffer(GLenum_GL_READ_FRAMEBUFFER, o))
	}
}

func (t *tweaker) glReadBuffer(id GLenum) {
	fb := t.c.Bound.ReadFramebuffer
	if o := fb.ReadBuffer; o != id {
		t.doAndUndo(
			t.cb.GlReadBuffer(id),
			t.cb.GlReadBuffer(o))
	}
}

func (t *tweaker) glBindRenderbuffer(id RenderbufferId) {
	if o := t.c.Bound.Renderbuffer.GetID(); o != id {
		t.doAndUndo(
			t.cb.GlBindRenderbuffer(GLenum_GL_RENDERBUFFER, id),
			t.cb.GlBindRenderbuffer(GLenum_GL_RENDERBUFFER, o))
	}
}

func (t *tweaker) glBindTexture_2D(id TextureId) {
	if o := t.c.Bound.TextureUnit.Binding2d.GetID(); o != id {
		t.doAndUndo(
			t.cb.GlBindTexture(GLenum_GL_TEXTURE_2D, id),
			t.cb.GlBindTexture(GLenum_GL_TEXTURE_2D, o))
	}
}

func (t *tweaker) glBindVertexArray(id VertexArrayId) {
	if o := t.c.Bound.VertexArray.GetID(); o != id {
		t.doAndUndo(
			t.cb.GlBindVertexArray(id),
			t.cb.GlBindVertexArray(o))
	}
}

func (t *tweaker) glUseProgram(id ProgramId) {
	if o := t.c.Bound.Program.GetID(); o != id {
		t.doAndUndo(
			t.cb.GlUseProgram(id),
			t.cb.GlUseProgram(o))
	}
}

func (t *tweaker) glActiveTexture(unit GLenum) {
	if o := GLenum(t.c.Bound.TextureUnit.ID) + GLenum_GL_TEXTURE0; o != unit {
		t.doAndUndo(
			t.cb.GlActiveTexture(unit),
			t.cb.GlActiveTexture(o))
	}
}

func (t *tweaker) setPackStorage(state PixelStorageState, bufferId BufferId) {
	origState := map[GLenum]GLint{}
	forEachPackStorageState(t.c.Other.Pack, func(n GLenum, v GLint) { origState[n] = v })
	forEachPackStorageState(state, func(name GLenum, value GLint) {
		if o := origState[name]; o != value {
			t.doAndUndo(
				t.cb.GlPixelStorei(name, value),
				t.cb.GlPixelStorei(name, o))
		}
	})
	if o := t.c.Bound.PixelPackBuffer.GetID(); o != bufferId {
		t.doAndUndo(
			t.cb.GlBindBuffer(GLenum_GL_PIXEL_PACK_BUFFER, bufferId),
			t.cb.GlBindBuffer(GLenum_GL_PIXEL_PACK_BUFFER, o))
	}
}

func forEachPackStorageState(state PixelStorageState, action func(n GLenum, v GLint)) {
	action(GLenum_GL_PACK_ALIGNMENT, state.Alignment)
	action(GLenum_GL_PACK_IMAGE_HEIGHT, state.ImageHeight)
	action(GLenum_GL_PACK_ROW_LENGTH, state.RowLength)
	action(GLenum_GL_PACK_SKIP_IMAGES, state.SkipImages)
	action(GLenum_GL_PACK_SKIP_PIXELS, state.SkipPixels)
	action(GLenum_GL_PACK_SKIP_ROWS, state.SkipRows)
}

func (t *tweaker) setUnpackStorage(state PixelStorageState, bufferId BufferId) {
	origState := map[GLenum]GLint{}
	forEachUnpackStorageState(t.c.Other.Unpack, func(n GLenum, v GLint) { origState[n] = v })
	forEachUnpackStorageState(state, func(name GLenum, value GLint) {
		if o := origState[name]; o != value {
			t.doAndUndo(
				t.cb.GlPixelStorei(name, value),
				t.cb.GlPixelStorei(name, o))
		}
	})
	if o := t.c.Bound.PixelUnpackBuffer.GetID(); o != bufferId {
		t.doAndUndo(
			t.cb.GlBindBuffer(GLenum_GL_PIXEL_UNPACK_BUFFER, bufferId),
			t.cb.GlBindBuffer(GLenum_GL_PIXEL_UNPACK_BUFFER, o))
	}
}

func forEachUnpackStorageState(state PixelStorageState, action func(n GLenum, v GLint)) {
	action(GLenum_GL_UNPACK_ALIGNMENT, state.Alignment)
	action(GLenum_GL_UNPACK_IMAGE_HEIGHT, state.ImageHeight)
	action(GLenum_GL_UNPACK_ROW_LENGTH, state.RowLength)
	action(GLenum_GL_UNPACK_SKIP_IMAGES, state.SkipImages)
	action(GLenum_GL_UNPACK_SKIP_PIXELS, state.SkipPixels)
	action(GLenum_GL_UNPACK_SKIP_ROWS, state.SkipRows)
}
