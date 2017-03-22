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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
)

func (i VkInstance) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkPhysicalDevice) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDevice) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkQueue) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkCommandBuffer) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkSemaphore) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkFence) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDeviceMemory) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkBuffer) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkImage) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkEvent) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkQueryPool) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkBufferView) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkImageView) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkShaderModule) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkPipelineCache) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkPipelineLayout) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkRenderPass) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkPipeline) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDescriptorSetLayout) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkSampler) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDescriptorPool) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDescriptorSet) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkFramebuffer) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkCommandPool) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkSurfaceKHR) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkSwapchainKHR) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDisplayKHR) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDisplayModeKHR) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDebugReportCallbackEXT) remap(_ atom.Atom, _ *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (a *VkCreateInstance) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	// Hijack VkCreateInstance's Mutate() method entirely with our ReplayCreateVkInstance's Mutate().

	// As long as we guarantee that the synthetic replayCreateVkInstance API function has the same
	// logic as the real vkCreateInstance API function, we can do observation correctly. Additionally,
	// ReplayCreateVkInstance's Mutate() will invoke our custom wrapper function replayCreateVkInstance()
	// in vulkan_gfx_api_extras.cpp, which modifies VkInstanceCreateInfo to enable virtual swapchain
	// layer before delegating the real work back to the normal flow.

	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer(a.PAllocator)
	pInstance := memory.Pointer(a.PInstance)
	hijack := NewReplayCreateVkInstance(createInfo, allocator, pInstance, a.Result)
	hijack.Extras().Add(a.Extras().All()...)
	err := hijack.Mutate(ctx, s, b)

	if b == nil || err != nil {
		return err
	}

	// Call the replayRegisterVkInstance() synthetic API function.
	instance := a.PInstance.Read(ctx, a, s, b)
	return NewReplayRegisterVkInstance(instance).Mutate(ctx, s, b)
}

func (a *VkDestroyInstance) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	// Call the underlying vkDestroyInstance() and do the observation.
	err := a.mutate(ctx, s, b)
	if b == nil || err != nil {
		return err
	}
	// Call the replayUnregisterVkInstance() synthetic API function.
	return NewReplayUnregisterVkInstance(a.Instance).Mutate(ctx, s, b)
}

func (a *RecreateInstance) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pInstance := memory.Pointer(a.PInstance)
	hijack := NewVkCreateInstance(createInfo, allocator, pInstance, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreatePhysicalDevices) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	count := memory.Pointer(a.Count)
	pPhysicalDevices := memory.Pointer(a.PPhysicalDevices)
	hijack := NewVkEnumeratePhysicalDevices(a.Instance, count, pPhysicalDevices, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateDevice) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pDevice := memory.Pointer(a.PDevice)
	hijack := NewVkCreateDevice(a.PhysicalDevice, createInfo, allocator, pDevice, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}
func (a *RecreateQueue) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	pQueue := memory.Pointer(a.PQueue)
	hijack := NewVkGetDeviceQueue(a.Device, a.QueueFamilyIndex, a.QueueIndex, pQueue)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}
func (a *RecreateDeviceMemory) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	pAllocateInfo := memory.Pointer(a.PAllocateInfo)
	allocator := memory.Pointer{}
	pDeviceMemory := memory.Pointer(a.PMemory)
	ppData := memory.Pointer(a.PpData)
	hijack := NewVkAllocateMemory(a.Device, pAllocateInfo, allocator, pDeviceMemory, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	err := hijack.Mutate(ctx, s, b)
	if err != nil {
		return err
	}
	if a.MappedSize > 0 {
		memory := a.PMemory.Read(ctx, a, s, b)
		bind := NewVkMapMemory(a.Device, memory, a.MappedOffset, a.MappedSize, VkMemoryMapFlags(0),
			ppData, VkResult(0))
		bind.Extras().Add(a.Extras().All()...)
		err = bind.Mutate(ctx, s, b)
	}
	return err
}

func (a *RecreateVkCommandBuffer) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	pAllocateInfo := memory.Pointer(a.PAllocateInfo)
	pCommandBuffer := memory.Pointer(a.PCommandBuffer)
	pBeginInfo := memory.Pointer(a.PBeginInfo)
	hijack := NewVkAllocateCommandBuffers(a.Device, pAllocateInfo, pCommandBuffer, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	err := hijack.Mutate(ctx, s, b)
	if err != nil {
		return err
	}

	nullptr := memory.Pointer{}
	if pBeginInfo != nullptr {
		commandBuffer := a.PCommandBuffer.Read(ctx, a, s, b)
		begin := NewVkBeginCommandBuffer(commandBuffer, pBeginInfo, VkResult(0))
		begin.Extras().Add(a.Extras().All()...)
		err = begin.Mutate(ctx, s, b)
	}
	return err
}

