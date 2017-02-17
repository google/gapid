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
	"fmt"
	"math/rand"

	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
)

type readFramebuffer struct {
	injections map[atom.ID][]func(ctx log.Context, out transform.Writer)
}

func newReadFramebuffer(ctx log.Context) *readFramebuffer {
	return &readFramebuffer{
		injections: make(map[atom.ID][]func(ctx log.Context, out transform.Writer)),
	}
}

// If we are acutally swapping, we really do want toshow the image before
// the framebuffer read.
func (t *readFramebuffer) Transform(ctx log.Context, id atom.ID, a atom.Atom, out transform.Writer) {
	isEof := a.AtomFlags().IsEndOfFrame()
	doMutate := func() {
		out.MutateAndWrite(ctx, id, a)
	}

	if !isEof {
		doMutate()
	}

	if r, ok := t.injections[id]; ok {
		for _, injection := range r {
			injection(ctx, out)
		}
		delete(t.injections, id)
	}

	if isEof {
		doMutate()
	}
}

func (t *readFramebuffer) Flush(ctx log.Context, out transform.Writer) {}

func (t *readFramebuffer) Depth(id atom.ID, res chan<- imgRes) {
	t.injections[id] = append(t.injections[id], func(ctx log.Context, out transform.Writer) {
		s := out.State()
		attachment := gfxapi.FramebufferAttachment_Depth
		w, h, form, attachmentIndex, err := GetState(s).getFramebufferAttachmentInfo(attachment)
		if err != nil {
			res <- imgRes{err: &service.ErrDataUnavailable{Reason: messages.ErrMessage("Invalid Depth attachment")}}
			return
		}
		imageViewDepth := GetState(s).LastUsedFramebuffer.ImageAttachments[attachmentIndex]
		depthImageObject := imageViewDepth.Image
		postImageData(ctx, s, depthImageObject, form, VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT, w, h, w, h, out, func(i imgRes) { res <- i })
	})
}

func (t *readFramebuffer) Color(id atom.ID, width, height, bufferIdx uint32, res chan<- imgRes) {
	t.injections[id] = append(t.injections[id], func(ctx log.Context, out transform.Writer) {
		s := out.State()
		attachment := gfxapi.FramebufferAttachment_Color0 + gfxapi.FramebufferAttachment(bufferIdx)
		w, h, form, attachmentIndex, err := GetState(s).getFramebufferAttachmentInfo(attachment)
		if err != nil {
			res <- imgRes{err: &service.ErrDataUnavailable{Reason: messages.ErrMessage("Invalid Color attachment")}}
			return
		}

		// TODO: Figure out a better way to select the framebuffer here.
		imageView := GetState(s).LastUsedFramebuffer.ImageAttachments[attachmentIndex]
		imageObject := imageView.Image
		postImageData(ctx, s, imageObject, form, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, w, h, width, height, out, func(i imgRes) { res <- i })
	})
}

