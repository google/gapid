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
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/gapis/api"
)

type drawCallIndices struct {
	indices  []uint32
	drawMode GLenum
	indexed  bool
}

// drawCall is the interface implemented by all GLES draw call commands.
type drawCall interface {
	api.Cmd
	getIndices(ctx context.Context, c Contextʳ, s *api.GlobalState) (drawCallIndices, error)
	getDrawMode() GLenum
}

func (a *GlDrawArrays) getIndices(ctx context.Context, c Contextʳ, s *api.GlobalState) (drawCallIndices, error) {
	indices := make([]uint32, a.IndicesCount())
	for i := range indices {
		indices[i] = uint32(a.FirstIndex()) + uint32(i)
	}
	return drawCallIndices{indices, a.DrawMode(), false}, nil
}

func (a *GlDrawArrays) getDrawMode() GLenum {
	return a.DrawMode()
}

func (a *GlDrawElements) getIndices(ctx context.Context, c Contextʳ, s *api.GlobalState) (drawCallIndices, error) {
	return getIndices(ctx, c, s, a.IndicesType(), a.DrawMode(), 0, a.IndicesCount(), a.Indices())
}

func (a *GlDrawElements) getDrawMode() GLenum {
	return a.DrawMode()
}

func (a *GlDrawRangeElements) getIndices(ctx context.Context, c Contextʳ, s *api.GlobalState) (drawCallIndices, error) {
	return getIndices(ctx, c, s, a.IndicesType(), a.DrawMode(), 0, a.IndicesCount(), a.Indices())
}

func (a *GlDrawRangeElements) getDrawMode() GLenum {
	return a.DrawMode()
}

func getIndices(
	ctx context.Context,
	c Contextʳ,
	s *api.GlobalState,
	ty, drawMode GLenum,
	first, count GLsizei,
	ptr IndicesPointer) (drawCallIndices, error) {

	indexSize := uint64(DataTypeSize(ty))
	indexBuffer := c.Bound().VertexArray().ElementArrayBuffer()
	size := uint64(count) * indexSize
	offset := uint64(first) * indexSize

	var reader binary.Reader
	if indexBuffer.IsNil() {
		// Get the index buffer data from pointer
		reader = ptr.Slice(offset, size, s.MemoryLayout).Reader(ctx, s)
	} else {
		// Get the index buffer data from buffer, offset by the 'indices' pointer.
		offset += ptr.Address()
		count := indexBuffer.Data().Count()
		start := u64.Min(offset, count)
		end := u64.Min(offset+size, count)
		reader = indexBuffer.Data().Slice(start, end).Reader(ctx, s)
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
func (GlDrawArraysIndirect) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawArraysIndirect.getIndices() not implemented")
}
func (a *GlDrawArraysIndirect) getDrawMode() GLenum {
	return a.DrawMode()
}

func (GlDrawArraysInstanced) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawArraysInstanced.getIndices() not implemented")
}
func (a *GlDrawArraysInstanced) getDrawMode() GLenum {
	return a.DrawMode()
}

func (GlDrawArraysInstancedANGLE) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawArraysInstancedANGLE.getIndices() not implemented")
}
func (a *GlDrawArraysInstancedANGLE) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawArraysInstancedBaseInstanceEXT) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawArraysInstancedBaseInstanceEXT.getIndices() not implemented")
}
func (a *GlDrawArraysInstancedBaseInstanceEXT) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawArraysInstancedEXT) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawArraysInstancedEXT.getIndices() not implemented")
}
func (a *GlDrawArraysInstancedEXT) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawArraysInstancedNV) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawArraysInstancedNV.getIndices() not implemented")
}
func (a *GlDrawArraysInstancedNV) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawElementsBaseVertex) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsBaseVertex.getIndices() not implemented")
}
func (a *GlDrawElementsBaseVertex) getDrawMode() GLenum {
	return a.DrawMode()
}

func (GlDrawElementsBaseVertexEXT) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsBaseVertexEXT.getIndices() not implemented")
}
func (a *GlDrawElementsBaseVertexEXT) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawElementsBaseVertexOES) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsBaseVertexOES.getIndices() not implemented")
}
func (a *GlDrawElementsBaseVertexOES) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawElementsIndirect) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsIndirect.getIndices() not implemented")
}
func (a *GlDrawElementsIndirect) getDrawMode() GLenum {
	return a.DrawMode()
}

