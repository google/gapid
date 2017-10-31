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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
)

// tweaker provides a set of methods for temporarily changing the GLES state.
type tweaker struct {
	out  transform.Writer
	cb   CommandBuilder
	dID  api.CmdID // Derived ID to use for generated atoms. Can be NoID.
	s    *api.GlobalState
	c    *Context
	undo []func(context.Context)
}

func newTweaker(out transform.Writer, id api.CmdID, cb CommandBuilder) *tweaker {
	s := out.State()
	c := GetContext(s, cb.Thread)
	dID := id.Derived()
	return &tweaker{out: out, cb: cb, dID: dID, s: s, c: c}
}

// revert undoes all the changes made by the tweaker.
func (t *tweaker) revert(ctx context.Context) {
	for i := len(t.undo) - 1; i >= 0; i-- {
		t.undo[i](ctx)
	}
	t.undo = nil
}

func (t *tweaker) doAndUndo(ctx context.Context, do, undo api.Cmd) {
	t.out.MutateAndWrite(ctx, t.dID, do)
	t.undo = append(t.undo, func(ctx context.Context) {
		t.out.MutateAndWrite(ctx, t.dID, undo)
	})
}

func (t *tweaker) AllocData(ctx context.Context, v ...interface{}) api.AllocResult {
	tmp := t.s.AllocDataOrPanic(ctx, v...)
	t.undo = append(t.undo, func(ctx context.Context) { tmp.Free() })
	return tmp
}

func (t *tweaker) getCapability(ctx context.Context, name GLenum) bool {
	a := t.cb.GlIsEnabled(name, 0)
	o := a.Extras().Observations()
	s := t.out.State()
	i := GLuint(0) // capability index.
	res, err := subGetCapability(ctx, a, t.dID, o, s, GetState(s), a.thread, nil, name, i)
	if err != nil {
		panic(err)
	}
	return res != 0
}

func (t *tweaker) glEnable(ctx context.Context, name GLenum) {
	// TODO: This does not correctly handle indexed state.
	if o := t.getCapability(ctx, name); o != true {
		t.doAndUndo(ctx,
			t.cb.GlEnable(name),
			t.cb.GlDisable(name))
	}
}

func (t *tweaker) glDisable(ctx context.Context, name GLenum) {
	// TODO: This does not correctly handle indexed state.
	if o := t.getCapability(ctx, name); o != false {
		t.doAndUndo(ctx,
			t.cb.GlDisable(name),
			t.cb.GlEnable(name))
	}
}

func (t *tweaker) glDepthMask(ctx context.Context, v GLboolean) {
	if o := t.c.Pixel.DepthWritemask; o != v {
		t.doAndUndo(ctx,
			t.cb.GlDepthMask(v),
			t.cb.GlDepthMask(o))
	}
}

func (t *tweaker) glDepthFunc(ctx context.Context, v GLenum) {
	if o := t.c.Pixel.Depth.Func; o != v {
		t.doAndUndo(ctx,
			t.cb.GlDepthFunc(v),
			t.cb.GlDepthFunc(o))
	}
}

func (t *tweaker) glBlendColor(ctx context.Context, r, g, b, a GLfloat) {
	n := Color{Red: r, Green: g, Blue: b, Alpha: a}
	if o := t.c.Pixel.BlendColor; o != n {
		t.doAndUndo(ctx,
			t.cb.GlBlendColor(r, g, b, a),
			t.cb.GlBlendColor(o.Red, o.Green, o.Blue, o.Alpha))
	}
}

func (t *tweaker) glBlendFunc(ctx context.Context, src, dst GLenum) {
	t.glBlendFuncSeparate(ctx, src, dst, src, dst)
}

func (t *tweaker) glBlendFuncSeparate(ctx context.Context, srcRGB, dstRGB, srcA, dstA GLenum) {
	// TODO: This does not correctly handle indexed state.
	o := t.c.Pixel.Blend.Get(0)
	n := o
	n.SrcRgb, n.DstRgb, n.SrcAlpha, n.DstAlpha = srcRGB, dstRGB, srcA, dstA
	if o != n {
		t.doAndUndo(ctx,
			t.cb.GlBlendFuncSeparate(srcRGB, dstRGB, srcA, dstA),
			t.cb.GlBlendFuncSeparate(o.SrcRgb, o.DstRgb, o.SrcAlpha, o.DstAlpha))
	}
}