func writeEach(ctx log.Context, out transform.Writer, atoms ...atom.Atom) {
	for _, a := range atoms {
		out.MutateAndWrite(ctx, atom.NoID, a)
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

func postImageData(ctx log.Context,
	s *gfxapi.State,
	imageObject *ImageObject,
	vkFormat VkFormat,
	aspectMask VkImageAspectFlagBits,
	imgWidth,
	imgHeight,
	reqWidth,
	reqHeight uint32,
	out transform.Writer,
	callback func(imgRes)) {
	attachmentImageFormat, err := getImageFormatFromVulkanFormat(vkFormat)
	if err != nil {
		callback(imgRes{err: &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}})
		return
	}
	// When depth image is requested, the format, which is used for resolving/bliting/copying attachment image data
	// to the mapped buffer might be different with the format used in image resource. This is because we need to
	// strip the stencil data if the source attachment image contains both depth and stencil data.
	formatOfImgRes := attachmentImageFormat
	if aspectMask == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT {
		formatOfImgRes, err = getDepthImageFormatFromVulkanFormat(vkFormat)
		if err != nil {
			callback(imgRes{err: &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}})
			return
		}
	}

	queue := imageObject.LastBoundQueue
	vkQueue := queue.VulkanHandle
	vkDevice := queue.Device
	device := GetState(s).Devices[vkDevice]
	vkPhysicalDevice := device.PhysicalDevice
	physicalDevice := GetState(s).PhysicalDevices[vkPhysicalDevice]
	// Rendered image should always has a graphics-capable queue bound, if none
	// of such a queue found for this image or the bound queue does not have
	// graphics capability, throw error messages and return.
	if queue == nil {
		callback(imgRes{err: &service.ErrDataUnavailable{Reason: messages.ErrMessage("The target image object has not been bound with a vkQueue")}})
		return
	}
	if properties, ok := physicalDevice.QueueFamilyProperties[queue.Family]; ok {
		if properties.QueueFlags&VkQueueFlags(VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT) == 0 {
			callback(imgRes{err: &service.ErrDataUnavailable{Reason: messages.ErrMessage("The bound vkQueue does not have VK_QUEUE_GRAPHICS_BIT capability")}})
			return
		}
	} else {
		callback(imgRes{err: &service.ErrDataUnavailable{Reason: messages.ErrMessage("Not found the properties information of the bound vkQueue")}})
		return
	}

	// Wraps the data allocation so the data get freed at the end.
	var allocated []*atom.AllocResult
	defer func() {
		for _, d := range allocated {
			d.Free()
		}
	}()
	MustAllocData := func(
		ctx log.Context, s *gfxapi.State, v ...interface{}) atom.AllocResult {
		allocate_result := atom.Must(atom.AllocData(ctx, s, v...))
		allocated = append(allocated, &allocate_result)
		return allocate_result
	}

	fenceId := VkFence(newUnusedID(false, func(x uint64) bool { _, ok := GetState(s).Fences[VkFence(x)]; return ok }))
	fenceCreateInfo := VkFenceCreateInfo{
		SType: VkStructureType_VK_STRUCTURE_TYPE_FENCE_CREATE_INFO,
		PNext: NewVoidᶜᵖ(0),
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
		t := physicalDevice.MemoryProperties.MemoryTypes.Elements[i]
		if 0 != (t.PropertyFlags & VkMemoryPropertyFlags(VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT|
			VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_HOST_COHERENT_BIT)) {
			bufferMemoryTypeIndex = i
			break
		}
	}

	bufferSize := uint64(formatOfImgRes.Size(int(reqWidth), int(reqHeight)))

	// Data and info for destination buffer creation
	bufferId := VkBuffer(newUnusedID(false, func(x uint64) bool { _, ok := GetState(s).Buffers[VkBuffer(x)]; return ok }))
	bufferMemoryId := VkDeviceMemory(newUnusedID(false, func(x uint64) bool { _, ok := GetState(s).DeviceMemories[VkDeviceMemory(x)]; return ok }))
	bufferMemoryAllocInfo := VkMemoryAllocateInfo{
		SType:           VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
		PNext:           NewVoidᶜᵖ(0),
		AllocationSize:  VkDeviceSize(bufferSize),
		MemoryTypeIndex: bufferMemoryTypeIndex,
	}
	bufferMemoryAllocateInfoData := MustAllocData(ctx, s, bufferMemoryAllocInfo)
	bufferMemoryData := MustAllocData(ctx, s, bufferMemoryId)
	bufferCreateInfo := VkBufferCreateInfo{
		SType:                 VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
		PNext:                 NewVoidᶜᵖ(0),
		Flags:                 VkBufferCreateFlags(0),
		Size:                  VkDeviceSize(bufferSize),
		Usage:                 VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT),
		SharingMode:           VkSharingMode_VK_SHARING_MODE_EXCLUSIVE,
		QueueFamilyIndexCount: 0,
		PQueueFamilyIndices:   NewU32ᶜᵖ(0),
	}
	bufferCreateInfoData := MustAllocData(ctx, s, bufferCreateInfo)
	bufferData := MustAllocData(ctx, s, bufferId)

	// Data and info for staging image creation
	stagingImageId := VkImage(newUnusedID(false, func(x uint64) bool { _, ok := GetState(s).Images[VkImage(x)]; return ok }))
	stagingImageCreateInfo := VkImageCreateInfo{
		SType:     VkStructureType_VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
		PNext:     NewVoidᶜᵖ(0),
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
		PQueueFamilyIndices:   NewU32ᶜᵖ(0),
		InitialLayout:         VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
	}
	stagingImageCreateInfoData := MustAllocData(ctx, s, stagingImageCreateInfo)
	stagingImageData := MustAllocData(ctx, s, stagingImageId)
	stagingImageMemoryId := VkDeviceMemory(newUnusedID(false, func(x uint64) bool {
		_, ok := GetState(s).DeviceMemories[VkDeviceMemory(x)]
		ok = ok || VkDeviceMemory(x) == bufferMemoryId
		return ok
	}))
	stagingImageMemoryData := MustAllocData(ctx, s, stagingImageMemoryId)

	// Data and info for resolve image creation. Resolve image is used when the attachment image is multi-sampled
	resolveImageId := VkImage(newUnusedID(false, func(x uint64) bool { _, ok := GetState(s).Images[VkImage(x)]; return ok }))
	resolveImageCreateInfo := VkImageCreateInfo{
		SType:     VkStructureType_VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
		PNext:     NewVoidᶜᵖ(0),
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
		PQueueFamilyIndices:   NewU32ᶜᵖ(0),
		InitialLayout:         VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
	}
	resolveImageCreateInfoData := MustAllocData(ctx, s, resolveImageCreateInfo)
	resolveImageData := MustAllocData(ctx, s, resolveImageId)
	// TODO: We should definitely figure out a better source for the size of this
	// allocation, it may need to be larger
	resolveImageMemoryId := VkDeviceMemory(newUnusedID(false, func(x uint64) bool {
		_, ok := GetState(s).DeviceMemories[VkDeviceMemory(x)]
		ok = ok || VkDeviceMemory(x) == bufferMemoryId || VkDeviceMemory(x) == stagingImageMemoryId
		return ok
	}))
	resolveImageMemoryData := MustAllocData(ctx, s, resolveImageMemoryId)

	// Command pool and command buffer
	commandPoolId := VkCommandPool(newUnusedID(false, func(x uint64) bool { _, ok := GetState(s).CommandPools[VkCommandPool(x)]; return ok }))
	commandPoolCreateInfo := VkCommandPoolCreateInfo{
		SType:            VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
		PNext:            NewVoidᶜᵖ(0),
		Flags:            VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT),
		QueueFamilyIndex: queue.Family,
	}
	commandPoolCreateInfoData := MustAllocData(ctx, s, commandPoolCreateInfo)
	commandPoolData := MustAllocData(ctx, s, commandPoolId)
	commandBufferAllocateInfo := VkCommandBufferAllocateInfo{
		SType:              VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		PNext:              NewVoidᶜᵖ(0),
		CommandPool:        commandPoolId,
		Level:              VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	}
	commandBufferAllocateInfoData := MustAllocData(ctx, s, commandBufferAllocateInfo)
	commandBufferId := VkCommandBuffer(newUnusedID(true, func(x uint64) bool { _, ok := GetState(s).CommandBuffers[VkCommandBuffer(x)]; return ok }))
	commandBufferData := MustAllocData(ctx, s, commandBufferId)

	// Data and info for Vulkan commands in command buffers
	beginCommandBufferInfo := VkCommandBufferBeginInfo{
		SType:            VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
		PNext:            NewVoidᶜᵖ(0),
		Flags:            VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT),
		PInheritanceInfo: NewVkCommandBufferInheritanceInfoᶜᵖ(0),
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
		PNext:                NewVoidᶜᵖ(0),
		WaitSemaphoreCount:   0,
		PWaitSemaphores:      NewVkSemaphoreᶜᵖ(0),
		PWaitDstStageMask:    NewVkPipelineStageFlagsᶜᵖ(0),
		CommandBufferCount:   1,
		PCommandBuffers:      NewVkCommandBufferᶜᵖ(commandBuffers.Address()),
		SignalSemaphoreCount: 0,
		PSignalSemaphores:    NewVkSemaphoreᶜᵖ(0),
	}
	submitInfoData := MustAllocData(ctx, s, submitInfo)

	mappedMemoryRange := VkMappedMemoryRange{
		SType:  VkStructureType_VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE,
		PNext:  NewVoidᶜᵖ(0),
		Memory: bufferMemoryId,
		Offset: VkDeviceSize(0),
		Size:   VkDeviceSize(0xFFFFFFFFFFFFFFFF),
	}
	mappedMemoryRangeData := MustAllocData(ctx, s, mappedMemoryRange)
	at, err := s.Allocator.Alloc(bufferSize, 8)
	if err != nil {
		callback(imgRes{err: &service.ErrDataUnavailable{Reason: messages.ErrMessage("Device Memory -> Host mapping failed")}})
	}
	mappedPointer := MustAllocData(ctx, s, NewVoidᶜᵖ(at))

	// Barrier data for layout transitions of staging image
	stagingImageToDstBarrier := VkImageMemoryBarrier{
		SType: VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		PNext: NewVoidᶜᵖ(0),
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
		PNext: NewVoidᶜᵖ(0),
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
		PNext: NewVoidᶜᵖ(0),
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
		PNext: NewVoidᶜᵖ(0),
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
		PNext: NewVoidᶜᵖ(0),
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
		PNext: NewVoidᶜᵖ(0),
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
			Elements: [2]VkOffset3D{
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
		},
		DstSubresource: VkImageSubresourceLayers{
			AspectMask:     VkImageAspectFlags(aspectMask),
			MipLevel:       0,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
		DstOffsets: VkOffset3Dː2ᵃ{
			Elements: [2]VkOffset3D{
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
		NewVkCreateImage(
			vkDevice,
			stagingImageCreateInfoData.Ptr(),
			memory.Pointer{},
			stagingImageData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			stagingImageCreateInfoData.Data(),
		).AddWrite(
			stagingImageData.Data(),
		),
		NewReplayAllocateImageMemory(
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
		NewVkBindImageMemory(
			vkDevice,
			stagingImageId,
			stagingImageMemoryId,
			VkDeviceSize(0),
			VkResult_VK_SUCCESS,
		),
	)
	// Create buffer, allocate and bind memory
	writeEach(ctx, out,
		NewVkCreateBuffer(
			vkDevice,
			bufferCreateInfoData.Ptr(),
			memory.Pointer{},
			bufferData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			bufferCreateInfoData.Data(),
		).AddWrite(
			bufferData.Data(),
		),
		NewVkAllocateMemory(
			vkDevice,
			bufferMemoryAllocateInfoData.Ptr(),
			memory.Pointer{},
			bufferMemoryData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			bufferMemoryAllocateInfoData.Data(),
		).AddWrite(
			bufferMemoryData.Data(),
		),
		NewVkBindBufferMemory(
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
			NewVkCreateImage(
				vkDevice,
				resolveImageCreateInfoData.Ptr(),
				memory.Pointer{},
				resolveImageData.Ptr(),
				VkResult_VK_SUCCESS,
			).AddRead(
				resolveImageCreateInfoData.Data(),
			).AddWrite(
				resolveImageData.Data(),
			),
			NewReplayAllocateImageMemory(
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
			NewVkBindImageMemory(
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
		NewVkCreateCommandPool(
			vkDevice,
			commandPoolCreateInfoData.Ptr(),
			memory.Pointer{},
			commandPoolData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			commandPoolCreateInfoData.Data(),
		).AddWrite(
			commandPoolData.Data(),
		),
		NewVkAllocateCommandBuffers(
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
		NewVkCreateFence(
			vkDevice,
			fenceCreateData.Ptr(),
			memory.Pointer{},
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
		NewVkBeginCommandBuffer(
			commandBufferId,
			beginCommandBufferInfoData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			beginCommandBufferInfoData.Data(),
		),
		NewVkCmdPipelineBarrier(
			commandBufferId,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkDependencyFlags(0),
			0,
			memory.Pointer{},
			0,
			memory.Pointer{},
			1,
			attachmentImageToSrcBarrierData.Ptr(),
		).AddRead(
			attachmentImageToSrcBarrierData.Data(),
		),
		NewVkCmdPipelineBarrier(
			commandBufferId,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkDependencyFlags(0),
			0,
			memory.Pointer{},
			0,
			memory.Pointer{},
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
			NewVkCmdPipelineBarrier(
				commandBufferId,
				VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
				VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
				VkDependencyFlags(0),
				0,
				memory.Pointer{},
				0,
				memory.Pointer{},
				1,
				resolveImageToDstBarrierData.Ptr(),
			).AddRead(
				resolveImageToDstBarrierData.Data(),
			),
			NewVkCmdResolveImage(
				commandBufferId,
				imageObject.VulkanHandle,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
				resolveImageId,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				1,
				imageResolveData.Ptr(),
			).AddRead(imageResolveData.Data()),
			NewVkCmdPipelineBarrier(
				commandBufferId,
				VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
				VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
				VkDependencyFlags(0),
				0,
				memory.Pointer{},
				0,
				memory.Pointer{},
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
	writeEach(ctx, out,
		NewVkCmdBlitImage(
			commandBufferId,
			blitSrcImage,
			VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
			stagingImageId,
			VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
			1,
			imageBlitData.Ptr(),
			VkFilter_VK_FILTER_LINEAR,
		).AddRead(imageBlitData.Data()),
	)

	// Change the layout of staging image and attachment image, copy staging image to buffer,
	// end command buffer
	writeEach(ctx, out,
		NewVkCmdPipelineBarrier(
			commandBufferId,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkDependencyFlags(0),
			0,
			memory.Pointer{},
			0,
			memory.Pointer{},
			1,
			stagingImageToSrcBarrierData.Ptr(),
		).AddRead(
			stagingImageToSrcBarrierData.Data(),
		),
		NewVkCmdPipelineBarrier(
			commandBufferId,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkDependencyFlags(0),
			0,
			memory.Pointer{},
			0,
			memory.Pointer{},
			1,
			attachmentImageResetLayoutBarrierData.Ptr(),
		).AddRead(
			attachmentImageResetLayoutBarrierData.Data(),
		),
		NewVkCmdCopyImageToBuffer(
			commandBufferId,
			stagingImageId,
			VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
			bufferId,
			1,
			bufferImageCopyData.Ptr(),
		).AddRead(
			bufferImageCopyData.Data(),
		),
		NewVkEndCommandBuffer(
			commandBufferId,
			VkResult_VK_SUCCESS,
		))

	// Submit all the commands above, wait until finish.
	writeEach(ctx, out,
		NewVkDeviceWaitIdle(vkDevice, VkResult_VK_SUCCESS),
		NewVkQueueSubmit(
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
		NewVkWaitForFences(
			vkDevice,
			1,
			fenceData.Ptr(),
			1,
			0xFFFFFFFFFFFFFFFF,
			VkResult_VK_SUCCESS,
		).AddRead(
			fenceData.Data(),
		),
		NewVkDeviceWaitIdle(vkDevice, VkResult_VK_SUCCESS),
	)

	// Dump the buffer data to host
	writeEach(ctx, out,
		NewVkMapMemory(
			vkDevice,
			bufferMemoryId,
			VkDeviceSize(0),
			VkDeviceSize(bufferSize),
			VkMemoryMapFlags(0),
			mappedPointer.Ptr(),
			VkResult_VK_SUCCESS,
		).AddWrite(mappedPointer.Data()),
		NewVkInvalidateMappedMemoryRanges(
			vkDevice,
			1,
			mappedMemoryRangeData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(mappedMemoryRangeData.Data()),
	)

	// Add post atom
	writeEach(ctx, out,
		replay.Custom(func(ctx log.Context, s *gfxapi.State, b *builder.Builder) error {
			b.Post(value.ObservedPointer(at), uint64(bufferSize), func(r pod.Reader, err error) error {
				var data []byte
				if err == nil {
					data = make([]byte, bufferSize)
					r.Data(data)
					r.Error()

					// Flip the image in Y axis
					rowSizeInBytes := uint64(formatOfImgRes.Size(int(reqWidth), 1))
					top := uint64(0)
					bottom := bufferSize - rowSizeInBytes
					var temp = make([]byte, rowSizeInBytes)
					for top <= bottom {
						copy(temp, data[top:top+rowSizeInBytes])
						copy(data[top:top+rowSizeInBytes], data[bottom:bottom+rowSizeInBytes])
						copy(data[bottom:bottom+rowSizeInBytes], temp)
						top += rowSizeInBytes
						bottom -= rowSizeInBytes
					}
				}
				if err != nil {
					err = fmt.Errorf("Could not read framebuffer data (expected length %d bytes): %v", bufferSize, err)
					data = nil
				}

				img := &image.Image2D{
					Data:   data,
					Width:  uint32(reqWidth),
					Height: uint32(reqHeight),
					Format: formatOfImgRes,
				}

				callback(imgRes{img: img, err: err})
				return err
			})
			return nil
		}),
	)
	// Free the device resources used for reading framebuffer
	writeEach(ctx, out,
		NewVkUnmapMemory(vkDevice, bufferMemoryId),
		NewVkDestroyBuffer(vkDevice, bufferId, memory.Pointer{}),
		NewVkDestroyCommandPool(vkDevice, commandPoolId, memory.Pointer{}),
		NewVkDestroyImage(vkDevice, stagingImageId, memory.Pointer{}),
		NewVkFreeMemory(vkDevice, stagingImageMemoryId, memory.Pointer{}),
		NewVkFreeMemory(vkDevice, bufferMemoryId, memory.Pointer{}))
	if imageObject.Info.Samples != VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT {
		writeEach(ctx, out,
			NewVkDestroyImage(vkDevice, resolveImageId, memory.Pointer{}),
			NewVkFreeMemory(vkDevice, resolveImageMemoryId, memory.Pointer{}))
	}
	writeEach(ctx, out, NewVkDestroyFence(vkDevice, fenceId, memory.Pointer{}))
}
