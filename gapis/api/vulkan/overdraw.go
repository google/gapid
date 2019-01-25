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

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/core/stream/fmts"
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

type stencilOverdraw struct {
	rewrite    map[api.CmdID]replay.Result
	lastSubIdx map[api.CmdID]api.SubCmdIdx
}

func newStencilOverdraw() *stencilOverdraw {
	return &stencilOverdraw{
		rewrite:    map[api.CmdID]replay.Result{},
		lastSubIdx: map[api.CmdID]api.SubCmdIdx{},
	}
}

func (s *stencilOverdraw) add(ctx context.Context, extraCommands uint64, after []uint64, capt *path.Capture, res replay.Result) {
	c, err := capture.ResolveGraphicsFromPath(ctx, capt)
	if err != nil {
		res(nil, err)
		return
	}
	for lastSubmit := int64(after[0]); lastSubmit >= 0; lastSubmit-- {
		switch (c.Commands[lastSubmit]).(type) {
		case *VkQueueSubmit:
			id := api.CmdID(uint64(lastSubmit) + extraCommands)
			s.rewrite[id] = res
			s.lastSubIdx[id] = api.SubCmdIdx(after[1:])
			log.D(ctx, "Overdraw marked for submit id %v", lastSubmit)
			return
		}
	}
	res(nil, &service.ErrDataUnavailable{
		Reason: messages.ErrMessage("No last queue submission"),
	})
}

func (s *stencilOverdraw) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	gs := out.State()
	st := GetState(gs)
	arena := gs.Arena

	var allocated []*api.AllocResult
	defer func() {
		for _, d := range allocated {
			d.Free()
		}
	}()
	mustAllocData := func(v ...interface{}) api.AllocResult {
		res := gs.AllocDataOrPanic(ctx, v...)
		allocated = append(allocated, &res)
		return res
	}
	var cleanups []func()
	addCleanup := func(f func()) {
		cleanups = append(cleanups, f)
	}

	cb := CommandBuilder{Thread: cmd.Thread(), Arena: gs.Arena}

	switch c := cmd.(type) {
	case *VkCreateImage:
		// Need to make sure depth images are created with transfer
		// source mode, just in case they're being used in load mode
		// and we need to copy from them.
		s.rewriteImageCreate(ctx, cb, gs, st, arena, id, c, mustAllocData, out)
		return
	}
	res, ok := s.rewrite[id]
	if !ok {
		out.MutateAndWrite(ctx, id, cmd)
		return
	}

	submit, ok := cmd.(*VkQueueSubmit)
	if !ok {
		res(nil, &service.ErrDataUnavailable{Reason: messages.ErrMessage("Overdraw change marked for non-VkQueueSubmit")})
		out.MutateAndWrite(ctx, id, cmd)
		return
	}

	lastRenderPassArgs, lastRenderPassIdx, err :=
		s.getLastRenderPass(ctx, gs, st, submit, s.lastSubIdx[id])
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

	img, err := s.rewriteQueueSubmit(ctx, cb, gs, st, arena, submit,
		lastRenderPassArgs, lastRenderPassIdx, id,
		mustAllocData, addCleanup, out)
	if err != nil {
		res(nil, &service.ErrDataUnavailable{
			Reason: messages.ErrMessage(fmt.Sprintf(
				"Could not get overdraw: %v", err))})
		out.MutateAndWrite(ctx, id, cmd)
		return
	}

	checkImage := func(img *image.Data) error {
		// Check if any bytes are 255, which indicates potential saturation
		for _, byt := range img.Bytes {
			if byt == 255 {
				log.W(ctx, "Overdraw hit limit of 255, further overdraw cannot be measured")
				break
			}
		}
		// Even though the image comes from a stencil, content-wise
		// it's a gray image.
		img.Format = image.NewUncompressed("Count_U8", fmts.Count_U8)
		return nil
	}
	postImageData(ctx, cb, gs,
		st.Images().Get(img.handle),
		img.format,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT,
		0,
		0,
		img.width,
		img.height,
		img.width,
		img.height,
		checkImage,
		out,
		res,
	)
	// Don't defer this because we don't want these to execute if something panics
	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}
}

func (*stencilOverdraw) rewriteImageCreate(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	id api.CmdID,
	cmd *VkCreateImage,
	alloc func(...interface{}) api.AllocResult,
	out transform.Writer,
) {
	allReads := []api.AllocResult{}
	allocAndRead := func(v ...interface{}) api.AllocResult {
		res := alloc(v)
		allReads = append(allReads, res)
		return res
	}
	cmd.Extras().Observations().ApplyReads(gs.Memory.ApplicationPool())

	createInfo := cmd.PCreateInfo().MustRead(ctx, cmd, gs, nil)
	mask := VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_SRC_BIT)
	if !isDepthFormat(createInfo.Fmt()) || (createInfo.Usage()&mask == mask) {

		out.MutateAndWrite(ctx, id, cmd)
		return
	}

	newCreateInfo := createInfo.Clone(a, api.CloneContext{})

	if !newCreateInfo.PQueueFamilyIndices().IsNullptr() {
		indices := newCreateInfo.PQueueFamilyIndices().Slice(0,
			uint64(newCreateInfo.QueueFamilyIndexCount()), gs.MemoryLayout).
			MustRead(ctx, cmd, gs, nil)
		data := allocAndRead(indices)
		newCreateInfo.SetPQueueFamilyIndices(NewU32ᶜᵖ(data.Ptr()))
	}

	// If the image could be used as a depth buffer, make sure we can transfer from it
	newCreateInfo.SetUsage(newCreateInfo.Usage() | mask)

	newCreateInfoPtr := allocAndRead(newCreateInfo).Ptr()

	allocatorPtr := memory.Nullptr
	if !cmd.PAllocator().IsNullptr() {
		allocatorPtr = allocAndRead(
			cmd.PAllocator().MustRead(ctx, cmd, gs, nil)).Ptr()
	}

	cmd.Extras().Observations().ApplyWrites(gs.Memory.ApplicationPool())
	idData := alloc(cmd.PImage().MustRead(ctx, cmd, gs, nil))

	newCmd := cb.VkCreateImage(cmd.Device(), newCreateInfoPtr,
		allocatorPtr, idData.Ptr(),
		VkResult_VK_SUCCESS).AddWrite(idData.Data())
	for _, read := range allReads {
		newCmd.AddRead(read.Data())
	}

	out.MutateAndWrite(ctx, id, newCmd)
}

