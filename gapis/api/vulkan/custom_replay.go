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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
)

const virtualSwapchainStruct = 0xFFFFFFAA

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
	cb := CommandBuilder{Thread: a.Thread(), Arena: s.Arena}
	// Hijack VkCreateInstance's Mutate() method entirely with our ReplayCreateVkInstance's Mutate().

	// As long as we guarantee that the synthetic replayCreateVkInstance API function has the same
	// logic as the real vkCreateInstance API function, we can do observation correctly. Additionally,
	// ReplayCreateVkInstance's Mutate() will invoke our custom wrapper function replayCreateVkInstance()
	// in vulkan_gfx_api_extras.cpp, which modifies VkInstanceCreateInfo to enable virtual swapchain
	// layer before delegating the real work back to the normal flow.

	hijack := cb.ReplayCreateVkInstance(a.PCreateInfo(), a.PAllocator(), a.PInstance(), a.Result())
	hijack.Extras().MustClone(a.Extras().All()...)
	err := hijack.Mutate(ctx, id, s, b)

	if b == nil || err != nil {
		return err
	}

	// Call the replayRegisterVkInstance() synthetic API function.
	instance := a.PInstance().MustRead(ctx, a, s, b)
	return cb.ReplayRegisterVkInstance(instance).Mutate(ctx, id, s, b)
}

func (a *VkDestroyInstance) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	cb := CommandBuilder{Thread: a.Thread(), Arena: s.Arena}
	// Call the underlying vkDestroyInstance() and do the observation.
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}
	// Call the replayUnregisterVkInstance() synthetic API function.
	return cb.ReplayUnregisterVkInstance(a.Instance()).Mutate(ctx, id, s, b)
}

func EnterRecreate(ctx context.Context, s *api.GlobalState) func() {
	GetState(s).SetIsRebuilding(true)
	return func() { GetState(s).SetIsRebuilding(false) }
}

func (a *VkCreateDevice) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	// Hijack VkCreateDevice's Mutate() method entirely with our
	// ReplayCreateVkDevice's Mutate(). Similar to VkCreateInstance's Mutate()
	// above.
	// And we need to strip off the VK_EXT_debug_marker extension name when
	// building instructions for replay.
	createInfoPtr := a.PCreateInfo()
	allocated := []*api.AllocResult{}
	if b != nil {
		a.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
		createInfo := a.PCreateInfo().MustRead(ctx, a, s, nil)
		defer func() {
			for _, d := range allocated {
				d.Free()
			}
		}()
		extensionCount := uint64(createInfo.EnabledExtensionCount())
		newExtensionNames := []memory.Pointer{}
		for _, e := range createInfo.PpEnabledExtensionNames().Slice(0, extensionCount, s.MemoryLayout).MustRead(ctx, a, s, nil) {
			extensionName := string(memory.CharToBytes(e.StringSlice(ctx, s).MustRead(ctx, a, s, nil)))
			if !strings.Contains(extensionName, "VK_EXT_debug_marker") {
				nameSliceData := s.AllocDataOrPanic(ctx, extensionName)
				allocated = append(allocated, &nameSliceData)
				newExtensionNames = append(newExtensionNames, nameSliceData.Ptr())
			}
		}
		newExtensionNamesData := s.AllocDataOrPanic(ctx, newExtensionNames)
		allocated = append(allocated, &newExtensionNamesData)
		createInfo.SetEnabledExtensionCount(uint32(len(newExtensionNames)))
		createInfo.SetPpEnabledExtensionNames(NewCharᶜᵖᶜᵖ(newExtensionNamesData.Ptr()))

		newCreateInfoData := s.AllocDataOrPanic(ctx, createInfo)
		allocated = append(allocated, &newCreateInfoData)
		createInfoPtr = NewVkDeviceCreateInfoᶜᵖ(newCreateInfoData.Ptr())

		cb := CommandBuilder{Thread: a.Thread(), Arena: s.Arena}
		hijack := cb.ReplayCreateVkDevice(a.PhysicalDevice(), createInfoPtr, a.PAllocator(), a.PDevice(), a.Result())
		hijack.Extras().MustClone(a.Extras().All()...)

		for _, d := range allocated {
			hijack.AddRead(d.Data())
		}

		err := hijack.Mutate(ctx, id, s, b)
		if err != nil {
			return err
		}
		// Call the replayRegisterVkDevice() synthetic API function.
		device := a.PDevice().MustRead(ctx, a, s, b)
		return cb.ReplayRegisterVkDevice(a.PhysicalDevice(), device, a.PCreateInfo()).Mutate(ctx, id, s, b)
	}

	return a.mutate(ctx, id, s, b)
}

