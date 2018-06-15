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
	capture *capture.Capture
}

func newStencilOverdraw(ctx context.Context, capt *path.Capture) *stencilOverdraw {
	cr, err := capture.ResolveFromPath(ctx, capt)
	if err != nil {
		return nil
	}
	return &stencilOverdraw{
		rewrite: map[api.CmdID]replay.Result{},
		capture: cr,
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
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("No last queue submission")})
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

	st := out.State()
	vkState := GetState(st)
	arena := st.Arena

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
		res := st.AllocDataOrPanic(ctx, v...)
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

	lastRenderPassArgs, lastRenderPassIdx := s.getLastRenderPass(ctx, st, vkState, submit)
	if lastRenderPassArgs.IsNil() {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("No render pass in queue submit")})
		out.MutateAndWrite(ctx, id, cmd)
		return
	}

	renderPass := lastRenderPassArgs.RenderPass()
	framebuffer := lastRenderPassArgs.Framebuffer()

	submit.Extras().Observations().ApplyReads(st.Memory.ApplicationPool())
	device := vkState.Queues().Get(submit.Queue()).Device()
	cb := CommandBuilder{Thread: submit.Thread(), Arena: st.Arena}

	// Create the image to write the results to
	stencilImage, width, height := s.createImage(ctx, cb, vkState, arena,
		device, framebuffer, mustAllocData, addCleanup, out)
	stencilImageView := s.createImageView(ctx, cb, vkState, arena,
		device, stencilImage, mustAllocData, addCleanup, out)
	newRenderPass, err := s.createRenderPass(ctx, cb, vkState, arena,
		device, renderPass, mustAllocData, addCleanup, out)
	if err != nil {
		res(nil, err)
		out.MutateAndWrite(ctx, id, cmd)
		return
	}
	newFramebuffer := s.createFramebuffer(ctx, cb, vkState, arena,
		device, framebuffer, newRenderPass, stencilImageView, mustAllocData,
		addCleanup, out)

	if err := s.rewriteQueueSubmit(ctx, cb, st, vkState, arena, submit,
		device, newRenderPass, newFramebuffer, lastRenderPassIdx, id,
		mustAllocData, addCleanup, out); err != nil {

		res(nil, err)
		out.MutateAndWrite(ctx, id, cmd)
		return
	}
	postImageData(ctx, cb, st,
		vkState.Images().Get(stencilImage),
		// FIXME other formats
		VkFormat_VK_FORMAT_S8_UINT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT,
		0,
		0,
		width,
		height,
		width,
		height,
		out,
		res,
	)
}

