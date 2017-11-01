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
	"math/rand"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
)

type readFramebuffer struct {
	injections map[api.CmdID][]func(context.Context, api.Cmd, transform.Writer)
}

func newReadFramebuffer(ctx context.Context) *readFramebuffer {
	return &readFramebuffer{
		injections: make(map[api.CmdID][]func(context.Context, api.Cmd, transform.Writer)),
	}
}

// If we are acutally swapping, we really do want to show the image before
// the framebuffer read.
func (t *readFramebuffer) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	s := out.State()
	isEOF := cmd.CmdFlags(ctx, id, s).IsEndOfFrame()
	doMutate := func() {
		out.MutateAndWrite(ctx, id, cmd)
	}

	if !isEOF {
		doMutate()
	} else {
		// This is a VkQueuePresent, we need to extract the information out of this,
		// so that we can correctly display the image.
		cmd.Mutate(ctx, id, out.State(), nil)
	}

	if r, ok := t.injections[id]; ok {
		for _, injection := range r {
			injection(ctx, cmd, out)
		}
		delete(t.injections, id)
	}

	if isEOF {
		doMutate()
	}
}

func (t *readFramebuffer) Flush(ctx context.Context, out transform.Writer) {}

func (t *readFramebuffer) Depth(id api.CmdID, idx uint32, res replay.Result) {
	t.injections[id] = append(t.injections[id], func(ctx context.Context, cmd api.Cmd, out transform.Writer) {
		s := out.State()

		c := GetState(s)
		lastQueue := c.LastBoundQueue
		if lastQueue == nil {
			res(nil, fmt.Errorf("No previous queue submission"))
			return
		}

		lastDrawInfo, ok := c.LastDrawInfos.Lookup(lastQueue.VulkanHandle)
		if !ok {
			res(nil, fmt.Errorf("There have been no previous draws"))
			return
		}
		w, h := lastDrawInfo.Framebuffer.Width, lastDrawInfo.Framebuffer.Height

		imageViewDepth := lastDrawInfo.Framebuffer.ImageAttachments.Get(idx)
		depthImageObject := imageViewDepth.Image
		cb := CommandBuilder{Thread: cmd.Thread()}
		postImageData(ctx, cb, s, depthImageObject, imageViewDepth.Format, VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT, w, h, w, h, out, res)
	})
}

func (t *readFramebuffer) Color(id api.CmdID, width, height, bufferIdx uint32, res replay.Result) {
	t.injections[id] = append(t.injections[id], func(ctx context.Context, cmd api.Cmd, out transform.Writer) {
		s := out.State()
		c := GetState(s)

		cb := CommandBuilder{Thread: cmd.Thread()}

		// TODO: Figure out a better way to select the framebuffer here.
		if GetState(s).LastSubmission == LastSubmissionType_SUBMIT {
			lastQueue := c.LastBoundQueue
			if lastQueue == nil {
				res(nil, fmt.Errorf("No previous queue submission"))
				return
			}

			lastDrawInfo, ok := c.LastDrawInfos.Lookup(lastQueue.VulkanHandle)
			if !ok {
				res(nil, fmt.Errorf("There have been no previous draws"))
				return
			}
			if lastDrawInfo.Framebuffer == nil {
				res(nil, fmt.Errorf("There has been no framebuffer"))
				return
			}

			imageView, ok := lastDrawInfo.Framebuffer.ImageAttachments.Lookup(bufferIdx)
			if !ok {
				res(nil, fmt.Errorf("There has been no attchment %v in the framebuffer", bufferIdx))
				return
			}
			imageObject := imageView.Image
			w, h, form := lastDrawInfo.Framebuffer.Width, lastDrawInfo.Framebuffer.Height, imageView.Format
			postImageData(ctx, cb, s, imageObject, form, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, w, h, width, height, out, res)
		} else {
			imageObject := GetState(s).LastPresentInfo.PresentImages.Get(bufferIdx)
			if imageObject == nil {
				res(nil, fmt.Errorf("Could not find imageObject %v, %v", id, bufferIdx))
				return
			}
			w, h, form := imageObject.Info.Extent.Width, imageObject.Info.Extent.Height, imageObject.Info.Format
			postImageData(ctx, cb, s, imageObject, form, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, w, h, width, height, out, res)
		}
	})
}

func writeEach(ctx context.Context, out transform.Writer, cmds ...api.Cmd) {
	for _, cmd := range cmds {
		out.MutateAndWrite(ctx, api.CmdNoID, cmd)
	}
}