func (a *VkDestroyDevice) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	// Call the underlying vkDestroyDevice() and do the observation.
	cb := CommandBuilder{Thread: a.Thread(), Arena: s.Arena}
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}
	// Call the replayUnregisterVkDevice() synthetic API function.
	return cb.ReplayUnregisterVkDevice(a.Device()).Mutate(ctx, id, s, b)
}

func (a *VkAllocateCommandBuffers) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	// Call the underlying vkAllocateCommandBuffers() and do the observation.
	cb := CommandBuilder{Thread: a.Thread(), Arena: s.Arena}
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}
	// Call the replayRegisterVkCommandBuffers() synthetic API function to link these command buffers to the device.
	count := a.PAllocateInfo().MustRead(ctx, a, s, b).CommandBufferCount()
	return cb.ReplayRegisterVkCommandBuffers(a.Device(), count, a.PCommandBuffers()).Mutate(ctx, id, s, b)
}

func (a *VkFreeCommandBuffers) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	// Call the underlying vkFreeCommandBuffers() and do the observation.
	cb := CommandBuilder{Thread: a.Thread(), Arena: s.Arena}
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}
	// Call the replayUnregisterVkCommandBuffers() synthetic API function to discard the link of these command buffers.
	count := a.CommandBufferCount()
	return cb.ReplayUnregisterVkCommandBuffers(count, a.PCommandBuffers()).Mutate(ctx, id, s, b)
}

func (a *VkCreateSwapchainKHR) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	if b == nil {
		return a.mutate(ctx, id, s, b)
	}

	cb := CommandBuilder{Thread: a.Thread(), Arena: s.Arena}
	hijack := cb.ReplayCreateSwapchain(a.Device(), a.PCreateInfo(), a.PAllocator(), a.PSwapchain(), a.Result())
	hijack.Extras().MustClone(a.Extras().All()...)
	err := hijack.Mutate(ctx, id, s, b)

	return err
}

func (a *VkAcquireNextImageKHR) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	l := s.MemoryLayout
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory.ApplicationPool())
	// Apply the write observation before having the replay device calling the
	// vkAcquireNextImageKHR() command. This is to pass the returned image index
	// value captured in the trace, into the replay device to acquire for the
	// specific image.
	// Note that this is only necessary for building replay instructions
	if b != nil {
		o.ApplyWrites(s.Memory.ApplicationPool())
	}
	_ = a.PImageIndex().Slice(0, 1, l).MustRead(ctx, a, s, b)

	if b != nil {
		a.Call(ctx, s, b)
	}

	// When not building for replay instructions, apply write observations after
	// the call
	if b == nil {
		o.ApplyWrites(s.Memory.ApplicationPool())
	}
	a.PImageIndex().Slice(0, 1, l).Write(ctx, a.PImageIndex().Slice(0, 1, l).MustRead(ctx, a, s, nil), a, s, b)
	_ = a.Result()
	if a.Semaphore() != VkSemaphore(0) {
		GetState(s).Semaphores().Get(a.Semaphore()).SetSignaled(true)
	}
	if a.Fence() != VkFence(0) {
		GetState(s).Fences().Get(a.Fence()).SetSignaled(true)
	}
	return nil
}

