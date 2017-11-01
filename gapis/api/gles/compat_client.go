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

	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/resolve"
)

var _ = []drawElements{
	&GlDrawElements{},
	&GlDrawRangeElements{},
}

type drawElements interface {
	api.Cmd
	withIndicesNull() drawElements
	indexLimits() (resolve.IndexRange, bool)
	drawMode() GLenum
	indicesCount() GLsizei
	indicesType() GLenum
	indices() IndicesPointer
}

func (d *GlDrawElements) withIndicesNull() drawElements {
	out := *d
	out.Indices.addr = 0
	return &out
}
func (d *GlDrawElements) indexLimits() (resolve.IndexRange, bool) { return resolve.IndexRange{}, false }
func (d *GlDrawElements) drawMode() GLenum                        { return d.DrawMode }
func (d *GlDrawElements) indicesCount() GLsizei                   { return d.IndicesCount }
func (d *GlDrawElements) indicesType() GLenum                     { return d.IndicesType }
func (d *GlDrawElements) indices() IndicesPointer                 { return d.Indices }

func (d *GlDrawRangeElements) withIndicesNull() drawElements {
	out := *d
	out.Indices.addr = 0
	return &out
}
func (d *GlDrawRangeElements) indexLimits() (resolve.IndexRange, bool) {
	return resolve.IndexRange{
		First: uint32(d.Start),
		Count: uint32(d.End-d.Start) + 1,
	}, true
}
func (d *GlDrawRangeElements) drawMode() GLenum        { return d.DrawMode }
func (d *GlDrawRangeElements) indicesCount() GLsizei   { return d.IndicesCount }
func (d *GlDrawRangeElements) indicesType() GLenum     { return d.IndicesType }
func (d *GlDrawRangeElements) indices() IndicesPointer { return d.Indices }

// compatDrawElements performs compatibility logic to translate a draw elements
// call, moving all client-side pointers to buffers.
func compatDrawElements(
	ctx context.Context,
	t *tweaker,
	clientVAs map[*VertexAttributeArray]*GlVertexAttribPointer,
	id api.CmdID,
	cmd drawElements,
	s *api.GlobalState,
	out transform.Writer) {

	c := GetContext(s, cmd.Thread())
	e := externs{ctx: ctx, cmd: cmd, s: s}

	ib := c.Bound.VertexArray.ElementArrayBuffer
	clientIB := ib == nil
	clientVB := clientVAsBound(c, clientVAs)

	if clientIB {
		// The indices for the glDrawElements call is in client memory.
		// We need to move this into a temporary buffer.

		// Generate a new element array buffer and bind it.
		bufID := t.glGenBuffer(ctx)
		t.GlBindBuffer_ElementArrayBuffer(ctx, bufID)

		// By moving the draw call's observations earlier, populate the element array buffer.
		size, base := DataTypeSize(cmd.indicesType())*int(cmd.indicesCount()), memory.Pointer(cmd.indices())
		glBufferData := t.cb.GlBufferData(GLenum_GL_ELEMENT_ARRAY_BUFFER, GLsizeiptr(size), memory.Pointer(base), GLenum_GL_STATIC_DRAW)
		glBufferData.extras = *cmd.Extras()
		out.MutateAndWrite(ctx, t.dID, glBufferData)

		if clientVB {
			// Some of the vertex arrays for the glDrawElements call is in
			// client memory and we need to move this into temporary buffer(s).
			// The indices are also in client memory, so we need to apply the
			// command's reads now so that the indices can be read from the
			// application pool.
			cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
			indexSize := DataTypeSize(cmd.indicesType())
			data := U8ᵖ(cmd.indices()).Slice(0, uint64(indexSize*int(cmd.indicesCount())), s.MemoryLayout)
			limits, ok := cmd.indexLimits()
			if !ok {
				limits = e.calcIndexLimits(data, indexSize)
			}
			moveClientVBsToVAs(ctx, t, clientVAs, limits.First, limits.Count, id, cmd, s, c, out)
		}

		cmd := cmd.withIndicesNull()
		compatMultiviewDraw(ctx, id, cmd, out)
		return

	} else if clientVB { // GL_ELEMENT_ARRAY_BUFFER is bound
		// Some of the vertex arrays for the glDrawElements call is in
		// client memory and we need to move this into temporary buffer(s).
		// The indices are server-side, so can just be read from the internal
		// pooled buffer.
		data := ib.Data
		indexSize := DataTypeSize(cmd.indicesType())
		start := u64.Min(cmd.indices().addr, data.count)                               // Clamp
		end := u64.Min(start+uint64(indexSize)*uint64(cmd.indicesCount()), data.count) // Clamp
		limits, ok := cmd.indexLimits()
		if !ok {
			limits = e.calcIndexLimits(data.Slice(start, end, s.MemoryLayout), indexSize)
		}
		moveClientVBsToVAs(ctx, t, clientVAs, limits.First, limits.Count, id, cmd, s, c, out)
	}
	compatMultiviewDraw(ctx, id, cmd, out)
}

