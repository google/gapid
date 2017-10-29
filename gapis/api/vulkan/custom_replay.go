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
	"strings"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
)

func (i VkInstance) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkPhysicalDevice) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDevice) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkQueue) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkCommandBuffer) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkSemaphore) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkFence) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDeviceMemory) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkBuffer) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkImage) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkEvent) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkQueryPool) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkBufferView) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkImageView) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkShaderModule) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkPipelineCache) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkPipelineLayout) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkRenderPass) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkPipeline) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDescriptorSetLayout) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkSampler) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDescriptorPool) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDescriptorSet) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkFramebuffer) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkCommandPool) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkSurfaceKHR) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkSwapchainKHR) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDisplayKHR) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDisplayModeKHR) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (i VkDebugReportCallbackEXT) remap(api.Cmd, *api.GlobalState) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = i, true
	}
	return
}

func (a *VkCreateInstance) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	cb := CommandBuilder{Thread: a.thread}
	// Hijack VkCreateInstance's Mutate() method entirely with our ReplayCreateVkInstance's Mutate().

	// As long as we guarantee that the synthetic replayCreateVkInstance API function has the same
	// logic as the real vkCreateInstance API function, we can do observation correctly. Additionally,
	// ReplayCreateVkInstance's Mutate() will invoke our custom wrapper function replayCreateVkInstance()
	// in vulkan_gfx_api_extras.cpp, which modifies VkInstanceCreateInfo to enable virtual swapchain
	// layer before delegating the real work back to the normal flow.

	hijack := cb.ReplayCreateVkInstance(a.PCreateInfo, a.PAllocator, a.PInstance, a.Result)
	hijack.Extras().MustClone(a.Extras().All()...)
	err := hijack.Mutate(ctx, id, s, b)

	if b == nil || err != nil {
		return err
	}

	// Call the replayRegisterVkInstance() synthetic API function.
	instance := a.PInstance.MustRead(ctx, a, s, b)
	return cb.ReplayRegisterVkInstance(instance).Mutate(ctx, id, s, b)
}

func (a *VkDestroyInstance) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	cb := CommandBuilder{Thread: a.thread}
	// Call the underlying vkDestroyInstance() and do the observation.
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}
	// Call the replayUnregisterVkInstance() synthetic API function.
	return cb.ReplayUnregisterVkInstance(a.Instance).Mutate(ctx, id, s, b)
}

func EnterRecreate(ctx context.Context, s *api.GlobalState) func() {
	GetState(s).IsRebuilding = true
	return func() { GetState(s).IsRebuilding = false }
}

func (a *RecreateInstance) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateInstance(a.PCreateInfo, allocator, a.PInstance, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateState) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	return nil
}

func (a *RecreatePhysicalDevices) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	hijack := cb.VkEnumeratePhysicalDevices(a.Instance, a.Count, a.PPhysicalDevices, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateDevice) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateDevice(a.PhysicalDevice, a.PCreateInfo, allocator, a.PDevice, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}
func (a *RecreateQueue) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	hijack := cb.VkGetDeviceQueue(a.Device, a.QueueFamilyIndex, a.QueueIndex, a.PQueue)
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}
func (a *RecreateDeviceMemory) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkAllocateMemory(a.Device, a.PAllocateInfo, allocator, a.PMemory, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	err := hijack.Mutate(ctx, id, s, b)
	if err != nil {
		return err
	}
	if a.MappedSize > 0 {
		memory := a.PMemory.MustRead(ctx, a, s, b)
		bind := cb.VkMapMemory(a.Device, memory, a.MappedOffset, a.MappedSize, VkMemoryMapFlags(0),
			a.PpData, VkResult(0))
		bind.Extras().MustClone(a.Extras().All()...)
		err = bind.Mutate(ctx, id, s, b)
	}
	return err
}

func (a *RecreateAndBeginCommandBuffer) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	hijack := cb.VkAllocateCommandBuffers(a.Device, a.PAllocateInfo, a.PCommandBuffer, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	err := hijack.Mutate(ctx, id, s, b)
	if err != nil {
		return err
	}

	if !a.PBeginInfo.IsNullptr() {
		commandBuffer := a.PCommandBuffer.MustRead(ctx, a, s, b)
		begin := cb.VkBeginCommandBuffer(commandBuffer, a.PBeginInfo, VkResult(0))
		begin.Extras().MustClone(a.Extras().All()...)
		err = begin.Mutate(ctx, id, s, b)
	}
	return err
}

func (a *RecreateEndCommandBuffer) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	hijack := cb.VkEndCommandBuffer(a.CommandBuffer, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreatePhysicalDeviceProperties) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	hijack := cb.VkGetPhysicalDeviceQueueFamilyProperties(
		a.PhysicalDevice,
		a.PQueueFamilyPropertyCount,
		a.PQueueFamilyProperties)
	hijack.Extras().MustClone(a.Extras().All()...)
	if err := hijack.Mutate(ctx, id, s, b); err != nil {
		return err
	}
	memoryProperties := cb.VkGetPhysicalDeviceMemoryProperties(
		a.PhysicalDevice,
		a.PMemoryProperties,
	)
	memoryProperties.Extras().MustClone(a.Extras().All()...)
	return memoryProperties.Mutate(ctx, id, s, b)
}

func (a *RecreateSemaphore) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateSemaphore(a.Device, a.PCreateInfo, allocator, a.PSemaphore, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	if err := hijack.Mutate(ctx, id, s, b); err != nil {
		return err
	}
	if a.Signaled != VkBool32(0) {
		queue := findGraphicsAndComputeQueueForDevice(a.Device, s)
		semaphore := a.PSemaphore.MustRead(ctx, a, s, b)

		semaphores := s.AllocDataOrPanic(ctx, semaphore)
		defer semaphores.Free()
		submitInfo := VkSubmitInfo{
			SType:                VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO,
			PNext:                NewVoidᶜᵖ(memory.Nullptr),
			WaitSemaphoreCount:   0,
			PWaitSemaphores:      NewVkSemaphoreᶜᵖ(memory.Nullptr),
			PWaitDstStageMask:    NewVkPipelineStageFlagsᶜᵖ(memory.Nullptr),
			CommandBufferCount:   0,
			PCommandBuffers:      NewVkCommandBufferᶜᵖ(memory.Nullptr),
			SignalSemaphoreCount: 1,
			PSignalSemaphores:    VkSemaphoreᶜᵖ{semaphores.Address(), memory.ApplicationPool},
		}
		submitInfoData := s.AllocDataOrPanic(ctx, submitInfo)
		defer submitInfoData.Free()

		err := cb.VkQueueSubmit(
			queue,
			1,
			submitInfoData.Ptr(),
			VkFence(0),
			VkResult_VK_SUCCESS,
		).AddRead(
			submitInfoData.Data(),
		).AddRead(
			semaphores.Data(),
		).Mutate(ctx, id, s, b)

		return err
	}
	return nil

}

