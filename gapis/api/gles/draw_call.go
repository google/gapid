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

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/gapis/api"
)

type drawCallIndices struct {
	indices  []uint32
	drawMode GLenum
	indexed  bool
}

// drawCall is the interface implemented by all GLES draw call atoms.
type drawCall interface {
	api.Cmd
	getIndices(ctx context.Context, c *Context, s *api.GlobalState) (drawCallIndices, error)
}

func (a *GlDrawArrays) getIndices(ctx context.Context, c *Context, s *api.GlobalState) (drawCallIndices, error) {
	indices := make([]uint32, a.IndicesCount)
	for i := range indices {
		indices[i] = uint32(a.FirstIndex) + uint32(i)
	}
	return drawCallIndices{indices, a.DrawMode, false}, nil
}

func (a *GlDrawElements) getIndices(ctx context.Context, c *Context, s *api.GlobalState) (drawCallIndices, error) {
	return getIndices(ctx, c, s, a.IndicesType, a.DrawMode, 0, a.IndicesCount, a.Indices)
}

func (a *GlDrawRangeElements) getIndices(ctx context.Context, c *Context, s *api.GlobalState) (drawCallIndices, error) {
	return getIndices(ctx, c, s, a.IndicesType, a.DrawMode, 0, a.IndicesCount, a.Indices)
}

func getIndices(
	ctx context.Context,
	c *Context,
	s *api.GlobalState,
	ty, drawMode GLenum,
	first, count GLsizei,
	ptr IndicesPointer) (drawCallIndices, error) {

	indexSize := map[GLenum]uint64{
		GLenum_GL_UNSIGNED_BYTE:  1,
		GLenum_GL_UNSIGNED_SHORT: 2,
		GLenum_GL_UNSIGNED_INT:   4,
	}[ty]
	indexBuffer := c.Bound.VertexArray.ElementArrayBuffer
	size := uint64(count) * indexSize
	offset := uint64(first) * indexSize

	var reader binary.Reader
	if indexBuffer == nil {
		// Get the index buffer data from pointer
		reader = ptr.Slice(offset, size, s.MemoryLayout).Reader(ctx, s)
	} else {
		// Get the index buffer data from buffer, offset by the 'indices' pointer.
		offset += ptr.addr
		reader = indexBuffer.Data.Slice(offset, offset+size, s.MemoryLayout).Reader(ctx, s)
	}

	indices, err := decodeIndices(reader, ty)
	if err != nil {
		return drawCallIndices{}, err
	}
	return drawCallIndices{indices, drawMode, true}, err
}

// decodeIndices assumes little endian encoding
func decodeIndices(r binary.Reader, indicesType GLenum) ([]uint32, error) {
	var indices []uint32
	switch indicesType {
	case GLenum_GL_UNSIGNED_BYTE:
		for {
			if val := r.Uint8(); r.Error() == nil {
				indices = append(indices, uint32(val))
			} else {
				return indices, nil
			}
		}

	case GLenum_GL_UNSIGNED_SHORT:
		for {
			if val := r.Uint16(); r.Error() == nil {
				indices = append(indices, uint32(val))
			} else {
				return indices, nil
			}
		}

	case GLenum_GL_UNSIGNED_INT:
		for {
			if val := r.Uint32(); r.Error() == nil {
				indices = append(indices, val)
			} else {
				return indices, nil
			}
		}

	default:
		return nil, fmt.Errorf("Invalid index type: %v", indicesType)
	}
}

// The draw calls below are stubbed.
func (GlDrawArraysIndirect) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawArraysIndirect.getIndices() not implemented")
}
func (GlDrawArraysInstanced) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawArraysInstanced.getIndices() not implemented")
}
func (GlDrawArraysInstancedANGLE) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawArraysInstancedANGLE.getIndices() not implemented")
}
func (GlDrawArraysInstancedBaseInstanceEXT) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawArraysInstancedBaseInstanceEXT.getIndices() not implemented")
}
func (GlDrawArraysInstancedEXT) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawArraysInstancedEXT.getIndices() not implemented")
}
func (GlDrawArraysInstancedNV) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawArraysInstancedNV.getIndices() not implemented")
}
func (GlDrawElementsBaseVertex) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsBaseVertex.getIndices() not implemented")
}
func (GlDrawElementsBaseVertexEXT) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsBaseVertexEXT.getIndices() not implemented")
}
func (GlDrawElementsBaseVertexOES) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsBaseVertexOES.getIndices() not implemented")
}
func (GlDrawElementsIndirect) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsIndirect.getIndices() not implemented")
}
func (GlDrawElementsInstanced) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstanced.getIndices() not implemented")
}
func (GlDrawElementsInstancedANGLE) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedANGLE.getIndices() not implemented")
}
func (GlDrawElementsInstancedBaseInstanceEXT) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedBaseInstanceEXT.getIndices() not implemented")
}
func (GlDrawElementsInstancedBaseVertex) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedBaseVertex.getIndices() not implemented")
}
func (GlDrawElementsInstancedBaseVertexBaseInstanceEXT) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedBaseVertexBaseInstanceEXT.getIndices() not implemented")
}
func (GlDrawElementsInstancedBaseVertexEXT) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedBaseVertexEXT.getIndices() not implemented")
}
func (GlDrawElementsInstancedBaseVertexOES) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedBaseVertexOES.getIndices() not implemented")
}
func (GlDrawElementsInstancedEXT) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedEXT.getIndices() not implemented")
}
func (GlDrawElementsInstancedNV) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedNV.getIndices() not implemented")
}
func (GlDrawRangeElementsBaseVertex) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawRangeElementsBaseVertex.getIndices() not implemented")
}
func (GlDrawRangeElementsBaseVertexEXT) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawRangeElementsBaseVertexEXT.getIndices() not implemented")
}
func (GlDrawRangeElementsBaseVertexOES) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawRangeElementsBaseVertexOES.getIndices() not implemented")
}
func (GlDrawTexfOES) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexfOES.getIndices() not implemented")
}
func (GlDrawTexfvOES) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexfvOES.getIndices() not implemented")
}
func (GlDrawTexiOES) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexiOES.getIndices() not implemented")
}
func (GlDrawTexivOES) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexivOES.getIndices() not implemented")
}
func (GlDrawTexsOES) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexsOES.getIndices() not implemented")
}
func (GlDrawTexsvOES) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexsvOES.getIndices() not implemented")
}
func (GlDrawTexxOES) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexxOES.getIndices() not implemented")
}
func (GlDrawTexxvOES) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexxvOES.getIndices() not implemented")
}
func (GlDrawTransformFeedbackEXT) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTransformFeedbackEXT.getIndices() not implemented")
}
func (GlDrawTransformFeedbackInstancedEXT) getIndices(context.Context, *Context, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTransformFeedbackInstancedEXT.getIndices() not implemented")
}
