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

	"github.com/google/gapid/gapis/api"
	rb "github.com/google/gapid/gapis/replay/builder"
)

func (s *State) SetupInitialState() {
	for _, o := range s.CommandBuffers {
		for i := uint32(0); i < uint32(len(o.CommandReferences)); i++ {
			r := o.CommandReferences[i]
			switch r.Type {
			case CommandType_cmd_vkCmdBeginRenderPass:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdBeginRenderPass(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdBeginRenderPass[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdEndRenderPass:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdEndRenderPass(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdEndRenderPass[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdNextSubpass:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdNextSubpass(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdNextSubpass[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdBindPipeline:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdBindPipeline(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdBindPipeline[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdBindDescriptorSets:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdBindDescriptorSets(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdBindDescriptorSets[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdBindVertexBuffers:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdBindVertexBuffers(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdBindVertexBuffers[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdBindIndexBuffer:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdBindIndexBuffer(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdBindIndexBuffer[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdPipelineBarrier:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdPipelineBarrier(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdPipelineBarrier[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdWaitEvents:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdWaitEvents(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdWaitEvents[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdBeginQuery:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdBeginQuery(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdBeginQuery[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdBlitImage:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdBlitImage(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdBlitImage[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdClearAttachments:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdClearAttachments(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdClearAttachments[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdClearColorImage:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdClearColorImage(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdClearColorImage[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdClearDepthStencilImage:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdClearDepthStencilImage(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdClearDepthStencilImage[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdCopyBuffer:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdCopyBuffer(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdCopyBuffer[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdCopyBufferToImage:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdCopyBufferToImage(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdCopyBufferToImage[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdCopyImage:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdCopyImage(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdCopyImage[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdCopyImageToBuffer:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdCopyImageToBuffer(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdCopyImageToBuffer[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdCopyQueryPoolResults:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdCopyQueryPoolResults(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdCopyQueryPoolResults[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdDispatch:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdDispatch(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdDispatch[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdDispatchIndirect:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdDispatchIndirect(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdDispatchIndirect[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdDraw:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdDraw(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdDraw[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdDrawIndexed:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdDrawIndexed(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdDrawIndexed[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdDrawIndexedIndirect:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdDrawIndexedIndirect(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdDrawIndexedIndirect[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdDrawIndirect:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdDrawIndirect(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdDrawIndirect[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdEndQuery:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdEndQuery(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdEndQuery[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdExecuteCommands:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdExecuteCommands(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdExecuteCommands[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdFillBuffer:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdFillBuffer(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdFillBuffer[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdPushConstants:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdPushConstants(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdPushConstants[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdResetQueryPool:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdResetQueryPool(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdResetQueryPool[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdResolveImage:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdResolveImage(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdResolveImage[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdSetBlendConstants:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdSetBlendConstants(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdSetBlendConstants[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdSetDepthBias:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdSetDepthBias(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdSetDepthBias[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdSetDepthBounds:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdSetDepthBounds(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdSetDepthBounds[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdSetEvent:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdSetEvent(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdSetEvent[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdResetEvent:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdResetEvent(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdResetEvent[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdSetLineWidth:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdSetLineWidth(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdSetLineWidth[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdSetScissor:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdSetScissor(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdSetScissor[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdSetStencilCompareMask:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdSetStencilCompareMask(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdSetStencilCompareMask[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdSetStencilReference:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdSetStencilReference(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdSetStencilReference[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdSetStencilWriteMask:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdSetStencilWriteMask(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdSetStencilWriteMask[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdSetViewport:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdSetViewport(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdSetViewport[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdUpdateBuffer:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdUpdateBuffer(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdUpdateBuffer[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdWriteTimestamp:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdWriteTimestamp(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdWriteTimestamp[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdDebugMarkerBeginEXT:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdDebugMarkerBeginEXT(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdDebugMarkerBeginEXT[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdDebugMarkerEndEXT:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdDebugMarkerEndEXT(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdDebugMarkerEndEXT[r.MapIndex])
					}})
			case CommandType_cmd_vkCmdDebugMarkerInsertEXT:
				o.Commands = append(o.Commands, CommandBufferCommand{
					func(ctx context.Context, cmd api.Cmd, id api.CmdID, gs *api.GlobalState, b *rb.Builder) {
						subDovkCmdDebugMarkerInsertEXT(ctx, cmd, id, &api.CmdObservations{}, gs, s, cmd.Thread(), b,
							s.CommandBuffers[r.Buffer].BufferCommands.VkCmdDebugMarkerInsertEXT[r.MapIndex])
					}})
			}
			if s.AddCommand != nil {
				s.AddCommand(o.CommandReferences[uint32(len(o.Commands)-1)])
			}
		}
	}
}
