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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/vertex"
)

// drawCallMesh builds a mesh for dc at p.
func drawCallMesh(ctx context.Context, dc *VkQueueSubmit, p *path.Mesh, r *path.ResolveConfig) (*api.Mesh, error) {
	cmdPath := path.FindCommand(p)
	if cmdPath == nil {
		log.W(ctx, "Couldn't find command at path '%v'", p)
		return nil, api.ErrMeshNotAvailable
	}

	cmd, err := resolve.Cmd(ctx, cmdPath, r)
	if err != nil {
		return nil, err
	}

	if !cmd.CmdFlags().IsExecutedDraw() {
		return nil, api.ErrMeshNotAvailable
	}

	s, err := resolve.GlobalState(ctx, cmdPath.GlobalStateAfter(), r)
	if err != nil {
		return nil, err
	}

	c := getStateObject(s)

	lastQueue := c.LastBoundQueue()
	if lastQueue.IsNil() {
		return nil, fmt.Errorf("No previous queue submission")
	}

	lastDrawInfo, ok := c.LastDrawInfos().Lookup(lastQueue.VulkanHandle())
	if !ok {
		return nil, fmt.Errorf("There have been no previous draws")
	}

	// Get the draw primitive from the currente graphics pipeline
	if lastDrawInfo.GraphicsPipeline().IsNil() {
		return nil, fmt.Errorf("Cannot find last used graphics pipeline")
	}
	drawPrimitive := func() api.DrawPrimitive {
		switch lastDrawInfo.GraphicsPipeline().InputAssemblyState().Topology() {
		case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_POINT_LIST:
			return api.DrawPrimitive_Points
		case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_LINE_LIST:
			return api.DrawPrimitive_Lines
		case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_LINE_STRIP:
			return api.DrawPrimitive_LineStrip
		case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST:
			return api.DrawPrimitive_Triangles
		case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_STRIP:
			return api.DrawPrimitive_TriangleStrip
		case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_FAN:
			return api.DrawPrimitive_TriangleFan
		}
		return api.DrawPrimitive_Points
	}()

	// Index buffer
	ib := &api.IndexBuffer{}
	// Vertex buffer streams
	vb := &vertex.Buffer{}

	stats := &api.Mesh_Stats{}

	noData := p.GetOptions().GetExcludeData()

	// In total there are four kinds of draw calls: vkCmdDraw, vkCmdDrawIndexed,
	// vkCmdDrawIndirect, vkCmdDrawIndexedIndirect. Each is processed in one of
	// the branches.
	if p := lastDrawInfo.CommandParameters().Draw(); !p.IsNil() {
		// Last draw call is vkCmdDraw
		// Generate an index buffer with value: 0, 1, 2, 3 ... vertexCount-1
		var indices []uint32
		if !noData {
			indices := make([]uint32, p.VertexCount())
			for i := range indices {
				indices[i] = uint32(i)
			}
		}
		ib = &api.IndexBuffer{Indices: indices}

		// Get the current bound vertex buffers
		vb, err = getVertexBuffers(ctx, s, dc.Thread(), p.VertexCount(), p.FirstVertex(), noData)
		if err != nil {
			return nil, err
		}
		stats.Vertices = p.VertexCount()
		stats.Primitives = drawPrimitive.Count(p.VertexCount())
	} else if p := lastDrawInfo.CommandParameters().DrawIndexed(); !p.IsNil() {
		// Last draw call is vkCmdDrawIndexed
		// Get the current bound index buffer
		if lastDrawInfo.BoundIndexBuffer().BoundBuffer().Buffer().IsNil() {
			return nil, fmt.Errorf("Cannot find last used index buffer")
		}

		var indices []uint32
		if !noData {
			indices, err = getIndicesData(ctx, s, dc.Thread(), lastDrawInfo.BoundIndexBuffer(), p.IndexCount(), p.FirstIndex(), p.VertexOffset())
			if err != nil {
				return nil, err
			}
		}

		// Calculate the vertex count and the first vertex
		maxIndex := uint32(0)
		minIndex := uint32(0xFFFFFFFF)
		uniqueIndices := make(map[uint32]bool)
		for _, i := range indices {
			if maxIndex < i {
				maxIndex = i
			}
			if i < minIndex {
				minIndex = i
			}
			uniqueIndices[i] = true
		}
		vertexCount := maxIndex - minIndex + 1
		// Get the current bound vertex buffers
		vb, err = getVertexBuffers(ctx, s, dc.Thread(), vertexCount, minIndex, noData)
		if err != nil {
			return nil, err
		}

		// Shift indices, as we only extract the vertex data from minIndex to
		// maxIndex, we need to minus the minimum index value make the new indices
		// value valid for the extracted vertices value.
		shiftedIndices := make([]uint32, len(indices))
		for i, index := range indices {
			shiftedIndices[i] = index - minIndex
		}
		ib = &api.IndexBuffer{
			Indices: shiftedIndices,
		}
		stats.Vertices = uint32(len(uniqueIndices))
		stats.Indices = p.IndexCount()
		stats.Primitives = drawPrimitive.Count(p.IndexCount())
	} else if p := lastDrawInfo.CommandParameters().DrawIndirect(); !p.IsNil() {
		return nil, fmt.Errorf("Draw mesh for vkCmdDrawIndirect not implemented")
	} else if p := lastDrawInfo.CommandParameters().DrawIndexedIndirect(); !p.IsNil() {
		return nil, fmt.Errorf("Draw mesh for vkCmdDrawIndexedIndirect not implemented")
	} else if p := lastDrawInfo.CommandParameters().DrawIndirectCountKHR(); !p.IsNil() {
		return nil, fmt.Errorf("Draw mesh for vkCmdDrawIndirectCountKHR not implemented")
	} else if p := lastDrawInfo.CommandParameters().DrawIndexedIndirectCountKHR(); !p.IsNil() {
		return nil, fmt.Errorf("Draw mesh for vkCmdDrawIndexedIndirectCountKHR not implemented")
	} else if p := lastDrawInfo.CommandParameters().DrawIndirectCountAMD(); !p.IsNil() {
		return nil, fmt.Errorf("Draw mesh for vkCmdDrawIndirectCountAMD not implemented")
	} else if p := lastDrawInfo.CommandParameters().DrawIndexedIndirectCountAMD(); !p.IsNil() {
		return nil, fmt.Errorf("Draw mesh for vkCmdDrawIndexedIndirectCountAMD not implemented")
	}

	guessSemantics(vb, p.Options.Hints())

	mesh := &api.Mesh{
		DrawPrimitive: drawPrimitive,
		VertexBuffer:  vb,
		IndexBuffer:   ib,
		Stats:         stats,
	}

	if p.Options != nil && p.Options.Faceted {
		return mesh.Faceted(ctx)
	}
	return mesh, nil
}

