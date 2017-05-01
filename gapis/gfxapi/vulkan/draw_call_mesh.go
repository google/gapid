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

package vulkan

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/stream"
	"github.com/google/gapid/core/stream/fmts"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/vertex"
)

// drawCallMesh builds a mesh for dc at p.
func drawCallMesh(ctx context.Context, dc *VkQueueSubmit, p *path.Mesh) (*gfxapi.Mesh, error) {
	cmdPath := path.FindCommand(p)
	if cmdPath == nil {
		log.W(ctx, "Couldn't find command at path '%v'", p)
		return nil, nil
	}

	s, err := resolve.GlobalState(ctx, cmdPath.StateAfter())
	if err != nil {
		return nil, err
	}

	lastDrawInfo := getStateObject(s).LastDrawInfo

	// Get the draw primitive from the currente graphics pipeline
	if lastDrawInfo.GraphicsPipeline == nil {
		return nil, fmt.Errorf("Cannot found last used graphics pipeline")
	}
	drawPrimitive := func() gfxapi.DrawPrimitive {
		switch lastDrawInfo.GraphicsPipeline.InputAssemblyState.Topology {
		case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_POINT_LIST:
			return gfxapi.DrawPrimitive_Points
		case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_LINE_LIST:
			return gfxapi.DrawPrimitive_Lines
		case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_LINE_STRIP:
			return gfxapi.DrawPrimitive_LineStrip
		case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST:
			return gfxapi.DrawPrimitive_Triangles
		case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_STRIP:
			return gfxapi.DrawPrimitive_TriangleStrip
		case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_FAN:
			return gfxapi.DrawPrimitive_TriangleFan
		}
		return gfxapi.DrawPrimitive_Points
	}()

	// Index buffer
	ib := &gfxapi.IndexBuffer{}
	// Vertex buffer streams
	vb := &vertex.Buffer{}

	// In total there are four kinds of draw calls: vkCmdDraw, vkCmdDrawIndexed,
	// vkCmdDrawIndirect, vkCmdDrawIndexedIndirect. Each is processed in one of
	// the branches.
	if p := lastDrawInfo.CommandParameters.Draw; p != nil {
		// Last draw call is vkCmdDraw
		// Generate an index buffer with value: 0, 1, 2, 3 ... vertexCount-1
		indices := make([]uint32, p.VertexCount)
		for i, _ := range indices {
			indices[i] = uint32(i)
		}
		ib = &gfxapi.IndexBuffer{Indices: []uint32(indices)}

		// Get the current bound vertex buffers
		vb, err = getVertexBuffers(ctx, s, p.VertexCount, p.FirstVertex)
		if err != nil {
			log.W(ctx, "err is not NIL!")
			return nil, err
		}

	} else if p := lastDrawInfo.CommandParameters.DrawIndexed; p != nil {
		// Last draw call is vkCmdDrawIndexed
		// Get the current bound index buffer
		if lastDrawInfo.BoundIndexBuffer.BoundBuffer.Buffer == nil {
			return nil, fmt.Errorf("Cannot found last used index buffer")
		}
		indices := getIndicesData(ctx, s, lastDrawInfo.BoundIndexBuffer, p.IndexCount, p.FirstIndex, p.VertexOffset)
		ib = &gfxapi.IndexBuffer{
			Indices: []uint32(indices),
		}

		// Calculate the vertex count and the first vertex
		maxIndex := uint32(0)
		minIndex := uint32(0xFFFFFFFF)
		for _, i := range indices {
			if maxIndex < i {
				maxIndex = i
			}
			if i < minIndex {
				minIndex = i
			}
		}
		vertexCount := maxIndex - minIndex + 1

		// Get the current bound vertex buffers
		vb, err = getVertexBuffers(ctx, s, vertexCount, minIndex)
		if err != nil {
			return nil, err
		}

	} else if p := lastDrawInfo.CommandParameters.DrawIndirect; p != nil {
		return nil, fmt.Errorf("Draw mesh for vkCmdDrawIndirect not implemented")
	} else if p := lastDrawInfo.CommandParameters.DrawIndexedIndirect; p != nil {
		return nil, fmt.Errorf("Draw mesh for vkCmdDrawIndexedIndirect not implemented")
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

func getIndicesData(ctx context.Context, s *gfxapi.State, boundIndexBuffer *BoundIndexBuffer, indexCount, firstIndex uint32, vertexOffset int32) []uint32 {
	backingMem := boundIndexBuffer.BoundBuffer.Buffer.Memory
	if backingMem == nil {
		return []uint32{}
	}
	bufferMemoryOffset := uint64(boundIndexBuffer.BoundBuffer.Buffer.MemoryOffset)
	indexBindingOffset := uint64(boundIndexBuffer.BoundBuffer.Offset)
	// TODO(qining): Get the maximum size of the bound buffer here from BoundBuffer.Range.
	offset := bufferMemoryOffset + indexBindingOffset

	extractIndices := func(sizeOfIndex uint64) []uint32 {
		indices := []uint32{}
		start := offset + uint64(firstIndex)*sizeOfIndex
		size := uint64(indexCount) * sizeOfIndex
		end := start + size
		indicesSlice := backingMem.Data.Slice(start, end, s)
		for i := uint64(0); (i < size) && (i+sizeOfIndex-1 < size); i += sizeOfIndex {
			index := int32(0)
			for j := uint64(0); j < sizeOfIndex; j++ {
				oneByte := int32(indicesSlice.Index(i+j, s).Read(ctx, nil, s, nil))
				index += oneByte << (8 * j)
			}
			index += vertexOffset
			if index < 0 {
				// TODO(qining): The index value is invalid, need to emit error mesage
				// here.
				index = 0
			}
			indices = append(indices, uint32(index))
		}
		return indices
	}

	switch boundIndexBuffer.Type {
	case VkIndexType_VK_INDEX_TYPE_UINT16:
		return extractIndices(2)
	case VkIndexType_VK_INDEX_TYPE_UINT32:
		return extractIndices(4)
	}
	return []uint32{}
}

func getVertexBuffers(ctx context.Context, s *gfxapi.State,
	vertexCount, firstVertex uint32) (*vertex.Buffer, error) {

	if vertexCount == 0 {
		return nil, fmt.Errorf("Number of vertices must be greater than 0.")
	}

	vb := &vertex.Buffer{}
	// Get the current bound vertex buffers
	lastDrawInfo := getStateObject(s).LastDrawInfo
	attributes := lastDrawInfo.GraphicsPipeline.VertexInputState.AttributeDescriptions
	bindings := lastDrawInfo.GraphicsPipeline.VertexInputState.BindingDescriptions
	// For each attribute, get the vertex buffer data
	for _, attributeIndex := range attributes.KeysSorted() {
		attribute := attributes.Get(attributeIndex)
		if !bindings.Contains(attribute.Binding) {
			// TODO(qining): This is an error, should emit error message here.
			continue
		}
		binding := bindings.Get(attribute.Binding)
		if !lastDrawInfo.BoundVertexBuffers.Contains(binding.Binding) {
			// TODO(qining): This is an error, should emit error message here.
			continue
		}
		boundVertexBuffer := lastDrawInfo.BoundVertexBuffers.Get(binding.Binding)
		vertexData, err := getVerticesData(ctx, s, boundVertexBuffer,
			vertexCount, firstVertex, binding, attribute)
		if err != nil {
			return nil, err
		}
		translatedFormat, err := translateVertexFormat(attribute.Format)
		if err != nil {
			// TODO(qining): This is an error, should emit error message here
			continue
		}
		// TODO: We can disassemble the shader to pull out the debug name if the
		// shader has debug info.
		name := fmt.Sprintf("binding=%v, location=%v", binding.Binding, attribute.Location)
		vb.Streams = append(vb.Streams,
			&vertex.Stream{
				Name:     name,
				Data:     vertexData,
				Format:   translatedFormat,
				Semantic: &vertex.Semantic{},
			})
	}
	guessSemantics(vb)
	return vb, nil
}

func getVerticesData(ctx context.Context, s *gfxapi.State, boundVertexBuffer BoundBuffer,
	vertexCount, firstVertex uint32, binding VkVertexInputBindingDescription,
	attribute VkVertexInputAttributeDescription) ([]byte, error) {

	if vertexCount == 0 {
		return nil, fmt.Errorf("Number of vertices must be greater than 0.")
	}
	if binding.InputRate == VkVertexInputRate_VK_VERTEX_INPUT_RATE_INSTANCE {
		return nil, fmt.Errorf("Instanced draw calls not currently supported.")
	}

	backingMemoryData := boundVertexBuffer.Buffer.Memory.Data
	sliceOffset := uint64(boundVertexBuffer.Offset + boundVertexBuffer.Buffer.MemoryOffset)
	sliceSize := uint64(boundVertexBuffer.Range)
	vertexSlice := backingMemoryData.Slice(sliceOffset, sliceOffset+sliceSize, s)

	formatElementAndTexelBlockSize, err :=
		subGetElementAndTexelBlockSize(ctx, nil, nil, s, nil, nil, attribute.Format)
	if err != nil {
		return nil, err
	}
	perVertexSize := uint64(formatElementAndTexelBlockSize.ElementSize)
	stride := uint64(binding.Stride)

	compactOutputSize := perVertexSize * uint64(vertexCount)
	out := make([]byte, compactOutputSize)

	fullSize := uint64(vertexCount-1)*stride + perVertexSize
	if uint64(attribute.Offset) >= vertexSlice.Count {
		// First vertex sits beyond the end of the buffer.
		// Instead of erroring just return a 0-initialized buffer so other
		// streams can be visualized. The report should display an error to
		// alert the user to the bad data
		// TODO: Actually add this as a report error
		return out, nil
	}
	offset := uint64(attribute.Offset) + (uint64(firstVertex) * stride)
	data := vertexSlice.Slice(offset, offset+fullSize, s).Read(ctx, nil, s, nil)
	if stride > perVertexSize {
		// There are gaps between vertices.
		for i := uint64(0); i < uint64(vertexCount); i++ {
			copy(out[i*perVertexSize:(i+1)*perVertexSize], data[i*stride:])
		}
	} else {
		// No gap between each vertex.
		copy(out, data)
	}
	return out, nil
}

// Translate Vulkan vertex buffer format. Vulkan uses RGBA formats for vertex
// data, the mapping from RGBA channels to XYZW channels are done here.
func translateVertexFormat(vkFormat VkFormat) (*stream.Format, error) {
	switch vkFormat {
	case VkFormat_VK_FORMAT_R8_UNORM:
		return fmts.X_U8_NORM, nil
	case VkFormat_VK_FORMAT_R8_SNORM:
		return fmts.X_S8_NORM, nil
	case VkFormat_VK_FORMAT_R8_UINT:
		return fmts.X_U8, nil
	case VkFormat_VK_FORMAT_R8_SINT:
		return fmts.X_S8, nil

	case VkFormat_VK_FORMAT_R8G8_UNORM:
		return fmts.XY_U8_NORM, nil
	case VkFormat_VK_FORMAT_R8G8_SNORM:
		return fmts.XY_S8_NORM, nil
	case VkFormat_VK_FORMAT_R8G8_UINT:
		return fmts.XY_U8, nil
	case VkFormat_VK_FORMAT_R8G8_SINT:
		return fmts.XY_S8, nil

	case VkFormat_VK_FORMAT_R8G8B8A8_UNORM:
		return fmts.XYZW_U8_NORM, nil
	case VkFormat_VK_FORMAT_R8G8B8A8_SNORM:
		return fmts.XYZW_S8_NORM, nil
	case VkFormat_VK_FORMAT_R8G8B8A8_UINT:
		return fmts.XYZW_U8, nil
	case VkFormat_VK_FORMAT_R8G8B8A8_SINT:
		return fmts.XYZW_S8, nil
	case VkFormat_VK_FORMAT_B8G8R8A8_UNORM:
		return fmts.XYZW_U8_NORM, nil

	case VkFormat_VK_FORMAT_R16_UNORM:
		return fmts.X_U16_NORM, nil
	case VkFormat_VK_FORMAT_R16_SNORM:
		return fmts.X_S16_NORM, nil
	case VkFormat_VK_FORMAT_R16_UINT:
		return fmts.X_U16, nil
	case VkFormat_VK_FORMAT_R16_SINT:
		return fmts.X_S16, nil
	case VkFormat_VK_FORMAT_R16_SFLOAT:
		return fmts.X_F16, nil

	case VkFormat_VK_FORMAT_R16G16_UNORM:
		return fmts.XY_U16_NORM, nil
	case VkFormat_VK_FORMAT_R16G16_SNORM:
		return fmts.XY_S16_NORM, nil
	case VkFormat_VK_FORMAT_R16G16_UINT:
		return fmts.XY_U16, nil
	case VkFormat_VK_FORMAT_R16G16_SINT:
		return fmts.XY_S16, nil
	case VkFormat_VK_FORMAT_R16G16_SFLOAT:
		return fmts.XY_F16, nil

	case VkFormat_VK_FORMAT_R16G16B16A16_UNORM:
		return fmts.XYZW_U16_NORM, nil
	case VkFormat_VK_FORMAT_R16G16B16A16_SNORM:
		return fmts.XYZW_S16_NORM, nil
	case VkFormat_VK_FORMAT_R16G16B16A16_UINT:
		return fmts.XYZW_U16, nil
	case VkFormat_VK_FORMAT_R16G16B16A16_SINT:
		return fmts.XYZW_S16, nil
	case VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT:
		return fmts.XYZW_F16, nil

	case VkFormat_VK_FORMAT_R32_UINT:
		return fmts.X_U32, nil
	case VkFormat_VK_FORMAT_R32_SINT:
		return fmts.X_S32, nil
	case VkFormat_VK_FORMAT_R32_SFLOAT:
		return fmts.X_F32, nil

	case VkFormat_VK_FORMAT_R32G32_UINT:
		return fmts.XY_U32, nil
	case VkFormat_VK_FORMAT_R32G32_SINT:
		return fmts.XY_S32, nil
	case VkFormat_VK_FORMAT_R32G32_SFLOAT:
		return fmts.XY_F32, nil

	case VkFormat_VK_FORMAT_R32G32B32_UINT:
		return fmts.XYZ_U32, nil
	case VkFormat_VK_FORMAT_R32G32B32_SINT:
		return fmts.XYZ_S32, nil
	case VkFormat_VK_FORMAT_R32G32B32_SFLOAT:
		return fmts.XYZ_F32, nil

	case VkFormat_VK_FORMAT_R32G32B32A32_UINT:
		return fmts.XYZW_U32, nil
	case VkFormat_VK_FORMAT_R32G32B32A32_SINT:
		return fmts.XYZW_S32, nil
	case VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT:
		return fmts.XYZW_F32, nil

	// TODO(qining): Support packed format
	case VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32,
		VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32,
		VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32,
		VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32:
		return nil, fmt.Errorf("Packed format not supported yet")

	default:
		return nil, fmt.Errorf("Unsupported format as vertex format")
	}
	return nil, fmt.Errorf("Unsupported format as vertex format")
}

func guessSemantics(vb *vertex.Buffer) {
	// TODO: We may disassemble the shader to pull out the debug name to help
	// this semantics guessing, if the shader has debug info.
	numOfElementsToSemanticTypes := map[uint32][]vertex.Semantic_Type{
		4: {vertex.Semantic_Position,
			vertex.Semantic_Normal,
			vertex.Semantic_Color},
		3: {vertex.Semantic_Position,
			vertex.Semantic_Normal,
			vertex.Semantic_Color},
		2: {vertex.Semantic_Position,
			vertex.Semantic_Texcoord},
	}

	taken := map[vertex.Semantic_Type]bool{}
	for _, s := range vb.Streams {
		numOfElements := uint32(len(s.Format.Components))
		for _, t := range numOfElementsToSemanticTypes[numOfElements] {
			if taken[t] {
				continue
			}
			s.Semantic.Type = t
			taken[t] = true
			break
		}
	}
}
