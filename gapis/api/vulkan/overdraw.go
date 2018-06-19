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

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

var (
	_ = transform.Transformer(&stencilOverdraw{})
)

// TODO: add support for specific renderpass in a queue submit
type stencilOverdraw struct {
	rewrite map[api.CmdID]replay.Result
}

func newStencilOverdraw() *stencilOverdraw {
	return &stencilOverdraw{
		rewrite: map[api.CmdID]replay.Result{},
	}
}

func (s *stencilOverdraw) add(ctx context.Context, after []uint64, capt *path.Capture, res replay.Result) {
	c, err := capture.ResolveFromPath(ctx, capt)
	if err != nil {
		res(nil, err)
	}
	// TODO: Ideally this would be smarter, but without duplicating the
	// state and mutating it, it's hard to tell what the right
	// vkQueueSubmit to modify is.
	lastSubmit := ^uint64(0)
	for i, cmd := range c.Commands {
		if uint64(i) > after[0] {
			break
		}
		switch cmd.(type) {
		case *VkQueueSubmit:
			lastSubmit = uint64(i)
		}
	}
	if lastSubmit == ^uint64(0) {
		res(nil, &service.ErrDataUnavailable{
			Reason: messages.ErrMessage("No last queue submission"),
		})
		return
	}

	s.rewrite[api.CmdID(lastSubmit)] = res
}

func (s *stencilOverdraw) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	res, ok := s.rewrite[id]
	if !ok {
		out.MutateAndWrite(ctx, id, cmd)
		return
	}

	gs := out.State()
	st := GetState(gs)
	arena := gs.Arena

	var allocated []*api.AllocResult
	var cleanups []func()
	defer func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
		for _, d := range allocated {
			d.Free()
		}
	}()
	mustAllocData := func(v ...interface{}) api.AllocResult {
		res := gs.AllocDataOrPanic(ctx, v...)
		allocated = append(allocated, &res)
		return res
	}
	addCleanup := func(f func()) {
		cleanups = append(cleanups, f)
	}

	submit, ok := cmd.(*VkQueueSubmit)
	if !ok {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Overdraw change marked for non-VkQueueSubmit")})
		out.MutateAndWrite(ctx, id, cmd)
		return
	}

	lastRenderPassArgs, lastRenderPassIdx, err :=
		s.getLastRenderPass(ctx, gs, st, submit)
	if err != nil {
		res(nil, &service.ErrDataUnavailable{
			Reason: messages.ErrMessage(fmt.Sprintf(
				"Could not get overdraw: %v", err))})
		out.MutateAndWrite(ctx, id, cmd)
		return
	}

	if lastRenderPassArgs.IsNil() {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("No render pass in queue submit")})
		out.MutateAndWrite(ctx, id, cmd)
		return
	}

	cb := CommandBuilder{Thread: submit.Thread(), Arena: gs.Arena}
	image, err := s.rewriteQueueSubmit(ctx, cb, gs, st, arena, submit,
		lastRenderPassArgs, lastRenderPassIdx, id,
		mustAllocData, addCleanup, out)
	if err != nil {
		res(nil, &service.ErrDataUnavailable{
			Reason: messages.ErrMessage(fmt.Sprintf(
				"Could not get overdraw: %v", err))})
		out.MutateAndWrite(ctx, id, cmd)
		return
	}
	postImageData(ctx, cb, gs,
		st.Images().Get(image.handle),
		image.format,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT,
		0,
		0,
		image.width,
		image.height,
		image.width,
		image.height,
		out,
		res,
	)
}

func (*stencilOverdraw) getLastRenderPass(ctx context.Context,
	gs *api.GlobalState,
	st *State,
	submit *VkQueueSubmit,
) (VkCmdBeginRenderPassArgsʳ, api.SubCmdIdx, error) {
	lastRenderPassArgs := NilVkCmdBeginRenderPassArgsʳ
	var lastRenderPassIdx api.SubCmdIdx
	submit.Extras().Observations().ApplyReads(gs.Memory.ApplicationPool())
	submitInfos := submit.PSubmits().Slice(0, uint64(submit.SubmitCount()),
		gs.MemoryLayout).MustRead(ctx, submit, gs, nil)
	for i, si := range submitInfos {
		cmdBuffers := si.PCommandBuffers().Slice(0, uint64(si.CommandBufferCount()),
			gs.MemoryLayout).MustRead(ctx, submit, gs, nil)
		for j, buf := range cmdBuffers {
			cb, ok := st.CommandBuffers().Lookup(buf)
			if !ok {
				return lastRenderPassArgs, lastRenderPassIdx,
					fmt.Errorf("Invalid command buffer %v", buf)
			}
			// vkCmdBeginRenderPass can only be in a primary command buffer,
			// so we don't need to check secondary command buffers
			for k := 0; k < cb.CommandReferences().Len(); k++ {
				cr := cb.CommandReferences().Get(uint32(k))
				if cr.Type() == CommandType_cmd_vkCmdBeginRenderPass {
					lastRenderPassArgs = cb.BufferCommands().
						VkCmdBeginRenderPass().
						Get(cr.MapIndex())
					lastRenderPassIdx = api.SubCmdIdx{
						uint64(i), uint64(j), uint64(k)}
				}
			}
		}
	}

	return lastRenderPassArgs, lastRenderPassIdx, nil
}

type stencilImage struct {
	handle VkImage
	format VkFormat
	width  uint32
	height uint32
}

type renderInfo struct {
	renderPass  VkRenderPass
	depthIdx    uint32
	framebuffer VkFramebuffer
	image       stencilImage
}