type structWithPNext interface {
	PNext() Voidᶜᵖ
	SetPNext(v Voidᶜᵖ)
}

func insertVirtualSwapchainPNext(ctx context.Context, cmd api.Cmd, id api.CmdID,
	info structWithPNext, g *api.GlobalState) (api.AllocResult, api.AllocResult) {
	pNextData := g.AllocDataOrPanic(ctx, NewVulkanStructHeader(
		g.Arena,
		virtualSwapchainStruct, // sType
		0, // pNext
	))
	if info.PNext().IsNullptr() {
		info.SetPNext(NewVoidᶜᵖ(pNextData.Ptr()))
	} else {
		pNext := NewVoidᵖ(info.PNext())
		for !pNext.IsNullptr() {
			structHeader := NewVulkanStructHeaderᵖ(pNext).MustRead(ctx, cmd, g, nil)
			if !structHeader.PNext().IsNullptr() {
				structHeader.SetPNext(NewVoidᵖ(pNextData.Ptr()))
				break
			}
			pNext = structHeader.PNext()
		}
	}
	newInfoData := g.AllocDataOrPanic(ctx, info)
	return newInfoData, pNextData
}

func (c *VkCreateXlibSurfaceKHR) Mutate(ctx context.Context, id api.CmdID, g *api.GlobalState, b *builder.Builder) error {
	if b == nil {
		return c.mutate(ctx, id, g, b)
	}
	// When building replay instructions, insert a pNext struct to enable the
	// virtual surface on the replay device.
	c.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	newInfoData, pNextData := insertVirtualSwapchainPNext(ctx, c, id, c.PCreateInfo().MustRead(ctx, c, g, nil), g)
	defer newInfoData.Free()
	defer pNextData.Free()
	cb := CommandBuilder{Thread: c.Thread(), Arena: g.Arena}
	hijack := cb.VkCreateXlibSurfaceKHR(
		c.Instance(), newInfoData.Ptr(), c.PAllocator(), c.PSurface(), c.Result(),
	).AddRead(newInfoData.Data()).AddRead(pNextData.Data())
	for _, r := range c.Extras().Observations().Reads {
		hijack.AddRead(r.Range, r.ID)
	}
	for _, w := range c.Extras().Observations().Writes {
		hijack.AddWrite(w.Range, w.ID)
	}
	hijack.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	info := hijack.PCreateInfo().MustRead(ctx, hijack, g, b)
	if (info.PNext()) != (Voidᶜᵖ(0)) {
		numPNext := (externs{ctx, hijack, id, g, b}.numberOfPNext(info.PNext()))
		next := NewMutableVoidPtr(g.Arena, Voidᵖ(info.PNext()))
		for i := uint32(0); i < numPNext; i++ {
			VkStructureTypeᶜᵖ(next.Ptr()).MustRead(ctx, hijack, g, b)
			next.SetPtr(VulkanStructHeaderᵖ(next.Ptr()).MustRead(ctx, hijack, g, b).PNext())
		}
	}
	surface := NewSurfaceObjectʳ(
		g.Arena, VkInstance(0), VkSurfaceKHR(0), SurfaceType(0),
		NilVulkanDebugMarkerInfoʳ, NewVkPhysicalDeviceːQueueFamilySupportsʳᵐ(g.Arena))
	surface.SetInstance(hijack.Instance())
	surface.SetType(SurfaceType_SURFACE_TYPE_XLIB)

	hijack.Call(ctx, g, b)

	hijack.Extras().Observations().ApplyWrites(g.Memory.ApplicationPool())
	handle := hijack.PSurface().MustRead(ctx, hijack, g, nil)
	hijack.PSurface().MustWrite(ctx, handle, hijack, g, b)
	surface.SetVulkanHandle(handle)
	GetState(g).Surfaces().Add(handle, surface)
	hijack.Result()
	return nil
}

