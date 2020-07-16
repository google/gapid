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

package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/f32"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/stream"
	"github.com/google/gapid/core/stream/fmts"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/vertex"
)

var (
	// ErrMeshNotAvailable is an error returned by MeshProvider if a mesh is
	// requested on an object that does not have a mesh (e.g. non-draw call).
	ErrMeshNotAvailable = errors.New("Mesh not available at this command")
)

// MeshProvider is the interface implemented by types that provide meshes.
type MeshProvider interface {
	// Mesh returns the mesh representation of the object o.
	Mesh(ctx context.Context, o interface{}, p *path.Mesh, r *path.ResolveConfig) (*Mesh, error)
}

// Faceted returns a new mesh with each shared vertex split, and each normal set
// to the triangle's normal.
func (m *Mesh) Faceted(ctx context.Context) (*Mesh, error) {
	switch m.DrawPrimitive {
	case DrawPrimitive_Lines, DrawPrimitive_LineStrip, DrawPrimitive_LineLoop:
		return m, nil // These are already as faceted as they're going to get.
	}

	triangleCount := m.TriangleCount()
	vertexCount := triangleCount * 3

	// Get all the sequential indices for all the triangles
	indices := make([]uint32, 0, vertexCount)
	for t := 0; t < triangleCount; t++ {
		a, b, c := m.Triangle(t)
		indices = append(indices, a, b, c)
	}

	streams := make([]*vertex.Stream, 0, len(m.VertexBuffer.Streams))
	addStream := func(
		name string,
		format *stream.Format,
		semantic *vertex.Semantic,
		data []byte) error {

		stream := &vertex.Stream{
			Name:     name,
			Data:     data,
			Format:   format,
			Semantic: semantic,
		}
		streams = append(streams, stream)
		return nil
	}

	for _, s := range m.VertexBuffer.Streams {
		if s.Semantic.Type == vertex.Semantic_Normal {
			continue // Going to overrite this.
		}

		// Explode the vertices by triangles
		vertexStride := uint32(s.Format.Stride())
		vertices := make([]byte, s.Format.Size(vertexCount))
		for i, j := range indices {
			i := uint32(i)
			copy(vertices[i*vertexStride:(i+1)*vertexStride],
				s.Data[j*vertexStride:(j+1)*vertexStride])
		}

		// Transform the vertices back to their original format
		addStream(s.Name, s.Format, s.Semantic, vertices)

		if s.Semantic.Type == vertex.Semantic_Position {
			// Convert position stream to something we can work with
			posData, err := stream.Convert(fmts.XYZ_F32, s.Format, vertices)
			if err != nil {
				return nil, log.Err(ctx, err, "Couldn't convert position stream")
			}
			vectors := bytesToVec3Ds(posData)
			// Build the per-triangle normals
			for t := 0; t < triangleCount; t++ {
				i := t * 3
				a, b, c := vectors[i+0], vectors[i+1], vectors[i+2]
				ab, ac := f32.Sub3D(b, a), f32.Sub3D(c, a)
				normal := f32.Cross3D(ab, ac).Normalize()
				vectors[i+0], vectors[i+1], vectors[i+2] = normal, normal, normal
			}
			normals := vec3DsToBytes(vectors)
			semantic := &vertex.Semantic{Type: vertex.Semantic_Normal, Index: s.Semantic.Index}
			addStream("normals", fmts.XYZ_F32, semantic, normals)
		}
	}

	// Build an index buffer with sequential indices.
	ib := &IndexBuffer{
		Indices: make([]uint32, vertexCount),
	}
	for i := range ib.Indices {
		ib.Indices[i] = uint32(i)
	}

	return &Mesh{
		DrawPrimitive: DrawPrimitive_Triangles,
		VertexBuffer:  &vertex.Buffer{Streams: streams},
		IndexBuffer:   ib,
		Stats:         m.Stats,
	}, nil
}

// TriangleCount returns the number of triangles this mesh contains.
func (m *Mesh) TriangleCount() int {
	switch m.DrawPrimitive {
	case DrawPrimitive_Triangles, DrawPrimitive_TriangleStrip, DrawPrimitive_TriangleFan:
		return int(m.DrawPrimitive.Count(uint32(len(m.IndexBuffer.Indices))))
	default:
		return 0
	}
}

// Triangle returns the 3 vertex indices for the i'th triangle.
func (m *Mesh) Triangle(i int) (a, b, c uint32) {
	indices := m.IndexBuffer.Indices
	switch m.DrawPrimitive {
	case DrawPrimitive_Lines, DrawPrimitive_LineStrip, DrawPrimitive_LineLoop:
		return 0, 0, 0
	case DrawPrimitive_Triangles:
		t := i * 3
		return indices[t], indices[t+1], indices[t+2]
	case DrawPrimitive_TriangleStrip:
		//  0---2---4
		//  | / | / |
		//  1---3---5
		if i&1 == 0 {
			return indices[i], indices[i+1], indices[i+2]
		}
		return indices[i+2], indices[i+1], indices[i]
	case DrawPrimitive_TriangleFan:
		//  1--2
		//  | /|
		//  0--3
		//  | \|
		//  5--4
		return indices[0], indices[i+1], indices[i+2]
	default:
		panic(fmt.Errorf("Unknown DrawPrimitive value: %v", m.DrawPrimitive))
	}
}

func bytesToVec3Ds(data []byte) []f32.Vec3 {
	r := endian.Reader(bytes.NewReader(data), device.LittleEndian)
	out := make([]f32.Vec3, len(data)/(3*4))
	for i := range out {
		for j := 0; j < 3; j++ {
			out[i][j] = r.Float32()
		}
	}
	return out
}

func vec3DsToBytes(vecs []f32.Vec3) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, len(vecs)*(3*4)))
	w := endian.Writer(buf, device.LittleEndian)
	for _, v := range vecs {
		for i := 0; i < 3; i++ {
			w.Float32(v[i])
		}
	}
	return buf.Bytes()
}

// ConvertTo converts the vertex buffer to the requested format.
func (m *Mesh) ConvertTo(ctx context.Context, f *vertex.BufferFormat) (*Mesh, error) {
	vb, err := m.VertexBuffer.ConvertTo(ctx, f)
	if err != nil {
		return nil, err
	}
	return &Mesh{
		DrawPrimitive: m.DrawPrimitive,
		VertexBuffer:  vb,
		IndexBuffer:   m.IndexBuffer,
		Stats:         m.Stats,
	}, nil
}

// Count returns the primitive count for the given number of vertices.
func (dp DrawPrimitive) Count(vertices uint32) uint32 {
	switch dp {
	case DrawPrimitive_Points, DrawPrimitive_LineLoop:
		return vertices
	case DrawPrimitive_Lines:
		return vertices / 2
	case DrawPrimitive_LineStrip:
		if vertices < 2 {
			return 0
		}
		return vertices - 1
	case DrawPrimitive_Triangles:
		return vertices / 3
	case DrawPrimitive_TriangleStrip, DrawPrimitive_TriangleFan:
		if vertices < 3 {
			return 0
		}
		return vertices - 2
	default:
		return 0
	}
}