func (a *RecreateFence) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateFence(a.Device, a.PCreateInfo, allocator, a.PFence, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateEvent) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr

	hijack := cb.VkCreateEvent(a.Device, a.PCreateInfo, allocator, a.PEvent, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	if err := hijack.Mutate(ctx, id, s, b); err != nil {
		return err
	}
	if a.Signaled != VkBool32(0) {
		event := a.PEvent.MustRead(ctx, a, s, b)
		err := cb.VkSetEvent(
			a.Device,
			event,
			VkResult_VK_SUCCESS,
		).Mutate(ctx, id, s, b)

		return err
	}
	return nil
}

func (a *RecreateCommandPool) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateCommandPool(a.Device, a.PCreateInfo, allocator, a.PCommandPool, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreatePipelineCache) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreatePipelineCache(a.Device, a.PCreateInfo, allocator, a.PPipelineCache, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateDescriptorSetLayout) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateDescriptorSetLayout(a.Device, a.PCreateInfo, allocator, a.PSetLayout, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreatePipelineLayout) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreatePipelineLayout(a.Device, a.PCreateInfo, allocator, a.PPipelineLayout, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateRenderPass) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateRenderPass(a.Device, a.PCreateInfo, allocator, a.PRenderPass, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateShaderModule) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateShaderModule(a.Device, a.PCreateInfo, allocator, a.PShaderModule, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateDestroyShaderModule) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkDestroyShaderModule(a.Device, a.ShaderModule, allocator)
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateDestroyRenderPass) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkDestroyRenderPass(a.Device, a.RenderPass, allocator)
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateDescriptorPool) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateDescriptorPool(a.Device, a.PCreateInfo, allocator, a.PDescriptorPool, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateXCBSurfaceKHR) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateXcbSurfaceKHR(a.Instance, a.PCreateInfo, allocator, a.PSurface, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateXlibSurfaceKHR) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateXlibSurfaceKHR(a.Instance, a.PCreateInfo, allocator, a.PSurface, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateWaylandSurfaceKHR) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateWaylandSurfaceKHR(a.Instance, a.PCreateInfo, allocator, a.PSurface, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateMirSurfaceKHR) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateMirSurfaceKHR(a.Instance, a.PCreateInfo, allocator, a.PSurface, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateAndroidSurfaceKHR) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateAndroidSurfaceKHR(a.Instance, a.PCreateInfo, allocator, a.PSurface, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateImageView) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateImageView(a.Device, a.PCreateInfo, allocator, a.PImageView, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateSampler) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	hijack := cb.VkCreateSampler(
		a.Device,
		a.PCreateInfo,
		memory.Nullptr,
		a.PSampler,
		VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateFramebuffer) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateFramebuffer(a.Device, a.PCreateInfo, allocator, a.PFramebuffer, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateDescriptorSet) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	hijack := cb.VkAllocateDescriptorSets(a.Device, a.PAllocateInfo, a.PDescriptorSet, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	if err := hijack.Mutate(ctx, id, s, b); err != nil {
		return err
	}
	write := cb.VkUpdateDescriptorSets(a.Device, a.DescriptorWriteCount,
		a.PDescriptorWrites, 0, memory.Nullptr)
	return write.Mutate(ctx, id, s, b)
}

func (a *RecreateGraphicsPipeline) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	hijack := cb.VkCreateGraphicsPipelines(a.Device, a.PipelineCache, uint32(1), a.PCreateInfo, memory.Nullptr, a.PPipeline, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *RecreateComputePipeline) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	hijack := cb.VkCreateComputePipelines(a.Device, a.PipelineCache, uint32(1), a.PCreateInfo, memory.Nullptr, a.PPipeline, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func (a *VkCreateDevice) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	// Hijack VkCreateDevice's Mutate() method entirely with our
	// ReplayCreateVkDevice's Mutate(). Similar to VkCreateInstance's Mutate()
	// above.
	// And we need to strip off the VK_EXT_debug_marker extension name when
	// building instructions for replay.
	createInfoPtr := a.PCreateInfo
	allocated := []*api.AllocResult{}
	if b != nil {
		a.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
		createInfo := a.PCreateInfo.MustRead(ctx, a, s, nil)
		defer func() {
			for _, d := range allocated {
				d.Free()
			}
		}()
		extensionCount := uint64(createInfo.EnabledExtensionCount)
		newExtensionNames := []memory.Pointer{}
		for _, e := range createInfo.PpEnabledExtensionNames.Slice(0, extensionCount, s.MemoryLayout).MustRead(ctx, a, s, nil) {
			extensionName := string(memory.CharToBytes(e.StringSlice(ctx, s).MustRead(ctx, a, s, nil)))
			if !strings.Contains(extensionName, "VK_EXT_debug_marker") {
				nameSliceData := s.AllocDataOrPanic(ctx, extensionName)
				allocated = append(allocated, &nameSliceData)
				newExtensionNames = append(newExtensionNames, nameSliceData.Ptr())
			}
		}
		new_extensionNamesData := s.AllocDataOrPanic(ctx, newExtensionNames)
		allocated = append(allocated, &new_extensionNamesData)
		createInfo.EnabledExtensionCount = uint32(len(newExtensionNames))
		createInfo.PpEnabledExtensionNames = NewCharᶜᵖᶜᵖ(new_extensionNamesData.Ptr())

		newCreateInfoData := s.AllocDataOrPanic(ctx, createInfo)
		allocated = append(allocated, &newCreateInfoData)
		createInfoPtr = NewVkDeviceCreateInfoᶜᵖ(newCreateInfoData.Ptr())
	}

	cb := CommandBuilder{Thread: a.thread}
	hijack := cb.ReplayCreateVkDevice(a.PhysicalDevice, createInfoPtr, a.PAllocator, a.PDevice, a.Result)
	hijack.Extras().MustClone(a.Extras().All()...)

	if b != nil {
		for _, d := range allocated {
			hijack.AddRead(d.Data())
		}
	}

	err := hijack.Mutate(ctx, id, s, b)

	if b == nil || err != nil {
		return err
	}

	// Call the replayRegisterVkDevice() synthetic API function.
	device := a.PDevice.MustRead(ctx, a, s, b)
	return cb.ReplayRegisterVkDevice(a.PhysicalDevice, device, a.PCreateInfo).Mutate(ctx, id, s, b)
}