// glPolygonOffset adjusts the offset depth factor and units. Unlike the original glPolygonOffset,
// this function adds the given values to the current values rather than setting them.
func (t *tweaker) glPolygonOffset(ctx context.Context, factor, units GLfloat) {
	origFactor, origUnits := t.c.Rasterization.PolygonOffsetFactor, t.c.Rasterization.PolygonOffsetUnits
	t.doAndUndo(ctx,
		t.cb.GlPolygonOffset(origFactor+factor, origUnits+units),
		t.cb.GlPolygonOffset(origFactor, origUnits))
}

func (t *tweaker) glLineWidth(ctx context.Context, width GLfloat) {
	if o := t.c.Rasterization.LineWidth; o != width {
		t.doAndUndo(ctx,
			t.cb.GlLineWidth(width),
			t.cb.GlLineWidth(o))
	}
}

// This will either bind new VAO (GLES 3.x) or save state of the default one (GLES 2.0).
func (t *tweaker) makeVertexArray(ctx context.Context, enabledLocations ...AttributeLocation) {
	if t.c.Constants.MajorVersion >= 3 {
		// GLES 3.0 and 3.1 introduce a lot of new state which would be hard to restore.
		// It is much easier to just create a fresh Vertex Array Object to work with.
		vertexArrayID := t.glGenVertexArray(ctx)
		t.glBindVertexArray(ctx, vertexArrayID)
		for _, location := range enabledLocations {
			t.out.MutateAndWrite(ctx, t.dID, t.cb.GlEnableVertexAttribArray(location))
		}
	} else {
		// GLES 2.0 does not have Vertex Array Objects, but the state is fairly simple.
		vao := t.c.Bound.VertexArray
		// Disable all vertex attribute arrays
		for location, origVertexAttrib := range vao.VertexAttributeArrays.Range() {
			if origVertexAttrib.Enabled == GLboolean_GL_TRUE {
				t.doAndUndo(ctx,
					t.cb.GlDisableVertexAttribArray(location),
					t.cb.GlEnableVertexAttribArray(location))
			}
		}
		// Enable and save state for the attribute arrays that we will use
		origArrayBufferID := t.c.Bound.ArrayBuffer.GetID()
		for _, location := range enabledLocations {
			location := location
			t.doAndUndo(ctx,
				t.cb.GlEnableVertexAttribArray(location),
				t.cb.GlDisableVertexAttribArray(location))
			origVertexAttrib := *(vao.VertexAttributeArrays.Get(location))
			origVertexBinding := *(vao.VertexBufferBindings.Get(VertexBufferBindingIndex(location)))
			t.undo = append(t.undo, func(ctx context.Context) {
				t.out.MutateAndWrite(ctx, t.dID, t.cb.GlBindBuffer(GLenum_GL_ARRAY_BUFFER, origVertexBinding.Buffer))
				t.out.MutateAndWrite(ctx, t.dID, t.cb.GlVertexAttribPointer(location, origVertexAttrib.Size, origVertexAttrib.Type, origVertexAttrib.Normalized, origVertexAttrib.Stride, memory.Pointer(origVertexAttrib.Pointer)))
				t.out.MutateAndWrite(ctx, t.dID, t.cb.GlBindBuffer(GLenum_GL_ARRAY_BUFFER, origArrayBufferID))
			})
		}
	}
}

func (t *tweaker) glGenBuffer(ctx context.Context) BufferId {
	id := BufferId(newUnusedID(ctx, 'B', func(x uint32) bool { return t.c.Objects.Buffers.Get(BufferId(x)) != nil }))
	tmp := t.AllocData(ctx, id)
	t.doAndUndo(ctx,
		t.cb.GlGenBuffers(1, tmp.Ptr()).AddWrite(tmp.Data()),
		t.cb.GlDeleteBuffers(1, tmp.Ptr()).AddRead(tmp.Data()))
	return id
}

