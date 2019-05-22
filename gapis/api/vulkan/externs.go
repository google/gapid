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
	"math/bits"
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
	w     api.StateWatcher
}

func (e externs) hasDynamicProperty(info VkPipelineDynamicStateCreateInfoᶜᵖ,
	state VkDynamicState) bool {
	if info == 0 {
		return false
	}
	l := e.s.MemoryLayout
	dynamicStateInfo := info.Slice(0, 1, l).MustReadʷ(e.ctx, e.cmd, e.s, e.b, e.w)[0]
	states := dynamicStateInfo.PDynamicStatesʷ(e.ctx, e.w, true).Slice(0, uint64(dynamicStateInfo.DynamicStateCountʷ(e.ctx, e.w, true)), l).MustReadʷ(e.ctx, e.cmd, e.s, e.b, e.w)
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

type mappedMemory VkDeviceMemory

func (e externs) mapMemory(handle VkDeviceMemory, value Voidᵖᵖ, slice memory.Slice) {
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
	if e.w != nil {
		e.w.OpenForwardDependency(e.ctx, mappedMemory(handle))
	}
}

// CallReflectedCommand unpacks the given subcommand and arguments, and calls the method
func CallReflectedCommand(ctx context.Context, cmd api.Cmd, id api.CmdID, s *api.GlobalState, b *rb.Builder, w api.StateWatcher, sub, data interface{}) {
	reflect.ValueOf(sub).Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(cmd),
		reflect.ValueOf(id),
		reflect.ValueOf(&api.CmdObservations{}),
		reflect.ValueOf(s),
		reflect.ValueOf(GetState(s)),
		reflect.ValueOf(cmd.Thread()),
		reflect.ValueOf(b),
		reflect.ValueOf(&w).Elem(),
		reflect.ValueOf(data),
	})
}

func (e externs) resetCmd(commandBuffer VkCommandBuffer) {
	delete(GetState(e.s).initialCommands, commandBuffer)
}

func (e externs) notifyPendingCommandAdded(queue VkQueue) {
	s := GetState(e.s)
	queueObject := s.Queuesʷ(e.ctx, e.w, true).Getʷ(e.ctx, e.w, true, queue)
	command := queueObject.PendingCommandsʷ(e.ctx, e.w, false).Getʷ(e.ctx, e.w, true, uint32(queueObject.PendingCommandsʷ(e.ctx, e.w, true).Lenʷ(e.ctx, e.w, false)-1))
	s.SubCmdIdx[len(s.SubCmdIdx)-1] = uint64(command.CommandIndexʷ(e.ctx, e.w, true))
	s.queuedCommands[command] = QueuedCommand{
		submit:          e.cmd,
		submissionIndex: append([]uint64(nil), s.SubCmdIdx...),
	}

	queueObject.PendingCommandsʷ(e.ctx, e.w, true).Addʷ(e.ctx, e.w, false, uint32(queueObject.PendingCommandsʷ(e.ctx, e.w, true).Lenʷ(e.ctx, e.w, false)-1), command)
}

