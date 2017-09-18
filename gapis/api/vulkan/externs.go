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
	"reflect"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
	rb "github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/protocol"
)

type externs struct {
	ctx   context.Context // Allowed because the externs struct is only a parameter proxy for a single call
	cmd   api.Cmd
	cmdID api.CmdID
	s     *api.GlobalState
	b     *rb.Builder
}

func (e externs) hasDynamicProperty(info VkPipelineDynamicStateCreateInfoᶜᵖ,
	state VkDynamicState) bool {
	if (info) == (VkPipelineDynamicStateCreateInfoᶜᵖ{}) {
		return false
	}
	l := e.s.MemoryLayout
	dynamic_state_info := info.Slice(uint64(0), uint64(1), l).Index(uint64(0), l).Read(e.ctx, e.cmd, e.s, e.b)
	states := dynamic_state_info.PDynamicStates.Slice(uint64(uint32(0)), uint64(dynamic_state_info.DynamicStateCount), l).Read(e.ctx, e.cmd, e.s, e.b)
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

func (e externs) mapMemory(value Voidᵖᵖ, slice memory.Slice) {
	ctx := e.ctx
	if b := e.b; b != nil {
		switch e.cmd.(type) {
		case *VkMapMemory:
			b.Load(protocol.Type_AbsolutePointer, value.value(e.b, e.cmd, e.s))
			b.MapMemory(slice.Range(e.s.MemoryLayout))
		default:
			log.E(ctx, "mapBuffer extern called for unsupported command: %v", e.cmd)
		}
	}
}

func (e externs) callSub(ctx context.Context, cmd api.Cmd, id api.CmdID, s *api.GlobalState, b *rb.Builder, sub, data interface{}) {
	reflect.ValueOf(sub).Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(cmd),
		reflect.ValueOf(id),
		reflect.ValueOf(&api.CmdObservations{}),
		reflect.ValueOf(s),
		reflect.ValueOf(GetState(s)),
		reflect.ValueOf(cmd.Thread()),
		reflect.ValueOf(b),
		reflect.ValueOf(data),
	})
}

