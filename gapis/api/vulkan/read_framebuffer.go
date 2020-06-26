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

type pendingRead struct {
	device             VkDevice
	buffer             VkBuffer
	bufferSize         uint64
	bufferMemory       VkDeviceMemory
	commandPool        VkCommandPool
	stagingImage       VkImage
	stagingMemory      VkDeviceMemory
	resolveImage       VkImage
	resolveImageMemory VkDeviceMemory
	aspect             VkImageAspectFlagBits
	format             VkFormat
	imageFormat        *image.Format
	requestWidth       uint32
	requestHeight      uint32
	res                replay.Result
}

type injection struct {
	res replay.Result
	fn  func(context.Context, *InsertionCommand, replay.Result, transform.Writer) error
}

type readFramebuffer struct {
	injections         map[string][]injection
	numInitialCommands int
	pendingReads       []pendingRead
}

func newReadFramebuffer(ctx context.Context) *readFramebuffer {
	return &readFramebuffer{
		injections:         make(map[string][]injection),
		numInitialCommands: 0,
		pendingReads:       make([]pendingRead, 0),
	}
}

// If we are acutally swapping, we really do want to show the image before
// the framebuffer read.
func (t *readFramebuffer) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	if cmd, ok := cmd.(*InsertionCommand); ok {
		idx_string := keyFromIndex(cmd.idx)
		if r, ok := t.injections[idx_string]; ok {
			// If this command is FOR an EOF command, we want to mutate it, so that
			// we have the presentation info available.
			if cmd.callee != nil && cmd.callee.CmdFlags().IsEndOfFrame() {
				cmd.callee.Mutate(ctx, id, out.State(), nil, nil)
			}
			for _, injection := range r {
				if err := injection.fn(ctx, cmd, injection.res, out); err != nil {
					return err
				}
			}

			return nil
		}
	}
	if err := out.MutateAndWrite(ctx, id, cmd); err != nil {
		return err
	}
	if len(t.pendingReads) > 0 {
		if id != api.CmdNoID {
			return t.FlushPending(ctx, out)
		}
	}
	return nil
}

func (t *readFramebuffer) PreLoop(ctx context.Context, output transform.Writer)  {}
func (t *readFramebuffer) PostLoop(ctx context.Context, output transform.Writer) {}
func (t *readFramebuffer) BuffersCommands() bool                                 { return false }

func keyFromIndex(idx api.SubCmdIdx) string {
	return fmt.Sprintf("%v", idx)
}

func (t *readFramebuffer) Depth(ctx context.Context, id api.SubCmdIdx, requestWidth, requestHeight, idx uint32, res replay.Result) {
	t.injections[keyFromIndex(id)] = append(t.injections[keyFromIndex(id)], injection{res,
		func(ctx context.Context, cmd *InsertionCommand, res replay.Result, out transform.Writer) error {
			s := out.State()
			st := GetState(s)
			if cmd.cmdBuffer == VkCommandBuffer(0) {
				res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Please select a draw-call")})
				return nil
			}

			cmdBuff := st.CommandBuffers().Get(cmd.cmdBuffer)

			fb := cmdBuff.PreviousFramebuffer()
			rp := cmdBuff.PreviouslyStartedRenderpass()

			if fb.IsNil() {
				res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Please select a draw-call inside a renderpass")})
				return nil
			}

			if rp.IsNil() {
				res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Please select a draw-call inside a renderpass")})
				return nil
			}

			w, h := fb.Width(), fb.Height()

			imageViewDepth := fb.ImageAttachments().Get(idx)
			if imageViewDepth.IsNil() {
				res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Invalid depth attachment in the framebuffer, the attachment VkImageView might have been destroyed")})
				return nil
			}
			depthImageObject := imageViewDepth.Image()
			if depthImageObject.IsNil() {
				res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Invalid depth attachment in the framebuffer, the attachment VkImage might have been destroyed")})
				return nil
			}
			cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}
			// Imageviews that are used in framebuffer attachments must contains
			// only one mip level.
			level := imageViewDepth.SubresourceRange().BaseMipLevel()
			// There might be multiple layers, currently we only support the
			// first one.
			// TODO: support multi-layer rendering.
			layer := imageViewDepth.SubresourceRange().BaseArrayLayer()
			return t.postImageData(ctx, cb, id, s, cmd.cmdBuffer, cmd.pendingCommandBuffers, depthImageObject, imageViewDepth.Fmt(), VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT, layer, level, w, h, requestWidth, requestHeight, out, res)
		}})
}