func (e externs) onCommandAdded(buffer VkCommandBuffer) {
	o := GetState(e.s)
	o.initialCommands[buffer] =
		append(o.initialCommands[buffer], e.cmd)
	b := o.CommandBuffersʷ(e.ctx, e.w, true).Getʷ(e.ctx, e.w, true, buffer)
	refs := b.CommandReferencesʷ(e.ctx, e.w, true)
	idx := uint32(refs.Lenʷ(e.ctx, e.w, false) - 1)
	c := refs.Getʷ(e.ctx, e.w, false, idx)
	if o.AddCommand != nil {
		o.AddCommand(c)
	}
	if e.w != nil {
		e.w.OnRecordSubCmd(e.ctx, api.RecordIdx{uint64(buffer), uint64(idx)})
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

func (e externs) onPreSubcommand(ref CommandReferenceʳ) {
	o := GetState(e.s)
	cmd := o.queuedCommands[ref]
	o.CurrentSubmission = cmd.submit
	if o.PreSubcommand != nil {
		o.PreSubcommand(ref)
	}
	if e.w != nil {
		e.w.OnBeginSubCmd(e.ctx, o.SubCmdIdx, api.RecordIdx{uint64(ref.Buffer()), uint64(ref.CommandIndex())})
	}
}

func (e externs) onPreProcessCommand(ref CommandReferenceʳ) {
	o := GetState(e.s)
	cmd := o.queuedCommands[ref]
	o.SubCmdIdx = append([]uint64{}, cmd.submissionIndex...)
}

func (e externs) onPostSubcommand(ref CommandReferenceʳ) {
	o := GetState(e.s)
	if o.PostSubcommand != nil {
		o.PostSubcommand(ref)
	}
	if e.w != nil {
		e.w.OnEndSubCmd(e.ctx)
	}
}

func (e externs) onDeferSubcommand(ref CommandReferenceʳ) {
	o := GetState(e.s)
	r := o.queuedCommands[ref]
	r.submit = o.CurrentSubmission
	o.queuedCommands[ref] = r
}

func (e externs) postBindSparse(binds QueuedSparseBindsʳ) {
	o := GetState(e.s)
	if o.postBindSparse != nil {
		o.postBindSparse(binds)
	}
}

func (e externs) unmapMemory(handle VkDeviceMemory, slice memory.Slice) {
	if b := e.b; b != nil {
		b.UnmapMemory(memory.Range{Base: slice.Base(), Size: slice.Size()})
	}
	if e.w != nil {
		e.w.CloseForwardDependency(e.ctx, mappedMemory(handle))
	}
}

func (e externs) trackMappedCoherentMemory(start uint64, size memory.Size) {}
func (e externs) readMappedCoherentMemory(memoryHandle VkDeviceMemory, offsetInMapped uint64, readSize memory.Size) {
	l := e.s.MemoryLayout
	mem := GetState(e.s).DeviceMemoriesʷ(e.ctx, e.w, true).Getʷ(e.ctx, e.w, true, memoryHandle)
	mappedOffset := uint64(mem.MappedOffsetʷ(e.ctx, e.w, true))
	dstStart := mappedOffset + offsetInMapped
	srcStart := offsetInMapped

	absSrcStart := mem.MappedLocationʷ(e.ctx, e.w, true).Address() + offsetInMapped
	absSrcMemRng := memory.Range{Base: absSrcStart, Size: uint64(readSize)}

	writeRngList := e.s.Memory.ApplicationPool().Slice(absSrcMemRng).ValidRanges()
	for _, r := range writeRngList {
		mem.Dataʷ(e.ctx, e.w, true).Slice(dstStart+r.Base, dstStart+r.Base+r.Size).
			Copy(e.ctx, U8ᵖ(mem.MappedLocationʷ(e.ctx, e.w, true)).Slice(srcStart+r.Base, srcStart+r.Base+r.Size, l), e.cmd, e.s, e.b, e.w)
	}
}
func (e externs) untrackMappedCoherentMemory(start uint64, size memory.Size) {}

func (e externs) numberOfPNext(pNext Voidᶜᵖ) uint32 {
	l := e.s.MemoryLayout
	counter := uint32(0)
	for pNext != 0 {
		counter++
		pNext = Voidᶜᵖᵖ(pNext).Slice(1, 2, l).MustReadʷ(e.ctx, e.cmd, e.s, e.b, e.w)[0]
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
		rpObj := GetState(e.s).RenderPassesʷ(e.ctx, e.w, true).Getʷ(e.ctx, e.w, true, rp)
		var name string
		if !rpObj.DebugInfoʷ(e.ctx, e.w, true).IsNil() && len(rpObj.DebugInfoʷ(e.ctx, e.w, true).ObjectNameʷ(e.ctx, e.w, true)) > 0 {
			name = rpObj.DebugInfoʷ(e.ctx, e.w, true).ObjectNameʷ(e.ctx, e.w, true)
		} else {
			name = fmt.Sprintf("RenderPass: %v", rp)
		}
		GetState(e.s).pushMarkerGroup(name, false, RenderPassMarker)
		if rpObj.SubpassDescriptionsʷ(e.ctx, e.w, true).Lenʷ(e.ctx, e.w, true) > 1 {
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

func (e externs) fetchPhysicalDeviceProperties(inst VkInstance, devs VkPhysicalDeviceˢ) PhysicalDevicesAndPropertiesʳ {
	for _, ee := range e.cmd.Extras().All() {
		if p, ok := ee.(PhysicalDevicesAndProperties); ok {
			return MakePhysicalDevicesAndPropertiesʳ(e.s.Arena).Set(p).Clone(e.s.Arena, api.CloneContext{})
		}
	}
	return NilPhysicalDevicesAndPropertiesʳ
}

func (e externs) fetchPhysicalDeviceMemoryProperties(inst VkInstance, devs VkPhysicalDeviceˢ) PhysicalDevicesMemoryPropertiesʳ {
	for _, ee := range e.cmd.Extras().All() {
		if p, ok := ee.(PhysicalDevicesMemoryProperties); ok {
			return MakePhysicalDevicesMemoryPropertiesʳ(e.s.Arena).Set(p)
		}
	}
	return NilPhysicalDevicesMemoryPropertiesʳ
}

func (e externs) fetchPhysicalDeviceQueueFamilyProperties(inst VkInstance, devs VkPhysicalDeviceˢ) PhysicalDevicesAndQueueFamilyPropertiesʳ {
	for _, ee := range e.cmd.Extras().All() {
		if p, ok := ee.(PhysicalDevicesAndQueueFamilyProperties); ok {
			return MakePhysicalDevicesAndQueueFamilyPropertiesʳ(e.s.Arena).Set(p).Clone(e.s.Arena, api.CloneContext{})
		}
	}
	return NilPhysicalDevicesAndQueueFamilyPropertiesʳ
}

func (e externs) fetchPhysicalDeviceFormatProperties(inst VkInstance, devs VkPhysicalDeviceˢ) PhysicalDevicesFormatPropertiesʳ {
	for _, ee := range e.cmd.Extras().All() {
		if p, ok := ee.(PhysicalDevicesFormatProperties); ok {
			return MakePhysicalDevicesFormatPropertiesʳ(e.s.Arena).Set(p).Clone(e.s.Arena, api.CloneContext{})
		}
	}
	return NilPhysicalDevicesFormatPropertiesʳ
}

func (e externs) fetchImageMemoryRequirements(dev VkDevice, img ImageObjectʳ, hasSparseBit bool) FetchedImageMemoryRequirementsʳ {
	// Only fetch memory requirements for application commands, skip any commands
	// inserted by GAPID
	if e.cmdID == api.CmdNoID {
		return NilFetchedImageMemoryRequirementsʳ
	}
	for _, ee := range e.cmd.Extras().All() {
		if r, ok := ee.(FetchedImageMemoryRequirements); ok {
			return MakeFetchedImageMemoryRequirementsʳ(e.s.Arena).Set(r).Clone(e.s.Arena, api.CloneContext{})
		}
	}
	return NilFetchedImageMemoryRequirementsʳ
}

func (e externs) fetchUsedDescriptors(ShaderModuleObjectʳ) DescriptorInfoʳ {
	for _, ee := range e.cmd.Extras().All() {
		if p, ok := ee.(DescriptorInfo); ok {
			return MakeDescriptorInfoʳ(e.s.Arena).Set(p).Clone(e.s.Arena, api.CloneContext{})
		}
	}
	return NilDescriptorInfoʳ
}

func (e externs) fetchBufferMemoryRequirements(dev VkDevice, buf VkBuffer) VkMemoryRequirements {
	// Only fetch memory requirements for application commands, skip any commands
	// inserted by GAPID
	if e.cmdID == api.CmdNoID {
		return MakeVkMemoryRequirements(e.s.Arena)
	}
	for _, ee := range e.cmd.Extras().All() {
		if r, ok := ee.(VkMemoryRequirements); ok {
			return r.Clone(e.s.Arena, api.CloneContext{})
		}
	}
	return MakeVkMemoryRequirements(e.s.Arena)
}

func (e externs) fetchLinearImageSubresourceLayouts(dev VkDevice, img ImageObjectʳ, rng VkImageSubresourceRange) LinearImageLayoutsʳ {
	// Only fetch linear image layouts for application commands, skip any commands
	// inserted by GAPID
	if e.cmdID == api.CmdNoID {
		return NilLinearImageLayoutsʳ
	}
	for _, ee := range e.cmd.Extras().All() {
		if r, ok := ee.(LinearImageLayouts); ok {
			return MakeLinearImageLayoutsʳ(e.s.Arena).Set(r).Clone(e.s.Arena, api.CloneContext{})
		}
	}
	return NilLinearImageLayoutsʳ
}

func (e externs) onVkError(issue replay.Issue) {
	if f := e.s.OnError; f != nil {
		f(issue)
	}
}

func (e externs) vkErrInvalidHandle(handleType string, handle uint64) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Invalid %s: %v", handleType, handle)
	e.onVkError(issue)
}

func (e externs) vkErrNullPointer(pointerType string) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Null pointer of %s", pointerType)
	e.onVkError(issue)
}

func (e externs) vkErrNotNullPointer(pointerType string) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Not Null pointer of %s", pointerType)
	e.onVkError(issue)
}

func (e externs) vkErrUnrecognizedExtension(name string) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_WarningLevel
	issue.Error = fmt.Errorf("Unsupported extension: %s", name)
	e.onVkError(issue)
}

func (e externs) vkErrExpectNVDedicatedlyAllocatedHandle(handleType string, handle uint64) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_WarningLevel
	issue.Error = fmt.Errorf("%v: %v is not created with VK_NV_dedicated_allocation extension structure, but is bound to a dedicatedly allocated handle", handleType, handle)
	e.onVkError(issue)
}