func (e externs) addCmd(commandBuffer VkCommandBuffer, data interface{}, functionToCall interface{}) {
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)

	switch d := data.(type) {
	case (*VkCmdBindPipelineArgs):
		o.BufferCommands.VkCmdBindPipeline[uint32(len(o.BufferCommands.VkCmdBindPipeline))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBindPipeline,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdBindPipeline) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdSetViewportArgs):
		o.BufferCommands.VkCmdSetViewport[uint32(len(o.BufferCommands.VkCmdSetViewport))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetViewport,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdSetViewport) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdSetScissorArgs):
		o.BufferCommands.VkCmdSetScissor[uint32(len(o.BufferCommands.VkCmdSetScissor))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetScissor,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdSetScissor) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdSetLineWidthArgs):
		o.BufferCommands.VkCmdSetLineWidth[uint32(len(o.BufferCommands.VkCmdSetLineWidth))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetLineWidth,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdSetLineWidth) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdSetDepthBiasArgs):
		o.BufferCommands.VkCmdSetDepthBias[uint32(len(o.BufferCommands.VkCmdSetDepthBias))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetDepthBias,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdSetDepthBias) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdSetBlendConstantsArgs):
		o.BufferCommands.VkCmdSetBlendConstants[uint32(len(o.BufferCommands.VkCmdSetBlendConstants))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetBlendConstants,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdSetBlendConstants) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdSetDepthBoundsArgs):
		o.BufferCommands.VkCmdSetDepthBounds[uint32(len(o.BufferCommands.VkCmdSetDepthBounds))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetDepthBounds,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdSetDepthBounds) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdSetStencilCompareMaskArgs):
		o.BufferCommands.VkCmdSetStencilCompareMask[uint32(len(o.BufferCommands.VkCmdSetStencilCompareMask))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetStencilCompareMask,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdSetStencilCompareMask) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdSetStencilWriteMaskArgs):
		o.BufferCommands.VkCmdSetStencilWriteMask[uint32(len(o.BufferCommands.VkCmdSetStencilWriteMask))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetStencilWriteMask,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdSetStencilWriteMask) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdSetStencilReferenceArgs):
		o.BufferCommands.VkCmdSetStencilReference[uint32(len(o.BufferCommands.VkCmdSetStencilReference))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetStencilReference,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdSetStencilReference) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdBindDescriptorSetsArgs):
		o.BufferCommands.VkCmdBindDescriptorSets[uint32(len(o.BufferCommands.VkCmdBindDescriptorSets))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBindDescriptorSets,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdBindDescriptorSets) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdBindIndexBufferArgs):
		o.BufferCommands.VkCmdBindIndexBuffer[uint32(len(o.BufferCommands.VkCmdBindIndexBuffer))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBindIndexBuffer,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdBindIndexBuffer) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdBindVertexBuffersArgs):
		o.BufferCommands.VkCmdBindVertexBuffers[uint32(len(o.BufferCommands.VkCmdBindVertexBuffers))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBindVertexBuffers,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdBindVertexBuffers) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdDrawArgs):
		o.BufferCommands.VkCmdDraw[uint32(len(o.BufferCommands.VkCmdDraw))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDraw,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdDraw) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdDrawIndexedArgs):
		o.BufferCommands.VkCmdDrawIndexed[uint32(len(o.BufferCommands.VkCmdDrawIndexed))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDrawIndexed,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdDrawIndexed) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdDrawIndirectArgs):
		o.BufferCommands.VkCmdDrawIndirect[uint32(len(o.BufferCommands.VkCmdDrawIndirect))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDrawIndirect,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdDrawIndirect) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdDrawIndexedIndirectArgs):
		o.BufferCommands.VkCmdDrawIndexedIndirect[uint32(len(o.BufferCommands.VkCmdDrawIndexedIndirect))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDrawIndexedIndirect,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdDrawIndexedIndirect) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdDispatchArgs):
		o.BufferCommands.VkCmdDispatch[uint32(len(o.BufferCommands.VkCmdDispatch))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDispatch,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdDispatch) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdDispatchIndirectArgs):
		o.BufferCommands.VkCmdDispatchIndirect[uint32(len(o.BufferCommands.VkCmdDispatchIndirect))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDispatchIndirect,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdDispatchIndirect) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdCopyBufferArgs):
		o.BufferCommands.VkCmdCopyBuffer[uint32(len(o.BufferCommands.VkCmdCopyBuffer))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdCopyBuffer,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdCopyBuffer) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdCopyImageArgs):
		o.BufferCommands.VkCmdCopyImage[uint32(len(o.BufferCommands.VkCmdCopyImage))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdCopyImage,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdCopyImage) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdBlitImageArgs):
		o.BufferCommands.VkCmdBlitImage[uint32(len(o.BufferCommands.VkCmdBlitImage))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBlitImage,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdBlitImage) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdCopyBufferToImageArgs):
		o.BufferCommands.VkCmdCopyBufferToImage[uint32(len(o.BufferCommands.VkCmdCopyBufferToImage))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdCopyBufferToImage,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdCopyBufferToImage) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdCopyImageToBufferArgs):
		o.BufferCommands.VkCmdCopyImageToBuffer[uint32(len(o.BufferCommands.VkCmdCopyImageToBuffer))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdCopyImageToBuffer,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdCopyImageToBuffer) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdUpdateBufferArgs):
		o.BufferCommands.VkCmdUpdateBuffer[uint32(len(o.BufferCommands.VkCmdUpdateBuffer))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdUpdateBuffer,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdUpdateBuffer) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdFillBufferArgs):
		o.BufferCommands.VkCmdFillBuffer[uint32(len(o.BufferCommands.VkCmdFillBuffer))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdFillBuffer,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdFillBuffer) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdClearColorImageArgs):
		o.BufferCommands.VkCmdClearColorImage[uint32(len(o.BufferCommands.VkCmdClearColorImage))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdClearColorImage,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdClearColorImage) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdClearDepthStencilImageArgs):
		o.BufferCommands.VkCmdClearDepthStencilImage[uint32(len(o.BufferCommands.VkCmdClearDepthStencilImage))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdClearDepthStencilImage,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdClearDepthStencilImage) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdClearAttachmentsArgs):
		o.BufferCommands.VkCmdClearAttachments[uint32(len(o.BufferCommands.VkCmdClearAttachments))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdClearAttachments,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdClearAttachments) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdResolveImageArgs):
		o.BufferCommands.VkCmdResolveImage[uint32(len(o.BufferCommands.VkCmdResolveImage))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdResolveImage,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdResolveImage) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdSetEventArgs):
		o.BufferCommands.VkCmdSetEvent[uint32(len(o.BufferCommands.VkCmdSetEvent))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetEvent,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdSetEvent) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdResetEventArgs):
		o.BufferCommands.VkCmdResetEvent[uint32(len(o.BufferCommands.VkCmdResetEvent))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdResetEvent,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdResetEvent) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdWaitEventsArgs):
		o.BufferCommands.VkCmdWaitEvents[uint32(len(o.BufferCommands.VkCmdWaitEvents))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdWaitEvents,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdWaitEvents) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdPipelineBarrierArgs):
		o.BufferCommands.VkCmdPipelineBarrier[uint32(len(o.BufferCommands.VkCmdPipelineBarrier))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdPipelineBarrier,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdPipelineBarrier) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdBeginQueryArgs):
		o.BufferCommands.VkCmdBeginQuery[uint32(len(o.BufferCommands.VkCmdBeginQuery))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBeginQuery,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdBeginQuery) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdEndQueryArgs):
		o.BufferCommands.VkCmdEndQuery[uint32(len(o.BufferCommands.VkCmdEndQuery))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdEndQuery,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdEndQuery) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdResetQueryPoolArgs):
		o.BufferCommands.VkCmdResetQueryPool[uint32(len(o.BufferCommands.VkCmdResetQueryPool))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdResetQueryPool,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdResetQueryPool) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdWriteTimestampArgs):
		o.BufferCommands.VkCmdWriteTimestamp[uint32(len(o.BufferCommands.VkCmdWriteTimestamp))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdWriteTimestamp,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdWriteTimestamp) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdCopyQueryPoolResultsArgs):
		o.BufferCommands.VkCmdCopyQueryPoolResults[uint32(len(o.BufferCommands.VkCmdCopyQueryPoolResults))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdCopyQueryPoolResults,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdCopyQueryPoolResults) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdPushConstantsArgs):
		o.BufferCommands.VkCmdPushConstants[uint32(len(o.BufferCommands.VkCmdPushConstants))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdPushConstants,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdPushConstants) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdBeginRenderPassArgs):
		o.BufferCommands.VkCmdBeginRenderPass[uint32(len(o.BufferCommands.VkCmdBeginRenderPass))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBeginRenderPass,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdBeginRenderPass) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdNextSubpassArgs):
		o.BufferCommands.VkCmdNextSubpass[uint32(len(o.BufferCommands.VkCmdNextSubpass))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdNextSubpass,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdNextSubpass) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdEndRenderPassArgs):
		o.BufferCommands.VkCmdEndRenderPass[uint32(len(o.BufferCommands.VkCmdEndRenderPass))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdEndRenderPass,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdEndRenderPass) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdExecuteCommandsArgs):
		o.BufferCommands.VkCmdExecuteCommands[uint32(len(o.BufferCommands.VkCmdExecuteCommands))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdExecuteCommands,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdExecuteCommands) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdDebugMarkerBeginEXTArgs):
		o.BufferCommands.VkCmdDebugMarkerBeginEXT[uint32(len(o.BufferCommands.VkCmdDebugMarkerBeginEXT))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDebugMarkerBeginEXT,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdDebugMarkerBeginEXT) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdDebugMarkerEndEXTArgs):
		o.BufferCommands.VkCmdDebugMarkerEndEXT[uint32(len(o.BufferCommands.VkCmdDebugMarkerEndEXT))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDebugMarkerEndEXT,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdDebugMarkerEndEXT) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	case (*VkCmdDebugMarkerInsertEXTArgs):
		o.BufferCommands.VkCmdDebugMarkerInsertEXT[uint32(len(o.BufferCommands.VkCmdDebugMarkerInsertEXT))] = d
		o.CommandReferences[uint32(len(o.CommandReferences))] = CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDebugMarkerInsertEXT,
			MapIndex:        uint32(len(o.BufferCommands.VkCmdDebugMarkerInsertEXT) - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      &e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		}
	default:
	}

	o.Commands = append(o.Commands, CommandBufferCommand{
		func(ctx context.Context, cmd api.Cmd, id api.CmdID, s *api.GlobalState, b *rb.Builder) {
			e.callSub(ctx, cmd, id, s, b, functionToCall, data)
		}})

	if GetState(e.s).AddCommand != nil {
		GetState(e.s).AddCommand(o.CommandReferences[uint32(len(o.Commands)-1)])
	}

}