func (a *VkDestroyDevice) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	// Call the underlying vkDestroyDevice() and do the observation.
	cb := CommandBuilder{Thread: a.thread}
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}
	// Call the replayUnregisterVkDevice() synthetic API function.
	return cb.ReplayUnregisterVkDevice(a.Device).Mutate(ctx, id, s, b)
}

func (a *VkAllocateCommandBuffers) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	// Call the underlying vkAllocateCommandBuffers() and do the observation.
	cb := CommandBuilder{Thread: a.thread}
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}
	// Call the replayRegisterVkCommandBuffers() synthetic API function to link these command buffers to the device.
	count := a.PAllocateInfo.MustRead(ctx, a, s, b).CommandBufferCount
	return cb.ReplayRegisterVkCommandBuffers(a.Device, count, a.PCommandBuffers).Mutate(ctx, id, s, b)
}

func (a *VkFreeCommandBuffers) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	// Call the underlying vkFreeCommandBuffers() and do the observation.
	cb := CommandBuilder{Thread: a.thread}
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}
	// Call the replayUnregisterVkCommandBuffers() synthetic API function to discard the link of these command buffers.
	count := a.CommandBufferCount
	return cb.ReplayUnregisterVkCommandBuffers(count, a.PCommandBuffers).Mutate(ctx, id, s, b)
}

func (a *VkCreateSwapchainKHR) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	// Call the underlying VkCreateSwapchainKHR() and do the observation
	cb := CommandBuilder{Thread: a.thread}
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}
	return cb.ToggleVirtualSwapchainReturnAcquiredImage(a.PSwapchain).Mutate(ctx, id, s, b)
}

func (a *VkAcquireNextImageKHR) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	l := s.MemoryLayout
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory.ApplicationPool())
	// Apply the write observation before having the replay device calling the vkAcquireNextImageKHR() command.
	// This is to pass the returned image index value captured in the trace, into the replay device to acquire for the specific image.
	o.ApplyWrites(s.Memory.ApplicationPool())
	_ = a.PImageIndex.Slice(uint64(0), uint64(1), l).Index(uint64(0), l).MustRead(ctx, a, s, b)
	if b != nil {
		a.Call(ctx, s, b)
	}
	a.PImageIndex.Slice(uint64(0), uint64(1), l).Index(uint64(0), l).Write(ctx, a.PImageIndex.Slice(uint64(0), uint64(1), l).Index(uint64(0), l).MustRead(ctx, a, s, nil), a, s, b)
	_ = a.Result
	if a.Semaphore != VkSemaphore(0) {
		GetState(s).Semaphores.Get(a.Semaphore).Signaled = true
	}
	if a.Fence != VkFence(0) {
		GetState(s).Fences.Get(a.Fence).Signaled = true
	}
	return nil
}

func (a *VkGetFenceStatus) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	cb := CommandBuilder{Thread: a.thread}
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}

	return cb.ReplayGetFenceStatus(a.Device, a.Fence, a.Result, a.Result).Mutate(ctx, id, s, b)
}

func (a *VkGetEventStatus) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	cb := CommandBuilder{Thread: a.thread}
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}
	var wait bool
	switch a.Result {
	case VkResult_VK_EVENT_SET:
		wait = GetState(s).Events.Get(a.Event).Signaled == true
	case VkResult_VK_EVENT_RESET:
		wait = GetState(s).Events.Get(a.Event).Signaled == false
	default:
		wait = false
	}

	return cb.ReplayGetEventStatus(a.Device, a.Event, a.Result, wait, a.Result).Mutate(ctx, id, s, b)
}

func (a *RecreateDebugMarkerSetObjectNameEXT) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory.ApplicationPool())
	nameInfo := a.PNameInfo.MustRead(ctx, a, s, nil)
	err := subSetDebugMarkerObjectName(ctx, a, id, o, s, GetState(s), a.thread, nil, nameInfo)
	if err != nil {
		return err
	}
	return a.mutate(ctx, id, s, b)
}

func (a *RecreateDebugMarkerSetObjectTagEXT) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory.ApplicationPool())
	tagInfo := a.PTagInfo.MustRead(ctx, a, s, nil)
	err := subSetDebugMarkerObjectTag(ctx, a, id, o, s, GetState(s), a.thread, nil, tagInfo)
	if err != nil {
		return err
	}
	return a.mutate(ctx, id, s, b)
}

func (a *ReplayAllocateImageMemory) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	if err := a.mutate(ctx, id, s, b); err != nil {
		return err
	}
	l := s.MemoryLayout
	c := GetState(s)
	memory := a.PMemory.Slice(uint64(0), uint64(1), l).Index(uint64(0), l).MustRead(ctx, a, s, nil)
	imageObject := c.Images.Get(a.Image)
	imageWidth := imageObject.Layers.Get(0).Levels.Get(0).Width
	imageHeight := imageObject.Layers.Get(0).Levels.Get(0).Height
	imageFormat, err := getImageFormatFromVulkanFormat(imageObject.Info.Format)
	imageSize := VkDeviceSize(imageFormat.Size(int(imageWidth), int(imageHeight), 1))
	memoryObject := &DeviceMemoryObject{
		Device:          a.Device,
		VulkanHandle:    memory,
		AllocationSize:  imageSize,
		BoundObjects:    NewU64ːVkDeviceSizeᵐ(),
		MappedOffset:    VkDeviceSize(uint64(0)),
		MappedSize:      VkDeviceSize(uint64(0)),
		MappedLocation:  Voidᵖ{},
		MemoryTypeIndex: 0,
		Data:            MakeU8ˢ(uint64(imageSize), s)}
	c.DeviceMemories.Set(memory, memoryObject)
	a.PMemory.Slice(uint64(0), uint64(1), l).Index(uint64(0), l).Write(ctx, memory, a, s, b)
	return err
}

func createEndCommandBufferAndQueueSubmit(ctx context.Context, id api.CmdID, cb CommandBuilder, s *api.GlobalState, b *builder.Builder, queue VkQueue, commandBuffer VkCommandBuffer) error {
	commandBuffers := s.AllocDataOrPanic(ctx, commandBuffer)
	defer commandBuffers.Free()
	submitInfo := VkSubmitInfo{
		SType:                VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO,
		PNext:                NewVoidᶜᵖ(memory.Nullptr),
		WaitSemaphoreCount:   0,
		PWaitSemaphores:      NewVkSemaphoreᶜᵖ(memory.Nullptr),
		PWaitDstStageMask:    NewVkPipelineStageFlagsᶜᵖ(memory.Nullptr),
		CommandBufferCount:   1,
		PCommandBuffers:      VkCommandBufferᶜᵖ{commandBuffers.Address(), memory.ApplicationPool},
		SignalSemaphoreCount: 0,
		PSignalSemaphores:    NewVkSemaphoreᶜᵖ(memory.Nullptr),
	}
	submitInfoData := s.AllocDataOrPanic(ctx, submitInfo)
	defer submitInfoData.Free()

	if err := cb.VkEndCommandBuffer(
		commandBuffer,
		VkResult_VK_SUCCESS,
	).Mutate(ctx, id, s, b); err != nil {
		return err
	}

	return cb.VkQueueSubmit(
		queue,
		1,
		submitInfoData.Ptr(),
		VkFence(0),
		VkResult_VK_SUCCESS,
	).AddRead(
		submitInfoData.Data(),
	).AddRead(
		commandBuffers.Data(),
	).Mutate(ctx, id, s, b)
}