func newUnusedID(isDispatchable bool, existenceTest func(uint64) bool) uint64 {
	for {
		x := uint64(rand.Uint32())
		if !isDispatchable {
			x = x<<32 | uint64(rand.Uint32())
		}
		if !existenceTest(x) && x != 0 {
			return x
		}
	}
}

func postImageData(ctx context.Context,
	cb CommandBuilder,
	s *api.GlobalState,
	imageObject *ImageObject,
	vkFormat VkFormat,
	aspectMask VkImageAspectFlagBits,
	imgWidth,
	imgHeight,
	reqWidth,
	reqHeight uint32,
	out transform.Writer,
	res replay.Result) {

	// This is the format used for building the final image resource and
	// calculating the data size for the final resource. Note that the staging
	// image is not created with this format.
	var formatOfImgRes *image.Format = nil
	var err error = nil
	if aspectMask == VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT {
		formatOfImgRes, err = getImageFormatFromVulkanFormat(vkFormat)
	} else if aspectMask == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT {
		// When depth image is requested, the format, which is used for
		// resolving/bliting/copying attachment image data to the mapped buffer
		// might be different with the format used in image resource. This is
		// because we need to strip the stencil data if the source attachment image
		// contains both depth and stencil data.
		formatOfImgRes, err = getDepthImageFormatFromVulkanFormat(vkFormat)
	} else {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()})
		return
	}
	if err != nil {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()})
		return
	}

	queue := imageObject.LastBoundQueue
	vkQueue := queue.VulkanHandle
	vkDevice := queue.Device
	device := GetState(s).Devices.Get(vkDevice)
	vkPhysicalDevice := device.PhysicalDevice
	physicalDevice := GetState(s).PhysicalDevices.Get(vkPhysicalDevice)
	// Rendered image should always has a graphics-capable queue bound, if none
	// of such a queue found for this image or the bound queue does not have
	// graphics capability, throw error messages and return.
	if queue == nil {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("The target image object has not been bound with a vkQueue")})
		return
	}
	if properties, ok := physicalDevice.QueueFamilyProperties.Lookup(queue.Family); ok {
		if properties.QueueFlags&VkQueueFlags(VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT) == 0 {
			res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("The bound vkQueue does not have VK_QUEUE_GRAPHICS_BIT capability")})
			return
		}
	} else {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Not found the properties information of the bound vkQueue")})
		return
	}

	// Wraps the data allocation so the data get freed at the end.
	var allocated []*api.AllocResult
	defer func() {
		for _, d := range allocated {
			d.Free()
		}
	}()
	MustAllocData := func(ctx context.Context, s *api.GlobalState, v ...interface{}) api.AllocResult {
		allocate_result := s.AllocDataOrPanic(ctx, v...)
		allocated = append(allocated, &allocate_result)
		return allocate_result
	}

	fenceId := VkFence(newUnusedID(false, func(x uint64) bool { return GetState(s).Fences.Contains(VkFence(x)) }))
	fenceCreateInfo := VkFenceCreateInfo{
		SType: VkStructureType_VK_STRUCTURE_TYPE_FENCE_CREATE_INFO,
		PNext: NewVoidᶜᵖ(memory.Nullptr),
		Flags: VkFenceCreateFlags(0),
	}
	fenceCreateData := MustAllocData(ctx, s, fenceCreateInfo)
	fenceData := MustAllocData(ctx, s, fenceId)

	// The physical device memory properties are used for
	// replayAllocateImageMemory to find the correct memory type index and
	// allocate proper memory for our staging and resolving image.
	physicalDeviceMemoryPropertiesData := MustAllocData(ctx, s, physicalDevice.MemoryProperties)
	bufferMemoryTypeIndex := uint32(0)
	for i := uint32(0); i < physicalDevice.MemoryProperties.MemoryTypeCount; i++ {
		t := physicalDevice.MemoryProperties.MemoryTypes[i]
		if 0 != (t.PropertyFlags & VkMemoryPropertyFlags(VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT|
			VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_HOST_COHERENT_BIT)) {
			bufferMemoryTypeIndex = i
			break
		}
	}

	bufferSize := uint64(formatOfImgRes.Size(int(reqWidth), int(reqHeight), 1))

	// Data and info for destination buffer creation
	bufferId := VkBuffer(newUnusedID(false, func(x uint64) bool { ok := GetState(s).Buffers.Contains(VkBuffer(x)); return ok }))
	bufferMemoryId := VkDeviceMemory(newUnusedID(false, func(x uint64) bool { ok := GetState(s).DeviceMemories.Contains(VkDeviceMemory(x)); return ok }))
	bufferMemoryAllocInfo := VkMemoryAllocateInfo{
		SType:           VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
		PNext:           NewVoidᶜᵖ(memory.Nullptr),
		AllocationSize:  VkDeviceSize(bufferSize),
		MemoryTypeIndex: bufferMemoryTypeIndex,
	}
	bufferMemoryAllocateInfoData := MustAllocData(ctx, s, bufferMemoryAllocInfo)
	bufferMemoryData := MustAllocData(ctx, s, bufferMemoryId)
	bufferCreateInfo := VkBufferCreateInfo{
		SType:                 VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
		PNext:                 NewVoidᶜᵖ(memory.Nullptr),
		Flags:                 VkBufferCreateFlags(0),
		Size:                  VkDeviceSize(bufferSize),
		Usage:                 VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT),
		SharingMode:           VkSharingMode_VK_SHARING_MODE_EXCLUSIVE,
		QueueFamilyIndexCount: 0,
		PQueueFamilyIndices:   NewU32ᶜᵖ(memory.Nullptr),
	}
	bufferCreateInfoData := MustAllocData(ctx, s, bufferCreateInfo)
	bufferData := MustAllocData(ctx, s, bufferId)

	// Data and info for staging image creation
	stagingImageId := VkImage(newUnusedID(false, func(x uint64) bool { ok := GetState(s).Images.Contains(VkImage(x)); return ok }))
	stagingImageCreateInfo := VkImageCreateInfo{
		SType:     VkStructureType_VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
		PNext:     NewVoidᶜᵖ(memory.Nullptr),
		Flags:     VkImageCreateFlags(0),
		ImageType: VkImageType_VK_IMAGE_TYPE_2D,
		Format:    vkFormat,
		Extent: VkExtent3D{
			Width:  reqWidth,
			Height: reqHeight,
			Depth:  1,
		},
		MipLevels:   1,
		ArrayLayers: 1,
		Samples:     VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT,
		Tiling:      VkImageTiling_VK_IMAGE_TILING_OPTIMAL,
		Usage: VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_SRC_BIT |
			VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT),
		SharingMode:           VkSharingMode_VK_SHARING_MODE_EXCLUSIVE,
		QueueFamilyIndexCount: 0,
		PQueueFamilyIndices:   NewU32ᶜᵖ(memory.Nullptr),
		InitialLayout:         VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
	}
	stagingImageCreateInfoData := MustAllocData(ctx, s, stagingImageCreateInfo)
	stagingImageData := MustAllocData(ctx, s, stagingImageId)
	stagingImageMemoryId := VkDeviceMemory(newUnusedID(false, func(x uint64) bool {
		ok := GetState(s).DeviceMemories.Contains(VkDeviceMemory(x))
		ok = ok || VkDeviceMemory(x) == bufferMemoryId
		return ok
	}))
	stagingImageMemoryData := MustAllocData(ctx, s, stagingImageMemoryId)

	// Data and info for resolve image creation. Resolve image is used when the attachment image is multi-sampled
	resolveImageId := VkImage(newUnusedID(false, func(x uint64) bool { ok := GetState(s).Images.Contains(VkImage(x)); return ok }))
	resolveImageCreateInfo := VkImageCreateInfo{
		SType:     VkStructureType_VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
		PNext:     NewVoidᶜᵖ(memory.Nullptr),
		Flags:     VkImageCreateFlags(0),
		ImageType: VkImageType_VK_IMAGE_TYPE_2D,
		Format:    vkFormat,
		Extent: VkExtent3D{
			Width:  imgWidth,  // same width as the attachment image, not the request
			Height: imgHeight, // same height as the attachment image, not the request
			Depth:  1,
		},
		MipLevels:   1,
		ArrayLayers: 1,
		Samples:     VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT,
		Tiling:      VkImageTiling_VK_IMAGE_TILING_OPTIMAL,
		Usage: VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_SRC_BIT |
			VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT),
		SharingMode:           VkSharingMode_VK_SHARING_MODE_EXCLUSIVE,
		QueueFamilyIndexCount: 0,
		PQueueFamilyIndices:   NewU32ᶜᵖ(memory.Nullptr),
		InitialLayout:         VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
	}
	resolveImageCreateInfoData := MustAllocData(ctx, s, resolveImageCreateInfo)
	resolveImageData := MustAllocData(ctx, s, resolveImageId)
	resolveImageMemoryId := VkDeviceMemory(newUnusedID(false, func(x uint64) bool {
		ok := GetState(s).DeviceMemories.Contains(VkDeviceMemory(x))
		ok = ok || VkDeviceMemory(x) == bufferMemoryId || VkDeviceMemory(x) == stagingImageMemoryId
		return ok
	}))
	resolveImageMemoryData := MustAllocData(ctx, s, resolveImageMemoryId)

	// Command pool and command buffer
	commandPoolId := VkCommandPool(newUnusedID(false, func(x uint64) bool { ok := GetState(s).CommandPools.Contains(VkCommandPool(x)); return ok }))
	commandPoolCreateInfo := VkCommandPoolCreateInfo{
		SType:            VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
		PNext:            NewVoidᶜᵖ(memory.Nullptr),
		Flags:            VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT),
		QueueFamilyIndex: queue.Family,
	}
	commandPoolCreateInfoData := MustAllocData(ctx, s, commandPoolCreateInfo)
	commandPoolData := MustAllocData(ctx, s, commandPoolId)
	commandBufferAllocateInfo := VkCommandBufferAllocateInfo{
		SType:              VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		PNext:              NewVoidᶜᵖ(memory.Nullptr),
		CommandPool:        commandPoolId,
		Level:              VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	}
	commandBufferAllocateInfoData := MustAllocData(ctx, s, commandBufferAllocateInfo)
	commandBufferId := VkCommandBuffer(newUnusedID(true, func(x uint64) bool { ok := GetState(s).CommandBuffers.Contains(VkCommandBuffer(x)); return ok }))
	commandBufferData := MustAllocData(ctx, s, commandBufferId)

	// Data and info for Vulkan commands in command buffers
	beginCommandBufferInfo := VkCommandBufferBeginInfo{
		SType:            VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
		PNext:            NewVoidᶜᵖ(memory.Nullptr),
		Flags:            VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT),
		PInheritanceInfo: NewVkCommandBufferInheritanceInfoᶜᵖ(memory.Nullptr),
	}
	beginCommandBufferInfoData := MustAllocData(ctx, s, beginCommandBufferInfo)

	bufferImageCopy := VkBufferImageCopy{
		BufferOffset:      VkDeviceSize(0),
		BufferRowLength:   0,
		BufferImageHeight: 0,
		ImageSubresource: VkImageSubresourceLayers{
			AspectMask:     VkImageAspectFlags(aspectMask),
			MipLevel:       0,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
		ImageOffset: VkOffset3D{X: 0, Y: 0, Z: 0},
		ImageExtent: VkExtent3D{Width: reqWidth, Height: reqHeight, Depth: 1},
	}
	bufferImageCopyData := MustAllocData(ctx, s, bufferImageCopy)

	commandBuffers := MustAllocData(ctx, s, commandBufferId)
	submitInfo := VkSubmitInfo{
		SType:                VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO,
		PNext:                NewVoidᶜᵖ(memory.Nullptr),
		WaitSemaphoreCount:   0,
		PWaitSemaphores:      NewVkSemaphoreᶜᵖ(memory.Nullptr),
		PWaitDstStageMask:    NewVkPipelineStageFlagsᶜᵖ(memory.Nullptr),
		CommandBufferCount:   1,
		PCommandBuffers:      NewVkCommandBufferᶜᵖ(commandBuffers.Ptr()),
		SignalSemaphoreCount: 0,
		PSignalSemaphores:    NewVkSemaphoreᶜᵖ(memory.Nullptr),
	}
	submitInfoData := MustAllocData(ctx, s, submitInfo)

	mappedMemoryRange := VkMappedMemoryRange{
		SType:  VkStructureType_VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE,
		PNext:  NewVoidᶜᵖ(memory.Nullptr),
		Memory: bufferMemoryId,
		Offset: VkDeviceSize(0),
		Size:   VkDeviceSize(0xFFFFFFFFFFFFFFFF),
	}
	mappedMemoryRangeData := MustAllocData(ctx, s, mappedMemoryRange)
	at, err := s.Allocator.Alloc(bufferSize, 8)
	if err != nil {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Device Memory -> Host mapping failed")})
	}
	mappedPointer := MustAllocData(ctx, s, Voidᶜᵖ{at, memory.ApplicationPool})

	// Barrier data for layout transitions of staging image
	stagingImageToDstBarrier := VkImageMemoryBarrier{
		SType: VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		PNext: NewVoidᶜᵖ(memory.Nullptr),
		SrcAccessMask: VkAccessFlags(
			VkAccessFlagBits_VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_SHADER_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT,
		),
		DstAccessMask:       VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT),
		NewLayout:           VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
		OldLayout:           VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
		SrcQueueFamilyIndex: 0xFFFFFFFF,
		DstQueueFamilyIndex: 0xFFFFFFFF,
		Image:               stagingImageId,
		SubresourceRange: VkImageSubresourceRange{
			AspectMask:     VkImageAspectFlags(aspectMask),
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}
	stagingImageToDstBarrierData := MustAllocData(ctx, s, stagingImageToDstBarrier)

	stagingImageToSrcBarrier := VkImageMemoryBarrier{
		SType: VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		PNext: NewVoidᶜᵖ(memory.Nullptr),
		SrcAccessMask: VkAccessFlags(
			VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT,
		),
		DstAccessMask:       VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT),
		NewLayout:           VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
		OldLayout:           VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
		SrcQueueFamilyIndex: 0xFFFFFFFF,
		DstQueueFamilyIndex: 0xFFFFFFFF,
		Image:               stagingImageId,
		SubresourceRange: VkImageSubresourceRange{
			AspectMask:     VkImageAspectFlags(aspectMask),
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}
	stagingImageToSrcBarrierData := MustAllocData(ctx, s, stagingImageToSrcBarrier)

	// Barrier data for layout transitions of resolve image. This only used when the attachment image is
	// multi-sampled.
	resolveImageToDstBarrier := VkImageMemoryBarrier{
		SType: VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		PNext: NewVoidᶜᵖ(memory.Nullptr),
		SrcAccessMask: VkAccessFlags(
			VkAccessFlagBits_VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_SHADER_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT,
		),
		DstAccessMask:       VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT),
		NewLayout:           VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
		OldLayout:           VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
		SrcQueueFamilyIndex: 0xFFFFFFFF,
		DstQueueFamilyIndex: 0xFFFFFFFF,
		Image:               resolveImageId,
		SubresourceRange: VkImageSubresourceRange{
			AspectMask:     VkImageAspectFlags(aspectMask),
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}
	resolveImageToDstBarrierData := MustAllocData(ctx, s, resolveImageToDstBarrier)

	resolveImageToSrcBarrier := VkImageMemoryBarrier{
		SType: VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		PNext: NewVoidᶜᵖ(memory.Nullptr),
		SrcAccessMask: VkAccessFlags(
			VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT,
		),
		DstAccessMask:       VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT),
		NewLayout:           VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
		OldLayout:           VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
		SrcQueueFamilyIndex: 0xFFFFFFFF,
		DstQueueFamilyIndex: 0xFFFFFFFF,
		Image:               resolveImageId,
		SubresourceRange: VkImageSubresourceRange{
			AspectMask:     VkImageAspectFlags(aspectMask),
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}
	resolveImageToSrcBarrierData := MustAllocData(ctx, s, resolveImageToSrcBarrier)

	// Barrier data for layout transitions of attachment image
	attachmentImageToSrcBarrier := VkImageMemoryBarrier{
		SType: VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		PNext: NewVoidᶜᵖ(memory.Nullptr),
		SrcAccessMask: VkAccessFlags(
			VkAccessFlagBits_VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_SHADER_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT,
		),
		DstAccessMask:       VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT),
		NewLayout:           VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
		OldLayout:           imageObject.Info.Layout,
		SrcQueueFamilyIndex: 0xFFFFFFFF,
		DstQueueFamilyIndex: 0xFFFFFFFF,
		Image:               imageObject.VulkanHandle,
		SubresourceRange: VkImageSubresourceRange{
			AspectMask:     VkImageAspectFlags(aspectMask),
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}
	attachmentImageToSrcBarrierData := MustAllocData(ctx, s, attachmentImageToSrcBarrier)

	attachmentImageResetLayoutBarrier := VkImageMemoryBarrier{
		SType: VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		PNext: NewVoidᶜᵖ(memory.Nullptr),
		SrcAccessMask: VkAccessFlags(
			VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT,
		),
		DstAccessMask: VkAccessFlags(
			VkAccessFlagBits_VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_SHADER_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT |
				VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT),
		NewLayout:           imageObject.Info.Layout,
		OldLayout:           VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
		SrcQueueFamilyIndex: 0xFFFFFFFF,
		DstQueueFamilyIndex: 0xFFFFFFFF,
		Image:               imageObject.VulkanHandle,
		SubresourceRange: VkImageSubresourceRange{
			AspectMask:     VkImageAspectFlags(aspectMask),
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}
	attachmentImageResetLayoutBarrierData := MustAllocData(ctx, s, attachmentImageResetLayoutBarrier)

	// Observation data for vkCmdBlitImage
	imageBlit := VkImageBlit{
		SrcSubresource: VkImageSubresourceLayers{
			AspectMask:     VkImageAspectFlags(aspectMask),
			MipLevel:       0,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
		SrcOffsets: VkOffset3Dː2ᵃ{
			{
				X: 0,
				Y: 0,
				Z: 0,
			},
			{
				X: int32(imgWidth),
				Y: int32(imgHeight),
				Z: 1,
			},
		},
		DstSubresource: VkImageSubresourceLayers{
			AspectMask:     VkImageAspectFlags(aspectMask),
			MipLevel:       0,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
		DstOffsets: VkOffset3Dː2ᵃ{
			{
				X: int32(0),
				Y: int32(0),
				Z: 0,
			},
			{
				X: int32(reqWidth),
				Y: int32(reqHeight),
				Z: 1,
			},
		},
	}
	imageBlitData := MustAllocData(ctx, s, imageBlit)

	// Observation data for vkCmdResolveImage
	imageResolve := VkImageResolve{
		SrcSubresource: VkImageSubresourceLayers{
			AspectMask:     VkImageAspectFlags(aspectMask),
			MipLevel:       0,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
		SrcOffset: VkOffset3D{
			X: 0,
			Y: 0,
			Z: 0,
		},
		DstSubresource: VkImageSubresourceLayers{
			AspectMask:     VkImageAspectFlags(aspectMask),
			MipLevel:       0,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
		DstOffset: VkOffset3D{
			X: 0,
			Y: 0,
			Z: 0,
		},
		Extent: VkExtent3D{
			Width:  uint32(imgWidth),
			Height: uint32(imgHeight),
			Depth:  1,
		},
	}
	imageResolveData := MustAllocData(ctx, s, imageResolve)

	// Write atoms to writer
	// Create staging image, allocate and bind memory
	writeEach(ctx, out,
		cb.VkCreateImage(
			vkDevice,
			stagingImageCreateInfoData.Ptr(),
			memory.Nullptr,
			stagingImageData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			stagingImageCreateInfoData.Data(),
		).AddWrite(
			stagingImageData.Data(),
		),
		cb.ReplayAllocateImageMemory(
			vkDevice,
			physicalDeviceMemoryPropertiesData.Ptr(),
			stagingImageId,
			stagingImageMemoryData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			physicalDeviceMemoryPropertiesData.Data(),
		).AddWrite(
			stagingImageMemoryData.Data(),
		),
		cb.VkBindImageMemory(
			vkDevice,
			stagingImageId,
			stagingImageMemoryId,
			VkDeviceSize(0),
			VkResult_VK_SUCCESS,
		),
	)
	// Create buffer, allocate and bind memory
	writeEach(ctx, out,
		cb.VkCreateBuffer(
			vkDevice,
			bufferCreateInfoData.Ptr(),
			memory.Nullptr,
			bufferData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			bufferCreateInfoData.Data(),
		).AddWrite(
			bufferData.Data(),
		),
		cb.VkAllocateMemory(
			vkDevice,
			bufferMemoryAllocateInfoData.Ptr(),
			memory.Nullptr,
			bufferMemoryData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			bufferMemoryAllocateInfoData.Data(),
		).AddWrite(
			bufferMemoryData.Data(),
		),
		cb.VkBindBufferMemory(
			vkDevice,
			bufferId,
			bufferMemoryId,
			VkDeviceSize(0),
			VkResult_VK_SUCCESS,
		),
	)

	// If the attachment image is multi-sampled, an resolve image is required
	// Create resolve image, allocate and bind memory
	if imageObject.Info.Samples != VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT {
		writeEach(ctx, out,
			cb.VkCreateImage(
				vkDevice,
				resolveImageCreateInfoData.Ptr(),
				memory.Nullptr,
				resolveImageData.Ptr(),
				VkResult_VK_SUCCESS,
			).AddRead(
				resolveImageCreateInfoData.Data(),
			).AddWrite(
				resolveImageData.Data(),
			),
			cb.ReplayAllocateImageMemory(
				vkDevice,
				physicalDeviceMemoryPropertiesData.Ptr(),
				resolveImageId,
				resolveImageMemoryData.Ptr(),
				VkResult_VK_SUCCESS,
			).AddRead(
				physicalDeviceMemoryPropertiesData.Data(),
			).AddWrite(
				resolveImageMemoryData.Data(),
			),
			cb.VkBindImageMemory(
				vkDevice,
				resolveImageId,
				resolveImageMemoryId,
				VkDeviceSize(0),
				VkResult_VK_SUCCESS,
			),
		)
	}

	// Create command pool, allocate command buffer
	writeEach(ctx, out,
		cb.VkCreateCommandPool(
			vkDevice,
			commandPoolCreateInfoData.Ptr(),
			memory.Nullptr,
			commandPoolData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			commandPoolCreateInfoData.Data(),
		).AddWrite(
			commandPoolData.Data(),
		),
		cb.VkAllocateCommandBuffers(
			vkDevice,
			commandBufferAllocateInfoData.Ptr(),
			commandBufferData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			commandBufferAllocateInfoData.Data(),
		).AddWrite(
			commandBufferData.Data(),
		),
	)

	// Create a fence
	writeEach(ctx, out,
		cb.VkCreateFence(
			vkDevice,
			fenceCreateData.Ptr(),
			memory.Nullptr,
			fenceData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			fenceCreateData.Data(),
		).AddWrite(
			fenceData.Data(),
		),
	)

	// Begin command buffer, change attachment image and staging image layout
	writeEach(ctx, out,
		cb.VkBeginCommandBuffer(
			commandBufferId,
			beginCommandBufferInfoData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			beginCommandBufferInfoData.Data(),
		),
		cb.VkCmdPipelineBarrier(
			commandBufferId,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkDependencyFlags(0),
			0,
			memory.Nullptr,
			0,
			memory.Nullptr,
			1,
			attachmentImageToSrcBarrierData.Ptr(),
		).AddRead(
			attachmentImageToSrcBarrierData.Data(),
		),
		cb.VkCmdPipelineBarrier(
			commandBufferId,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkDependencyFlags(0),
			0,
			memory.Nullptr,
			0,
			memory.Nullptr,
			1,
			stagingImageToDstBarrierData.Ptr(),
		).AddRead(
			stagingImageToDstBarrierData.Data(),
		),
	)

	// If the attachment image is multi-sampled, resolve the attchment image to resolve image before
	// blit the image. Change the resolve image layout, call vkCmdResolveImage, change the resolve
	// image layout again.fmt
	if imageObject.Info.Samples != VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT {
		writeEach(ctx, out,
			cb.VkCmdPipelineBarrier(
				commandBufferId,
				VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
				VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
				VkDependencyFlags(0),
				0,
				memory.Nullptr,
				0,
				memory.Nullptr,
				1,
				resolveImageToDstBarrierData.Ptr(),
			).AddRead(
				resolveImageToDstBarrierData.Data(),
			),
			cb.VkCmdResolveImage(
				commandBufferId,
				imageObject.VulkanHandle,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
				resolveImageId,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				1,
				imageResolveData.Ptr(),
			).AddRead(imageResolveData.Data()),
			cb.VkCmdPipelineBarrier(
				commandBufferId,
				VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
				VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
				VkDependencyFlags(0),
				0,
				memory.Nullptr,
				0,
				memory.Nullptr,
				1,
				resolveImageToSrcBarrierData.Ptr(),
			).AddRead(
				resolveImageToSrcBarrierData.Data(),
			),
		)
	}

	// Blit image, if the attachment image is multi-sampled, use resolve image as the source image, otherwise
	// use the attachment image as the source image directly
	blitSrcImage := imageObject.VulkanHandle
	if imageObject.Info.Samples != VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT {
		blitSrcImage = resolveImageId
	}
	// If the src image is a depth/stencil image, the filter must be NEAREST
	filter := VkFilter_VK_FILTER_LINEAR
	if aspectMask != VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT {
		filter = VkFilter_VK_FILTER_NEAREST
	}
	writeEach(ctx, out,
		cb.VkCmdBlitImage(
			commandBufferId,
			blitSrcImage,
			VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
			stagingImageId,
			VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
			1,
			imageBlitData.Ptr(),
			filter,
		).AddRead(imageBlitData.Data()),
	)

	// Change the layout of staging image and attachment image, copy staging image to buffer,
	// end command buffer
	writeEach(ctx, out,
		cb.VkCmdPipelineBarrier(
			commandBufferId,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkDependencyFlags(0),
			0,
			memory.Nullptr,
			0,
			memory.Nullptr,
			1,
			stagingImageToSrcBarrierData.Ptr(),
		).AddRead(
			stagingImageToSrcBarrierData.Data(),
		),
		cb.VkCmdPipelineBarrier(
			commandBufferId,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkDependencyFlags(0),
			0,
			memory.Nullptr,
			0,
			memory.Nullptr,
			1,
			attachmentImageResetLayoutBarrierData.Ptr(),
		).AddRead(
			attachmentImageResetLayoutBarrierData.Data(),
		),
		cb.VkCmdCopyImageToBuffer(
			commandBufferId,
			stagingImageId,
			VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
			bufferId,
			1,
			bufferImageCopyData.Ptr(),
		).AddRead(
			bufferImageCopyData.Data(),
		),
		cb.VkEndCommandBuffer(
			commandBufferId,
			VkResult_VK_SUCCESS,
		))

	// Submit all the commands above, wait until finish.
	writeEach(ctx, out,
		cb.VkDeviceWaitIdle(vkDevice, VkResult_VK_SUCCESS),
		cb.VkQueueSubmit(
			vkQueue,
			1,
			submitInfoData.Ptr(),
			fenceId,
			VkResult_VK_SUCCESS,
		).AddRead(
			submitInfoData.Data(),
		).AddRead(
			commandBuffers.Data(),
		),
		cb.VkWaitForFences(
			vkDevice,
			1,
			fenceData.Ptr(),
			1,
			0xFFFFFFFFFFFFFFFF,
			VkResult_VK_SUCCESS,
		).AddRead(
			fenceData.Data(),
		),
		cb.VkDeviceWaitIdle(vkDevice, VkResult_VK_SUCCESS),
	)

	// Dump the buffer data to host
	writeEach(ctx, out,
		cb.VkMapMemory(
			vkDevice,
			bufferMemoryId,
			VkDeviceSize(0),
			VkDeviceSize(bufferSize),
			VkMemoryMapFlags(0),
			mappedPointer.Ptr(),
			VkResult_VK_SUCCESS,
		).AddWrite(mappedPointer.Data()),
		cb.VkInvalidateMappedMemoryRanges(
			vkDevice,
			1,
			mappedMemoryRangeData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(mappedMemoryRangeData.Data()),
	)

	// Add post command
	writeEach(ctx, out,
		cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
			b.Post(value.ObservedPointer(at), uint64(bufferSize), func(r binary.Reader, err error) error {
				var bytes []byte
				if err == nil {
					bytes = make([]byte, bufferSize)
					r.Data(bytes)
					r.Error()

					// Flip the image in Y axis
					rowSizeInBytes := uint64(formatOfImgRes.Size(int(reqWidth), 1, 1))
					top := uint64(0)
					bottom := bufferSize - rowSizeInBytes
					var temp = make([]byte, rowSizeInBytes)
					for top < bottom {
						copy(temp, bytes[top:top+rowSizeInBytes])
						copy(bytes[top:top+rowSizeInBytes], bytes[bottom:bottom+rowSizeInBytes])
						copy(bytes[bottom:bottom+rowSizeInBytes], temp)
						top += rowSizeInBytes
						bottom -= rowSizeInBytes
					}
				}
				if err != nil {
					err = fmt.Errorf("Could not read framebuffer data (expected length %d bytes): %v", bufferSize, err)
					bytes = nil
				}

				img := &image.Data{
					Bytes:  bytes,
					Width:  uint32(reqWidth),
					Height: uint32(reqHeight),
					Depth:  1,
					Format: formatOfImgRes,
				}

				res(img, err)
				return err
			})
			return nil
		}),
	)
	// Free the device resources used for reading framebuffer
	writeEach(ctx, out,
		cb.VkUnmapMemory(vkDevice, bufferMemoryId),
		cb.VkDestroyBuffer(vkDevice, bufferId, memory.Nullptr),
		cb.VkDestroyCommandPool(vkDevice, commandPoolId, memory.Nullptr),
		cb.VkDestroyImage(vkDevice, stagingImageId, memory.Nullptr),
		cb.VkFreeMemory(vkDevice, stagingImageMemoryId, memory.Nullptr),
		cb.VkFreeMemory(vkDevice, bufferMemoryId, memory.Nullptr))
	if imageObject.Info.Samples != VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT {
		writeEach(ctx, out,
			cb.VkDestroyImage(vkDevice, resolveImageId, memory.Nullptr),
			cb.VkFreeMemory(vkDevice, resolveImageMemoryId, memory.Nullptr))
	}
	writeEach(ctx, out, cb.VkDestroyFence(vkDevice, fenceId, memory.Nullptr))
}