func (e externs) resetCmd(commandBuffer VkCommandBuffer) {
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)
	o.Commands = []CommandBufferCommand{}
	o.CommandReferences = U32ːCommandReferenceᵐ{}
}

func (e externs) notifyPendingCommandAdded(queue VkQueue) {
	s := GetState(e.s)
	queueObject := s.Queues[queue]
	command := queueObject.PendingCommands[uint32(len(queueObject.PendingCommands)-1)]
	s.SubCmdIdx[len(s.SubCmdIdx)-1] = uint64(command.CommandIndex)
	command.QueuedCommandData.submit = &e.cmd
	command.QueuedCommandData.submissionIndex = append([]uint64(nil), s.SubCmdIdx...)
	command.QueuedCommandData.actualSubmission = true
	queueObject.PendingCommands[uint32(len(queueObject.PendingCommands)-1)] = command
}

func (e externs) enterSubcontext() {
	o := GetState(e.s)
	o.SubCmdIdx = append(o.SubCmdIdx, 0)
}

func (e externs) leaveSubcontext() {
	o := GetState(e.s)
	o.SubCmdIdx = o.SubCmdIdx[:len(o.SubCmdIdx)-1]
}

func (e externs) nextSubcontext() {
	o := GetState(e.s)
	o.SubCmdIdx[len(o.SubCmdIdx)-1] += 1
}

