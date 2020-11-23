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
	"sort"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

const (
	queueFamilyIgnore = uint32(0xFFFFFFFF)
)

type stateBuilder struct {
	ctx                   context.Context
	s                     *State
	oldState              *api.GlobalState
	newState              *api.GlobalState
	out                   stateBuilderOutput
	cb                    CommandBuilder
	readMemories          []*api.AllocResult
	writeMemories         []*api.AllocResult
	extraReadIDsAndRanges []idAndRng
	memoryIntervals       interval.U64RangeList
	scratchRes            *scratchResources
}

type stateBuilderOutput interface {
	write(ctx context.Context, cmd api.Cmd, id api.CmdID)
	getOldState() *api.GlobalState
	getNewState() *api.GlobalState
}

type initialStateOutput struct {
	oldState *api.GlobalState
	newState *api.GlobalState
	cmds     []api.Cmd
}

func newInitialStateOutput(oldState *api.GlobalState) *initialStateOutput {
	return &initialStateOutput{
		oldState: oldState,
		newState: api.NewStateWithAllocator(memory.NewBasicAllocator(
			oldState.Allocator.FreeList()), oldState.MemoryLayout),
		cmds: []api.Cmd{},
	}
}

func (o *initialStateOutput) write(ctx context.Context, cmd api.Cmd, id api.CmdID) {
	if err := cmd.Mutate(ctx, id, o.newState, nil, nil); err != nil {
		log.W(ctx, "Initial cmd %v: %v - %v", len(o.cmds), cmd, err)
	} else {
		log.D(ctx, "Initial cmd %v: %v", len(o.cmds), cmd)
	}
	o.cmds = append(o.cmds, cmd)
}

func (o *initialStateOutput) getOldState() *api.GlobalState {
	return o.oldState
}

func (o *initialStateOutput) getNewState() *api.GlobalState {
	return o.newState
}

type transformerOutput struct {
	out transform.Writer
}

func newTransformerOutput(out transform.Writer) *transformerOutput {
	return &transformerOutput{out}
}

func (o *transformerOutput) write(ctx context.Context, cmd api.Cmd, id api.CmdID) {
	o.out.MutateAndWrite(ctx, id, cmd)
}

func (o *transformerOutput) getOldState() *api.GlobalState {
	return o.out.State()
}

func (o *transformerOutput) getNewState() *api.GlobalState {
	return o.out.State()
}

type idAndRng struct {
	id  id.ID
	rng memory.Range
}

func (s *State) newStateBuilder(ctx context.Context, out stateBuilderOutput) *stateBuilder {
	newState := out.getNewState()
	return &stateBuilder{
		ctx:             ctx,
		s:               s,
		oldState:        out.getOldState(),
		newState:        newState,
		out:             out,
		cb:              CommandBuilder{Thread: 0},
		memoryIntervals: interval.U64RangeList{},
		scratchRes:      newScratchResources(),
	}
}

// RebuildState returns a set of commands which, if executed on a new clean
// state, will reproduce the API's state in s.
// The segments of memory that were used to create these commands are returned
// in the rangeList.
func (API) RebuildState(ctx context.Context, oldState *api.GlobalState) ([]api.Cmd, interval.U64RangeList) {
	s, hasState := oldState.APIs[ID].(*State)
	if !hasState {
		return nil, nil
	}

	// TODO: wherever possible, use old resources instead of doing full reads on
	// the old pools. This is especially useful for things that are internal
	// pools, (Shader words for example)

	// TODO: Debug Info
	out := newInitialStateOutput(oldState)
	sb := s.newStateBuilder(ctx, out)

	sb.newState.Memory.NewAt(sb.oldState.Memory.NextPoolID())

	for _, k := range s.Instances().Keys() {
		sb.createInstance(k, s.Instances().Get(k))
	}

	sb.createPhysicalDevices(s.PhysicalDevices())

	for _, su := range s.Surfaces().Keys() {
		sb.createSurface(s.Surfaces().Get(su))
	}

	for _, d := range s.Devices().Keys() {
		sb.createDevice(s.Devices().Get(d))
	}

	for _, q := range s.Queues().Keys() {
		sb.createQueue(s.Queues().Get(q))
	}

	for _, swp := range s.Swapchains().Keys() {
		sb.createSwapchain(s.Swapchains().Get(swp))
	}

	// Create all non-dedicated allocations.
	// Dedicated allocations will be created with their
	// objects
	for _, mem := range s.DeviceMemories().Keys() {
		// TODO: Handle KHR dedicated allocation as well as NV
		sb.createDeviceMemory(s.DeviceMemories().Get(mem), false)
	}

	for _, buf := range s.Buffers().Keys() {
		sb.createBuffer(s.Buffers().Get(buf))
	}
	imgPrimer := newImagePrimer(sb)
	for _, img := range s.Images().Keys() {
		subRange, err := sb.createImage(s.Images().Get(img), sb.oldState, img)
		if len(subRange) != 0 && err == nil {
			sb.primeImage(s.Images().Get(img), imgPrimer, subRange)
		}
	}

	for _, conv := range s.SamplerYcbcrConversions().Keys() {
		sb.createSamplerYcbcrConversion(s.SamplerYcbcrConversions().Get(conv))
	}

	for _, smp := range s.Samplers().Keys() {
		sb.createSampler(s.Samplers().Get(smp))
	}

	for _, fnc := range s.Fences().Keys() {
		sb.createFence(s.Fences().Get(fnc))
	}

	for _, sem := range s.Semaphores().Keys() {
		sb.createSemaphore(s.Semaphores().Get(sem))
	}

	for _, evt := range s.Events().Keys() {
		sb.createEvent(s.Events().Get(evt))
	}

	for _, cp := range s.CommandPools().Keys() {
		sb.createCommandPool(s.CommandPools().Get(cp))
	}

	for _, pc := range s.PipelineCaches().Keys() {
		sb.createPipelineCache(s.PipelineCaches().Get(pc))
	}

	for _, dsl := range s.DescriptorSetLayouts().Keys() {
		sb.createDescriptorSetLayout(s.DescriptorSetLayouts().Get(dsl))
	}

	for _, dut := range s.DescriptorUpdateTemplates().Keys() {
		sb.createDescriptorUpdateTemplate(s.DescriptorUpdateTemplates().Get(dut))
	}

	for _, pl := range s.PipelineLayouts().Keys() {
		sb.createPipelineLayout(s.PipelineLayouts().Get(pl))
	}

	for _, rp := range s.RenderPasses().Keys() {
		sb.createRenderPass(s.RenderPasses().Get(rp))
	}

	for _, sm := range s.ShaderModules().Keys() {
		sb.createShaderModule(s.ShaderModules().Get(sm))
	}

	for _, cp := range getPipelinesInOrder(s, true) {
		sb.createComputePipeline(s.ComputePipelines().Get(cp))
	}

	for _, gp := range getPipelinesInOrder(s, false) {
		sb.createGraphicsPipeline(s.GraphicsPipelines().Get(gp))
	}

	for _, iv := range s.ImageViews().Keys() {
		sb.createImageView(s.ImageViews().Get(iv))
	}

	for _, bv := range s.BufferViews().Keys() {
		sb.createBufferView(s.BufferViews().Get(bv))
	}

	for _, dp := range s.DescriptorPools().Keys() {
		sb.createDescriptorPoolAndAllocateDescriptorSets(s.DescriptorPools().Get(dp))
	}

	for _, fb := range s.Framebuffers().Keys() {
		sb.createFramebuffer(s.Framebuffers().Get(fb))
	}

	for _, ds := range s.DescriptorSets().Keys() {
		sb.writeDescriptorSet(s.DescriptorSets().Get(ds))
	}

	for _, qp := range s.QueryPools().Keys() {
		sb.createQueryPool(s.QueryPools().Get(qp))
	}

	for _, cb := range s.CommandBuffers().Keys() {
		sb.createCommandBuffer(s.CommandBuffers().Get(cb), VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_SECONDARY)
		sb.recordCommandBuffer(s.CommandBuffers().Get(cb), VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_SECONDARY, sb.oldState)
	}

	for _, cb := range s.CommandBuffers().Keys() {
		sb.createCommandBuffer(s.CommandBuffers().Get(cb), VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY)
		sb.recordCommandBuffer(s.CommandBuffers().Get(cb), VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY, sb.oldState)
	}

	sb.scratchRes.Free(sb)
	imgPrimer.Free()
	return out.cmds, sb.memoryIntervals
}