func (*stencilOverdraw) getLastRenderPass(ctx context.Context,
	gs *api.GlobalState,
	st *State,
	submit *VkQueueSubmit,
	lastIdx api.SubCmdIdx,
) (VkCmdBeginRenderPassArgsʳ, api.SubCmdIdx, error) {
	lastRenderPassArgs := NilVkCmdBeginRenderPassArgsʳ
	var lastRenderPassIdx api.SubCmdIdx
	submit.Extras().Observations().ApplyReads(gs.Memory.ApplicationPool())
	submitInfos := submit.PSubmits().Slice(0, uint64(submit.SubmitCount()),
		gs.MemoryLayout).MustRead(ctx, submit, gs, nil)
	for i, si := range submitInfos {
		if len(lastIdx) >= 1 && lastIdx[0] < uint64(i) {
			break
		}
		cmdBuffers := si.PCommandBuffers().Slice(0, uint64(si.CommandBufferCount()),
			gs.MemoryLayout).MustRead(ctx, submit, gs, nil)
		for j, buf := range cmdBuffers {
			if len(lastIdx) >= 2 && lastIdx[0] == uint64(i) && lastIdx[1] < uint64(j) {
				break
			}
			cb, ok := st.CommandBuffers().Lookup(buf)
			if !ok {
				return lastRenderPassArgs, lastRenderPassIdx,
					fmt.Errorf("Invalid command buffer %v", buf)
			}
			// vkCmdBeginRenderPass can only be in a primary command buffer,
			// so we don't need to check secondary command buffers
			for k := 0; k < cb.CommandReferences().Len(); k++ {
				if len(lastIdx) >= 3 && lastIdx[0] == uint64(i) &&
					lastIdx[1] == uint64(j) && lastIdx[2] < uint64(k) {
					break
				}
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
	view        VkImageView
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
		s.getStencilAttachmentDescription(ctx, st, a, oldRpInfo)
	if err != nil {
		return renderInfo{}, err
	}

	width, height := oldFbInfo.Width(), oldFbInfo.Height()
	// If we have a pre-existing depth image match our width and height to
	// that for when we render from one to the other.
	if depthIdx != ^uint32(0) {
		depthImage := oldFbInfo.ImageAttachments().Get(depthIdx).Image()
		width = depthImage.Info().Extent().Width()
		height = depthImage.Info().Extent().Height()
	}
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

	return renderInfo{renderPass, depthIdx, framebuffer, image, imageView}, nil
}

func (s *stencilOverdraw) getStencilAttachmentDescription(ctx context.Context,
	st *State,
	a arena.Arena,
	rpInfo RenderPassObjectʳ,
) (VkAttachmentDescription, uint32, error) {

	depthDesc, idx, err := s.getDepthAttachment(a, rpInfo)
	if err != nil {
		return NilVkAttachmentDescription, idx, err
	}

	// Clone it, but with a stencil-friendly format
	var stencilDesc VkAttachmentDescription
	var prefFmt VkFormat
	if idx != ^uint32(0) {
		stencilDesc = depthDesc.Clone(a, api.CloneContext{})

		prefFmt, err = s.depthToStencilFormat(depthDesc.Fmt())
		if err != nil {
			return NilVkAttachmentDescription, idx, err
		}
	} else {
		stencilDesc = MakeVkAttachmentDescription(a)
		prefFmt = 0xFFFFFFFF // defer to preference order
		stencilDesc.SetSamples(VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT)
		stencilDesc.SetLoadOp(VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE)
		stencilDesc.SetStoreOp(VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE)
	}
	format, err := s.getBestStencilFormat(ctx, st, rpInfo.Device(), prefFmt)
	if err != nil {
		return NilVkAttachmentDescription, idx, err
	}
	stencilDesc.SetFmt(format)
	stencilDesc.SetStencilLoadOp(VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_CLEAR)
	stencilDesc.SetStencilStoreOp(VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE)
	stencilDesc.SetInitialLayout(VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL)
	stencilDesc.SetFinalLayout(VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL)
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
			// TODO: Handle using separate depth attachments (make
			// a separate image for each one and combine them at
			// the end perhaps?)
			return NilVkAttachmentDescription, 0, fmt.Errorf(
				"The subpasses don't have matching depth attachments")
		}
	}
	if attachment0.IsNil() ||
		// VK_ATTACHMENT_UNUSED
		attachment0.Attachment() == ^uint32(0) {
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

func (s *stencilOverdraw) getBestStencilFormat(ctx context.Context,
	st *State,
	device VkDevice,
	preferred VkFormat,
) (VkFormat, error) {
	deviceInfo := st.Devices().Get(device)
	physicalDeviceInfo := st.PhysicalDevices().Get(deviceInfo.PhysicalDevice())
	formatProps := physicalDeviceInfo.FormatProperties()
	// It should have an entry for every format
	if !formatProps.Contains(VkFormat_VK_FORMAT_UNDEFINED) {
		if preferred == 0xFFFFFFFF {
			// Most likely to be supported
			preferred = VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT
		}
		log.W(ctx, "Format support information not available, assuming %v is ok.", preferred)
		return preferred, nil
	}

	supported := func(fmt VkFormat) bool {
		return (formatProps.Get(fmt).OptimalTilingFeatures() &
			VkFormatFeatureFlags(VkFormatFeatureFlagBits_VK_FORMAT_FEATURE_DEPTH_STENCIL_ATTACHMENT_BIT)) != 0
	}

	if supported(preferred) {
		return preferred, nil
	}

	var order []VkFormat
	if isDepthFormat(preferred) {
		// Use as many depth bits as we can
		order = []VkFormat{
			VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT,
			VkFormat_VK_FORMAT_D24_UNORM_S8_UINT,
			VkFormat_VK_FORMAT_D16_UNORM_S8_UINT,
		}
	} else {
		// Use as little space as possible
		order = []VkFormat{
			VkFormat_VK_FORMAT_S8_UINT,
			VkFormat_VK_FORMAT_D16_UNORM_S8_UINT,
			VkFormat_VK_FORMAT_D24_UNORM_S8_UINT,
			VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT,
		}
	}

	for _, fmt := range order {
		if supported(fmt) {
			if isDepthFormat(preferred) {
				log.W(ctx, "Format %v not supported, using %v instead.  Depth buffer may not act exactly as original.", preferred, fmt)
			}
			return fmt, nil
		}
	}
	return 0, fmt.Errorf("No depth/stencil formats supported")
}

func (s *stencilOverdraw) depthToStencilFormat(depthFormat VkFormat) (VkFormat, error) {
	switch depthFormat {
	case VkFormat_VK_FORMAT_D16_UNORM:
		return VkFormat_VK_FORMAT_D16_UNORM_S8_UINT, nil
	case VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32:
		return VkFormat_VK_FORMAT_D24_UNORM_S8_UINT, nil
	case VkFormat_VK_FORMAT_D32_SFLOAT:
		return VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT, nil

	case VkFormat_VK_FORMAT_D16_UNORM_S8_UINT,
		VkFormat_VK_FORMAT_D24_UNORM_S8_UINT,
		VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
		return depthFormat, nil
	default:
		return 0, fmt.Errorf("Unrecognized depth format %v",
			depthFormat)
	}
}

func (s *stencilOverdraw) depthStencilToDepthFormat(depthStencilFormat VkFormat) (VkFormat, error) {
	switch depthStencilFormat {
	case VkFormat_VK_FORMAT_D16_UNORM,
		VkFormat_VK_FORMAT_D16_UNORM_S8_UINT:
		return VkFormat_VK_FORMAT_D16_UNORM, nil
	case VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32,
		VkFormat_VK_FORMAT_D24_UNORM_S8_UINT:
		return VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32, nil
	case VkFormat_VK_FORMAT_D32_SFLOAT,
		VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
		return VkFormat_VK_FORMAT_D32_SFLOAT, nil
	default:
		return 0, fmt.Errorf("Unrecognized depth/stencil format %v",
			depthStencilFormat)
	}
}

func (s *stencilOverdraw) depthToStageFormat(depthFormat VkFormat) (VkFormat, error) {
	switch depthFormat {
	case VkFormat_VK_FORMAT_D16_UNORM:
		return VkFormat_VK_FORMAT_R16_UINT, nil
	case VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32,
		VkFormat_VK_FORMAT_D32_SFLOAT:
		return VkFormat_VK_FORMAT_R32_UINT, nil
	default:
		return 0, fmt.Errorf("Unrecognized depth format %v",
			depthFormat)
	}
}

func isDepthFormat(depthFormat VkFormat) bool {
	return depthBits(depthFormat) != 0
}

func depthBits(depthFormat VkFormat) int {
	switch depthFormat {
	case VkFormat_VK_FORMAT_D16_UNORM,
		VkFormat_VK_FORMAT_D16_UNORM_S8_UINT:
		return 16
	case VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32,
		VkFormat_VK_FORMAT_D24_UNORM_S8_UINT:
		return 24
	case VkFormat_VK_FORMAT_D32_SFLOAT,
		VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
		return 32
	default:
		return 0
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
		0,                                  // pNext
		0,                                  // flags
		VkImageType_VK_IMAGE_TYPE_2D,       // imageType
		format,                             // format
		NewVkExtent3D(a, width, height, 1), // extent
		1,                                  // mipLevels
		1,                                  // arrayLevels
		VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT, // samples
		VkImageTiling_VK_IMAGE_TILING_OPTIMAL,       // tiling
		VkImageUsageFlags( // usage
			VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT|
				VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_SRC_BIT|
				VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT|
				VkImageUsageFlagBits_VK_IMAGE_USAGE_INPUT_ATTACHMENT_BIT),
		VkSharingMode_VK_SHARING_MODE_EXCLUSIVE, // sharingMode
		0,                                       // queueFamilyIndexCount
		0,                                       // pQueueFamilyIndices
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
	imageObject := st.Images().Get(image)
	createInfo := NewVkImageViewCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO, // sType
		0,                                     // pNext
		0,                                     // flags
		image,                                 // image
		VkImageViewType_VK_IMAGE_VIEW_TYPE_2D, // viewType
		imageObject.Info().Fmt(),              // format
		NewVkComponentMapping(a,
			VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
			VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
			VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
			VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
		), // components
		NewVkImageSubresourceRange(a,
			imageObject.ImageAspect(), // aspectMask
			0,                         // baseMipLevel
			1,                         // levelCount
			0,                         // baseArrayLayer
			1,                         // layerCount
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
	newAttachmentsData, newAttachmentsLen := unpackMapWithAllocator(allocAndRead,
		newAttachments)

	stencilAttachmentReference := NewVkAttachmentReference(a,
		uint32(len(attachments)),
		stencilAttachment.InitialLayout(),
	)
	stencilAttachmentReferencePtr := allocAndRead(stencilAttachmentReference).Ptr()

	subpasses := make([]VkSubpassDescription,
		rpInfo.SubpassDescriptions().Len())
	for idx, subpass := range rpInfo.SubpassDescriptions().All() {
		subpasses[idx] = subpassToSubpassDescription(a, subpass,
			stencilAttachmentReferencePtr, allocAndRead)
	}
	subpassesData := allocAndRead(subpasses)

	subpassDependenciesData, subpassDependenciesLen := unpackMapWithAllocator(allocAndRead,
		rpInfo.SubpassDependencies())

	renderPassCreateInfo := NewVkRenderPassCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO, // sType
		0,                 // pNext
		0,                 // flags
		newAttachmentsLen, // attachmentCount
		NewVkAttachmentDescriptionᶜᵖ(newAttachmentsData.Ptr()), // pAttachments
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
			allocation, count := unpackMapWithAllocator(allocAndRead, m)
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
		0,                           // pNext
		0,                           // flags
		renderPass,                  // renderPass
		uint32(len(newAttachments)), // attachmentCount
		NewVkImageViewᶜᵖ(newAttachmentsData.Ptr()), // pAttachments
		fbInfo.Width(),  // width
		fbInfo.Height(), // height
		fbInfo.Layers(), // layers
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
			allocation, count := unpackMapWithAllocator(allocAndRead, m)
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
				attributeCount, // vertexAttributeDescriptionCount
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
				0,                         // pNext
				0,                         // flags
				info.PatchControlPoints(), // patchControlPoints
			)).Ptr()
	}

	viewportPtr := memory.Nullptr
	if !pInfo.ViewportState().IsNil() {
		info := pInfo.ViewportState()
		viewPtr, _ := unpackMapMaybeEmpty(info.Viewports())
		scissorPtr, _ := unpackMapMaybeEmpty(info.Scissors())
		viewportPtr = allocAndRead(
			NewVkPipelineViewportStateCreateInfo(a,
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO, // sType
				0,                         // pNext
				0,                         // flags
				info.ViewportCount(),      // viewportCount
				NewVkViewportᶜᵖ(viewPtr),  // pViewports
				info.ScissorCount(),       // scissorCount
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
			0,                                // depthFailOp
			VkCompareOp_VK_COMPARE_OP_ALWAYS, // compareOp
			255,                              // compareMask
			255,                              // writeMask
			0,                                // reference
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

	flags := pInfo.Flags()
	basePipelineHandle := VkPipeline(0)
	if flags&VkPipelineCreateFlags(
		VkPipelineCreateFlagBits_VK_PIPELINE_CREATE_ALLOW_DERIVATIVES_BIT) != 0 {

		flags |= VkPipelineCreateFlags(
			VkPipelineCreateFlagBits_VK_PIPELINE_CREATE_DERIVATIVE_BIT)
		basePipelineHandle = pipeline
	}

	return NewVkGraphicsPipelineCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO, // sType
		0,                 // pNext
		0,                 // flags
		shaderStagesCount, // stageCount
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
		basePipelineHandle,                                            // basePipelineHandle
		-1,                                                            // basePipelineIndex
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
	mapEntries, mapEntryCount := unpackMapWithAllocator(allocAndRead, info.Specializations().All())
	data := info.Data().MustRead(ctx, nil, gs, nil)
	return NewVkSpecializationInfoᶜᵖ(allocAndRead(
		NewVkSpecializationInfo(a,
			mapEntryCount, // mapEntryCount
			NewVkSpecializationMapEntryᶜᵖ(mapEntries.Ptr()), // pMapEntries
			memory.Size(len(data)),                          // dataSize,
			NewVoidᶜᵖ(allocAndRead(data).Ptr()),             // pData
		)).Ptr())
}

func (*stencilOverdraw) createDepthCopyBuffer(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	device VkDevice,
	format VkFormat,
	width uint32,
	height uint32,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) VkBuffer {
	elsize := VkDeviceSize(4)
	if format == VkFormat_VK_FORMAT_D16_UNORM ||
		format == VkFormat_VK_FORMAT_D16_UNORM_S8_UINT {

		elsize = VkDeviceSize(2)
	}

	bufferSize := elsize * VkDeviceSize(width*height)

	buffer := VkBuffer(newUnusedID(false, func(id uint64) bool {
		return st.Buffers().Contains(VkBuffer(id))
	}))
	bufferData := alloc(buffer)

	bufferInfo := NewVkBufferCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO, // sType
		0,          // pNext
		0,          // flags
		bufferSize, // size
		VkBufferUsageFlags(
			VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT|
				VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT), // usage
		VkSharingMode_VK_SHARING_MODE_EXCLUSIVE, // sharingMode
		0,                                       // queueFamilyIndexCount
		0,                                       // pQueueFamilyIndices
	)
	bufferInfoData := alloc(bufferInfo)

	bufferMemoryTypeIndex := uint32(0)
	physicalDevice := st.PhysicalDevices().Get(
		st.Devices().Get(device).PhysicalDevice(),
	)
	for i := uint32(0); i < physicalDevice.MemoryProperties().MemoryTypeCount(); i++ {
		t := physicalDevice.MemoryProperties().MemoryTypes().Get(int(i))
		if 0 != (t.PropertyFlags() & VkMemoryPropertyFlags(
			VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT)) {
			bufferMemoryTypeIndex = i
			break
		}
	}

	deviceMemory := VkDeviceMemory(newUnusedID(false, func(id uint64) bool {
		return st.DeviceMemories().Contains(VkDeviceMemory(id))
	}))
	deviceMemoryData := alloc(deviceMemory)
	memoryAllocateInfo := NewVkMemoryAllocateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, // sType
		0,                     // pNext
		bufferSize*2,          // allocationSize
		bufferMemoryTypeIndex, // memoryTypeIndex
	)
	memoryAllocateInfoData := alloc(memoryAllocateInfo)

	writeEach(ctx, out,
		cb.VkCreateBuffer(
			device,
			bufferInfoData.Ptr(),
			memory.Nullptr,
			bufferData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			bufferInfoData.Data(),
		).AddWrite(
			bufferData.Data(),
		),
		cb.VkAllocateMemory(
			device,
			memoryAllocateInfoData.Ptr(),
			memory.Nullptr,
			deviceMemoryData.Ptr(),
			VkResult_VK_SUCCESS,
		).AddRead(
			memoryAllocateInfoData.Data(),
		).AddWrite(
			deviceMemoryData.Data(),
		),
		cb.VkBindBufferMemory(
			device,
			buffer,
			deviceMemory,
			0,
			VkResult_VK_SUCCESS,
		),
	)

	addCleanup(func() {
		writeEach(ctx, out,
			cb.VkDestroyBuffer(
				device,
				buffer,
				memory.Nullptr,
			),
			cb.VkFreeMemory(
				device,
				deviceMemory,
				memory.Nullptr,
			),
		)
	})

	return buffer
}

type imageDesc struct {
	image       ImageObjectʳ
	subresource VkImageSubresourceRange
	layout      VkImageLayout
	// This might not be the same as subresource.aspectMask if we only want
	// to copy one aspect
	aspect VkImageAspectFlagBits
}

// Facilitate copying the depth aspect of an image from one image to another,
// either for going from the original depth buffer to our depth buffer,
// or copying back the new depth buffer to the original depth buffer.
func (s *stencilOverdraw) copyImageAspect(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	device VkDevice,
	cmdBuffer VkCommandBuffer,
	srcImgDesc imageDesc,
	dstImgDesc imageDesc,
	extent VkExtent3D,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) {
	srcImg := srcImgDesc.image
	dstImg := dstImgDesc.image
	copyBuffer := s.createDepthCopyBuffer(ctx, cb, gs, st, a, device,
		srcImg.Info().Fmt(),
		extent.Width(), extent.Height(),
		alloc, addCleanup, out)

	allCommandsStage := VkPipelineStageFlags(
		VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT)
	allMemoryAccess := VkAccessFlags(
		VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT |
			VkAccessFlagBits_VK_ACCESS_MEMORY_READ_BIT)

	imgBarriers0 := make([]VkImageMemoryBarrier, 2)
	imgBarriers1 := make([]VkImageMemoryBarrier, 2)
	// Transition the src image in and out of the required layouts
	imgBarriers0[0] = NewVkImageMemoryBarrier(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
		0,               // pNext
		allMemoryAccess, // srcAccessMask
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT), // dstAccessMask
		srcImgDesc.layout, // oldLayout
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, // newLayout
		^uint32(0),             // srcQueueFamilyIndex: VK_QUEUE_FAMILY_IGNORED
		^uint32(0),             // dstQueueFamilyIndex
		srcImg.VulkanHandle(),  // image
		srcImgDesc.subresource, // subresourceRange
	)
	srcFinalLayout := srcImgDesc.layout
	if srcFinalLayout == VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED ||
		srcFinalLayout == VkImageLayout_VK_IMAGE_LAYOUT_PREINITIALIZED {
		srcFinalLayout = VkImageLayout_VK_IMAGE_LAYOUT_GENERAL
	}
	imgBarriers1[0] = NewVkImageMemoryBarrier(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
		0, // pNext
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT), // srcAccessMask
		allMemoryAccess, // dstAccessMask
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, // oldLayout
		srcFinalLayout,         // newLayout
		^uint32(0),             // srcQueueFamilyIndex: VK_QUEUE_FAMILY_IGNORED
		^uint32(0),             // dstQueueFamilyIndex
		srcImg.VulkanHandle(),  // image
		srcImgDesc.subresource, // subresourceRange
	)

	// Transition the new image in and out of its required layouts
	imgBarriers0[1] = NewVkImageMemoryBarrier(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
		0,               // pNext
		allMemoryAccess, // srcAccessMask
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT), // dstAccessMask
		dstImgDesc.layout, // oldLayout
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, // newLayout
		^uint32(0),             // srcQueueFamilyIndex: VK_QUEUE_FAMILY_IGNORED
		^uint32(0),             // dstQueueFamilyIndex
		dstImg.VulkanHandle(),  // image
		dstImgDesc.subresource, // subresourceRange
	)

	dstFinalLayout := dstImgDesc.layout
	if dstFinalLayout == VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED ||
		dstFinalLayout == VkImageLayout_VK_IMAGE_LAYOUT_PREINITIALIZED {
		dstFinalLayout = VkImageLayout_VK_IMAGE_LAYOUT_GENERAL
	}
	imgBarriers1[1] = NewVkImageMemoryBarrier(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
		0, // pNext
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT), // srcAccessMask
		allMemoryAccess, // dstAccessMask
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, // oldLayout
		dstFinalLayout,         // newLayout
		^uint32(0),             // srcQueueFamilyIndex: VK_QUEUE_FAMILY_IGNORED
		^uint32(0),             // dstQueueFamilyIndex
		dstImg.VulkanHandle(),  // image
		dstImgDesc.subresource, // subresourceRange
	)

	bufBarrier := NewVkBufferMemoryBarrier(a,
		VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, // sType
		0, // pNext
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT), // srcAccessMask
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT),  // dstAccessMask
		^uint32(0),       // srcQueueFamilyIndex: VK_QUEUE_FAMILY_IGNORED
		^uint32(0),       // dstQueueFamilyIndex
		copyBuffer,       // buffer
		0,                // offset
		^VkDeviceSize(0), // size: VK_WHOLE_SIZE
	)

	ibCopy := NewVkBufferImageCopy(a,
		0, // bufferOffset
		0, // bufferRowLength
		0, // bufferImageHeight
		NewVkImageSubresourceLayers(a,
			VkImageAspectFlags(srcImgDesc.aspect),   // aspectMask
			srcImgDesc.subresource.BaseMipLevel(),   // mipLevel
			srcImgDesc.subresource.BaseArrayLayer(), // baseArrayLayer
			1,                                       // layerCount
		), // srcSubresource
		NewVkOffset3D(a, 0, 0, 0),                            // offset
		NewVkExtent3D(a, extent.Width(), extent.Height(), 1), // extent
	)

	biCopy := NewVkBufferImageCopy(a,
		0, // bufferOffset
		0, // bufferRowLength
		0, // bufferImageHeight
		NewVkImageSubresourceLayers(a,
			VkImageAspectFlags(dstImgDesc.aspect),   // aspectMask
			dstImgDesc.subresource.BaseMipLevel(),   // mipLevel
			dstImgDesc.subresource.BaseArrayLayer(), // baseArrayLayer
			1,                                       // layerCount
		), // srcSubresource
		NewVkOffset3D(a, 0, 0, 0),                            // offset
		NewVkExtent3D(a, extent.Width(), extent.Height(), 1), // extent
	)

	imgBarriers0Data := alloc(imgBarriers0)
	ibCopyData := alloc(ibCopy)
	bufBarrierData := alloc(bufBarrier)
	biCopyData := alloc(biCopy)
	imgBarriers1Data := alloc(imgBarriers1)

	writeEach(ctx, out,
		cb.VkCmdPipelineBarrier(cmdBuffer,
			allCommandsStage, // srcStageMask
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_TRANSFER_BIT), // dstStageMask
			0,                      // dependencyFlags
			0,                      // memoryBarrierCount
			memory.Nullptr,         // pMemoryBarriers
			0,                      // bufferMemoryBarrierCount
			memory.Nullptr,         // pBufferMemoryBarriers
			2,                      // imageMemoryBarrierCount
			imgBarriers0Data.Ptr(), // pImageMemoryBarriers
		).AddRead(imgBarriers0Data.Data()),
		cb.VkCmdCopyImageToBuffer(cmdBuffer,
			srcImg.VulkanHandle(),                              // srcImage
			VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, // srcImageLayout
			copyBuffer,       // dstBuffer
			1,                // regionCount
			ibCopyData.Ptr(), // pRegions
		).AddRead(ibCopyData.Data()),
		cb.VkCmdPipelineBarrier(cmdBuffer,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_TRANSFER_BIT), // srcStageMask
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_TRANSFER_BIT), // dstStageMask
			0,                    // dependencyFlags
			0,                    // memoryBarrierCount
			memory.Nullptr,       // pMemoryBarriers
			1,                    // bufferMemoryBarrierCount
			bufBarrierData.Ptr(), // pBufferMemoryBarriers
			0,                    // imageMemoryBarrierCount
			memory.Nullptr,       // pImageMemoryBarriers
		).AddRead(bufBarrierData.Data()),
		cb.VkCmdCopyBufferToImage(cmdBuffer,
			copyBuffer,            // srcBuffer
			dstImg.VulkanHandle(), // dstImage
			VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, // dstImageLayout
			1,                // regionCount
			biCopyData.Ptr(), // pRegions
		).AddRead(biCopyData.Data()),
		cb.VkCmdPipelineBarrier(cmdBuffer,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_TRANSFER_BIT), // srcStageMask
			allCommandsStage,       // dstStageMask
			0,                      // dependencyFlags
			0,                      // memoryBarrierCount
			memory.Nullptr,         // pMemoryBarriers
			0,                      // bufferMemoryBarrierCount
			memory.Nullptr,         // pBufferMemoryBarriers
			2,                      // imageMemoryBarrierCount
			imgBarriers1Data.Ptr(), // pImageMemoryBarriers
		).AddRead(imgBarriers1Data.Data()),
	)
}

