// Copyright (C) 2018 Google Inc.
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

// #include "gapis/api/vulkan/ctypes.h"
import "C"

import (
	"unsafe"

	"github.com/google/gapid/gapil/executor"
	"github.com/google/gapid/gapis/memory"
)

func init() {
	executor.RegisterGoExtern("vulkan.enterSubcontext", externEnterSubcontext)
	executor.RegisterGoExtern("vulkan.fetchBufferMemoryRequirements", externFetchBufferMemoryRequirements)
	executor.RegisterGoExtern("vulkan.fetchImageMemoryRequirements", externFetchImageMemoryRequirements)
	executor.RegisterGoExtern("vulkan.fetchLinearImageSubresourceLayouts", externFetchLinearImageSubresourceLayouts)
	executor.RegisterGoExtern("vulkan.fetchPhysicalDeviceMemoryProperties", externFetchPhysicalDeviceMemoryProperties)
	executor.RegisterGoExtern("vulkan.fetchPhysicalDeviceProperties", externFetchPhysicalDeviceProperties)
	executor.RegisterGoExtern("vulkan.fetchPhysicalDeviceQueueFamilyProperties", externFetchPhysicalDeviceQueueFamilyProperties)
	executor.RegisterGoExtern("vulkan.fetchPhysicalDeviceFormatProperties", externFetchPhysicalDeviceFormatProperties)
	executor.RegisterGoExtern("vulkan.hasDynamicProperty", externHasDynamicProperty)
	executor.RegisterGoExtern("vulkan.leaveSubcontext", externLeaveSubcontext)
	executor.RegisterGoExtern("vulkan.mapMemory", externMapMemory)
	executor.RegisterGoExtern("vulkan.nextSubcontext", externNextSubcontext)
	executor.RegisterGoExtern("vulkan.notifyPendingCommandAdded", externNotifyPendingCommandAdded)
	executor.RegisterGoExtern("vulkan.numberOfPNext", externNumberOfPNext)
	executor.RegisterGoExtern("vulkan.onCommandAdded", externOnCommandAdded)
	executor.RegisterGoExtern("vulkan.onDeferSubcommand", externOnDeferSubcommand)
	executor.RegisterGoExtern("vulkan.onPostSubcommand", externOnPostSubcommand)
	executor.RegisterGoExtern("vulkan.onPreProcessCommand", externOnPreProcessCommand)
	executor.RegisterGoExtern("vulkan.onPreSubcommand", externOnPreSubcommand)
	executor.RegisterGoExtern("vulkan.popAndPushMarkerForNextSubpass", externPopAndPushMarkerForNextSubpass)
	executor.RegisterGoExtern("vulkan.popDebugMarker", externPopDebugMarker)
	executor.RegisterGoExtern("vulkan.popRenderPassMarker", externPopRenderPassMarker)
	executor.RegisterGoExtern("vulkan.postBindSparse", externPostBindSparse)
	executor.RegisterGoExtern("vulkan.pushDebugMarker", externPushDebugMarker)
	executor.RegisterGoExtern("vulkan.pushRenderPassMarker", externPushRenderPassMarker)
	executor.RegisterGoExtern("vulkan.readMappedCoherentMemory", externReadMappedCoherentMemory)
	executor.RegisterGoExtern("vulkan.resetCmd", externResetCmd)
	executor.RegisterGoExtern("vulkan.resetSubcontext", externResetSubcontext)
	executor.RegisterGoExtern("vulkan.trackMappedCoherentMemory", externTrackMappedCoherentMemory)
	executor.RegisterGoExtern("vulkan.unmapMemory", externUnmapMemory)
	executor.RegisterGoExtern("vulkan.untrackMappedCoherentMemory", externUntrackMappedCoherentMemory)
	executor.RegisterGoExtern("vulkan.vkErrCommandBufferIncomplete", externVkErrCommandBufferIncomplete)
	executor.RegisterGoExtern("vulkan.vkErrExpectNVDedicatedlyAllocatedHandle", externVkErrExpectNVDedicatedlyAllocatedHandle)
	executor.RegisterGoExtern("vulkan.vkErrInvalidDescriptorArrayElement", externVkErrInvalidDescriptorArrayElement)
	executor.RegisterGoExtern("vulkan.vkErrInvalidHandle", externVkErrInvalidHandle)
	executor.RegisterGoExtern("vulkan.vkErrInvalidImageLayout", externVkErrInvalidImageLayout)
	executor.RegisterGoExtern("vulkan.vkErrInvalidImageSubresource", externVkErrInvalidImageSubresource)
	executor.RegisterGoExtern("vulkan.vkErrNotNullPointer", externVkErrNotNullPointer)
	executor.RegisterGoExtern("vulkan.vkErrNullPointer", externVkErrNullPointer)
	executor.RegisterGoExtern("vulkan.vkErrUnrecognizedExtension", externVkErrUnrecognizedExtension)
	executor.RegisterGoExtern("vulkan.recordFenceSignal", externRecordFenceSignal)
	executor.RegisterGoExtern("vulkan.recordFenceWait", externRecordFenceWait)
	executor.RegisterGoExtern("vulkan.recordFenceReset", externRecordFenceReset)
	executor.RegisterGoExtern("vulkan.recordAcquireNextImage", externRecordAcquireNextImage)
	executor.RegisterGoExtern("vulkan.recordPresentSwapchainImage", externRecordPresentSwapchainImage)
}