func getPipelinesInOrder(s *State, compute bool) []VkPipeline {
	pipelines := []VkPipeline{}
	unhandledPipelines := map[VkPipeline]VkPipeline{}
	handledPipelines := map[VkPipeline]bool{}
	if compute {
		for _, p := range s.ComputePipelines().Keys() {
			pp := s.ComputePipelines().Get(p)
			unhandledPipelines[pp.VulkanHandle()] = pp.BasePipeline()
		}
	} else {
		for _, p := range s.GraphicsPipelines().Keys() {
			pp := s.GraphicsPipelines().Get(p)
			unhandledPipelines[pp.VulkanHandle()] = pp.BasePipeline()
		}
	}

	// In the following, make sure to always iterate over the
	// unhandledPipelines map in order, to make state rebuilding
	// deterministic.
	keys := make([]VkPipeline, 0, len(unhandledPipelines))
	for len(unhandledPipelines) != 0 {
		numHandled := 0
		keys = []VkPipeline{}
		for k := range unhandledPipelines {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		for _, k := range keys {
			v := unhandledPipelines[k]
			handled := false
			if v == 0 {
				pipelines = append(pipelines, k)
				handled = true
			} else if _, ok := handledPipelines[v]; ok {
				pipelines = append(pipelines, k)
				handled = true
			}
			if handled {
				handledPipelines[k] = true
				delete(unhandledPipelines, k)
				numHandled++
			}
		}
		if numHandled == 0 {
			// There is a cycle in the basePipeline indices.
			// Or no base pipelines does exist.
			// Create the rest without base pipelines
			keys = []VkPipeline{}
			for k := range unhandledPipelines {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
			for _, k := range keys {
				pipelines = append(pipelines, k)
			}
			unhandledPipelines = map[VkPipeline]VkPipeline{}
			break
		}
	}
	return pipelines
}

func (sb *stateBuilder) MustAllocReadData(v ...interface{}) api.AllocResult {
	allocateResult := sb.newState.AllocDataOrPanic(sb.ctx, v...)
	sb.readMemories = append(sb.readMemories, &allocateResult)
	rng := allocateResult.Range()
	interval.Merge(&sb.memoryIntervals, interval.U64Span{rng.Base, rng.Base + rng.Size}, true)
	return allocateResult
}

func (sb *stateBuilder) MustAllocWriteData(v ...interface{}) api.AllocResult {
	allocateResult := sb.newState.AllocDataOrPanic(sb.ctx, v...)
	sb.writeMemories = append(sb.writeMemories, &allocateResult)
	rng := allocateResult.Range()
	interval.Merge(&sb.memoryIntervals, interval.U64Span{rng.Base, rng.Base + rng.Size}, true)
	return allocateResult
}

func (sb *stateBuilder) MustUnpackReadMap(v interface{}) api.AllocResult {
	allocateResult, _ := unpackMap(sb.ctx, sb.newState, v)
	sb.readMemories = append(sb.readMemories, &allocateResult)
	rng := allocateResult.Range()
	interval.Merge(&sb.memoryIntervals, interval.U64Span{rng.Base, rng.Base + rng.Size}, true)
	return allocateResult
}

func (sb *stateBuilder) MustUnpackWriteMap(v interface{}) api.AllocResult {
	allocateResult, _ := unpackMap(sb.ctx, sb.newState, v)
	sb.writeMemories = append(sb.writeMemories, &allocateResult)
	rng := allocateResult.Range()
	interval.Merge(&sb.memoryIntervals, interval.U64Span{rng.Base, rng.Base + rng.Size}, true)
	return allocateResult
}

func (sb *stateBuilder) MustReserve(size uint64) api.AllocResult {
	res := sb.newState.AllocOrPanic(sb.ctx, size)
	interval.Merge(&sb.memoryIntervals, res.Range().Span(), true)
	return res
}

func (sb *stateBuilder) ReadDataAt(dataID id.ID, base, size uint64) {
	rng := memory.Range{base, size}
	interval.Merge(&sb.memoryIntervals, rng.Span(), true)
	sb.extraReadIDsAndRanges = append(sb.extraReadIDsAndRanges, idAndRng{
		id:  dataID,
		rng: rng,
	})
}

type sliceWithID interface {
	memory.Slice
	ResourceID(ctx context.Context, state *api.GlobalState) id.ID
}

func (sb *stateBuilder) mustReadSlice(v sliceWithID) api.AllocResult {
	res := sb.MustReserve(v.Size())
	sb.readMemories = append(sb.readMemories, &res)
	sb.ReadDataAt(v.ResourceID(sb.ctx, sb.oldState), res.Address(), v.Size())
	return res
}

func (sb *stateBuilder) endSubmitAndDestroyCommandBuffer(queue QueueObjectʳ, commandBuffer VkCommandBuffer, commandPool VkCommandPool) {
	sb.write(sb.cb.VkEndCommandBuffer(
		commandBuffer,
		VkResult_VK_SUCCESS,
	))

	sb.write(sb.cb.VkQueueSubmit(
		queue.VulkanHandle(),
		1,
		sb.MustAllocReadData(NewVkSubmitInfo(
			VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
			0, // pNext
			0, // waitSemaphoreCount
			0, // pWaitSemaphores
			0, // pWaitDstStageMask
			1, // commandBufferCount
			NewVkCommandBufferᶜᵖ(sb.MustAllocReadData(commandBuffer).Ptr()), // pCommandBuffers
			0, // signalSemaphoreCount
			0, // pSignalSemaphores
		)).Ptr(),
		VkFence(0),
		VkResult_VK_SUCCESS,
	))

	sb.write(sb.cb.VkQueueWaitIdle(queue.VulkanHandle(), VkResult_VK_SUCCESS))
	sb.write(sb.cb.VkDestroyCommandPool(
		queue.Device(),
		commandPool,
		memory.Nullptr,
	))
}

func (sb *stateBuilder) write(cmd api.Cmd) {
	for _, read := range sb.readMemories {
		cmd.Extras().GetOrAppendObservations().AddRead(read.Data())
	}
	for _, ir := range sb.extraReadIDsAndRanges {
		cmd.Extras().GetOrAppendObservations().AddRead(ir.rng, ir.id)
	}
	for _, write := range sb.writeMemories {
		cmd.Extras().GetOrAppendObservations().AddWrite(write.Data())
	}

	sb.out.write(sb.ctx, cmd, api.CmdNoID)
	for _, read := range sb.readMemories {
		read.Free()
	}
	for _, write := range sb.writeMemories {
		write.Free()
	}
	sb.readMemories = []*api.AllocResult{}
	sb.writeMemories = []*api.AllocResult{}
	sb.extraReadIDsAndRanges = []idAndRng{}
}

func (sb *stateBuilder) createInstance(vk VkInstance, inst InstanceObjectʳ) {
	enabledLayers := []Charᶜᵖ{}
	for _, k := range inst.EnabledLayers().Keys() {
		layer := inst.EnabledLayers().Get(k)
		enabledLayers = append(enabledLayers, NewCharᶜᵖ(sb.MustAllocReadData(layer).Ptr()))
	}
	enabledExtensions := []Charᶜᵖ{}
	for _, k := range inst.EnabledExtensions().Keys() {
		ext := inst.EnabledExtensions().Get(k)
		enabledExtensions = append(enabledExtensions, NewCharᶜᵖ(sb.MustAllocReadData(ext).Ptr()))
	}

	appInfo := memory.Nullptr
	if !inst.ApplicationInfo().IsNil() {
		appName := memory.Nullptr
		if inst.ApplicationInfo().ApplicationName() != "" {
			appName = NewCharᶜᵖ(sb.MustAllocReadData(inst.ApplicationInfo().ApplicationName()).Ptr())
		}
		engineName := memory.Nullptr
		if inst.ApplicationInfo().EngineName() != "" {
			engineName = NewCharᶜᵖ(sb.MustAllocReadData(inst.ApplicationInfo().EngineName()).Ptr())
		}
		appInfo = sb.MustAllocReadData(
			NewVkApplicationInfo(
				VkStructureType_VK_STRUCTURE_TYPE_APPLICATION_INFO, // sType
				0,                  // pNext
				NewCharᶜᵖ(appName), // applicationName
				inst.ApplicationInfo().ApplicationVersion(), // applicationVersion
				NewCharᶜᵖ(engineName),                       // engineName
				inst.ApplicationInfo().EngineVersion(),      // engineVersion
				inst.ApplicationInfo().ApiVersion(),         // apiVersion
			)).Ptr()
	}

	sb.write(sb.cb.VkCreateInstance(
		sb.MustAllocReadData(NewVkInstanceCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO, // sType
			0,                                  // pNext
			0,                                  // flags
			NewVkApplicationInfoᶜᵖ(appInfo),    // pApplicationInfo
			uint32(inst.EnabledLayers().Len()), // enabledLayerCount
			NewCharᶜᵖᶜᵖ(sb.MustAllocReadData(enabledLayers).Ptr()),     // ppEnabledLayerNames
			uint32(inst.EnabledExtensions().Len()),                     // enabledExtensionCount
			NewCharᶜᵖᶜᵖ(sb.MustAllocReadData(enabledExtensions).Ptr()), // ppEnabledExtensionNames
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(vk).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func (sb *stateBuilder) createPhysicalDevices(Map VkPhysicalDeviceːPhysicalDeviceObjectʳᵐ) {
	devices := map[VkInstance][]VkPhysicalDevice{}
	for _, k := range Map.Keys() {
		v := Map.Get(k)
		_, ok := devices[v.Instance()]
		if !ok {
			devices[v.Instance()] = []VkPhysicalDevice{}
		}

		devices[v.Instance()] = append(devices[v.Instance()], k)
	}

	for i, devs := range devices {
		sb.write(sb.cb.VkEnumeratePhysicalDevices(
			i,
			NewU32ᶜᵖ(sb.MustAllocWriteData(len(devs)).Ptr()),
			NewVkPhysicalDeviceᵖ(memory.Nullptr),
			VkResult_VK_SUCCESS,
		))
		props := MakePhysicalDevicesAndProperties()
		formatProps := MakePhysicalDevicesFormatProperties()
		for _, dev := range devs {
			v := Map.Get(dev)
			props.PhyDevToProperties().Add(dev, v.PhysicalDeviceProperties())
			formatProps.PhyDevToFormatProperties().Add(dev, v.FormatProperties())
		}
		enumerateWithProps := sb.cb.VkEnumeratePhysicalDevices(
			i,
			NewU32ᶜᵖ(sb.MustAllocReadData(len(devs)).Ptr()),
			NewVkPhysicalDeviceᵖ(sb.MustAllocReadData(devs).Ptr()),
			VkResult_VK_SUCCESS,
		)
		enumerateWithProps.Extras().Add(props)
		enumerateWithProps.Extras().Add(formatProps)
		sb.write(enumerateWithProps)

		for _, device := range devs {
			pd := Map.Get(device)
			sb.write(sb.cb.VkGetPhysicalDeviceProperties(
				device,
				NewVkPhysicalDevicePropertiesᵖ(sb.MustAllocWriteData(pd.PhysicalDeviceProperties()).Ptr()),
			))
			sb.write(sb.cb.VkGetPhysicalDeviceMemoryProperties(
				device,
				NewVkPhysicalDeviceMemoryPropertiesᵖ(sb.MustAllocWriteData(pd.MemoryProperties()).Ptr()),
			))
			sb.write(sb.cb.VkGetPhysicalDeviceQueueFamilyProperties(
				device,
				NewU32ᶜᵖ(sb.MustAllocWriteData(pd.QueueFamilyProperties().Len()).Ptr()),
				NewVkQueueFamilyPropertiesᵖ(memory.Nullptr),
			))
			sb.write(sb.cb.VkGetPhysicalDeviceQueueFamilyProperties(
				device,
				NewU32ᶜᵖ(sb.MustAllocReadData(pd.QueueFamilyProperties().Len()).Ptr()),
				NewVkQueueFamilyPropertiesᵖ(sb.MustUnpackWriteMap(pd.QueueFamilyProperties()).Ptr()),
			))
		}
	}
}

func (sb *stateBuilder) createSurface(s SurfaceObjectʳ) {
	switch s.Type() {
	case SurfaceType_SURFACE_TYPE_XCB:
		sb.write(sb.cb.VkCreateXcbSurfaceKHR(
			s.Instance(),
			sb.MustAllocReadData(NewVkXcbSurfaceCreateInfoKHR(
				VkStructureType_VK_STRUCTURE_TYPE_XCB_SURFACE_CREATE_INFO_KHR, // sType
				0, // pNext
				0, // flags
				0, // connection
				0, // window
			)).Ptr(),
			memory.Nullptr,
			sb.MustAllocWriteData(s.VulkanHandle()).Ptr(),
			VkResult_VK_SUCCESS,
		))
	case SurfaceType_SURFACE_TYPE_ANDROID:
		sb.write(sb.cb.VkCreateAndroidSurfaceKHR(
			s.Instance(),
			sb.MustAllocReadData(NewVkAndroidSurfaceCreateInfoKHR(
				VkStructureType_VK_STRUCTURE_TYPE_ANDROID_SURFACE_CREATE_INFO_KHR, // sType
				0, // pNext
				0, // flags
				0, // window
			)).Ptr(),
			memory.Nullptr,
			sb.MustAllocWriteData(s.VulkanHandle()).Ptr(),
			VkResult_VK_SUCCESS,
		))
	case SurfaceType_SURFACE_TYPE_WIN32:
		sb.write(sb.cb.VkCreateWin32SurfaceKHR(
			s.Instance(),
			sb.MustAllocReadData(NewVkWin32SurfaceCreateInfoKHR(
				VkStructureType_VK_STRUCTURE_TYPE_WIN32_SURFACE_CREATE_INFO_KHR, // sType
				0, // pNext
				0, // flags
				0, // hinstance
				0, // hwnd
			)).Ptr(),
			memory.Nullptr,
			sb.MustAllocWriteData(s.VulkanHandle()).Ptr(),
			VkResult_VK_SUCCESS,
		))
	case SurfaceType_SURFACE_TYPE_WAYLAND:
		sb.write(sb.cb.VkCreateWaylandSurfaceKHR(
			s.Instance(),
			sb.MustAllocReadData(NewVkWaylandSurfaceCreateInfoKHR(
				VkStructureType_VK_STRUCTURE_TYPE_WAYLAND_SURFACE_CREATE_INFO_KHR, // sType
				0, // pNext
				0, // flags
				0, // display
				0, // surface
			)).Ptr(),
			memory.Nullptr,
			sb.MustAllocWriteData(s.VulkanHandle()).Ptr(),
			VkResult_VK_SUCCESS,
		))
	case SurfaceType_SURFACE_TYPE_XLIB:
		sb.write(sb.cb.VkCreateXlibSurfaceKHR(
			s.Instance(),
			sb.MustAllocReadData(NewVkXlibSurfaceCreateInfoKHR(
				VkStructureType_VK_STRUCTURE_TYPE_XLIB_SURFACE_CREATE_INFO_KHR, // sType
				0, // pNext
				0, // flags
				0, // dpy
				0, // window
			)).Ptr(),
			memory.Nullptr,
			sb.MustAllocWriteData(s.VulkanHandle()).Ptr(),
			VkResult_VK_SUCCESS,
		))
	case SurfaceType_SURFACE_TYPE_GGP:
		sb.write(sb.cb.VkCreateStreamDescriptorSurfaceGGP(
			s.Instance(),
			sb.MustAllocReadData(NewVkStreamDescriptorSurfaceCreateInfoGGP(
				VkStructureType_VK_STRUCTURE_TYPE_STREAM_DESCRIPTOR_SURFACE_CREATE_INFO_GGP,
				0, // pNext
				0, // flags
				0, // streamDescriptor
			)).Ptr(),
			memory.Nullptr,
			sb.MustAllocWriteData(s.VulkanHandle()).Ptr(),
			VkResult_VK_SUCCESS,
		))
	case SurfaceType_SURFACE_TYPE_MACOS_MVK:
		sb.write(sb.cb.VkCreateMacOSSurfaceMVK(
			s.Instance(),
			sb.MustAllocReadData(NewVkMacOSSurfaceCreateInfoMVK(
				VkStructureType_VK_STRUCTURE_TYPE_MACOS_SURFACE_CREATE_INFO_MVK, // sType
				0, // pNext
				0, // flags
				0, // window
			)).Ptr(),
			memory.Nullptr,
			sb.MustAllocWriteData(s.VulkanHandle()).Ptr(),
			VkResult_VK_SUCCESS,
		))
	}
	for _, phyDev := range s.PhysicalDeviceSupports().Keys() {
		familyIndices := s.PhysicalDeviceSupports().Get(phyDev)
		for _, index := range familyIndices.QueueFamilySupports().Keys() {
			supported := familyIndices.QueueFamilySupports().Get(index)
			sb.write(sb.cb.VkGetPhysicalDeviceSurfaceSupportKHR(
				phyDev,
				index,
				s.VulkanHandle(),
				sb.MustAllocWriteData(supported).Ptr(),
				VkResult_VK_SUCCESS,
			))
		}
	}
}

func (sb *stateBuilder) createDevice(d DeviceObjectʳ) {
	enabledLayers := []Charᶜᵖ{}
	for _, k := range d.EnabledLayers().Keys() {
		layer := d.EnabledLayers().Get(k)
		enabledLayers = append(enabledLayers, NewCharᶜᵖ(sb.MustAllocReadData(layer).Ptr()))
	}
	enabledExtensions := []Charᶜᵖ{}
	for _, k := range d.EnabledExtensions().Keys() {
		ext := d.EnabledExtensions().Get(k)
		enabledExtensions = append(enabledExtensions, NewCharᶜᵖ(sb.MustAllocReadData(ext).Ptr()))
	}

	queueCreate := map[uint32]VkDeviceQueueCreateInfo{}
	queuePriorities := map[uint32][]float32{}

	for _, k := range d.Queues().Keys() {
		q := d.Queues().Get(k)
		if _, ok := queueCreate[q.QueueFamilyIndex()]; !ok {
			queueCreate[q.QueueFamilyIndex()] = NewVkDeviceQueueCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO, // sType
				0,                    // pNext
				0,                    // flags
				q.QueueFamilyIndex(), // queueFamilyIndex
				0,                    // queueCount
				0,                    // pQueuePriorities - This gets filled in later
			)
			queuePriorities[q.QueueFamilyIndex()] = []float32{}
		}
		x := queueCreate[q.QueueFamilyIndex()]
		x.SetQueueCount(x.QueueCount() + 1)
		queueCreate[q.QueueFamilyIndex()] = x
		if uint32(len(queuePriorities[q.QueueFamilyIndex()])) < q.QueueIndex()+1 {
			t := make([]float32, q.QueueIndex()+1)
			copy(t, queuePriorities[q.QueueFamilyIndex()])
			queuePriorities[q.QueueFamilyIndex()] = t
		}
		queuePriorities[q.QueueFamilyIndex()][q.QueueIndex()] = q.Priority()
	}
	reorderedQueueCreates := map[uint32]VkDeviceQueueCreateInfo{}
	{
		keys := []uint32{}
		for k := range queueCreate {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		i := uint32(0)
		for _, k := range keys {
			v := queueCreate[k]
			v.SetPQueuePriorities(NewF32ᶜᵖ(sb.MustAllocReadData(queuePriorities[k]).Ptr()))
			reorderedQueueCreates[i] = v
			i++
		}
	}

	pNext := NewVoidᵖ(memory.Nullptr)
	if !d.VariablePointerFeatures().IsNil() {
		pNext = NewVoidᵖ(sb.MustAllocReadData(
			NewVkPhysicalDeviceVariablePointerFeatures(
				VkStructureType_VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VARIABLE_POINTER_FEATURES, // sType
				pNext, // pNext
				d.VariablePointerFeatures().VariablePointersStorageBuffer(), // variablePointersStorageBuffer
				d.VariablePointerFeatures().VariablePointers(),              // variablePointers
			),
		).Ptr())
	}
	if !d.HalfPrecisionStorageFeatures().IsNil() {
		pNext = NewVoidᵖ(sb.MustAllocReadData(
			NewVkPhysicalDevice16BitStorageFeatures(
				VkStructureType_VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_16BIT_STORAGE_FEATURES, // sType
				pNext, // pNext
				d.HalfPrecisionStorageFeatures().StorageBuffer16BitAccess(),           // storageBuffer16BitAccess
				d.HalfPrecisionStorageFeatures().UniformAndStorageBuffer16BitAccess(), // uniformAndStorageBuffer16BitAccess
				d.HalfPrecisionStorageFeatures().StoragePushConstant16(),              // storagePushConstant16
				d.HalfPrecisionStorageFeatures().StorageInputOutput16(),               // storageInputOutput16
			),
		).Ptr())
	}
	if !d.Vk8BitStorageFeatures().IsNil() {
		pNext = NewVoidᵖ(sb.MustAllocReadData(
			NewVkPhysicalDevice8BitStorageFeaturesKHR(
				VkStructureType_VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_8BIT_STORAGE_FEATURES_KHR, // sType
				pNext, // pNext
				d.Vk8BitStorageFeatures().StorageBuffer8BitAccess(),           // storageBuffer8BitAccess
				d.Vk8BitStorageFeatures().UniformAndStorageBuffer8BitAccess(), // uniformAndStorageBuffer8BitAccess
				d.Vk8BitStorageFeatures().StoragePushConstant8(),              // storagePushConstant8
			),
		).Ptr())
	}
	if !d.SamplerYcbcrConversionFeatures().IsNil() {
		pNext = NewVoidᵖ(sb.MustAllocReadData(
			NewVkPhysicalDeviceSamplerYcbcrConversionFeatures(
				VkStructureType_VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SAMPLER_YCBCR_CONVERSION_FEATURES, // sType
				pNext, // pNext
				d.SamplerYcbcrConversionFeatures().SamplerYcbcrConversion(), // samplerYcbcrConversion
			),
		).Ptr())
	}
	if !d.PhysicalDeviceScalarBlockLayoutFeaturesEXT().IsNil() {
		pNext = NewVoidᵖ(sb.MustAllocReadData(
			NewVkPhysicalDeviceScalarBlockLayoutFeaturesEXT(
				VkStructureType_VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SCALAR_BLOCK_LAYOUT_FEATURES_EXT, // sType
				pNext, // pNext
				d.PhysicalDeviceScalarBlockLayoutFeaturesEXT().ScalarBlockLayout(), // samplerYcbcrConversion
			),
		).Ptr())
	}
	if d.HostQueryReset() != 0 {
		pNext = NewVoidᵖ(sb.MustAllocReadData(
			NewVkPhysicalDeviceHostQueryResetFeaturesEXT(
				VkStructureType_VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_HOST_QUERY_RESET_FEATURES_EXT, // sType
				pNext,              // pNext
				d.HostQueryReset(), // hostQueryReset
			),
		).Ptr())
	}
	if !d.PhysicalDeviceUniformBufferStandardLayoutFeaturesKHR().IsNil() {
		pNext = NewVoidᵖ(sb.MustAllocReadData(
			NewVkPhysicalDeviceUniformBufferStandardLayoutFeaturesKHR(
				VkStructureType_VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_UNIFORM_BUFFER_STANDARD_LAYOUT_FEATURES_KHR, // sType
				pNext, // pNext
				d.PhysicalDeviceUniformBufferStandardLayoutFeaturesKHR().UniformBufferStandardLayout(), // uniformBufferStandardLayout
			),
		).Ptr())
	}
	if !d.PhysicalDeviceShaderSubgroupExtendedTypesFeaturesKHR().IsNil() {
		pNext = NewVoidᵖ(sb.MustAllocReadData(
			NewVkPhysicalDeviceShaderSubgroupExtendedTypesFeaturesKHR(
				VkStructureType_VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SHADER_SUBGROUP_EXTENDED_TYPES_FEATURES_KHR, // sType
				pNext, // pNext
				d.PhysicalDeviceShaderSubgroupExtendedTypesFeaturesKHR().ShaderSubgroupExtendedTypes(), // shaderSubgroupExtendedTypes
			),
		).Ptr())
	}
	if !d.PhysicalDeviceShaderClockFeaturesKHR().IsNil() {
		pNext = NewVoidᵖ(sb.MustAllocReadData(
			NewVkPhysicalDeviceShaderClockFeaturesKHR(
				VkStructureType_VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SHADER_CLOCK_FEATURES_KHR, // sType
				pNext, // pNext
				d.PhysicalDeviceShaderClockFeaturesKHR().ShaderSubgroupClock(), // shaderSubgroupClock
				d.PhysicalDeviceShaderClockFeaturesKHR().ShaderDeviceClock(),   // shaderDeviceClock
			),
		).Ptr())
	}
	if !d.PhysicalDeviceVulkanMemoryModelFeaturesKHR().IsNil() {
		pNext = NewVoidᵖ(sb.MustAllocReadData(
			NewVkPhysicalDeviceVulkanMemoryModelFeaturesKHR(
				VkStructureType_VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_MEMORY_MODEL_FEATURES_KHR, // sType
				pNext, // pNext
				d.PhysicalDeviceVulkanMemoryModelFeaturesKHR().VulkanMemoryModel(),                             // vulkanMemoryModel
				d.PhysicalDeviceVulkanMemoryModelFeaturesKHR().VulkanMemoryModelDeviceScope(),                  // vulkanMemoryModelDeviceScope
				d.PhysicalDeviceVulkanMemoryModelFeaturesKHR().VulkanMemoryModelAvailabilityVisibilityChains(), // vulkanMemoryModelAvailabilityVisibilityChains
			),
		).Ptr())
	}
	if !d.PhysicalDeviceShaderFloat16Int8FeaturesKHR().IsNil() {
		pNext = NewVoidᵖ(sb.MustAllocReadData(
			NewVkPhysicalDeviceShaderFloat16Int8FeaturesKHR(
				VkStructureType_VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_FLOAT_CONTROLS_PROPERTIES_KHR, // sType
				pNext, // pNext
				d.PhysicalDeviceShaderFloat16Int8FeaturesKHR().ShaderFloat16(), // shaderFloat16
				d.PhysicalDeviceShaderFloat16Int8FeaturesKHR().ShaderInt8(),    // shaderInt8
			),
		).Ptr())
	}

	sb.write(sb.cb.VkCreateDevice(
		d.PhysicalDevice(),
		sb.MustAllocReadData(NewVkDeviceCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO, // sType
			NewVoidᶜᵖ(pNext),                   // pNext
			0,                                  // flags
			uint32(len(reorderedQueueCreates)), // queueCreateInfoCount
			NewVkDeviceQueueCreateInfoᶜᵖ(sb.MustUnpackReadMap(reorderedQueueCreates).Ptr()), // pQueueCreateInfos
			uint32(len(enabledLayers)),                                                     // enabledLayerCount
			NewCharᶜᵖᶜᵖ(sb.MustAllocReadData(enabledLayers).Ptr()),                         // ppEnabledLayerNames
			uint32(len(enabledExtensions)),                                                 // enabledExtensionCount
			NewCharᶜᵖᶜᵖ(sb.MustAllocReadData(enabledExtensions).Ptr()),                     // ppEnabledExtensionNames
			NewVkPhysicalDeviceFeaturesᶜᵖ(sb.MustAllocReadData(d.EnabledFeatures()).Ptr()), // pEnabledFeatures
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(d.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func (sb *stateBuilder) createQueue(q QueueObjectʳ) {
	sb.write(sb.cb.VkGetDeviceQueue(
		q.Device(),
		q.Family(),
		q.Index(),
		sb.MustAllocWriteData(q.VulkanHandle()).Ptr(),
	))
}

type imageQueueFamilyTransferInfo struct {
	image      VkImage
	aspectMask VkImageAspectFlags
	layer      uint32
	level      uint32
	layout     VkImageLayout
	oldQueue   VkQueue
	newQueue   VkQueue
}

func (sb *stateBuilder) transferImageQueueFamilyOwnership(infos ...imageQueueFamilyTransferInfo) error {
	makeBarrier := func(info imageQueueFamilyTransferInfo) VkImageMemoryBarrier {
		newFamily := GetState(sb.newState).Queues().Get(info.newQueue).Family()
		oldFamily := newFamily
		if info.oldQueue != VkQueue(0) {
			oldFamily = sb.s.Queues().Get(info.oldQueue).Family()
		}
		return NewVkImageMemoryBarrier(
			VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
			0, // pNext
			VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // srcAccessMask
			VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // dstAccessMask
			info.layout, // oldLayout
			info.layout, // newLayout
			oldFamily,   // srcQueueFamilyIndex
			newFamily,   // dstQueueFamilyIndex
			info.image,  // image
			NewVkImageSubresourceRange(
				info.aspectMask,
				info.level,
				1,
				info.layer,
				1,
			), // subresourceRange
		)
	}

	releaseBarriers := map[VkQueue][]VkImageMemoryBarrier{}
	acquireBarriers := map[VkQueue][]VkImageMemoryBarrier{}
	for _, info := range infos {
		if info.oldQueue == VkQueue(0) {
			// no need to transfer
			continue
		}
		oldFamily := GetState(sb.newState).Queues().Get(info.oldQueue).Family()
		newFamily := GetState(sb.newState).Queues().Get(info.newQueue).Family()
		if oldFamily == newFamily {
			// no need to transfer
			continue
		}
		releaseBarriers[info.oldQueue] = append(releaseBarriers[info.oldQueue], makeBarrier(info))
		acquireBarriers[info.newQueue] = append(acquireBarriers[info.newQueue], makeBarrier(info))
	}

	releaseHandlers := make([]*queueCommandHandler, 0, len(releaseBarriers))
	for releaseQ, barriers := range releaseBarriers {
		releaseHandler := sb.scratchRes.GetQueueCommandHandler(sb, releaseQ)
		err := ipRecordImageMemoryBarriers(sb, releaseHandler, barriers...)
		if err != nil {
			return log.Errf(sb.ctx, err, "Error at recording queue family ownership releasing barriers")
		}
		releaseHandlers = append(releaseHandlers, releaseHandler)
	}
	// force synchronization to make sure the releasing is done
	for _, h := range releaseHandlers {
		h.Submit(sb)
	}
	for _, h := range releaseHandlers {
		h.WaitUntilFinish(sb)
	}

	for acquireQ, barriers := range acquireBarriers {
		acquireHandler := sb.scratchRes.GetQueueCommandHandler(sb, acquireQ)
		err := ipRecordImageMemoryBarriers(sb, acquireHandler, barriers...)
		if err != nil {
			return log.Errf(sb.ctx, err, "Error at recording queue family ownership acquiring barriers")
		}
	}
	return nil
}

func (sb *stateBuilder) transitImageLayoutTransferImageOwnership(image VkImage)

type imageSubRangeInfo struct {
	aspectMask     VkImageAspectFlags
	baseMipLevel   uint32
	levelCount     uint32
	baseArrayLayer uint32
	layerCount     uint32
	oldLayout      VkImageLayout
	newLayout      VkImageLayout
	oldQueue       VkQueue
	newQueue       VkQueue
}

func (sb *stateBuilder) createSwapchain(swp SwapchainObjectʳ) {
	extent := NewVkExtent2D(
		swp.Info().Extent().Width(),
		swp.Info().Extent().Height(),
	)

	pNext := NewVoidᶜᵖ(memory.Nullptr)
	if !swp.Info().ViewFormatList().IsNil() {
		pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
			NewVkImageFormatListCreateInfoKHR(
				VkStructureType_VK_STRUCTURE_TYPE_IMAGE_FORMAT_LIST_CREATE_INFO_KHR, // sType
				pNext, // pNext
				uint32(swp.Info().ViewFormatList().ViewFormats().Len()),                                    // viewFormatCount
				NewVkFormatᶜᵖ(sb.MustUnpackReadMap(swp.Info().ViewFormatList().ViewFormats().All()).Ptr()), // pViewFormats
			),
		).Ptr())
	}

	sb.write(sb.cb.VkCreateSwapchainKHR(
		swp.Device(),
		sb.MustAllocReadData(NewVkSwapchainCreateInfoKHR(
			VkStructureType_VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR, // sType
			0,                                   // pNext
			swp.Flags(),                         // flags
			swp.Surface().VulkanHandle(),        // surface
			uint32(swp.SwapchainImages().Len()), // minImageCount
			swp.Info().Fmt(),                    // imageFormat
			swp.ColorSpace(),                    // imageColorSpace
			extent,                              // imageExtent
			swp.Info().ArrayLayers(),            // imageArrayLayers
			swp.Info().Usage(),                  // imageUsage
			swp.Info().SharingMode(),            // imageSharingMode
			uint32(swp.Info().QueueFamilyIndices().Len()),                         // queueFamilyIndexCount
			NewU32ᶜᵖ(sb.MustUnpackReadMap(swp.Info().QueueFamilyIndices()).Ptr()), // pQueueFamilyIndices
			swp.PreTransform(),   // preTransform
			swp.CompositeAlpha(), // compositeAlpha
			swp.PresentMode(),    // presentMode
			swp.Clipped(),        // clipped
			0,                    // oldSwapchain
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(swp.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))

	if !swp.HDRMetadata().IsNil() {
		sb.write(sb.cb.VkSetHdrMetadataEXT(
			swp.Device(),
			1,
			sb.MustAllocReadData(swp.VulkanHandle()).Ptr(),
			sb.MustAllocReadData(NewVkHdrMetadataEXT(
				VkStructureType_VK_STRUCTURE_TYPE_HDR_METADATA_EXT,
				0, // pNext
				swp.HDRMetadata().DisplayPrimaryRed(),
				swp.HDRMetadata().DisplayPrimaryGreen(),
				swp.HDRMetadata().DisplayPrimaryBlue(),
				swp.HDRMetadata().WhitePoint(),
				swp.HDRMetadata().MaxLuminance(),
				swp.HDRMetadata().MinLuminance(),
				swp.HDRMetadata().MaxContentLightLevel(),
				swp.HDRMetadata().MaxFrameAverageLightLevel(),
			)).Ptr(),
		))
	}

	images := []VkImage{}
	for _, v := range swp.SwapchainImages().Keys() {
		images = append(images, swp.SwapchainImages().Get(v).VulkanHandle())
	}

	sb.write(sb.cb.VkGetSwapchainImagesKHR(
		swp.Device(),
		swp.VulkanHandle(),
		NewU32ᶜᵖ(sb.MustAllocReadData(uint32(swp.SwapchainImages().Len())).Ptr()),
		sb.MustAllocWriteData(images).Ptr(),
		VkResult_VK_SUCCESS,
	))

	for _, k := range swp.SwapchainImages().Keys() {
		v := swp.SwapchainImages().Get(k)
		layoutBarriers := ipImageLayoutTransitionBarriers(sb, v, useSpecifiedLayout(VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED), sameLayoutsOfImage(v))
		layoutQueue := sb.getQueueFor(
			VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT|VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT,
			queueFamilyIndicesToU32Slice(v.Info().QueueFamilyIndices()),
			v.Device(),
			v.LastBoundQueue())
		if layoutQueue.IsNil() {
			log.E(sb.ctx, "could not get a queue for swapchain image layout transition")
			return
		}
		queueHandler := sb.scratchRes.GetQueueCommandHandler(sb, layoutQueue.VulkanHandle())
		err := ipRecordImageMemoryBarriers(sb, queueHandler, layoutBarriers...)
		if err != nil {
			log.E(sb.ctx, "failed at swapchain image layout transition, err: %v", err)
		}

		ownerTransferInfo := []imageQueueFamilyTransferInfo{}
		walkImageSubresourceRange(sb, v, sb.imageWholeSubresourceRange(v),
			func(aspect VkImageAspectFlagBits, layer, level uint32, unused byteSizeAndExtent) {
				l := v.Aspects().Get(aspect).Layers().Get(layer).Levels().Get(level)
				if !l.LastBoundQueue().IsNil() && (layoutQueue.Family() != l.LastBoundQueue().Family()) {
					ownerTransferInfo = append(ownerTransferInfo, imageQueueFamilyTransferInfo{
						image:      v.VulkanHandle(),
						aspectMask: VkImageAspectFlags(aspect),
						layer:      layer,
						level:      level,
						layout:     l.Layout(),
						oldQueue:   layoutQueue.VulkanHandle(),
						newQueue:   l.LastBoundQueue().VulkanHandle(),
					})
				}
			})
		sb.transferImageQueueFamilyOwnership(ownerTransferInfo...)
	}
}

func (sb *stateBuilder) createDeviceMemory(mem DeviceMemoryObjectʳ, allowDedicatedNV bool) {
	if !allowDedicatedNV && !mem.DedicatedAllocationNV().IsNil() {
		return
	}

	pNext := NewVoidᶜᵖ(memory.Nullptr)

	if !mem.DedicatedAllocationNV().IsNil() {
		pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
			NewVkDedicatedAllocationMemoryAllocateInfoNV(
				VkStructureType_VK_STRUCTURE_TYPE_DEDICATED_ALLOCATION_MEMORY_ALLOCATE_INFO_NV, // sType
				0,                                    // pNext
				mem.DedicatedAllocationNV().Image(),  // image
				mem.DedicatedAllocationNV().Buffer(), // buffer
			),
		).Ptr())
	}

	if !mem.MemoryAllocateFlagsInfo().IsNil() {
		flags := mem.MemoryAllocateFlagsInfo()
		pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
			NewVkMemoryAllocateFlagsInfo(
				VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_FLAGS_INFO,
				pNext,              // pNext
				flags.Flags(),      // flags
				flags.DeviceMask(), // deviceMask
			),
		).Ptr())
	}

	sb.write(sb.cb.VkAllocateMemory(
		mem.Device(),
		NewVkMemoryAllocateInfoᶜᵖ(sb.MustAllocReadData(
			NewVkMemoryAllocateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, // sType
				pNext,                 // pNext
				mem.AllocationSize(),  // allocationSize
				mem.MemoryTypeIndex(), // memoryTypeIndex
			)).Ptr()),
		memory.Nullptr,
		sb.MustAllocWriteData(mem.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))

	if mem.MappedLocation().Address() != 0 {
		sb.write(sb.cb.VkMapMemory(
			mem.Device(),
			mem.VulkanHandle(),
			mem.MappedOffset(),
			mem.MappedSize(),
			VkMemoryMapFlags(0),
			NewVoidᵖᵖ(sb.MustAllocWriteData(mem.MappedLocation()).Ptr()),
			VkResult_VK_SUCCESS,
		))
	}
}

func (sb *stateBuilder) GetScratchBufferMemoryIndex(device DeviceObjectʳ) uint32 {
	physicalDeviceObject := sb.s.PhysicalDevices().Get(device.PhysicalDevice())

	typeBits := uint32((uint64(1) << uint64(physicalDeviceObject.MemoryProperties().MemoryTypeCount())) - 1)
	if sb.s.TransferBufferMemoryRequirements().Contains(device.VulkanHandle()) {
		typeBits = sb.s.TransferBufferMemoryRequirements().Get(device.VulkanHandle()).MemoryTypeBits()
	}
	index := memoryTypeIndexFor(typeBits, physicalDeviceObject.MemoryProperties(), VkMemoryPropertyFlags(VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT))
	if index >= 0 {
		return uint32(index)
	}
	log.E(sb.ctx, "cannnot get the memory type index for host visible memory to create scratch buffer, fallback to use index 0")
	return 0
}

// Find the index of the memory type that satisfies the specified memory property
// flags.
func memoryTypeIndexFor(memTypeBits uint32, props VkPhysicalDeviceMemoryProperties, flags VkMemoryPropertyFlags) int {
	for i := 0; i < int(props.MemoryTypeCount()); i++ {
		if (memTypeBits & (1 << uint(i))) == 0 {
			continue
		}
		t := props.MemoryTypes().Get(i)
		if flags == (t.PropertyFlags() & flags) {
			return i
		}
	}
	return -1
}

type bufferSubRangeFillInfo struct {
	rng        interval.U64Range // Do not use memory.Range because this is not a range in memory
	data       []uint8
	hash       id.ID
	hasNewData bool
}

func newBufferSubRangeFillInfoFromNewData(data []uint8, offsetInBuf uint64) bufferSubRangeFillInfo {
	return bufferSubRangeFillInfo{
		rng:        interval.U64Range{offsetInBuf, uint64(len(data))},
		data:       data,
		hash:       id.ID{},
		hasNewData: true,
	}
}

func newBufferSubRangeFillInfoFromSlice(sb *stateBuilder, slice U8ˢ, offsetInBuf uint64) bufferSubRangeFillInfo {
	return bufferSubRangeFillInfo{
		rng:        interval.U64Range{offsetInBuf, slice.Size()},
		data:       []uint8{},
		hash:       slice.ResourceID(sb.ctx, sb.oldState),
		hasNewData: false,
	}
}

func (i bufferSubRangeFillInfo) size() uint64 {
	return i.rng.Count
}

func (i *bufferSubRangeFillInfo) storeNewData(sb *stateBuilder) {
	if i.hasNewData {
		hash, err := database.Store(sb.ctx, i.data)
		if err != nil {
			panic(err)
		}
		i.hash = hash
		i.hasNewData = false
	}
}

func (i *bufferSubRangeFillInfo) setOffsetInBuffer(offsetInBuf uint64) {
	i.rng = interval.U64Range{offsetInBuf, i.size()}
}

// getQueueFor returns a queue object from the old state. The returned queue
// must 1) has ANY of the bits in the given queue flags, 2) is created with one
// of the given queue family indices, if the given queue family indices is not
// empty, 3) is created from the given VkDevice in the old state.
// The given candidates will be checked first in order, and if none of the
// candidates meets the requirements, select an eligible one from all the
// existing queues in the old state. Returns NilQueueObjectʳ if none of the
// existing queues is eligible.
func (sb *stateBuilder) getQueueFor(queueFlagBits VkQueueFlagBits, queueFamilyIndices []uint32, dev VkDevice, candidates ...QueueObjectʳ) QueueObjectʳ {
	indicesPass := func(q QueueObjectʳ) bool {
		if len(queueFamilyIndices) == 0 {
			return true
		}
		for _, i := range queueFamilyIndices {
			if q.Family() == i {
				return true
			}
		}
		return false
	}
	flagPass := func(q QueueObjectʳ) bool {
		dev := sb.s.Devices().Get(q.Device())
		phyDev := sb.s.PhysicalDevices().Get(dev.PhysicalDevice())
		familyProp := phyDev.QueueFamilyProperties().Get(q.Family())
		if uint32(familyProp.QueueFlags())&uint32(queueFlagBits) != 0 {
			return true
		}
		return false
	}

	for _, c := range candidates {
		if c.IsNil() {
			continue
		}
		if flagPass(c) && indicesPass(c) && c.Device() == dev {
			return c
		}
	}
	for _, k := range sb.s.Queues().Keys() {
		q := sb.s.Queues().Get(k)
		if flagPass(q) && indicesPass(q) && q.Device() == dev {
			return q
		}
	}
	return NilQueueObjectʳ
}

func (sb *stateBuilder) createBuffer(buffer BufferObjectʳ) {
	queue := sb.getQueueFor(
		VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT|VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT,
		queueFamilyIndicesToU32Slice(buffer.Info().QueueFamilyIndices()),
		buffer.Device(),
		buffer.LastBoundQueue())
	newBuffer := buffer.VulkanHandle()

	mem := GetState(sb.newState).DeviceMemories().Get(buffer.Memory().VulkanHandle())
	if err := sb.createSameBuffer(buffer, newBuffer, mem, buffer.MemoryOffset()); err != nil {
		log.E(sb.ctx, "create buffer %v failed", buffer)
		return
	}
	tsk := newQueueCommandBatch(
		fmt.Sprintf("Prime buffer: %v's data", newBuffer),
	)
	defer func() {
		if err := tsk.Commit(sb, sb.scratchRes.GetQueueCommandHandler(sb, queue.VulkanHandle())); err != nil {
			log.E(sb.ctx, "[Priming data for buffer: %v]: %v", buffer.VulkanHandle(), err)
		}
	}()
	dataBuffer, err := sb.loadHostDatatoStagingBuffer(buffer, tsk)
	if err != nil {
		log.E(sb.ctx, "[failed to load host data to staging buffer for buffer: %v]: %v", buffer.VulkanHandle(), err)
		return
	}

	sb.copyBuffer(dataBuffer, newBuffer, queue, tsk)
}

func nextMultipleOf(v, a uint64) uint64 {
	if a == 0 {
		return v
	}
	return (v + a - 1) / a * a
}

type byteSizeAndExtent struct {
	levelSize             uint64
	alignedLevelSize      uint64
	levelSizeInBuf        uint64
	alignedLevelSizeInBuf uint64
	width                 uint64
	height                uint64
	depth                 uint64
}

func (sb *stateBuilder) levelSize(extent VkExtent3D, format VkFormat, mipLevel uint32, aspect VkImageAspectFlagBits) byteSizeAndExtent {
	elementAndTexelBlockSize, _ :=
		subGetElementAndTexelBlockSizeForAspect(sb.ctx, nil, api.CmdNoID, nil, sb.oldState, nil, 0, nil, nil, format, aspect)

	texelWidth := elementAndTexelBlockSize.TexelBlockSize().Width()
	texelHeight := elementAndTexelBlockSize.TexelBlockSize().Height()

	width, _ := subGetMipSize(sb.ctx, nil, api.CmdNoID, nil, sb.oldState, nil, 0, nil, nil, extent.Width(), mipLevel)
	height, _ := subGetMipSize(sb.ctx, nil, api.CmdNoID, nil, sb.oldState, nil, 0, nil, nil, extent.Height(), mipLevel)
	depth, _ := subGetMipSize(sb.ctx, nil, api.CmdNoID, nil, sb.oldState, nil, 0, nil, nil, extent.Depth(), mipLevel)
	widthInBlocks, _ := subRoundUpTo(sb.ctx, nil, api.CmdNoID, nil, sb.oldState, nil, 0, nil, nil, width, texelWidth)
	heightInBlocks, _ := subRoundUpTo(sb.ctx, nil, api.CmdNoID, nil, sb.oldState, nil, 0, nil, nil, height, texelHeight)
	elementSize := elementAndTexelBlockSize.ElementSize()

	// The Depth element size might be different when it is in buffer instead of image.
	elementSizeInBuf := elementSize
	if aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT {
		elementSizeInBuf, _ = subGetDepthElementSize(sb.ctx, nil, api.CmdNoID, nil, sb.oldState, nil, 0, nil, nil, format, true)
	}

	size := uint64(widthInBlocks) * uint64(heightInBlocks) * uint64(depth) * uint64(elementSize)
	sizeInBuf := uint64(widthInBlocks) * uint64(heightInBlocks) * uint64(depth) * uint64(elementSizeInBuf)

	return byteSizeAndExtent{
		levelSize:             size,
		alignedLevelSize:      nextMultipleOf(size, 8),
		levelSizeInBuf:        sizeInBuf,
		alignedLevelSizeInBuf: nextMultipleOf(sizeInBuf, 8),
		width:                 uint64(width),
		height:                uint64(height),
		depth:                 uint64(depth),
	}
}

func (sb *stateBuilder) imageAspectFlagBits(img ImageObjectʳ, flag VkImageAspectFlags) []VkImageAspectFlagBits {
	bits := []VkImageAspectFlagBits{}
	b, _ := subUnpackImageAspectFlags(
		sb.ctx, nil, api.CmdNoID, nil, sb.oldState, GetState(sb.oldState), 0, nil, nil, img, flag)
	for _, k := range b.Keys() {
		bit := b.Get(k)
		bits = append(bits, bit)
	}
	return bits
}

// imageWholeSubresourceRange creates a VkImageSubresourceRange that covers the
// whole given image.
func (sb *stateBuilder) imageWholeSubresourceRange(img ImageObjectʳ) VkImageSubresourceRange {
	return NewVkImageSubresourceRange(
		img.ImageAspect(),        // aspectMask
		0,                        // baseMipLevel
		img.Info().MipLevels(),   // levelCount
		0,                        // baseArrayLayer
		img.Info().ArrayLayers(), // layerCount
	)
}

// imageAllBoundQueues returns the all the last bound queues for all the image
// subresource ranges (image level).
func (sb *stateBuilder) imageAllLastBoundQueues(img ImageObjectʳ) []VkQueue {
	seen := map[VkQueue]struct{}{}
	result := []VkQueue{}
	walkImageSubresourceRange(sb, img, sb.imageWholeSubresourceRange(img),
		func(aspect VkImageAspectFlagBits, layer, level uint32, unused byteSizeAndExtent) {
			// No need to handle for undefined layout
			imgLevel := img.Aspects().Get(aspect).Layers().Get(layer).Levels().Get(level)
			if imgLevel.LastBoundQueue().IsNil() {
				return
			}
			q := imgLevel.LastBoundQueue().VulkanHandle()
			if _, ok := seen[q]; !ok {
				seen[q] = struct{}{}
				result = append(result, q)
			}
		})
	return result
}

// IsFullyBound returns true if the resource range from offset with size is
// fully covered in the bindings. If the size ofd the resource range is 0,
// returns false.
func IsFullyBound(offset VkDeviceSize, size VkDeviceSize,
	bindings U64ːVkSparseMemoryBindᵐ) bool {
	if size == VkDeviceSize(0) {
		return false
	}
	resourceOffsets := bindings.Keys()
	coveredCounter := 0
	addAtBoundaries := map[uint64]int{}
	boundaries := make([]uint64, 0, len(resourceOffsets)*2)
	for o, b := range bindings.All() {
		// Skip the zero sized binds (they are actually invalid)
		bindSize := uint64(b.Size())
		bindBegin := uint64(o)
		bindEnd := bindBegin + bindSize
		if bindSize == uint64(0) {
			continue
		}
		// Get the number of binds that cover the start of the requested range.
		if bindBegin <= uint64(offset) && bindEnd > uint64(offset) {
			coveredCounter++
		}
		// Count the number of bind begin boundaries in the requested range.
		if bindBegin > uint64(offset) && bindBegin < uint64(offset+size) {
			boundaries = append(boundaries, bindBegin)
			if _, ok := addAtBoundaries[bindBegin]; !ok {
				addAtBoundaries[bindBegin] = 0
			}
			addAtBoundaries[bindBegin]++
		}
		// Count the number of bind end boundaries in the reqested range
		if bindEnd < uint64(offset+size) && bindEnd > uint64(offset) {
			boundaries = append(boundaries, bindEnd)
			if _, ok := addAtBoundaries[bindEnd]; !ok {
				addAtBoundaries[bindEnd] = 0
			}
			addAtBoundaries[bindEnd]--
		}
	}
	sort.Slice(boundaries, func(i, j int) bool {
		return boundaries[i] < boundaries[j]
	})

	// No bind covers the begin of the requested range.
	if coveredCounter == 0 {
		return false
	}

	// Scan through all the boundaries in the requested range. If the coveredCounter
	// drops to 0, it means a hole in the requested range.
	for _, b := range boundaries {
		coveredCounter += addAtBoundaries[b]
		if coveredCounter == 0 {
			return false
		}
	}
	return true
}

func (sb *stateBuilder) createImage(img ImageObjectʳ, srcState *api.GlobalState, vkImage VkImage) ([]VkImageSubresourceRange, error) {
	if img.IsSwapchainImage() {
		return nil, fmt.Errorf("swapchian image needs to be skipped")
	}

	vkCreateImage(sb, img.Device(), img.Info(), vkImage)
	planeMemInfo, _ := subGetImagePlaneMemoryInfo(sb.ctx, nil, api.CmdNoID, nil, srcState, GetState(srcState), 0, nil, nil, img, VkImageAspectFlagBits(0))
	planeMemRequirements := planeMemInfo.MemoryRequirements()
	vkGetImageMemoryRequirements(sb, img.Device(), vkImage, planeMemRequirements)

	denseBound := isDenseBound(img)
	sparseBound := isSparseBound(img)
	sparseResidency := isSparseResidency(img)

	// Dedicated allocation buffer/image must NOT be a sparse binding one.
	// Checking the dedicated allocation info on both the memory and the buffer
	// side, because we've found applications that do miss one of them.
	// TODO: Handle multi-planar images
	dedicatedMemoryNV := !planeMemInfo.BoundMemory().IsNil() && (!img.Info().DedicatedAllocationNV().IsNil() || !planeMemInfo.BoundMemory().DedicatedAllocationNV().IsNil())
	// Emit error message to report view if we found one of the dedicate allocation
	// info struct is missing.
	if dedicatedMemoryNV && img.Info().DedicatedAllocationNV().IsNil() {
		subVkErrorExpectNVDedicatedlyAllocatedHandle(sb.ctx, nil, api.CmdNoID, nil,
			srcState, GetState(srcState), 0, nil, nil, "VkImage", uint64(vkImage))
	}
	if dedicatedMemoryNV && planeMemInfo.BoundMemory().DedicatedAllocationNV().IsNil() {
		subVkErrorExpectNVDedicatedlyAllocatedHandle(sb.ctx, nil, api.CmdNoID, nil,
			srcState, GetState(srcState), 0, nil, nil, "VkDeviceMemory", uint64(planeMemInfo.BoundMemory().VulkanHandle()))
	}

	if dedicatedMemoryNV {
		sb.createDeviceMemory(planeMemInfo.BoundMemory(), true)
	}

	if !denseBound && !sparseBound {
		return nil, fmt.Errorf("image is neither dense bound nor spare bound")
	}

	var sparseQueue QueueObjectʳ
	opaqueRanges := []VkImageSubresourceRange{}
	// appendImageLevelToOpaqueRanges is a helper function to collect image levels
	// from the current processing source image that do not have an undefined
	// layout. The unused byteSizeAndExtent is to meet the requirement of
	// walkImageSubresourceRange()
	appendImageLevelToOpaqueRanges := func(aspect VkImageAspectFlagBits, layer, level uint32, unused byteSizeAndExtent) {
		imgLevel := img.Aspects().Get(aspect).Layers().Get(layer).Levels().Get(level)
		if imgLevel.Layout() == VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED {
			return
		}
		opaqueRanges = append(opaqueRanges, NewVkImageSubresourceRange(
			VkImageAspectFlags(aspect), // aspectMask
			level,                      // baseMipLevel
			1,                          // levelCount
			layer,                      // baseArrayLayer
			1,                          // layerCount
		))
	}

	if img.OpaqueSparseMemoryBindings().Len() > 0 || img.SparseImageMemoryBindings().Len() > 0 {
		// If this img has sparse memory bindings, then we have to set them all
		// now
		candidates := []QueueObjectʳ{}
		for _, q := range sb.imageAllLastBoundQueues(img) {
			candidates = append(candidates, sb.s.Queues().Get(q))
		}
		sparseQueue = sb.getQueueFor(
			VkQueueFlagBits_VK_QUEUE_SPARSE_BINDING_BIT,
			queueFamilyIndicesToU32Slice(img.Info().QueueFamilyIndices()),
			img.Device(), candidates...)

		memories := make(map[VkDeviceMemory]bool)

		nonSparseInfos := []VkSparseImageMemoryBind{}

		for _, aspect := range img.SparseImageMemoryBindings().Keys() {
			info := img.SparseImageMemoryBindings().Get(aspect)
			for _, layer := range info.Layers().Keys() {
				layerInfo := info.Layers().Get(layer)
				for _, level := range layerInfo.Levels().Keys() {
					levelInfo := layerInfo.Levels().Get(level)
					for _, k := range levelInfo.Blocks().Keys() {
						block := levelInfo.Blocks().Get(k)
						if !img.Info().DedicatedAllocationNV().IsNil() {
							// If this was a dedicated allocation set it here
							if _, ok := memories[block.Memory()]; !ok {
								memories[block.Memory()] = true
								sb.createDeviceMemory(sb.s.DeviceMemories().Get(block.Memory()), true)
							}
							nonSparseInfos = append(nonSparseInfos, NewVkSparseImageMemoryBind(
								NewVkImageSubresource( // subresource
									VkImageAspectFlags(aspect), // aspectMask
									level,                      // mipLevel
									layer,                      // arrayLayer
								),
								block.Offset(),       // offset
								block.Extent(),       // extent
								block.Memory(),       // memory
								block.MemoryOffset(), // memoryOffset
								block.Flags(),        // flags
							))
						}
					}
				}
			}
		}

		opaqueSparseBindings := make([]VkSparseMemoryBind, 0, img.OpaqueSparseMemoryBindings().Len())
		for _, k := range img.OpaqueSparseMemoryBindings().Keys() {
			obd := img.OpaqueSparseMemoryBindings().Get(k)
			opaqueSparseBindings = append(opaqueSparseBindings, obd)
		}

		sb.write(sb.cb.VkQueueBindSparse(
			sparseQueue.VulkanHandle(),
			1,
			sb.MustAllocReadData(
				NewVkBindSparseInfo(
					VkStructureType_VK_STRUCTURE_TYPE_BIND_SPARSE_INFO, // sType
					0, // pNext
					0, // waitSemaphoreCount
					0, // pWaitSemaphores
					0, // bufferBindCount
					0, // pBufferBinds
					1, // imageOpaqueBindCount
					NewVkSparseImageOpaqueMemoryBindInfoᶜᵖ(sb.MustAllocReadData( // pImageOpaqueBinds
						NewVkSparseImageOpaqueMemoryBindInfo(
							vkImage, // image
							uint32(img.OpaqueSparseMemoryBindings().Len()), // bindCount
							NewVkSparseMemoryBindᶜᵖ( // pBinds
								sb.MustAllocReadData(opaqueSparseBindings).Ptr(),
							),
						)).Ptr()),
					0, // imageBindCount
					NewVkSparseImageMemoryBindInfoᶜᵖ(sb.MustAllocReadData( // pImageBinds
						NewVkSparseImageMemoryBindInfo(
							vkImage,                     // image
							uint32(len(nonSparseInfos)), // bindCount
							NewVkSparseImageMemoryBindᶜᵖ( // pBinds
								sb.MustAllocReadData(nonSparseInfos).Ptr(),
							),
						)).Ptr()),
					0, // signalSemaphoreCount
					0, // pSignalSemaphores
				)).Ptr(),
			VkFence(0),
			VkResult_VK_SUCCESS,
		))

		if sparseResidency {
			isMetadataBound := false
			for _, k := range img.SparseMemoryRequirements().Keys() {
				req := img.SparseMemoryRequirements().Get(k)
				prop := req.FormatProperties()
				if uint64(prop.AspectMask())&uint64(VkImageAspectFlagBits_VK_IMAGE_ASPECT_METADATA_BIT) != 0 {
					isMetadataBound = IsFullyBound(req.ImageMipTailOffset(), req.ImageMipTailSize(), img.OpaqueSparseMemoryBindings())
				}
			}
			if !isMetadataBound {
				// If we have no metadata then the image can have no "real"
				// contents
			} else {
				for _, k := range img.SparseMemoryRequirements().Keys() {
					req := img.SparseMemoryRequirements().Get(k)
					prop := req.FormatProperties()
					if (uint64(prop.Flags()) & uint64(VkSparseImageFormatFlagBits_VK_SPARSE_IMAGE_FORMAT_SINGLE_MIPTAIL_BIT)) != 0 {
						if !IsFullyBound(req.ImageMipTailOffset(), req.ImageMipTailSize(), img.OpaqueSparseMemoryBindings()) {
							continue
						}
						subRng := NewVkImageSubresourceRange(
							img.ImageAspect(),                                 // aspectMask
							req.ImageMipTailFirstLod(),                        // baseMipLevel
							img.Info().MipLevels()-req.ImageMipTailFirstLod(), // levelCount
							0,                        // baseArrayLayer
							img.Info().ArrayLayers(), // layerCount
						)
						walkImageSubresourceRange(sb, img, subRng, appendImageLevelToOpaqueRanges)
					} else {
						for i := uint32(0); i < uint32(img.Info().ArrayLayers()); i++ {
							offset := req.ImageMipTailOffset() + VkDeviceSize(i)*req.ImageMipTailStride()
							if !IsFullyBound(offset, req.ImageMipTailSize(), img.OpaqueSparseMemoryBindings()) {
								continue
							}
							subRng := NewVkImageSubresourceRange(
								img.ImageAspect(),                                 // aspectMask
								req.ImageMipTailFirstLod(),                        // baseMipLevel
								img.Info().MipLevels()-req.ImageMipTailFirstLod(), // levelCount
								i, // baseArrayLayer
								1, // layerCount
							)
							walkImageSubresourceRange(sb, img, subRng, appendImageLevelToOpaqueRanges)
						}
					}
				}
			}
		} else {
			// TODO: Handle multi-planar images
			planeMemInfo, _ := subGetImagePlaneMemoryInfo(sb.ctx, nil, api.CmdNoID, nil, srcState, GetState(srcState), 0, nil, nil, img, VkImageAspectFlagBits(0))
			planeMemRequirements := planeMemInfo.MemoryRequirements()
			if IsFullyBound(0, planeMemRequirements.Size(), img.OpaqueSparseMemoryBindings()) {
				walkImageSubresourceRange(sb, img, sb.imageWholeSubresourceRange(img), appendImageLevelToOpaqueRanges)
			}
		}
	} else {
		// Otherwise, we have no sparse bindings, we are either non-sparse, or empty.
		if !isDenseBound(img) {
			return opaqueRanges, nil
		}
		walkImageSubresourceRange(sb, img, sb.imageWholeSubresourceRange(img), appendImageLevelToOpaqueRanges)
		// TODO: Handle multi-planar images
		planeMemInfo, _ := subGetImagePlaneMemoryInfo(sb.ctx, nil, api.CmdNoID, nil, srcState, GetState(srcState), 0, nil, nil, img, VkImageAspectFlagBits(0))

		if !planeMemInfo.ImageDeviceGroupBinding().IsNil() {
			dg := planeMemInfo.ImageDeviceGroupBinding()
			sb.write(sb.cb.VkBindImageMemory2(
				img.Device(),
				1,
				sb.MustAllocReadData(
					NewVkBindImageMemoryInfo(
						VkStructureType_VK_STRUCTURE_TYPE_BIND_IMAGE_MEMORY_INFO, // sType
						NewVoidᶜᵖ(sb.MustAllocReadData(
							NewVkBindImageMemoryDeviceGroupInfo(
								VkStructureType_VK_STRUCTURE_TYPE_BIND_IMAGE_MEMORY_DEVICE_GROUP_INFO, // sType
								NewVoidᶜᵖ(memory.Nullptr),
								uint32(dg.Bindings().Len()),
								NewU32ᶜᵖ(sb.MustUnpackReadMap(dg.Bindings().All()).Ptr()),
								uint32(dg.SplitInstanceBindings().Len()),
								NewVkRect2Dᶜᵖ(sb.MustUnpackReadMap(dg.SplitInstanceBindings().All()).Ptr()),
							),
						).Ptr()),
						img.VulkanHandle(),
						planeMemInfo.BoundMemory().VulkanHandle(),
						planeMemInfo.BoundMemoryOffset(),
					)).Ptr(),
				VkResult_VK_SUCCESS,
			))
		} else {
			vkBindImageMemory(sb, img.Device(), vkImage,
				planeMemInfo.BoundMemory().VulkanHandle(), planeMemInfo.BoundMemoryOffset())
		}
	}

	return opaqueRanges, nil
}

func (sb *stateBuilder) primeImage(img ImageObjectʳ, imgPrimer *imagePrimer, opaqueRanges []VkImageSubresourceRange) {
	// We don't currently prime the data in any of these formats.
	if img.Info().Samples() != VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT {
		layoutBarriers := ipImageLayoutTransitionBarriers(sb, img, useSpecifiedLayout(VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED), sameLayoutsOfImage(img))
		layoutQueue := sb.getQueueFor(
			VkQueueFlagBits_VK_QUEUE_TRANSFER_BIT|VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT,
			queueFamilyIndicesToU32Slice(img.Info().QueueFamilyIndices()),
			img.Device(), img.LastBoundQueue())
		if layoutQueue.IsNil() {
			log.E(sb.ctx, "could not get queue fro image layout transition: img %v", img.VulkanHandle())
			return
		}
		layoutHandler := sb.scratchRes.GetQueueCommandHandler(sb, layoutQueue.VulkanHandle())
		err := ipRecordImageMemoryBarriers(sb, layoutHandler, layoutBarriers...)
		if err != nil {
			log.E(sb.ctx, "could not record barriers to transition image layout for MS image, err: %v", err)
		}

		ownerTransferInfo := []imageQueueFamilyTransferInfo{}
		walkImageSubresourceRange(sb, img, sb.imageWholeSubresourceRange(img),
			func(aspect VkImageAspectFlagBits, layer, level uint32, unused byteSizeAndExtent) {
				// No need to handle for undefined layout
				imgLevel := img.Aspects().Get(aspect).Layers().Get(layer).Levels().Get(level)
				if imgLevel.Layout() == VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED || imgLevel.LastBoundQueue().IsNil() {
					return
				}
				if !imgLevel.LastBoundQueue().IsNil() && (imgLevel.LastBoundQueue().Family() != layoutQueue.Family()) {
					ownerTransferInfo = append(ownerTransferInfo, imageQueueFamilyTransferInfo{
						image:      img.VulkanHandle(),
						aspectMask: ipImageBarrierAspectFlags(aspect, img.Info().Fmt()),
						layer:      layer,
						level:      level,
						layout:     imgLevel.Layout(),
						oldQueue:   layoutQueue.VulkanHandle(),
						newQueue:   imgLevel.LastBoundQueue().VulkanHandle(),
					})
				}
			})
		err = sb.transferImageQueueFamilyOwnership(ownerTransferInfo...)
		if err != nil {
			log.E(sb.ctx, "failed at transferring queue family ownership for MS image, err: %v", err)
		}
		log.E(sb.ctx, "[Priming the data of image: %v] priming data for MS images not implemented", img.VulkanHandle())
		return
	}
	// We have to handle the above cases at some point.

	primeable, err := imgPrimer.newPrimeableImageDataFromHost(img.VulkanHandle(), opaqueRanges)
	if err != nil {
		log.E(sb.ctx, "Create primeable image data: %v", err)
		return
	}
	defer primeable.free(sb)
	err = primeable.prime(sb, useSpecifiedLayout(img.Info().InitialLayout()), sameLayoutsOfImage(img))
	if err != nil {
		log.E(sb.ctx, "Priming image data: %v", err)
		return
	}

	queue := GetState(sb.newState).Queues().Get(primeable.primingQueue())

	if !queue.IsNil() {
		ownerTransferInfo := []imageQueueFamilyTransferInfo{}
		walkImageSubresourceRange(sb, img, sb.imageWholeSubresourceRange(img),
			func(aspect VkImageAspectFlagBits, layer, level uint32, unused byteSizeAndExtent) {
				imgLevel := img.Aspects().Get(aspect).Layers().Get(layer).Levels().Get(level)
				if imgLevel.Layout() == VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED {
					return
				}
				toQ := img.LastBoundQueue()
				if toQ.IsNil() {
					toQ = imgLevel.LastBoundQueue()
				}
				if !toQ.IsNil() && queue.Family() != toQ.Family() {
					ownerTransferInfo = append(ownerTransferInfo, imageQueueFamilyTransferInfo{
						image:      img.VulkanHandle(),
						aspectMask: ipImageBarrierAspectFlags(aspect, img.Info().Fmt()),
						layer:      layer,
						level:      level,
						layout:     imgLevel.Layout(),
						oldQueue:   queue.VulkanHandle(),
						newQueue:   toQ.VulkanHandle(),
					})
				}
			})
		sb.transferImageQueueFamilyOwnership(ownerTransferInfo...)
	}
}

func (sb *stateBuilder) createSamplerYcbcrConversion(conv SamplerYcbcrConversionObjectʳ) {
	if conv.IsFromExtension() {
		sb.write(sb.cb.VkCreateSamplerYcbcrConversionKHR(
			conv.Device(),
			sb.MustAllocReadData(NewVkSamplerYcbcrConversionCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_SAMPLER_YCBCR_CONVERSION_CREATE_INFO_KHR, // sType
				0,                                  // pNext
				conv.Fmt(),                         // format
				conv.YcbcrModel(),                  // ycbcrModel
				conv.YcbcrRange(),                  // ycbcrRange
				conv.Components(),                  // components
				conv.XChromaOffset(),               // xChromaOffset
				conv.YChromaOffset(),               // yChromaOffset
				conv.ChromaFilter(),                // chromaFilter
				conv.ForceExplicitReconstruction(), // forceExplicitReconstruction
			)).Ptr(),
			memory.Nullptr,
			sb.MustAllocWriteData(conv.VulkanHandle()).Ptr(),
			VkResult_VK_SUCCESS,
		))
	} else {
		sb.write(sb.cb.VkCreateSamplerYcbcrConversion(
			conv.Device(),
			sb.MustAllocReadData(NewVkSamplerYcbcrConversionCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_SAMPLER_YCBCR_CONVERSION_CREATE_INFO, // sType
				0,                                  // pNext
				conv.Fmt(),                         // format
				conv.YcbcrModel(),                  // ycbcrModel
				conv.YcbcrRange(),                  // ycbcrRange
				conv.Components(),                  // components
				conv.XChromaOffset(),               // xChromaOffset
				conv.YChromaOffset(),               // yChromaOffset
				conv.ChromaFilter(),                // chromaFilter
				conv.ForceExplicitReconstruction(), // forceExplicitReconstruction
			)).Ptr(),
			memory.Nullptr,
			sb.MustAllocWriteData(conv.VulkanHandle()).Ptr(),
			VkResult_VK_SUCCESS,
		))
	}
}

func (sb *stateBuilder) createSampler(smp SamplerObjectʳ) {
	pNext := NewVoidᶜᵖ(memory.Nullptr)
	if !smp.YcbcrConversion().IsNil() {
		pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
			NewVkSamplerYcbcrConversionInfo(
				VkStructureType_VK_STRUCTURE_TYPE_SAMPLER_YCBCR_CONVERSION_INFO, // sType
				pNext,                                // pNext
				smp.YcbcrConversion().VulkanHandle(), // conversion
			),
		).Ptr())
	}

	sb.write(sb.cb.VkCreateSampler(
		smp.Device(),
		sb.MustAllocReadData(NewVkSamplerCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO, // sType
			pNext,                         // pNext
			0,                             // flags
			smp.MagFilter(),               // magFilter
			smp.MinFilter(),               // minFilter
			smp.MipMapMode(),              // mipmapMode
			smp.AddressModeU(),            // addressModeU
			smp.AddressModeV(),            // addressModeV
			smp.AddressModeW(),            // addressModeW
			smp.MipLodBias(),              // mipLodBias
			smp.AnisotropyEnable(),        // anisotropyEnable
			smp.MaxAnisotropy(),           // maxAnisotropy
			smp.CompareEnable(),           // compareEnable
			smp.CompareOp(),               // compareOp
			smp.MinLod(),                  // minLod
			smp.MaxLod(),                  // maxLod
			smp.BorderColor(),             // borderColor
			smp.UnnormalizedCoordinates(), // unnormalizedCoordinates
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(smp.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func (sb *stateBuilder) createFence(fnc FenceObjectʳ) {
	flags := VkFenceCreateFlags(0)
	if fnc.Signaled() {
		flags = VkFenceCreateFlags(VkFenceCreateFlagBits_VK_FENCE_CREATE_SIGNALED_BIT)
	}
	sb.write(sb.cb.VkCreateFence(
		fnc.Device(),
		sb.MustAllocReadData(NewVkFenceCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_FENCE_CREATE_INFO, // sType
			0,     // pNext
			flags, // flags
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(fnc.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func (sb *stateBuilder) createSemaphore(sem SemaphoreObjectʳ) {
	sb.write(sb.cb.VkCreateSemaphore(
		sem.Device(),
		sb.MustAllocReadData(NewVkSemaphoreCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO, // sType
			0, // pNext
			0, // flags
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(sem.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))

	if !sem.Signaled() {
		return
	}

	queue := sem.LastQueue()
	if !sb.s.Queues().Contains(queue) {
		// find a queue with the same device
		for _, k := range sb.s.Queues().Keys() {
			q := sb.s.Queues().Get(k)
			if q.Device() == sem.Device() {
				queue = q.VulkanHandle()
			}
		}
	}

	sb.write(sb.cb.VkQueueSubmit(
		queue,
		1,
		sb.MustAllocReadData(NewVkSubmitInfo(
			VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
			0, // pNext
			0, // waitSemaphoreCount
			0, // pWaitSemaphores
			0, // pWaitDstStageMask
			0, // commandBufferCount
			0, // pCommandBuffers
			1, // signalSemaphoreCount
			NewVkSemaphoreᶜᵖ(sb.MustAllocReadData(sem.VulkanHandle()).Ptr()), // pSignalSemaphores
		)).Ptr(),
		VkFence(0),
		VkResult_VK_SUCCESS,
	))
}

func (sb *stateBuilder) createEvent(evt EventObjectʳ) {
	sb.write(sb.cb.VkCreateEvent(
		evt.Device(),
		sb.MustAllocReadData(NewVkEventCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_EVENT_CREATE_INFO, // sType
			0, // pNext
			0, // flags
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(evt.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))

	if evt.Signaled() {
		sb.write(sb.cb.VkSetEvent(
			evt.Device(),
			evt.VulkanHandle(),
			VkResult_VK_SUCCESS,
		))
	}
}

func (sb *stateBuilder) createCommandPool(cp CommandPoolObjectʳ) {
	sb.write(sb.cb.VkCreateCommandPool(
		cp.Device(),
		sb.MustAllocReadData(NewVkCommandPoolCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO, // sType
			0,                     // pNext
			cp.Flags(),            // flags
			cp.QueueFamilyIndex(), // queueFamilyIndex
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(cp.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func (sb *stateBuilder) createPipelineCache(pc PipelineCacheObjectʳ) {
	sb.write(sb.cb.VkCreatePipelineCache(
		pc.Device(),
		sb.MustAllocReadData(NewVkPipelineCacheCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_CACHE_CREATE_INFO, // sType
			0,                             // pNext
			0,                             // flags
			memory.Size(pc.Data().Size()), // initialDataSize
			NewVoidᶜᵖ(sb.mustReadSlice(pc.Data()).Ptr()), // pInitialData
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(pc.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func (sb *stateBuilder) createDescriptorSetLayout(dsl DescriptorSetLayoutObjectʳ) {
	bindings := []VkDescriptorSetLayoutBinding{}
	for _, k := range dsl.Bindings().Keys() {
		b := dsl.Bindings().Get(k)
		smp := NewVkSamplerᶜᵖ(memory.Nullptr)
		if b.ImmutableSamplers().Len() > 0 {
			immutableSamplers := []VkSampler{}
			for _, kk := range b.ImmutableSamplers().Keys() {
				immutableSamplers = append(immutableSamplers, b.ImmutableSamplers().Get(kk).VulkanHandle())
			}
			allocateResult := sb.MustAllocReadData(immutableSamplers)
			smp = NewVkSamplerᶜᵖ(allocateResult.Ptr())
		}

		bindings = append(bindings, NewVkDescriptorSetLayoutBinding(
			k,          // binding
			b.Type(),   // descriptorType
			b.Count(),  // descriptorCount
			b.Stages(), // stageFlags
			smp,        // pImmutableSamplers
		))
	}

	sb.write(sb.cb.VkCreateDescriptorSetLayout(
		dsl.Device(),
		sb.MustAllocReadData(NewVkDescriptorSetLayoutCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO, // sType
			0,                     // pNext
			0,                     // flags
			uint32(len(bindings)), // bindingCount
			NewVkDescriptorSetLayoutBindingᶜᵖ( // pBindings
				sb.MustAllocReadData(bindings).Ptr(),
			),
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(dsl.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func (sb *stateBuilder) createDescriptorUpdateTemplate(dut DescriptorUpdateTemplateObjectʳ) {
	entries := []VkDescriptorUpdateTemplateEntry{}
	for _, k := range dut.Entries().Keys() {
		entry := dut.Entries().Get(k)
		entries = append(entries, entry)
	}

	createInfo := sb.MustAllocReadData(NewVkDescriptorUpdateTemplateCreateInfo(
		VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_UPDATE_TEMPLATE_CREATE_INFO, //sType
		0,                    // pNext
		0,                    // flags
		uint32(len(entries)), // descriptorUpdateEntryCount
		NewVkDescriptorUpdateTemplateEntryᶜᵖ( // pDescriptorUpdateEntries
			sb.MustAllocReadData(entries).Ptr(),
		),
		dut.TemplateType(),        // templateType
		dut.DescriptorSetLayout(), // descriptorSetLayout
		dut.PipelineBindPoint(),   // pipelineBindPoint
		dut.PipelineLayout(),      // pipelineLayout
		0,                         // set
	)).Ptr()

	if dut.FromKHR() {
		sb.write(sb.cb.VkCreateDescriptorUpdateTemplateKHR(
			dut.Device(),
			createInfo,
			memory.Nullptr, // pAllocator
			sb.MustAllocWriteData(dut.VulkanHandle()).Ptr(), // pDescriptorUpdateTemplate
			VkResult_VK_SUCCESS,
		))
	} else {
		sb.write(sb.cb.VkCreateDescriptorUpdateTemplate(
			dut.Device(),
			createInfo,
			memory.Nullptr, // pAllocator
			sb.MustAllocWriteData(dut.VulkanHandle()).Ptr(), // pDescriptorUpdateTemplate
			VkResult_VK_SUCCESS,
		))
	}
}

func (sb *stateBuilder) createPipelineLayout(pl PipelineLayoutObjectʳ) {

	temporaryDescriptorSetLayouts := []VkDescriptorSetLayout{}
	descriptorSets := []VkDescriptorSetLayout{}
	for _, k := range pl.SetLayouts().Keys() {
		if isDescriptorSetLayoutInState(pl.SetLayouts().Get(k), sb.oldState) {
			descriptorSets = append(descriptorSets, pl.SetLayouts().Get(k).VulkanHandle())
		} else {
			temporaryDescriptorSetLayout := pl.SetLayouts().Get(k).Clone(api.CloneContext{})
			temporaryDescriptorSetLayout.SetVulkanHandle(
				VkDescriptorSetLayout(newUnusedID(true, func(x uint64) bool {
					return GetState(sb.newState).DescriptorSetLayouts().Contains(VkDescriptorSetLayout(x))
				})))
			sb.createDescriptorSetLayout(temporaryDescriptorSetLayout)
			descriptorSets = append(descriptorSets, temporaryDescriptorSetLayout.VulkanHandle())
			temporaryDescriptorSetLayouts = append(temporaryDescriptorSetLayouts, temporaryDescriptorSetLayout.VulkanHandle())
		}
	}

	sb.write(sb.cb.VkCreatePipelineLayout(
		pl.Device(),
		sb.MustAllocReadData(NewVkPipelineLayoutCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO, // sType
			0,                           // pNext
			0,                           // flags
			uint32(len(descriptorSets)), // setLayoutCount
			NewVkDescriptorSetLayoutᶜᵖ( // pSetLayouts
				sb.MustAllocReadData(descriptorSets).Ptr(),
			),
			uint32(pl.PushConstantRanges().Len()),                                               // pushConstantRangeCount
			NewVkPushConstantRangeᶜᵖ(sb.MustUnpackReadMap(pl.PushConstantRanges().All()).Ptr()), // pPushConstantRanges
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(pl.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))

	for _, td := range temporaryDescriptorSetLayouts {
		sb.write(sb.cb.VkDestroyDescriptorSetLayout(
			pl.Device(),
			td,
			memory.Nullptr,
		))
	}
}

func (sb *stateBuilder) createRenderPass(rp RenderPassObjectʳ) {
	subpassDescriptions := []VkSubpassDescription{}
	for _, k := range rp.SubpassDescriptions().Keys() {
		sd := rp.SubpassDescriptions().Get(k)
		depthStencil := NewVkAttachmentReferenceᶜᵖ(memory.Nullptr)
		if !sd.DepthStencilAttachment().IsNil() {
			depthStencil = NewVkAttachmentReferenceᶜᵖ(sb.MustAllocReadData(sd.DepthStencilAttachment().Get()).Ptr())
		}
		resolveAttachments := NewVkAttachmentReferenceᶜᵖ(memory.Nullptr)
		if sd.ResolveAttachments().Len() > 0 {
			resolveAttachments = NewVkAttachmentReferenceᶜᵖ(sb.MustUnpackReadMap(sd.ResolveAttachments().All()).Ptr())
		}

		subpassDescriptions = append(subpassDescriptions, NewVkSubpassDescription(
			sd.Flags(),                          // flags
			sd.PipelineBindPoint(),              // pipelineBindPoint
			uint32(sd.InputAttachments().Len()), // inputAttachmentCount
			NewVkAttachmentReferenceᶜᵖ(sb.MustUnpackReadMap(sd.InputAttachments().All()).Ptr()), // pInputAttachments
			uint32(sd.ColorAttachments().Len()),                                                 // colorAttachmentCount
			NewVkAttachmentReferenceᶜᵖ(sb.MustUnpackReadMap(sd.ColorAttachments().All()).Ptr()), // pColorAttachments
			resolveAttachments,                     // pResolveAttachments
			depthStencil,                           // pDepthStencilAttachment
			uint32(sd.PreserveAttachments().Len()), // preserveAttachmentCount
			NewU32ᶜᵖ(sb.MustUnpackReadMap(sd.PreserveAttachments().All()).Ptr()), // pPreserveAttachments
		))
	}

	pNext := NewVoidᶜᵖ(memory.Nullptr)
	if !rp.InputAttachmentAspectInfo().IsNil() {
		pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
			NewVkRenderPassInputAttachmentAspectCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_RENDER_PASS_INPUT_ATTACHMENT_ASPECT_CREATE_INFO_KHR, // sType
				pNext, // pNext
				uint32(rp.InputAttachmentAspectInfo().AspectReferences().Len()), // aspectReferenceCount
				NewVkInputAttachmentAspectReferenceᶜᵖ(
					sb.MustUnpackReadMap(
						rp.InputAttachmentAspectInfo().AspectReferences().All(),
					).Ptr(),
				), // pAsepctReferences
			),
		).Ptr())
	}

	sb.write(sb.cb.VkCreateRenderPass(
		rp.Device(),
		sb.MustAllocReadData(NewVkRenderPassCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO, // sType
			pNext, // pNext
			0,     // flags
			uint32(rp.AttachmentDescriptions().Len()),                                                   // attachmentCount
			NewVkAttachmentDescriptionᶜᵖ(sb.MustUnpackReadMap(rp.AttachmentDescriptions().All()).Ptr()), // pAttachments
			uint32(len(subpassDescriptions)),                                                            // subpassCount
			NewVkSubpassDescriptionᶜᵖ(sb.MustAllocReadData(subpassDescriptions).Ptr()),                  // pSubpasses
			uint32(rp.SubpassDependencies().Len()),                                                      // dependencyCount
			NewVkSubpassDependencyᶜᵖ(sb.MustUnpackReadMap(rp.SubpassDependencies().All()).Ptr()),        // pDependencies
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(rp.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func (sb *stateBuilder) createShaderModule(sm ShaderModuleObjectʳ) {
	csm := sb.cb.VkCreateShaderModule(
		sm.Device(),
		sb.MustAllocReadData(NewVkShaderModuleCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO, // sType
			0,                                 // pNext
			0,                                 // flags
			memory.Size(sm.Words().Count()*4), // codeSize
			NewU32ᶜᵖ(sb.mustReadSlice(sm.Words()).Ptr()), // pCode
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(sm.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	)
	if !sm.Descriptors().IsNil() {
		sbExtra := sm.Descriptors().Get().Clone(api.CloneContext{})
		csm.Extras().Add(sbExtra)
	}
	sb.write(csm)
}

func isShaderModuleInState(sm ShaderModuleObjectʳ, st *api.GlobalState) bool {
	shaders := GetState(st).ShaderModules()
	if shaders.Contains(sm.VulkanHandle()) {
		if shaders.Get(sm.VulkanHandle()) == sm {
			return true
		}
	}
	return false
}

func isRenderPassInState(rp RenderPassObjectʳ, st *api.GlobalState) bool {
	passes := GetState(st).RenderPasses()
	if passes.Contains(rp.VulkanHandle()) {
		if passes.Get(rp.VulkanHandle()) == rp {
			return true
		}
	}
	return false
}

func isDescriptorSetLayoutInState(dl DescriptorSetLayoutObjectʳ, st *api.GlobalState) bool {
	layouts := GetState(st).DescriptorSetLayouts()
	if layouts.Contains(dl.VulkanHandle()) {
		if layouts.Get(dl.VulkanHandle()) == dl {
			return true
		}
	}
	return false
}

func isPipelineLayoutInState(pl PipelineLayoutObjectʳ, st *api.GlobalState) bool {
	layouts := GetState(st).PipelineLayouts()
	if layouts.Contains(pl.VulkanHandle()) {
		if layouts.Get(pl.VulkanHandle()) == pl {
			return true
		}
	}
	return false
}

func (sb *stateBuilder) createComputePipeline(cp ComputePipelineObjectʳ) {
	cache := VkPipelineCache(0)
	if !cp.PipelineCache().IsNil() {
		cache = cp.PipelineCache().VulkanHandle()
	}

	basePipeline := VkPipeline(0)
	if cp.BasePipeline() != 0 {
		if GetState(sb.newState).ComputePipelines().Contains(cp.BasePipeline()) {
			basePipeline = cp.BasePipeline()
		}
	}

	// Check if the shader module exist in the old state. If so, it MUST has
	// been handled by createShaderModule() BEFORE this function get called.
	// If not, create temporary shasder modules.
	var temporaryShaderModule ShaderModuleObjectʳ
	if !isShaderModuleInState(cp.Stage().Module(), sb.oldState) {
		// This is a previously deleted shader module, recreate it, then clear it
		temporaryShaderModule = cp.Stage().Module().Clone(api.CloneContext{})
		temporaryShaderModule.SetVulkanHandle(
			VkShaderModule(newUnusedID(true, func(x uint64) bool {
				return GetState(sb.newState).ShaderModules().Contains(VkShaderModule(x))
			})))
		sb.createShaderModule(temporaryShaderModule)
		cp.Stage().SetModule(temporaryShaderModule)
	}

	// Same as above, create temporary pipeline layout if it does not exist in
	// the old state.
	var temporaryPipelineLayout PipelineLayoutObjectʳ
	if !isPipelineLayoutInState(cp.PipelineLayout(), sb.oldState) {
		// create temporary pipeline layout for the pipeline to be created.
		temporaryPipelineLayout = cp.PipelineLayout().Clone(api.CloneContext{})
		temporaryPipelineLayout.SetVulkanHandle(
			VkPipelineLayout(newUnusedID(true, func(x uint64) bool {
				return GetState(sb.newState).PipelineLayouts().Contains(VkPipelineLayout(x))
			})))
		sb.createPipelineLayout(temporaryPipelineLayout)
		cp.SetPipelineLayout(temporaryPipelineLayout)
	}

	specializationInfo := NewVkSpecializationInfoᶜᵖ(memory.Nullptr)
	if !cp.Stage().Specialization().IsNil() {
		data := cp.Stage().Specialization().Data()
		specializationInfo = NewVkSpecializationInfoᶜᵖ(sb.MustAllocReadData(NewVkSpecializationInfo(
			uint32(cp.Stage().Specialization().Specializations().Len()),                                                    // mapEntryCount
			NewVkSpecializationMapEntryᶜᵖ(sb.MustUnpackReadMap(cp.Stage().Specialization().Specializations().All()).Ptr()), // pMapEntries
			memory.Size(data.Size()),                // dataSize
			NewVoidᶜᵖ(sb.mustReadSlice(data).Ptr()), // pData
		)).Ptr())
	}

	sb.write(sb.cb.VkCreateComputePipelines(
		cp.Device(),
		cache,
		1,
		sb.MustAllocReadData(NewVkComputePipelineCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO, // sType
			0,          // pNext
			cp.Flags(), // flags
			NewVkPipelineShaderStageCreateInfo( // stage
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO, // sType
				0,                                  // pNext
				0,                                  // flags
				cp.Stage().Stage(),                 // stage
				cp.Stage().Module().VulkanHandle(), // module
				NewCharᶜᵖ(sb.MustAllocReadData(cp.Stage().EntryPoint()).Ptr()), // pName
				specializationInfo, // pSpecializationInfo
			),
			cp.PipelineLayout().VulkanHandle(), // layout
			basePipeline,                       // basePipelineHandle
			-1,                                 // basePipelineIndex
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(cp.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))

	if !temporaryShaderModule.IsNil() {
		sb.write(sb.cb.VkDestroyShaderModule(
			temporaryShaderModule.Device(),
			temporaryShaderModule.VulkanHandle(),
			memory.Nullptr,
		))
	}
	if !temporaryPipelineLayout.IsNil() {
		sb.write(sb.cb.VkDestroyPipelineLayout(
			temporaryPipelineLayout.Device(),
			temporaryPipelineLayout.VulkanHandle(),
			memory.Nullptr,
		))
	}
}

func (sb *stateBuilder) createGraphicsPipeline(gp GraphicsPipelineObjectʳ) {
	cache := VkPipelineCache(0)
	if !gp.PipelineCache().IsNil() {
		cache = gp.PipelineCache().VulkanHandle()
	}

	basePipeline := VkPipeline(0)
	if gp.BasePipeline() != 0 {
		if GetState(sb.newState).GraphicsPipelines().Contains(gp.BasePipeline()) {
			basePipeline = gp.BasePipeline()
		}
	}

	stagesInOrder := gp.Stages().Keys()

	// Check if the shader modules exist in the old state. If so, they MUST have
	// been handled by createShaderModule() BEFORE reaching here. If not, create
	// temporary shader module here.
	temporaryShaderModules := []ShaderModuleObjectʳ{}
	for _, ss := range stagesInOrder {
		s := gp.Stages().Get(ss)
		if !isShaderModuleInState(s.Module(), sb.oldState) {
			// create temporary shader module the pipeline to be created.
			temporaryShaderModule := s.Module().Clone(api.CloneContext{})
			temporaryShaderModule.SetVulkanHandle(
				VkShaderModule(newUnusedID(true, func(x uint64) bool {
					return GetState(sb.newState).ShaderModules().Contains(VkShaderModule(x))
				})))
			sb.createShaderModule(temporaryShaderModule)
			s.SetModule(temporaryShaderModule)
			temporaryShaderModules = append(temporaryShaderModules, temporaryShaderModule)
		}
	}

	// Handled in the same way as the shader modules above.
	var temporaryPipelineLayout PipelineLayoutObjectʳ
	if !isPipelineLayoutInState(gp.Layout(), sb.oldState) {
		// create temporary pipeline layout for the pipeline to be created.
		temporaryPipelineLayout = gp.Layout().Clone(api.CloneContext{})
		temporaryPipelineLayout.SetVulkanHandle(
			VkPipelineLayout(newUnusedID(true, func(x uint64) bool {
				return GetState(sb.newState).PipelineLayouts().Contains(VkPipelineLayout(x))
			})))
		sb.createPipelineLayout(temporaryPipelineLayout)
		gp.SetLayout(temporaryPipelineLayout)
	}

	// Handled in the same way as the shader modules above.
	var temporaryRenderPass RenderPassObjectʳ
	if !isRenderPassInState(gp.RenderPass(), sb.oldState) {
		// create temporary renderpass for the pipeline to be created.
		temporaryRenderPass = gp.RenderPass().Clone(api.CloneContext{})
		temporaryRenderPass.SetVulkanHandle(
			VkRenderPass(newUnusedID(true, func(x uint64) bool {
				return GetState(sb.newState).RenderPasses().Contains(VkRenderPass(x))
			})))
		sb.createRenderPass(temporaryRenderPass)
		gp.SetRenderPass(temporaryRenderPass)
	}

	// DO NOT! coalesce the prevous calls with this one. createShaderModule()
	// makes calls which means pending read/write observations will get
	// shunted off with it instead of on the VkCreateGraphicsPipelines call
	stages := []VkPipelineShaderStageCreateInfo{}
	for _, ss := range stagesInOrder {
		s := gp.Stages().Get(ss)
		specializationInfo := NewVkSpecializationInfoᶜᵖ(memory.Nullptr)
		if !s.Specialization().IsNil() {
			data := s.Specialization().Data()
			specializationInfo = NewVkSpecializationInfoᶜᵖ(sb.MustAllocReadData(
				NewVkSpecializationInfo(
					uint32(s.Specialization().Specializations().Len()),                                                    // mapEntryCount
					NewVkSpecializationMapEntryᶜᵖ(sb.MustUnpackReadMap(s.Specialization().Specializations().All()).Ptr()), // pMapEntries
					memory.Size(data.Size()),                // dataSize
					NewVoidᶜᵖ(sb.mustReadSlice(data).Ptr()), // pData
				)).Ptr())
		}
		stages = append(stages, NewVkPipelineShaderStageCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO, // sType
			0,                         // pNext
			0,                         // flags
			s.Stage(),                 // stage
			s.Module().VulkanHandle(), // module
			NewCharᶜᵖ(sb.MustAllocReadData(s.EntryPoint()).Ptr()), // pName
			specializationInfo, // pSpecializationInfo
		))
	}

	tessellationState := NewVkPipelineTessellationStateCreateInfoᶜᵖ(memory.Nullptr)
	if !gp.TessellationState().IsNil() {
		pNext := NewVoidᶜᵖ(memory.Nullptr)
		if !gp.TessellationState().TessellationDomainOriginState().IsNil() {
			pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
				NewVkPipelineTessellationDomainOriginStateCreateInfo(
					VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_TESSELLATION_DOMAIN_ORIGIN_STATE_CREATE_INFO, // sType
					pNext, // pNext
					gp.TessellationState().TessellationDomainOriginState().DomainOrigin(), // usage
				),
			).Ptr())
		}
		tessellationState = NewVkPipelineTessellationStateCreateInfoᶜᵖ(sb.MustAllocReadData(
			NewVkPipelineTessellationStateCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_TESSELLATION_STATE_CREATE_INFO, // sType
				pNext, // pNext
				0,     // flags
				gp.TessellationState().PatchControlPoints(), // patchControlPoints
			)).Ptr())
	}

	viewportState := NewVkPipelineViewportStateCreateInfoᶜᵖ(memory.Nullptr)
	if !gp.ViewportState().IsNil() {
		viewports := NewVkViewportᶜᵖ(memory.Nullptr)
		if gp.ViewportState().Viewports().Len() > 0 {
			viewports = NewVkViewportᶜᵖ(sb.MustUnpackReadMap(gp.ViewportState().Viewports().All()).Ptr())
		}
		scissors := NewVkRect2Dᶜᵖ(memory.Nullptr)
		if gp.ViewportState().Scissors().Len() > 0 {
			scissors = NewVkRect2Dᶜᵖ(sb.MustUnpackReadMap(gp.ViewportState().Scissors().All()).Ptr())
		}

		viewportState = NewVkPipelineViewportStateCreateInfoᶜᵖ(sb.MustAllocReadData(
			NewVkPipelineViewportStateCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO, // sType
				0,                                  // pNext
				0,                                  // flags
				gp.ViewportState().ViewportCount(), // viewportCount
				viewports,                          // pViewports
				gp.ViewportState().ScissorCount(),  // scissorCount
				scissors,                           // pScissors
			)).Ptr())
	}

	multisampleState := NewVkPipelineMultisampleStateCreateInfoᶜᵖ(memory.Nullptr)
	if !gp.MultisampleState().IsNil() {
		sampleMask := NewVkSampleMaskᶜᵖ(memory.Nullptr)
		if gp.MultisampleState().SampleMask().Len() > 0 {
			sampleMask = NewVkSampleMaskᶜᵖ(sb.MustUnpackReadMap(gp.MultisampleState().SampleMask().All()).Ptr())
		}
		multisampleState = NewVkPipelineMultisampleStateCreateInfoᶜᵖ(sb.MustAllocReadData(
			NewVkPipelineMultisampleStateCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO, // sType
				0, // pNext
				0, // flags
				gp.MultisampleState().RasterizationSamples(),  // rasterizationSamples
				gp.MultisampleState().SampleShadingEnable(),   // sampleShadingEnable
				gp.MultisampleState().MinSampleShading(),      // minSampleShading
				sampleMask,                                    // pSampleMask
				gp.MultisampleState().AlphaToCoverageEnable(), // alphaToCoverageEnable
				gp.MultisampleState().AlphaToOneEnable(),      // alphaToOneEnable
			)).Ptr())
	}

	depthState := NewVkPipelineDepthStencilStateCreateInfoᶜᵖ(memory.Nullptr)
	if !gp.DepthState().IsNil() {
		depthState = NewVkPipelineDepthStencilStateCreateInfoᶜᵖ(sb.MustAllocReadData(
			NewVkPipelineDepthStencilStateCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO, // sType
				0,                                       // pNext
				0,                                       // flags
				gp.DepthState().DepthTestEnable(),       // depthTestEnable
				gp.DepthState().DepthWriteEnable(),      // depthWriteEnable
				gp.DepthState().DepthCompareOp(),        // depthCompareOp
				gp.DepthState().DepthBoundsTestEnable(), // depthBoundsTestEnable
				gp.DepthState().StencilTestEnable(),     // stencilTestEnable
				gp.DepthState().Front(),                 // front
				gp.DepthState().Back(),                  // back
				gp.DepthState().MinDepthBounds(),        // minDepthBounds
				gp.DepthState().MaxDepthBounds(),        // maxDepthBounds
			)).Ptr())
	}

	colorBlendState := NewVkPipelineColorBlendStateCreateInfoᶜᵖ(memory.Nullptr)
	if !gp.ColorBlendState().IsNil() {
		colorblendAttachments := NewVkPipelineColorBlendAttachmentStateᶜᵖ(memory.Nullptr)
		if gp.ColorBlendState().Attachments().Len() > 0 {
			colorblendAttachments = NewVkPipelineColorBlendAttachmentStateᶜᵖ(sb.MustUnpackReadMap(gp.ColorBlendState().Attachments().All()).Ptr())
		}
		colorBlendState = NewVkPipelineColorBlendStateCreateInfoᶜᵖ(sb.MustAllocReadData(
			NewVkPipelineColorBlendStateCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO, // sType
				0,                                    // pNext
				0,                                    // flags
				gp.ColorBlendState().LogicOpEnable(), // logicOpEnable
				gp.ColorBlendState().LogicOp(),       // logicOp
				uint32(gp.ColorBlendState().Attachments().Len()), // attachmentCount
				colorblendAttachments,                            // pAttachments
				gp.ColorBlendState().BlendConstants(),            // blendConstants
			)).Ptr())
	}

	dynamicState := NewVkPipelineDynamicStateCreateInfoᶜᵖ(memory.Nullptr)
	if !gp.DynamicState().IsNil() {
		dynamicStates := NewVkDynamicStateᶜᵖ(memory.Nullptr)
		if gp.DynamicState().DynamicStates().Len() > 0 {
			dynamicStates = NewVkDynamicStateᶜᵖ(sb.MustUnpackReadMap(gp.DynamicState().DynamicStates().All()).Ptr())
		}
		dynamicState = NewVkPipelineDynamicStateCreateInfoᶜᵖ(sb.MustAllocReadData(
			NewVkPipelineDynamicStateCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO, // sType
				0, // pNext
				0, // flags
				uint32(gp.DynamicState().DynamicStates().Len()), // dynamicStateCount
				dynamicStates, // pDynamicStates
			)).Ptr())
	}

	sb.write(sb.cb.VkCreateGraphicsPipelines(
		gp.Device(),
		cache,
		1,
		sb.MustAllocReadData(NewVkGraphicsPipelineCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO, // sType
			0,                   // pNext
			gp.Flags(),          // flags
			uint32(len(stages)), // stageCount
			NewVkPipelineShaderStageCreateInfoᶜᵖ(sb.MustAllocReadData(stages).Ptr()), // pStages
			NewVkPipelineVertexInputStateCreateInfoᶜᵖ(sb.MustAllocReadData( // pVertexInputState
				NewVkPipelineVertexInputStateCreateInfo(
					VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO, // sType
					0, // pNext
					0, // flags
					uint32(gp.VertexInputState().BindingDescriptions().Len()),                                                               // vertexBindingDescriptionCount
					NewVkVertexInputBindingDescriptionᶜᵖ(sb.MustUnpackReadMap(gp.VertexInputState().BindingDescriptions().All()).Ptr()),     // pVertexBindingDescriptions
					uint32(gp.VertexInputState().AttributeDescriptions().Len()),                                                             // vertexAttributeDescriptionCount
					NewVkVertexInputAttributeDescriptionᶜᵖ(sb.MustUnpackReadMap(gp.VertexInputState().AttributeDescriptions().All()).Ptr()), // pVertexAttributeDescriptions
				)).Ptr()),
			NewVkPipelineInputAssemblyStateCreateInfoᶜᵖ(sb.MustAllocReadData( // pInputAssemblyState
				NewVkPipelineInputAssemblyStateCreateInfo(
					VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO, // sType
					0,                                  // pNext
					0,                                  // flags
					gp.InputAssemblyState().Topology(), // topology
					gp.InputAssemblyState().PrimitiveRestartEnable(), // primitiveRestartEnable
				)).Ptr()),
			tessellationState, // pTessellationState
			viewportState,     // pViewportState
			NewVkPipelineRasterizationStateCreateInfoᶜᵖ(sb.MustAllocReadData( // pRasterizationState
				NewVkPipelineRasterizationStateCreateInfo(
					VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO, // sType
					0, // pNext
					0, // flags
					gp.RasterizationState().DepthClampEnable(),        // depthClampEnable
					gp.RasterizationState().RasterizerDiscardEnable(), // rasterizerDiscardEnable
					gp.RasterizationState().PolygonMode(),             // polygonMode
					gp.RasterizationState().CullMode(),                // cullMode
					gp.RasterizationState().FrontFace(),               // frontFace
					gp.RasterizationState().DepthBiasEnable(),         // depthBiasEnable
					gp.RasterizationState().DepthBiasConstantFactor(), // depthBiasConstantFactor
					gp.RasterizationState().DepthBiasClamp(),          // depthBiasClamp
					gp.RasterizationState().DepthBiasSlopeFactor(),    // depthBiasSlopeFactor
					gp.RasterizationState().LineWidth(),               // lineWidth
				)).Ptr()),
			multisampleState,               // pMultisampleState
			depthState,                     // pDepthStencilState
			colorBlendState,                // pColorBlendState
			dynamicState,                   // pDynamicState
			gp.Layout().VulkanHandle(),     // layout
			gp.RenderPass().VulkanHandle(), // renderPass
			gp.Subpass(),                   // subpass
			basePipeline,                   // basePipelineHandle
			-1,                             // basePipelineIndex
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(gp.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))

	for _, m := range temporaryShaderModules {
		sb.write(sb.cb.VkDestroyShaderModule(
			m.Device(),
			m.VulkanHandle(),
			memory.Nullptr,
		))
	}

	if !temporaryRenderPass.IsNil() {
		sb.write(sb.cb.VkDestroyRenderPass(
			temporaryRenderPass.Device(),
			temporaryRenderPass.VulkanHandle(),
			memory.Nullptr,
		))
	}

	if !temporaryPipelineLayout.IsNil() {
		sb.write(sb.cb.VkDestroyPipelineLayout(
			temporaryPipelineLayout.Device(),
			temporaryPipelineLayout.VulkanHandle(),
			memory.Nullptr,
		))
	}
}

func (sb *stateBuilder) createImageView(iv ImageViewObjectʳ) {
	if iv.Image().IsNil() ||
		!GetState(sb.newState).Images().Contains(iv.Image().VulkanHandle()) {
		// If the image that this image view points to has been deleted,
		// then don't even re-create the image view
		log.W(sb.ctx, "The image of image view %v is invalid, image view %v will not be created", iv.VulkanHandle(), iv.VulkanHandle())
		return
	}

	pNext := NewVoidᶜᵖ(memory.Nullptr)
	if !iv.UsageInfo().IsNil() {
		pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
			NewVkImageViewUsageCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_IMAGE_VIEW_USAGE_CREATE_INFO, // sType
				pNext,                  // pNext
				iv.UsageInfo().Usage(), // usage
			),
		).Ptr())
	}
	if !iv.YcbcrConversion().IsNil() {
		pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
			NewVkSamplerYcbcrConversionInfo(
				VkStructureType_VK_STRUCTURE_TYPE_SAMPLER_YCBCR_CONVERSION_INFO, // sType
				pNext,                               // pNext
				iv.YcbcrConversion().VulkanHandle(), // conversion
			),
		).Ptr())
	}

	sb.write(sb.cb.VkCreateImageView(
		iv.Device(),
		sb.MustAllocReadData(NewVkImageViewCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO, // sType
			NewVoidᶜᵖ(pNext),          // pNext
			0,                         // flags
			iv.Image().VulkanHandle(), // image
			iv.Type(),                 // viewType
			iv.Fmt(),                  // format
			iv.Components(),           // components
			iv.SubresourceRange(),     // subresourceRange
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(iv.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func (sb *stateBuilder) createBufferView(bv BufferViewObjectʳ) {
	if !GetState(sb.newState).Buffers().Contains(bv.Buffer().VulkanHandle()) {
		// If the image that this image view points to has been deleted,
		// then don't even re-create the image view
		log.W(sb.ctx, "The buffer of buffer view %v is invalid, buffer view %v will not be created", bv.VulkanHandle(), bv.VulkanHandle())
		return
	}

	sb.write(sb.cb.VkCreateBufferView(
		bv.Device(),
		sb.MustAllocReadData(NewVkBufferViewCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_BUFFER_VIEW_CREATE_INFO, // sType
			0,                          // pNext
			0,                          // flags
			bv.Buffer().VulkanHandle(), // buffer
			bv.Fmt(),                   // format
			bv.Offset(),                // offset
			bv.Range(),                 // range
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(bv.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func (sb *stateBuilder) createDescriptorPoolAndAllocateDescriptorSets(dp DescriptorPoolObjectʳ) {
	sb.write(sb.cb.VkCreateDescriptorPool(
		dp.Device(),
		sb.MustAllocReadData(NewVkDescriptorPoolCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO, // sType
			0,                        // pNext
			dp.Flags(),               // flags
			dp.MaxSets(),             // maxSets
			uint32(dp.Sizes().Len()), // poolSizeCount
			NewVkDescriptorPoolSizeᶜᵖ(sb.MustUnpackReadMap(dp.Sizes().All()).Ptr()), // pPoolSizes
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(dp.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))

	descSetHandles := make([]VkDescriptorSet, 0, dp.DescriptorSets().Len())
	descSetLayoutHandles := make([]VkDescriptorSetLayout, 0, dp.DescriptorSets().Len())
	for _, vkDescSet := range dp.DescriptorSets().Keys() {
		descSetObj := dp.DescriptorSets().Get(vkDescSet)
		if vkDescSet != VkDescriptorSet(0) && sb.s.DescriptorSets().Contains(vkDescSet) && sb.s.DescriptorSetLayouts().Contains(descSetObj.Layout().VulkanHandle()) {
			descSetHandles = append(descSetHandles, vkDescSet)
			descSetLayoutHandles = append(descSetLayoutHandles, descSetObj.Layout().VulkanHandle())
		}
	}

	sb.allocateDescriptorSets(dp, descSetHandles, descSetLayoutHandles)
}

func (sb *stateBuilder) allocateDescriptorSets(dp DescriptorPoolObjectʳ, descSetHandles []VkDescriptorSet, descSetLayoutHandles []VkDescriptorSetLayout) {
	if len(descSetHandles) != 0 && len(descSetLayoutHandles) != 0 {
		sb.write(sb.cb.VkAllocateDescriptorSets(
			dp.Device(),
			sb.MustAllocReadData(NewVkDescriptorSetAllocateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO, // sType
				0,                           // pNext
				dp.VulkanHandle(),           // descriptorPool
				uint32(len(descSetHandles)), // descriptorSetCount
				NewVkDescriptorSetLayoutᶜᵖ(sb.MustAllocReadData(descSetLayoutHandles).Ptr()), // pSetLayouts
			)).Ptr(),
			sb.MustAllocWriteData(descSetHandles).Ptr(),
			VkResult_VK_SUCCESS,
		))
	}
}

func (sb *stateBuilder) createFramebuffer(fb FramebufferObjectʳ) {
	var temporaryRenderPass RenderPassObjectʳ
	for _, k := range fb.ImageAttachments().Keys() {
		v := fb.ImageAttachments().Get(k)
		if !GetState(sb.newState).ImageViews().Contains(v.VulkanHandle()) {
			log.W(sb.ctx, "Image View %v is invalid, framebuffer %v will not be created", v.VulkanHandle(), fb.VulkanHandle())
			return
		}
	}

	if !GetState(sb.newState).RenderPasses().Contains(fb.RenderPass().VulkanHandle()) {
		sb.createRenderPass(fb.RenderPass())
		temporaryRenderPass = GetState(sb.newState).RenderPasses().Get(fb.RenderPass().VulkanHandle())
	}

	imageViews := []VkImageView{}
	for _, v := range fb.ImageAttachments().Keys() {
		imageViews = append(imageViews, fb.ImageAttachments().Get(v).VulkanHandle())
	}

	sb.write(sb.cb.VkCreateFramebuffer(
		fb.Device(),
		sb.MustAllocReadData(NewVkFramebufferCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO, // sType
			0,                              // pNext
			0,                              // flags
			fb.RenderPass().VulkanHandle(), // renderPass
			uint32(len(imageViews)),        // attachmentCount
			NewVkImageViewᶜᵖ(sb.MustAllocReadData(imageViews).Ptr()), // pAttachments
			fb.Width(),  // width
			fb.Height(), // height
			fb.Layers(), // layers
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(fb.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))

	if !temporaryRenderPass.IsNil() {
		sb.write(sb.cb.VkDestroyRenderPass(
			temporaryRenderPass.Device(),
			temporaryRenderPass.VulkanHandle(),
			memory.Nullptr,
		))
	}
}

func (sb *stateBuilder) writeDescriptorSet(ds DescriptorSetObjectʳ) {
	ns := GetState(sb.newState)
	if !ns.DescriptorPools().Contains(ds.DescriptorPool()) {
		return
	}
	if !ns.DescriptorSets().Contains(ds.VulkanHandle()) {
		return
	}

	writes := []VkWriteDescriptorSet{}
	for _, k := range ds.Bindings().Keys() {
		binding := ds.Bindings().Get(k)
		switch binding.BindingType() {
		case VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLER:
			numImages := uint32(binding.ImageBinding().Len())
			for i := uint32(0); i < numImages; i++ {
				im := binding.ImageBinding().Get(i)
				if im.Sampler() == 0 {
					continue
				}

				if im.Sampler() != 0 && !ns.Samplers().Contains(im.Sampler()) {
					continue
				}

				writes = append(writes, NewVkWriteDescriptorSet(
					VkStructureType_VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, // sType
					0,                     // pNext
					ds.VulkanHandle(),     // dstSet
					k,                     // dstBinding
					i,                     // dstArrayElement
					1,                     // descriptorCount
					binding.BindingType(), // descriptorType
					NewVkDescriptorImageInfoᶜᵖ(sb.MustAllocReadData(im.Get()).Ptr()), // pImageInfo
					0, // pBufferInfo
					0, // pTexelBufferView
				))
			}
		case VkDescriptorType_VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
			VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE,
			VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
			VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT:

			numImages := uint32(binding.ImageBinding().Len())
			for i := uint32(0); i < numImages; i++ {
				im := binding.ImageBinding().Get(i)
				if im.Sampler() == 0 && im.ImageView() == 0 {
					continue
				}
				// If this is a combined image sampler but we have an immutable sampler,
				// we should still be setting the image view
				if binding.BindingType() == VkDescriptorType_VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER {
					if ds.Layout().Bindings().Get(k).ImmutableSamplers().Get(i).IsNil() && im.Sampler() == 0 {
						continue
					}
					if im.ImageView() == 0 {
						continue
					}
				}

				if im.Sampler() != 0 && !ns.Samplers().Contains(im.Sampler()) {
					continue
				}
				if im.ImageView() != 0 && !ns.ImageViews().Contains(im.ImageView()) {
					continue
				}

				writes = append(writes, NewVkWriteDescriptorSet(
					VkStructureType_VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, // sType
					0,                     // pNext
					ds.VulkanHandle(),     // dstSet
					k,                     // dstBinding
					i,                     // dstArrayElement
					1,                     // descriptorCount
					binding.BindingType(), // descriptorType
					NewVkDescriptorImageInfoᶜᵖ(sb.MustAllocReadData(im.Get()).Ptr()), // pImageInfo
					0, // pBufferInfo
					0, // pTexelBufferView
				))
			}

		case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
			VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER,
			VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC,
			VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC:
			numBuffers := uint32(binding.BufferBinding().Len())
			for i := uint32(0); i < numBuffers; i++ {
				buff := binding.BufferBinding().Get(i)
				if buff.Buffer() == 0 {
					continue
				}
				if buff.Buffer() != 0 && !ns.Buffers().Contains(buff.Buffer()) {
					continue
				}
				writes = append(writes, NewVkWriteDescriptorSet(
					VkStructureType_VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, // sType
					0,                     // pNext
					ds.VulkanHandle(),     // dstSet
					k,                     // dstBinding
					i,                     // dstArrayElement
					1,                     // descriptorCount
					binding.BindingType(), // descriptorType
					0,                     // pImageInfo
					NewVkDescriptorBufferInfoᶜᵖ(sb.MustAllocReadData(buff.Get()).Ptr()), // pBufferInfo
					0, // pTexelBufferView
				))
			}

		case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER,
			VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER:
			numBuffers := uint32(binding.BufferViewBindings().Len())
			for i := uint32(0); i < numBuffers; i++ {
				bv := binding.BufferViewBindings().Get(i)
				if bv == 0 {
					continue
				}
				if bv != 0 && !ns.BufferViews().Contains(bv) {
					continue
				}
				writes = append(writes, NewVkWriteDescriptorSet(
					VkStructureType_VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, // sType
					0,                     // pNext
					ds.VulkanHandle(),     // dstSet
					k,                     // dstBinding
					i,                     // dstArrayElement
					1,                     // descriptorCount
					binding.BindingType(), // descriptorType
					0,                     // pImageInfo
					0,                     // pBufferInfo
					NewVkBufferViewᶜᵖ(sb.MustAllocReadData(bv).Ptr()), // pTexelBufferView
				))
			}
		}
	}
	if len(writes) > 0 {
		sb.write(sb.cb.VkUpdateDescriptorSets(
			ds.Device(),
			uint32(len(writes)),
			sb.MustAllocReadData(writes).Ptr(),
			0,
			memory.Nullptr,
		))
	}
}

func (sb *stateBuilder) createQueryPool(qp QueryPoolObjectʳ) {
	sb.write(sb.cb.VkCreateQueryPool(
		qp.Device(),
		sb.MustAllocReadData(NewVkQueryPoolCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_QUERY_POOL_CREATE_INFO, // sType
			0,                       // pNext
			0,                       // flags
			qp.QueryType(),          // queryType
			qp.QueryCount(),         // queryCount
			qp.PipelineStatistics(), // pipelineStatistics
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(qp.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))

	anyInitialized := false
	for _, k := range qp.Status().Keys() {
		k := qp.Status().Get(k)
		if k != QueryStatus_QUERY_STATUS_UNINITIALIZED {
			anyInitialized = true
			break
		}
	}
	if !anyInitialized {
		return
	}
	queue := sb.getQueueFor(
		VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT|VkQueueFlagBits_VK_QUEUE_COMPUTE_BIT,
		[]uint32{}, qp.Device(), qp.LastBoundQueue())
	if queue.IsNil() {
		return
	}

	queueHandler := sb.scratchRes.GetQueueCommandHandler(sb, queue.VulkanHandle())
	queueHandler.RecordCommands(sb, debugMarkerName(
		"restoring query states",
	), func(commandBuffer VkCommandBuffer) {
		for i := uint32(0); i < qp.QueryCount(); i++ {
			switch qp.Status().Get(i) {
			case QueryStatus_QUERY_STATUS_UNINITIALIZED:
				// do nothing
			case QueryStatus_QUERY_STATUS_INACTIVE:
				sb.write(sb.cb.VkCmdResetQueryPool(commandBuffer, qp.VulkanHandle(), i, 1))
			case QueryStatus_QUERY_STATUS_ACTIVE:
				sb.write(sb.cb.VkCmdResetQueryPool(commandBuffer, qp.VulkanHandle(), i, 1))
				sb.write(sb.cb.VkCmdBeginQuery(commandBuffer, qp.VulkanHandle(), i, VkQueryControlFlags(0)))
			case QueryStatus_QUERY_STATUS_COMPLETE:
				sb.write(sb.cb.VkCmdResetQueryPool(commandBuffer, qp.VulkanHandle(), i, 1))
				if qp.QueryType() == VkQueryType_VK_QUERY_TYPE_TIMESTAMP {
					sb.write(sb.cb.VkCmdWriteTimestamp(commandBuffer, VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, qp.VulkanHandle(), i))
				} else {
					sb.write(sb.cb.VkCmdBeginQuery(commandBuffer, qp.VulkanHandle(), i, VkQueryControlFlags(0)))
					sb.write(sb.cb.VkCmdEndQuery(commandBuffer, qp.VulkanHandle(), i))
				}
			}
		}
	})
}

func (sb *stateBuilder) createCommandBuffer(cb CommandBufferObjectʳ, level VkCommandBufferLevel) {
	if cb.Level() != level {
		return
	}

	sb.write(sb.cb.VkAllocateCommandBuffers(
		cb.Device(),
		sb.MustAllocReadData(NewVkCommandBufferAllocateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
			0,          // pNext
			cb.Pool(),  // commandPool
			cb.Level(), // level
			1,          // commandBufferCount
		)).Ptr(),
		sb.MustAllocWriteData(cb.VulkanHandle()).Ptr(),
		VkResult_VK_SUCCESS,
	))

}

func (sb *stateBuilder) recordCommandBuffer(cb CommandBufferObjectʳ, level VkCommandBufferLevel, srcState *api.GlobalState) {
	if cb.Level() != level {
		return
	}

	if cb.Recording() == RecordingState_NOT_STARTED || cb.Recording() == RecordingState_TO_BE_RESET {
		return
	}

	pNext := NewVoidᶜᵖ(memory.Nullptr)
	if !cb.BeginInfo().DeviceGroupBegin().IsNil() {
		pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
			NewVkDeviceGroupCommandBufferBeginInfo(
				VkStructureType_VK_STRUCTURE_TYPE_DEVICE_GROUP_COMMAND_BUFFER_BEGIN_INFO, // sType
				pNext, // pNext
				cb.BeginInfo().DeviceGroupBegin().DeviceMask(), // deviceMask
			),
		).Ptr())
	}

	beginInfo := NewVkCommandBufferBeginInfo(
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
		pNext, // pNext
		VkCommandBufferUsageFlags(cb.BeginInfo().Flags()), // flags
		0, // pInheritanceInfo
	)
	if cb.BeginInfo().Inherited() {
		if cb.BeginInfo().InheritedFramebuffer() != VkFramebuffer(0) &&
			!GetState(sb.newState).Framebuffers().Contains(cb.BeginInfo().InheritedFramebuffer()) {
			log.W(sb.ctx, "Command Buffer %v is invalid, it will not be recorded: - Inherited framebuffer does not exist", cb.VulkanHandle())
			return
		}

		inheritanceInfo := sb.MustAllocReadData(NewVkCommandBufferInheritanceInfo(
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_INHERITANCE_INFO, // sType
			0,                                            // pNext
			cb.BeginInfo().InheritedRenderPass(),         // renderPass
			cb.BeginInfo().InheritedSubpass(),            // subpass
			cb.BeginInfo().InheritedFramebuffer(),        // framebuffer
			cb.BeginInfo().InheritedOcclusionQuery(),     // occlusionQueryEnable
			cb.BeginInfo().InheritedQueryFlags(),         // queryFlags
			cb.BeginInfo().InheritedPipelineStatsFlags(), // pipelineStatistics
		))
		beginInfo.SetPInheritanceInfo(NewVkCommandBufferInheritanceInfoᶜᵖ(inheritanceInfo.Ptr()))
	}

	sb.write(sb.cb.VkBeginCommandBuffer(
		cb.VulkanHandle(),
		sb.MustAllocReadData(beginInfo).Ptr(),
		VkResult_VK_SUCCESS,
	))

	hasError := false
	// fill command buffer
	for i, c := uint32(0), uint32(cb.CommandReferences().Len()); i < c; i++ {
		arg := GetCommandArgs(sb.ctx, cb.CommandReferences().Get(i), GetState(srcState))
		cleanup, cmd, err := AddCommand(sb.ctx, sb.cb, cb.VulkanHandle(), srcState, sb.newState, arg)
		if err != nil {
			log.W(sb.ctx, "Command Buffer %v is invalid, it will not be recorded: - %v", cb.VulkanHandle(), err)
			hasError = true
			break
		}
		sb.write(cmd)
		cleanup()
	}
	if hasError {
		return
	}
	if cb.Recording() == RecordingState_COMPLETED {
		sb.write(sb.cb.VkEndCommandBuffer(
			cb.VulkanHandle(),
			VkResult_VK_SUCCESS,
		))
	}
}

func queueFamilyIndicesToU32Slice(m U32ːu32ᵐ) []uint32 {
	r := make([]uint32, 0, m.Len())
	for _, k := range m.Keys() {
		r = append(r, m.Get(k))
	}
	return r
}

// createSameBuffer creates a new buffer with ID |buffer| using the same information from |src| object
func (sb *stateBuilder) createSameBuffer(src BufferObjectʳ, buffer VkBuffer, mem DeviceMemoryObjectʳ, memoryOffset VkDeviceSize) error {
	os := sb.s
	pNext := NewVoidᶜᵖ(memory.Nullptr)

	if !src.Info().DedicatedAllocationNV().IsNil() {
		pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
			NewVkDedicatedAllocationBufferCreateInfoNV(
				VkStructureType_VK_STRUCTURE_TYPE_DEDICATED_ALLOCATION_BUFFER_CREATE_INFO_NV, // sType
				0, // pNext
				src.Info().DedicatedAllocationNV().DedicatedAllocation(), // dedicatedAllocation
			),
		).Ptr())
	}

	memReq := NewVkMemoryRequirements(
		src.MemoryRequirements().Size(), src.MemoryRequirements().Alignment(), src.MemoryRequirements().MemoryTypeBits())

	createWithMemReq := sb.cb.VkCreateBuffer(
		src.Device(),
		sb.MustAllocReadData(
			NewVkBufferCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO, // sType
				pNext,                    // pNext
				src.Info().CreateFlags(), // flags
				src.Info().Size(),        // size
				VkBufferUsageFlags(uint32(src.Info().Usage())|uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT)), // usage
				src.Info().SharingMode(),                                                    // sharingMode
				uint32(src.Info().QueueFamilyIndices().Len()),                               // queueFamilyIndexCount
				NewU32ᶜᵖ(sb.MustUnpackReadMap(src.Info().QueueFamilyIndices().All()).Ptr()), // pQueueFamilyIndices
			)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(buffer).Ptr(),
		VkResult_VK_SUCCESS,
	)
	createWithMemReq.Extras().Add(memReq)
	sb.write(createWithMemReq)

	dst := os.Buffers().Get(buffer)
	sb.write(sb.cb.VkGetBufferMemoryRequirements(
		dst.Device(),
		dst.VulkanHandle(),
		sb.MustAllocWriteData(memReq).Ptr(),
	))

	// Dedicated allocation buffer/image must NOT be a sparse binding one.
	// Checking the dedicated allocation info on both the memory and the buffer
	// side, because we've found applications that do miss one of them.
	dedicatedMemoryNV := !src.Memory().IsNil() && (!src.Info().DedicatedAllocationNV().IsNil() || !src.Memory().DedicatedAllocationNV().IsNil())
	// Emit error message to report view if we found one of the dedicate allocation
	// info struct is missing.
	if dedicatedMemoryNV && src.Info().DedicatedAllocationNV().IsNil() {
		subVkErrorExpectNVDedicatedlyAllocatedHandle(sb.ctx, nil, api.CmdNoID, nil,
			sb.oldState, GetState(sb.oldState), 0, nil, nil, "VkBuffer", uint64(src.VulkanHandle()))
	}
	if dedicatedMemoryNV && src.Memory().DedicatedAllocationNV().IsNil() {
		subVkErrorExpectNVDedicatedlyAllocatedHandle(sb.ctx, nil, api.CmdNoID, nil,
			sb.oldState, GetState(sb.oldState), 0, nil, nil, "VkDeviceMemory", uint64(src.Memory().VulkanHandle()))
	}

	if dedicatedMemoryNV {
		sb.createDeviceMemory(mem, true)
	}

	denseBound := !src.Memory().IsNil()
	sparseBound := src.SparseMemoryBindings().Len() > 0
	if !denseBound && !sparseBound {
		return fmt.Errorf("source buffer %s is neither dense bound nor sparse bound", src)
	}

	if src.SparseMemoryBindings().Len() > 0 {
		// If this buffer has sparse memory bindings, then we have to set them all
		// now
		if src.IsNil() {
			return fmt.Errorf("queue is not found for buffer %s", src)
		}
		memories := make(map[VkDeviceMemory]bool)
		sparseQueue := sb.getQueueFor(
			VkQueueFlagBits_VK_QUEUE_SPARSE_BINDING_BIT,
			queueFamilyIndicesToU32Slice(src.Info().QueueFamilyIndices()),
			src.Device(), src.LastBoundQueue())
		if !src.Info().DedicatedAllocationNV().IsNil() {
			for _, k := range src.SparseMemoryBindings().Keys() {
				bind := src.SparseMemoryBindings().Get(k)
				if _, ok := memories[bind.Memory()]; !ok {
					memories[bind.Memory()] = true
					sb.createDeviceMemory(os.DeviceMemories().Get(bind.Memory()), true)
				}
			}
		}

		bufSparseBindings := make([]VkSparseMemoryBind, 0, src.SparseMemoryBindings().Len())
		for _, k := range src.SparseMemoryBindings().Keys() {
			bd := src.SparseMemoryBindings().Get(k)
			bufSparseBindings = append(bufSparseBindings, bd)
		}

		sb.write(sb.cb.VkQueueBindSparse(
			sparseQueue.VulkanHandle(),
			1,
			sb.MustAllocReadData(
				NewVkBindSparseInfo(
					VkStructureType_VK_STRUCTURE_TYPE_BIND_SPARSE_INFO, // sType
					0, // pNext
					0, // waitSemaphoreCount
					0, // pWaitSemaphores
					1, // bufferBindCount
					NewVkSparseBufferMemoryBindInfoᶜᵖ(sb.MustAllocReadData( // pBufferBinds
						NewVkSparseBufferMemoryBindInfo(
							src.VulkanHandle(),                       // buffer
							uint32(src.SparseMemoryBindings().Len()), // bindCount
							NewVkSparseMemoryBindᶜᵖ( // pBinds
								sb.MustAllocReadData(bufSparseBindings).Ptr(),
							),
						)).Ptr()),
					0, // imageOpaqueBindCount
					0, // pImageOpaqueBinds
					0, // imageBindCount
					0, // pImageBinds
					0, // signalSemaphoreCount
					0, // pSignalSemaphores
				)).Ptr(),
			VkFence(0),
			VkResult_VK_SUCCESS,
		))

	} else {
		// Otherwise, we have no sparse bindings, we are either non-sparse, or empty.
		if mem.IsNil() {
			return fmt.Errorf("buffer memory is nil for buffer %v", src)
		}

		if !src.DeviceGroupBinding().IsNil() {
			dg := src.DeviceGroupBinding()
			sb.write(sb.cb.VkBindBufferMemory2(
				dst.Device(),
				1,
				sb.MustAllocReadData(
					NewVkBindBufferMemoryInfo(
						VkStructureType_VK_STRUCTURE_TYPE_BIND_BUFFER_MEMORY_INFO, // sType
						NewVoidᶜᵖ(sb.MustAllocReadData(
							NewVkBindBufferMemoryDeviceGroupInfo(
								VkStructureType_VK_STRUCTURE_TYPE_BIND_BUFFER_MEMORY_DEVICE_GROUP_INFO, // sType
								0, // pNext
								uint32(dg.Bindings().Len()),
								NewU32ᶜᵖ(sb.MustUnpackReadMap(dg.Bindings().All()).Ptr()),
							),
						).Ptr()),
						dst.VulkanHandle(),
						mem.VulkanHandle(),
						memoryOffset,
					)).Ptr(),
				VkResult_VK_SUCCESS,
			))
		} else {
			sb.write(sb.cb.VkBindBufferMemory(
				dst.Device(),
				dst.VulkanHandle(),
				mem.VulkanHandle(),
				memoryOffset,
				VkResult_VK_SUCCESS,
			))
		}
	}

	return nil
}

// loadHostDatatoStagingBuffer creates a staging buffer and loads the data from the trace to the buffer.
func (sb *stateBuilder) loadHostDatatoStagingBuffer(buffer BufferObjectʳ, tsk *queueCommandBatch) (VkBuffer, error) {
	offset := VkDeviceSize(0)
	contents := []hashedDataAndOffset{}
	sparseBinding :=
		(uint64(buffer.Info().CreateFlags()) &
			uint64(VkBufferCreateFlagBits_VK_BUFFER_CREATE_SPARSE_BINDING_BIT)) != 0
	sparseResidency :=
		sparseBinding &&
			(uint64(buffer.Info().CreateFlags())&
				uint64(VkBufferCreateFlagBits_VK_BUFFER_CREATE_SPARSE_RESIDENCY_BIT)) != 0

	if buffer.SparseMemoryBindings().Len() > 0 {
		if sparseResidency || IsFullyBound(0, buffer.Info().Size(), buffer.SparseMemoryBindings()) {
			for _, k := range buffer.SparseMemoryBindings().Keys() {
				bind := buffer.SparseMemoryBindings().Get(k)
				size := bind.Size()
				dataSlice := sb.s.DeviceMemories().Get(bind.Memory()).Data().Slice(
					uint64(bind.MemoryOffset()),
					uint64(bind.MemoryOffset()+size))
				hd := newHashedDataFromSlice(sb.ctx, sb.oldState, dataSlice)
				contents = append(contents, newHashedDataAndOffset(hd, uint64(offset)))
				offset += size
				offset = (offset + VkDeviceSize(7)) & (^VkDeviceSize(7))
			}
		}
	} else {
		size := buffer.Info().Size()
		dataSlice := buffer.Memory().Data().Slice(
			uint64(buffer.MemoryOffset()),
			uint64(buffer.MemoryOffset()+size))
		hd := newHashedDataFromSlice(sb.ctx, sb.oldState, dataSlice)
		contents = append(contents, newHashedDataAndOffset(hd, uint64(offset)))
	}

	// scratch buffer will be automatically destroyed when the task is done
	scratchBuffer := tsk.NewScratchBuffer(sb, "buf->buf copy staginig buffer",
		sb.scratchRes.GetMemory(sb, buffer.Device()),
		buffer.Device(), VkBufferUsageFlags(
			VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT), contents...)
	return scratchBuffer, nil

}

// copyBuffer copies data from the src buffer to the dst buffer.
func (sb *stateBuilder) copyBuffer(src, dst VkBuffer, queue QueueObjectʳ, tsk *queueCommandBatch) {
	os := sb.s
	copies := []VkBufferCopy{}
	offset := VkDeviceSize(0)
	dstObj := os.Buffers().Get(dst)
	if dstObj == NilBufferObjectʳ {
		log.I(sb.ctx, "copyBuffer failed: dst buffer is nil")
		return
	}

	sparseBinding :=
		(uint64(dstObj.Info().CreateFlags()) &
			uint64(VkBufferCreateFlagBits_VK_BUFFER_CREATE_SPARSE_BINDING_BIT)) != 0
	sparseResidency :=
		sparseBinding &&
			(uint64(dstObj.Info().CreateFlags())&
				uint64(VkBufferCreateFlagBits_VK_BUFFER_CREATE_SPARSE_RESIDENCY_BIT)) != 0

	oldFamilyIndex := queueFamilyIgnore

	if dstObj.SparseMemoryBindings().Len() > 0 {
		oldFamilyIndex = dstObj.LastBoundQueue().Family()
		if sparseResidency || IsFullyBound(0, dstObj.Info().Size(), dstObj.SparseMemoryBindings()) {
			for _, k := range dstObj.SparseMemoryBindings().Keys() {
				bind := dstObj.SparseMemoryBindings().Get(k)
				size := bind.Size()
				copies = append(copies, NewVkBufferCopy(
					offset,                // srcOffset
					bind.ResourceOffset(), // dstOffset
					size,                  // size
				))
				offset += size
				offset = (offset + VkDeviceSize(7)) & (^VkDeviceSize(7))
			}
		}
	} else {
		size := dstObj.Info().Size()
		copies = append(copies, NewVkBufferCopy(
			offset, // srcOffset
			0,      // dstOffset
			size,   // size
		))
	}

	newFamilyIndex := queueFamilyIgnore
	if oldFamilyIndex != queueFamilyIgnore {
		newFamilyIndex = queue.Family()
	}

	tsk.RecordCommandsOnCommit(func(commandBuffer VkCommandBuffer) {
		sb.write(sb.cb.VkCmdPipelineBarrier(
			commandBuffer,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkDependencyFlags(0),
			0,
			memory.Nullptr,
			1,
			sb.MustAllocReadData(
				NewVkBufferMemoryBarrier(
					VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, // sType
					0, // pNext
					VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // srcAccessMask
					VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // dstAccessMask
					queueFamilyIgnore,                // srcQueueFamilyIndex
					queueFamilyIgnore,                // dstQueueFamilyIndex
					src,                              // buffer
					0,                                // offset
					VkDeviceSize(0xFFFFFFFFFFFFFFFF), // size
				)).Ptr(),
			0,
			memory.Nullptr,
		))
		sb.write(sb.cb.VkCmdCopyBuffer(
			commandBuffer,
			src,
			dst,
			uint32(len(copies)),
			sb.MustAllocReadData(copies).Ptr(),
		))

		sb.write(sb.cb.VkCmdPipelineBarrier(
			commandBuffer,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkDependencyFlags(0),
			0,
			memory.Nullptr,
			1,
			sb.MustAllocReadData(
				NewVkBufferMemoryBarrier(
					VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, // sType
					0, // pNext
					VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // srcAccessMask
					VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // dstAccessMask
					oldFamilyIndex,                   // srcQueueFamilyIndex
					newFamilyIndex,                   // dstQueueFamilyIndex
					dst,                              // buffer
					0,                                // offset
					VkDeviceSize(0xFFFFFFFFFFFFFFFF), // size
				)).Ptr(),
			0,
			memory.Nullptr,
		))
	})
}