func createImageTransition(ctx context.Context, id api.CmdID, cb CommandBuilder, s *api.GlobalState, b *builder.Builder,
	srcLayout VkImageLayout, dstLayout VkImageLayout,
	image VkImage, aspectMask VkImageAspectFlags, commandBuffer VkCommandBuffer) error {

	allBits := uint32(VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT<<1) - 1
	imageObject := GetState(s).Images.Get(image)

	imageBarrier := VkImageMemoryBarrier{
		SType:               VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		PNext:               NewVoidᶜᵖ(memory.Nullptr),
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
	imageBarrierData := s.AllocDataOrPanic(ctx, imageBarrier)
	defer imageBarrierData.Free()

	transfer := cb.VkCmdPipelineBarrier(
		commandBuffer,
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkDependencyFlags(0),
		0,
		memory.Nullptr,
		0,
		memory.Nullptr,
		1,
		imageBarrierData.Ptr(),
	).AddRead(
		imageBarrierData.Data(),
	)

	return transfer.Mutate(ctx, id, s, b)
}

func createAndBeginCommandBuffer(ctx context.Context, id api.CmdID, cb CommandBuilder, s *api.GlobalState, b *builder.Builder, device VkDevice, commandPool VkCommandPool) (VkCommandBuffer, error) {
	commandBufferAllocateInfo := VkCommandBufferAllocateInfo{
		SType:              VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		PNext:              NewVoidᶜᵖ(memory.Nullptr),
		CommandPool:        commandPool,
		Level:              VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	}
	commandBufferAllocateInfoData := s.AllocDataOrPanic(ctx, commandBufferAllocateInfo)
	defer commandBufferAllocateInfoData.Free()
	commandBufferId := VkCommandBuffer(newUnusedID(true, func(x uint64) bool { return GetState(s).CommandBuffers.Contains(VkCommandBuffer(x)) }))
	commandBufferData := s.AllocDataOrPanic(ctx, commandBufferId)
	defer commandBufferData.Free()

	// Data and info for Vulkan commands in command buffers
	beginCommandBufferInfo := VkCommandBufferBeginInfo{
		SType:            VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
		PNext:            NewVoidᶜᵖ(memory.Nullptr),
		Flags:            VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT),
		PInheritanceInfo: NewVkCommandBufferInheritanceInfoᶜᵖ(memory.Nullptr),
	}
	beginCommandBufferInfoData := s.AllocDataOrPanic(ctx, beginCommandBufferInfo)
	defer beginCommandBufferInfoData.Free()

	if err := cb.VkAllocateCommandBuffers(
		device,
		commandBufferAllocateInfoData.Ptr(),
		commandBufferData.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		commandBufferAllocateInfoData.Data(),
	).AddWrite(
		commandBufferData.Data(),
	).Mutate(ctx, id, s, b); err != nil {
		return commandBufferId, err
	}

	return commandBufferId, cb.VkBeginCommandBuffer(
		commandBufferId,
		beginCommandBufferInfoData.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		beginCommandBufferInfoData.Data(),
	).Mutate(ctx, id, s, b)
}

func createAndBindSourceBuffer(ctx context.Context, id api.CmdID, cb CommandBuilder, s *api.GlobalState, b *builder.Builder, device VkDevice, size VkDeviceSize, memoryIndex uint32) (VkBuffer, VkDeviceMemory, error) {
	bufferCreateInfo := VkBufferCreateInfo{
		SType:                 VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
		PNext:                 NewVoidᶜᵖ(memory.Nullptr),
		Flags:                 VkBufferCreateFlags(0),
		Size:                  size,
		Usage:                 VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT),
		SharingMode:           VkSharingMode_VK_SHARING_MODE_EXCLUSIVE,
		QueueFamilyIndexCount: 0,
		PQueueFamilyIndices:   NewU32ᶜᵖ(memory.Nullptr),
	}

	bufferId := VkBuffer(newUnusedID(true, func(x uint64) bool { return GetState(s).Buffers.Contains(VkBuffer(x)) }))
	bufferAllocateInfoData := s.AllocDataOrPanic(ctx, bufferCreateInfo)
	defer bufferAllocateInfoData.Free()
	bufferData := s.AllocDataOrPanic(ctx, bufferId)
	defer bufferData.Free()

	if err := cb.VkCreateBuffer(
		device,
		bufferAllocateInfoData.Ptr(),
		memory.Nullptr,
		bufferData.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		bufferAllocateInfoData.Data(),
	).AddWrite(
		bufferData.Data(),
	).Mutate(ctx, id, s, b); err != nil {
		return VkBuffer(0), VkDeviceMemory(0), err
	}

	memoryAllocateInfo := VkMemoryAllocateInfo{
		SType:           VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
		PNext:           NewVoidᶜᵖ(memory.Nullptr),
		AllocationSize:  size,
		MemoryTypeIndex: memoryIndex,
	}
	memoryId := VkDeviceMemory(newUnusedID(true, func(x uint64) bool { return GetState(s).DeviceMemories.Contains(VkDeviceMemory(x)) }))
	memoryAllocateInfoData := s.AllocDataOrPanic(ctx, memoryAllocateInfo)
	defer memoryAllocateInfoData.Free()
	memoryData := s.AllocDataOrPanic(ctx, memoryId)
	defer memoryData.Free()

	if err := cb.VkAllocateMemory(
		device,
		memoryAllocateInfoData.Ptr(),
		memory.Nullptr,
		memoryData.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		memoryAllocateInfoData.Data(),
	).AddWrite(
		memoryData.Data(),
	).Mutate(ctx, id, s, b); err != nil {
		return VkBuffer(0), VkDeviceMemory(0), err
	}

	if err := cb.VkBindBufferMemory(
		device, bufferId, memoryId, VkDeviceSize(0), VkResult_VK_SUCCESS,
	).Mutate(ctx, id, s, b); err != nil {
		return VkBuffer(0), VkDeviceMemory(0), err
	}

	return bufferId, memoryId, nil
}