func getIndicesData(ctx context.Context, s *api.GlobalState, thread uint64, boundIndexBuffer BoundIndexBufferʳ, indexCount, firstIndex uint32, vertexOffset int32) ([]uint32, error) {
	backingMem := boundIndexBuffer.BoundBuffer().Buffer().Memory()
	if backingMem.IsNil() {
		return []uint32{}, nil
	}

	extractIndices := func(sizeOfIndex uint64) ([]uint32, error) {
		indices := []uint32{}
		size := uint64(indexCount) * sizeOfIndex

		backingMemoryPieces, err := subGetBufferBoundMemoryPiecesInRange(
			ctx, nil, api.CmdNoID, nil, s, nil, thread, nil, nil, boundIndexBuffer.BoundBuffer().Buffer(),
			boundIndexBuffer.BoundBuffer().Offset()+VkDeviceSize(uint64(firstIndex)*sizeOfIndex),
			VkDeviceSize(size))
		if err != nil {
			return []uint32{}, err
		}
		rawIndicesData := make([]byte, 0, uint64(indexCount)*sizeOfIndex)
		// In the order of the offsets in the buffer
		for _, bufOffset := range backingMemoryPieces.Keys() {
			piece := backingMemoryPieces.Get(bufOffset)
			data, err := piece.DeviceMemory().Data().Slice(
				uint64(piece.MemoryOffset()),
				uint64(piece.MemoryOffset()+piece.Size())).Read(ctx, nil, s, nil)
			if err != nil {
				return []uint32{}, err
			}
			rawIndicesData = append(rawIndicesData, data...)
		}
		if uint64(len(rawIndicesData)) < size {
			log.E(ctx, "Shadow memory of index buffer is not big enough")
			return []uint32{}, nil
		}

		for i := uint64(0); (i < size) && (i+sizeOfIndex-1 < size); i += sizeOfIndex {
			index := int32(0)
			for j := uint64(0); j < sizeOfIndex; j++ {
				if i+j > uint64(len(rawIndicesData)) {
					return nil, err
				}
				oneByte := rawIndicesData[i+j]
				index += int32(oneByte) << (8 * j)
			}
			index += vertexOffset
			if index < 0 {
				// TODO(qining): The index value is invalid, need to emit error mesage
				// here.
				index = 0
			}
			indices = append(indices, uint32(index))
		}
		return indices, nil
	}

	switch boundIndexBuffer.Type() {
	case VkIndexType_VK_INDEX_TYPE_UINT8_EXT:
		return extractIndices(1)
	case VkIndexType_VK_INDEX_TYPE_UINT16:
		return extractIndices(2)
	case VkIndexType_VK_INDEX_TYPE_UINT32:
		return extractIndices(4)
	default:
		return []uint32{}, fmt.Errorf("Missing index buffer type %v", boundIndexBuffer.Type())
	}
}

