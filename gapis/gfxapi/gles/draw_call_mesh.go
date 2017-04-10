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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/core/stream"
	"github.com/google/gapid/core/stream/fmts"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/vertex"
)

// drawCallMesh builds a mesh for dc at p.
func drawCallMesh(ctx context.Context, dc drawCall, p *path.Mesh) (*gfxapi.Mesh, error) {
	cmdPath := path.FindCommand(p)
	if cmdPath == nil {
		log.W(ctx, "Couldn't find command at path '%v'", p)
		return nil, nil
	}

	s, err := resolve.GlobalState(ctx, cmdPath.StateAfter())
	if err != nil {
		return nil, err
	}

	c := GetContext(s)

	indices, glPrimitive, err := dc.getIndices(ctx, c, s)
	if err != nil {
		return nil, err
	}

	drawPrimitive, err := translateDrawPrimitive(glPrimitive)
	if err != nil {
		// There are extensions like GL_QUADS_OES that do not translate directly
		// to a gfxapi.DrawPrimitive. For now, log the error, and return
		// (nil, nil) (no mesh available). See b/29416809.
		log.E(ctx, "Couldn't translate GL primitive to gfxapi.DrawPrimitive: %v", err)
		return nil, nil
	}

	// Look at the indices to find the number of vertices we're dealing with.
	count := 0
	for _, i := range indices {
		if count <= int(i) {
			count = int(i) + 1
		}
	}

	if count == 0 {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrMeshHasNoVertices()}
	}

	program, found := c.SharedObjects.Programs[c.BoundProgram]
	if !found {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoProgramBound()}
	}

	vb := &vertex.Buffer{}
	va := c.Objects.VertexArrays[c.BoundVertexArray]
	for _, attr := range program.ActiveAttributes {
		vaa := va.VertexAttributeArrays[attr.Location]
		if vaa.Enabled == GLboolean_GL_FALSE {
			continue
		}
		vbb := va.VertexBufferBindings[vaa.Binding]

		format, err := translateVertexFormat(vaa)
		if err != nil {
			return nil, err
		}

		var slice U8ˢ
		if vbb.Buffer == 0 {
			// upper bound doesn't really matter here, so long as it's big.
			slice = U8ˢ(vaa.Pointer.Slice(0, 1<<30, s))
		} else {
			slice = c.SharedObjects.Buffers[vbb.Buffer].Data
		}
		data, err := vertexStreamData(ctx, vaa, vbb, count, slice, s)
		if err != nil {
			return nil, err
		}

		vb.Streams = append(vb.Streams,
			&vertex.Stream{
				Name:     attr.Name,
				Data:     data,
				Format:   format,
				Semantic: &vertex.Semantic{},
			},
		)
	}

	guessSemantics(vb)

	ib := &gfxapi.IndexBuffer{
		Indices: []uint32(indices),
	}

	mesh := &gfxapi.Mesh{
		DrawPrimitive: drawPrimitive,
		VertexBuffer:  vb,
		IndexBuffer:   ib,
	}

	if p.Options != nil && p.Options.Faceted {
		return mesh.Faceted(ctx)
	}

	return mesh, nil
}