func (a *RecreateBufferView) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateBufferView(a.Device, a.PCreateInfo, allocator, a.PBufferView, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	return hijack.Mutate(ctx, id, s, b)
}

func mapBufferMemory(ctx context.Context, id api.CmdID, cb CommandBuilder, s *api.GlobalState, b *builder.Builder, cmd api.Cmd, device VkDevice, size VkDeviceSize, mem VkDeviceMemory) (Voidᵖ, uint64, error) {
	at, err := s.Allocator.Alloc(uint64(size), 8)
	if err != nil {
		return NewVoidᵖ(memory.Nullptr), at, err
	}
	mappedPointer := s.AllocDataOrPanic(ctx, Voidᶜᵖ{at, memory.ApplicationPool})
	defer mappedPointer.Free()

	if err := cb.VkMapMemory(
		device, mem, VkDeviceSize(0), size, VkMemoryMapFlags(0), mappedPointer.Ptr(), VkResult_VK_SUCCESS,
	).AddWrite(mappedPointer.Data()).Mutate(ctx, id, s, b); err != nil {
		return NewVoidᵖ(memory.Nullptr), at, err
	}

	return NewVoidᵖᵖ(mappedPointer.Ptr()).MustRead(ctx, cmd, s, b), at, err
}

func flushBufferMemory(ctx context.Context, id api.CmdID, cb CommandBuilder, s *api.GlobalState, b *builder.Builder, device VkDevice, size VkDeviceSize, mem VkDeviceMemory, mapped U8ᵖ) error {
	flushRange := VkMappedMemoryRange{
		SType:  VkStructureType_VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE,
		PNext:  NewVoidᶜᵖ(memory.Nullptr),
		Memory: mem,
		Offset: VkDeviceSize(0),
		Size:   VkDeviceSize(0xFFFFFFFFFFFFFFFF),
	}
	flushData := s.AllocDataOrPanic(ctx, flushRange)
	defer flushData.Free()
	slice := mapped.Slice(0, uint64(size), s.MemoryLayout)

	return cb.VkFlushMappedMemoryRanges(
		device, uint32(1), flushData.Ptr(), VkResult_VK_SUCCESS,
	).AddRead(flushData.Data()).
		AddRead(slice.Range(s.MemoryLayout), slice.ResourceID(ctx, s)).Mutate(ctx, id, s, b)
}

func createBufferBarrier(ctx context.Context, id api.CmdID, cb CommandBuilder, s *api.GlobalState, b *builder.Builder, buffer VkBuffer, size VkDeviceSize, commandBuffer VkCommandBuffer) error {
	allBits := uint32(VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT<<1) - 1

	bufferBarrier := VkBufferMemoryBarrier{
		SType:               VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
		PNext:               NewVoidᶜᵖ(memory.Nullptr),
		SrcAccessMask:       VkAccessFlags(allBits),
		DstAccessMask:       VkAccessFlags(allBits),
		SrcQueueFamilyIndex: 0xFFFFFFFF,
		DstQueueFamilyIndex: 0xFFFFFFFF,
		Buffer:              buffer,
		Offset:              VkDeviceSize(0),
		Size:                size,
	}
	bufferBarrierData := s.AllocDataOrPanic(ctx, bufferBarrier)
	defer bufferBarrierData.Free()

	transfer := cb.VkCmdPipelineBarrier(
		commandBuffer,
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkDependencyFlags(0),
		0,
		memory.Nullptr,
		1,
		bufferBarrierData.Ptr(),
		0,
		memory.Nullptr,
	).AddRead(
		bufferBarrierData.Data(),
	)

	return transfer.Mutate(ctx, id, s, b)
}

func createCommandPool(ctx context.Context, id api.CmdID, cb CommandBuilder, s *api.GlobalState, b *builder.Builder, queue VkQueue, device VkDevice) (VkCommandPool, error) {
	// Command pool and command buffer
	commandPoolId := VkCommandPool(newUnusedID(false, func(x uint64) bool { return GetState(s).CommandPools.Contains(VkCommandPool(x)) }))
	queueObject := GetState(s).Queues.Get(queue)

	commandPoolCreateInfo := VkCommandPoolCreateInfo{
		SType:            VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
		PNext:            NewVoidᶜᵖ(memory.Nullptr),
		Flags:            VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT),
		QueueFamilyIndex: queueObject.Family,
	}
	commandPoolCreateInfoData := s.AllocDataOrPanic(ctx, commandPoolCreateInfo)
	defer commandPoolCreateInfoData.Free()
	commandPoolData := s.AllocDataOrPanic(ctx, commandPoolId)
	defer commandPoolData.Free()

	return commandPoolId, cb.VkCreateCommandPool(
		device,
		commandPoolCreateInfoData.Ptr(),
		memory.Nullptr,
		commandPoolData.Ptr(),
		VkResult_VK_SUCCESS).AddRead(
		commandPoolCreateInfoData.Data(),
	).AddWrite(
		commandPoolData.Data(),
	).Mutate(ctx, id, s, b)
}

func destroyCommandPool(ctx context.Context, id api.CmdID, cb CommandBuilder, s *api.GlobalState, b *builder.Builder, device VkDevice, commandPool VkCommandPool) error {
	return cb.VkDestroyCommandPool(device, commandPool, memory.Nullptr).Mutate(ctx, id, s, b)
}

func (a *RecreateImage) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	l := s.MemoryLayout
	cb := CommandBuilder{Thread: a.thread}
	allocator := memory.Nullptr
	hijack := cb.VkCreateImage(a.Device, a.PCreateInfo, allocator, a.PImage, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	if err := hijack.Mutate(ctx, id, s, b); err != nil {
		return err
	}
	img := a.PImage.MustRead(ctx, a, s, nil)
	if a.PMemoryRequirements != (VkMemoryRequirementsᵖ{}) {
		memReqs := a.PMemoryRequirements.MustRead(ctx, a, s, nil)
		GetState(s).Images.Get(img).MemoryRequirements = memReqs
	}
	if (a.SparseMemoryRequirementCount > uint32(0)) && (a.PSparseMemoryRequirements != VkSparseImageMemoryRequirementsᵖ{}) {
		for i, req := range a.PSparseMemoryRequirements.Slice(0, uint64(a.SparseMemoryRequirementCount), l).MustRead(ctx, a, s, nil) {
			GetState(s).Images.Get(img).SparseMemoryRequirements.Set(uint32(i), req)
		}
	}
	return nil
}