func (s *stencilOverdraw) createNewRenderPassFramebuffer(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	oldRenderPass VkRenderPass,
	oldFramebuffer VkFramebuffer,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) (renderInfo, error) {
	oldRpInfo, ok := st.RenderPasses().Lookup(oldRenderPass)
	if !ok {
		return renderInfo{},
			fmt.Errorf("Invalid renderpass %v",
				oldRenderPass)
	}

	oldFbInfo, ok := st.Framebuffers().Lookup(oldFramebuffer)
	if !ok {
		return renderInfo{},
			fmt.Errorf("Invalid framebuffer %v",
				oldFramebuffer)
	}

	attachDesc, depthIdx, err :=
		s.getStencilAttachmentDescription(st, a, oldRpInfo)
	if err != nil {
		return renderInfo{}, err
	}

	width, height := oldFbInfo.Width(), oldFbInfo.Height()
	device := oldFbInfo.Device()
	image, err := s.createImage(ctx, cb, st, a, device, attachDesc.Fmt(),
		width, height, alloc, addCleanup, out)
	if err != nil {
		return renderInfo{}, err
	}

	imageView := s.createImageView(ctx, cb, st, a, device,
		image.handle, alloc, addCleanup, out)

	renderPass := s.createRenderPass(ctx, cb, st, a, device, oldRpInfo,
		attachDesc, alloc, addCleanup, out)
	framebuffer := s.createFramebuffer(ctx, cb, st, a, device, oldFbInfo,
		renderPass, imageView, alloc, addCleanup, out)

	return renderInfo{renderPass, depthIdx, framebuffer, image}, nil
}

func (s *stencilOverdraw) getStencilAttachmentDescription(st *State,
	a arena.Arena,
	rpInfo RenderPassObjectʳ,
) (VkAttachmentDescription, uint32, error) {

	depthDesc, idx, err := s.getDepthAttachment(a, rpInfo)
	if err != nil {
		return NilVkAttachmentDescription, idx, err
	}

	// Clone it, but with a stencil-friendly format
	var stencilDesc VkAttachmentDescription
	if idx != ^uint32(0) {
		stencilDesc = depthDesc.Clone(a, api.CloneContext{})

		format, err := s.depthToStencilFormat(depthDesc.Fmt())
		if err != nil {
			return NilVkAttachmentDescription, idx, err
		}
		stencilDesc.SetFmt(format)
		if stencilDesc.FinalLayout() !=
			VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL {
			stencilDesc.SetFinalLayout(
				VkImageLayout_VK_IMAGE_LAYOUT_GENERAL)
		}
	} else {
		stencilDesc = MakeVkAttachmentDescription(a)
		// Use this format because it is the most commonly supported.
		// Ideally we would be able to do
		// VkGetPhysicalDeviceImageFormatProperties to determine what
		// we can use, but for now assume this is available.
		stencilDesc.SetFmt(VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT)
		stencilDesc.SetSamples(VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT)
		stencilDesc.SetLoadOp(VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE)
		stencilDesc.SetStoreOp(VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE)
		stencilDesc.SetInitialLayout(VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED)
		stencilDesc.SetFinalLayout(
			VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL)
	}

	return stencilDesc, idx, nil
}

// TODO: see if we can use the existing depth attachment in place
func (s *stencilOverdraw) getDepthAttachment(a arena.Arena,
	rpInfo RenderPassObjectʳ,
) (VkAttachmentDescription, uint32, error) {
	if rpInfo.SubpassDescriptions().Len() == 0 {
		return NilVkAttachmentDescription, 0,
			fmt.Errorf("RenderPass %v has no subpasses",
				rpInfo.VulkanHandle())
	}
	// depth attachment: don't support them not all using the same one for now
	attachment0 := rpInfo.SubpassDescriptions().Get(0).DepthStencilAttachment()
	for i := uint32(1); i < uint32(rpInfo.SubpassDescriptions().Len()); i++ {
		attachment := rpInfo.SubpassDescriptions().Get(i).DepthStencilAttachment()
		var match bool
		if attachment0.IsNil() {
			match = attachment.IsNil()
		} else {
			match = !attachment.IsNil() &&
				attachment0.Attachment() == attachment.Attachment()
		}
		if !match {
			return NilVkAttachmentDescription, 0, fmt.Errorf(
				"The subpasses don't have matching depth attachments")
		}
	}
	if attachment0.IsNil() {
		return NilVkAttachmentDescription, ^uint32(0), nil
	}

	attachmentDesc, ok := rpInfo.AttachmentDescriptions().Lookup(
		attachment0.Attachment(),
	)
	if !ok {
		return NilVkAttachmentDescription, 0,
			fmt.Errorf("Invalid depth attachment")
	}

	return attachmentDesc, attachment0.Attachment(), nil
}

func (*stencilOverdraw) depthToStencilFormat(depthFormat VkFormat) (VkFormat, error) {
	switch depthFormat {
	case VkFormat_VK_FORMAT_D16_UNORM:
		return VkFormat_VK_FORMAT_D16_UNORM_S8_UINT, nil
	case VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32:
		return VkFormat_VK_FORMAT_D24_UNORM_S8_UINT, nil
	case VkFormat_VK_FORMAT_D32_SFLOAT:
		return VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT, nil

	case VkFormat_VK_FORMAT_D16_UNORM_S8_UINT:
		fallthrough
	case VkFormat_VK_FORMAT_D24_UNORM_S8_UINT:
		fallthrough
	case VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
		return depthFormat, nil
	default:
		return 0, fmt.Errorf("Unrecognized depth format %v",
			depthFormat)
	}
}