func (a *RecreateVkEndCommandBuffer) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkEndCommandBuffer(a.CommandBuffer, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

////////////// Command Buffer Commands

func (a *RecreateUpdateBuffer) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdUpdateBuffer(a.CommandBuffer, a.DstBuffer, a.DstOffset, a.DataSize, memory.Pointer(a.PData))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdPipelineBarrier) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdPipelineBarrier(a.CommandBuffer,
		a.SrcStageMask,
		a.DstStageMask,
		a.DependencyFlags,
		a.MemoryBarrierCount,
		memory.Pointer(a.PMemoryBarriers),
		a.BufferMemoryBarrierCount,
		memory.Pointer(a.PBufferMemoryBarriers),
		a.ImageMemoryBarrierCount,
		memory.Pointer(a.PImageMemoryBarriers))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdCopyBuffer) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdCopyBuffer(a.CommandBuffer, a.SrcBuffer, a.DstBuffer, a.RegionCount, memory.Pointer(a.PRegions))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdResolveImage) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdResolveImage(a.CommandBuffer, a.SrcImage, a.SrcImageLayout, a.DstImage, a.DstImageLayout, a.RegionCount, memory.Pointer(a.PRegions))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdBeginRenderPass) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdBeginRenderPass(a.CommandBuffer, memory.Pointer(a.PRenderPassBegin), a.Contents)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdBindPipeline) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdBindPipeline(a.CommandBuffer, a.PipelineBindPoint, a.Pipeline)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdBindDescriptorSets) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdBindDescriptorSets(
		a.CommandBuffer,
		a.PipelineBindPoint,
		a.Layout,
		a.FirstSet,
		a.DescriptorSetCount,
		memory.Pointer(a.PDescriptorSets),
		a.DynamicOffsetCount,
		memory.Pointer(a.PDynamicOffsets))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateBindVertexBuffers) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdBindVertexBuffers(
		a.CommandBuffer,
		a.FirstBinding,
		a.BindingCount,
		memory.Pointer(a.PBuffers),
		memory.Pointer(a.POffsets))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdBindIndexBuffer) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdBindIndexBuffer(
		a.CommandBuffer,
		a.Buffer,
		a.Offset,
		a.IndexType)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateEndRenderPass) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdEndRenderPass(
		a.CommandBuffer)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdExecuteCommands) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdExecuteCommands(
		a.CommandBuffer,
		a.CommandBufferCount,
		memory.Pointer(a.PCommandBuffers),
	)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdDrawIndexed) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdDrawIndexed(
		a.CommandBuffer,
		a.IndexCount,
		a.InstanceCount,
		a.FirstIndex,
		a.VertexOffset,
		a.FirstInstance)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdDispatch) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdDispatch(
		a.CommandBuffer,
		a.X,
		a.Y,
		a.Z)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdDispatchIndirect) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdDispatchIndirect(
		a.CommandBuffer,
		a.Buffer,
		a.Offset)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdDrawIndirect) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdDrawIndirect(
		a.CommandBuffer,
		a.Buffer,
		a.Offset,
		a.DrawCount,
		a.Stride)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdDrawIndexedIndirect) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdDrawIndexedIndirect(
		a.CommandBuffer,
		a.Buffer,
		a.Offset,
		a.DrawCount,
		a.Stride)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdSetDepthBias) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdSetDepthBias(
		a.CommandBuffer,
		a.DepthBiasConstantFactor,
		a.DepthBiasClamp,
		a.DepthBiasSlopeFactor)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdSetBlendConstants) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdSetBlendConstants(
		a.CommandBuffer,
		a.BlendConstants)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdFillBuffer) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdFillBuffer(
		a.CommandBuffer,
		a.DstBuffer,
		a.DstOffset,
		a.Size,
		a.Data)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdSetLineWidth) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdSetLineWidth(
		a.CommandBuffer,
		a.LineWidth)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdCopyBufferToImage) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdCopyBufferToImage(
		a.CommandBuffer,
		a.SrcBuffer,
		a.DstImage,
		a.DstImageLayout,
		a.RegionCount,
		memory.Pointer(a.PRegions))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdCopyImageToBuffer) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdCopyImageToBuffer(
		a.CommandBuffer,
		a.SrcImage,
		a.SrcImageLayout,
		a.DstBuffer,
		a.RegionCount,
		memory.Pointer(a.PRegions))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdBlitImage) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdBlitImage(
		a.CommandBuffer,
		a.SrcImage,
		a.SrcImageLayout,
		a.DstImage,
		a.DstImageLayout,
		a.RegionCount,
		memory.Pointer(a.PRegions),
		a.Filter,
	)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdCopyImage) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdCopyImage(
		a.CommandBuffer,
		a.SrcImage,
		a.SrcImageLayout,
		a.DstImage,
		a.DstImageLayout,
		a.RegionCount,
		memory.Pointer(a.PRegions))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdPushConstants) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdPushConstants(
		a.CommandBuffer,
		a.Layout,
		a.StageFlags,
		a.Offset,
		a.Size,
		memory.Pointer(a.PValues))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdDraw) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdDraw(
		a.CommandBuffer,
		a.VertexCount,
		a.InstanceCount,
		a.FirstVertex,
		a.FirstInstance)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdSetScissor) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdSetScissor(
		a.CommandBuffer,
		a.FirstScissor,
		a.ScissorCount,
		memory.Pointer(a.PScissors))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdSetViewport) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdSetViewport(
		a.CommandBuffer,
		a.FirstViewport,
		a.ViewportCount,
		memory.Pointer(a.PViewports))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdBeginQuery) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdBeginQuery(
		a.CommandBuffer,
		a.QueryPool,
		a.Query,
		a.Flags)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdEndQuery) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdEndQuery(
		a.CommandBuffer,
		a.QueryPool,
		a.Query)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdClearAttachments) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdClearAttachments(
		a.CommandBuffer,
		a.AttachmentCount,
		memory.Pointer(a.PAttachments),
		a.RectCount,
		memory.Pointer(a.PRects),
	)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdClearColorImage) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdClearColorImage(
		a.CommandBuffer,
		a.Image,
		a.ImageLayout,
		memory.Pointer(a.PColor),
		a.RangeCount,
		memory.Pointer(a.PRanges),
	)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdClearDepthStencilImage) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdClearDepthStencilImage(
		a.CommandBuffer,
		a.Image,
		a.ImageLayout,
		memory.Pointer(a.PDepthStencil),
		a.RangeCount,
		memory.Pointer(a.PRanges),
	)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdResetQueryPool) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdResetQueryPool(
		a.CommandBuffer,
		a.QueryPool,
		a.FirstQuery,
		a.QueryCount)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCmdCopyQueryPoolResults) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCmdCopyQueryPoolResults(
		a.CommandBuffer,
		a.QueryPool,
		a.FirstQuery,
		a.QueryCount,
		a.DstBuffer,
		a.DstOffset,
		a.Stride,
		a.Flags,
	)
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreatePhysicalDeviceProperties) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkGetPhysicalDeviceQueueFamilyProperties(
		a.PhysicalDevice,
		memory.Pointer(a.PQueueFamilyPropertyCount),
		memory.Pointer(a.PQueueFamilyProperties))
	hijack.Extras().Add(a.Extras().All()...)
	if err := hijack.Mutate(ctx, s, b); err != nil {
		return err
	}
	memoryProperties := NewVkGetPhysicalDeviceMemoryProperties(
		a.PhysicalDevice,
		memory.Pointer(a.PMemoryProperties),
	)
	memoryProperties.Extras().Add(a.Extras().All()...)
	return memoryProperties.Mutate(ctx, s, b)
}