func (s *stencilOverdraw) renderAspect(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	device VkDevice,
	queue VkQueue,
	cmdBuffer VkCommandBuffer,
	srcImg imageDesc,
	dstImg imageDesc,
	inputFormat VkFormat,
	addCleanup func(func()),
	out transform.Writer,
) error {
	sb := st.newStateBuilder(ctx, newTransformerOutput(out))
	ip := newImagePrimer(sb)
	queueHandler, err := newQueueCommandHandler(sb, queue, cmdBuffer)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at creating queue command handler")
	}
	dstLayer := dstImg.subresource.BaseArrayLayer()
	if srcImg.subresource.BaseArrayLayer() != dstLayer {
		return fmt.Errorf("input attachment and render target layer do not match")
	}
	dstLevel := dstImg.subresource.BaseMipLevel()
	if srcImg.subresource.BaseMipLevel() != dstLevel {
		return fmt.Errorf("input attachment and render target mip level do not match")
	}
	sizes := sb.levelSize(dstImg.image.Info().Extent(),
		dstImg.image.Info().Fmt(), dstLevel, dstImg.aspect)
	recipe := ipRenderRecipe{
		inputAttachmentImage:  srcImg.image.VulkanHandle(),
		inputAttachmentAspect: srcImg.aspect,
		renderImage:           dstImg.image.VulkanHandle(),
		renderAspect:          dstImg.aspect,
		layer:                 dstLayer,
		level:                 dstLevel,
		renderRectX:           int32(0),
		renderRectY:           int32(0),
		renderRectWidth:       uint32(sizes.width),
		renderRectHeight:      uint32(sizes.height),
		wordIndex:             uint32(0),
		framebufferWidth:      uint32(sizes.width),
		framebufferHeight:     uint32(sizes.height),
	}
	renderKitBuilder := ip.GetRenderKitBuilder(device)
	kits, err := renderKitBuilder.BuildRenderKits(sb, recipe)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at building render kits")
	}
	if len(kits) != 1 {
		return fmt.Errorf("unexpected length of render kits, actual: %v, expected: 1", len(kits))
	}
	renderingLayout := ipRenderColorOutputLayout
	if (dstImg.image.Info().Usage() & VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)) != 0 {
		renderingLayout = ipRenderDepthStencilOutputLayout
	}

	inputPreBarrier := ipImageSubresourceLayoutTransitionBarrier(sb,
		srcImg.image,
		srcImg.aspect,
		srcImg.subresource.BaseArrayLayer(),
		srcImg.subresource.BaseMipLevel(),
		srcImg.layout,
		ipRenderInputAttachmentLayout,
	)
	inputPostBarrier := ipImageSubresourceLayoutTransitionBarrier(sb,
		srcImg.image,
		srcImg.aspect,
		srcImg.subresource.BaseArrayLayer(),
		srcImg.subresource.BaseMipLevel(),
		ipRenderInputAttachmentLayout,
		srcImg.layout,
	)
	outputPreBarrier := ipImageSubresourceLayoutTransitionBarrier(sb,
		dstImg.image,
		dstImg.aspect,
		dstImg.subresource.BaseArrayLayer(),
		dstImg.subresource.BaseMipLevel(),
		dstImg.layout,
		renderingLayout,
	)
	outputPostBarrier := ipImageSubresourceLayoutTransitionBarrier(sb,
		dstImg.image,
		dstImg.aspect,
		dstImg.subresource.BaseArrayLayer(),
		dstImg.subresource.BaseMipLevel(),
		renderingLayout,
		dstImg.layout,
	)

	err = ipRecordImageMemoryBarriers(sb, queueHandler, inputPreBarrier, outputPreBarrier)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at recording pre rendering image layout transition")
	}
	renderCmds := kits[0].BuildRenderCommands(sb)
	err = renderCmds.Commit(sb, queueHandler)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at commiting rendering commands")
	}
	err = ipRecordImageMemoryBarriers(sb, queueHandler, inputPostBarrier, outputPostBarrier)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at recording post rendering image layout transition")
	}

	// Make sure it doesn't use temporary memory as that would cause a flush of the scratch resources
	// queueScratch.memorySize = scratchTask.totalAllocationSize

	queueHandler.RecordPostExecuted(func() { ip.Free() })
	addCleanup(func() {
		queueHandler.Submit(sb)
		queueHandler.WaitUntilFinish(sb)
	})

	return nil
}