func externsFromEnv(env *executor.Env) *externs {
	return &externs{
		ctx:   env.Context(),
		cmd:   env.Cmd(),
		cmdID: env.CmdID(),
		s:     env.State,
		b:     nil,
	}
}

// enterSubcontext
func externEnterSubcontext(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	e.enterSubcontext()
}

// fetchBufferMemoryRequirements
func externFetchBufferMemoryRequirements(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.fetchBufferMemoryRequirements_args)(args)
	o := (*C.fetchBufferMemoryRequirements_res)(out)

	*o = *e.fetchBufferMemoryRequirements(
		VkDevice(a.device),
		VkBuffer(a.buffer),
	).c
}

// fetchImageMemoryRequirements
func externFetchImageMemoryRequirements(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.fetchImageMemoryRequirements_args)(args)
	o := (*C.fetchImageMemoryRequirements_res)(out)

	*o = e.fetchImageMemoryRequirements(
		VkDevice(a.device),
		VkImage(a.image),
		bool(a.hasSparseBit),
	).c
}

// fetchLinearImageSubresourceLayouts
func externFetchLinearImageSubresourceLayouts(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.fetchLinearImageSubresourceLayouts_args)(args)
	o := (*C.fetchLinearImageSubresourceLayouts_res)(out)

	*o = e.fetchLinearImageSubresourceLayouts(
		VkDevice(a.device),
		ImageObjectʳ{a.image},
		VkImageSubresourceRange{c: &a.rng, a: e.s.Arena},
	).c
}

// fetchPhysicalDeviceMemoryProperties
func externFetchPhysicalDeviceMemoryProperties(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.fetchPhysicalDeviceMemoryProperties_args)(args)
	o := (*C.fetchPhysicalDeviceMemoryProperties_res)(out)

	*o = e.fetchPhysicalDeviceMemoryProperties(
		VkInstance(a.instance),
		VkPhysicalDeviceˢ{&a.devs},
	).c
}

// fetchPhysicalDeviceProperties
func externFetchPhysicalDeviceProperties(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.fetchPhysicalDeviceProperties_args)(args)
	o := (*C.fetchPhysicalDeviceProperties_res)(out)

	*o = e.fetchPhysicalDeviceProperties(
		VkInstance(a.instance),
		VkPhysicalDeviceˢ{&a.devs},
	).c
}

// fetchPhysicalDeviceQueueFamilyProperties
func externFetchPhysicalDeviceQueueFamilyProperties(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.fetchPhysicalDeviceQueueFamilyProperties_args)(args)
	o := (*C.fetchPhysicalDeviceQueueFamilyProperties_res)(out)

	*o = e.fetchPhysicalDeviceQueueFamilyProperties(
		VkInstance(a.instance),
		VkPhysicalDeviceˢ{&a.devs},
	).c
}

// fetchPhysicalDeviceFormatProperties
func externFetchPhysicalDeviceFormatProperties(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.fetchPhysicalDeviceFormatProperties_args)(args)
	o := (*C.fetchPhysicalDeviceFormatProperties_res)(out)

	*o = e.fetchPhysicalDeviceFormatProperties(
		VkInstance(a.instance),
		VkPhysicalDeviceˢ{&a.devs},
	).c
}