func (t *tweaker) glGenRenderbuffer(ctx context.Context) RenderbufferId {
	id := RenderbufferId(newUnusedID(ctx, 'R', func(x uint32) bool { return t.c.Objects.Renderbuffers.Get(RenderbufferId(x)) != nil }))
	tmp := t.AllocData(ctx, id)
	t.doAndUndo(ctx,
		t.cb.GlGenRenderbuffers(1, tmp.Ptr()).AddWrite(tmp.Data()),
		t.cb.GlDeleteRenderbuffers(1, tmp.Ptr()).AddRead(tmp.Data()))
	return id
}

func (t *tweaker) glGenFramebuffer(ctx context.Context) FramebufferId {
	id := FramebufferId(newUnusedID(ctx, 'F', func(x uint32) bool { return t.c.Objects.Framebuffers.Get(FramebufferId(x)) != nil }))
	tmp := t.AllocData(ctx, id)
	t.doAndUndo(ctx,
		t.cb.GlGenFramebuffers(1, tmp.Ptr()).AddWrite(tmp.Data()),
		t.cb.GlDeleteFramebuffers(1, tmp.Ptr()).AddRead(tmp.Data()))
	return id
}

func (t *tweaker) glGenTexture(ctx context.Context) TextureId {
	id := TextureId(newUnusedID(ctx, 'T', func(x uint32) bool { return t.c.Objects.Textures.Get(TextureId(x)) != nil }))
	tmp := t.AllocData(ctx, id)
	t.doAndUndo(ctx,
		t.cb.GlGenTextures(1, tmp.Ptr()).AddWrite(tmp.Data()),
		t.cb.GlDeleteTextures(1, tmp.Ptr()).AddRead(tmp.Data()))
	return id
}

func (t *tweaker) glGenVertexArray(ctx context.Context) VertexArrayId {
	id := VertexArrayId(newUnusedID(ctx, 'V', func(x uint32) bool { return t.c.Objects.VertexArrays.Get(VertexArrayId(x)) != nil }))
	tmp := t.AllocData(ctx, id)
	t.doAndUndo(ctx,
		t.cb.GlGenVertexArrays(1, tmp.Ptr()).AddWrite(tmp.Data()),
		t.cb.GlDeleteVertexArrays(1, tmp.Ptr()).AddRead(tmp.Data()))
	return id
}

func (t *tweaker) glCreateProgram(ctx context.Context) ProgramId {
	id := ProgramId(newUnusedID(ctx, 'P', func(x uint32) bool {
		return t.c.Objects.Programs.Get(ProgramId(x)) != nil || t.c.Objects.Shaders.Get(ShaderId(x)) != nil
	}))
	t.doAndUndo(ctx,
		t.cb.GlCreateProgram(id),
		t.cb.GlDeleteProgram(id))
	return id
}

func (t *tweaker) makeProgram(ctx context.Context, vertexShaderSource, fragmentShaderSource string) ProgramId {
	programID := t.glCreateProgram(ctx)
	vertexShaderID := t.glCreateShader(ctx, GLenum_GL_VERTEX_SHADER)
	t.glShaderSource(ctx, vertexShaderID, vertexShaderSource)
	t.out.MutateAndWrite(ctx, t.dID, t.cb.GlCompileShader(vertexShaderID))
	fragmentShaderID := t.glCreateShader(ctx, GLenum_GL_FRAGMENT_SHADER)
	t.glShaderSource(ctx, fragmentShaderID, fragmentShaderSource)
	t.out.MutateAndWrite(ctx, t.dID, t.cb.GlCompileShader(fragmentShaderID))
	t.out.MutateAndWrite(ctx, t.dID, t.cb.GlAttachShader(programID, vertexShaderID))
	t.out.MutateAndWrite(ctx, t.dID, t.cb.GlAttachShader(programID, fragmentShaderID))
	return programID
}

func (t *tweaker) glCreateShader(ctx context.Context, shaderType GLenum) ShaderId {
	id := ShaderId(newUnusedID(ctx, 'S', func(x uint32) bool {
		return t.c.Objects.Programs.Get(ProgramId(x)) != nil || t.c.Objects.Shaders.Get(ShaderId(x)) != nil
	}))
	// We need to mutate the state, as otherwise two consecutive calls can return the same ShaderId.
	t.doAndUndo(ctx,
		t.cb.GlCreateShader(shaderType, id),
		t.cb.GlDeleteShader(id))
	return id
}