func (s *stencilOverdraw) transferDepthValues(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	device VkDevice,
	queue VkQueue,
	cmdBuffer VkCommandBuffer,
	width uint32,
	height uint32,
	srcImgDesc imageDesc,
	dstImgDesc imageDesc,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) error {
	srcFmt := srcImgDesc.image.Info().Fmt()
	dstFmt := dstImgDesc.image.Info().Fmt()
	if depthBits(srcFmt) == depthBits(dstFmt) {
		s.copyImageAspect(ctx, cb, gs, st, a, device, cmdBuffer,
			srcImgDesc,
			dstImgDesc,
			NewVkExtent3D(a, width, height, 1),
			alloc, addCleanup, out)
		return nil
	} else {
		// This would have errored previously if it was going to error now
		depthFmt, _ := s.depthStencilToDepthFormat(srcFmt)
		stageFmt, _ := s.depthToStageFormat(depthFmt)
		stageImgInfo, err := s.createImage(ctx, cb, st, a, device,
			stageFmt, width, height, alloc, addCleanup,
			out)
		if err != nil {
			return err
		}
		stageImg := st.Images().Get(stageImgInfo.handle)
		stageImgDesc := imageDesc{
			stageImg,
			NewVkImageSubresourceRange(a,
				VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT),
				0,
				1,
				0,
				1,
			),
			VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED, // this will be transitioned to general
			VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
		}
		s.copyImageAspect(ctx, cb, gs, st, a, device, cmdBuffer,
			srcImgDesc,
			stageImgDesc,
			NewVkExtent3D(a, width, height, 1),
			alloc, addCleanup, out)
		stageImgDesc.layout = VkImageLayout_VK_IMAGE_LAYOUT_GENERAL
		return s.renderAspect(ctx, cb, gs, st, a, device, queue,
			cmdBuffer, stageImgDesc, dstImgDesc, srcFmt, addCleanup, out)
	}
}

