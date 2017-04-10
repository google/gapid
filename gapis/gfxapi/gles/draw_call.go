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

	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
)

// drawCall is the interface implemented by all GLES draw call atoms.
type drawCall interface {
	atom.Atom
	getIndices(
		ctx context.Context,
		c *Context,
		s *gfxapi.State) ([]uint32, GLenum, error)
}

func (a *GlDrawArrays) getIndices(
	ctx context.Context,
	c *Context,
	s *gfxapi.State) ([]uint32, GLenum, error) {

	indices := make([]uint32, a.IndicesCount)
	for i := range indices {
		indices[i] = uint32(a.FirstIndex) + uint32(i)
	}
	return indices, a.DrawMode, nil
}

func (a *GlDrawElements) getIndices(
	ctx context.Context,
	c *Context,
	s *gfxapi.State) ([]uint32, GLenum, error) {

	indexSize := map[GLenum]uint64{
		GLenum_GL_UNSIGNED_BYTE:  1,
		GLenum_GL_UNSIGNED_SHORT: 2,
		GLenum_GL_UNSIGNED_INT:   4,
	}[a.IndicesType]
	indexBufferID := c.Objects.VertexArrays[c.BoundVertexArray].ElementArrayBuffer
	size := uint64(a.IndicesCount) * indexSize

	var decoder pod.Reader
	if indexBufferID == 0 {
		// Get the index buffer data from pointer
		decoder = a.Indices.Slice(0, size, s).Decoder(ctx, s)
	} else {
		// Get the index buffer data from buffer, offset by the 'indices' pointer.
		indexBuffer := c.SharedObjects.Buffers[indexBufferID]
		if indexBuffer == nil {
			return nil, 0, fmt.Errorf("Can not find buffer %v", indexBufferID)
		}
		offset := uint64(a.Indices.Address)
		decoder = indexBuffer.Data.Slice(offset, offset+size, s).Decoder(ctx, s)
	}

	indices, err := decodeIndices(decoder, a.IndicesType)
	return indices, a.DrawMode, err
}

// decodeIndices assumes little endian encoding
func decodeIndices(r pod.Reader, indicesType GLenum) ([]uint32, error) {
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
func (GlDrawArraysIndirect) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawArraysIndirect.getIndices() not implemented")
}
func (GlDrawArraysInstanced) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawArraysInstanced.getIndices() not implemented")
}
func (GlDrawArraysInstancedANGLE) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawArraysInstancedANGLE.getIndices() not implemented")
}
func (GlDrawArraysInstancedBaseInstanceEXT) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawArraysInstancedBaseInstanceEXT.getIndices() not implemented")
}
func (GlDrawArraysInstancedEXT) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawArraysInstancedEXT.getIndices() not implemented")
}
func (GlDrawArraysInstancedNV) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawArraysInstancedNV.getIndices() not implemented")
}
func (GlDrawElementsBaseVertex) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsBaseVertex.getIndices() not implemented")
}
func (GlDrawElementsBaseVertexEXT) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsBaseVertexEXT.getIndices() not implemented")
}
func (GlDrawElementsBaseVertexOES) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsBaseVertexOES.getIndices() not implemented")
}
func (GlDrawElementsIndirect) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsIndirect.getIndices() not implemented")
}
func (GlDrawElementsInstanced) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsInstanced.getIndices() not implemented")
}
func (GlDrawElementsInstancedANGLE) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsInstancedANGLE.getIndices() not implemented")
}
func (GlDrawElementsInstancedBaseInstanceEXT) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsInstancedBaseInstanceEXT.getIndices() not implemented")
}
func (GlDrawElementsInstancedBaseVertex) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsInstancedBaseVertex.getIndices() not implemented")
}
func (GlDrawElementsInstancedBaseVertexBaseInstanceEXT) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsInstancedBaseVertexBaseInstanceEXT.getIndices() not implemented")
}
func (GlDrawElementsInstancedBaseVertexEXT) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsInstancedBaseVertexEXT.getIndices() not implemented")
}
func (GlDrawElementsInstancedBaseVertexOES) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsInstancedBaseVertexOES.getIndices() not implemented")
}
func (GlDrawElementsInstancedEXT) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsInstancedEXT.getIndices() not implemented")
}
func (GlDrawElementsInstancedNV) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawElementsInstancedNV.getIndices() not implemented")
}
func (GlDrawRangeElements) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawRangeElements.getIndices() not implemented")
}
func (GlDrawRangeElementsBaseVertex) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawRangeElementsBaseVertex.getIndices() not implemented")
}
func (GlDrawRangeElementsBaseVertexEXT) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawRangeElementsBaseVertexEXT.getIndices() not implemented")
}
func (GlDrawRangeElementsBaseVertexOES) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawRangeElementsBaseVertexOES.getIndices() not implemented")
}
func (GlDrawTexfOES) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawTexfOES.getIndices() not implemented")
}
func (GlDrawTexfvOES) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawTexfvOES.getIndices() not implemented")
}
func (GlDrawTexiOES) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawTexiOES.getIndices() not implemented")
}
func (GlDrawTexivOES) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawTexivOES.getIndices() not implemented")
}
func (GlDrawTexsOES) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawTexsOES.getIndices() not implemented")
}
func (GlDrawTexsvOES) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawTexsvOES.getIndices() not implemented")
}
func (GlDrawTexxOES) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawTexxOES.getIndices() not implemented")
}
func (GlDrawTexxvOES) getIndices(context.Context, *Context, *gfxapi.State) ([]uint32, GLenum, error) {
	return nil, 0, fmt.Errorf("GlDrawTexxvOES.getIndices() not implemented")
}