func (a *RecreateSemaphore) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pSemaphore := memory.Pointer(a.PSemaphore)

	hijack := NewVkCreateSemaphore(a.Device, createInfo, allocator, pSemaphore, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	if err := hijack.Mutate(ctx, s, b); err != nil {
		return err
	}
	if a.Signaled != VkBool32(0) {
		queue := findGraphicsAndComputeQueueForDevice(a.Device, s)
		semaphore := a.PSemaphore.Read(ctx, a, s, b)

		semaphores := atom.Must(atom.AllocData(ctx, s, semaphore))
		submitInfo := VkSubmitInfo{
			SType:                VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO,
			PNext:                NewVoidᶜᵖ(0),
			WaitSemaphoreCount:   0,
			PWaitSemaphores:      NewVkSemaphoreᶜᵖ(0),
			PWaitDstStageMask:    NewVkPipelineStageFlagsᶜᵖ(0),
			CommandBufferCount:   0,
			PCommandBuffers:      NewVkCommandBufferᶜᵖ(0),
			SignalSemaphoreCount: 1,
			PSignalSemaphores:    NewVkSemaphoreᶜᵖ(semaphores.Address()),
		}
		submitInfoData := atom.Must(atom.AllocData(ctx, s, submitInfo))

		err := NewVkQueueSubmit(
			queue,
			1,
			submitInfoData.Ptr(),
			VkFence(0),
			VkResult_VK_SUCCESS,
		).AddRead(
			submitInfoData.Data(),
		).AddRead(
			semaphores.Data(),
		).Mutate(ctx, s, b)

		semaphores.Free()
		submitInfoData.Free()
		return err
	}
	return nil

}

func (a *RecreateFence) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pFence := memory.Pointer(a.PFence)
	hijack := NewVkCreateFence(a.Device, createInfo, allocator, pFence, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateCommandPool) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pCommandPool := memory.Pointer(a.PCommandPool)
	hijack := NewVkCreateCommandPool(a.Device, createInfo, allocator, pCommandPool, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreatePipelineCache) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pPipelineCache := memory.Pointer(a.PPipelineCache)
	hijack := NewVkCreatePipelineCache(a.Device, createInfo, allocator, pPipelineCache, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateDescriptorSetLayout) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pDescriptorSetLayout := memory.Pointer(a.PSetLayout)
	hijack := NewVkCreateDescriptorSetLayout(a.Device, createInfo, allocator, pDescriptorSetLayout, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreatePipelineLayout) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pPipelineLayout := memory.Pointer(a.PPipelineLayout)
	hijack := NewVkCreatePipelineLayout(a.Device, createInfo, allocator, pPipelineLayout, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateRenderPass) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pRenderPass := memory.Pointer(a.PRenderPass)
	hijack := NewVkCreateRenderPass(a.Device, createInfo, allocator, pRenderPass, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateShaderModule) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pShaderModule := memory.Pointer(a.PShaderModule)
	hijack := NewVkCreateShaderModule(a.Device, createInfo, allocator, pShaderModule, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateDescriptorPool) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pDescriptorPool := memory.Pointer(a.PDescriptorPool)
	hijack := NewVkCreateDescriptorPool(a.Device, createInfo, allocator, pDescriptorPool, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateXCBSurfaceKHR) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pSurface := memory.Pointer(a.PSurface)
	hijack := NewVkCreateXcbSurfaceKHR(a.Instance, createInfo, allocator, pSurface, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateAndroidSurfaceKHR) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pSurface := memory.Pointer(a.PSurface)
	hijack := NewVkCreateAndroidSurfaceKHR(a.Instance, createInfo, allocator, pSurface, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateImageView) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pImageView := memory.Pointer(a.PImageView)
	hijack := NewVkCreateImageView(a.Device, createInfo, allocator, pImageView, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateSampler) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	hijack := NewVkCreateSampler(
		a.Device,
		memory.Pointer(a.PCreateInfo),
		memory.Pointer{},
		memory.Pointer(a.PSampler),
		VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateFramebuffer) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pFramebuffer := memory.Pointer(a.PFramebuffer)
	hijack := NewVkCreateFramebuffer(a.Device, createInfo, allocator, pFramebuffer, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateDescriptorSet) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	allocateInfo := memory.Pointer(a.PAllocateInfo)
	pDescriptorSet := memory.Pointer(a.PDescriptorSet)
	hijack := NewVkAllocateDescriptorSets(a.Device, allocateInfo, pDescriptorSet, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	if err := hijack.Mutate(ctx, s, b); err != nil {
		return err
	}
	pDescriptorSetWrites := memory.Pointer(a.PDescriptorWrites)
	write := NewVkUpdateDescriptorSets(a.Device, a.DescriptorWriteCount,
		pDescriptorSetWrites, 0, memory.Pointer{})
	return write.Mutate(ctx, s, b)
}

func (a *RecreateGraphicsPipeline) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	pCreateInfo := memory.Pointer(a.PCreateInfo)
	pPipeline := memory.Pointer(a.PPipeline)
	hijack := NewVkCreateGraphicsPipelines(a.Device, a.PipelineCache, uint32(1), pCreateInfo, memory.Pointer{}, pPipeline, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *RecreateComputePipeline) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	pCreateInfo := memory.Pointer(a.PCreateInfo)
	pPipeline := memory.Pointer(a.PPipeline)
	hijack := NewVkCreateComputePipelines(a.Device, a.PipelineCache, uint32(1), pCreateInfo, memory.Pointer{}, pPipeline, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func (a *VkCreateDevice) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	// Hijack VkCreateDevice's Mutate() method entirely with our ReplayCreateVkDevice's Mutate().
	// Similar to VkCreateInstance's Mutate() above.

	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer(a.PAllocator)
	pDevice := memory.Pointer(a.PDevice)
	hijack := NewReplayCreateVkDevice(a.PhysicalDevice, createInfo, allocator, pDevice, a.Result)
	hijack.Extras().Add(a.Extras().All()...)
	err := hijack.Mutate(ctx, s, b)

	if b == nil || err != nil {
		return err
	}

	// Call the replayRegisterVkDevice() synthetic API function.
	device := a.PDevice.Read(ctx, a, s, b)
	return NewReplayRegisterVkDevice(a.PhysicalDevice, device, createInfo).Mutate(ctx, s, b)
}

func (a *VkDestroyDevice) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	// Call the underlying vkDestroyDevice() and do the observation.
	err := a.mutate(ctx, s, b)
	if b == nil || err != nil {
		return err
	}
	// Call the replayUnregisterVkDevice() synthetic API function.
	return NewReplayUnregisterVkDevice(a.Device).Mutate(ctx, s, b)
}