func (e externs) vkErrInvalidDescriptorArrayElement(set uint64, binding, arrayIndex uint32) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_WarningLevel
	issue.Error = fmt.Errorf("Invalid descriptor array element specified by descriptor set: %v, binding: %v array index: %v", set, binding, arrayIndex)
	e.onVkError(issue)
}

func (e externs) vkErrCommandBufferIncomplete(cmdbuf VkCommandBuffer) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Executing command buffer %v was not in the COMPLETED state", cmdbuf)
	e.onVkError(issue)
}

func (e externs) vkErrCommandBufferNotRecording(cmdbuf VkCommandBuffer) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Executing command buffer %v was not in the RECORDING state", cmdbuf)
	e.onVkError(issue)
}

func (e externs) vkErrQueryOutOfRange(queryPool VkQueryPool, query uint32) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Query %v in QueryPool %v was out of range", query, queryPool)
	e.onVkError(issue)
}

func (e externs) vkErrQueryUninitialized(queryPool VkQueryPool, query uint32) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Query %v in QueryPool %v was uninitialized", query, queryPool)
	e.onVkError(issue)
}

func (e externs) vkErrQueryNotInactive(queryPool VkQueryPool, query uint32) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Query %v in QueryPool %v was not in the INACTIVE state", query, queryPool)
	e.onVkError(issue)
}