func (*stencilOverdraw) createImage(ctx context.Context,
	cb CommandBuilder,
	st *State,
	a arena.Arena,
	device VkDevice,
	format VkFormat,
	width uint32,
	height uint32,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) (stencilImage, error) {
	imageCreateInfo := NewVkImageCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO, // sType
		0, // pNext
		0, // flags
		VkImageType_VK_IMAGE_TYPE_2D, // imageType
		format, // format
		NewVkExtent3D(a, width, height, 1), // extent
		1, // mipLevels
		1, // arrayLevels
		VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT, // samples
		VkImageTiling_VK_IMAGE_TILING_OPTIMAL,       // tiling
		VkImageUsageFlags( // usage
			VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT|
				VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_SRC_BIT),
		VkSharingMode_VK_SHARING_MODE_EXCLUSIVE, // sharingMode
		0, // queueFamilyIndexCount
		0, // pQueueFamilyIndices
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED, // initialLayout
	)
	imageCreateInfoData := alloc(imageCreateInfo)

	image := VkImage(newUnusedID(false, func(id uint64) bool {
		return st.Images().Contains(VkImage(id))
	}))
	imageData := alloc(image)
	writeEach(ctx, out,
		cb.VkCreateImage(device,
			imageCreateInfoData.Ptr(),
			memory.Nullptr,
			imageData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(imageCreateInfoData.Data()).AddWrite(imageData.Data()))

	// The physical device memory properties are used to find the correct
	// memory type index and allocate proper memory for our stencil image.
	deviceInfo, ok := st.Devices().Lookup(device)
	if !ok {
		return stencilImage{}, fmt.Errorf("Invalid device %v",
			device)
	}
	physicalDeviceInfo, ok := st.PhysicalDevices().Lookup(
		deviceInfo.PhysicalDevice())
	if !ok {
		return stencilImage{}, fmt.Errorf("Invalid physical device %v",
			deviceInfo.PhysicalDevice())
	}
	physicalDeviceMemoryPropertiesData := alloc(physicalDeviceInfo.MemoryProperties())

	imageMemory := VkDeviceMemory(newUnusedID(false, func(id uint64) bool {
		return st.DeviceMemories().Contains(VkDeviceMemory(id))
	}))
	imageMemoryData := alloc(imageMemory)
	writeEach(ctx, out,
		cb.ReplayAllocateImageMemory(
			device,
			physicalDeviceMemoryPropertiesData.Ptr(),
			image,
			imageMemoryData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			physicalDeviceMemoryPropertiesData.Data(),
		).AddWrite(
			imageMemoryData.Data(),
		),
		cb.VkBindImageMemory(
			device, image, imageMemory, VkDeviceSize(0),
			VkResult_VK_SUCCESS))

	addCleanup(func() {
		writeEach(ctx, out,
			cb.VkDestroyImage(
				device,
				image,
				memory.Nullptr),
			cb.VkFreeMemory(
				device,
				imageMemory,
				memory.Nullptr),
		)
	})

	return stencilImage{image, format, width, height}, nil
}

func (*stencilOverdraw) createImageView(ctx context.Context,
	cb CommandBuilder,
	st *State,
	a arena.Arena,
	device VkDevice,
	image VkImage,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) VkImageView {
	createInfo := NewVkImageViewCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO, // sType
		0,     // pNext
		0,     // flags
		image, // image
		VkImageViewType_VK_IMAGE_VIEW_TYPE_2D, // viewType
		VkFormat_VK_FORMAT_S8_UINT,            // format
		NewVkComponentMapping(a,
			VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
			VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
			VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
			VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
		), // components
		NewVkImageSubresourceRange(a,
			VkImageAspectFlags(
				VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT,
			), // aspectMask
			0, // baseMipLevel
			1, // levelCount
			0, // baseArrayLayer
			1, // layerCount
		), // subresourceRange
	)
	createInfoData := alloc(createInfo)

	imageView := VkImageView(newUnusedID(false, func(id uint64) bool {
		return st.ImageViews().Contains(VkImageView(id))
	}))
	imageViewData := alloc(imageView)

	writeEach(ctx, out,
		cb.VkCreateImageView(
			device,
			createInfoData.Ptr(),
			memory.Nullptr,
			imageViewData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			createInfoData.Data(),
		).AddWrite(
			imageViewData.Data(),
		),
	)

	addCleanup(func() {
		writeEach(ctx, out,
			cb.VkDestroyImageView(
				device,
				imageView,
				memory.Nullptr),
		)
	})
	return imageView
}

