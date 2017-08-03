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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
	rb "github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/protocol"
)

type externs struct {
	ctx context.Context // Allowed because the externs struct is only a parameter proxy for a single call
	cmd api.Cmd
	s   *api.State
	b   *rb.Builder
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

func (e externs) addCmd(commandBuffer VkCommandBuffer, recreate_data interface{}, data interface{}, functionToCall interface{}) {
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)

	o.Commands = append(o.Commands, CommandBufferCommand{func(ctx context.Context,
		cmd api.Cmd, s *api.State, b *rb.Builder) {
		args := []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(cmd),
			reflect.ValueOf(&api.CmdObservations{}),
			reflect.ValueOf(s),
			reflect.ValueOf(GetState(s)),
			reflect.ValueOf(cmd.Thread()),
			reflect.ValueOf(b),
			reflect.ValueOf(data),
		}
		reflect.ValueOf(functionToCall).Call(args)
	}, &e.cmd, nil, []uint64(nil), recreate_data, true})
	if GetState(e.s).AddCommand != nil {
		GetState(e.s).AddCommand(o.Commands[len(o.Commands)-1])
	}
}

func (e externs) resetCmd(commandBuffer VkCommandBuffer) {
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)
	o.Commands = []CommandBufferCommand{}
}

func (e externs) execCommands(commandBuffer VkCommandBuffer) {
	s := GetState(e.s)
	o := GetState(e.s).CommandBuffers.Get(commandBuffer)
	if _, ok := e.cmd.(*VkQueueSubmit); ok {
		s.CurrentSubmission = &e.cmd
	}
	e.enterSubcontext()
	defer e.leaveSubcontext()
	lastBoundQueue := GetState(e.s).LastBoundQueue
	for _, command := range o.Commands {
		if len(lastBoundQueue.PendingEvents) != 0 {
			c := command
			c.actualSubmission = true
			c.submit = s.CurrentSubmission
			c.submissionIndex = append([]uint64(nil), s.SubCmdIdx...)
			lastBoundQueue.PendingCommands = append(lastBoundQueue.PendingCommands,
				c)
		} else {
			command.function(e.ctx, e.cmd, e.s, e.b)
			if command.actualSubmission && s.HandleSubcommand != nil {
				s.HandleSubcommand(command)
			}
			// If a vkCmdWaitEvents is hit in the commands, it will set the pending
			// events list of the current LastBoundQueue. Once that happens, we should
			// records all the following commands to the pending commands list.
			if len(lastBoundQueue.PendingEvents) != 0 {
				// We end up submitting VkCmdWaitEvents twice, once
				// "call" it in the VkQueueSubmit, and again later to register
				// the semaphores. Keep track of which of these states we are in.
				c := command
				c.actualSubmission = false
				c.submit = s.CurrentSubmission
				c.submissionIndex = append([]uint64(nil), s.SubCmdIdx...)
				// The vkCmdWaitEvents carries memory barriers, those should take
				// effect when the event is signaled.
				lastBoundQueue.PendingCommands = append(lastBoundQueue.PendingCommands,
					c)
			}
		}
		e.nextSubcontext()
	}
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
	newPendingCommands := []CommandBufferCommand{}

	// Store off state.IdxList (Should be empty)
	for _, command := range lastBoundQueue.PendingCommands {
		// Set the state.IdxList to command.Indices
		// Set the state.Queue to command.Queue

		// lastBoundQueue.PendingEvents will be 0 the first time
		// through. (ExecPending could not have been called otherwise).
		// Therefore o.CurrentSubmission will be set by the else
		// branch at least once.
		if len(lastBoundQueue.PendingEvents) != 0 {
			c := command
			c.actualSubmission = true
			c.submit = o.CurrentSubmission
			c.submissionIndex = append([]uint64(nil), o.SubCmdIdx...)
			newPendingCommands = append(newPendingCommands, c)
		} else {
			o.CurrentSubmission = command.submit
			o.SubCmdIdx = append([]uint64(nil), command.submissionIndex...)
			command.function(e.ctx, e.cmd, e.s, e.b)
			if command.actualSubmission && o.HandleSubcommand != nil {
				o.HandleSubcommand(command)
			}
			// If a vkCmdWaitEvent is hit in the pending commands, it will set a new
			// list of pending events to the LastBoundQueue. Once that happens, we
			// should start a new pending command list.
			if len(lastBoundQueue.PendingEvents) != 0 {
				c := command
				c.actualSubmission = false
				c.submit = o.CurrentSubmission
				newPendingCommands = append(newPendingCommands, c)
			}
		}
		if command.actualSubmission {
			o.SubCmdIdx[len(o.SubCmdIdx)-1] += 1
		}
	}
	o.SubCmdIdx = []uint64(nil)
	// Reset state.IdxList
	// Refresh or clear the pending commands in LastBoundQueue
	lastBoundQueue.PendingCommands = newPendingCommands
}

