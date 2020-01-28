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
	"bytes"
	"context"
	"fmt"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/database"
)

// wireframe returns a command transform that replaces all draw calls of
// triangle primitives with draw calls of a wireframe equivalent.
func wireframe(ctx context.Context, framebuffer FramebufferId) transform.Transformer {
	ctx = log.Enter(ctx, "Wireframe")
	return transform.Transform("Wireframe", func(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
		if dc, ok := cmd.(drawCall); ok {
			s := out.State()
			c := GetContext(s, cmd.Thread())

			fb := c.Bound().DrawFramebuffer()
			if fb.IsNil() {
				return out.MutateAndWrite(ctx, id, cmd)
			}

			if fb.ID() != framebuffer {
				return out.MutateAndWrite(ctx, id, cmd)
			}

			dID := id.Derived()
			cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}

			t := newTweaker(out, dID, cb)
			defer t.revert(ctx)

			t.glEnable(ctx, GLenum_GL_LINE_SMOOTH)
			t.glEnable(ctx, GLenum_GL_BLEND)
			t.glBlendFunc(ctx, GLenum_GL_SRC_ALPHA, GLenum_GL_ONE_MINUS_SRC_ALPHA)
			if err := drawWireframe(ctx, id, dc, s, out); err != nil {
				log.E(ctx, "%v", err)
			}

			t.revert(ctx)
		} else {
			return out.MutateAndWrite(ctx, id, cmd)
		}
		return nil
	})
}

// wireframeOverlay returns a command transform that renders the wireframe of
// the mesh over of the specified draw call.
func wireframeOverlay(ctx context.Context, i api.CmdID) transform.Transformer {
	ctx = log.Enter(ctx, "DrawMode_WIREFRAME_OVERLAY")
	return transform.Transform("DrawMode_WIREFRAME_OVERLAY", func(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
		if i == id {
			if dc, ok := cmd.(drawCall); ok {
				s := out.State()
				out.MutateAndWrite(ctx, id, dc)

				dID := id.Derived()
				cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}
				t := newTweaker(out, dID, cb)
				t.glEnable(ctx, GLenum_GL_POLYGON_OFFSET_LINE)
				t.glPolygonOffset(ctx, -1, -1)
				t.glEnable(ctx, GLenum_GL_BLEND)
				t.glBlendColor(ctx, 1.0, 0.5, 1.0, 1.0)
				t.glBlendFunc(ctx, GLenum_GL_CONSTANT_COLOR, GLenum_GL_ZERO)
				t.glEnable(ctx, GLenum_GL_LINE_SMOOTH)
				t.glLineWidth(ctx, 1.5)

				if err := drawWireframe(ctx, i, dc, s, out); err != nil {
					log.E(ctx, "%v", err)
				}

				t.revert(ctx)
				return nil
			}
		}

		return out.MutateAndWrite(ctx, id, cmd)
	})
}

func drawWireframe(ctx context.Context, i api.CmdID, dc drawCall, s *api.GlobalState, out transform.Writer) error {
	c := GetContext(s, dc.Thread())
	cb := CommandBuilder{Thread: dc.Thread(), Arena: s.Arena}
	dID := i.Derived()

	dci, err := dc.getIndices(ctx, c, s)
	if err != nil {
		return err
	}
	indices, drawMode, err := makeWireframe(dci.indices, dci.drawMode)
	if err != nil {
		return err
	}

	// Store the wire-frame data to a temporary address.
	wireframeData, wireframeDataType := encodeIndices(indices)
	resID, err := database.Store(ctx, wireframeData)
	if err != nil {
		return err
	}

	// Unbind the index buffer
	tmp := s.AllocOrPanic(ctx, uint64(len(wireframeData)))
	oldIndexBuffer := c.Bound().VertexArray().ElementArrayBuffer()
	out.MutateAndWrite(ctx, dID,
		cb.GlBindBuffer(GLenum_GL_ELEMENT_ARRAY_BUFFER, 0).
			AddRead(tmp.Range(), resID))

	// Draw the wire-frame
	out.MutateAndWrite(ctx, i, cb.GlDrawElements(
		drawMode, GLsizei(len(indices)), wireframeDataType, tmp.Ptr()))

	// Rebind the old index buffer
	out.MutateAndWrite(ctx, dID, cb.GlBindBuffer(
		GLenum_GL_ELEMENT_ARRAY_BUFFER, oldIndexBuffer.GetID()))

	return nil
}

// encodeIndices assumes little endian encoding
func encodeIndices(indices []uint32) ([]byte, GLenum) {
	maxIndex := uint32(0)
	for _, v := range indices {
		if v > maxIndex {
			maxIndex = v
		}
	}
	buf := &bytes.Buffer{}
	w := endian.Writer(buf, device.LittleEndian)
	switch {
	case maxIndex > 0xFFFF:
		// TODO: GL_UNSIGNED_INT in glDrawElements is supported only since GLES 3.0
		for _, v := range indices {
			w.Uint32(v)
		}
		return buf.Bytes(), GLenum_GL_UNSIGNED_INT

	case maxIndex > 0xFF:
		for _, v := range indices {
			w.Uint16(uint16(v))
		}
		return buf.Bytes(), GLenum_GL_UNSIGNED_SHORT

	default:
		for _, v := range indices {
			w.Uint8(uint8(v))
		}
		return buf.Bytes(), GLenum_GL_UNSIGNED_BYTE
	}
}

func appendWireframeOfTriangle(lines []uint32, v0, v1, v2 uint32) []uint32 {
	if v0 == v1 || v1 == v2 || v2 == v0 {
		return lines // Ignore degenerate triangle
	}
	return append(lines, v0, v1, v1, v2, v2, v0)
}

func makeWireframe(indices []uint32, drawMode GLenum) ([]uint32, GLenum, error) {
	switch drawMode {
	case GLenum_GL_POINTS, GLenum_GL_LINES, GLenum_GL_LINE_STRIP, GLenum_GL_LINE_LOOP:
		return indices, drawMode, nil

	case GLenum_GL_TRIANGLES:
		numTriangles := len(indices) / 3
		lines := make([]uint32, 0, numTriangles*6)
		for i := 0; i < numTriangles; i++ {
			lines = appendWireframeOfTriangle(lines, indices[i*3], indices[i*3+1], indices[i*3+2])
		}
		return lines, GLenum_GL_LINES, nil

	case GLenum_GL_TRIANGLE_STRIP:
		numTriangles := len(indices) - 2
		if numTriangles > 0 {
			lines := make([]uint32, 0, numTriangles*6)
			for i := 0; i < numTriangles; i++ {
				lines = appendWireframeOfTriangle(lines, indices[i], indices[i+1], indices[i+2])
			}
			return lines, GLenum_GL_LINES, nil
		}
		return []uint32{}, GLenum_GL_LINES, nil

	case GLenum_GL_TRIANGLE_FAN:
		numTriangles := len(indices) - 2
		if numTriangles > 0 {
			lines := make([]uint32, 0, numTriangles*6)
			for i := 0; i < numTriangles; i++ {
				lines = appendWireframeOfTriangle(lines, indices[0], indices[i+1], indices[i+2])
			}
			return lines, GLenum_GL_LINES, nil
		}
		return []uint32{}, GLenum_GL_LINES, nil

	default:
		return nil, 0, fmt.Errorf("Unknown mode: %v", drawMode)
	}
}