// If the depth attachment is in "load" mode we need to copy the depth values
// over to the depth aspect of our new depth/stencil buffer.
func (s *stencilOverdraw) loadExistingDepthValues(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	device VkDevice,
	queue VkQueue,
	cmdBuffer VkCommandBuffer,
	renderInfo renderInfo,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) error {
	if renderInfo.depthIdx == ^uint32(0) {
		return nil
	}
	rpInfo := st.RenderPasses().Get(renderInfo.renderPass)
	daInfo := rpInfo.AttachmentDescriptions().Get(renderInfo.depthIdx)

	if daInfo.LoadOp() != VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD {
		return nil
	}

	fbInfo := st.Framebuffers().Get(renderInfo.framebuffer)

	oldImageView := fbInfo.ImageAttachments().Get(renderInfo.depthIdx)
	newImageView := fbInfo.ImageAttachments().Get(uint32(fbInfo.ImageAttachments().Len() - 1))

	oldImageDesc := imageDesc{
		oldImageView.Image(),
		oldImageView.SubresourceRange(),
		daInfo.InitialLayout(),
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT,
	}
	newImageDesc := imageDesc{
		newImageView.Image(),
		newImageView.SubresourceRange(),
		VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT,
	}

	return s.transferDepthValues(ctx, cb, gs, st, a,
		device, queue, cmdBuffer,
		fbInfo.Width(), fbInfo.Height(),
		oldImageDesc, newImageDesc,
		alloc, addCleanup, out)
}