func (a *RecreateBindImageMemory) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	if a.Memory != VkDeviceMemory(0) {
		cb := CommandBuilder{Thread: a.thread}
		if err := cb.VkBindImageMemory(a.Device, a.Image, a.Memory, a.Offset, VkResult_VK_SUCCESS).Mutate(ctx, id, s, b); err != nil {
			return err
		}
	}
	if a.OpaqueSparseBindCount > 0 {
		cb := CommandBuilder{Thread: a.thread}
		for _, bind := range a.POpaqueSparseBinds.Slice(0, uint64(a.OpaqueSparseBindCount), s.MemoryLayout).MustRead(ctx, a, s, nil) {
			if !GetState(s).DeviceMemories.Contains(bind.Memory) {
				// TODO: Move this message to report view
				log.E(ctx, "Sparse memory binding for opaque image: %v, Memory: %v does not exist.", a.Image, bind.Memory)
			}
		}
		opaqueMemBindInfo := VkSparseImageOpaqueMemoryBindInfo{
			Image:     a.Image,
			BindCount: a.OpaqueSparseBindCount,
			PBinds:    a.POpaqueSparseBinds,
		}
		opaqueMemBindInfoData := s.AllocDataOrPanic(ctx, opaqueMemBindInfo)
		defer opaqueMemBindInfoData.Free()
		queueBindInfo := VkBindSparseInfo{
			SType:                VkStructureType_VK_STRUCTURE_TYPE_BIND_SPARSE_INFO,
			PNext:                NewVoidᶜᵖ(memory.Nullptr),
			WaitSemaphoreCount:   0,
			PWaitSemaphores:      NewVkSemaphoreᶜᵖ(memory.Nullptr),
			BufferBindCount:      0,
			PBufferBinds:         NewVkSparseBufferMemoryBindInfoᶜᵖ(memory.Nullptr),
			ImageOpaqueBindCount: 1,
			PImageOpaqueBinds:    NewVkSparseImageOpaqueMemoryBindInfoᶜᵖ(opaqueMemBindInfoData.Ptr()),
			ImageBindCount:       0,
			PImageBinds:          NewVkSparseImageMemoryBindInfoᶜᵖ(memory.Nullptr),
			SignalSemaphoreCount: 0,
			PSignalSemaphores:    NewVkSemaphoreᶜᵖ(memory.Nullptr),
		}
		queueBindInfoData := s.AllocDataOrPanic(ctx, queueBindInfo)
		defer queueBindInfoData.Free()

		queue := findSupportedQueueForDevice(a.Device, s, VkQueueFlags(VkQueueFlagBits_VK_QUEUE_SPARSE_BINDING_BIT))
		err := cb.VkQueueBindSparse(
			queue,
			1,
			queueBindInfoData.Ptr(),
			VkFence(0),
			VkResult_VK_SUCCESS,
		).AddRead(
			queueBindInfoData.Data(),
		).AddRead(
			opaqueMemBindInfoData.Data(),
		).Mutate(ctx, id, s, b)

		return err
	}
	return nil
}

func (a *RecreateImageData) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	l := s.MemoryLayout
	t := a.thread
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory.ApplicationPool())
	imageObject := GetState(s).Images.Get(a.Image)
	cb := CommandBuilder{Thread: a.thread}
	if a.LastBoundQueue != VkQueue(0) && a.LastLayout != VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED {
		queueObject := GetState(s).Queues.Get(a.LastBoundQueue)
		device := queueObject.Device
		commandPool, err := createCommandPool(ctx, id, cb, s, b, a.LastBoundQueue, device)
		if err != nil {
			return err
		}
		commandBuffer, err := createAndBeginCommandBuffer(ctx, id, cb, s, b, device, commandPool)
		if err != nil {
			return err
		}

		bufferId := VkBuffer(0)
		memoryId := VkDeviceMemory(0)
		mem := uint64(0)
		srcLayout := VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED

		if !a.Data.IsNullptr() {
			imageInfo := imageObject.Info
			bufferId, memoryId, err = createAndBindSourceBuffer(ctx, id, cb, s, b, device, a.DataSize, a.HostMemoryIndex)
			if err != nil {
				return err
			}
			mappedLocation := NewVoidᵖ(memory.Nullptr)
			mappedLocation, mem, err = mapBufferMemory(ctx, id, cb, s, b, a, device, a.DataSize, memoryId)
			if err != nil {
				return err
			}
			mappedChars := U8ᵖ(mappedLocation)
			dataP := U8ᵖ(a.Data)
			mappedChars.Slice(uint64(0), uint64(a.DataSize), l).Copy(ctx, dataP.Slice(uint64(0), uint64(a.DataSize), l), a, s, b)

			if err := flushBufferMemory(ctx, id, cb, s, b, device, a.DataSize, memoryId, mappedChars); err != nil {
				return err
			}

			if err := createImageTransition(ctx, id, cb, s, b,
				srcLayout,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				a.Image,
				imageObject.ImageAspect,
				commandBuffer); err != nil {
				return err
			}
			srcLayout = VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
			copies := []VkBufferImageCopy{}

			offset := VkDeviceSize(0)

			for i := uint32(0); i < imageInfo.MipLevels; i++ {
				width, _ := subGetMipSize(ctx, a, id, nil, s, nil, t, nil, imageInfo.Extent.Width, i)
				height, _ := subGetMipSize(ctx, a, id, nil, s, nil, t, nil, imageInfo.Extent.Height, i)
				depth, _ := subGetMipSize(ctx, a, id, nil, s, nil, t, nil, imageInfo.Extent.Depth, i)
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

				infer_level_size, err := subInferImageLevelSize(ctx, a, id, nil, s, nil, t, nil, imageObject, i)
				if err != nil {
					return err
				}
				offset += infer_level_size
			}

			pointer := s.AllocDataOrPanic(ctx, copies)
			defer pointer.Free()

			copy := cb.VkCmdCopyBufferToImage(commandBuffer, bufferId, a.Image,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, uint32(len(copies)), pointer.Ptr())
			copy.AddRead(pointer.Data())
			if err := copy.Mutate(ctx, id, s, b); err != nil {
				return err
			}
		}

		if err := createImageTransition(ctx, id, cb, s, b,
			srcLayout,
			a.LastLayout,
			a.Image,
			imageObject.ImageAspect,
			commandBuffer); err != nil {
			return err
		}
		if err := createEndCommandBufferAndQueueSubmit(ctx, id, cb, s, b, a.LastBoundQueue, commandBuffer); err != nil {
			return err
		}
		if err := cb.VkQueueWaitIdle(a.LastBoundQueue, VkResult_VK_SUCCESS).Mutate(ctx, id, s, b); err != nil {
			return err
		}
		if err := destroyCommandPool(ctx, id, cb, s, b, device, commandPool); err != nil {
			return err
		}
		if bufferId != VkBuffer(0) {
			if err := cb.VkDestroyBuffer(
				device, bufferId, memory.Nullptr,
			).Mutate(ctx, id, s, b); err != nil {
				return err
			}
			if err := cb.VkFreeMemory(
				device, memoryId, memory.Nullptr,
			).Mutate(ctx, id, s, b); err != nil {
				return err
			}
			s.Allocator.Free(mem)
		}

	}

	return nil
}