func (a *VkAllocateCommandBuffers) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	// Call the underlying vkAllocateCommandBuffers() and do the observation.
	err := a.mutate(ctx, s, b)
	if b == nil || err != nil {
		return err
	}
	// Call the replayRegisterVkCommandBuffers() synthetic API function to link these command buffers to the device.
	count := a.PAllocateInfo.Read(ctx, a, s, b).CommandBufferCount
	cmdbuffers := memory.Pointer(a.PCommandBuffers)
	return NewReplayRegisterVkCommandBuffers(a.Device, count, cmdbuffers).Mutate(ctx, s, b)
}

func (a *VkFreeCommandBuffers) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	// Call the underlying vkFreeCommandBuffers() and do the observation.
	err := a.mutate(ctx, s, b)
	if b == nil || err != nil {
		return err
	}
	// Call the replayUnregisterVkCommandBuffers() synthetic API function to discard the link of these command buffers.
	count := a.CommandBufferCount
	cmdbuffers := memory.Pointer(a.PCommandBuffers)
	return NewReplayUnregisterVkCommandBuffers(count, cmdbuffers).Mutate(ctx, s, b)
}

func (a *VkCreateSwapchainKHR) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	// Call the underlying VkCreateSwapchainKHR() and do the observation
	err := a.mutate(ctx, s, b)
	if b == nil || err != nil {
		return err
	}
	pSwapchain := memory.Pointer(a.PSwapchain)
	return NewToggleVirtualSwapchainReturnAcquiredImage(pSwapchain).Mutate(ctx, s, b)
}

func (a *VkAcquireNextImageKHR) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory[memory.ApplicationPool])
	// Apply the write observation before having the replay device calling the vkAcquireNextImageKHR() command.
	// This is to pass the returned image index value captured in the trace, into the replay device to acquire for the specific image.
	o.ApplyWrites(s.Memory[memory.ApplicationPool])
	_ = a.PImageIndex.Slice(uint64(0), uint64(1), s).Index(uint64(0), s).Read(ctx, a, s, b)
	if b != nil {
		a.Call(ctx, s, b)
	}
	a.PImageIndex.Slice(uint64(0), uint64(1), s).Index(uint64(0), s).Write(ctx, a.PImageIndex.Slice(uint64(0), uint64(1), s).Index(uint64(0), s).Read(ctx, a, s, nil), a, s, b)
	_ = a.Result
	return nil
}

func (a *VkGetFenceStatus) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	err := a.mutate(ctx, s, b)
	if b == nil || err != nil {
		return err
	}

	return NewReplayGetFenceStatus(a.Device, a.Fence, a.Result, a.Result).Mutate(ctx, s, b)
}

func (a *ReplayAllocateImageMemory) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	if err := a.mutate(ctx, s, b); err != nil {
		return err
	}
	c := GetState(s)
	memory := a.PMemory.Slice(uint64(0), uint64(1), s).Index(uint64(0), s).Read(ctx, a, s, nil)
	imageObject := c.Images.Get(a.Image)
	imageWidth := imageObject.Layers.Get(0).Levels.Get(0).Width
	imageHeight := imageObject.Layers.Get(0).Levels.Get(0).Height
	imageFormat, err := getImageFormatFromVulkanFormat(imageObject.Info.Format)
	imageSize := VkDeviceSize(imageFormat.Size(int(imageWidth), int(imageHeight)))
	memoryObject := &DeviceMemoryObject{
		Device:          a.Device,
		VulkanHandle:    memory,
		AllocationSize:  imageSize,
		BoundObjects:    U64ːVkDeviceSizeᵐ{},
		MappedOffset:    VkDeviceSize(uint64(0)),
		MappedSize:      VkDeviceSize(uint64(0)),
		MappedLocation:  Voidᵖ{},
		MemoryTypeIndex: 0,
		Data:            MakeU8ˢ(uint64(imageSize), s)}
	c.DeviceMemories[memory] = memoryObject
	a.PMemory.Slice(uint64(0), uint64(1), s).Index(uint64(0), s).Write(ctx, memory, a, s, b)
	return err
}

func createEndCommandBufferAndQueueSubmit(ctx log.Context, s *gfxapi.State, b *builder.Builder, queue VkQueue, commandBuffer VkCommandBuffer) error {
	commandBuffers := atom.Must(atom.AllocData(ctx, s, commandBuffer))
	submitInfo := VkSubmitInfo{
		SType:                VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO,
		PNext:                NewVoidᶜᵖ(0),
		WaitSemaphoreCount:   0,
		PWaitSemaphores:      NewVkSemaphoreᶜᵖ(0),
		PWaitDstStageMask:    NewVkPipelineStageFlagsᶜᵖ(0),
		CommandBufferCount:   1,
		PCommandBuffers:      NewVkCommandBufferᶜᵖ(commandBuffers.Address()),
		SignalSemaphoreCount: 0,
		PSignalSemaphores:    NewVkSemaphoreᶜᵖ(0),
	}
	submitInfoData := atom.Must(atom.AllocData(ctx, s, submitInfo))

	if err := NewVkEndCommandBuffer(
		commandBuffer,
		VkResult_VK_SUCCESS,
	).Mutate(ctx, s, b); err != nil {
		return err
	}

	return NewVkQueueSubmit(
		queue,
		1,
		submitInfoData.Ptr(),
		VkFence(0),
		VkResult_VK_SUCCESS,
	).AddRead(
		submitInfoData.Data(),
	).AddRead(
		commandBuffers.Data(),
	).Mutate(ctx, s, b)
}

func createImageTransition(ctx log.Context, s *gfxapi.State, b *builder.Builder,
	srcLayout VkImageLayout, dstLayout VkImageLayout,
	image VkImage, aspectMask VkImageAspectFlags, commandBuffer VkCommandBuffer) error {

	allBits := uint32(VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT<<1) - 1
	imageObject := GetState(s).Images[image]

	imageBarrier := VkImageMemoryBarrier{
		SType:               VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		PNext:               NewVoidᶜᵖ(0),
		SrcAccessMask:       VkAccessFlags(allBits),
		DstAccessMask:       VkAccessFlags(allBits),
		NewLayout:           dstLayout,
		OldLayout:           srcLayout,
		SrcQueueFamilyIndex: 0xFFFFFFFF,
		DstQueueFamilyIndex: 0xFFFFFFFF,
		Image:               image,
		SubresourceRange: VkImageSubresourceRange{
			AspectMask:     aspectMask,
			BaseMipLevel:   0,
			LevelCount:     imageObject.Info.MipLevels,
			BaseArrayLayer: 0,
			LayerCount:     imageObject.Info.ArrayLayers,
		},
	}
	imageBarrierData := atom.Must(atom.AllocData(ctx, s, imageBarrier))

	transfer := NewVkCmdPipelineBarrier(
		commandBuffer,
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkDependencyFlags(0),
		0,
		memory.Pointer{},
		0,
		memory.Pointer{},
		1,
		imageBarrierData.Ptr(),
	).AddRead(
		imageBarrierData.Data(),
	)

	return transfer.Mutate(ctx, s, b)
}