// clientVAsBound returns true if there are any vertex attribute arrays enabled
// with pointers to client-side memory.
func clientVAsBound(c *Context, clientVAs map[*VertexAttributeArray]*GlVertexAttribPointer) bool {
	for _, arr := range c.Bound.VertexArray.VertexAttributeArrays.Range() {
		if arr.Enabled == GLboolean_GL_TRUE {
			if _, ok := clientVAs[arr]; ok {
				return true
			}
		}
	}
	return false
}

// moveClientVBsToVAs is a compatability helper for transforming client-side
// vertex array data (which is not supported by glVertexAttribPointer in later
// versions of GL), into array-buffers.
func moveClientVBsToVAs(
	ctx context.Context,
	t *tweaker,
	clientVAs map[*VertexAttributeArray]*GlVertexAttribPointer,
	first, count uint32, // vertex indices
	id api.CmdID,
	cmd api.Cmd,
	s *api.GlobalState,
	c *Context,
	out transform.Writer) {

	if count == 0 {
		return
	}

	cb := CommandBuilder{Thread: cmd.Thread()}
	rngs := interval.U64RangeList{}
	// Gather together all the client-buffers in use by the vertex-attribs.
	// Merge together all the memory intervals that these use.
	va := c.Bound.VertexArray
	for _, arr := range va.VertexAttributeArrays.Range() {
		if arr.Enabled == GLboolean_GL_TRUE {
			vb := va.VertexBufferBindings.Get(arr.Binding)
			if cmd, ok := clientVAs[arr]; ok {
				// TODO: We're currently ignoring the Offset and Stride fields of the VBB.
				// TODO: We're currently ignoring the RelativeOffset field of the VA.
				// TODO: Merge logic with ReadVertexArrays macro in vertex_arrays.api.
				if vb.Divisor != 0 {
					panic("Instanced draw calls not currently supported by the compatibility layer")
				}
				stride, size := int(cmd.Stride), DataTypeSize(cmd.Type)*int(cmd.Size)
				if stride == 0 {
					stride = size
				}
				rng := memory.Range{
					Base: cmd.Data.addr, // Always start from the 0'th vertex to simplify logic.
					Size: uint64(int(first+count-1)*stride + size),
				}
				interval.Merge(&rngs, rng.Span(), true)
			}
		}
	}

	// Create an array-buffer for each chunk of overlapping client-side buffers in
	// use. These are populated with data below.
	ids := make([]BufferId, len(rngs))
	for i := range rngs {
		ids[i] = t.glGenBuffer(ctx)
	}

	// Apply the memory observations that were made by the draw call now.
	// We need to do this as the glBufferData calls below will require the data.
	dID := id.Derived()
	out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
		return nil
	}))

	// Note: be careful of overwriting the observations made above, before the
	// calls to glBufferData below.

	// Fill the array-buffers with the observed memory data.
	for i, rng := range rngs {
		base := memory.BytePtr(rng.First, memory.ApplicationPool)
		size := GLsizeiptr(rng.Count)
		t.GlBindBuffer_ArrayBuffer(ctx, ids[i])
		out.MutateAndWrite(ctx, dID, cb.GlBufferData(GLenum_GL_ARRAY_BUFFER, size, base, GLenum_GL_STATIC_DRAW))
	}

	// Redirect all the vertex attrib arrays to point to the array-buffer data.
	for _, l := range va.VertexAttributeArrays.KeysSorted() {
		arr := va.VertexAttributeArrays.Get(l)
		if arr.Enabled == GLboolean_GL_TRUE {
			if glVAP, ok := clientVAs[arr]; ok {
				glVAP := *glVAP // Copy
				i := interval.IndexOf(&rngs, glVAP.Data.addr)
				t.GlBindBuffer_ArrayBuffer(ctx, ids[i])
				// The glVertexAttribPointer call may have come from a different thread
				// and there's no guarantees that the thread still has the context bound.
				// Use the draw call's thread instead.
				glVAP.SetThread(cmd.Thread())
				glVAP.Data = VertexPointer{glVAP.Data.addr - rngs[i].First, memory.ApplicationPool} // Offset
				out.MutateAndWrite(ctx, dID, &glVAP)
			}
		}
	}
}