func (*stencilOverdraw) createRenderPass(ctx context.Context,
	cb CommandBuilder,
	st *State,
	a arena.Arena,
	device VkDevice,
	rpInfo RenderPassObjectʳ,
	stencilAttachment VkAttachmentDescription,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) VkRenderPass {
	allReads := []api.AllocResult{}
	allocAndRead := func(v ...interface{}) api.AllocResult {
		res := alloc(v)
		allReads = append(allReads, res)
		return res
	}

	attachments := rpInfo.AttachmentDescriptions().All()
	newAttachments := rpInfo.AttachmentDescriptions().Clone(a, api.CloneContext{})
	newAttachments.Add(uint32(newAttachments.Len()), stencilAttachment)
	newAttachmentsData, newAttachmentsLen := unpackMapCustom(allocAndRead,
		newAttachments)

	stencilAttachmentReference := NewVkAttachmentReference(a,
		uint32(len(attachments)),
		stencilAttachment.FinalLayout(),
	)
	stencilAttachmentReferencePtr := allocAndRead(stencilAttachmentReference).Ptr()

	subpasses := make([]VkSubpassDescription,
		rpInfo.SubpassDescriptions().Len())
	for idx, subpass := range rpInfo.SubpassDescriptions().All() {
		subpasses[idx] = subpassToSubpassDescription(a, subpass,
			stencilAttachmentReferencePtr, allocAndRead)
	}
	subpassesData := allocAndRead(subpasses)

	subpassDependenciesData, subpassDependenciesLen := unpackMapCustom(allocAndRead,
		rpInfo.SubpassDependencies())

	renderPassCreateInfo := NewVkRenderPassCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO, // sType
		0,                                                       // pNext
		0,                                                       // flags
		newAttachmentsLen,                                       // attachmentCount
		NewVkAttachmentDescriptionᶜᵖ(newAttachmentsData.Ptr()),  // pAttachments
		uint32(len(subpasses)),                                  // subpassCount
		NewVkSubpassDescriptionᶜᵖ(subpassesData.Ptr()),          // pSubpasses
		subpassDependenciesLen,                                  // dependencyCount
		NewVkSubpassDependencyᶜᵖ(subpassDependenciesData.Ptr()), // pDependencies
	)
	renderPassCreateInfoData := allocAndRead(renderPassCreateInfo)

	newRenderPass := VkRenderPass(newUnusedID(false, func(id uint64) bool {
		return st.RenderPasses().Contains(VkRenderPass(id))
	}))
	newRenderPassData := alloc(newRenderPass)

	createRenderPass := cb.VkCreateRenderPass(
		device,
		renderPassCreateInfoData.Ptr(),
		memory.Nullptr,
		newRenderPassData.Ptr(),
		VkResult_VK_SUCCESS,
	).AddWrite(
		newRenderPassData.Data(),
	)
	for _, read := range allReads {
		createRenderPass.AddRead(read.Data())
	}

	writeEach(ctx, out, createRenderPass)

	addCleanup(func() {
		writeEach(ctx, out,
			cb.VkDestroyRenderPass(
				device,
				newRenderPass,
				memory.Nullptr,
			))
	})

	return newRenderPass
}

func subpassToSubpassDescription(a arena.Arena,
	subpass SubpassDescription,
	attachRefPtr memory.Pointer,
	allocAndRead func(v ...interface{}) api.AllocResult,
) VkSubpassDescription {
	unpackMapMaybeEmpty := func(m interface{}) (memory.Pointer, uint32) {
		type HasLen interface {
			Len() int
		}
		if m.(HasLen).Len() > 0 {
			allocation, count := unpackMapCustom(allocAndRead, m)
			return allocation.Ptr(), count
		} else {
			return memory.Nullptr, 0
		}
	}

	inputAttachmentsPtr, inputAttachmentsCount :=
		unpackMapMaybeEmpty(subpass.InputAttachments())
	colorAttachmentsPtr, colorAttachmentsCount :=
		unpackMapMaybeEmpty(subpass.ColorAttachments())
	resolveAttachmentsPtr, _ := unpackMapMaybeEmpty(subpass.ResolveAttachments())

	preserveAttachmentsPtr, preserveAttachmentsCount :=
		unpackMapMaybeEmpty(subpass.PreserveAttachments())

	return NewVkSubpassDescription(a,
		subpass.Flags(),                                   // flags
		subpass.PipelineBindPoint(),                       // pipelineBindPoint
		inputAttachmentsCount,                             // inputAttachmentCount
		NewVkAttachmentReferenceᶜᵖ(inputAttachmentsPtr),   // pInputAttachments
		colorAttachmentsCount,                             // colorAttachmentCount
		NewVkAttachmentReferenceᶜᵖ(colorAttachmentsPtr),   // pColorAttachments
		NewVkAttachmentReferenceᶜᵖ(resolveAttachmentsPtr), // pResolveAttachments
		NewVkAttachmentReferenceᶜᵖ(attachRefPtr),          // pDepthStencilAttachment
		preserveAttachmentsCount,                          // preserveAttachmentCount
		NewU32ᶜᵖ(preserveAttachmentsPtr),                  // pPreserveAttachments
	)
}

func (*stencilOverdraw) createFramebuffer(ctx context.Context,
	cb CommandBuilder,
	st *State,
	a arena.Arena,
	device VkDevice,
	fbInfo FramebufferObjectʳ,
	renderPass VkRenderPass,
	stencilImageView VkImageView,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) VkFramebuffer {
	attachments := fbInfo.ImageAttachments().All()
	newAttachments := make([]VkImageView, len(attachments)+1)
	for idx, imageView := range attachments {
		newAttachments[idx] = imageView.VulkanHandle()
	}
	newAttachments[len(attachments)] = stencilImageView
	newAttachmentsData := alloc(newAttachments)

	createInfo := NewVkFramebufferCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO, // sType
		0,                                          // pNext
		0,                                          // flags
		renderPass,                                 // renderPass
		uint32(len(newAttachments)),                // attachmentCount
		NewVkImageViewᶜᵖ(newAttachmentsData.Ptr()), // pAttachments
		fbInfo.Width(),                             // width
		fbInfo.Height(),                            // height
		fbInfo.Layers(),                            // layers
	)
	createInfoData := alloc(createInfo)

	newFramebuffer := VkFramebuffer(newUnusedID(false, func(id uint64) bool {
		return st.Framebuffers().Contains(VkFramebuffer(id))
	}))
	newFramebufferData := alloc(newFramebuffer)

	writeEach(ctx, out,
		cb.VkCreateFramebuffer(
			device,
			createInfoData.Ptr(),
			memory.Nullptr,
			newFramebufferData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			createInfoData.Data(),
		).AddRead(
			newAttachmentsData.Data(),
		).AddWrite(
			newFramebufferData.Data(),
		),
	)

	addCleanup(func() {
		writeEach(ctx, out,
			cb.VkDestroyFramebuffer(
				device,
				newFramebuffer,
				memory.Nullptr,
			))
	})

	return newFramebuffer
}