func createAndBeginCommandBuffer(ctx log.Context, s *gfxapi.State, b *builder.Builder, device VkDevice, commandPool VkCommandPool) (VkCommandBuffer, error) {
	commandBufferAllocateInfo := VkCommandBufferAllocateInfo{
		SType:              VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		PNext:              NewVoidᶜᵖ(0),
		CommandPool:        commandPool,
		Level:              VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	}
	commandBufferAllocateInfoData := atom.Must(atom.AllocData(ctx, s, commandBufferAllocateInfo))
	commandBufferId := VkCommandBuffer(newUnusedID(true, func(x uint64) bool { _, ok := GetState(s).CommandBuffers[VkCommandBuffer(x)]; return ok }))
	commandBufferData := atom.Must(atom.AllocData(ctx, s, commandBufferId))

	// Data and info for Vulkan commands in command buffers
	beginCommandBufferInfo := VkCommandBufferBeginInfo{
		SType:            VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
		PNext:            NewVoidᶜᵖ(0),
		Flags:            VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT),
		PInheritanceInfo: NewVkCommandBufferInheritanceInfoᶜᵖ(0),
	}
	beginCommandBufferInfoData := atom.Must(atom.AllocData(ctx, s, beginCommandBufferInfo))

	if err := NewVkAllocateCommandBuffers(
		device,
		commandBufferAllocateInfoData.Ptr(),
		commandBufferData.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		commandBufferAllocateInfoData.Data(),
	).AddWrite(
		commandBufferData.Data(),
	).Mutate(ctx, s, b); err != nil {
		return commandBufferId, err
	}

	return commandBufferId, NewVkBeginCommandBuffer(
		commandBufferId,
		beginCommandBufferInfoData.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		beginCommandBufferInfoData.Data(),
	).Mutate(ctx, s, b)
}

func createAndBindSourceBuffer(ctx log.Context, s *gfxapi.State, b *builder.Builder, device VkDevice, size VkDeviceSize, memoryIndex uint32) (VkBuffer, VkDeviceMemory, error) {
	bufferCreateInfo := VkBufferCreateInfo{
		SType:                 VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
		PNext:                 NewVoidᶜᵖ(0),
		Flags:                 VkBufferCreateFlags(0),
		Size:                  size,
		Usage:                 VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT),
		SharingMode:           VkSharingMode_VK_SHARING_MODE_EXCLUSIVE,
		QueueFamilyIndexCount: 0,
		PQueueFamilyIndices:   NewU32ᶜᵖ(0),
	}

	bufferId := VkBuffer(newUnusedID(true, func(x uint64) bool { _, ok := GetState(s).Buffers[VkBuffer(x)]; return ok }))
	bufferAllocateInfoData := atom.Must(atom.AllocData(ctx, s, bufferCreateInfo))
	bufferData := atom.Must(atom.AllocData(ctx, s, bufferId))

	if err := NewVkCreateBuffer(
		device,
		bufferAllocateInfoData.Ptr(),
		memory.Pointer{},
		bufferData.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		bufferAllocateInfoData.Data(),
	).AddWrite(
		bufferData.Data(),
	).Mutate(ctx, s, b); err != nil {
		return VkBuffer(0), VkDeviceMemory(0), err
	}

	memoryAllocateInfo := VkMemoryAllocateInfo{
		SType:           VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
		PNext:           NewVoidᶜᵖ(0),
		AllocationSize:  size,
		MemoryTypeIndex: memoryIndex,
	}
	memoryId := VkDeviceMemory(newUnusedID(true, func(x uint64) bool { _, ok := GetState(s).DeviceMemories[VkDeviceMemory(x)]; return ok }))
	memoryAllocateInfoData := atom.Must(atom.AllocData(ctx, s, memoryAllocateInfo))
	memoryData := atom.Must(atom.AllocData(ctx, s, memoryId))

	if err := NewVkAllocateMemory(
		device,
		memoryAllocateInfoData.Ptr(),
		memory.Pointer{},
		memoryData.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		memoryAllocateInfoData.Data(),
	).AddWrite(
		memoryData.Data(),
	).Mutate(ctx, s, b); err != nil {
		return VkBuffer(0), VkDeviceMemory(0), err
	}

	if err := NewVkBindBufferMemory(
		device, bufferId, memoryId, VkDeviceSize(0), VkResult_VK_SUCCESS,
	).Mutate(ctx, s, b); err != nil {
		return VkBuffer(0), VkDeviceMemory(0), err
	}

	return bufferId, memoryId, nil
}

func (a *RecreateBufferView) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pBufferView := memory.Pointer(a.PBufferView)
	hijack := NewVkCreateBufferView(a.Device, createInfo, allocator, pBufferView, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	return hijack.Mutate(ctx, s, b)
}

func mapBufferMemory(ctx log.Context, s *gfxapi.State, b *builder.Builder, a atom.Atom, device VkDevice, size VkDeviceSize, memory VkDeviceMemory) (Voidᵖ, uint64, error) {
	at, err := s.Allocator.Alloc(uint64(size), 8)
	if err != nil {
		return NewVoidᵖ(0), at, err
	}
	mappedPointer := atom.Must(atom.AllocData(ctx, s, NewVoidᶜᵖ(at)))

	if err := NewVkMapMemory(
		device, memory, VkDeviceSize(0), size, VkMemoryMapFlags(0), mappedPointer.Ptr(), VkResult_VK_SUCCESS,
	).AddWrite(mappedPointer.Data()).Mutate(ctx, s, b); err != nil {
		return NewVoidᵖ(0), at, err
	}

	return NewVoidᵖᵖ(mappedPointer.Ptr().Address).Read(ctx, a, s, b), at, err
}