func (c *VkCreateXcbSurfaceKHR) Mutate(ctx context.Context, id api.CmdID, g *api.GlobalState, b *builder.Builder) error {
	if b == nil {
		return c.mutate(ctx, id, g, b)
	}
	// When building replay instructions, insert a pNext struct to enable the
	// virtual surface on the replay device.
	c.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	newInfoData, pNextData := insertVirtualSwapchainPNext(ctx, c, id, c.PCreateInfo().MustRead(ctx, c, g, nil), g)
	defer newInfoData.Free()
	defer pNextData.Free()
	cb := CommandBuilder{Thread: c.Thread(), Arena: g.Arena}
	hijack := cb.VkCreateXcbSurfaceKHR(
		c.Instance(), newInfoData.Ptr(), c.PAllocator(), c.PSurface(), c.Result(),
	).AddRead(newInfoData.Data()).AddRead(pNextData.Data())
	for _, r := range c.Extras().Observations().Reads {
		hijack.AddRead(r.Range, r.ID)
	}
	for _, w := range c.Extras().Observations().Writes {
		hijack.AddWrite(w.Range, w.ID)
	}

	hijack.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	info := hijack.PCreateInfo().MustRead(ctx, hijack, g, b)
	if (info.PNext()) != (Voidᶜᵖ(0)) {
		numPNext := (externs{ctx, hijack, id, g, b}.numberOfPNext(info.PNext()))
		next := NewMutableVoidPtr(g.Arena, Voidᵖ(info.PNext()))
		for i := uint32(0); i < numPNext; i++ {
			VkStructureTypeᶜᵖ(next.Ptr()).MustRead(ctx, hijack, g, b)
			next.SetPtr(VulkanStructHeaderᵖ(next.Ptr()).MustRead(ctx, hijack, g, b).PNext())
		}
	}
	surface := NewSurfaceObjectʳ(
		g.Arena, VkInstance(0), VkSurfaceKHR(0), SurfaceType(0),
		NilVulkanDebugMarkerInfoʳ, NewVkPhysicalDeviceːQueueFamilySupportsʳᵐ(g.Arena))
	surface.SetInstance(hijack.Instance())
	surface.SetType(SurfaceType_SURFACE_TYPE_XCB)

	hijack.Call(ctx, g, b)

	hijack.Extras().Observations().ApplyWrites(g.Memory.ApplicationPool())
	handle := hijack.PSurface().MustRead(ctx, hijack, g, nil)
	hijack.PSurface().MustWrite(ctx, handle, hijack, g, b)
	surface.SetVulkanHandle(handle)
	GetState(g).Surfaces().Add(handle, surface)
	hijack.Result()
	return nil
}

func (c *VkCreateWaylandSurfaceKHR) Mutate(ctx context.Context, id api.CmdID, g *api.GlobalState, b *builder.Builder) error {
	if b == nil {
		return c.mutate(ctx, id, g, b)
	}
	// When building replay instructions, insert a pNext struct to enable the
	// virtual surface on the replay device.
	c.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	newInfoData, pNextData := insertVirtualSwapchainPNext(ctx, c, id, c.PCreateInfo().MustRead(ctx, c, g, nil), g)
	defer newInfoData.Free()
	defer pNextData.Free()
	cb := CommandBuilder{Thread: c.Thread(), Arena: g.Arena}
	hijack := cb.VkCreateWaylandSurfaceKHR(
		c.Instance(), newInfoData.Ptr(), c.PAllocator(), c.PSurface(), c.Result(),
	).AddRead(newInfoData.Data()).AddRead(pNextData.Data())
	for _, r := range c.Extras().Observations().Reads {
		hijack.AddRead(r.Range, r.ID)
	}
	for _, w := range c.Extras().Observations().Writes {
		hijack.AddWrite(w.Range, w.ID)
	}
	hijack.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	info := hijack.PCreateInfo().MustRead(ctx, hijack, g, b)
	if (info.PNext()) != (Voidᶜᵖ(0)) {
		numPNext := (externs{ctx, hijack, id, g, b}.numberOfPNext(info.PNext()))
		next := NewMutableVoidPtr(g.Arena, Voidᵖ(info.PNext()))
		for i := uint32(0); i < numPNext; i++ {
			VkStructureTypeᶜᵖ(next.Ptr()).MustRead(ctx, hijack, g, b)
			next.SetPtr(VulkanStructHeaderᵖ(next.Ptr()).MustRead(ctx, hijack, g, b).PNext())
		}
	}
	surface := NewSurfaceObjectʳ(
		g.Arena, VkInstance(0), VkSurfaceKHR(0), SurfaceType(0),
		NilVulkanDebugMarkerInfoʳ, NewVkPhysicalDeviceːQueueFamilySupportsʳᵐ(g.Arena))
	surface.SetInstance(hijack.Instance())
	surface.SetType(SurfaceType_SURFACE_TYPE_WAYLAND)

	hijack.Call(ctx, g, b)

	hijack.Extras().Observations().ApplyWrites(g.Memory.ApplicationPool())
	handle := hijack.PSurface().MustRead(ctx, hijack, g, nil)
	hijack.PSurface().MustWrite(ctx, handle, hijack, g, b)
	surface.SetVulkanHandle(handle)
	GetState(g).Surfaces().Add(handle, surface)
	hijack.Result()
	return nil
}