func (t *readFramebuffer) Color(ctx context.Context, id api.SubCmdIdx, width, height, bufferIdx uint32, res replay.Result) {
	t.injections[keyFromIndex(id)] = append(t.injections[keyFromIndex(id)], injection{res,
		func(ctx context.Context, cmd *InsertionCommand, res replay.Result, out transform.Writer) error {
			s := out.State()
			c := GetState(s)

			cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}

			isPresent := cmd.callee != nil && cmd.callee.CmdFlags().IsEndOfFrame()

			// TODO: Figure out a better way to select the framebuffer here.

			if !isPresent {
				if cmd.cmdBuffer == VkCommandBuffer(0) {
					res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Please select a draw-call or VkQueuePresent")})
					return nil
				}
				cmdBuff := c.CommandBuffers().Get(cmd.cmdBuffer)

				fb := cmdBuff.PreviousFramebuffer()
				rp := cmdBuff.PreviouslyStartedRenderpass()

				if fb.IsNil() {
					res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Please select a draw-call inside a renderpass")})
					return nil
				}

				if rp.IsNil() {
					res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Please select a draw-call inside a renderpass")})
					return nil
				}

				imageView, ok := fb.ImageAttachments().Lookup(bufferIdx)
				if !ok {
					res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("There has been no attchment in the framebuffer")})
					return nil
				}
				if imageView.IsNil() {
					res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Invalid attachment in the framebuffer, the attachment VkImageView might have been destroyed")})
					return nil
				}
				// Imageviews that are used in framebuffer attachments must contains
				// only one mip level.
				level := imageView.SubresourceRange().BaseMipLevel()
				// There might be multiple layers, currently we only support the
				// first one.
				// TODO: support multi-layer rendering.
				layer := imageView.SubresourceRange().BaseArrayLayer()
				imageObject := imageView.Image()
				if imageObject.IsNil() {
					res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Invalid attachment in the framebuffer, the attachment VkImage might have been destroyed")})
					return nil
				}
				w, h, form := fb.Width(), fb.Height(), imageView.Fmt()
				return t.postImageData(ctx, cb, id, s, cmd.cmdBuffer, cmd.pendingCommandBuffers, imageObject, form, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, layer, level, w, h, width, height, out, res)
			}

			imageObject := GetState(s).LastPresentInfo().PresentImages().Get(bufferIdx)
			if imageObject.IsNil() {
				res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Could not find imageObject")})
				return nil
			}
			w, h, form := imageObject.Info().Extent().Width(), imageObject.Info().Extent().Height(), imageObject.Info().Fmt()
			// There might be multiple layers for an image created by swapchain
			// but currently we only support layer 0.
			// TODO: support multi-layer swapchain images.
			return t.postImageData(ctx, cb, id, s, VkCommandBuffer(0), []VkCommandBuffer{}, imageObject, form, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, 0, 0, w, h, width, height, out, res)
		}})
}