func (t *tweaker) glShaderSource(ctx context.Context, shaderID ShaderId, shaderSource string) {
	tmpSrc := t.AllocData(ctx, shaderSource)
	tmpSrcLen := t.AllocData(ctx, GLint(len(shaderSource)))
	tmpPtrToSrc := t.AllocData(ctx, tmpSrc.Ptr())
	t.out.MutateAndWrite(ctx, t.dID, t.cb.GlShaderSource(shaderID, 1, tmpPtrToSrc.Ptr(), tmpSrcLen.Ptr()).
		AddRead(tmpPtrToSrc.Data()).
		AddRead(tmpSrcLen.Data()).
		AddRead(tmpSrc.Data()))
	return
}

func (t *tweaker) glScissor(ctx context.Context, x, y GLint, w, h GLsizei) {
	v := Rect{X: x, Y: y, Width: w, Height: h}
	if o := t.c.Pixel.Scissor.Box; o != v {
		t.doAndUndo(ctx,
			t.cb.GlScissor(x, y, w, h),
			t.cb.GlScissor(o.X, o.Y, o.Width, o.Height))
	}
}

func (t *tweaker) GlBindBuffer_ArrayBuffer(ctx context.Context, id BufferId) {
	if o := t.c.Bound.ArrayBuffer.GetID(); o != id {
		t.doAndUndo(ctx,
			t.cb.GlBindBuffer(GLenum_GL_ARRAY_BUFFER, id),
			t.cb.GlBindBuffer(GLenum_GL_ARRAY_BUFFER, o))
	}
}

func (t *tweaker) GlBindBuffer_ElementArrayBuffer(ctx context.Context, id BufferId) {
	vao := t.c.Bound.VertexArray
	if o := vao.ElementArrayBuffer.GetID(); o != id {
		t.doAndUndo(ctx,
			t.cb.GlBindBuffer(GLenum_GL_ELEMENT_ARRAY_BUFFER, id),
			t.cb.GlBindBuffer(GLenum_GL_ELEMENT_ARRAY_BUFFER, o))
	}
}

func (t *tweaker) glBindFramebuffer_Draw(ctx context.Context, id FramebufferId) {
	if o := t.c.Bound.DrawFramebuffer.GetID(); o != id {
		t.doAndUndo(ctx,
			t.cb.GlBindFramebuffer(GLenum_GL_DRAW_FRAMEBUFFER, id),
			t.cb.GlBindFramebuffer(GLenum_GL_DRAW_FRAMEBUFFER, o))
	}
}

func (t *tweaker) glBindFramebuffer_Read(ctx context.Context, id FramebufferId) {
	if o := t.c.Bound.ReadFramebuffer.GetID(); o != id {
		t.doAndUndo(ctx,
			t.cb.GlBindFramebuffer(GLenum_GL_READ_FRAMEBUFFER, id),
			t.cb.GlBindFramebuffer(GLenum_GL_READ_FRAMEBUFFER, o))
	}
}

func (t *tweaker) glReadBuffer(ctx context.Context, id GLenum) {
	fb := t.c.Bound.ReadFramebuffer
	if o := fb.ReadBuffer; o != id {
		t.doAndUndo(ctx,
			t.cb.GlReadBuffer(id),
			t.cb.GlReadBuffer(o))
	}
}

func (t *tweaker) glBindRenderbuffer(ctx context.Context, id RenderbufferId) {
	if o := t.c.Bound.Renderbuffer.GetID(); o != id {
		t.doAndUndo(ctx,
			t.cb.GlBindRenderbuffer(GLenum_GL_RENDERBUFFER, id),
			t.cb.GlBindRenderbuffer(GLenum_GL_RENDERBUFFER, o))
	}
}