func (c *VkCreateMirSurfaceKHR) Mutate(ctx context.Context, id api.CmdID, g *api.GlobalState, b *builder.Builder) error {
	if b == nil {
		return c.mutate(ctx, id, g, b)
	}
	// When building replay instructions, insert a pNext struct to enable the
	// virtual surface on the replay device.
	c.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	newInfoData, pNextData := insertVirtualSwapchainPNext(ctx, c, id, c.PCreateInfo().MustRead(ctx, c, g, nil), g)
	defer newInfoData.Free()
	defer pNextData.Free()
	cb := CommandBuilder{Thread: c.Thread(), Arena: g.Arena}
	hijack := cb.VkCreateMirSurfaceKHR(
		c.Instance(), newInfoData.Ptr(), c.PAllocator(), c.PSurface(), c.Result(),
	).AddRead(newInfoData.Data()).AddRead(pNextData.Data())
	for _, r := range c.Extras().Observations().Reads {
		hijack.AddRead(r.Range, r.ID)
	}
	for _, w := range c.Extras().Observations().Writes {
		hijack.AddWrite(w.Range, w.ID)
	}
	hijack.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	info := hijack.PCreateInfo().MustRead(ctx, hijack, g, b)
	if (info.PNext()) != (Voidᶜᵖ(0)) {
		numPNext := (externs{ctx, hijack, id, g, b}.numberOfPNext(info.PNext()))
		next := NewMutableVoidPtr(g.Arena, Voidᵖ(info.PNext()))
		for i := uint32(0); i < numPNext; i++ {
			VkStructureTypeᶜᵖ(next.Ptr()).MustRead(ctx, hijack, g, b)
			next.SetPtr(VulkanStructHeaderᵖ(next.Ptr()).MustRead(ctx, hijack, g, b).PNext())
		}
	}
	surface := NewSurfaceObjectʳ(
		g.Arena, VkInstance(0), VkSurfaceKHR(0), SurfaceType(0),
		NilVulkanDebugMarkerInfoʳ, NewVkPhysicalDeviceːQueueFamilySupportsʳᵐ(g.Arena))
	surface.SetInstance(hijack.Instance())
	surface.SetType(SurfaceType_SURFACE_TYPE_MIR)

	hijack.Call(ctx, g, b)

	hijack.Extras().Observations().ApplyWrites(g.Memory.ApplicationPool())
	handle := hijack.PSurface().MustRead(ctx, hijack, g, nil)
	hijack.PSurface().MustWrite(ctx, handle, hijack, g, b)
	surface.SetVulkanHandle(handle)
	GetState(g).Surfaces().Add(handle, surface)
	hijack.Result()
	return nil
}