func flushBufferMemory(ctx log.Context, s *gfxapi.State, b *builder.Builder, device VkDevice, size VkDeviceSize, memory VkDeviceMemory, mapped U8ᵖ) error {
	flushRange := VkMappedMemoryRange{
		SType:  VkStructureType_VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE,
		PNext:  NewVoidᶜᵖ(0),
		Memory: memory,
		Offset: VkDeviceSize(0),
		Size:   VkDeviceSize(0xFFFFFFFFFFFFFFFF),
	}
	flushData := atom.Must(atom.AllocData(ctx, s, flushRange))
	slice := mapped.Slice(0, uint64(size), s)

	return NewVkFlushMappedMemoryRanges(
		device, uint32(1), flushData.Ptr(), VkResult_VK_SUCCESS,
	).AddRead(flushData.Data()).
		AddRead(slice.Range(s), slice.ResourceID(ctx, s)).Mutate(ctx, s, b)
}

func createBufferBarrier(ctx log.Context, s *gfxapi.State, b *builder.Builder, buffer VkBuffer, size VkDeviceSize, commandBuffer VkCommandBuffer) error {
	allBits := uint32(VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT<<1) - 1

	bufferBarrier := VkBufferMemoryBarrier{
		SType:               VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
		PNext:               NewVoidᶜᵖ(0),
		SrcAccessMask:       VkAccessFlags(allBits),
		DstAccessMask:       VkAccessFlags(allBits),
		SrcQueueFamilyIndex: 0xFFFFFFFF,
		DstQueueFamilyIndex: 0xFFFFFFFF,
		Buffer:              buffer,
		Offset:              VkDeviceSize(0),
		Size:                size,
	}
	bufferBarrierData := atom.Must(atom.AllocData(ctx, s, bufferBarrier))

	transfer := NewVkCmdPipelineBarrier(
		commandBuffer,
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkDependencyFlags(0),
		0,
		memory.Pointer{},
		1,
		bufferBarrierData.Ptr(),
		0,
		memory.Pointer{},
	).AddRead(
		bufferBarrierData.Data(),
	)

	return transfer.Mutate(ctx, s, b)
}

func createCommandPool(ctx log.Context, s *gfxapi.State, b *builder.Builder, queue VkQueue, device VkDevice) (VkCommandPool, error) {
	// Command pool and command buffer
	commandPoolId := VkCommandPool(newUnusedID(false, func(x uint64) bool { _, ok := GetState(s).CommandPools[VkCommandPool(x)]; return ok }))
	queueObject := GetState(s).Queues[queue]

	commandPoolCreateInfo := VkCommandPoolCreateInfo{
		SType:            VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
		PNext:            NewVoidᶜᵖ(0),
		Flags:            VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT),
		QueueFamilyIndex: queueObject.Family,
	}
	commandPoolCreateInfoData := atom.Must(atom.AllocData(ctx, s, commandPoolCreateInfo))
	commandPoolData := atom.Must(atom.AllocData(ctx, s, commandPoolId))

	return commandPoolId, NewVkCreateCommandPool(
		device,
		commandPoolCreateInfoData.Ptr(),
		memory.Pointer{},
		commandPoolData.Ptr(),
		VkResult_VK_SUCCESS).AddRead(
		commandPoolCreateInfoData.Data(),
	).AddWrite(
		commandPoolData.Data(),
	).Mutate(ctx, s, b)
}

func destroyCommandPool(ctx log.Context, s *gfxapi.State, b *builder.Builder, device VkDevice, commandPool VkCommandPool) error {
	return NewVkDestroyCommandPool(device, commandPool, memory.Pointer{}).Mutate(ctx, s, b)
}

func (a *RecreateImage) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pImage := memory.Pointer(a.PImage)
	hijack := NewVkCreateImage(a.Device, createInfo, allocator, pImage, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	if err := hijack.Mutate(ctx, s, b); err != nil {
		return err
	}
	return nil
}