func (e externs) recordUpdateSemaphoreSignal(semaphore VkSemaphore, Signaled bool) {
	signal_semaphore := CommandBufferCommand{
		function: func(ctx context.Context, cmd api.Cmd, s *api.State, b *rb.Builder) {
			if s, ok := GetState(s).Semaphores[semaphore]; ok {
				s.Signaled = Signaled
			}
		},
		actualSubmission: false,
	}
	lastBoundQueue := GetState(e.s).LastBoundQueue
	if len(lastBoundQueue.PendingEvents) != 0 {
		lastBoundQueue.PendingCommands = append(lastBoundQueue.PendingCommands,
			signal_semaphore)
	} else {
		signal_semaphore.function(e.ctx, e.cmd, e.s, e.b)
	}
}

func (e externs) createUpdateBufferData(buffer VkBuffer, offset VkDeviceSize, size VkDeviceSize, data Voidᶜᵖ) *RecreateCmdUpdateBufferData {
	d := U8ᵖ(data).Slice(uint64(uint32(0)), uint64(size), e.s.MemoryLayout).Read(e.ctx, e.cmd, e.s, e.b)
	return &RecreateCmdUpdateBufferData{
		buffer, offset, size, d,
	}
}

func (e externs) doUpdateBuffer(args *RecreateCmdUpdateBufferData) {
	l := e.s.MemoryLayout
	o := GetState(e.s)
	buffer := o.Buffers[args.DstBuffer]
	bufferOffset := buffer.MemoryOffset

	buffer.Memory.Data.Slice(uint64(bufferOffset+args.DstOffset), uint64((bufferOffset)+args.DataSize+args.DstOffset), l).Write(e.ctx, args.Data, e.cmd, e.s, e.b)
}

type RecreateCmdPushConstantsDataExpanded struct {
	Layout     VkPipelineLayout
	StageFlags VkShaderStageFlags
	Offset     uint32
	Size       uint32
	Data       []uint8
}

func (e externs) createPushConstantsData(layout VkPipelineLayout, stageFlags VkShaderStageFlags, offset uint32, size uint32, data Voidᶜᵖ) *RecreateCmdPushConstantsDataExpanded {
	d := U8ᵖ(data).Slice(uint64(uint32(0)), uint64(size), e.s.MemoryLayout).Read(e.ctx, e.cmd, e.s, e.b)
	return &RecreateCmdPushConstantsDataExpanded{
		Layout:     layout,
		StageFlags: stageFlags,
		Offset:     offset,
		Size:       size,
		Data:       d,
	}
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
	mem := GetState(e.s).DeviceMemories.Get(memory_handle)
	mapped_offset := uint64(mem.MappedOffset)
	dstStart := mapped_offset + offset_in_mapped
	srcStart := offset_in_mapped

	absSrcStart := mem.MappedLocation.Address() + offset_in_mapped
	absSrcMemRng := memory.Range{Base: absSrcStart, Size: uint64(read_size)}

	writeRngList := e.s.Memory[memory.ApplicationPool].Slice(absSrcMemRng).ValidRanges()
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