func writeEach(ctx context.Context, out transform.Writer, cmds ...api.Cmd) error {
	for _, cmd := range cmds {
		if err := out.MutateAndWrite(ctx, api.CmdNoID, cmd); err != nil {
			return err
		}
	}
	return nil
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

func (t *readFramebuffer) getLayout(ctx context.Context,
	s *api.GlobalState,
	cmdBuff VkCommandBuffer,
	pendingCommandBuffers []VkCommandBuffer,
	aspect VkImageAspectFlagBits,
	layer uint32,
	level uint32,
	img ImageObjectʳ) VkImageLayout {
	st := GetState(s)
	layout := img.Aspects().Get(aspect).Layers().Get(layer).Levels().Get(level).Layout()
	if cmdBuff == VkCommandBuffer(0) {
		return layout
	}

	cbs := append(pendingCommandBuffers, cmdBuff)
	// Walk through all pending transitions for this image, and make sure they
	// we reflect the most recent one.
	for _, cb := range cbs {
		cb := st.CommandBuffers().Get(cb)
		if cb.IsNil() {
			continue
		}

		if !cb.ImageTransitions().Contains(img.VulkanHandle()) {
			continue
		}
		transitions := cb.ImageTransitions().Get(img.VulkanHandle())
		if !transitions.AspectTransitions().Contains(aspect) {
			continue
		}
		transition_key := (uint64(layer) << 8) | uint64(level&0xFF)
		aspect_transition := transitions.AspectTransitions().Get(aspect)
		if !aspect_transition.Layouts().Contains(transition_key) {
			continue
		}
		layout = aspect_transition.Layouts().Get(transition_key)
	}
	return layout
}

func (t *readFramebuffer) postImageData(ctx context.Context,
	cb CommandBuilder,
	idx api.SubCmdIdx,
	s *api.GlobalState,
	cmdBuff VkCommandBuffer,
	pendingCommandBuffers []VkCommandBuffer,
	imageObject ImageObjectʳ,
	vkFormat VkFormat,
	aspect VkImageAspectFlagBits,
	layer,
	level,
	imgWidth,
	imgHeight,
	requestWidth,
	requestHeight uint32,
	out transform.Writer,
	res replay.Result) error {
	st := GetState(s)

	a := s.Arena // TODO: Use a temporary arena?

	// This is the format used for building the final image resource and
	// calculating the data size for the final resource. Note that the staging
	// image is not created with this format.
	var formatOfImgRes *image.Format
	var err error
	if aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT {
		formatOfImgRes, err = getImageFormatFromVulkanFormat(vkFormat)
	} else if aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT {
		// When depth image is requested, the format, which is used for
		// resolving/bliting/copying attachment image data to the mapped buffer
		// might be different with the format used in image resource. This is
		// because we need to strip the stencil data if the source attachment image
		// contains both depth and stencil data.
		formatOfImgRes, err = getDepthImageFormatFromVulkanFormat(vkFormat)
	} else if aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT {
		// Similarly to above, we may need to strip the depth data if the
		// source attachment image contains both depth and stencil data.
		formatOfImgRes, err = getStencilImageFormatFromVulkanFormat(vkFormat)
	} else {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()})
		return nil
	}
	if err != nil {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()})
		return nil
	}

	resolveSrcDepth := int32(0)
	blitSrcDepth := int32(0)
	copySrcDepth := int32(0)

	if imageObject.Info().ImageType() == VkImageType_VK_IMAGE_TYPE_3D {
		resolveSrcDepth = int32(layer)
		blitSrcDepth = int32(layer)
		copySrcDepth = int32(layer)
		layer = 0
	}
	resolveSrcLayer := layer
	blitSrcLayer := layer
	copySrcLayer := layer
	if imageObject.Info().Samples() != VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT {
		resolveSrcLayer = layer
		blitSrcDepth = 0
		blitSrcLayer = 0
		copySrcDepth = 0
		copySrcLayer = 0
	}
	doBlit := !(requestWidth == imgWidth && requestHeight == imgHeight)
	if doBlit {
		copySrcDepth = 0
		copySrcLayer = 0
	}

	origLayout := t.getLayout(ctx, s, cmdBuff, pendingCommandBuffers, aspect, layer, level, imageObject)

	queue := NilQueueObjectʳ
	if cmdBuff != VkCommandBuffer(0) {
		cbo := st.CommandBuffers().Get(cmdBuff)
		cp := st.CommandPools().Get(cbo.Pool())
		for _, v := range st.Queues().All() {
			if v.Family() == cp.QueueFamilyIndex() {
				queue = v
			}
		}
	}
	if queue.IsNil() {
		queue = imageObject.Aspects().Get(aspect).Layers().Get(layer).Levels().Get(level).LastBoundQueue()
	}

	if queue.IsNil() {
		queue = imageObject.LastBoundQueue()
		if queue.IsNil() {
			res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("The target image object has not been bound with a vkQueue")})
			return nil
		}
	}
	queueFamily := queue.Family()

	vkQueue := queue.VulkanHandle()
	vkDevice := queue.Device()
	device := st.Devices().Get(vkDevice)
	vkPhysicalDevice := device.PhysicalDevice()
	physicalDevice := st.PhysicalDevices().Get(vkPhysicalDevice)

	if properties, ok := physicalDevice.QueueFamilyProperties().Lookup(queueFamily); ok {
		if properties.QueueFlags()&VkQueueFlags(VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT) == 0 {
			if imageObject.Info().Samples() == VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT &&
				aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT {
				// If this is on the compute queue, we cannot do a blit operation,
				// We can however do it on the CPU afterwards, or let the
				// client deal with it
				requestWidth = imgWidth
				requestHeight = imgHeight
			} else {
				res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Unhandled: Reading a multisampled or depth image on the compute queue")})
				return nil
			}
		}
	} else {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Not found the properties information of the bound vkQueue")})
		return nil
	}

	// Wraps the data allocation so the data get freed at the end.
	var allocated []*api.AllocResult
	defer func() {
		for _, d := range allocated {
			d.Free()
		}
	}()
	MustAllocData := func(ctx context.Context, s *api.GlobalState, v ...interface{}) api.AllocResult {
		res := s.AllocDataOrPanic(ctx, v...)
		allocated = append(allocated, &res)
		return res
	}

	// The physical device memory properties are used for
	// replayAllocateImageMemory to find the correct memory type index and
	// allocate proper memory for our staging and resolving image.
	physicalDeviceMemoryPropertiesData := MustAllocData(ctx, s, physicalDevice.MemoryProperties())
	bufferMemoryTypeIndex := uint32(0)
	for i := uint32(0); i < physicalDevice.MemoryProperties().MemoryTypeCount(); i++ {
		t := physicalDevice.MemoryProperties().MemoryTypes().Get(int(i))
		if 0 != (t.PropertyFlags() & VkMemoryPropertyFlags(
			VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT|
				VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_HOST_COHERENT_BIT)) {
			bufferMemoryTypeIndex = i
			break
		}
	}

	bufferSize := uint64(formatOfImgRes.Size(int(requestWidth), int(requestHeight), 1))
	// For the depth aspect of VK_FORMAT_X8_D24_UNORM_PACK32 and
	// VK_FORMAT_D24_UNORM_S8_UINT format, each depth element requires 4 bytes in
	// the buffer when used in buffer image copy.
	if aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT && (vkFormat == VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32 || vkFormat == VkFormat_VK_FORMAT_D24_UNORM_S8_UINT) {
		r32Fmt, _ := getImageFormatFromVulkanFormat(VkFormat_VK_FORMAT_R32_UINT)
		bufferSize = uint64(r32Fmt.Size(int(requestWidth), int(requestHeight), 1))
	}

	// Data and info for destination buffer creation
	bufferID := VkBuffer(newUnusedID(false, func(x uint64) bool { ok := st.Buffers().Contains(VkBuffer(x)); return ok }))
	bufferMemoryID := VkDeviceMemory(newUnusedID(false, func(x uint64) bool { ok := st.DeviceMemories().Contains(VkDeviceMemory(x)); return ok }))
	bufferMemoryAllocInfo := NewVkMemoryAllocateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, // sType
		0,                          // pNext
		VkDeviceSize(bufferSize*2), // allocationSize
		bufferMemoryTypeIndex,      // memoryTypeIndex
	)
	bufferMemoryAllocateInfoData := MustAllocData(ctx, s, bufferMemoryAllocInfo)
	bufferMemoryData := MustAllocData(ctx, s, bufferMemoryID)
	bufferCreateInfo := NewVkBufferCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,                       // sType
		NewVoidᶜᵖ(memory.Nullptr),                                                  // pNext
		VkBufferCreateFlags(0),                                                     // flags
		VkDeviceSize(bufferSize),                                                   // size
		VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT), // usage
		VkSharingMode_VK_SHARING_MODE_EXCLUSIVE,                                    // sharingMode
		0,                                                                          // queueFamilyIndexCount
		NewU32ᶜᵖ(memory.Nullptr),                                                   // pQueueFamilyIndices
	)
	bufferCreateInfoData := MustAllocData(ctx, s, bufferCreateInfo)
	bufferData := MustAllocData(ctx, s, bufferID)

	// Data and info for staging image creation
	stagingImageID := VkImage(newUnusedID(false, func(x uint64) bool { ok := st.Images().Contains(VkImage(x)); return ok }))
	stagingImageCreateInfo := NewVkImageCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO, // sType
		0,                            // pNext
		0,                            // flags
		VkImageType_VK_IMAGE_TYPE_2D, // imageType
		vkFormat,                     // format
		NewVkExtent3D(a, // extent
			requestWidth,
			requestHeight,
			1,
		),
		1, // mipLevels
		1, // arrayLayers
		VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT, // samples
		VkImageTiling_VK_IMAGE_TILING_OPTIMAL,       // tiling
		VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_SRC_BIT|
			VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT), // usage
		VkSharingMode_VK_SHARING_MODE_EXCLUSIVE, // sharingMode
		0,                                       // queueFamilyIndexCount
		0,                                       // pQueueFamilyIndices
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED, // initialLayout
	)
	stagingImageCreateInfoData := MustAllocData(ctx, s, stagingImageCreateInfo)
	stagingImageData := MustAllocData(ctx, s, stagingImageID)
	stagingImageMemoryID := VkDeviceMemory(newUnusedID(false, func(x uint64) bool {
		ok := st.DeviceMemories().Contains(VkDeviceMemory(x))
		ok = ok || VkDeviceMemory(x) == bufferMemoryID
		return ok
	}))
	stagingImageMemoryData := MustAllocData(ctx, s, stagingImageMemoryID)

	// Data and info for resolve image creation. Resolve image is used when the attachment image is multi-sampled
	resolveImageID := VkImage(newUnusedID(false, func(x uint64) bool { ok := st.Images().Contains(VkImage(x)); return ok }))
	resolveImageCreateInfo := NewVkImageCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO, // sType
		0,                            // pNext
		0,                            // flags
		VkImageType_VK_IMAGE_TYPE_2D, // imageType
		vkFormat,                     // format
		NewVkExtent3D(a, // extent
			imgWidth,  // same width as the attachment image, not the request
			imgHeight, // same height as the attachment image, not the request
			1),
		1, // mipLevels
		1, // arrayLayers
		VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT, // samples
		VkImageTiling_VK_IMAGE_TILING_OPTIMAL,       // tiling
		VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_SRC_BIT|
			VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT), // usage
		VkSharingMode_VK_SHARING_MODE_EXCLUSIVE, // sharingMode
		0,                                       // queueFamilyIndexCount
		0,                                       // pQueueFamilyIndices
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED, // initialLayout
	)
	resolveImageCreateInfoData := MustAllocData(ctx, s, resolveImageCreateInfo)
	resolveImageData := MustAllocData(ctx, s, resolveImageID)
	resolveImageMemoryID := VkDeviceMemory(newUnusedID(false, func(x uint64) bool {
		ok := st.DeviceMemories().Contains(VkDeviceMemory(x))
		ok = ok || VkDeviceMemory(x) == bufferMemoryID || VkDeviceMemory(x) == stagingImageMemoryID
		return ok
	}))
	resolveImageMemoryData := MustAllocData(ctx, s, resolveImageMemoryID)

	commandBufferID := cmdBuff
	commandPoolID := VkCommandPool(0)
	if cmdBuff == VkCommandBuffer(0) {
		// Command pool and command buffer
		commandPoolID = VkCommandPool(newUnusedID(false, func(x uint64) bool { ok := st.CommandPools().Contains(VkCommandPool(x)); return ok }))
		commandPoolCreateInfo := NewVkCommandPoolCreateInfo(a,
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,                                 // sType
			NewVoidᶜᵖ(memory.Nullptr),                                                                  // pNext
			VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT), // flags
			queue.Family(), // queueFamilyIndex
		)
		commandPoolCreateInfoData := MustAllocData(ctx, s, commandPoolCreateInfo)
		commandPoolData := MustAllocData(ctx, s, commandPoolID)
		commandBufferAllocateInfo := NewVkCommandBufferAllocateInfo(a,
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
			NewVoidᶜᵖ(memory.Nullptr),                                      // pNext
			commandPoolID,                                                  // commandPool
			VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY,           // level
			1, // commandBufferCount
		)
		commandBufferAllocateInfoData := MustAllocData(ctx, s, commandBufferAllocateInfo)

		commandBufferID = VkCommandBuffer(newUnusedID(true, func(x uint64) bool { ok := st.CommandBuffers().Contains(VkCommandBuffer(x)); return ok }))
		commandBufferData := MustAllocData(ctx, s, commandBufferID)

		// Data and info for Vulkan commands in command buffers
		beginCommandBufferInfo := NewVkCommandBufferBeginInfo(a,
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
			0, // pNext
			VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT), // flags
			0, // pInheritanceInfo
		)
		beginCommandBufferInfoData := MustAllocData(ctx, s, beginCommandBufferInfo)

		// Create command pool, allocate command buffer, and begin it
		if err := writeEach(ctx, out,
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
			cb.VkBeginCommandBuffer(
				commandBufferID,
				beginCommandBufferInfoData.Ptr(),
				VkResult_VK_SUCCESS,
			).AddRead(
				beginCommandBufferInfoData.Data(),
			),
		); err != nil {
			return err
		}
	}

	bufferImageCopy := NewVkBufferImageCopy(a,
		0, // bufferOffset
		0, // bufferRowLength
		0, // bufferImageHeight
		NewVkImageSubresourceLayers(a, // imageSubresource
			VkImageAspectFlags(aspect), // aspectMask
			level,                      // mipLevel
			copySrcLayer,               // baseArrayLayer
			1,                          // layerCount
		),
		NewVkOffset3D(a, int32(0), int32(0), copySrcDepth), // imageOffset
		NewVkExtent3D(a, requestWidth, requestHeight, 1),   // imageExtent
	)
	bufferImageCopyData := MustAllocData(ctx, s, bufferImageCopy)

	barrierAspectMask := VkImageAspectFlags(aspect)
	depthStencilMask := VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT |
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT
	if VkImageAspectFlagBits(imageObject.ImageAspect())&depthStencilMask == depthStencilMask {
		barrierAspectMask |= VkImageAspectFlags(depthStencilMask)
	}
	// Barrier data for layout transitions of staging image
	stagingImageToDstBarrier := NewVkImageMemoryBarrier(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
		0, // pNext
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT|
			VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT|
			VkAccessFlagBits_VK_ACCESS_SHADER_WRITE_BIT|
			VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT), // srcAccessMask
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT), // dstAccessMask
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,                      // oldLayout
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,           // newLayout
		0xFFFFFFFF,     // srcQueueFamilyIndex
		0xFFFFFFFF,     // dstQueueFamilyIndex
		stagingImageID, // image
		NewVkImageSubresourceRange(a, // subresourceRange
			barrierAspectMask, // aspectMask
			0,                 // baseMipLevel
			1,                 // levelCount
			0,                 // baseArrayLayer
			1,                 // layerCount
		),
	)
	stagingImageToDstBarrierData := MustAllocData(ctx, s, stagingImageToDstBarrier)

	stagingImageToSrcBarrier := NewVkImageMemoryBarrier(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
		0, // pNext
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT), // srcAccessMask
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT),  // dstAccessMask
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,           // oldLayout
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,           // newLayout
		0xFFFFFFFF,     // srcQueueFamilyIndex
		0xFFFFFFFF,     // dstQueueFamilyIndex
		stagingImageID, // image
		NewVkImageSubresourceRange(a, // subresourceRange
			barrierAspectMask, // aspectMask
			0,                 // baseMipLevel
			1,                 // levelCount
			0,                 // baseArrayLayer
			1,                 // layerCount
		),
	)
	stagingImageToSrcBarrierData := MustAllocData(ctx, s, stagingImageToSrcBarrier)

	// Barrier data for layout transitions of resolve image. This only used when the attachment image is
	// multi-sampled.
	resolveImageToDstBarrier := NewVkImageMemoryBarrier(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
		0, // pNext
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT|
			VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT|
			VkAccessFlagBits_VK_ACCESS_SHADER_WRITE_BIT|
			VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT), // srcAccessMask
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT), // dstAccessMask
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,                      // oldLayout
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,           // newLayout
		0xFFFFFFFF,     // srcQueueFamilyIndex
		0xFFFFFFFF,     // dstQueueFamilyIndex
		resolveImageID, // image
		NewVkImageSubresourceRange(a, // subresourceRange
			barrierAspectMask, // aspectMask
			0,                 // baseMipLevel
			1,                 // levelCount
			0,                 // baseArrayLayer
			1,                 // layerCount
		),
	)
	resolveImageToDstBarrierData := MustAllocData(ctx, s, resolveImageToDstBarrier)

	resolveImageToSrcBarrier := NewVkImageMemoryBarrier(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
		0, // pNext
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT), // srcAccessMask
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT),  // dstAccessMask
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,           // oldLayout
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,           // newLayout
		0xFFFFFFFF,     // srcQueueFamilyIndex
		0xFFFFFFFF,     // dstQueueFamilyIndex
		resolveImageID, // image
		NewVkImageSubresourceRange(a, // subresourceRange
			barrierAspectMask, // aspectMask
			0,                 // baseMipLevel
			1,                 // levelCount
			0,                 // baseArrayLayer
			1,                 // layerCount
		),
	)
	resolveImageToSrcBarrierData := MustAllocData(ctx, s, resolveImageToSrcBarrier)

	// Barrier data for layout transitions of attachment image
	attachmentImageToSrcBarrier := NewVkImageMemoryBarrier(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
		0, // pNext
		VkAccessFlags( // srcAccessMask
			VkAccessFlagBits_VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT|
				VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT|
				VkAccessFlagBits_VK_ACCESS_SHADER_WRITE_BIT|
				VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT|
				VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT,
		),
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT), // dstAccessMask
		origLayout, // oldLayout
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, // newLayout
		0xFFFFFFFF,                 // srcQueueFamilyIndex
		0xFFFFFFFF,                 // dstQueueFamilyIndex
		imageObject.VulkanHandle(), // image
		NewVkImageSubresourceRange(a, // subresourceRange
			barrierAspectMask, // aspectMask
			0,                 // baseMipLevel
			1,                 // levelCount
			0,                 // baseArrayLayer
			1,                 // layerCount
		),
	)
	attachmentImageToSrcBarrierData := MustAllocData(ctx, s, attachmentImageToSrcBarrier)

	attachmentImageResetLayoutBarrier := NewVkImageMemoryBarrier(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
		0, // pNext
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT), // srcAccessMask
		VkAccessFlags( // dstAccessMask
			VkAccessFlagBits_VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT|
				VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT|
				VkAccessFlagBits_VK_ACCESS_SHADER_WRITE_BIT|
				VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT|
				VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT),
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, // oldLayout
		origLayout,                 // newLayout
		0xFFFFFFFF,                 // srcQueueFamilyIndex
		0xFFFFFFFF,                 // dstQueueFamilyIndex
		imageObject.VulkanHandle(), // image
		NewVkImageSubresourceRange(a, // subresourceRange
			barrierAspectMask, // aspectMask
			0,                 // baseMipLevel
			1,                 // levelCount
			0,                 // baseArrayLayer
			1,                 // layerCount
		),
	)
	attachmentImageResetLayoutBarrierData := MustAllocData(ctx, s, attachmentImageResetLayoutBarrier)
	// Observation data for vkCmdBlitImage
	imageBlit := NewVkImageBlit(a,
		NewVkImageSubresourceLayers(a, // srcSubresource
			VkImageAspectFlags(aspect), // aspectMask
			0,                          // mipLevel
			blitSrcLayer,               // baseArrayLayer
			1,                          // layerCount
		),
		NewVkOffset3Dː2ᵃ(a, // srcOffsets
			NewVkOffset3D(a, int32(0), int32(0), blitSrcDepth),
			NewVkOffset3D(a, int32(imgWidth), int32(imgHeight), blitSrcDepth+int32(1)),
		),
		NewVkImageSubresourceLayers(a, // dstSubresource
			VkImageAspectFlags(aspect), // aspectMask
			0,                          // mipLevel
			0,                          // baseArrayLayer
			1,                          // layerCount
		),
		NewVkOffset3Dː2ᵃ(a, // dstOffsets
			MakeVkOffset3D(a),
			NewVkOffset3D(a, int32(requestWidth), int32(requestHeight), 1),
		),
	)
	imageBlitData := MustAllocData(ctx, s, imageBlit)

	// Observation data for vkCmdResolveImage
	imageResolve := NewVkImageResolve(a,
		NewVkImageSubresourceLayers(a, // srcSubresource
			VkImageAspectFlags(aspect), // aspectMask
			0,                          // mipLevel
			resolveSrcLayer,            // baseArrayLayer
			1,                          // layerCount
		),
		NewVkOffset3D(a, int32(0), int32(0), resolveSrcDepth), // srcOffset
		NewVkImageSubresourceLayers(a, // dstSubresource
			VkImageAspectFlags(aspect), // aspectMask
			0,                          // mipLevel
			0,                          // baseArrayLayer
			1,                          // layerCount
		),
		MakeVkOffset3D(a), // dstOffset
		NewVkExtent3D(a, uint32(imgWidth), uint32(imgHeight), 1), // extent
	)
	imageResolveData := MustAllocData(ctx, s, imageResolve)

	// Write commands to writer
	// Create staging image, allocate and bind memory
	if err := writeEach(ctx, out,
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
			stagingImageID,
			stagingImageMemoryData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			physicalDeviceMemoryPropertiesData.Data(),
		).AddWrite(
			stagingImageMemoryData.Data(),
		),
		cb.VkBindImageMemory(
			vkDevice,
			stagingImageID,
			stagingImageMemoryID,
			VkDeviceSize(0),
			VkResult_VK_SUCCESS,
		),
	); err != nil {
		return err
	}
	// Create buffer, allocate and bind memory
	if err := writeEach(ctx, out,
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
			bufferID,
			bufferMemoryID,
			VkDeviceSize(0),
			VkResult_VK_SUCCESS,
		),
	); err != nil {
		return err
	}

	// If the attachment image is multi-sampled, an resolve image is required
	// Create resolve image, allocate and bind memory
	if imageObject.Info().Samples() != VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT {
		if err := writeEach(ctx, out,
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
				resolveImageID,
				resolveImageMemoryData.Ptr(),
				VkResult_VK_SUCCESS,
			).AddRead(
				physicalDeviceMemoryPropertiesData.Data(),
			).AddWrite(
				resolveImageMemoryData.Data(),
			),
			cb.VkBindImageMemory(
				vkDevice,
				resolveImageID,
				resolveImageMemoryID,
				VkDeviceSize(0),
				VkResult_VK_SUCCESS,
			),
		); err != nil {
			return err
		}
	} else {
		resolveImageID = VkImage(0)
		resolveImageMemoryID = VkDeviceMemory(0)
	}

	// Change attachment image and staging image layout
	if err := writeEach(ctx, out,
		cb.VkCmdPipelineBarrier(
			commandBufferID,
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
			commandBufferID,
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
	); err != nil {
		return err
	}

	// If the attachment image is multi-sampled, resolve the attchment image to resolve image before
	// blit the image. Change the resolve image layout, call vkCmdResolveImage, change the resolve
	// image layout again.fmt
	if imageObject.Info().Samples() != VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT {
		if err := writeEach(ctx, out,
			cb.VkCmdPipelineBarrier(
				commandBufferID,
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
				commandBufferID,
				imageObject.VulkanHandle(),
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
				resolveImageID,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				1,
				imageResolveData.Ptr(),
			).AddRead(imageResolveData.Data()),
			cb.VkCmdPipelineBarrier(
				commandBufferID,
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
		); err != nil {
			return err
		}
	}

	// Blit image, if the attachment image is multi-sampled, use resolve image as the source image, otherwise
	// use the attachment image as the source image directly
	blitSrcImage := imageObject.VulkanHandle()
	if imageObject.Info().Samples() != VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT {
		blitSrcImage = resolveImageID
	}
	// If the src image is a depth/stencil image, the filter must be NEAREST
	filter := VkFilter_VK_FILTER_LINEAR
	if aspect != VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT {
		filter = VkFilter_VK_FILTER_NEAREST
	}

	copySrc := blitSrcImage

	if doBlit {
		copySrc = stagingImageID
		if err := writeEach(ctx, out,
			cb.VkCmdBlitImage(
				commandBufferID,
				blitSrcImage,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
				stagingImageID,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				1,
				imageBlitData.Ptr(),
				filter,
			).AddRead(imageBlitData.Data()),
			// Set the blit image to transfer src
			cb.VkCmdPipelineBarrier(
				commandBufferID,
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
		); err != nil {
			return err
		}
	}

	if err := writeEach(ctx, out,
		cb.VkCmdCopyImageToBuffer(
			commandBufferID,
			copySrc,
			VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
			bufferID,
			1,
			bufferImageCopyData.Ptr(),
		).AddRead(
			bufferImageCopyData.Data(),
		),
	); err != nil {
		return err
	}

	if err := writeEach(ctx, out,
		// Reset the image, and end the command buffer.
		cb.VkCmdPipelineBarrier(
			commandBufferID,
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
	); err != nil {
		return err
	}
	if cmdBuff == VkCommandBuffer(0) {
		if err := writeEach(ctx, out,
			cb.VkEndCommandBuffer(
				commandBufferID,
				VkResult_VK_SUCCESS,
			),
		); err != nil {
			return err
		}
	}

	// If we had to allocate this command buff ourselves, that means we need to submit it ourselves.
	if cmdBuff == VkCommandBuffer(0) {
		commandBuffers := MustAllocData(ctx, s, commandBufferID)
		submitInfo := NewVkSubmitInfo(a,
			VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
			0, // pNext
			0, // waitSemaphoreCount
			0, // pWaitSemaphores
			0, // pWaitDstStageMask
			1, // commandBufferCount
			NewVkCommandBufferᶜᵖ(commandBuffers.Ptr()), // pCommandBuffers
			0, // signalSemaphoreCount
			0, // pSignalSemaphores
		)
		submitInfoData := MustAllocData(ctx, s, submitInfo)
		if err := writeEach(ctx, out,
			cb.VkQueueSubmit(
				vkQueue,
				1,
				submitInfoData.Ptr(),
				VkFence(0),
				VkResult_VK_SUCCESS,
			).AddRead(
				submitInfoData.Data(),
			).AddRead(
				commandBuffers.Data(),
			),
			cb.VkDeviceWaitIdle(vkDevice, VkResult_VK_SUCCESS),
		); err != nil {
			return err
		}
	}

	t.pendingReads = append(t.pendingReads, pendingRead{
		device:             vkDevice,
		buffer:             bufferID,
		bufferMemory:       bufferMemoryID,
		bufferSize:         bufferSize,
		commandPool:        commandPoolID,
		stagingImage:       stagingImageID,
		stagingMemory:      stagingImageMemoryID,
		resolveImage:       resolveImageID,
		resolveImageMemory: resolveImageMemoryID,
		aspect:             aspect,
		format:             vkFormat,
		imageFormat:        formatOfImgRes,
		requestWidth:       requestWidth,
		requestHeight:      requestHeight,
		res:                res,
	})
	return nil
}

func (t *readFramebuffer) Flush(ctx context.Context, out transform.Writer) error {
	return t.FlushPending(ctx, out)
}

// Wrapper to avoid lambdas
type pendingReadWrapper struct {
	r  *pendingRead
	at api.AllocResult
}

func (w *pendingReadWrapper) postPendingRead(r binary.Reader, err error) {
	var bytes []byte
	if err == nil {
		bufferSize := w.r.bufferSize
		bytes = make([]byte, bufferSize)
		r.Data(bytes)
		r.Error()

		// For the depth aspect of VK_FORMAT_X8_D24_UNORM_PACK32 and
		// VK_FORMAT_D24_UNORM_S8_UINT format, we need to strip the
		// undefined value in the MSB byte.
		if w.r.aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT && (w.r.format == VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32 || w.r.format == VkFormat_VK_FORMAT_D24_UNORM_S8_UINT) {
			inBufSize := 4
			inImgSize := 3
			count := len(bytes) / inBufSize
			for i := 0; i < count; i++ {
				copy(bytes[i*inImgSize:(i+1)*inImgSize], bytes[i*inBufSize:(i+1)*inBufSize])
			}
			bufferSize = uint64(count * inImgSize)
			bytes = bytes[0:bufferSize]
		}

		// Flip the image in Y axis
		rowSizeInBytes := uint64(w.r.imageFormat.Size(int(w.r.requestWidth), 1, 1))
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
		err = fmt.Errorf("Could not read framebuffer data (expected length %d bytes): %v", w.r.bufferSize, err)
		bytes = nil
	}

	img := &image.Data{
		Bytes:  bytes,
		Width:  uint32(w.r.requestWidth),
		Height: uint32(w.r.requestHeight),
		Depth:  1,
		Format: w.r.imageFormat,
	}

	w.r.res(img, err)
}

func (w *pendingReadWrapper) customPost(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
	b.Post(value.ObservedPointer(w.at.Address()), uint64(w.r.bufferSize), w.postPendingRead)
	return nil
}

func (t *readFramebuffer) FlushPending(ctx context.Context, out transform.Writer) error {
	s := out.State()
	cb := CommandBuilder{Thread: 0, Arena: s.Arena}
	a := s.Arena // TODO: Use a temporary arena?
	for i := range t.pendingReads {
		// DO NOT BE TEMPTED TO TURN THIS INTO
		// for _, f := range t.pendingReads.
		// It will not do what you want.
		// There is a lambda capture down there, it will not do what
		// you expect.
		r := t.pendingReads[i]

		if err := writeEach(ctx, out,
			cb.VkDeviceWaitIdle(r.device, VkResult_VK_SUCCESS),
		); err != nil {
			return err
		}

		// Wraps the data allocation so the data get freed at the end.
		var allocated []*api.AllocResult
		MustAllocData := func(ctx context.Context, s *api.GlobalState, v ...interface{}) api.AllocResult {
			res := s.AllocDataOrPanic(ctx, v...)
			allocated = append(allocated, &res)
			return res
		}
		mappedMemoryRange := NewVkMappedMemoryRange(a,
			VkStructureType_VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, // sType
			0,                                // pNext
			r.bufferMemory,                   // memory
			VkDeviceSize(0),                  // offset
			VkDeviceSize(0xFFFFFFFFFFFFFFFF), // size
		)
		mappedMemoryRangeData := MustAllocData(ctx, s, mappedMemoryRange)
		at, err := s.Alloc(ctx, r.bufferSize)
		if err != nil {
			r.res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Device Memory -> Host mapping failed")})
		}
		allocated = append(allocated, &at)

		mappedPointer := MustAllocData(ctx, s, at.Address())

		// Dump the buffer data to host
		if err := writeEach(ctx, out,
			cb.VkMapMemory(
				r.device,
				r.bufferMemory,
				VkDeviceSize(0),
				VkDeviceSize(r.bufferSize),
				VkMemoryMapFlags(0),
				mappedPointer.Ptr(),
				VkResult_VK_SUCCESS,
			).AddWrite(mappedPointer.Data()),
			cb.VkInvalidateMappedMemoryRanges(
				r.device,
				1,
				mappedMemoryRangeData.Ptr(),
				VkResult_VK_SUCCESS,
			).AddRead(mappedMemoryRangeData.Data()),
		); err != nil {
			return err
		}

		wrap := &pendingReadWrapper{&r, at}
		// Add post command
		if err := writeEach(ctx, out, cb.Custom(wrap.customPost)); err != nil {
			return err
		}

		// Free the device resources used for reading framebuffer
		if err := writeEach(ctx, out,
			cb.VkUnmapMemory(r.device, r.bufferMemory),
			cb.VkDestroyBuffer(r.device, r.buffer, memory.Nullptr),
			cb.VkDestroyCommandPool(r.device, r.commandPool, memory.Nullptr),
			cb.VkDestroyImage(r.device, r.stagingImage, memory.Nullptr),
			cb.VkFreeMemory(r.device, r.stagingMemory, memory.Nullptr),
			cb.VkFreeMemory(r.device, r.bufferMemory, memory.Nullptr),
			cb.VkDestroyImage(r.device, r.resolveImage, memory.Nullptr),
			cb.VkFreeMemory(r.device, r.resolveImageMemory, memory.Nullptr),
		); err != nil {
			return err
		}

		for _, d := range allocated {
			d.Free()
		}
	}
	t.pendingReads = []pendingRead{}
	return nil
}