func (a *RecreateBindAndFillImageMemory) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory[memory.ApplicationPool])

	if a.Memory != VkDeviceMemory(0) {
		if err := NewVkBindImageMemory(a.Device, a.Image, a.Memory, a.Offset, VkResult_VK_SUCCESS).Mutate(ctx, s, b); err != nil {
			return err
		}
	}

	imageObject := GetState(s).Images[a.Image]
	if a.LastBoundQueue != VkQueue(0) && a.LastLayout != VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED {
		queueObject := GetState(s).Queues[a.LastBoundQueue]
		device := queueObject.Device
		commandPool, err := createCommandPool(ctx, s, b, a.LastBoundQueue, device)
		if err != nil {
			return err
		}
		commandBuffer, err := createAndBeginCommandBuffer(ctx, s, b, device, commandPool)
		if err != nil {
			return err
		}

		bufferId := VkBuffer(0)
		memoryId := VkDeviceMemory(0)
		mem := uint64(0)
		srcLayout := VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED

		if a.Data != NewVoidᵖ(0) {
			imageInfo := imageObject.Info
			bufferId, memoryId, err = createAndBindSourceBuffer(ctx, s, b, device, a.DataSize, a.HostMemoryIndex)
			if err != nil {
				return err
			}
			mappedLocation := NewVoidᵖ(0)
			mappedLocation, mem, err = mapBufferMemory(ctx, s, b, a, device, a.DataSize, memoryId)
			if err != nil {
				return err
			}
			mappedChars := U8ᵖ(mappedLocation)
			dataP := U8ᵖ(a.Data)
			mappedChars.Slice(uint64(0), uint64(a.DataSize), s).Copy(ctx, dataP.Slice(uint64(0), uint64(a.DataSize), s), a, s, b)

			if err := flushBufferMemory(ctx, s, b, device, a.DataSize, memoryId, mappedChars); err != nil {
				return err
			}

			if err := createImageTransition(ctx, s, b,
				srcLayout,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				a.Image,
				imageObject.ImageAspect,
				commandBuffer); err != nil {
				return err
			}
			srcLayout = VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
			copies := []VkBufferImageCopy{}

			block_info, _ := subGetElementAndTexelBlockSize(ctx, a, nil, nil, nil, nil, imageInfo.Format)
			offset := VkDeviceSize(0)

			for i := uint32(0); i < imageInfo.MipLevels; i++ {
				width, _ := subRoundUpTo(ctx, a, nil, nil, nil, nil, imageInfo.Extent.Width, 1<<i)
				height, _ := subRoundUpTo(ctx, a, nil, nil, nil, nil, imageInfo.Extent.Height, 1<<i)
				depth, _ := subRoundUpTo(ctx, a, nil, nil, nil, nil, imageInfo.Extent.Depth, 1<<i)
				width_in_blocks, _ := subRoundUpTo(ctx, a, nil, nil, nil, nil, width, block_info.TexelBlockSize.Width)
				height_in_blocks, _ := subRoundUpTo(ctx, a, nil, nil, nil, nil, height, block_info.TexelBlockSize.Height)
				copies = append(copies, VkBufferImageCopy{
					BufferOffset:      offset,
					BufferRowLength:   0, // Tightly packed
					BufferImageHeight: 0, // Tightly packed
					ImageSubresource: VkImageSubresourceLayers{
						AspectMask:     imageObject.ImageAspect,
						MipLevel:       i,
						BaseArrayLayer: 0,
						LayerCount:     imageObject.Info.ArrayLayers,
					},
					ImageOffset: VkOffset3D{
						X: 0,
						Y: 0,
						Z: 0,
					},
					ImageExtent: VkExtent3D{
						Width:  width,
						Height: height,
						Depth:  depth,
					},
				})

				offset += VkDeviceSize(width_in_blocks * height_in_blocks * depth * block_info.ElementSize * imageObject.Info.ArrayLayers)
			}

			pointer := atom.Must(atom.AllocData(ctx, s, copies))

			copy := NewVkCmdCopyBufferToImage(commandBuffer, bufferId, a.Image,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, uint32(len(copies)), pointer.Ptr())
			copy.AddRead(pointer.Data())
			if err := copy.Mutate(ctx, s, b); err != nil {
				return err
			}

			pointer.Free()
		}

		if err := createImageTransition(ctx, s, b,
			srcLayout,
			a.LastLayout,
			a.Image,
			imageObject.ImageAspect,
			commandBuffer); err != nil {
			return err
		}
		if err := createEndCommandBufferAndQueueSubmit(ctx, s, b, a.LastBoundQueue, commandBuffer); err != nil {
			return err
		}
		if err := NewVkQueueWaitIdle(a.LastBoundQueue, VkResult_VK_SUCCESS).Mutate(ctx, s, b); err != nil {
			return err
		}
		if err := destroyCommandPool(ctx, s, b, device, commandPool); err != nil {
			return err
		}
		if bufferId != VkBuffer(0) {
			if err := NewVkDestroyBuffer(
				device, bufferId, memory.Pointer{},
			).Mutate(ctx, s, b); err != nil {
				return err
			}
			if err := NewVkFreeMemory(
				device, memoryId, memory.Pointer{},
			).Mutate(ctx, s, b); err != nil {
				return err
			}
			s.Allocator.Free(mem)
		}

	}

	return nil
}

func (a *RecreateBuffer) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	// ApplyReads() is necessary only because we need to access the Read
	// observation data prior to calling the VkCreateBuffer's Mutate().
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory[memory.ApplicationPool])

	createInfo := a.PCreateInfo.Read(ctx, a, s, b)
	createInfo.Usage = createInfo.Usage | VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT)
	createInfoData := atom.Must(atom.AllocData(ctx, s, createInfo))
	defer createInfoData.Free()
	allocator := memory.Pointer{}
	pBuffer := memory.Pointer(a.PBuffer)
	hijack := NewVkCreateBuffer(a.Device, createInfoData.Ptr(), allocator, pBuffer, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	hijack.AddRead(createInfoData.Data())
	if err := hijack.Mutate(ctx, s, b); err != nil {
		return err
	}
	return nil
}

func (a *RecreateBindAndFillBufferMemory) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory[memory.ApplicationPool])

	bufferObject := GetState(s).Buffers[a.Buffer]
	bufferInfo := bufferObject.Info

	if a.Memory != VkDeviceMemory(0) {
		if err := NewVkBindBufferMemory(a.Device, a.Buffer, a.Memory, a.Offset, VkResult_VK_SUCCESS).Mutate(ctx, s, b); err != nil {
			return err
		}
	}

	// If we have data to fill this buffer with:
	if a.Data != NewVoidᵖ(0) {
		queue := a.LastBoundQueue
		queueObject := GetState(s).Queues[queue]
		device := queueObject.Device

		bufferId, memoryId, err := createAndBindSourceBuffer(ctx, s, b, device, bufferInfo.Size, a.HostBufferMemoryIndex)
		if err != nil {
			return err
		}
		mappedLocation, mem, err := mapBufferMemory(ctx, s, b, a, device, bufferInfo.Size, memoryId)
		if err != nil {
			return err
		}
		mappedChars := U8ᵖ(mappedLocation)
		dataP := U8ᵖ(a.Data)
		mappedChars.Slice(uint64(0), uint64(bufferInfo.Size), s).Copy(ctx, dataP.Slice(uint64(0), uint64(bufferInfo.Size), s), a, s, b)

		if err := flushBufferMemory(ctx, s, b, device, bufferInfo.Size, memoryId, mappedChars); err != nil {
			return err
		}

		commandPool, err := createCommandPool(ctx, s, b, queue, device)
		if err != nil {
			return err
		}

		commandBuffer, err := createAndBeginCommandBuffer(ctx, s, b, device, commandPool)
		if err != nil {
			return err
		}

		bufferCopy := VkBufferCopy{
			SrcOffset: VkDeviceSize(0),
			DstOffset: VkDeviceSize(0),
			Size:      bufferInfo.Size,
		}
		bufferData := atom.Must(atom.AllocData(ctx, s, bufferCopy))
		if err := NewVkCmdCopyBuffer(commandBuffer, bufferId, a.Buffer, 1, bufferData.Ptr()).
			AddRead(bufferData.Data()).Mutate(ctx, s, b); err != nil {
			return err
		}

		if err := createBufferBarrier(ctx, s, b, bufferId, bufferInfo.Size, commandBuffer); err != nil {
			return err
		}

		if err := createEndCommandBufferAndQueueSubmit(ctx, s, b, queue, commandBuffer); err != nil {
			return err
		}
		if err := NewVkQueueWaitIdle(queue, VkResult_VK_SUCCESS).Mutate(ctx, s, b); err != nil {
			return err
		}
		if err := destroyCommandPool(ctx, s, b, device, commandPool); err != nil {
			return err
		}
		if err := NewVkDestroyBuffer(
			device, bufferId, memory.Pointer{},
		).Mutate(ctx, s, b); err != nil {
			return err
		}
		if err := NewVkFreeMemory(
			device, memoryId, memory.Pointer{},
		).Mutate(ctx, s, b); err != nil {
			return err
		}
		s.Allocator.Free(mem)
	}
	return nil
}