func (c *VkCreateWin32SurfaceKHR) Mutate(ctx context.Context, id api.CmdID, g *api.GlobalState, b *builder.Builder) error {
	if b == nil {
		return c.mutate(ctx, id, g, b)
	}
	// When building replay instructions, insert a pNext struct to enable the
	// virtual surface on the replay device.
	c.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	newInfoData, pNextData := insertVirtualSwapchainPNext(ctx, c, id, c.PCreateInfo().MustRead(ctx, c, g, nil), g)
	defer newInfoData.Free()
	defer pNextData.Free()
	cb := CommandBuilder{Thread: c.Thread(), Arena: g.Arena}
	hijack := cb.VkCreateWin32SurfaceKHR(
		c.Instance(), newInfoData.Ptr(), c.PAllocator(), c.PSurface(), c.Result(),
	).AddRead(newInfoData.Data()).AddRead(pNextData.Data())
	for _, r := range c.Extras().Observations().Reads {
		hijack.AddRead(r.Range, r.ID)
	}
	for _, w := range c.Extras().Observations().Writes {
		hijack.AddWrite(w.Range, w.ID)
	}
	hijack.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	info := hijack.PCreateInfo().MustRead(ctx, hijack, g, b)
	if (info.PNext()) != (Voidᶜᵖ(0)) {
		numPNext := (externs{ctx, hijack, id, g, b}.numberOfPNext(info.PNext()))
		next := NewMutableVoidPtr(g.Arena, Voidᵖ(info.PNext()))
		for i := uint32(0); i < numPNext; i++ {
			VkStructureTypeᶜᵖ(next.Ptr()).MustRead(ctx, hijack, g, b)
			next.SetPtr(VulkanStructHeaderᵖ(next.Ptr()).MustRead(ctx, hijack, g, b).PNext())
		}
	}
	surface := NewSurfaceObjectʳ(
		g.Arena, VkInstance(0), VkSurfaceKHR(0), SurfaceType(0),
		NilVulkanDebugMarkerInfoʳ, NewVkPhysicalDeviceːQueueFamilySupportsʳᵐ(g.Arena))
	surface.SetInstance(hijack.Instance())
	surface.SetType(SurfaceType_SURFACE_TYPE_WIN32)

	hijack.Call(ctx, g, b)

	hijack.Extras().Observations().ApplyWrites(g.Memory.ApplicationPool())
	handle := hijack.PSurface().MustRead(ctx, hijack, g, nil)
	hijack.PSurface().MustWrite(ctx, handle, hijack, g, b)
	surface.SetVulkanHandle(handle)
	GetState(g).Surfaces().Add(handle, surface)
	hijack.Result()
	return nil
}