func (s *stencilOverdraw) createGraphicsPipeline(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	device VkDevice,
	pipeline VkPipeline,
	renderPass VkRenderPass,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) (VkPipeline, error) {
	reads := []api.AllocResult{}
	allocAndRead := func(v ...interface{}) api.AllocResult {
		res := alloc(v)
		reads = append(reads, res)
		return res
	}

	createInfo, err := s.createGraphicsPipelineCreateInfo(ctx,
		cb, gs, st, a, pipeline, renderPass, alloc, allocAndRead,
		addCleanup, out)
	if err != nil {
		return 0, err
	}

	createInfoData := allocAndRead(createInfo)

	newPipeline := VkPipeline(newUnusedID(false, func(id uint64) bool {
		return st.GraphicsPipelines().Contains(VkPipeline(id))
	}))
	newPipelineData := allocAndRead(newPipeline)

	cmd := cb.VkCreateGraphicsPipelines(
		device,                // device
		0,                     // pipelineCache: VK_NULL_HANDLE
		1,                     // createInfoCount
		createInfoData.Ptr(),  // pCreateInfos
		memory.Nullptr,        // pAllocator
		newPipelineData.Ptr(), // pPipelines
		VkResult_VK_SUCCESS,   // result
	).AddRead(
		createInfoData.Data(),
	).AddWrite(
		newPipelineData.Data(),
	)

	for _, read := range reads {
		cmd.AddRead(read.Data())
	}

	writeEach(ctx, out, cmd)

	addCleanup(func() {
		writeEach(ctx, out,
			cb.VkDestroyPipeline(
				device, newPipeline, memory.Nullptr,
			))
	})

	return newPipeline, nil
}