// If the depth attachment is in "store" mode we need to copy the depth values
// over from the depth aspect of our new depth/stencil buffer.
func (s *stencilOverdraw) storeNewDepthValues(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	device VkDevice,
	queue VkQueue,
	cmdBuffer VkCommandBuffer,
	renderInfo renderInfo,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) error {
	if renderInfo.depthIdx == ^uint32(0) {
		return nil
	}
	rpInfo := st.RenderPasses().Get(renderInfo.renderPass)
	daInfo := rpInfo.AttachmentDescriptions().Get(renderInfo.depthIdx)

	if daInfo.StoreOp() != VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE {
		return nil
	}

	fbInfo := st.Framebuffers().Get(renderInfo.framebuffer)

	oldImageView := fbInfo.ImageAttachments().Get(uint32(fbInfo.ImageAttachments().Len() - 1))
	newImageView := fbInfo.ImageAttachments().Get(renderInfo.depthIdx)

	oldImageDesc := imageDesc{
		oldImageView.Image(),
		oldImageView.SubresourceRange(),
		rpInfo.AttachmentDescriptions().Get(uint32(fbInfo.ImageAttachments().Len() - 1)).FinalLayout(),
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT,
	}
	newImageDesc := imageDesc{
		newImageView.Image(),
		newImageView.SubresourceRange(),
		daInfo.FinalLayout(),
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT,
	}
	return s.transferDepthValues(ctx, cb, gs, st, a,
		device, queue, cmdBuffer,
		fbInfo.Width(), fbInfo.Height(),
		oldImageDesc, newImageDesc,
		alloc, addCleanup, out)
}

