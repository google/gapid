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
	"github.com/google/gapid/gapis/replay"
	rb "github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/protocol"
	"github.com/google/gapid/gapis/service"
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
	dynamic_state_info := info.Slice(uint64(0), uint64(1), l).Index(uint64(0), l).MustRead(e.ctx, e.cmd, e.s, e.b)
	states := dynamic_state_info.PDynamicStates.Slice(uint64(uint32(0)), uint64(dynamic_state_info.DynamicStateCount), l).MustRead(e.ctx, e.cmd, e.s, e.b)
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

// CallReflectedCommand unpacks the given subcommand and arguments, and calls the method
func CallReflectedCommand(ctx context.Context, cmd api.Cmd, id api.CmdID, s *api.GlobalState, b *rb.Builder, sub, data interface{}) {
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

func (e externs) callSub(ctx context.Context, cmd api.Cmd, id api.CmdID, s *api.GlobalState, b *rb.Builder, sub, data interface{}) {
	CallReflectedCommand(ctx, cmd, id, s, b, sub, data)
}

func (e externs) addCmd(commandBuffer VkCommandBuffer, data interface{}, functionToCall interface{}) {
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)

	switch d := data.(type) {
	case (*VkCmdBindPipelineArgs):
		o.BufferCommands.VkCmdBindPipeline.Set(uint32(o.BufferCommands.VkCmdBindPipeline.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBindPipeline,
			MapIndex:        uint32(o.BufferCommands.VkCmdBindPipeline.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdSetViewportArgs):
		o.BufferCommands.VkCmdSetViewport.Set(uint32(o.BufferCommands.VkCmdSetViewport.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetViewport,
			MapIndex:        uint32(o.BufferCommands.VkCmdSetViewport.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdSetScissorArgs):
		o.BufferCommands.VkCmdSetScissor.Set(uint32(o.BufferCommands.VkCmdSetScissor.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetScissor,
			MapIndex:        uint32(o.BufferCommands.VkCmdSetScissor.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdSetLineWidthArgs):
		o.BufferCommands.VkCmdSetLineWidth.Set(uint32(o.BufferCommands.VkCmdSetLineWidth.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetLineWidth,
			MapIndex:        uint32(o.BufferCommands.VkCmdSetLineWidth.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdSetDepthBiasArgs):
		o.BufferCommands.VkCmdSetDepthBias.Set(uint32(o.BufferCommands.VkCmdSetDepthBias.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetDepthBias,
			MapIndex:        uint32(o.BufferCommands.VkCmdSetDepthBias.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdSetBlendConstantsArgs):
		o.BufferCommands.VkCmdSetBlendConstants.Set(uint32(o.BufferCommands.VkCmdSetBlendConstants.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetBlendConstants,
			MapIndex:        uint32(o.BufferCommands.VkCmdSetBlendConstants.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdSetDepthBoundsArgs):
		o.BufferCommands.VkCmdSetDepthBounds.Set(uint32(o.BufferCommands.VkCmdSetDepthBounds.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetDepthBounds,
			MapIndex:        uint32(o.BufferCommands.VkCmdSetDepthBounds.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdSetStencilCompareMaskArgs):
		o.BufferCommands.VkCmdSetStencilCompareMask.Set(uint32(o.BufferCommands.VkCmdSetStencilCompareMask.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetStencilCompareMask,
			MapIndex:        uint32(o.BufferCommands.VkCmdSetStencilCompareMask.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdSetStencilWriteMaskArgs):
		o.BufferCommands.VkCmdSetStencilWriteMask.Set(uint32(o.BufferCommands.VkCmdSetStencilWriteMask.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetStencilWriteMask,
			MapIndex:        uint32(o.BufferCommands.VkCmdSetStencilWriteMask.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdSetStencilReferenceArgs):
		o.BufferCommands.VkCmdSetStencilReference.Set(uint32(o.BufferCommands.VkCmdSetStencilReference.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetStencilReference,
			MapIndex:        uint32(o.BufferCommands.VkCmdSetStencilReference.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdBindDescriptorSetsArgs):
		o.BufferCommands.VkCmdBindDescriptorSets.Set(uint32(o.BufferCommands.VkCmdBindDescriptorSets.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBindDescriptorSets,
			MapIndex:        uint32(o.BufferCommands.VkCmdBindDescriptorSets.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdBindIndexBufferArgs):
		o.BufferCommands.VkCmdBindIndexBuffer.Set(uint32(o.BufferCommands.VkCmdBindIndexBuffer.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBindIndexBuffer,
			MapIndex:        uint32(o.BufferCommands.VkCmdBindIndexBuffer.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdBindVertexBuffersArgs):
		o.BufferCommands.VkCmdBindVertexBuffers.Set(uint32(o.BufferCommands.VkCmdBindVertexBuffers.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBindVertexBuffers,
			MapIndex:        uint32(o.BufferCommands.VkCmdBindVertexBuffers.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdDrawArgs):
		o.BufferCommands.VkCmdDraw.Set(uint32(o.BufferCommands.VkCmdDraw.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDraw,
			MapIndex:        uint32(o.BufferCommands.VkCmdDraw.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdDrawIndexedArgs):
		o.BufferCommands.VkCmdDrawIndexed.Set(uint32(o.BufferCommands.VkCmdDrawIndexed.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDrawIndexed,
			MapIndex:        uint32(o.BufferCommands.VkCmdDrawIndexed.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdDrawIndirectArgs):
		o.BufferCommands.VkCmdDrawIndirect.Set(uint32(o.BufferCommands.VkCmdDrawIndirect.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDrawIndirect,
			MapIndex:        uint32(o.BufferCommands.VkCmdDrawIndirect.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdDrawIndexedIndirectArgs):
		o.BufferCommands.VkCmdDrawIndexedIndirect.Set(uint32(o.BufferCommands.VkCmdDrawIndexedIndirect.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDrawIndexedIndirect,
			MapIndex:        uint32(o.BufferCommands.VkCmdDrawIndexedIndirect.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdDispatchArgs):
		o.BufferCommands.VkCmdDispatch.Set(uint32(o.BufferCommands.VkCmdDispatch.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDispatch,
			MapIndex:        uint32(o.BufferCommands.VkCmdDispatch.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdDispatchIndirectArgs):
		o.BufferCommands.VkCmdDispatchIndirect.Set(uint32(o.BufferCommands.VkCmdDispatchIndirect.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDispatchIndirect,
			MapIndex:        uint32(o.BufferCommands.VkCmdDispatchIndirect.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdCopyBufferArgs):
		o.BufferCommands.VkCmdCopyBuffer.Set(uint32(o.BufferCommands.VkCmdCopyBuffer.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdCopyBuffer,
			MapIndex:        uint32(o.BufferCommands.VkCmdCopyBuffer.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdCopyImageArgs):
		o.BufferCommands.VkCmdCopyImage.Set(uint32(o.BufferCommands.VkCmdCopyImage.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdCopyImage,
			MapIndex:        uint32(o.BufferCommands.VkCmdCopyImage.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdBlitImageArgs):
		o.BufferCommands.VkCmdBlitImage.Set(uint32(o.BufferCommands.VkCmdBlitImage.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBlitImage,
			MapIndex:        uint32(o.BufferCommands.VkCmdBlitImage.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdCopyBufferToImageArgs):
		o.BufferCommands.VkCmdCopyBufferToImage.Set(uint32(o.BufferCommands.VkCmdCopyBufferToImage.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdCopyBufferToImage,
			MapIndex:        uint32(o.BufferCommands.VkCmdCopyBufferToImage.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdCopyImageToBufferArgs):
		o.BufferCommands.VkCmdCopyImageToBuffer.Set(uint32(o.BufferCommands.VkCmdCopyImageToBuffer.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdCopyImageToBuffer,
			MapIndex:        uint32(o.BufferCommands.VkCmdCopyImageToBuffer.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdUpdateBufferArgs):
		o.BufferCommands.VkCmdUpdateBuffer.Set(uint32(o.BufferCommands.VkCmdUpdateBuffer.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdUpdateBuffer,
			MapIndex:        uint32(o.BufferCommands.VkCmdUpdateBuffer.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdFillBufferArgs):
		o.BufferCommands.VkCmdFillBuffer.Set(uint32(o.BufferCommands.VkCmdFillBuffer.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdFillBuffer,
			MapIndex:        uint32(o.BufferCommands.VkCmdFillBuffer.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdClearColorImageArgs):
		o.BufferCommands.VkCmdClearColorImage.Set(uint32(o.BufferCommands.VkCmdClearColorImage.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdClearColorImage,
			MapIndex:        uint32(o.BufferCommands.VkCmdClearColorImage.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdClearDepthStencilImageArgs):
		o.BufferCommands.VkCmdClearDepthStencilImage.Set(uint32(o.BufferCommands.VkCmdClearDepthStencilImage.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdClearDepthStencilImage,
			MapIndex:        uint32(o.BufferCommands.VkCmdClearDepthStencilImage.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdClearAttachmentsArgs):
		o.BufferCommands.VkCmdClearAttachments.Set(uint32(o.BufferCommands.VkCmdClearAttachments.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdClearAttachments,
			MapIndex:        uint32(o.BufferCommands.VkCmdClearAttachments.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdResolveImageArgs):
		o.BufferCommands.VkCmdResolveImage.Set(uint32(o.BufferCommands.VkCmdResolveImage.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdResolveImage,
			MapIndex:        uint32(o.BufferCommands.VkCmdResolveImage.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdSetEventArgs):
		o.BufferCommands.VkCmdSetEvent.Set(uint32(o.BufferCommands.VkCmdSetEvent.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdSetEvent,
			MapIndex:        uint32(o.BufferCommands.VkCmdSetEvent.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdResetEventArgs):
		o.BufferCommands.VkCmdResetEvent.Set(uint32(o.BufferCommands.VkCmdResetEvent.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdResetEvent,
			MapIndex:        uint32(o.BufferCommands.VkCmdResetEvent.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdWaitEventsArgs):
		o.BufferCommands.VkCmdWaitEvents.Set(uint32(o.BufferCommands.VkCmdWaitEvents.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdWaitEvents,
			MapIndex:        uint32(o.BufferCommands.VkCmdWaitEvents.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdPipelineBarrierArgs):
		o.BufferCommands.VkCmdPipelineBarrier.Set(uint32(o.BufferCommands.VkCmdPipelineBarrier.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdPipelineBarrier,
			MapIndex:        uint32(o.BufferCommands.VkCmdPipelineBarrier.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdBeginQueryArgs):
		o.BufferCommands.VkCmdBeginQuery.Set(uint32(o.BufferCommands.VkCmdBeginQuery.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBeginQuery,
			MapIndex:        uint32(o.BufferCommands.VkCmdBeginQuery.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdEndQueryArgs):
		o.BufferCommands.VkCmdEndQuery.Set(uint32(o.BufferCommands.VkCmdEndQuery.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdEndQuery,
			MapIndex:        uint32(o.BufferCommands.VkCmdEndQuery.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdResetQueryPoolArgs):
		o.BufferCommands.VkCmdResetQueryPool.Set(uint32(o.BufferCommands.VkCmdResetQueryPool.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdResetQueryPool,
			MapIndex:        uint32(o.BufferCommands.VkCmdResetQueryPool.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdWriteTimestampArgs):
		o.BufferCommands.VkCmdWriteTimestamp.Set(uint32(o.BufferCommands.VkCmdWriteTimestamp.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdWriteTimestamp,
			MapIndex:        uint32(o.BufferCommands.VkCmdWriteTimestamp.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdCopyQueryPoolResultsArgs):
		o.BufferCommands.VkCmdCopyQueryPoolResults.Set(uint32(o.BufferCommands.VkCmdCopyQueryPoolResults.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdCopyQueryPoolResults,
			MapIndex:        uint32(o.BufferCommands.VkCmdCopyQueryPoolResults.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdPushConstantsArgs):
		o.BufferCommands.VkCmdPushConstants.Set(uint32(o.BufferCommands.VkCmdPushConstants.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdPushConstants,
			MapIndex:        uint32(o.BufferCommands.VkCmdPushConstants.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdBeginRenderPassArgs):
		o.BufferCommands.VkCmdBeginRenderPass.Set(uint32(o.BufferCommands.VkCmdBeginRenderPass.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdBeginRenderPass,
			MapIndex:        uint32(o.BufferCommands.VkCmdBeginRenderPass.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdNextSubpassArgs):
		o.BufferCommands.VkCmdNextSubpass.Set(uint32(o.BufferCommands.VkCmdNextSubpass.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdNextSubpass,
			MapIndex:        uint32(o.BufferCommands.VkCmdNextSubpass.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdEndRenderPassArgs):
		o.BufferCommands.VkCmdEndRenderPass.Set(uint32(o.BufferCommands.VkCmdEndRenderPass.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdEndRenderPass,
			MapIndex:        uint32(o.BufferCommands.VkCmdEndRenderPass.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdExecuteCommandsArgs):
		o.BufferCommands.VkCmdExecuteCommands.Set(uint32(o.BufferCommands.VkCmdExecuteCommands.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdExecuteCommands,
			MapIndex:        uint32(o.BufferCommands.VkCmdExecuteCommands.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdDebugMarkerBeginEXTArgs):
		o.BufferCommands.VkCmdDebugMarkerBeginEXT.Set(uint32(o.BufferCommands.VkCmdDebugMarkerBeginEXT.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDebugMarkerBeginEXT,
			MapIndex:        uint32(o.BufferCommands.VkCmdDebugMarkerBeginEXT.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdDebugMarkerEndEXTArgs):
		o.BufferCommands.VkCmdDebugMarkerEndEXT.Set(uint32(o.BufferCommands.VkCmdDebugMarkerEndEXT.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDebugMarkerEndEXT,
			MapIndex:        uint32(o.BufferCommands.VkCmdDebugMarkerEndEXT.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	case (*VkCmdDebugMarkerInsertEXTArgs):
		o.BufferCommands.VkCmdDebugMarkerInsertEXT.Set(uint32(o.BufferCommands.VkCmdDebugMarkerInsertEXT.Len()), d)
		o.CommandReferences.Set(uint32(o.CommandReferences.Len()), CommandReference{
			Buffer:          commandBuffer,
			CommandIndex:    uint32(len(o.Commands)),
			Type:            CommandType_cmd_vkCmdDebugMarkerInsertEXT,
			MapIndex:        uint32(o.BufferCommands.VkCmdDebugMarkerInsertEXT.Len() - 1),
			SemaphoreUpdate: SemaphoreUpdate_None,
			Semaphore:       VkSemaphore(0),
			QueuedCommandData: QueuedCommand{
				initialCall:      e.cmd,
				submit:           nil,
				submissionIndex:  []uint64(nil),
				actualSubmission: true,
			},
		})
	default:
	}

	o.Commands = append(o.Commands, CommandBufferCommand{
		func(ctx context.Context, cmd api.Cmd, id api.CmdID, s *api.GlobalState, b *rb.Builder) {
			e.callSub(ctx, cmd, id, s, b, functionToCall, data)
		}})

	if GetState(e.s).AddCommand != nil {
		GetState(e.s).AddCommand(o.CommandReferences.Get(uint32(len(o.Commands) - 1)))
	}

}

func (e externs) resetCmd(commandBuffer VkCommandBuffer) {
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)
	o.Commands = []CommandBufferCommand{}
	o.CommandReferences = NewU32ːCommandReferenceᵐ()
}

func (e externs) notifyPendingCommandAdded(queue VkQueue) {
	s := GetState(e.s)
	queueObject := s.Queues.Get(queue)
	command := queueObject.PendingCommands.Get(uint32(queueObject.PendingCommands.Len() - 1))
	s.SubCmdIdx[len(s.SubCmdIdx)-1] = uint64(command.CommandIndex)
	command.QueuedCommandData.submit = e.cmd
	command.QueuedCommandData.submissionIndex = append([]uint64(nil), s.SubCmdIdx...)
	command.QueuedCommandData.actualSubmission = true
	queueObject.PendingCommands.Set(uint32(queueObject.PendingCommands.Len()-1), command)
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
	newPendingCommands := NewU32ːCommandReferenceᵐ()
	signaledQueues := []VkQueue{}

	// Store off state.IdxList (Should be empty)
	for _, i := range lastBoundQueue.PendingCommands.KeysSorted() {
		command := lastBoundQueue.PendingCommands.Get(i)
		// Set the state.IdxList to command.Indices
		// Set the state.Queue to command.Queue

		// lastBoundQueue.PendingEvents will be 0 the first time
		// through. (ExecPending could not have been called otherwise).
		// Therefore o.CurrentSubmission will be set by the else
		// branch at least once.
		if lastBoundQueue.PendingEvents.Len() != 0 || lastBoundQueue.PendingSemaphores.Len() != 0 {
			newPendingCommands.Set(uint32(newPendingCommands.Len()), command)
		} else {
			o.SubCmdIdx = append([]uint64{}, command.QueuedCommandData.submissionIndex...)
			if command.SemaphoreUpdate == SemaphoreUpdate_Signal {
				o.Semaphores.Get(command.Semaphore).Signaled = true
				if o.Semaphores.Get(command.Semaphore).WaitingQueue != VkQueue(0) {
					signaledQueue := o.Semaphores.Get(command.Semaphore).WaitingQueue
					o.Queues.Get(signaledQueue).PendingSemaphores.Delete(command.Semaphore)
					o.Semaphores.Get(command.Semaphore).WaitingQueue = VkQueue(0)
					signaledQueues = append(signaledQueues, o.Semaphores.Get(command.Semaphore).WaitingQueue)
				}
			}
			if command.SemaphoreUpdate == SemaphoreUpdate_Unsignal {
				if !o.Semaphores.Get(command.Semaphore).Signaled {
					o.Semaphores.Get(command.Semaphore).WaitingQueue = queue
					lastBoundQueue.PendingSemaphores.Set(command.Semaphore, o.Semaphores.Get(command.Semaphore))
					c := command
					c.QueuedCommandData.submit = o.CurrentSubmission
					newPendingCommands.Set(uint32(newPendingCommands.Len()), c)
					continue
				} else {
					o.Semaphores.Get(command.Semaphore).Signaled = false
				}
			}
			if command.SparseBinds != nil {
				bindSparse(e.ctx, e.cmd, e.cmdID, e.s, command.SparseBinds)
				if o.postBindSparse != nil {
					o.postBindSparse(command.SparseBinds)
				}
			}
			if command.Buffer == VkCommandBuffer(0) {
				continue
			}

			o.CurrentSubmission = command.QueuedCommandData.submit
			if command.QueuedCommandData.actualSubmission && o.PreSubcommand != nil {
				o.PreSubcommand(command)
			}
			buffer := o.CommandBuffers.Get(command.Buffer)
			buffer.Commands[command.CommandIndex].function(e.ctx, e.cmd, e.cmdID, e.s, e.b)
			if command.QueuedCommandData.actualSubmission && o.PostSubcommand != nil {
				// If the just executed subcommand blocks as there are pending events,
				// e.g.: vkCmdWaitEvents, this subcommand should not be considered
				// as finshed and the PostSubcommand callback should not be called.
				if lastBoundQueue.PendingEvents.Len() == 0 {
					o.PostSubcommand(command)
				}
			}
			// If a vkCmdWaitEvent is hit in the pending commands, it will set a new
			// list of pending events to the LastBoundQueue. Once that happens, we
			// should start a new pending command list.
			if lastBoundQueue.PendingEvents.Len() != 0 {
				c := command
				c.QueuedCommandData.submit = o.CurrentSubmission
				newPendingCommands.Set(uint32(newPendingCommands.Len()), c)
			}
		}
	}
	o.SubCmdIdx = []uint64(nil)
	// Reset state.IdxList
	// Refresh or clear the pending commands in LastBoundQueue
	lastBoundQueue.PendingCommands = newPendingCommands
	for _, sq := range signaledQueues {
		e.execPendingCommands(sq)
	}
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
		pNext = Voidᶜᵖᵖ(pNext).Slice(uint64(0), uint64(2), l).Index(uint64(1), l).MustRead(e.ctx, e.cmd, e.s, e.b)
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
		if rpObj.SubpassDescriptions.Len() > 1 {
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

func bindSparse(ctx context.Context, a api.Cmd, id api.CmdID, s *api.GlobalState, binds *QueuedSparseBinds) {
	// Do not use the subroutine: subRoundUpTo because the subroutine takes uint32 arguments
	roundUpTo := func(dividend, divisor VkDeviceSize) VkDeviceSize {
		return (dividend + divisor - 1) / divisor
	}
	st := GetState(s)
	for buffer, binds := range binds.BufferBinds.Range() {
		if !st.Buffers.Contains(buffer) {
			subVkErrorInvalidBuffer(ctx, a, id, nil, s, nil, a.Thread(), nil, buffer)
		}
		bufObj := st.Buffers.Get(buffer)
		blockSize := bufObj.MemoryRequirements.Alignment
		for _, bind := range binds.SparseMemoryBinds.Range() {
			// TODO: assert bind.Size and bind.MemoryOffset must be multiple times of
			// block size.
			numBlocks := roundUpTo(bind.Size, blockSize)
			memOffset := bind.MemoryOffset
			resOffset := bind.ResourceOffset
			for i := VkDeviceSize(0); i < numBlocks; i++ {
				bufObj.SparseMemoryBindings.Set(
					uint64(resOffset), VkSparseMemoryBind{
						ResourceOffset: resOffset,
						Size:           blockSize,
						Memory:         bind.Memory,
						MemoryOffset:   memOffset,
						Flags:          bind.Flags,
					})
				memOffset += blockSize
				resOffset += blockSize
			}
		}
	}
	for image, binds := range binds.OpaqueImageBinds.Range() {
		if !st.Images.Contains(image) {
			subVkErrorInvalidImage(ctx, a, id, nil, s, nil, a.Thread(), nil, image)
		}
		imgObj := st.Images.Get(image)
		blockSize := imgObj.MemoryRequirements.Alignment
		for _, bind := range binds.SparseMemoryBinds.Range() {
			// TODO: assert bind.Size and bind.MemoryOffset must be multiple times of
			// block size.
			numBlocks := roundUpTo(bind.Size, blockSize)
			memOffset := bind.MemoryOffset
			resOffset := bind.ResourceOffset
			for i := VkDeviceSize(0); i < numBlocks; i++ {
				imgObj.OpaqueSparseMemoryBindings.Set(
					uint64(resOffset), VkSparseMemoryBind{
						ResourceOffset: resOffset,
						Size:           blockSize,
						Memory:         bind.Memory,
						MemoryOffset:   memOffset,
						Flags:          bind.Flags,
					})
				memOffset += blockSize
				resOffset += blockSize
			}
		}
	}
	for image, binds := range binds.ImageBinds.Range() {
		if !st.Images.Contains(image) {
			subVkErrorInvalidImage(ctx, a, id, nil, s, nil, a.Thread(), nil, image)
		}
		imgObj := st.Images.Get(image)
		for _, bind := range binds.SparseImageMemoryBinds.Range() {
			if imgObj != nil {
				err := subAddSparseImageMemoryBinding(ctx, a, id, nil, s, nil, a.Thread(), nil, image, bind)
				if err != nil {
					return
				}
			}
		}
	}
}

// TODO: Change to take error message type once all the errors are merged to
// en-us.stb.md
func (e externs) onVkError(err interface{}) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	switch err := err.(type) {
	case *ERR_INVALID_HANDLE:
		issue.Error = fmt.Errorf("Invalid %s: %v", err.HandleType, err.Handle)
	case *ERR_NULL_POINTER:
		issue.Error = fmt.Errorf("Null pointer of %s", err.PointerType)
	case *ERR_UNRECOGNIZED_EXTENSION:
		issue.Severity = service.Severity_WarningLevel
		issue.Error = fmt.Errorf("Unsupported extension: %s", err.Name)
	default:
		log.W(e.ctx, "Unhandled Vulkan error (%T): %v", err, err)
		return
	}
	// Call the state's callback function for API error.
	if f := e.s.OnError; f != nil {
		f(issue)
	}
}