func (s *stencilOverdraw) createGraphicsPipelineCreateInfo(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	pipeline VkPipeline,
	renderPass VkRenderPass,
	alloc func(v ...interface{}) api.AllocResult,
	allocAndRead func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) (VkGraphicsPipelineCreateInfo, error) {
	unpackMapMaybeEmpty := func(m interface{}) (memory.Pointer, uint32) {
		type HasLen interface {
			Len() int
		}
		if m.(HasLen).Len() > 0 {
			allocation, count := unpackMapCustom(allocAndRead, m)
			return allocation.Ptr(), count
		} else {
			return memory.Nullptr, 0
		}
	}

	// TODO: Recreating a lot of work from state_rebuilder, look into merging with that
	pInfo, ok := st.GraphicsPipelines().Lookup(pipeline)
	if !ok {
		return NilVkGraphicsPipelineCreateInfo,
			fmt.Errorf("Invalid graphics pipeline %v", pipeline)
	}

	shaderStagesPtr := memory.Nullptr
	shaderStagesCount := uint32(0)
	if pInfo.Stages().Len() > 0 {
		stages := pInfo.Stages().All()
		data := make([]VkPipelineShaderStageCreateInfo, len(stages))
		for idx, stage := range stages {
			module := stage.Module().VulkanHandle()
			if !st.ShaderModules().Contains(module) {
				m := s.createShaderModule(ctx, cb,
					gs, st, a, stage.Module(), alloc, addCleanup, out)
				module = m
			}
			data[idx] = NewVkPipelineShaderStageCreateInfo(a,
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO, // sType
				0,             // pNext
				0,             // flags
				stage.Stage(), // stage
				module,        // module
				NewCharᶜᵖ(allocAndRead(stage.EntryPoint()).Ptr()), // pName
				s.createSpecializationInfo(ctx, gs, a,
					stage.Specialization(),
					allocAndRead,
				), // pSpecializationInfo
			)
		}
		allocation := allocAndRead(data)
		shaderStagesPtr = allocation.Ptr()
		shaderStagesCount = uint32(len(data))
	}

	vertexInputPtr := memory.Nullptr
	{
		bindingPtr, bindingCount := unpackMapMaybeEmpty(
			pInfo.VertexInputState().BindingDescriptions())
		attributePtr, attributeCount := unpackMapMaybeEmpty(
			pInfo.VertexInputState().AttributeDescriptions())
		vertexInputPtr = allocAndRead(
			NewVkPipelineVertexInputStateCreateInfo(a,
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO, // sType
				0,            // pNext
				0,            // flags
				bindingCount, // vertexBindingDescriptionCount
				NewVkVertexInputBindingDescriptionᶜᵖ(bindingPtr), // pVertexBindingDescriptions
				attributeCount,                                   // vertexAttributeDescriptionCount
				NewVkVertexInputAttributeDescriptionᶜᵖ(attributePtr), // pVertexAttributeDescriptions
			)).Ptr()
	}

	inputAssemblyPtr := memory.Nullptr
	{
		info := pInfo.InputAssemblyState()
		inputAssemblyPtr = allocAndRead(
			NewVkPipelineInputAssemblyStateCreateInfo(a,
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO, // sType
				0,                             // pNext
				0,                             // flags
				info.Topology(),               // topology
				info.PrimitiveRestartEnable(), // primitiveRestartEnable
			)).Ptr()
	}

	tessellationPtr := memory.Nullptr
	if !pInfo.TessellationState().IsNil() {
		info := pInfo.TessellationState()
		tessellationPtr = allocAndRead(
			NewVkPipelineTessellationStateCreateInfo(a,
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_TESSELLATION_STATE_CREATE_INFO, // sType
				0, // pNext
				0, // flags
				info.PatchControlPoints(), // patchControlPoints
			)).Ptr()
	}

	viewportPtr := memory.Nullptr
	if !pInfo.ViewportState().IsNil() {
		info := pInfo.ViewportState()
		viewPtr, viewCount := unpackMapMaybeEmpty(info.Viewports())
		scissorPtr, scissorCount := unpackMapMaybeEmpty(info.Scissors())
		viewportPtr = allocAndRead(
			NewVkPipelineViewportStateCreateInfo(a,
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO, // sType
				0,                         // pNext
				0,                         // flags
				viewCount,                 // viewportCount
				NewVkViewportᶜᵖ(viewPtr),  // pViewports
				scissorCount,              // scissorCount
				NewVkRect2Dᶜᵖ(scissorPtr), // pScissors
			)).Ptr()
	}

	rasterizationPtr := memory.Nullptr
	{
		info := pInfo.RasterizationState()
		rasterizationPtr = allocAndRead(
			NewVkPipelineRasterizationStateCreateInfo(a,
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO, // sType
				0, // pNext
				0, // flags
				info.DepthClampEnable(),
				info.RasterizerDiscardEnable(),
				info.PolygonMode(),
				info.CullMode(),
				info.FrontFace(),
				info.DepthBiasEnable(),
				info.DepthBiasConstantFactor(),
				info.DepthBiasClamp(),
				info.DepthBiasSlopeFactor(),
				info.LineWidth(),
			)).Ptr()
	}

	multisamplePtr := memory.Nullptr
	if !pInfo.MultisampleState().IsNil() {
		info := pInfo.MultisampleState()
		sampleMaskPtr, _ := unpackMapMaybeEmpty(info.SampleMask())
		multisamplePtr = allocAndRead(
			NewVkPipelineMultisampleStateCreateInfo(a,
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO, // sType
				0, // pNext
				0, // flags
				info.RasterizationSamples(),
				info.SampleShadingEnable(),
				info.MinSampleShading(),
				NewVkSampleMaskᶜᵖ(sampleMaskPtr), // pSampleMask
				info.AlphaToCoverageEnable(),
				info.AlphaToOneEnable(),
			)).Ptr()
	}

	var depthStencilPtr memory.Pointer
	{
		// FIXME: work with existing depth buffer
		stencilOp := NewVkStencilOpState(a,
			0, // failOp
			VkStencilOp_VK_STENCIL_OP_INCREMENT_AND_CLAMP, // passOp
			0, // depthFailOp
			VkCompareOp_VK_COMPARE_OP_ALWAYS, // compareOp
			255, // compareMask
			255, // writeMask
			0,   // reference
		)
		state := MakeVkPipelineDepthStencilStateCreateInfo(a)
		state.SetSType(
			VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO)
		state.SetStencilTestEnable(1)
		state.SetFront(stencilOp)
		state.SetBack(stencilOp)
		if !pInfo.DepthState().IsNil() {
			info := pInfo.DepthState()
			if info.StencilTestEnable() != 0 {
				return NilVkGraphicsPipelineCreateInfo,
					fmt.Errorf("The stencil buffer is already in use")
			}

			state.SetDepthTestEnable(info.DepthTestEnable())
			state.SetDepthWriteEnable(info.DepthWriteEnable())
			state.SetDepthCompareOp(info.DepthCompareOp())
			state.SetDepthBoundsTestEnable(info.DepthBoundsTestEnable())
			state.SetMinDepthBounds(info.MinDepthBounds())
			state.SetMaxDepthBounds(info.MaxDepthBounds())
		}
		depthStencilPtr = allocAndRead(state).Ptr()
	}

	colorBlendPtr := memory.Nullptr
	if !pInfo.ColorBlendState().IsNil() {
		info := pInfo.ColorBlendState()
		attachmentPtr, attachmentCount := unpackMapMaybeEmpty(info.Attachments())
		colorBlendPtr = allocAndRead(
			NewVkPipelineColorBlendStateCreateInfo(a,
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO, // sType
				0, // pNext
				0, // flags
				info.LogicOpEnable(),
				info.LogicOp(),
				attachmentCount, // attachmentCount
				NewVkPipelineColorBlendAttachmentStateᶜᵖ(
					attachmentPtr), // pAttachments
				info.BlendConstants(),
			)).Ptr()
	}

	dynamicPtr := memory.Nullptr
	if !pInfo.DynamicState().IsNil() {
		info := pInfo.DynamicState()
		statesPtr, statesCount := unpackMapMaybeEmpty(info.DynamicStates())
		dynamicPtr = allocAndRead(
			NewVkPipelineDynamicStateCreateInfo(a,
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO, // sType
				0,                              // pNext
				0,                              // flags
				statesCount,                    // dynamicStateCount
				NewVkDynamicStateᶜᵖ(statesPtr), // pDynamicStates
			)).Ptr()
	}

	// TODO: determine if basePipelineHandle is an issue

	return NewVkGraphicsPipelineCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO, // sType
		0,                                                             // pNext
		0,                                                             // flags
		shaderStagesCount,                                             // stageCount
		NewVkPipelineShaderStageCreateInfoᶜᵖ(shaderStagesPtr),         // pStages
		NewVkPipelineVertexInputStateCreateInfoᶜᵖ(vertexInputPtr),     // pVertexInputState
		NewVkPipelineInputAssemblyStateCreateInfoᶜᵖ(inputAssemblyPtr), // pInputAssemblyState
		NewVkPipelineTessellationStateCreateInfoᶜᵖ(tessellationPtr),   // pTessellationState
		NewVkPipelineViewportStateCreateInfoᶜᵖ(viewportPtr),           // pViewportState
		NewVkPipelineRasterizationStateCreateInfoᶜᵖ(rasterizationPtr), // pRasterizationState
		NewVkPipelineMultisampleStateCreateInfoᶜᵖ(multisamplePtr),     // pMultisampleState
		NewVkPipelineDepthStencilStateCreateInfoᶜᵖ(depthStencilPtr),   // pDepthStencilState
		NewVkPipelineColorBlendStateCreateInfoᶜᵖ(colorBlendPtr),       // pColorBlendState
		NewVkPipelineDynamicStateCreateInfoᶜᵖ(dynamicPtr),             // pDynamicState
		pInfo.Layout().VulkanHandle(),                                 // layout
		renderPass,                                                    // renderPass
		pInfo.Subpass(),                                               // subpass
		pInfo.BasePipeline(),                                          // basePipelineHandle
		pInfo.BasePipelineIndex(),                                     // basePipelineIndex
	), nil
}