func (GlDrawElementsInstanced) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstanced.getIndices() not implemented")
}
func (a *GlDrawElementsInstanced) getDrawMode() GLenum {
	return a.DrawMode()
}

func (GlDrawElementsInstancedANGLE) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedANGLE.getIndices() not implemented")
}
func (a *GlDrawElementsInstancedANGLE) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawElementsInstancedBaseInstanceEXT) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedBaseInstanceEXT.getIndices() not implemented")
}
func (a *GlDrawElementsInstancedBaseInstanceEXT) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawElementsInstancedBaseVertex) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedBaseVertex.getIndices() not implemented")
}
func (a *GlDrawElementsInstancedBaseVertex) getDrawMode() GLenum {
	return a.DrawMode()
}

func (GlDrawElementsInstancedBaseVertexBaseInstanceEXT) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedBaseVertexBaseInstanceEXT.getIndices() not implemented")
}
func (a *GlDrawElementsInstancedBaseVertexBaseInstanceEXT) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawElementsInstancedBaseVertexEXT) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedBaseVertexEXT.getIndices() not implemented")
}
func (a *GlDrawElementsInstancedBaseVertexEXT) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawElementsInstancedBaseVertexOES) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedBaseVertexOES.getIndices() not implemented")
}
func (a *GlDrawElementsInstancedBaseVertexOES) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawElementsInstancedEXT) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedEXT.getIndices() not implemented")
}
func (a *GlDrawElementsInstancedEXT) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawElementsInstancedNV) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawElementsInstancedNV.getIndices() not implemented")
}
func (a *GlDrawElementsInstancedNV) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawRangeElementsBaseVertex) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawRangeElementsBaseVertex.getIndices() not implemented")
}
func (a *GlDrawRangeElementsBaseVertex) getDrawMode() GLenum {
	return a.DrawMode()
}

func (GlDrawRangeElementsBaseVertexEXT) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawRangeElementsBaseVertexEXT.getIndices() not implemented")
}
func (a *GlDrawRangeElementsBaseVertexEXT) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawRangeElementsBaseVertexOES) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawRangeElementsBaseVertexOES.getIndices() not implemented")
}
func (a *GlDrawRangeElementsBaseVertexOES) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawTexfOES) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexfOES.getIndices() not implemented")
}
func (a *GlDrawTexfOES) getDrawMode() GLenum {
	return GLenum_GL_TRIANGLES
}

func (GlDrawTexfvOES) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexfvOES.getIndices() not implemented")
}
func (a *GlDrawTexfvOES) getDrawMode() GLenum {
	return GLenum_GL_TRIANGLES
}

func (GlDrawTexiOES) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexiOES.getIndices() not implemented")
}
func (a *GlDrawTexiOES) getDrawMode() GLenum {
	return GLenum_GL_TRIANGLES
}

func (GlDrawTexivOES) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexivOES.getIndices() not implemented")
}
func (a *GlDrawTexivOES) getDrawMode() GLenum {
	return GLenum_GL_TRIANGLES
}

func (GlDrawTexsOES) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexsOES.getIndices() not implemented")
}
func (a *GlDrawTexsOES) getDrawMode() GLenum {
	return GLenum_GL_TRIANGLES
}

func (GlDrawTexsvOES) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexsvOES.getIndices() not implemented")
}
func (a *GlDrawTexsvOES) getDrawMode() GLenum {
	return GLenum_GL_TRIANGLES
}

func (GlDrawTexxOES) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexxOES.getIndices() not implemented")
}
func (a *GlDrawTexxOES) getDrawMode() GLenum {
	return GLenum_GL_TRIANGLES
}

func (GlDrawTexxvOES) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTexxvOES.getIndices() not implemented")
}
func (a *GlDrawTexxvOES) getDrawMode() GLenum {
	return GLenum_GL_TRIANGLES
}

func (GlDrawTransformFeedbackEXT) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTransformFeedbackEXT.getIndices() not implemented")
}
func (a *GlDrawTransformFeedbackEXT) getDrawMode() GLenum {
	return a.Mode()
}

func (GlDrawTransformFeedbackInstancedEXT) getIndices(context.Context, Contextʳ, *api.GlobalState) (drawCallIndices, error) {
	return drawCallIndices{}, fmt.Errorf("GlDrawTransformFeedbackInstancedEXT.getIndices() not implemented")
}
func (a *GlDrawTransformFeedbackInstancedEXT) getDrawMode() GLenum {
	return a.Mode()
}