func (s *stencilOverdraw) transitionStencilImage(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	cmdBuffer VkCommandBuffer,
	renderInfo renderInfo,
	alloc func(v ...interface{}) api.AllocResult,
	out transform.Writer,
) {
	imageView := st.ImageViews().Get(renderInfo.view)
	imgBarrier := NewVkImageMemoryBarrier(a,
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
		0, // pNext
		0, // srcAccessMask
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT|
			VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_READ_BIT|
			VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT), // dstAccessMask
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,                        // oldLayout
		VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL, // newLayout
		^uint32(0),                       // srcQueueFamilyIndex: VK_QUEUE_FAMILY_IGNORED
		^uint32(0),                       // dstQueueFamilyIndex
		imageView.Image().VulkanHandle(), // image
		imageView.SubresourceRange(),     // subresourceRange
	)
	imgBarrierData := alloc(imgBarrier)

	writeEach(ctx, out,
		cb.VkCmdPipelineBarrier(cmdBuffer,
			VkPipelineStageFlags(
				VkPipelineStageFlagBits_VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT), // srcStageMask
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_TRANSFER_BIT|
				VkPipelineStageFlagBits_VK_PIPELINE_STAGE_EARLY_FRAGMENT_TESTS_BIT|
				VkPipelineStageFlagBits_VK_PIPELINE_STAGE_LATE_FRAGMENT_TESTS_BIT), // dstStageMask
			0,                    // dependencyFlags
			0,                    // memoryBarrierCount
			memory.Nullptr,       // pMemoryBarriers
			0,                    // bufferMemoryBarrierCount
			memory.Nullptr,       // pBufferMemoryBarriers
			1,                    // imageMemoryBarrierCount
			imgBarrierData.Ptr(), // pImageMemoryBarriers
		).AddRead(imgBarrierData.Data()),
	)
}