func (*stencilOverdraw) createShaderModule(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	info ShaderModuleObjectʳ,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) VkShaderModule {
	module := VkShaderModule(newUnusedID(false, func(id uint64) bool {
		return st.ShaderModules().Contains(VkShaderModule(id))
	}))
	moduleData := alloc(module)

	words := info.Words().MustRead(ctx, nil, gs, nil)
	wordsData := alloc(words)
	createInfoData := alloc(NewVkShaderModuleCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO, // sType
		0, // pNext
		0, // flags
		memory.Size(len(words)*4),
		NewU32ᶜᵖ(wordsData.Ptr()),
	))

	writeEach(ctx, out, cb.VkCreateShaderModule(
		info.Device(),
		createInfoData.Ptr(),
		memory.Nullptr,
		moduleData.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		createInfoData.Data(),
	).AddRead(
		wordsData.Data(),
	).AddWrite(
		moduleData.Data(),
	))

	addCleanup(func() {
		writeEach(ctx, out, cb.VkDestroyShaderModule(
			info.Device(),
			module,
			memory.Nullptr,
		))
	})

	return module
}

func (*stencilOverdraw) createSpecializationInfo(ctx context.Context,
	gs *api.GlobalState,
	a arena.Arena,
	info SpecializationInfoʳ,
	allocAndRead func(v ...interface{}) api.AllocResult,
) VkSpecializationInfoᶜᵖ {
	if info.IsNil() {
		return 0
	}
	mapEntries, mapEntryCount := unpackMapCustom(allocAndRead, info.Specializations().All())
	data := info.Data().MustRead(ctx, nil, gs, nil)
	return NewVkSpecializationInfoᶜᵖ(allocAndRead(
		NewVkSpecializationInfo(a,
			mapEntryCount,                                   // mapEntryCount
			NewVkSpecializationMapEntryᶜᵖ(mapEntries.Ptr()), // pMapEntries
			memory.Size(len(data)),                          // dataSize,
			NewVoidᶜᵖ(allocAndRead(data).Ptr()),             // pData
		)).Ptr())
}

// FIXME: handle secondary command buffers
func (s *stencilOverdraw) createCommandBuffer(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	cmdBuffer VkCommandBuffer,
	renderInfo renderInfo,
	rpStartIdx uint64,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) (VkCommandBuffer, error) {
	// TODO copy old depth data over if theres an original depth buffer in keep mode

	bInfo, ok := st.CommandBuffers().Lookup(cmdBuffer)
	if !ok {
		return 0, fmt.Errorf("Invalid command buffer %v", cmdBuffer)
	}
	device := bInfo.Device()

	newCmdBuffer, cmds, cleanup := allocateNewCmdBufFromExistingOneAndBegin(
		ctx, cb, cmdBuffer, gs)
	writeEach(ctx, out, cmds...)
	for _, f := range cleanup {
		f()
	}

	pipelines := map[VkPipeline]VkPipeline{}
	secCmdBuffers := map[VkCommandBuffer]VkCommandBuffer{}

	rpEnded := false
	for i := 0; i < bInfo.CommandReferences().Len(); i++ {
		cr := bInfo.CommandReferences().Get(uint32(i))
		args := GetCommandArgs(ctx, cr, st)
		if uint64(i) >= rpStartIdx && !rpEnded {
			switch ar := args.(type) {
			case VkCmdBeginRenderPassArgsʳ:
				newArgs := ar.Clone(a, api.CloneContext{})
				newArgs.SetRenderPass(renderInfo.renderPass)
				newArgs.SetFramebuffer(renderInfo.framebuffer)

				clearCount := uint32(newArgs.ClearValues().Len())
				newClear := NewU32ː4ᵃ(a)

				if renderInfo.depthIdx != ^uint32(0) {
					newClear.Set(0, newArgs.
						ClearValues().
						Get(renderInfo.depthIdx).
						Color().
						Uint32().
						Get(0))
				}
				// 0 initialize the stencil buffer
				newArgs.ClearValues().Add(clearCount,
					// Use VkClearColorValue instead of
					// VkClearDepthValue because it doesn't
					// seem like the union is set up in the
					// API DSL
					NewVkClearValue(a, NewVkClearColorValue(a,
						newClear)))
				args = newArgs
			case VkCmdEndRenderPassArgsʳ:
				rpEnded = true
			case VkCmdBindPipelineArgsʳ:
				newArgs := ar
				if ar.PipelineBindPoint() ==
					VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS {
					newArgs = ar.Clone(a, api.CloneContext{})

					pipe := ar.Pipeline()
					newPipe, ok := pipelines[pipe]
					if !ok {
						var err error
						newPipe, err = s.createGraphicsPipeline(ctx, cb, gs, st,
							a, device, pipe, renderInfo.renderPass, alloc,
							addCleanup, out)
						if err != nil {
							return 0, err
						}
						pipelines[pipe] = newPipe
					}
					newArgs.SetPipeline(newPipe)
				}
				args = newArgs
			case VkCmdExecuteCommandsArgsʳ:
				newArgs := ar
				for i := uint32(0); i < uint32(ar.CommandBuffers().Len()); i++ {
					cmdbuf := ar.CommandBuffers().Get(i)
					newCmdbuf, ok := secCmdBuffers[cmdbuf]
					if !ok {
						var err error
						newCmdbuf, err = s.createCommandBuffer(ctx,
							cb, gs, st, a, cmdbuf, renderInfo,
							0, alloc, addCleanup, out)
						if err != nil {
							return 0, err
						}
						secCmdBuffers[cmdbuf] = newCmdbuf
					}
					newArgs.CommandBuffers().Add(i, newCmdbuf)
				}
				args = newArgs
			}
		}
		cleanup, cmd, _ := AddCommand(ctx, cb, newCmdBuffer, gs,
			gs, args)

		writeEach(ctx, out, cmd)
		cleanup()
	}
	writeEach(ctx, out,
		cb.VkEndCommandBuffer(newCmdBuffer, VkResult_VK_SUCCESS))

	return newCmdBuffer, nil
}