func (a *RecreateBuffer) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	// ApplyReads() is necessary only because we need to access the Read
	// observation data prior to calling the VkCreateBuffer's Mutate().
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory.ApplicationPool())

	createInfo := a.PCreateInfo.MustRead(ctx, a, s, b)
	createInfo.Usage = createInfo.Usage | VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT)
	createInfoData := s.AllocDataOrPanic(ctx, createInfo)
	defer createInfoData.Free()
	allocator := memory.Nullptr
	cb := CommandBuilder{Thread: a.thread}
	hijack := cb.VkCreateBuffer(a.Device, createInfoData.Ptr(), allocator, a.PBuffer, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	hijack.AddRead(createInfoData.Data())
	if err := hijack.Mutate(ctx, id, s, b); err != nil {
		return err
	}
	return nil
}

func (a *RecreateBindBufferMemory) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	if a.Memory != VkDeviceMemory(0) {
		cb := CommandBuilder{Thread: a.thread}
		if err := cb.VkBindBufferMemory(a.Device, a.Buffer, a.Memory, a.Offset, VkResult_VK_SUCCESS).Mutate(ctx, id, s, b); err != nil {
			return err
		}
	}
	if a.SparseBindCount > 0 {
		cb := CommandBuilder{Thread: a.thread}
		for _, bind := range a.PSparseBinds.Slice(0, uint64(a.SparseBindCount), s.MemoryLayout).MustRead(ctx, a, s, nil) {
			if !GetState(s).DeviceMemories.Contains(bind.Memory) {
				// TODO: Move this message to report view
				log.E(ctx, "Sparse memory binding for buffer: %v, Memory: %v does not exist.", a.Buffer, bind.Memory)
			}
		}
		bufMemBindInfo := VkSparseBufferMemoryBindInfo{
			Buffer:    a.Buffer,
			BindCount: a.SparseBindCount,
			PBinds:    a.PSparseBinds,
		}
		bufMemBindInfoData := s.AllocDataOrPanic(ctx, bufMemBindInfo)
		defer bufMemBindInfoData.Free()
		queueBindInfo := VkBindSparseInfo{
			SType:                VkStructureType_VK_STRUCTURE_TYPE_BIND_SPARSE_INFO,
			PNext:                NewVoidᶜᵖ(memory.Nullptr),
			WaitSemaphoreCount:   0,
			PWaitSemaphores:      NewVkSemaphoreᶜᵖ(memory.Nullptr),
			BufferBindCount:      1,
			PBufferBinds:         NewVkSparseBufferMemoryBindInfoᶜᵖ(bufMemBindInfoData.Ptr()),
			ImageOpaqueBindCount: 0,
			PImageOpaqueBinds:    NewVkSparseImageOpaqueMemoryBindInfoᶜᵖ(memory.Nullptr),
			ImageBindCount:       0,
			PImageBinds:          NewVkSparseImageMemoryBindInfoᶜᵖ(memory.Nullptr),
			SignalSemaphoreCount: 0,
			PSignalSemaphores:    NewVkSemaphoreᶜᵖ(memory.Nullptr),
		}
		queueBindInfoData := s.AllocDataOrPanic(ctx, queueBindInfo)
		defer queueBindInfoData.Free()

		queue := findSupportedQueueForDevice(a.Device, s, VkQueueFlags(VkQueueFlagBits_VK_QUEUE_SPARSE_BINDING_BIT))
		err := cb.VkQueueBindSparse(
			queue,
			1,
			queueBindInfoData.Ptr(),
			VkFence(0),
			VkResult_VK_SUCCESS,
		).AddRead(
			queueBindInfoData.Data(),
		).AddRead(
			bufMemBindInfoData.Data(),
		).Mutate(ctx, id, s, b)

		return err
	}
	return nil
}

func (a *RecreateBufferData) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	l := s.MemoryLayout
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory.ApplicationPool())

	// If we have data to fill this buffer with:
	if !a.Data.IsNullptr() {
		queue := a.LastBoundQueue
		queueObject := GetState(s).Queues.Get(queue)
		device := queueObject.Device
		bufferObject := GetState(s).Buffers.Get(a.Buffer)
		bufferInfo := bufferObject.Info
		cb := CommandBuilder{Thread: a.thread}

		bufferId, memoryId, err := createAndBindSourceBuffer(ctx, id, cb, s, b, device, bufferInfo.Size, a.HostBufferMemoryIndex)
		if err != nil {
			return err
		}
		mappedLocation, mem, err := mapBufferMemory(ctx, id, cb, s, b, a, device, bufferInfo.Size, memoryId)
		if err != nil {
			return err
		}
		mappedChars := U8ᵖ(mappedLocation)
		dataP := U8ᵖ(a.Data)
		mappedChars.Slice(uint64(0), uint64(bufferInfo.Size), l).Copy(ctx, dataP.Slice(uint64(0), uint64(bufferInfo.Size), l), a, s, b)

		if err := flushBufferMemory(ctx, id, cb, s, b, device, bufferInfo.Size, memoryId, mappedChars); err != nil {
			return err
		}

		commandPool, err := createCommandPool(ctx, id, cb, s, b, queue, device)
		if err != nil {
			return err
		}

		commandBuffer, err := createAndBeginCommandBuffer(ctx, id, cb, s, b, device, commandPool)
		if err != nil {
			return err
		}

		bufferCopy := VkBufferCopy{
			SrcOffset: VkDeviceSize(0),
			DstOffset: VkDeviceSize(0),
			Size:      bufferInfo.Size,
		}
		bufferData := s.AllocDataOrPanic(ctx, bufferCopy)
		defer bufferData.Free()
		if err := cb.VkCmdCopyBuffer(commandBuffer, bufferId, a.Buffer, 1, bufferData.Ptr()).
			AddRead(bufferData.Data()).Mutate(ctx, id, s, b); err != nil {
			return err
		}

		if err := createBufferBarrier(ctx, id, cb, s, b, bufferId, bufferInfo.Size, commandBuffer); err != nil {
			return err
		}

		if err := createEndCommandBufferAndQueueSubmit(ctx, id, cb, s, b, queue, commandBuffer); err != nil {
			return err
		}
		if err := cb.VkQueueWaitIdle(queue, VkResult_VK_SUCCESS).Mutate(ctx, id, s, b); err != nil {
			return err
		}
		if err := destroyCommandPool(ctx, id, cb, s, b, device, commandPool); err != nil {
			return err
		}
		if err := cb.VkDestroyBuffer(
			device, bufferId, memory.Nullptr,
		).Mutate(ctx, id, s, b); err != nil {
			return err
		}
		if err := cb.VkFreeMemory(
			device, memoryId, memory.Nullptr,
		).Mutate(ctx, id, s, b); err != nil {
			return err
		}
		s.Allocator.Free(mem)
	}
	return nil
}

