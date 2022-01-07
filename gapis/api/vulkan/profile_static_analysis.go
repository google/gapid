// Copyright (C) 2021 Google Inc.
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
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/service/path"
)

type counterType uint32

const (
	vertexCountType counterType = iota
	primitiveCountType
	vertexSizeType
	byteCountType
	primitiveSizeType
)

var (
	counters = map[counterType]api.StaticAnalysisCounter{
		vertexCountType: {
			ID:          uint32(vertexCountType),
			Name:        "Vertex Count",
			Description: "The number of input vertices processed.",
			Unit:        "25", // VERTEX
		},
		primitiveCountType: {
			ID:          uint32(primitiveCountType),
			Name:        "Primitive Count",
			Description: "The number of primitives assembled.",
			Unit:        "38", // PRIMITIVE
		},
		vertexSizeType: {
			ID:          uint32(vertexSizeType),
			Name:        "Vertex Size",
			Description: "The combined size of all input bindings, per vertex.",
			Unit:        "7/25", // BYTE / VERTEX
		},
		byteCountType: {
			ID:          uint32(byteCountType),
			Name:        "Byte Count",
			Description: "The total bytes read from index and vertex buffers.",
			Unit:        "7", // BYTE
		},
		primitiveSizeType: {
			ID:          uint32(primitiveSizeType),
			Name:        "Primitive Size",
			Description: "The bytes read from index and vertex buffers, per primitive.",
			Unit:        "7/38", // BYTE / PRIMITIVE
		},
	}
)

type profilingDataBuilder struct {
	data         *api.StaticAnalysisProfileData
	seenCounters map[counterType]struct{}
}

func (db profilingDataBuilder) newSampler(idx api.SubCmdIdx) *api.StaticAnalysisCounterSamples {
	db.data.CounterData = append(db.data.CounterData, api.StaticAnalysisCounterSamples{Index: idx})
	return &db.data.CounterData[len(db.data.CounterData)-1]
}

func (db profilingDataBuilder) addSample(sampler *api.StaticAnalysisCounterSamples, counter counterType, value float64) {
	if _, ok := db.seenCounters[counter]; !ok {
		db.seenCounters[counter] = struct{}{}
		db.data.CounterSpecs = append(db.data.CounterSpecs, counters[counter])
	}
	sampler.Samples = append(sampler.Samples, api.StaticAnalysisCounterSample{uint32(counter), value})
}

// Construct a decoder for reading buffer memory
func getBufferDecoder(ctx context.Context, s *api.GlobalState, buffer BufferObjectʳ, offset VkDeviceSize, size VkDeviceSize) (*memory.Decoder, error) {
	backingMemoryPieces, err := subGetBufferBoundMemoryPiecesInRange(
		ctx, nil, api.CmdNoID, nil, s, nil, 0, nil, nil, buffer, offset, size)
	if err != nil {
		return nil, err
	}
	rawData := make([]byte, 0, size)
	for _, bufOffset := range backingMemoryPieces.Keys() {
		piece := backingMemoryPieces.Get(bufOffset)
		data, err := piece.DeviceMemory().Data().Slice(
			uint64(piece.MemoryOffset()),
			uint64(piece.MemoryOffset()+piece.Size())).Read(ctx, nil, s, nil)
		if err != nil {
			return nil, err
		}
		// This does not handle ranges with sparse residency, but it is assumed
		// that all buffer ranges referenced by a draw call will be backed by memory
		rawData = append(rawData, data...)
	}
	reader := endian.Reader(bytes.NewReader(rawData), s.MemoryLayout.GetEndian())
	return memory.NewDecoder(reader, s.MemoryLayout), nil
}

// Get the size in bytes for each index type
func getIndexSize(indexType VkIndexType) (uint32, error) {
	switch indexType {
	case VkIndexType_VK_INDEX_TYPE_UINT8_EXT:
		return 1, nil
	case VkIndexType_VK_INDEX_TYPE_UINT16:
		return 2, nil
	case VkIndexType_VK_INDEX_TYPE_UINT32:
		return 4, nil
	default:
		return 0, fmt.Errorf("Unhandled index type %v", indexType)
	}
}

// Helper functions to check if the next value in an index buffer is a primitive restart
func isRestartIndexU8(decoder *memory.Decoder) bool {
	return decoder.U8() == 0xFF
}
func isRestartIndexU16(decoder *memory.Decoder) bool {
	return decoder.U16() == 0xFFFF
}
func isRestartIndexU32(decoder *memory.Decoder) bool {
	return decoder.U32() == 0xFFFFFFFF
}