func findBinding(lastDrawInfo DrawInfoʳ, attribute VkVertexInputAttributeDescription) (VkVertexInputBindingDescription, bool) {
	bindings := lastDrawInfo.GraphicsPipeline().VertexInputState().BindingDescriptions()
	for _, b := range bindings.All() {
		if b.Binding() == attribute.Binding() {
			return b, true
		}
	}
	return VkVertexInputBindingDescription{}, false
}

func getVertexBuffers(ctx context.Context, s *api.GlobalState, thread uint64,
	vertexCount, firstVertex uint32, noData bool) (*vertex.Buffer, error) {

	if !noData && vertexCount == 0 {
		return nil, fmt.Errorf("Number of vertices must be greater than 0")
	}

	c := getStateObject(s)

	lastQueue := c.LastBoundQueue()
	if lastQueue.IsNil() {
		return nil, fmt.Errorf("No previous queue submission")
	}

	lastDrawInfo, ok := c.LastDrawInfos().Lookup(lastQueue.VulkanHandle())
	if !ok {
		return nil, fmt.Errorf("There have been no previous draws")
	}

	vb := &vertex.Buffer{}
	attributes := lastDrawInfo.GraphicsPipeline().VertexInputState().AttributeDescriptions()
	var err error
	// For each attribute, get the vertex buffer data
	for _, attributeIndex := range attributes.Keys() {
		attribute := attributes.Get(attributeIndex)
		binding, ok := findBinding(lastDrawInfo, attribute)
		if !ok {
			// TODO(qining): This is an error, should emit error message here.
			continue
		}
		if !lastDrawInfo.BoundVertexBuffers().Contains(binding.Binding()) {
			// TODO(qining): This is an error, should emit error message here.
			continue
		}

		var vertexData []byte
		if !noData {
			boundVertexBuffer := lastDrawInfo.BoundVertexBuffers().Get(binding.Binding())
			vertexData, err = getVerticesData(ctx, s, thread, boundVertexBuffer,
				vertexCount, firstVertex, binding, attribute)
			if err != nil {
				return nil, err
			}
		}
		if noData || vertexData != nil {
			translatedFormat, err := translateVertexFormat(attribute.Fmt())
			if err != nil {
				// TODO(qining): This is an error, should emit error message here
				continue
			}
			// TODO: We can disassemble the shader to pull out the debug name if the
			// shader has debug info.
			name := fmt.Sprintf("binding=%v, location=%v", binding.Binding(), attribute.Location())
			vb.Streams = append(vb.Streams,
				&vertex.Stream{
					Name:     name,
					Data:     vertexData,
					Format:   translatedFormat,
					Semantic: &vertex.Semantic{},
				})
		}
	}
	return vb, nil
}