func (c *VkCreateAndroidSurfaceKHR) Mutate(ctx context.Context, id api.CmdID, g *api.GlobalState, b *builder.Builder) error {
	if b == nil {
		return c.mutate(ctx, id, g, b)
	}
	// When building replay instructions, insert a pNext struct to enable the
	// virtual surface on the replay device.
	c.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	newInfoData, pNextData := insertVirtualSwapchainPNext(ctx, c, id, c.PCreateInfo().MustRead(ctx, c, g, nil), g)
	defer newInfoData.Free()
	defer pNextData.Free()
	cb := CommandBuilder{Thread: c.Thread(), Arena: g.Arena}
	hijack := cb.VkCreateAndroidSurfaceKHR(
		c.Instance(), newInfoData.Ptr(), c.PAllocator(), c.PSurface(), c.Result(),
	).AddRead(newInfoData.Data()).AddRead(pNextData.Data())
	for _, r := range c.Extras().Observations().Reads {
		hijack.AddRead(r.Range, r.ID)
	}
	for _, w := range c.Extras().Observations().Writes {
		hijack.AddWrite(w.Range, w.ID)
	}
	hijack.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	info := hijack.PCreateInfo().MustRead(ctx, hijack, g, b)
	if (info.PNext()) != (Voidᶜᵖ(0)) {
		numPNext := (externs{ctx, hijack, id, g, b}.numberOfPNext(info.PNext()))
		next := NewMutableVoidPtr(g.Arena, Voidᵖ(info.PNext()))
		for i := uint32(0); i < numPNext; i++ {
			VkStructureTypeᶜᵖ(next.Ptr()).MustRead(ctx, hijack, g, b)
			next.SetPtr(VulkanStructHeaderᵖ(next.Ptr()).MustRead(ctx, hijack, g, b).PNext())
		}
	}
	surface := NewSurfaceObjectʳ(
		g.Arena, VkInstance(0), VkSurfaceKHR(0), SurfaceType(0),
		NilVulkanDebugMarkerInfoʳ, NewVkPhysicalDeviceːQueueFamilySupportsʳᵐ(g.Arena))
	surface.SetInstance(hijack.Instance())
	surface.SetType(SurfaceType_SURFACE_TYPE_ANDROID)

	hijack.Call(ctx, g, b)

	hijack.Extras().Observations().ApplyWrites(g.Memory.ApplicationPool())
	handle := hijack.PSurface().MustRead(ctx, hijack, g, nil)
	hijack.PSurface().MustWrite(ctx, handle, hijack, g, b)
	surface.SetVulkanHandle(handle)
	GetState(g).Surfaces().Add(handle, surface)
	hijack.Result()
	return nil
}

func (c *VkGetPhysicalDeviceSurfaceFormatsKHR) Mutate(ctx context.Context, id api.CmdID, g *api.GlobalState, b *builder.Builder) error {
	if b == nil {
		return c.mutate(ctx, id, g, b)
	}
	// When building replay instructions, apply the write observations so that
	// the returned surface format count and formats, which are captured in
	// the trace, will be passed to the virtual swapchain. This is to cheat
	// the validation layers, as the returned surface formats will always match
	// with the format used in the trace.
	l := g.MemoryLayout
	c.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	c.Extras().Observations().ApplyWrites(g.Memory.ApplicationPool())
	givenCount := c.PSurfaceFormatCount().MustRead(ctx, c, g, b)
	if (c.PSurfaceFormats()) != (VkSurfaceFormatKHRᵖ(0)) {
		c.PSurfaceFormats().Slice(0, uint64(givenCount), l).OnRead(ctx, c, g, b)
	}
	c.Call(ctx, g, b)
	c.Extras().Observations().ApplyWrites(g.Memory.ApplicationPool())
	if (c.PSurfaceFormats()) == (VkSurfaceFormatKHRᵖ(0)) {
		c.PSurfaceFormatCount().MustWrite(ctx, c.PSurfaceFormatCount().MustRead(ctx, c, g, nil), c, g, b)
	} else {
		count := c.PSurfaceFormatCount().MustRead(ctx, c, g, nil)
		formats := c.PSurfaceFormats().Slice(0, uint64(count), l)
		for i := uint32(0); i < count; i++ {
			formats.Index(uint64(i)).MustWrite(ctx, []VkSurfaceFormatKHR{c.PSurfaceFormats().Slice(uint64(uint32(0)), uint64(count), l).Index(uint64(i)).MustRead(ctx, c, g, nil)[0]}, c, g, b)
		}
		c.PSurfaceFormatCount().MustWrite(ctx, count, c, g, b)
	}
	return nil
}