// Compute primitive count for a draw call
func getPrimitiveCount(ctx context.Context, s *api.GlobalState, isIndexedDraw bool, count uint32, first uint32) (uint32, error) {
	// Get the current draw state
	state := GetState(s)
	lastQueue := state.LastBoundQueue()
	if lastQueue.IsNil() {
		return 0, errors.New("Could not find current queue")
	}
	lastDrawInfo, ok := state.LastDrawInfos().Lookup(lastQueue.VulkanHandle())
	if !ok {
		return 0, errors.New("Could not find current draw info")
	}
	pipeline := lastDrawInfo.GraphicsPipeline()
	if pipeline.IsNil() {
		return 0, errors.New("Could not find current graphics pipeline")
	}

	// Number of vertices which define the first primitive and subsequent primitives
	firstPrimitiveVertices := uint32(0)
	nextPrimitiveVertices := uint32(0)
	switch pipeline.InputAssemblyState().Topology() {
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_LINE_LIST:
		firstPrimitiveVertices = 2
		nextPrimitiveVertices = 2
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_LINE_STRIP:
		firstPrimitiveVertices = 2
		nextPrimitiveVertices = 1
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST:
		firstPrimitiveVertices = 3
		nextPrimitiveVertices = 3
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_STRIP, VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_FAN:
		firstPrimitiveVertices = 3
		nextPrimitiveVertices = 1
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_LINE_LIST_WITH_ADJACENCY:
		firstPrimitiveVertices = 4
		nextPrimitiveVertices = 4
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_LINE_STRIP_WITH_ADJACENCY:
		firstPrimitiveVertices = 4
		nextPrimitiveVertices = 1
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST_WITH_ADJACENCY:
		firstPrimitiveVertices = 6
		nextPrimitiveVertices = 6
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_STRIP_WITH_ADJACENCY:
		firstPrimitiveVertices = 6
		nextPrimitiveVertices = 2
	default:
		return 0, fmt.Errorf("Unhandled primitive topology %v", pipeline.InputAssemblyState().Topology())
	}

	if isIndexedDraw && pipeline.InputAssemblyState().PrimitiveRestartEnable() == 1 {
		// Indexed draw with primitive restart enabled so we have to read the index buffer to count primitives
		indexBuffer := lastDrawInfo.BoundIndexBuffer()
		if indexBuffer.IsNil() {
			return 0, errors.New("Could not find bound index buffer")
		}
		indexSize, err := getIndexSize(indexBuffer.Type())
		if err != nil {
			return 0, err
		}
		decoder, err := getBufferDecoder(ctx, s, indexBuffer.BoundBuffer().Buffer(), VkDeviceSize(first*indexSize), VkDeviceSize(count*indexSize))
		if err != nil {
			return 0, err
		}

		// Magic restart value and index value getter
		var isRestartIndex func(*memory.Decoder) bool
		switch indexBuffer.Type() {
		case VkIndexType_VK_INDEX_TYPE_UINT8_EXT:
			isRestartIndex = isRestartIndexU8
		case VkIndexType_VK_INDEX_TYPE_UINT16:
			isRestartIndex = isRestartIndexU16
		case VkIndexType_VK_INDEX_TYPE_UINT32:
			isRestartIndex = isRestartIndexU32
		default:
			return 0, fmt.Errorf("Unhandled index type %v", indexBuffer.Type())
		}

		// Step through index buffer to count primitives
		primitiveCount := uint32(0)
		currentPrimitiveVertices := int(firstPrimitiveVertices)
		for i := uint32(0); i < count; i++ {
			if isRestartIndex(decoder) {
				// Primitive restart, reset count for first primitive
				currentPrimitiveVertices = int(firstPrimitiveVertices)
			} else {
				// Normal index value
				currentPrimitiveVertices--
				if currentPrimitiveVertices == 0 {
					// Primitive complete, reset count for next primitive
					primitiveCount++
					currentPrimitiveVertices = int(nextPrimitiveVertices)
				}
			}
		}
		return primitiveCount, nil
	} else {
		// No primitive restarts in this draw command, compute primitive count directly from vertex/index count
		if count < firstPrimitiveVertices {
			return 0, nil
		}
		return 1 + ((count - firstPrimitiveVertices) / nextPrimitiveVertices), nil
	}
}

// Compute vertex and primitive count for vkCmdDraw
func getDrawCounts(ctx context.Context, s *api.GlobalState, vertexCount uint32, instanceCount uint32, firstVertex uint32) (uint32, uint32, error) {
	primitiveCount, err := getPrimitiveCount(ctx, s, false, vertexCount, firstVertex)
	if err != nil {
		return 0, 0, err
	}
	return vertexCount * instanceCount, primitiveCount * instanceCount, nil
}

