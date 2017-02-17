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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/memory"
)

// tweaker provides a set of methods for temporarily changing the GLES state.
type tweaker struct {
	ctx  log.Context // Needed so functions match gl signature
	out  transform.Writer
	c    *Context
	undo []func()
}

// revert undoes all the changes made by the tweaker.
func (t *tweaker) revert() {
	for i := len(t.undo) - 1; i >= 0; i-- {
		t.undo[i]()
	}
	t.undo = nil
}

func (t *tweaker) getCapability(v GLenum) bool {
	a := NewGlIsEnabled(v, 0)
	o := a.Extras().Observations()
	s := t.out.State()
	i := GLuint(0) // capability index.
	res, err := subGetCapability(t.ctx, a, o, s, GetState(s), nil, v, i)
	if err != nil {
		panic(err)
	}
	return res != 0
}

func (t *tweaker) setCapability(v GLenum, e GLboolean) {
	a := NewGlEnable(v)
	o := a.Extras().Observations()
	s := t.out.State()
	i := GLuint(0) // capability index.
	subSetCapability(t.ctx, a, o, s, GetState(s), nil, v, false, i, e)
}

func (t *tweaker) glEnable(v GLenum) {
	ctx := t.ctx
	if !t.getCapability(v) {
		t.undo = append(t.undo, func() {
			// TODO: This does not correctly restore indexed capabilities.
			t.out.MutateAndWrite(ctx, atom.NoID, NewGlDisable(v))
			t.setCapability(v, 0)
		})
		t.out.MutateAndWrite(ctx, atom.NoID, NewGlEnable(v))
		t.setCapability(v, 1)
	}
}

func (t *tweaker) glDisable(v GLenum) {
	ctx := t.ctx
	if t.getCapability(v) {
		t.undo = append(t.undo, func() {
			// TODO: This does not correctly restore indexed capabilities.
			t.out.MutateAndWrite(ctx, atom.NoID, NewGlEnable(v))
			t.setCapability(v, 1)
		})
		t.out.MutateAndWrite(ctx, atom.NoID, NewGlDisable(v))
		t.setCapability(v, 0)
	}
}

func (t *tweaker) glDepthMask(v GLboolean) {
	ctx := t.ctx
	if o := t.c.Framebuffer.DepthWritemask; o != v {
		t.undo = append(t.undo, func() {
			t.out.MutateAndWrite(ctx, atom.NoID, NewGlDepthMask(o))
			t.c.Framebuffer.DepthWritemask = o
		})
		t.out.MutateAndWrite(ctx, atom.NoID, NewGlDepthMask(v))
		t.c.Framebuffer.DepthWritemask = v
	}
}

func (t *tweaker) glDepthFunc(v GLenum) {
	ctx := t.ctx
	if o := t.c.FragmentOperations.Depth.Func; o != v {
		t.undo = append(t.undo, func() {
			t.out.MutateAndWrite(ctx, atom.NoID, NewGlDepthFunc(o))
			t.c.FragmentOperations.Depth.Func = o
		})
		t.out.MutateAndWrite(ctx, atom.NoID, NewGlDepthFunc(v))
		t.c.FragmentOperations.Depth.Func = v
	}
}

func (t *tweaker) glBlendColor(r, g, b, a GLfloat) {
	ctx := t.ctx
	n := Color{Red: r, Green: g, Blue: b, Alpha: a}
	if o := t.c.FragmentOperations.BlendColor; o != n {
		t.undo = append(t.undo, func() {
			t.out.MutateAndWrite(ctx, atom.NoID, NewGlBlendColor(o.Red, o.Green, o.Blue, o.Alpha))
			t.c.FragmentOperations.BlendColor = o
		})
		t.out.MutateAndWrite(ctx, atom.NoID, NewGlBlendColor(r, g, b, a))
		t.c.FragmentOperations.BlendColor = n
	}
}

func (t *tweaker) glBlendFunc(src, dst GLenum) {
	t.glBlendFuncSeparate(src, dst, src, dst)
}

func (t *tweaker) glBlendFuncSeparate(srcRGB, dstRGB, srcA, dstA GLenum) {
	ctx := t.ctx
	for i := range t.c.FragmentOperations.Blend {
		idx := DrawBufferIndex(i)
		orig := t.c.FragmentOperations.Blend[idx]
		tweaked := orig
		tweaked.SrcRgb, tweaked.DstRgb, tweaked.SrcAlpha, tweaked.DstAlpha = srcRGB, dstRGB, srcA, dstA
		if orig != tweaked {
			t.undo = append(t.undo, func() {
				t.out.MutateAndWrite(ctx, atom.NoID, NewGlBlendFuncSeparate(orig.SrcRgb, orig.DstRgb, orig.SrcAlpha, orig.DstAlpha))
				t.c.FragmentOperations.Blend[idx] = orig
			})
			t.out.MutateAndWrite(ctx, atom.NoID, NewGlBlendFuncSeparate(srcRGB, dstRGB, srcA, dstA))
			t.c.FragmentOperations.Blend[idx] = tweaked
		}
	}
}