func (s *stencilOverdraw) createCommandBuffer(ctx context.Context,
	cb CommandBuilder,
	gs *api.GlobalState,
	st *State,
	a arena.Arena,
	queue VkQueue,
	cmdBuffer VkCommandBuffer,
	renderInfo renderInfo,
	rpStartIdx uint64,
	alloc func(v ...interface{}) api.AllocResult,
	addCleanup func(func()),
	out transform.Writer,
) (VkCommandBuffer, error) {
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
				// Transition the stencil image to the right layout
				s.transitionStencilImage(ctx, cb, gs, st, a,
					newCmdBuffer, renderInfo, alloc, out)
				// Add commands to handle copying the old depth
				// values if necessary
				if err := s.loadExistingDepthValues(ctx, cb,
					gs, st, a, device, queue, newCmdBuffer,
					renderInfo, alloc, addCleanup, out); err != nil {
					return 0, err
				}

				newArgs := ar.Clone(a, api.CloneContext{})
				newArgs.SetRenderPass(renderInfo.renderPass)
				newArgs.SetFramebuffer(renderInfo.framebuffer)

				rpInfo := st.RenderPasses().Get(renderInfo.renderPass)
				attachmentIdx := uint32(rpInfo.AttachmentDescriptions().Len()) - 1
				newClear := NewU32ː4ᵃ(a)

				if renderInfo.depthIdx != ^uint32(0) &&
					rpInfo.AttachmentDescriptions().Get(renderInfo.depthIdx).LoadOp() ==
						VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_CLEAR {
					newClear.Set(0, newArgs.
						ClearValues().
						Get(renderInfo.depthIdx).
						Color().
						Uint32().
						Get(0))
				}
				for j := uint32(0); j < attachmentIdx; j++ {
					if !newArgs.ClearValues().Contains(j) {
						newArgs.ClearValues().Add(j, NilVkClearValue)
					}
				}
				// 0 initialize the stencil buffer
				newArgs.ClearValues().Add(attachmentIdx,
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
						newCmdbuf, err = s.createCommandBuffer(ctx, cb,
							gs, st, a, queue, cmdbuf,
							renderInfo, 0, alloc,
							addCleanup, out)
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

		if _, ok := args.(VkCmdEndRenderPassArgsʳ); ok {
			// Add commands to handle storing the new depth values if necessary
			if err := s.storeNewDepthValues(ctx, cb, gs, st, a, device,
				queue, newCmdBuffer, renderInfo, alloc, addCleanup, out); err != nil {
				return 0, err
			}
		}
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
	// Need to deep clone all of the submit info so we can mark it as
	// reads.  TODO: We could possibly optimize this by copying the
	// pointers and using the fact that we know what size it should be to
	// create the observations.
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
						submit.Queue(),
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
			0,                            // pNext
			si.WaitSemaphoreCount(),      // waitSemaphoreCount
			NewVkSemaphoreᶜᵖ(waitSemPtr), // pWaitSemaphores
			NewVkPipelineStageFlagsᶜᵖ(waitDstStagePtr), // pWaitDstStageMask
			si.CommandBufferCount(),                    // commandBufferCount
			NewVkCommandBufferᶜᵖ(cmdBufferPtr),         // pCommandBuffers
			si.SignalSemaphoreCount(),                  // signalSemaphoreCount
			NewVkSemaphoreᶜᵖ(signalSemPtr),             // pSignalSemaphores
		)
	}
	submitInfoPtr := allocAndRead(newSubmitInfos).Ptr()

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

func (*stencilOverdraw) Flush(ctx context.Context, out transform.Writer) {}