func (e externs) vkErrQueryNotActive(queryPool VkQueryPool, query uint32) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Query %v in QueryPool %v was not in the ACTIVE state", query, queryPool)
	e.onVkError(issue)
}

func (e externs) vkErrInvalidImageLayout(img VkImage, aspect, layer, level uint32, layout VkImageLayout, expectedLayout VkImageLayout) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_WarningLevel
	issue.Error = fmt.Errorf("Image subsource at Image: %v AspectBit: %v, Layer: %v, Level: %v was in layout %v, but was expected to be in layout %v", uint64(img), aspect, layer, level, layout, expectedLayout)
	e.onVkError(issue)
}

func (e externs) vkErrInvalidImageSubresource(img VkImage, subresourceType string, value uint32) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_WarningLevel
	issue.Error = fmt.Errorf("Accessing invalid image subresource at Image: %v, %v: %v", uint64(img), subresourceType, value)
	e.onVkError(issue)
}

func (e externs) vkErrImageMemoryNotBound(img VkImage) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_WarningLevel
	issue.Error = fmt.Errorf("Image %v has no bound memory", uint64(img))
	e.onVkError(issue)
}

func (e externs) vkErrInvalidDescriptorBindingType(set VkDescriptorSet, binding uint32, layoutType, updateType VkDescriptorType) {
	var issue replay.Issue
	issue.Command = e.cmdID
	issue.Severity = service.Severity_ErrorLevel
	issue.Error = fmt.Errorf("Updating descriptor binding at: %v: %d with type: %v, but the type defined in descriptor set layout is: %v", set, binding, layoutType, updateType)
	e.onVkError(issue)
}

type fenceSignal uint64

func (e externs) recordFenceSignal(fence VkFence) {
	if e.w != nil {
		e.w.OpenForwardDependency(e.ctx, fenceSignal(fence))
	}
}

func (e externs) recordFenceWait(fence VkFence) {
	if e.w != nil {
		e.w.CloseForwardDependency(e.ctx, fenceSignal(fence))
	}
}

func (e externs) recordFenceReset(fence VkFence) {
	if e.w != nil {
		e.w.DropForwardDependency(e.ctx, fenceSignal(fence))
	}
}

type eventSignal uint64

func (e externs) recordEventWait(event VkEvent) {
	if e.w != nil {
		e.w.OpenForwardDependency(e.ctx, eventSignal(event))
	}
}

func (e externs) recordEventSet(event VkEvent) {
	if e.w != nil {
		e.w.CloseForwardDependency(e.ctx, eventSignal(event))
	}
}

type swapchainImage struct {
	swapchain  VkSwapchainKHR
	imageIndex uint32
}

func (e externs) recordAcquireNextImage(swapchain VkSwapchainKHR, imageIndex uint32) {
	if e.w != nil {
		e.w.OpenForwardDependency(e.ctx, swapchainImage{swapchain, imageIndex})
	}
}

func (e externs) recordPresentSwapchainImage(swapchain VkSwapchainKHR, imageIndex uint32) {
	if e.w != nil {
		e.w.CloseForwardDependency(e.ctx, swapchainImage{swapchain, imageIndex})
	}
}

func (e externs) recordBeginCommandBuffer(commandBuffer VkCommandBuffer) {
	if e.w != nil {
		e.w.OpenForwardDependency(e.ctx, commandBuffer)
	}
}

func (e externs) recordEndCommandBuffer(commandBuffer VkCommandBuffer) {
	if e.w != nil {
		e.w.CloseForwardDependency(e.ctx, commandBuffer)
	}
}

func (e externs) onesCount(a uint32) uint32 {
	return (uint32)(bits.OnesCount32(a))
}