func (t *tweaker) glBindTexture(ctx context.Context, tex *Texture) {
	var old TextureId
	switch tex.Kind {
	case GLenum_GL_TEXTURE_2D:
		old = t.c.Bound.TextureUnit.Binding2d.GetID()
	case GLenum_GL_TEXTURE_3D:
		old = t.c.Bound.TextureUnit.Binding3d.GetID()
	case GLenum_GL_TEXTURE_2D_ARRAY:
		old = t.c.Bound.TextureUnit.Binding2dArray.GetID()
	case GLenum_GL_TEXTURE_CUBE_MAP:
		old = t.c.Bound.TextureUnit.BindingCubeMap.GetID()
	case GLenum_GL_TEXTURE_CUBE_MAP_ARRAY:
		old = t.c.Bound.TextureUnit.BindingCubeMapArray.GetID()
	case GLenum_GL_TEXTURE_2D_MULTISAMPLE:
		old = t.c.Bound.TextureUnit.Binding2dMultisample.GetID()
	case GLenum_GL_TEXTURE_2D_MULTISAMPLE_ARRAY:
		old = t.c.Bound.TextureUnit.Binding2dMultisampleArray.GetID()
	case GLenum_GL_TEXTURE_EXTERNAL_OES:
		old = t.c.Bound.TextureUnit.BindingExternalOes.GetID()
	default:
		panic(fmt.Errorf("%v is not a texture kind", tex.Kind))
	}

	if old != tex.ID {
		t.doAndUndo(ctx,
			t.cb.GlBindTexture(tex.Kind, tex.ID),
			t.cb.GlBindTexture(tex.Kind, old))
	}
}

func (t *tweaker) glBindTexture_2D(ctx context.Context, id TextureId) {
	if o := t.c.Bound.TextureUnit.Binding2d.GetID(); o != id {
		t.doAndUndo(ctx,
			t.cb.GlBindTexture(GLenum_GL_TEXTURE_2D, id),
			t.cb.GlBindTexture(GLenum_GL_TEXTURE_2D, o))
	}
}

func (t *tweaker) glBindVertexArray(ctx context.Context, id VertexArrayId) {
	if o := t.c.Bound.VertexArray.GetID(); o != id {
		t.doAndUndo(ctx,
			t.cb.GlBindVertexArray(id),
			t.cb.GlBindVertexArray(o))
	}
}

func (t *tweaker) glUseProgram(ctx context.Context, id ProgramId) {
	if o := t.c.Bound.Program.GetID(); o != id {
		t.doAndUndo(ctx,
			t.cb.GlUseProgram(id),
			t.cb.GlUseProgram(o))
	}
}

func (t *tweaker) glActiveTexture(ctx context.Context, unit GLenum) {
	if o := GLenum(t.c.Bound.TextureUnit.ID) + GLenum_GL_TEXTURE0; o != unit {
		t.doAndUndo(ctx,
			t.cb.GlActiveTexture(unit),
			t.cb.GlActiveTexture(o))
	}
}

func (t *tweaker) setPackStorage(ctx context.Context, state PixelStorageState, bufferId BufferId) {
	origState := map[GLenum]GLint{}
	forEachPackStorageState(t.c.Other.Pack, func(n GLenum, v GLint) { origState[n] = v })
	forEachPackStorageState(state, func(name GLenum, value GLint) {
		if o := origState[name]; o != value {
			t.doAndUndo(ctx,
				t.cb.GlPixelStorei(name, value),
				t.cb.GlPixelStorei(name, o))
		}
	})
	if o := t.c.Bound.PixelPackBuffer.GetID(); o != bufferId {
		t.doAndUndo(ctx,
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

func (t *tweaker) setUnpackStorage(ctx context.Context, state PixelStorageState, bufferId BufferId) {
	origState := map[GLenum]GLint{}
	forEachUnpackStorageState(t.c.Other.Unpack, func(n GLenum, v GLint) { origState[n] = v })
	forEachUnpackStorageState(state, func(name GLenum, value GLint) {
		if o := origState[name]; o != value {
			t.doAndUndo(ctx,
				t.cb.GlPixelStorei(name, value),
				t.cb.GlPixelStorei(name, o))
		}
	})
	if o := t.c.Bound.PixelUnpackBuffer.GetID(); o != bufferId {
		t.doAndUndo(ctx,
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