// Compute vertex and primitive count for vkCmdDrawIndexed
func getDrawIndexedCounts(ctx context.Context, s *api.GlobalState, indexCount uint32, instanceCount uint32, firstIndex uint32) (uint32, uint32, error) {
	primitiveCount, err := getPrimitiveCount(ctx, s, true, indexCount, firstIndex)
	if err != nil {
		return 0, 0, err
	}
	return indexCount * instanceCount, primitiveCount * instanceCount, nil
}

// Compute vertex and primitive count for vkCmdDrawIndirect
func getDrawIndirectCounts(ctx context.Context, s *api.GlobalState, buffer VkBuffer, offset VkDeviceSize, drawCount uint32, stride uint32) (uint32, uint32, error) {
	if drawCount == 0 {
		return 0, 0, nil
	}

	cmdSize := VkDeviceSize(VkDrawIndirectCommandSize(s.MemoryLayout))
	cmdStride := VkDeviceSize(stride)
	cmdTotalSize := cmdSize + (VkDeviceSize(drawCount-1) * cmdStride)
	decoder, err := getBufferDecoder(ctx, s, GetState(s).Buffers().Get(buffer), offset, cmdTotalSize)
	if err != nil {
		return 0, 0, err
	}

	vertexCount := uint32(0)
	primitiveCount := uint32(0)
	for i := uint32(0); i < drawCount; i++ {
		cmd := DecodeVkDrawIndirectCommand(decoder)
		v, p, err := getDrawCounts(ctx, s, cmd.VertexCount(), cmd.InstanceCount(), cmd.FirstVertex())
		if err != nil {
			return 0, 0, err
		}
		vertexCount += v
		primitiveCount += p
		if cmdStride > cmdSize {
			decoder.Skip(uint64(cmdStride - cmdSize))
		}
	}
	return vertexCount, primitiveCount, nil
}

// Compute vertex and primitive count for vkCmdDrawIndexedIndirect
func getDrawIndexedIndirectCounts(ctx context.Context, s *api.GlobalState, buffer VkBuffer, offset VkDeviceSize, drawCount uint32, stride uint32) (uint32, uint32, error) {
	if drawCount == 0 {
		return 0, 0, nil
	}

	cmdSize := VkDeviceSize(VkDrawIndexedIndirectCommandSize(s.MemoryLayout))
	cmdStride := VkDeviceSize(stride)
	cmdTotalSize := cmdSize + (VkDeviceSize(drawCount-1) * cmdStride)
	decoder, err := getBufferDecoder(ctx, s, GetState(s).Buffers().Get(buffer), offset, cmdTotalSize)
	if err != nil {
		return 0, 0, err
	}

	vertexCount := uint32(0)
	primitiveCount := uint32(0)
	for i := uint32(0); i < drawCount; i++ {
		cmd := DecodeVkDrawIndexedIndirectCommand(decoder)
		v, p, err := getDrawIndexedCounts(ctx, s, cmd.IndexCount(), cmd.InstanceCount(), cmd.FirstIndex())
		if err != nil {
			return 0, 0, err
		}
		vertexCount += v
		primitiveCount += p
		if cmdStride > cmdSize {
			decoder.Skip(uint64(cmdStride - cmdSize))
		}
	}
	return vertexCount, primitiveCount, nil
}

// Fetch draw count from indirect count buffer
func getIndirectCount(ctx context.Context, s *api.GlobalState, countBuffer VkBuffer, countOffset VkDeviceSize, maxDrawCount uint32) (uint32, error) {
	decoder, err := getBufferDecoder(ctx, s, GetState(s).Buffers().Get(countBuffer), countOffset, VkDeviceSize(s.MemoryLayout.I32.GetSize()))
	if err != nil {
		return 0, err
	}
	count := decoder.U32()
	if count > maxDrawCount {
		count = maxDrawCount
	}
	return count, nil
}