// hasDynamicProperty
func externHasDynamicProperty(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.hasDynamicProperty_args)(args)
	o := (*C.hasDynamicProperty_res)(out)

	*o = C.bool(e.hasDynamicProperty(
		VkPipelineDynamicStateCreateInfoᶜᵖ(a.info),
		VkDynamicState(a.state),
	))
}

// leaveSubcontext
func externLeaveSubcontext(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	e.leaveSubcontext()
}

// mapMemory
func externMapMemory(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.mapMemory_args)(args)

	e.mapMemory(
		Voidᵖᵖ(a.mem),
		U8ˢ{&a.slice},
	)
}

// nextSubcontext
func externNextSubcontext(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	e.nextSubcontext()
}

// notifyPendingCommandAdded
func externNotifyPendingCommandAdded(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.notifyPendingCommandAdded_args)(args)

	e.notifyPendingCommandAdded(
		VkQueue(a.queue),
	)
}

// numberOfPNext
func externNumberOfPNext(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.numberOfPNext_args)(args)
	o := (*C.numberOfPNext_res)(out)

	*o = C.uint32_t(e.numberOfPNext(
		Voidᶜᵖ(a.pNext),
	))
}

// onCommandAdded
func externOnCommandAdded(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.onCommandAdded_args)(args)

	e.onCommandAdded(
		VkCommandBuffer(a.buffer),
	)
}

// onDeferSubcommand
func externOnDeferSubcommand(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.onDeferSubcommand_args)(args)

	e.onDeferSubcommand(
		CommandReferenceʳ{a.ref},
	)
}

// onPostSubcommand
func externOnPostSubcommand(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.onPostSubcommand_args)(args)

	e.onPostSubcommand(
		CommandReferenceʳ{a.ref},
	)
}

// onPreProcessCommand
func externOnPreProcessCommand(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.onPreProcessCommand_args)(args)

	e.onPreProcessCommand(
		CommandReferenceʳ{a.ref},
	)
}

// onPreSubcommand
func externOnPreSubcommand(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.onPreSubcommand_args)(args)

	e.onPreSubcommand(
		CommandReferenceʳ{a.ref},
	)
}

// popAndPushMarkerForNextSubpass
func externPopAndPushMarkerForNextSubpass(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.popAndPushMarkerForNextSubpass_args)(args)

	e.popAndPushMarkerForNextSubpass(
		uint32(a.nextSubpass),
	)
}

// popDebugMarker
func externPopDebugMarker(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	e.popDebugMarker()
}

// popRenderPassMarker
func externPopRenderPassMarker(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	e.popRenderPassMarker()
}

// postBindSparse
func externPostBindSparse(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.postBindSparse_args)(args)

	e.postBindSparse(
		QueuedSparseBindsʳ{a.binds},
	)
}

// pushDebugMarker
func externPushDebugMarker(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.pushDebugMarker_args)(args)

	e.pushDebugMarker(
		C.GoStringN((*C.char)(unsafe.Pointer(&a.name.data[0])), (C.int)(a.name.length)),
	)
}

// pushRenderPassMarker
func externPushRenderPassMarker(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.pushRenderPassMarker_args)(args)

	e.pushRenderPassMarker(
		VkRenderPass(a.renderPass),
	)
}

// readMappedCoherentMemory
func externReadMappedCoherentMemory(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.readMappedCoherentMemory_args)(args)

	e.readMappedCoherentMemory(
		VkDeviceMemory(a.memory),
		uint64(a.offset_in_mapped),
		memory.Size(a.readSize),
	)
}

// resetCmd
func externResetCmd(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.resetCmd_args)(args)

	e.resetCmd(
		VkCommandBuffer(a.buffer),
	)
}

// resetSubcontext
func externResetSubcontext(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	e.resetSubcontext()
}

// trackMappedCoherentMemory
func externTrackMappedCoherentMemory(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.trackMappedCoherentMemory_args)(args)

	e.trackMappedCoherentMemory(
		uint64(a.start),
		memory.Size(a.size),
	)
}

// unmapMemory
func externUnmapMemory(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.unmapMemory_args)(args)

	e.unmapMemory(
		U8ˢ{&a.slice},
	)
}