func (s *stencilOverdraw) rewriteQueueSubmit(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	submit *VkQueueSubmit,
	rpBeginArgs VkCmdBeginRenderPassArgsʳ,
	rpBeginIdx api.SubCmdIdx,
	cmdId api.CmdID,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) (stencilImage, error) {
	// TODO: check if we're allowed to modify the command directly, since
	// we won't be submitting the original one.  Need to deep clone all of
	// the submit info so we can mark it as reads.
	reads := []api.AllocResult{}
	allocAndRead := func(v ...interface{}) api.AllocResult {
		res := alloc(v)
		reads = append(reads, res)
		return res
	}

	renderInfo, err := s.createNewRenderPassFramebuffer(ctx, cb, gs, st,
		a, rpBeginArgs.RenderPass(), rpBeginArgs.Framebuffer(),
		alloc, addCleanup, out)
	if err != nil {
		return stencilImage{}, err
	}

	l := gs.MemoryLayout
	submit.Extras().Observations().ApplyReads(gs.Memory.ApplicationPool())
	submitCount := submit.SubmitCount()
	submitInfos := submit.PSubmits().Slice(0, uint64(submitCount), l).MustRead(
		ctx, submit, gs, nil)

	newSubmitInfos := make([]VkSubmitInfo, submitCount)
	for i := uint32(0); i < submitCount; i++ {
		si := submitInfos[i]

		waitSemPtr := memory.Nullptr
		waitDstStagePtr := memory.Nullptr
		if count := uint64(si.WaitSemaphoreCount()); count > 0 {
			waitSemPtr = allocAndRead(si.PWaitSemaphores().
				Slice(0, count, l).
				MustRead(ctx, submit, gs, nil)).Ptr()
			waitDstStagePtr = allocAndRead(si.PWaitDstStageMask().
				Slice(0, count, l).
				MustRead(ctx, submit, gs, nil)).Ptr()
		}

		signalSemPtr := memory.Nullptr
		if count := uint64(si.SignalSemaphoreCount()); count > 0 {
			signalSemPtr = allocAndRead(si.PSignalSemaphores().
				Slice(0, count, l).
				MustRead(ctx, submit, gs, nil)).Ptr()
		}

		cmdBufferPtr := memory.Nullptr
		if count := uint64(si.CommandBufferCount()); count > 0 {
			cmdBuffers := si.PCommandBuffers().
				Slice(0, count, l).
				MustRead(ctx, submit, gs, nil)
			if uint64(i) == rpBeginIdx[0] {
				newCommandBuffer, err :=
					s.createCommandBuffer(ctx, cb, gs, st, a,
						cmdBuffers[rpBeginIdx[1]],
						renderInfo,
						rpBeginIdx[2],
						alloc, addCleanup, out)
				if err != nil {
					return stencilImage{}, err
				}
				cmdBuffers[rpBeginIdx[1]] = newCommandBuffer
			}
			cmdBufferPtr = allocAndRead(cmdBuffers).Ptr()
		}

		newSubmitInfos[i] = NewVkSubmitInfo(a,
			VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO,
			0, // pNext
			si.WaitSemaphoreCount(),                    // waitSemaphoreCount
			NewVkSemaphoreᶜᵖ(waitSemPtr),               // pWaitSemaphores
			NewVkPipelineStageFlagsᶜᵖ(waitDstStagePtr), // pWaitDstStageMask
			si.CommandBufferCount(),                    // commandBufferCount
			NewVkCommandBufferᶜᵖ(cmdBufferPtr),         // pCommandBuffers
			si.SignalSemaphoreCount(),                  // signalSemaphoreCount
			NewVkSemaphoreᶜᵖ(signalSemPtr),             // pSignalSemaphores
		)
	}
	submitInfoPtr := allocAndRead(newSubmitInfos).Ptr()

	// TODO: check if we need to add synchronization here
	cmd := cb.VkQueueSubmit(
		submit.Queue(),
		submit.SubmitCount(),
		submitInfoPtr,
		submit.Fence(),
		VkResult_VK_SUCCESS,
	)
	for _, read := range reads {
		cmd.AddRead(read.Data())
	}

	out.MutateAndWrite(ctx, cmdId, cmd)
	return renderInfo.image, nil
}

func (s *stencilOverdraw) Flush(ctx context.Context, output transform.Writer) {}