func (*stencilOverdraw) getLastRenderPass(ctx context.Context,
	gs *api.GlobalState,
	st *State,
	submit *VkQueueSubmit,
) (VkCmdBeginRenderPassArgsʳ, api.SubCmdIdx) {
	lastRenderPassArgs := NilVkCmdBeginRenderPassArgsʳ
	var lastRenderPassIdx api.SubCmdIdx
	submit.Extras().Observations().ApplyReads(gs.Memory.ApplicationPool())
	submitInfos := submit.PSubmits().Slice(0, uint64(submit.SubmitCount()),
		gs.MemoryLayout).MustRead(ctx, submit, gs, nil)
	for i, si := range submitInfos {
		cmdBuffers := si.PCommandBuffers().Slice(0, uint64(si.CommandBufferCount()),
			gs.MemoryLayout).MustRead(ctx, submit, gs, nil)
		for j, buf := range cmdBuffers {
			// FIXME: lookups without existence checks
			cb := st.CommandBuffers().Get(buf)
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

	return lastRenderPassArgs, lastRenderPassIdx
}

func (*stencilOverdraw) createImage(ctx context.Context,
	cb CommandBuilder,
	st *State,
	a arena.Arena,
	device VkDevice,
	framebuffer VkFramebuffer,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) (VkImage, uint32, uint32) {
	framebufferData := st.Framebuffers().Get(framebuffer)
	// TODO: figure out how MSAA interacts here
	width, height := framebufferData.Width(), framebufferData.Height()

	format := VkFormat_VK_FORMAT_S8_UINT // FIXME handle depth combinations
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
	physicalDevice := st.PhysicalDevices().Get(
		st.Devices().Get(device).PhysicalDevice())
	physicalDeviceMemoryPropertiesData := alloc(physicalDevice.MemoryProperties())

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

	return image, width, height
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
	renderPass VkRenderPass,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) (VkRenderPass, error) {
	rpInfo := st.RenderPasses().Get(renderPass)

	attachments := rpInfo.AttachmentDescriptions().All()
	newAttachments := rpInfo.AttachmentDescriptions().Clone(a, api.CloneContext{})
	// FIXME: handle merging with depth attachment
	newAttachments.Add(uint32(newAttachments.Len()), NewVkAttachmentDescription(a,
		0, // flags
		VkFormat_VK_FORMAT_S8_UINT,                                     // format
		VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT,                    // samples
		VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE,             // loadOp
		VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE,           // storeOp
		VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_CLEAR,                 // stencilLoadOp
		VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE,               // stencilStoreOp
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,                        // initialLayout
		VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL, // finalLayout
	))
	newAttachmentsData, newAttachmentsLen := unpackMapCustom(alloc, newAttachments)

	subpasses := make([]VkSubpassDescription,
		rpInfo.SubpassDescriptions().Len())
	allReads := []api.AllocResult{}
	for idx, subpass := range rpInfo.SubpassDescriptions().All() {
		if !subpass.DepthStencilAttachment().IsNil() {
			return 0, &service.ErrDataUnavailable{Reason: messages.ErrMessage(
				"Pre-existing depth/stencil attachments not supported")}
		}
		subpassDesc, reads := subpassToSubpassDescription(a, subpass,
			uint32(len(attachments)), alloc)
		subpasses[idx] = subpassDesc
		allReads = append(allReads, reads...)
	}
	subpassesData := alloc(subpasses)

	subpassDependenciesData, subpassDependenciesLen := unpackMapCustom(alloc,
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
	renderPassCreateInfoData := alloc(renderPassCreateInfo)

	newRenderPass := VkRenderPass(newUnusedID(false, func(id uint64) bool {
		return st.RenderPasses().Contains(VkRenderPass(id))
	}))
	newRenderPassData := alloc(newRenderPass)

	allReads = append(allReads, newAttachmentsData, subpassesData,
		subpassDependenciesData, renderPassCreateInfoData)

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

	return newRenderPass, nil
}

func subpassToSubpassDescription(a arena.Arena,
	subpass SubpassDescription,
	stencilIdx uint32,
	alloc func(v ...interface{}) api.AllocResult,
) (VkSubpassDescription, []api.AllocResult) {
	stencilAttachment := NewVkAttachmentReference(a,
		stencilIdx,
		VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
	)
	stencilAttachmentData := alloc(stencilAttachment)
	stencilAttachmentPtr := NewVkAttachmentReferenceᶜᵖ(stencilAttachmentData.Ptr())
	allocs := []api.AllocResult{stencilAttachmentData}

	getPtr := func(refs U32ːVkAttachmentReferenceᵐ) (VkAttachmentReferenceᶜᵖ, uint32) {
		ptr := memory.Nullptr
		num := uint32(0)
		if refs.Len() > 0 {
			allocation, count := unpackMapCustom(alloc, refs)
			ptr = allocation.Ptr()
			num = count
			allocs = append(allocs, allocation)
		}
		return NewVkAttachmentReferenceᶜᵖ(ptr), num
	}
	inputAttachmentsPtr, inputAttachmentsCount := getPtr(subpass.InputAttachments())
	colorAttachmentsPtr, colorAttachmentsCount := getPtr(subpass.ColorAttachments())
	resolveAttachmentsPtr, _ := getPtr(subpass.ResolveAttachments())

	preserveAttachmentsPtr := U32ᶜᵖ(0)
	preserveAttachmentsCount := uint32(0)
	if subpass.PreserveAttachments().Len() > 0 {
		allocation, count := unpackMapCustom(alloc, subpass.PreserveAttachments().Len())
		preserveAttachmentsPtr = NewU32ᶜᵖ(allocation.Ptr())
		preserveAttachmentsCount = count
		allocs = append(allocs, allocation)
	}

	return NewVkSubpassDescription(a,
		subpass.Flags(),             // flags
		subpass.PipelineBindPoint(), // pipelineBindPoint
		inputAttachmentsCount,       // inputAttachmentCount
		inputAttachmentsPtr,         // pInputAttachments
		colorAttachmentsCount,       // colorAttachmentCount
		colorAttachmentsPtr,         // pColorAttachments
		resolveAttachmentsPtr,       // pResolveAttachments
		stencilAttachmentPtr,        // pDepthStencilAttachment
		preserveAttachmentsCount,    // preserveAttachmentCount
		preserveAttachmentsPtr,      // pPreserveAttachments
	), allocs
}

func (*stencilOverdraw) createFramebuffer(ctx context.Context,
	cb CommandBuilder,
	st *State,
	a arena.Arena,
	device VkDevice,
	framebuffer VkFramebuffer,
	renderPass VkRenderPass,
	stencilImageView VkImageView,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) VkFramebuffer {
	fbInfo := st.Framebuffers().Get(framebuffer)

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

func (s *stencilOverdraw) createGraphicsPipelines(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	device VkDevice,
	pipelines []VkPipeline,
	renderPass VkRenderPass,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) ([]VkPipeline, error) {
	reads := []api.AllocResult{}
	allocAndRead := func(v ...interface{}) api.AllocResult {
		res := alloc(v)
		reads = append(reads, res)
		return res
	}

	createInfos := make([]VkGraphicsPipelineCreateInfo, len(pipelines))
	for i, pipe := range pipelines {
		ci, err := s.createGraphicsPipelineCreateInfo(ctx,
			cb, gs, st, a, pipe, renderPass, alloc, allocAndRead,
			addCleanup, out)
		if err != nil {
			return nil, err
		}
		createInfos[i] = ci
	}

	createInfosData := allocAndRead(createInfos)

	newPipelines := make([]VkPipeline, len(pipelines))
	createdPipelines := map[VkPipeline]struct{}{}
	for i := range newPipelines {
		newPipelines[i] = VkPipeline(newUnusedID(false, func(id uint64) bool {
			_, ok := createdPipelines[VkPipeline(id)]
			return st.GraphicsPipelines().Contains(VkPipeline(id)) || ok
		}))
		createdPipelines[newPipelines[i]] = struct{}{}
	}
	newPipelinesData := allocAndRead(newPipelines)

	cmd := cb.VkCreateGraphicsPipelines(
		device, // device
		0,      // pipelineCache: VK_NULL_HANDLE
		uint32(len(pipelines)), // createInfoCount
		createInfosData.Ptr(),  // pCreateInfos
		memory.Nullptr,         // pAllocator
		newPipelinesData.Ptr(), // pPipelines
		VkResult_VK_SUCCESS,    // result
	).AddRead(
		createInfosData.Data(),
	).AddWrite(
		newPipelinesData.Data(),
	)

	for _, read := range reads {
		cmd.AddRead(read.Data())
	}

	writeEach(ctx, out, cmd)

	addCleanup(func() {
		for _, pipeline := range newPipelines {
			writeEach(ctx, out,
				cb.VkDestroyPipeline(
					device, pipeline, memory.Nullptr,
				))
		}
	})

	return newPipelines, nil
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
	pInfo := st.GraphicsPipelines().Get(pipeline)

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

	if !pInfo.DepthState().IsNil() {
		return NilVkGraphicsPipelineCreateInfo,
			&service.ErrDataUnavailable{Reason: messages.ErrMessage(
				"Pre-existing depth/stencil attachments not supported")}
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
	depthStencilPtr := allocAndRead(
		NewVkPipelineDepthStencilStateCreateInfo(a,
			VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO, // sType
			0,         // pNext
			0,         // flags
			0,         // depthTestEnable
			0,         // depthWriteEnable
			0,         // depthCompareOp
			0,         // depthBoundsTestEnable
			1,         // stencilTestEnable
			stencilOp, // front
			stencilOp, // back
			0,         // minDepthBounds
			0,         // maxDepthBounds
		)).Ptr()
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
	device VkDevice,
	cmdBuffer VkCommandBuffer,
	newRenderPass VkRenderPass,
	newFramebuffer VkFramebuffer,
	rpStartIdx uint64,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) (VkCommandBuffer, error) {
	bInfo := st.CommandBuffers().Get(cmdBuffer)

	pipelineMap := map[VkPipeline]VkPipeline{}
	rpEnded := false
	for i := 0; i < bInfo.CommandReferences().Len(); i++ {
		cr := bInfo.CommandReferences().Get(uint32(i))
		switch cr.Type() {
		case CommandType_cmd_vkCmdEndRenderPass:
			if uint64(i) >= rpStartIdx {
				rpEnded = true
			}
		case CommandType_cmd_vkCmdBindPipeline:
			// vkCmdBindPipeline can occur in secondary command
			// buffers, so we need to check those as well
			if !rpEnded && uint64(i) >= rpStartIdx {
				args := bInfo.
					BufferCommands().
					VkCmdBindPipeline().
					Get(cr.MapIndex())

				if args.PipelineBindPoint() ==
					VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS {
					// Record list of all pipelines used in renderpass
					pipelineMap[args.Pipeline()] = 0
				}
			}
		}
	}

	pipelineKeys := make([]VkPipeline, 0, len(pipelineMap))
	for key := range pipelineMap {
		pipelineKeys = append(pipelineKeys, key)
	}

	newPipelines, err := s.createGraphicsPipelines(ctx, cb, gs, st, a,
		device, pipelineKeys, newRenderPass, alloc, addCleanup, out)
	if err != nil {
		return 0, err
	}
	for i, key := range pipelineKeys {
		pipelineMap[key] = newPipelines[i]
	}

	newCmdBuffer, cmds, cleanup := allocateNewCmdBufFromExistingOneAndBegin(
		ctx, cb, cmdBuffer, gs)
	writeEach(ctx, out, cmds...)
	for _, f := range cleanup {
		f()
	}

	rpEnded = false
	for i := 0; i < bInfo.CommandReferences().Len(); i++ {
		cr := bInfo.CommandReferences().Get(uint32(i))
		args := GetCommandArgs(ctx, cr, st)
		if uint64(i) >= rpStartIdx && !rpEnded {
			switch ar := args.(type) {
			case VkCmdBeginRenderPassArgsʳ:
				newArgs := ar.Clone(a, api.CloneContext{})
				newArgs.SetRenderPass(newRenderPass)
				newArgs.SetFramebuffer(newFramebuffer)

				clearCount := uint32(newArgs.ClearValues().Len())
				// 0 initialize the stencil buffer
				newArgs.ClearValues().Add(clearCount,
					// Use VkClearColorValue instead of
					// VkClearDepthValue because it doesn't
					// seem like the union is set up in the
					// API DSL
					NewVkClearValue(a, NewVkClearColorValue(a,
						NewU32ː4ᵃ(a))))
				args = newArgs
			case VkCmdEndRenderPassArgsʳ:
				rpEnded = true
			case VkCmdBindPipelineArgsʳ:
				newArgs := ar
				if ar.PipelineBindPoint() ==
					VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS {
					newArgs = ar.Clone(a, api.CloneContext{})
					newArgs.SetPipeline(
						pipelineMap[ar.Pipeline()])
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
	device VkDevice,
	newRenderPass VkRenderPass,
	newFramebuffer VkFramebuffer,
	rpBeginIdx api.SubCmdIdx,
	cmdId api.CmdID,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) error {
	// TODO: check if we're allowed to modify the command directly, since
	// we won't be submitting the original one.  Need to deep clone all of
	// the submit info so we can mark it as reads.
	reads := []api.AllocResult{}
	allocAndRead := func(v ...interface{}) api.AllocResult {
		res := alloc(v)
		reads = append(reads, res)
		return res
	}

	submit.Extras().Observations().ApplyReads(gs.Memory.ApplicationPool())
	l := gs.MemoryLayout
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
						device, cmdBuffers[rpBeginIdx[1]],
						newRenderPass, newFramebuffer, rpBeginIdx[2],
						alloc, addCleanup, out)
				if err != nil {
					return err
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
	return nil
}

func (s *stencilOverdraw) Flush(ctx context.Context, output transform.Writer) {}