func vertexStreamData(
	ctx context.Context,
	vaa *VertexAttributeArray,
	vbb *VertexBufferBinding,
	vectorCount int,
	slice U8ˢ,
	s *gfxapi.State) ([]byte, error) {

	if vbb.Divisor != 0 {
		return nil, fmt.Errorf("Instanced draw calls not currently supported")
	}

	elementsPerVector := int(vaa.Size)
	vectorSize := elementsPerVector * DataTypeSize(vaa.Type)
	vectorStride := int(vaa.Stride)
	if vectorStride == 0 {
		vectorStride = vectorSize
	}
	gap := vectorStride - vectorSize // number of bytes between each vector

	compactSize := vectorSize * vectorCount
	out := make([]byte, compactSize)

	base := uint64(vaa.RelativeOffset) + uint64(vbb.Offset)
	if base >= slice.Count {
		// First vertex sits beyond the end of the buffer.
		// Instead of erroring just return a 0-initialized buffer so other
		// streams can be visualized. The report should display an error to
		// alert the user to the bad data.
		// TODO: Actually add this as a report error.
		return out, nil
	}

	// Only read as much data as we actually have.
	size := u64.Min(uint64(compactSize+ /* total size of gaps */ gap*(vectorCount-1)), slice.Count)
	data := slice.Slice(base, base+size, s).Read(ctx, nil, s, nil)
	if gap > 0 {
		// Adjust vectorCount to the number of complete vectors found in data.
		vectorCount := sint.Min((gap+len(data))/vectorStride, vectorCount)
		// Copy the vector elements to out removing any gaps.
		for i := 0; i < vectorCount; i++ {
			copy(out[i*vectorSize:(i+1)*vectorSize], data[i*vectorStride:])
		}
	} else {
		// No gaps between vector elements. Simple copy.
		copy(out, data)
	}

	return out, nil
}

func translateDrawPrimitive(e GLenum) (gfxapi.DrawPrimitive, error) {
	switch e {
	case GLenum_GL_POINTS:
		return gfxapi.DrawPrimitive_Points, nil
	case GLenum_GL_LINES:
		return gfxapi.DrawPrimitive_Lines, nil
	case GLenum_GL_LINE_STRIP:
		return gfxapi.DrawPrimitive_LineStrip, nil
	case GLenum_GL_LINE_LOOP:
		return gfxapi.DrawPrimitive_LineLoop, nil
	case GLenum_GL_TRIANGLES:
		return gfxapi.DrawPrimitive_Triangles, nil
	case GLenum_GL_TRIANGLE_STRIP:
		return gfxapi.DrawPrimitive_TriangleStrip, nil
	case GLenum_GL_TRIANGLE_FAN:
		return gfxapi.DrawPrimitive_TriangleFan, nil
	default:
		return 0, fmt.Errorf("Invalid draw mode %v", e)
	}
}

func translateVertexFormat(vaa *VertexAttributeArray) (*stream.Format, error) {
	switch vaa.Type {
	case GLenum_GL_INT_2_10_10_10_REV:
		return fmts.XYZW_S10S10S10S2, nil
	case GLenum_GL_UNSIGNED_INT_2_10_10_10_REV:
		return fmts.XYZW_U10U10U10U2, nil
	}

	var dt stream.DataType
	switch vaa.Type {
	case GLenum_GL_BYTE:
		dt = stream.S8
	case GLenum_GL_UNSIGNED_BYTE:
		dt = stream.U8
	case GLenum_GL_SHORT:
		dt = stream.S16
	case GLenum_GL_UNSIGNED_SHORT:
		dt = stream.U16
	case GLenum_GL_INT:
		dt = stream.S32
	case GLenum_GL_UNSIGNED_INT:
		dt = stream.U32
	case GLenum_GL_HALF_FLOAT, GLenum_GL_HALF_FLOAT_OES:
		dt = stream.F16
	case GLenum_GL_FLOAT:
		dt = stream.F32
	case GLenum_GL_FIXED:
		dt = stream.S16_16
	default:
		return nil, fmt.Errorf("Unsupported vertex type: %v", vaa.Type)
	}

	sampling := stream.Linear
	if vaa.Normalized != 0 {
		sampling = stream.LinearNormalized
	}

	fmt := &stream.Format{
		Components: make([]*stream.Component, vaa.Size),
	}

	xyzw := []stream.Channel{
		stream.Channel_X,
		stream.Channel_Y,
		stream.Channel_Z,
		stream.Channel_W,
	}
	for i := range fmt.Components {
		fmt.Components[i] = &stream.Component{
			DataType: &dt,
			Sampling: sampling,
			Channel:  xyzw[i],
		}
	}
	return fmt, nil
}