// ProfileStaticAnalysis computes the static analysis profiling data. It processes each command
// and for each submitted draw call command, it computes statistics to be shown as counters in
// PerfTab, combined with the hardware counters.
func (API) ProfileStaticAnalysis(ctx context.Context, p *path.Capture) (*api.StaticAnalysisProfileData, error) {
	data := profilingDataBuilder{
		data:         &api.StaticAnalysisProfileData{},
		seenCounters: map[counterType]struct{}{},
	}
	postSubCmdCb := func(s *api.GlobalState, idx api.SubCmdIdx, cmd api.Cmd, ref interface{}) {
		state := GetState(s)
		vertexCount := uint32(0)
		primitiveCount := uint32(0)
		isIndexedDraw := false
		cmdRef := ref.(CommandReferenceʳ)
		cmdArgs := GetCommandArgs(ctx, cmdRef, state)
		var err error
		switch args := cmdArgs.(type) {
		case VkCmdDrawArgsʳ:
			vertexCount, primitiveCount, err = getDrawCounts(ctx, s, args.VertexCount(), args.InstanceCount(), args.FirstVertex())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
		case VkCmdDrawIndexedArgsʳ:
			isIndexedDraw = true
			vertexCount, primitiveCount, err = getDrawIndexedCounts(ctx, s, args.IndexCount(), args.InstanceCount(), args.FirstIndex())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
		case VkCmdDrawIndirectArgsʳ:
			vertexCount, primitiveCount, err = getDrawIndirectCounts(ctx, s, args.Buffer(), args.Offset(), args.DrawCount(), args.Stride())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
		case VkCmdDrawIndexedIndirectArgsʳ:
			isIndexedDraw = true
			vertexCount, primitiveCount, err = getDrawIndexedIndirectCounts(ctx, s, args.Buffer(), args.Offset(), args.DrawCount(), args.Stride())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
		case VkCmdDrawIndirectCountKHRArgsʳ: // TODO VULKAN_1_2: Add core 1.2 VkCmdDrawIndirectCountArgsʳ
			drawCount, err := getIndirectCount(ctx, s, args.CountBuffer(), args.CountBufferOffset(), args.MaxDrawCount())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
			vertexCount, primitiveCount, err = getDrawIndirectCounts(ctx, s, args.Buffer(), args.Offset(), drawCount, args.Stride())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
		case VkCmdDrawIndirectCountAMDArgsʳ:
			drawCount, err := getIndirectCount(ctx, s, args.CountBuffer(), args.CountBufferOffset(), args.MaxDrawCount())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
			vertexCount, primitiveCount, err = getDrawIndirectCounts(ctx, s, args.Buffer(), args.Offset(), drawCount, args.Stride())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
		case VkCmdDrawIndexedIndirectCountKHRArgsʳ: // TODO VULKAN_1_2: Add core 1.2 VkCmdDrawIndexedIndirectCountArgsʳ
			isIndexedDraw = true
			drawCount, err := getIndirectCount(ctx, s, args.CountBuffer(), args.CountBufferOffset(), args.MaxDrawCount())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
			vertexCount, primitiveCount, err = getDrawIndexedIndirectCounts(ctx, s, args.Buffer(), args.Offset(), drawCount, args.Stride())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
		case VkCmdDrawIndexedIndirectCountAMDArgsʳ:
			isIndexedDraw = true
			drawCount, err := getIndirectCount(ctx, s, args.CountBuffer(), args.CountBufferOffset(), args.MaxDrawCount())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
			vertexCount, primitiveCount, err = getDrawIndexedIndirectCounts(ctx, s, args.Buffer(), args.Offset(), drawCount, args.Stride())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
		default:
			// Not a draw call.
			return
		}
		sampler := data.newSampler(idx)
		data.addSample(sampler, vertexCountType, float64(vertexCount))
		data.addSample(sampler, primitiveCountType, float64(primitiveCount))

		// Compute vertex size by summing all vertex attribute format sizes
		lastQueue := state.LastBoundQueue()
		if lastQueue.IsNil() {
			log.W(ctx, "Could not find current queue")
			return
		}
		lastDrawInfo, ok := state.LastDrawInfos().Lookup(lastQueue.VulkanHandle())
		if !ok {
			log.W(ctx, "Could not find current draw info")
			return
		}
		lastPipeline := lastDrawInfo.GraphicsPipeline()
		if lastPipeline.IsNil() {
			log.W(ctx, "Could not find current graphics pipeline")
			return
		}
		vertexSize := uint32(0)
		for _, attr := range lastPipeline.VertexInputState().AttributeDescriptions().All() {
			elementAndTexelBlockSize, err := subGetElementAndTexelBlockSize(ctx, nil, api.CmdNoID, nil, s, nil, 0, nil, nil, attr.Fmt())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
			vertexSize += elementAndTexelBlockSize.ElementSize()
		}
		data.addSample(sampler, vertexSizeType, float64(vertexSize))

		// Compute total bytes read through index and vertex buffers
		byteCount := vertexCount * vertexSize
		if isIndexedDraw {
			// Add index bytes for indexed draw calls
			indexBuffer := lastDrawInfo.BoundIndexBuffer()
			if indexBuffer.IsNil() {
				log.W(ctx, "Could not find bound index buffer")
				return
			}
			indexSize, err := getIndexSize(indexBuffer.Type())
			if err != nil {
				log.W(ctx, err.Error())
				return
			}
			byteCount += vertexCount * indexSize
		}
		data.addSample(sampler, byteCountType, float64(byteCount))

		// Compute bytes/primitive
		data.addSample(sampler, primitiveSizeType, float64(byteCount)/float64(primitiveCount))
	}

	if err := sync.MutateWithSubcommands(ctx, p, nil, nil, nil, postSubCmdCb); err != nil {
		return nil, err
	}
	return data.data, nil
}