// Returns a queue capable of graphics and compute operations if it could be
// found, a compute only queue or copy queue will be returned if it could not
// be found
func findGraphicsAndComputeQueueForDevice(device VkDevice, s *gfxapi.State) VkQueue {
	c := GetState(s)
	backupQueue := VkQueue(0)
	backupQueueFlags := uint32(0)
	for _, v := range c.Queues {
		if v.Device == device {
			family := c.PhysicalDevices[c.Devices[device].PhysicalDevice].QueueFamilyProperties[v.Family]
			expected := uint32(VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT) | uint32(VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT)
			if (uint32(family.QueueFlags) & expected) == expected {
				return v.VulkanHandle
			}
			if (uint32(family.QueueFlags) & uint32(VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT)) != 0 {
				backupQueue = v.VulkanHandle
				backupQueueFlags = uint32(family.QueueFlags)
			} else if backupQueueFlags == 0 {
				backupQueue = v.VulkanHandle
				backupQueueFlags = uint32(family.QueueFlags)
			}
		}
	}
	return backupQueue
}

func (a *RecreateQueryPool) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pQueryPool := memory.Pointer(a.PPool)

	hijack := NewVkCreateQueryPool(a.Device, createInfo, allocator, pQueryPool, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	if err := hijack.Mutate(ctx, s, b); err != nil {
		return err
	}

	createInfoObject := a.PCreateInfo.Read(ctx, a, s, b)
	queryStates := a.PQueryStatuses.Slice(0, uint64(createInfoObject.QueryCount), s).Read(ctx, a, s, b)
	pool := a.PPool.Read(ctx, a, s, b)

	anyActive := false
	for i := uint32(0); i < createInfoObject.QueryCount; i++ {
		if queryStates[i] != QueryStatus_QUERY_STATUS_INACTIVE {
			anyActive = true
			break
		}
	}

	if !anyActive {
		return nil
	}

	queue := findGraphicsAndComputeQueueForDevice(a.Device, s)
	commandPool, err := createCommandPool(ctx, s, b, queue, a.Device)
	if err != nil {
		return err
	}
	commandBuffer, err := createAndBeginCommandBuffer(ctx, s, b, a.Device, commandPool)
	if err != nil {
		return err
	}

	for i := uint32(0); i < createInfoObject.QueryCount; i++ {
		if queryStates[i] != QueryStatus_QUERY_STATUS_INACTIVE {
			if err := NewVkCmdBeginQuery(commandBuffer,
				pool, i, VkQueryControlFlags(0)).Mutate(ctx, s, b); err != nil {
				return err
			}

			if queryStates[i] == QueryStatus_QUERY_STATUS_COMPLETE {
				if err := NewVkCmdEndQuery(commandBuffer,
					pool, i).Mutate(ctx, s, b); err != nil {
					return err
				}
			}
		}
	}

	if err := createEndCommandBufferAndQueueSubmit(ctx, s, b, queue, commandBuffer); err != nil {
		return err
	}
	if err := NewVkQueueWaitIdle(queue, VkResult_VK_SUCCESS).Mutate(ctx, s, b); err != nil {
		return err
	}
	if err := destroyCommandPool(ctx, s, b, a.Device, commandPool); err != nil {
		return err
	}

	return nil

}

func (a *RecreateSwapchain) Mutate(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
	createInfo := memory.Pointer(a.PCreateInfo)
	allocator := memory.Pointer{}
	pSwapchain := memory.Pointer(a.PSwapchain)
	hijack := NewVkCreateSwapchainKHR(a.Device, createInfo, allocator, pSwapchain, VkResult(0))
	hijack.Extras().Add(a.Extras().All()...)
	if err := hijack.Mutate(ctx, s, b); err != nil {
		return err
	}
	pSwapchainImages := memory.Pointer(a.PSwapchainImages)
	swapchain := a.PSwapchain.Read(ctx, a, s, b)
	createInfoData := a.PCreateInfo.Read(ctx, a, s, b)
	swapchainCountData := atom.Must(atom.AllocData(ctx, s, createInfoData.MinImageCount))

	getImages := NewVkGetSwapchainImagesKHR(a.Device, swapchain, swapchainCountData.Ptr(), pSwapchainImages, VkResult(0))
	getImages.Extras().Add(a.Extras().All()...)
	getImages.AddRead(swapchainCountData.Data()).AddWrite(swapchainCountData.Data())
	if err := getImages.Mutate(ctx, s, b); err != nil {
		return err
	}

	images := a.PSwapchainImages.Slice(0, uint64(createInfoData.MinImageCount), s).Read(ctx, a, s, b)
	imageLayouts := a.PSwapchainLayouts.Slice(0, uint64(createInfoData.MinImageCount), s).Read(ctx, a, s, b)
	boundQueues := a.PInitialQueues.Slice(0, uint64(createInfoData.MinImageCount), s).Read(ctx, a, s, b)
	for i := 0; i < int(createInfoData.MinImageCount); i++ {
		imageObject := GetState(s).Images[images[i]]
		if boundQueues[i] != VkQueue(0) && imageLayouts[i] != VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED {
			queue := boundQueues[i]
			queueObject := GetState(s).Queues[queue]
			device := queueObject.Device
			commandPool, err := createCommandPool(ctx, s, b, queue, device)
			if err != nil {
				return err
			}
			commandBuffer, err := createAndBeginCommandBuffer(ctx, s, b, device, commandPool)
			if err != nil {
				return err
			}
			if err := createImageTransition(ctx, s, b,
				VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
				imageLayouts[i],
				imageObject.VulkanHandle,
				imageObject.ImageAspect,
				commandBuffer); err != nil {
				return err
			}
			if err := createEndCommandBufferAndQueueSubmit(ctx, s, b, queue, commandBuffer); err != nil {
				return err
			}
			if err := NewVkQueueWaitIdle(queue, VkResult_VK_SUCCESS).Mutate(ctx, s, b); err != nil {
				return err
			}
			if err := destroyCommandPool(ctx, s, b, device, commandPool); err != nil {
				return err
			}
		}
	}

	return nil
}