// Returns a queue capable of graphics and compute operations if it could be
// found, a compute only queue or copy queue will be returned if it could not
// be found
func findGraphicsAndComputeQueueForDevice(device VkDevice, s *api.GlobalState) VkQueue {
	c := GetState(s)
	backupQueue := VkQueue(0)
	backupQueueFlags := uint32(0)
	for _, v := range c.Queues.Range() {
		if v.Device == device {
			family := c.PhysicalDevices.Get(c.Devices.Get(device).PhysicalDevice).QueueFamilyProperties.Get(v.Family)
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

// Returns a queue capable of the sorts of operations specified in the flags.
// If such a queue cannot be found, returns a VkQueue(0).
func findSupportedQueueForDevice(device VkDevice, s *api.GlobalState, flags VkQueueFlags) VkQueue {
	c := GetState(s)
	for _, v := range c.Queues.Range() {
		if v.Device == device {
			family := c.PhysicalDevices.Get(c.Devices.Get(device).PhysicalDevice).QueueFamilyProperties.Get(v.Family)
			if uint32(family.QueueFlags)&uint32(flags) == uint32(flags) {
				return v.VulkanHandle
			}
		}
	}
	return VkQueue(0)
}

func (a *RecreateQueryPool) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	l := s.MemoryLayout
	allocator := memory.Nullptr
	cb := CommandBuilder{Thread: a.thread}

	hijack := cb.VkCreateQueryPool(a.Device, a.PCreateInfo, allocator, a.PPool, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	if err := hijack.Mutate(ctx, id, s, b); err != nil {
		return err
	}

	createInfoObject := a.PCreateInfo.MustRead(ctx, a, s, b)
	queryStates := a.PQueryStatuses.Slice(0, uint64(createInfoObject.QueryCount), l).MustRead(ctx, a, s, b)
	pool := a.PPool.MustRead(ctx, a, s, b)

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
	commandPool, err := createCommandPool(ctx, id, cb, s, b, queue, a.Device)
	if err != nil {
		return err
	}
	commandBuffer, err := createAndBeginCommandBuffer(ctx, id, cb, s, b, a.Device, commandPool)
	if err != nil {
		return err
	}

	for i := uint32(0); i < createInfoObject.QueryCount; i++ {
		if queryStates[i] != QueryStatus_QUERY_STATUS_INACTIVE {
			if err := cb.VkCmdBeginQuery(commandBuffer,
				pool, i, VkQueryControlFlags(0)).Mutate(ctx, id, s, b); err != nil {
				return err
			}

			if queryStates[i] == QueryStatus_QUERY_STATUS_COMPLETE {
				if err := cb.VkCmdEndQuery(commandBuffer,
					pool, i).Mutate(ctx, id, s, b); err != nil {
					return err
				}
			}
		}
	}

	if err := createEndCommandBufferAndQueueSubmit(ctx, id, cb, s, b, queue, commandBuffer); err != nil {
		return err
	}
	if err := cb.VkQueueWaitIdle(queue, VkResult_VK_SUCCESS).Mutate(ctx, id, s, b); err != nil {
		return err
	}
	if err := destroyCommandPool(ctx, id, cb, s, b, a.Device, commandPool); err != nil {
		return err
	}

	return nil

}

func (a *RecreateSwapchain) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	defer EnterRecreate(ctx, s)()
	l := s.MemoryLayout
	allocator := memory.Nullptr
	cb := CommandBuilder{Thread: a.thread}
	hijack := cb.VkCreateSwapchainKHR(a.Device, a.PCreateInfo, allocator, a.PSwapchain, VkResult(0))
	hijack.Extras().MustClone(a.Extras().All()...)
	if err := hijack.Mutate(ctx, id, s, b); err != nil {
		return err
	}
	swapchain := a.PSwapchain.MustRead(ctx, a, s, b)
	createInfoData := a.PCreateInfo.MustRead(ctx, a, s, b)
	swapchainCountData := s.AllocDataOrPanic(ctx, createInfoData.MinImageCount)
	defer swapchainCountData.Free()

	getImages := cb.VkGetSwapchainImagesKHR(a.Device, swapchain, swapchainCountData.Ptr(), a.PSwapchainImages, VkResult(0))
	getImages.Extras().MustClone(a.Extras().All()...)
	getImages.AddRead(swapchainCountData.Data()).AddWrite(swapchainCountData.Data())
	if err := getImages.Mutate(ctx, id, s, b); err != nil {
		return err
	}

	images := a.PSwapchainImages.Slice(0, uint64(createInfoData.MinImageCount), l).MustRead(ctx, a, s, b)
	imageLayouts := a.PSwapchainLayouts.Slice(0, uint64(createInfoData.MinImageCount), l).MustRead(ctx, a, s, b)
	boundQueues := a.PInitialQueues.Slice(0, uint64(createInfoData.MinImageCount), l).MustRead(ctx, a, s, b)
	for i := 0; i < int(createInfoData.MinImageCount); i++ {
		imageObject := GetState(s).Images.Get(images[i])
		if boundQueues[i] != VkQueue(0) && imageLayouts[i] != VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED {
			queue := boundQueues[i]
			queueObject := GetState(s).Queues.Get(queue)
			device := queueObject.Device
			commandPool, err := createCommandPool(ctx, id, cb, s, b, queue, device)
			if err != nil {
				return err
			}
			commandBuffer, err := createAndBeginCommandBuffer(ctx, id, cb, s, b, device, commandPool)
			if err != nil {
				return err
			}
			if err := createImageTransition(ctx, id, cb, s, b,
				VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
				imageLayouts[i],
				imageObject.VulkanHandle,
				imageObject.ImageAspect,
				commandBuffer); err != nil {
				return err
			}
			if err := createEndCommandBufferAndQueueSubmit(ctx, id, cb, s, b, queue, commandBuffer); err != nil {
				return err
			}
			if err := cb.VkQueueWaitIdle(queue, VkResult_VK_SUCCESS).Mutate(ctx, id, s, b); err != nil {
				return err
			}
			if err := destroyCommandPool(ctx, id, cb, s, b, device, commandPool); err != nil {
				return err
			}
		}
	}

	return nil
}
