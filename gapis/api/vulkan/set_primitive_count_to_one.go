// Copyright (C) 2019 Google Inc.
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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/u32"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

func getMinimumVertexCountForPrimitive(ctx context.Context, primitive VkPrimitiveTopology) uint32 {
	switch primitive {
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_POINT_LIST:
		return 1
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_LINE_LIST, VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_LINE_STRIP:
		return 2
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST, VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_STRIP, VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_FAN:
		return 3
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_LINE_LIST_WITH_ADJACENCY, VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_LINE_STRIP_WITH_ADJACENCY:
		return 4
	case VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST_WITH_ADJACENCY, VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_STRIP_WITH_ADJACENCY:
		return 6
	default: // VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_PATCH_LIST,
		log.W(ctx, "getMinimumVertexCountForPrimitive() not implemented for %v", primitive)
		return 0
	}
}

// setPrimitiveCountToOne returns a transform that sets the maximum
// number of primtives per draw call to 1
func setPrimitiveCountToOne(ctx context.Context) transform.Transformer {
	ctx = log.Enter(ctx, "setPrimitiveCountToOne")

	const instanceCount uint32 = 1

	perPipelineVertexCount := make(map[VkPipeline]uint32)
	cmdPipelineBindings := make(map[VkCommandBuffer]VkPipeline)

	return transform.Transform("setPrimitiveCountToOne", func(ctx context.Context,
		id api.CmdID, cmd api.Cmd, out transform.Writer) error {

		s := out.State()
		l := s.MemoryLayout
		cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}

		isIndirectDraw := false
		var cmdBuf VkCommandBuffer
		cmdInstanceCount := instanceCount

		switch cmd := cmd.(type) {
		case *VkCreateGraphicsPipelines:
			if err := out.MutateAndWrite(ctx, id, cmd); err != nil {
				return err
			}
			cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())

			createInfoCount := uint64(cmd.CreateInfoCount())
			createInfos := cmd.PCreateInfos().Slice(0, createInfoCount, l).MustRead(ctx, cmd, s, nil)
			pipelineHandles := cmd.PPipelines().Slice(0, createInfoCount, l).MustRead(ctx, cmd, s, nil)

			for i := uint64(0); i < createInfoCount; i++ {
				primitive := createInfos[i].PInputAssemblyState().MustRead(ctx, cmd, s, nil).Topology()
				perPipelineVertexCount[pipelineHandles[i]] = getMinimumVertexCountForPrimitive(ctx, primitive)
			}
		case *VkCmdBindPipeline:
			if err := out.MutateAndWrite(ctx, id, cmd); err != nil {
				return err
			}
			if cmd.PipelineBindPoint() == VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS {
				cmdPipelineBindings[cmd.commandBuffer] = cmd.Pipeline()
			}
		case *VkCmdDraw:
			vertexCount := perPipelineVertexCount[cmdPipelineBindings[cmd.commandBuffer]]
			newCmd := cb.VkCmdDraw(
				cmd.commandBuffer,
				u32.Min(vertexCount, cmd.VertexCount()),
				u32.Min(instanceCount, cmd.InstanceCount()),
				cmd.FirstVertex(),
				cmd.FirstInstance(),
			)
			if err := out.MutateAndWrite(ctx, id, newCmd); err != nil {
				return err
			}
		case *VkCmdDrawIndexed:
			vertexCount := perPipelineVertexCount[cmdPipelineBindings[cmd.commandBuffer]]
			newCmd := cb.VkCmdDrawIndexed(
				cmd.commandBuffer,
				u32.Min(vertexCount, cmd.IndexCount()),
				u32.Min(instanceCount, cmd.InstanceCount()),
				cmd.FirstIndex(),
				cmd.VertexOffset(),
				cmd.FirstInstance(),
			)
			if err := out.MutateAndWrite(ctx, id, newCmd); err != nil {
				return err
			}
		case *VkCmdDrawIndirect:
			isIndirectDraw = true
			cmdBuf = cmd.commandBuffer
			cmdInstanceCount = cmd.DrawCount()
		case *VkCmdDrawIndexedIndirect:
			isIndirectDraw = true
			cmdBuf = cmd.commandBuffer
			cmdInstanceCount = cmd.DrawCount()
		case *VkCmdDrawIndirectCountKHR:
			isIndirectDraw = true
			cmdBuf = cmd.commandBuffer
		case *VkCmdDrawIndexedIndirectCountKHR:
			isIndirectDraw = true
			cmdBuf = cmd.commandBuffer
		case *VkCmdDrawIndirectCountAMD:
			isIndirectDraw = true
			cmdBuf = cmd.commandBuffer
		case *VkCmdDrawIndexedIndirectCountAMD:
			isIndirectDraw = true
			cmdBuf = cmd.commandBuffer
		default:
			return out.MutateAndWrite(ctx, id, cmd)
		}

		// Replace indirect draw calls with direct ones.
		// TODO: Replace with appropriate indirect calls instead.
		if isIndirectDraw {
			newCmd := cb.VkCmdDraw(
				cmdBuf,
				0, // vertex count
				u32.Min(instanceCount, cmdInstanceCount),
				0, // first vertex
				0, // first instance
			)
			return out.MutateAndWrite(ctx, id, newCmd)
		}
		return nil
	})
}