// untrackMappedCoherentMemory
func externUntrackMappedCoherentMemory(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.untrackMappedCoherentMemory_args)(args)

	e.untrackMappedCoherentMemory(
		uint64(a.start),
		memory.Size(a.size),
	)
}

// vkErrCommandBufferIncomplete
func externVkErrCommandBufferIncomplete(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.vkErrCommandBufferIncomplete_args)(args)

	e.vkErrCommandBufferIncomplete(
		VkCommandBuffer(a.cmdbuf),
	)
}

// vkErrExpectNVDedicatedlyAllocatedHandle
func externVkErrExpectNVDedicatedlyAllocatedHandle(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.vkErrExpectNVDedicatedlyAllocatedHandle_args)(args)

	e.vkErrExpectNVDedicatedlyAllocatedHandle(
		C.GoStringN((*C.char)(unsafe.Pointer(&a.handleType.data[0])), (C.int)(a.handleType.length)),
		uint64(a.handle),
	)
}

// vkErrInvalidDescriptorArrayElement
func externVkErrInvalidDescriptorArrayElement(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.vkErrInvalidDescriptorArrayElement_args)(args)

	e.vkErrInvalidDescriptorArrayElement(
		uint64(a.set),
		uint32(a.binding),
		uint32(a.arrayIndex),
	)
}

// vkErrInvalidHandle
func externVkErrInvalidHandle(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.vkErrInvalidHandle_args)(args)

	e.vkErrInvalidHandle(
		C.GoStringN((*C.char)(unsafe.Pointer(&a.handleType.data[0])), (C.int)(a.handleType.length)),
		uint64(a.handle),
	)
}

// vkErrInvalidImageLayout
func externVkErrInvalidImageLayout(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.vkErrInvalidImageLayout_args)(args)

	e.vkErrInvalidImageLayout(
		VkImage(a.img),
		uint32(a.aspect),
		uint32(a.layer),
		uint32(a.level),
		VkImageLayout(a.layout),
		VkImageLayout(a.expectedLayout),
	)
}

// vkErrInvalidImageSubresource
func externVkErrInvalidImageSubresource(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.vkErrInvalidImageSubresource_args)(args)

	e.vkErrInvalidImageSubresource(
		VkImage(a.img),
		C.GoStringN((*C.char)(unsafe.Pointer(&a.subresourceType.data[0])), (C.int)(a.subresourceType.length)),
		uint32(a.value),
	)
}

// vkErrNotNullPointer
func externVkErrNotNullPointer(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.vkErrNotNullPointer_args)(args)

	e.vkErrNotNullPointer(
		C.GoStringN((*C.char)(unsafe.Pointer(&a.pointerType.data[0])), (C.int)(a.pointerType.length)),
	)
}

// vkErrNullPointer
func externVkErrNullPointer(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.vkErrNullPointer_args)(args)

	e.vkErrNullPointer(
		C.GoStringN((*C.char)(unsafe.Pointer(&a.pointerType.data[0])), (C.int)(a.pointerType.length)),
	)
}

// vkErrUnrecognizedExtension
func externVkErrUnrecognizedExtension(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.vkErrUnrecognizedExtension_args)(args)

	e.vkErrUnrecognizedExtension(
		C.GoStringN((*C.char)(unsafe.Pointer(&a.name.data[0])), (C.int)(a.name.length)),
	)
}

func externRecordFenceSignal(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.recordFenceSignal_args)(args)

	e.recordFenceSignal(
		VkFence(a.fence),
	)
}

func externRecordFenceWait(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.recordFenceWait_args)(args)

	e.recordFenceWait(
		VkFence(a.fence),
	)
}

func externRecordFenceReset(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.recordFenceReset_args)(args)

	e.recordFenceReset(
		VkFence(a.fence),
	)
}

func externRecordAcquireNextImage(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.recordAcquireNextImage_args)(args)

	e.recordAcquireNextImage(
		VkSwapchainKHR(a.swapchain), uint32(a.imageIndex),
	)
}

func externRecordPresentSwapchainImage(env *executor.Env, args, out unsafe.Pointer) {
	e := externsFromEnv(env)
	a := (*C.recordPresentSwapchainImage_args)(args)

	e.recordPresentSwapchainImage(
		VkSwapchainKHR(a.swapchain), uint32(a.imageIndex),
	)
}