// glPolygonOffset adjusts the offset depth factor and units. Unlike the original glPolygonOffset,
// this function adds the given values to the current values rather than setting them.
func (t *tweaker) glPolygonOffset(factor, units GLfloat) {
	ctx := t.ctx
	origFactor, origUnits := t.c.Rasterization.PolygonOffsetFactor, t.c.Rasterization.PolygonOffsetUnits
	t.undo = append(t.undo, func() {
		t.out.MutateAndWrite(ctx, atom.NoID, NewGlPolygonOffset(origFactor, origUnits))
		t.c.Rasterization.PolygonOffsetFactor = origFactor
		t.c.Rasterization.PolygonOffsetUnits = origUnits
	})
	t.out.MutateAndWrite(ctx, atom.NoID, NewGlPolygonOffset(origFactor+factor, origUnits+units))
	t.c.Rasterization.PolygonOffsetFactor = origFactor + factor
	t.c.Rasterization.PolygonOffsetUnits = origUnits + units
}

func (t *tweaker) glLineWidth(width GLfloat) {
	ctx := t.ctx
	orig := t.c.Rasterization.LineWidth
	if orig != width {
		t.undo = append(t.undo, func() {
			t.out.MutateAndWrite(ctx, atom.NoID, NewGlLineWidth(orig))
			t.c.Rasterization.LineWidth = orig
		})
		t.out.MutateAndWrite(ctx, atom.NoID, NewGlLineWidth(width))
		t.c.Rasterization.LineWidth = width
	}
}

// This will either bind new VAO (GLES 3.x) or save state of the default one (GLES 2.0).
func (t *tweaker) bindOrSaveVertexArray(version *Version, newArray VertexArrayId, locations ...AttributeLocation) {
	ctx := t.ctx
	s := t.out.State()
	if version.Major >= 3 {
		// GLES 3.0 and 3.1 introduce a lot of new state which would be hard to restore.
		// It is much easier to just create a fresh Vertex Array Object to work with.
		origArray := t.c.BoundVertexArray

		tmp := atom.Must(atom.AllocData(t.ctx, s, newArray))
		t.out.MutateAndWrite(ctx, atom.NoID, NewGlGenVertexArrays(1, tmp.Ptr()).AddWrite(tmp.Data()))
		t.out.MutateAndWrite(ctx, atom.NoID, NewGlBindVertexArray(newArray))
		t.undo = append(t.undo, func() {
			t.out.MutateAndWrite(ctx, atom.NoID, NewGlBindVertexArray(origArray))
			tmp := atom.Must(atom.AllocData(t.ctx, s, newArray))
			t.out.MutateAndWrite(ctx, atom.NoID, NewGlDeleteVertexArrays(1, tmp.Ptr()).AddRead(tmp.Data()))
		})
	} else {
		// GLES 2.0 does not have Vertex Array Objects, but the state is fairly simple.
		origArrayBufferID := t.c.BoundBuffers.ArrayBuffer
		for _, location := range locations {
			location := location
			vao := t.c.Instances.VertexArrays[t.c.BoundVertexArray]
			origVertexAttrib := *(vao.VertexAttributeArrays[location])
			origVertexBinding := *(vao.VertexBufferBindings[VertexBufferBindingIndex(location)])
			t.undo = append(t.undo, func() {
				t.out.MutateAndWrite(ctx, atom.NoID, NewGlBindBuffer(GLenum_GL_ARRAY_BUFFER, origVertexBinding.Buffer))
				if origVertexAttrib.Enabled == GLboolean_GL_TRUE {
					t.out.MutateAndWrite(ctx, atom.NoID, NewGlEnableVertexAttribArray(location))
				} else {
					t.out.MutateAndWrite(ctx, atom.NoID, NewGlDisableVertexAttribArray(location))
				}
				t.out.MutateAndWrite(ctx, atom.NoID, NewGlVertexAttribPointer(location, origVertexAttrib.Size, origVertexAttrib.Type, origVertexAttrib.Normalized, origVertexAttrib.Stride, memory.Pointer(origVertexAttrib.Pointer)))
				t.out.MutateAndWrite(ctx, atom.NoID, NewGlBindBuffer(GLenum_GL_ARRAY_BUFFER, origArrayBufferID))
			})
		}
	}
}