func getVerticesData(ctx context.Context, s *api.GlobalState, thread uint64,
	boundVertexBuffer BoundBuffer, vertexCount, firstVertex uint32,
	binding VkVertexInputBindingDescription,
	attribute VkVertexInputAttributeDescription) ([]byte, error) {

	if vertexCount == 0 {
		return nil, fmt.Errorf("Number of vertices must be greater than 0")
	}
	if binding.InputRate() == VkVertexInputRate_VK_VERTEX_INPUT_RATE_INSTANCE {
		// Instanced draws are not supported, but the first instance's geometry
		// might be still useful. So we ignore any bindings with a instance rate,
		// but do not report an error.
		return nil, nil
	}

	sliceSize := uint64(boundVertexBuffer.Range())

	formatElementAndTexelBlockSize, err :=
		subGetElementAndTexelBlockSize(ctx, nil, api.CmdNoID, nil, s, nil, thread, nil, nil, attribute.Fmt())
	if err != nil {
		return nil, err
	}
	perVertexSize := uint64(formatElementAndTexelBlockSize.ElementSize())
	stride := uint64(binding.Stride())

	compactOutputSize := perVertexSize * uint64(vertexCount)
	out := make([]byte, compactOutputSize)

	fullSize := uint64(vertexCount-1)*stride + perVertexSize

	offset := uint64(attribute.Offset()) + (uint64(firstVertex) * stride)
	if offset >= sliceSize || offset+fullSize > sliceSize {
		// We do not actually have a big enough buffer for this. Return
		// our zero-initialized buffer.
		return out, fmt.Errorf("Vertex data is out of range")
	}

	backingMemoryPieces, err := subGetBufferBoundMemoryPiecesInRange(
		ctx, nil, api.CmdNoID, nil, s, nil, thread, nil, nil, boundVertexBuffer.Buffer(),
		boundVertexBuffer.Offset()+VkDeviceSize(offset),
		VkDeviceSize(fullSize))
	if err != nil {
		return nil, err
	}
	rawData := make([]byte, 0, fullSize)
	for _, bo := range backingMemoryPieces.Keys() {
		ds := uint64(backingMemoryPieces.Get(bo).MemoryOffset())
		de := uint64(backingMemoryPieces.Get(bo).Size()) + ds
		data, err := backingMemoryPieces.Get(bo).DeviceMemory().Data().Slice(ds, de).Read(ctx, nil, s, nil)
		if err != nil {
			return nil, err
		}
		rawData = append(rawData, data...)
	}

	if err != nil {
		return nil, err
	}
	if stride > perVertexSize {
		// There are gaps between vertices.
		for i := uint64(0); i < uint64(vertexCount) && i*stride < uint64(len(rawData)); i++ {
			copy(out[i*perVertexSize:(i+1)*perVertexSize], rawData[i*stride:])
		}
	} else {
		// No gap between each vertex.
		copy(out, rawData)
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
}

func guessSemantics(vb *vertex.Buffer, hints map[string]vertex.Semantic_Type) {
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
	if len(hints) > 0 {
		for _, s := range vb.Streams {
			if t, ok := hints[s.Name]; ok && !taken[t] {
				s.Semantic.Type = t
				taken[t] = true
			}
		}
	}

	for _, s := range vb.Streams {
		if !needsGuess(s) {
			continue
		}

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

func needsGuess(s *vertex.Stream) bool {
	return s.Semantic.Type == vertex.Semantic_Unknown
}
