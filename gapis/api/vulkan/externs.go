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
	if info == 0 {
		return false
	}
	l := e.s.MemoryLayout
	dynamicStateInfo := info.Slice(0, 1, l).MustRead(e.ctx, e.cmd, e.s, e.b)[0]
	states := dynamicStateInfo.PDynamicStates.Slice(0, uint64(dynamicStateInfo.DynamicStateCount), l).MustRead(e.ctx, e.cmd, e.s, e.b)
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
			b.MapMemory(memory.Range{Base: slice.Base(), Size: slice.Size()})
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

func (e externs) resetCmd(commandBuffer VkCommandBuffer) {
	delete(GetState(e.s).initialCommands, commandBuffer)
}

func (e externs) notifyPendingCommandAdded(queue VkQueue) {
	s := GetState(e.s)
	queueObject := s.Queues.Get(queue)
	command := queueObject.PendingCommands.Get(uint32(queueObject.PendingCommands.Len() - 1))
	s.SubCmdIdx[len(s.SubCmdIdx)-1] = uint64(command.CommandIndex)
	s.queuedCommands[command] = QueuedCommand{
		submit:          e.cmd,
		submissionIndex: append([]uint64(nil), s.SubCmdIdx...),
	}

	queueObject.PendingCommands.Set(uint32(queueObject.PendingCommands.Len()-1), command)
}

func (e externs) onCommandAdded(buffer VkCommandBuffer) {
	o := GetState(e.s)
	o.initialCommands[buffer] =
		append(o.initialCommands[buffer], e.cmd)
	b := o.CommandBuffers.Get(buffer)
	if o.AddCommand != nil {
		o.AddCommand(b.CommandReferences.Get(uint32(len(*b.CommandReferences.Map) - 1)))
	}
}

func (e externs) enterSubcontext() {
	o := GetState(e.s)
	o.SubCmdIdx = append(o.SubCmdIdx, 0)
}

func (e externs) resetSubcontext() {
	o := GetState(e.s)
	o.SubCmdIdx = []uint64(nil)
}

func (e externs) leaveSubcontext() {
	o := GetState(e.s)
	o.SubCmdIdx = o.SubCmdIdx[:len(o.SubCmdIdx)-1]
}

func (e externs) nextSubcontext() {
	o := GetState(e.s)
	o.SubCmdIdx[len(o.SubCmdIdx)-1]++
}

func (e externs) onPreSubcommand(ref *CommandReference) {
	o := GetState(e.s)
	cmd := o.queuedCommands[ref]
	o.CurrentSubmission = cmd.submit
	if o.PreSubcommand != nil {
		o.PreSubcommand(ref)
	}
}

func (e externs) onPreProcessCommand(ref *CommandReference) {
	o := GetState(e.s)
	cmd := o.queuedCommands[ref]
	o.SubCmdIdx = append([]uint64{}, cmd.submissionIndex...)
}

func (e externs) onPostSubcommand(ref *CommandReference) {
	o := GetState(e.s)
	if o.PostSubcommand != nil {
		o.PostSubcommand(ref)
	}
}

func (e externs) onDeferSubcommand(ref *CommandReference) {
	o := GetState(e.s)
	r := o.queuedCommands[ref]
	r.submit = o.CurrentSubmission
	o.queuedCommands[ref] = r
}

func (e externs) postBindSparse(binds *QueuedSparseBinds) {
	o := GetState(e.s)
	if o.postBindSparse != nil {
		o.postBindSparse(binds)
	}
}

func (e externs) unmapMemory(slice memory.Slice) {
	if b := e.b; b != nil {
		b.UnmapMemory(memory.Range{Base: slice.Base(), Size: slice.Size()})
	}
}

func (e externs) trackMappedCoherentMemory(start uint64, size memory.Size) {}
func (e externs) readMappedCoherentMemory(memoryHandle VkDeviceMemory, offsetInMapped uint64, readSize memory.Size) {
	l := e.s.MemoryLayout
	mem := GetState(e.s).DeviceMemories.Get(memoryHandle)
	mappedOffset := uint64(mem.MappedOffset)
	dstStart := mappedOffset + offsetInMapped
	srcStart := offsetInMapped

	absSrcStart := mem.MappedLocation.Address() + offsetInMapped
	absSrcMemRng := memory.Range{Base: absSrcStart, Size: uint64(readSize)}

	writeRngList := e.s.Memory.ApplicationPool().Slice(absSrcMemRng).ValidRanges()
	for _, r := range writeRngList {
		mem.Data.Slice(dstStart+r.Base, dstStart+r.Base+r.Size).Copy(
			e.ctx, U8ᵖ(mem.MappedLocation).Slice(srcStart+r.Base, srcStart+r.Base+r.Size, l), e.cmd, e.s, e.b)
	}
}
func (e externs) untrackMappedCoherentMemory(start uint64, size memory.Size) {}

func (e externs) numberOfPNext(pNext Voidᶜᵖ) uint32 {
	l := e.s.MemoryLayout
	counter := uint32(0)
	for pNext != 0 {
		counter++
		pNext = Voidᶜᵖᵖ(pNext).Slice(1, 2, l).MustRead(e.ctx, e.cmd, e.s, e.b)[0]
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

func (e externs) vkErrInvalidHandle(handleType string, handle uint64) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Invalid %s: %v", handleType, handle)
}

func (e externs) vkErrNullPointer(pointerType string) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Null pointer of %s", pointerType)
}

func (e externs) vkErrUnrecognizedExtension(name string) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_WarningLevel
	issue.Error = fmt.Errorf("Unsupported extension: %s", name)
}

func (e externs) vkErrExpectNVDedicatedlyAllocatedHandle(handleType string, handle uint64) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_WarningLevel
	issue.Error = fmt.Errorf("%v: %v is not created with VK_NV_dedicated_allocation extension structure, but is bound to a dedicatedly allocated handle", handleType, handle)
}

func (e externs) vkErrInvalidDescriptorArrayElement(set uint64, binding, arrayIndex uint32) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_WarningLevel
	issue.Error = fmt.Errorf("Invalid descriptor array element specified by descriptor set: %v, binding: %v array index: %v", set, binding, arrayIndex)
}

func (e externs) vkErrCommandBufferIncomplete(cmdbuf VkCommandBuffer) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Executing command buffer %v was not in the COMPLETED state", cmdbuf)
}

func (e externs) vkErrImageLayout(layout VkImageLayout, expectedLayout VkImageLayout) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_WarningLevel
	issue.Error = fmt.Errorf("Image was in layout %v, but was expected to be in layout %v", layout, expectedLayout)
}