func (e externs) execPendingCommands(queue VkQueue) {
	o := GetState(e.s)
	// Set the global LastBoundQueue, so the next vkCmdWaitEvent in the pending
	// commands knows in which queue it will be waiting.
	GetState(e.s).LastBoundQueue = GetState(e.s).Queues.Get(queue)
	lastBoundQueue := GetState(e.s).LastBoundQueue
	newPendingCommands := U32ːCommandReferenceᵐ{}

	// Store off state.IdxList (Should be empty)
	for _, i := range lastBoundQueue.PendingCommands.KeysSorted() {
		command := lastBoundQueue.PendingCommands[i]
		// Set the state.IdxList to command.Indices
		// Set the state.Queue to command.Queue

		// lastBoundQueue.PendingEvents will be 0 the first time
		// through. (ExecPending could not have been called otherwise).
		// Therefore o.CurrentSubmission will be set by the else
		// branch at least once.
		if len(lastBoundQueue.PendingEvents) != 0 {
			newPendingCommands[uint32(len(newPendingCommands))] = command
		} else {
			o.SubCmdIdx = append([]uint64{}, command.QueuedCommandData.submissionIndex...)
			if command.SemaphoreUpdate == SemaphoreUpdate_Signal {
				o.Semaphores[command.Semaphore].Signaled = true
			}
			if command.SemaphoreUpdate == SemaphoreUpdate_Unsignal {
				o.Semaphores[command.Semaphore].Signaled = false
			}
			if command.Buffer == VkCommandBuffer(0) {
				continue
			}

			o.CurrentSubmission = command.QueuedCommandData.submit
			if command.QueuedCommandData.actualSubmission && o.PreSubcommand != nil {
				o.PreSubcommand(command)
			}
			buffer := o.CommandBuffers[command.Buffer]
			buffer.Commands[command.CommandIndex].function(e.ctx, e.cmd, e.cmdID, e.s, e.b)
			if command.QueuedCommandData.actualSubmission && o.PostSubcommand != nil {
				o.PostSubcommand(command)
			}
			// If a vkCmdWaitEvent is hit in the pending commands, it will set a new
			// list of pending events to the LastBoundQueue. Once that happens, we
			// should start a new pending command list.
			if len(lastBoundQueue.PendingEvents) != 0 {
				c := command
				c.QueuedCommandData.actualSubmission = false
				c.QueuedCommandData.submit = o.CurrentSubmission
				newPendingCommands[uint32(len(newPendingCommands))] = c
			}
		}
	}
	o.SubCmdIdx = []uint64(nil)
	// Reset state.IdxList
	// Refresh or clear the pending commands in LastBoundQueue
	lastBoundQueue.PendingCommands = newPendingCommands
}

