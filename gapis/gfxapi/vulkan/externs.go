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
	"reflect"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
	rb "github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/protocol"
)

type externs struct {
	ctx context.Context // Allowed because the externs struct is only a parameter proxy for a single call
	a   atom.Atom
	s   *gfxapi.State
	b   *rb.Builder
}

func (e externs) hasDynamicProperty(info VkPipelineDynamicStateCreateInfoᶜᵖ,
	state VkDynamicState) bool {
	if (info) == (VkPipelineDynamicStateCreateInfoᶜᵖ{}) {
		return false
	}
	l := e.s.MemoryLayout
	dynamic_state_info := info.Slice(uint64(0), uint64(1), l).Index(uint64(0), l).Read(e.ctx, e.a, e.s, e.b)
	states := dynamic_state_info.PDynamicStates.Slice(uint64(uint32(0)), uint64(dynamic_state_info.DynamicStateCount), l).Read(e.ctx, e.a, e.s, e.b)
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
		switch e.a.(type) {
		case *VkMapMemory:
			b.Load(protocol.Type_AbsolutePointer, value.value(e.b, e.a, e.s))
			b.MapMemory(slice.Range(e.s.MemoryLayout))
		default:
			log.E(ctx, "mapBuffer extern called for unsupported atom: %v", e.a)
		}
	}
}

func (e externs) addCmd(commandBuffer VkCommandBuffer, recreate_data interface{}, data interface{}, functionToCall interface{}) {
	args := []reflect.Value{
		reflect.ValueOf(e.ctx),
		reflect.ValueOf(e.a),
		reflect.ValueOf(&atom.Observations{}),
		reflect.ValueOf(e.s),
		reflect.ValueOf(GetState(e.s)),
		reflect.ValueOf(e.b),
		reflect.ValueOf(data),
	}
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)

	o.Commands = append(o.Commands, CommandBufferCommand{func() {
		reflect.ValueOf(functionToCall).Call(args)
	}, &e.a})
}

func (e externs) resetCmd(commandBuffer VkCommandBuffer) {
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)
	o.Commands = []CommandBufferCommand{}
}

func (e externs) execCommands(commandBuffer VkCommandBuffer) {
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)
	lastBoundQueue := GetState(e.s).LastBoundQueue
	for _, command := range o.Commands {
		if len(lastBoundQueue.PendingEvents) != 0 {
			lastBoundQueue.PendingCommands = append(lastBoundQueue.PendingCommands,
				command)
		} else {
			command.function()
			// If a vkCmdWaitEvents is hit in the commands, it will set the pending
			// events list of the current LastBoundQueue. Once that happens, we should
			// records all the following commands to the pending commands list.
			if len(lastBoundQueue.PendingEvents) != 0 {
				// The vkCmdWaitEvents carries memory barriers, those should take
				// effect when the event is signaled.
				lastBoundQueue.PendingCommands = append(lastBoundQueue.PendingCommands,
					command)
			}
		}
	}
}

func (e externs) execPendingCommands(queue VkQueue) {
	// Set the global LastBoundQueue, so the next vkCmdWaitEvent in the pending
	// commands knows in which queue it will be waiting.
	GetState(e.s).LastBoundQueue = GetState(e.s).Queues.Get(queue)
	lastBoundQueue := GetState(e.s).LastBoundQueue
	newPendingCommands := []CommandBufferCommand{}
	for _, command := range lastBoundQueue.PendingCommands {
		if len(lastBoundQueue.PendingEvents) != 0 {
			newPendingCommands = append(newPendingCommands, command)
		} else {
			command.function()
			// If a vkCmdWaitEvent is hit in the pending commands, it will set a new
			// list of pending events to the LastBoundQueue. Once that happens, we
			// should start a new pending command list.
			if len(lastBoundQueue.PendingEvents) != 0 {
				newPendingCommands = append(newPendingCommands, command)
			}
		}
	}
	// Refresh or clear the pending commands in LastBoundQueue
	lastBoundQueue.PendingCommands = newPendingCommands
}

func (e externs) recordUpdateSemaphoreSignal(semaphore VkSemaphore, Signaled bool) {
	signal_semaphore := CommandBufferCommand{
		function: func() {
			GetState(e.s).Semaphores[semaphore].Signaled = Signaled
		},
		a: &e.a,
	}
	lastBoundQueue := GetState(e.s).LastBoundQueue
	if len(lastBoundQueue.PendingEvents) != 0 {
		lastBoundQueue.PendingCommands = append(lastBoundQueue.PendingCommands,
			signal_semaphore)
	} else {
		signal_semaphore.function()
	}
}

func (e externs) createUpdateBufferData(buffer VkBuffer, offset VkDeviceSize, size VkDeviceSize, data interface{}) *RecreateAndBeginCommandBuffer {
	return nil
}

func (e externs) createPushConstantsData(layout VkPipelineLayout, stageFlags VkShaderStageFlags, offset uint32, size uint32, data interface{}) *RecreateCmdPushConstantsData {
	return nil
}

func (e externs) addWords(module VkShaderModule, numBytes interface{}, data interface{}) {
}

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
	memory := GetState(e.s).DeviceMemories.Get(memory_handle)
	mapped_offset := uint64(memory.MappedOffset)
	dstStart := mapped_offset + offset_in_mapped
	srcStart := offset_in_mapped
	srcEnd := offset_in_mapped + uint64(read_size)

	//TODO: Add the PageSize to the architecture header of trace.
	// Here we relay on the underlying optimization to avoid creating duplicated slice.
	// A larger copy size makes a fewer number of call to read() and results into a faster replay.
	// But a large copy size generates more data to be stored in the server and uses too much memory.
	// A smaller copy size saves memory, but slow down the replay speed.
	// By far, spliting the data into PageSize chunks seems like the best option.
	copySize := uint64(4196)

	for srcStart < srcEnd {
		if srcStart+copySize > srcEnd {
			copySize = srcEnd - srcStart
		}
		memory.Data.Slice(dstStart, dstStart+copySize, l).Copy(
			e.ctx, U8ᵖ(memory.MappedLocation).Slice(srcStart, srcStart+copySize, l), e.a, e.s, e.b)
		srcStart += copySize
		dstStart += copySize
	}
}
func (e externs) untrackMappedCoherentMemory(start uint64, size memory.Size) {}

func (e externs) numberOfPNext(pNext Voidᶜᵖ) uint32 {
	l := e.s.MemoryLayout
	counter := uint32(0)
	for (pNext) != (Voidᶜᵖ{}) {
		counter++
		pNext = Voidᶜᵖᵖ(pNext).Slice(uint64(0), uint64(2), l).Index(uint64(1), l).Read(e.ctx, e.a, e.s, e.b)
	}
	return counter
}
