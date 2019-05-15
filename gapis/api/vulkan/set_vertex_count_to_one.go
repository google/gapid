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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

// sets the number of vertices to be drawn to 1
func setVertexCountToOne(ctx context.Context) transform.Transformer {
	ctx = log.Enter(ctx, "setVertexCountToOne")

	const vertexCount = 1
	const instanceCount = 1
	const indexCount = 1

	return transform.Transform("setVertexCountToOne", func(ctx context.Context,
		id api.CmdID, cmd api.Cmd, out transform.Writer) {

		s := out.State()
		cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}
		var newCmd api.Cmd
		isIndirectDraw := false
		var cmdBuf VkCommandBuffer
		switch cmd := cmd.(type) {
		case *VkCmdDraw:
			newCmd = cb.VkCmdDraw(
				cmd.commandBuffer,
				vertexCount,
				instanceCount,
				cmd.FirstVertex(),
				cmd.FirstInstance(),
			)
		case *VkCmdDrawIndexed:
			newCmd = cb.VkCmdDrawIndexed(
				cmd.commandBuffer,
				indexCount,
				instanceCount,
				cmd.FirstIndex(),
				cmd.VertexOffset(),
				cmd.FirstInstance(),
			)
		case *VkCmdDrawIndirect:
			isIndirectDraw = true
			cmdBuf = cmd.commandBuffer
		case *VkCmdDrawIndexedIndirect:
			isIndirectDraw = true
			cmdBuf = cmd.commandBuffer
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
			out.MutateAndWrite(ctx, id, cmd)
			return
		}

		// Replace indirect draw calls with direct ones.
		// TODO: Replace with appropriate indirect calls instead.
		if isIndirectDraw {
			newCmd = cb.VkCmdDraw(
				cmdBuf,
				vertexCount,
				instanceCount,
				/* first vertex */ 0,
				/* first instance */ 0,
			)
		}

		out.MutateAndWrite(ctx, id, newCmd)
	})
}