func (c *VkGetPhysicalDeviceSurfacePresentModesKHR) Mutate(ctx context.Context, id api.CmdID, g *api.GlobalState, b *builder.Builder) error {
	if b == nil {
		return c.mutate(ctx, id, g, b)
	}
	l := g.MemoryLayout
	c.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	c.Extras().Observations().ApplyWrites(g.Memory.ApplicationPool())
	givenCount := c.PPresentModeCount().MustRead(ctx, c, g, b)
	if (c.PPresentModes()) != (VkPresentModeKHRᵖ(0)) {
		c.PPresentModes().Slice(0, uint64(givenCount), l).OnRead(ctx, c, g, b)
	}
	c.Call(ctx, g, b)
	c.Extras().Observations().ApplyWrites(g.Memory.ApplicationPool())
	if (c.PPresentModes()) == (VkPresentModeKHRᵖ(0)) {
		c.PPresentModeCount().MustWrite(ctx, c.PPresentModeCount().MustRead(ctx, c, g, nil), c, g, b)
	} else {
		count := c.PPresentModeCount().MustRead(ctx, c, g, nil)
		modes := c.PPresentModes().Slice(0, uint64(count), l)
		for i := uint32(0); i < count; i++ {
			modes.Index(uint64(i)).MustWrite(ctx, []VkPresentModeKHR{c.PPresentModes().Slice(0, uint64(count), l).Index(uint64(i)).MustRead(ctx, c, g, nil)[0]}, c, g, b)
		}
		c.PPresentModeCount().MustWrite(ctx, count, c, g, b)
	}
	return nil
}

func (a *VkGetFenceStatus) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	cb := CommandBuilder{Thread: a.Thread(), Arena: s.Arena}
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}

	return cb.ReplayGetFenceStatus(a.Device(), a.Fence(), a.Result(), a.Result()).Mutate(ctx, id, s, b)
}

func (a *VkGetEventStatus) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	cb := CommandBuilder{Thread: a.Thread(), Arena: s.Arena}
	err := a.mutate(ctx, id, s, b)
	if b == nil || err != nil {
		return err
	}
	var wait bool
	switch a.Result() {
	case VkResult_VK_EVENT_SET:
		wait = GetState(s).Events().Get(a.Event()).Signaled() == true
	case VkResult_VK_EVENT_RESET:
		wait = GetState(s).Events().Get(a.Event()).Signaled() == false
	default:
		wait = false
	}

	return cb.ReplayGetEventStatus(a.Device(), a.Event(), a.Result(), wait, a.Result()).Mutate(ctx, id, s, b)
}

func (a *ReplayAllocateImageMemory) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder) error {
	if err := a.mutate(ctx, id, s, b); err != nil {
		return err
	}
	l := s.MemoryLayout
	c := GetState(s)
	arena := s.Arena // TODO: Should this be a seperate temporary arena?
	memory := a.PMemory().Slice(0, 1, l).MustRead(ctx, a, s, nil)[0]
	imageObject := c.Images().Get(a.Image())
	imageWidth := imageObject.Info().Extent().Width()
	imageHeight := imageObject.Info().Extent().Height()
	imageFormat, err := getImageFormatFromVulkanFormat(imageObject.Info().Fmt())
	imageSize := VkDeviceSize(imageFormat.Size(int(imageWidth), int(imageHeight), 1))
	memoryObject := NewDeviceMemoryObjectʳ(arena,
		a.Device(),                  // Device
		memory,                      // VulkanHandle
		imageSize,                   // AllocationSize
		NewU64ːVkDeviceSizeᵐ(arena), // BoundObjects
		0, // MappedOffset
		0, // MappedSize
		0, // MappedLocation
		0, // MemoryTypeIndex
		MakeU8ˢ(uint64(imageSize), s),     // Data
		NilVulkanDebugMarkerInfoʳ,         // DebugInfo
		NilMemoryDedicatedAllocationInfoʳ, // DedicatedAllocationNV
		NilMemoryDedicatedAllocationInfoʳ, // DedicatedAllocationKHR
	)

	c.DeviceMemories().Add(memory, memoryObject)
	a.PMemory().Slice(0, 1, l).Write(ctx, []VkDeviceMemory{memory}, a, s, b)
	return err
}

func (i AllocationCallbacks) value(b *builder.Builder, cmd api.Cmd, s *api.GlobalState) value.Value {
	// Return 0 (nullptr) here. We don't have an allocator set up for replay. Since we cannot use the
	// application's allocator. If we pass in null for all allocator calls, then it will use the default
	// allocator.
	return value.AbsolutePointer(0)
}