func (e externs) addWords(module VkShaderModule, numBytes interface{}, data interface{}) {
}

func (e externs) addDebugMarkerTagBytes(*VulkanDebugMarkerInfo, interface{}, interface{}) {}

func (e externs) setSpecData(module *SpecializationInfo, numBytes interface{}, data interface{}) {
}

func (e externs) unmapMemory(slice memory.Slice) {
	if b := e.b; b != nil {
		b.UnmapMemory(slice.Range(e.s.MemoryLayout))
	}
}

func (e externs) trackMappedCoherentMemory(start uint64, size memory.Size) {}
func (e externs) readMappedCoherentMemory(memory_handle VkDeviceMemory, offset_in_mapped uint64, read_size memory.Size) {
	l := e.s.MemoryLayout
	mem := GetState(e.s).DeviceMemories.Get(memory_handle)
	mapped_offset := uint64(mem.MappedOffset)
	dstStart := mapped_offset + offset_in_mapped
	srcStart := offset_in_mapped

	absSrcStart := mem.MappedLocation.Address() + offset_in_mapped
	absSrcMemRng := memory.Range{Base: absSrcStart, Size: uint64(read_size)}

	writeRngList := e.s.Memory.ApplicationPool().Slice(absSrcMemRng).ValidRanges()
	for _, r := range writeRngList {
		mem.Data.Slice(dstStart+r.Base, dstStart+r.Base+r.Size, l).Copy(
			e.ctx, U8ᵖ(mem.MappedLocation).Slice(srcStart+r.Base, srcStart+r.Base+r.Size, l), e.cmd, e.s, e.b)
	}
}
func (e externs) untrackMappedCoherentMemory(start uint64, size memory.Size) {}

func (e externs) numberOfPNext(pNext Voidᶜᵖ) uint32 {
	l := e.s.MemoryLayout
	counter := uint32(0)
	for (pNext) != (Voidᶜᵖ{}) {
		counter++
		pNext = Voidᶜᵖᵖ(pNext).Slice(uint64(0), uint64(2), l).Index(uint64(1), l).Read(e.ctx, e.cmd, e.s, e.b)
	}
	return counter
}

func (e externs) pushDebugMarker(name string) {
	if GetState(e.s).pushMarkerGroup != nil {
		GetState(e.s).pushMarkerGroup(name, false, DebugMarker)
	}
}

func (e externs) popDebugMarker() {
	if GetState(e.s).popMarkerGroup != nil {
		GetState(e.s).popMarkerGroup(DebugMarker)
	}
}

func (e externs) pushRenderPassMarker(rp VkRenderPass) {
	if GetState(e.s).pushMarkerGroup != nil {
		rpObj := GetState(e.s).RenderPasses.Get(rp)
		var name string
		if rpObj.DebugInfo != nil && len(rpObj.DebugInfo.ObjectName) > 0 {
			name = rpObj.DebugInfo.ObjectName
		} else {
			name = fmt.Sprintf("RenderPass: %v", rp)
		}
		GetState(e.s).pushMarkerGroup(name, false, RenderPassMarker)
		if len(rpObj.SubpassDescriptions) > 1 {
			GetState(e.s).pushMarkerGroup("Subpass: 0", false, RenderPassMarker)
		}
	}
}

func (e externs) popRenderPassMarker() {
	if GetState(e.s).popMarkerGroup != nil {
		GetState(e.s).popMarkerGroup(RenderPassMarker)
	}
}

func (e externs) popAndPushMarkerForNextSubpass(nextSubpass uint32) {
	if GetState(e.s).popMarkerGroup != nil {
		GetState(e.s).popMarkerGroup(RenderPassMarker)
	}
	name := fmt.Sprintf("Subpass: %v", nextSubpass)
	if GetState(e.s).pushMarkerGroup != nil {
		GetState(e.s).pushMarkerGroup(name, true, RenderPassMarker)
	}
}
