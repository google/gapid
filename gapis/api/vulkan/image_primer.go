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
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/stream"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
)

type imagePrimer struct {
	sb *stateBuilder
	rh *ipRenderHandler
}

func newImagePrimer(sb *stateBuilder) *imagePrimer {
	p := &imagePrimer{
		sb: sb,
		rh: newImagePrimerRenderHandler(sb),
	}
	return p
}

// interfaces to interact with state rebuilder

func (p *imagePrimer) primeByBufferCopy(img *ImageObject, opaqueBoundRanges []VkImageSubresourceRange, queue *QueueObject, sparseBindingQueue *QueueObject) error {
	job := newImagePrimerBufCopyJob(img, img.Info.Layout)
	for _, aspect := range p.sb.imageAspectFlagBits(img.ImageAspect) {
		job.addDst(aspect, aspect, img)
	}
	bcs := newImagePrimerBufferCopySession(p.sb, job)
	for _, rng := range opaqueBoundRanges {
		bcs.collectCopiesFromSubresourceRange(rng)
	}
	if sparseResidency(img) {
		bcs.collectCopiesFromSparseImageBindings()
	}
	err := bcs.rolloutBufCopies(queue, sparseBindingQueue)
	if err != nil {
		return fmt.Errorf("[Priming image data by buffer->image copy] %v", err)
	}
	return nil
}

func (p *imagePrimer) primeByRendering(img *ImageObject, opaqueBoundRanges []VkImageSubresourceRange, queue *QueueObject, sparseBindingQueue *QueueObject) error {
	copyJob := newImagePrimerBufCopyJob(img, VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL)
	for _, aspect := range p.sb.imageAspectFlagBits(img.ImageAspect) {
		stagingImgs, stagingImgMems, err := p.allocStagingImages(img, aspect)
		if err != nil {
			return fmt.Errorf("[Creating staging image for priming image data by rendering] %v", err)
		}
		defer func() {
			for _, img := range stagingImgs {
				p.sb.write(p.sb.cb.VkDestroyImage(img.Device, img.VulkanHandle, memory.Nullptr))
			}
			for _, mem := range stagingImgMems {
				p.sb.write(p.sb.cb.VkFreeMemory(mem.Device, mem.VulkanHandle, memory.Nullptr))
			}
		}()
		copyJob.addDst(aspect, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, stagingImgs...)
	}
	bcs := newImagePrimerBufferCopySession(p.sb, copyJob)
	for _, rng := range opaqueBoundRanges {
		bcs.collectCopiesFromSubresourceRange(rng)
	}
	if sparseResidency(img) {
		bcs.collectCopiesFromSparseImageBindings()
	}
	err := bcs.rolloutBufCopies(queue, sparseBindingQueue)
	if err != nil {
		return fmt.Errorf("[Copying data to staging images for priming image: %v data by rendering] %v", img.VulkanHandle, err)
	}

	renderJobs := []*ipRenderJob{}
	for _, aspect := range p.sb.imageAspectFlagBits(img.ImageAspect) {
		for layer := uint32(0); layer < img.Info.ArrayLayers; layer++ {
			for level := uint32(0); level < img.Info.MipLevels; level++ {
				renderJobs = append(renderJobs, &ipRenderJob{
					inputAttachmentImages: copyJob.srcAspectsToDsts[aspect].dstImgs,
					renderTarget:          img,
					aspect:                aspect,
					layer:                 layer,
					level:                 level,
					finalLayout:           img.Info.Layout,
				})
			}
		}
	}
	for _, renderJob := range renderJobs {
		err := p.rh.render(renderJob, queue)
		if err != nil {
			log.E(p.sb.ctx, "[Priming image: %v, aspect: %v, layer: %v, level: %v data by rendering] %v",
				renderJob.renderTarget.VulkanHandle, renderJob.aspect, renderJob.layer, renderJob.level, err)
		}
	}

	return nil
}

func (p *imagePrimer) primeByShaderCopy(img *ImageObject) error {
	return fmt.Errorf("Not implemented")
}

func (p *imagePrimer) free() {
	p.rh.free()
}

// internal functions of image primer

func (p *imagePrimer) allocStagingImages(img *ImageObject, aspect VkImageAspectFlagBits) ([]*ImageObject, []*DeviceMemoryObject, error) {
	stagingImgs := []*ImageObject{}
	stagingMems := []*DeviceMemoryObject{}

	srcElementAndTexelInfo, err := subGetElementAndTexelBlockSize(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.oldState, GetState(p.sb.oldState), 0, nil, img.Info.Format)
	if err != nil {
		return []*ImageObject{}, []*DeviceMemoryObject{}, fmt.Errorf("[Getting element size and texel block info] %v", err)
	}
	if srcElementAndTexelInfo.TexelBlockSize.Width != uint32(1) || srcElementAndTexelInfo.TexelBlockSize.Height != uint32(1) {
		// compressed formats are not supported
		return []*ImageObject{}, []*DeviceMemoryObject{}, fmt.Errorf("allocating staging images for compressed format images is not supported")
	}
	srcElementSize := srcElementAndTexelInfo.ElementSize
	if aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT {
		srcElementSize, err = subGetDepthElementSize(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.oldState, GetState(p.sb.oldState), 0, nil, img.Info.Format, false)
		if err != nil {
			return []*ImageObject{}, []*DeviceMemoryObject{}, fmt.Errorf("[Getting element size for depth aspect] %v", err)
		}
	} else if aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT {
		srcElementSize = uint32(1)
	}

	stagingImgFormat := VkFormat_VK_FORMAT_UNDEFINED
	switch aspect {
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
		stagingImgFormat = VkFormat_VK_FORMAT_R32G32B32A32_UINT
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		stagingImgFormat = VkFormat_VK_FORMAT_R32_UINT
	}
	if stagingImgFormat == VkFormat_VK_FORMAT_UNDEFINED {
		return []*ImageObject{}, []*DeviceMemoryObject{}, fmt.Errorf("unsupported aspect: %v", aspect)
	}
	stagingElementInfo, _ := subGetElementAndTexelBlockSize(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.oldState, GetState(p.sb.oldState), 0, nil, stagingImgFormat)
	stagingElementSize := stagingElementInfo.ElementSize

	stagingInfo := img.Info
	stagingInfo.DedicatedAllocationNV = nil
	stagingInfo.Format = stagingImgFormat
	stagingInfo.Usage = VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT | VkImageUsageFlagBits_VK_IMAGE_USAGE_INPUT_ATTACHMENT_BIT | VkImageUsageFlagBits_VK_IMAGE_USAGE_SAMPLED_BIT)

	dev := p.sb.s.Devices.Get(img.Device)
	phyDevMemProps := p.sb.s.PhysicalDevices.Get(dev.PhysicalDevice).MemoryProperties
	memTypeBits := img.MemoryRequirements.MemoryTypeBits
	memIndex := memoryTypeIndexFor(memTypeBits, &phyDevMemProps, VkMemoryPropertyFlags(VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT))
	if memIndex < 0 {
		// fallback to use whatever type of memory available
		memIndex = memoryTypeIndexFor(memTypeBits, &phyDevMemProps, VkMemoryPropertyFlags(0))
	}
	if memIndex < 0 {
		return []*ImageObject{}, []*DeviceMemoryObject{}, fmt.Errorf("can't find an appropriate memory type index")
	}

	covered := uint32(0)
	for covered < srcElementSize {
		stagingImgHandle := VkImage(newUnusedID(true, func(x uint64) bool {
			return GetState(p.sb.newState).Images.Contains(VkImage(x))
		}))
		vkCreateImage(p.sb, dev.VulkanHandle, stagingInfo, stagingImgHandle)
		stagingImg := GetState(p.sb.newState).Images.Get(stagingImgHandle)
		// Query the memory requirements so validation layers are happy
		memReq := VkMemoryRequirements{}
		vkGetImageMemoryRequirements(p.sb, dev.VulkanHandle, stagingImgHandle, &memReq)

		stagingImgSize, err := subInferImageSize(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.oldState, GetState(p.sb.oldState), 0, nil, stagingImg)
		if err != nil {
			return []*ImageObject{}, []*DeviceMemoryObject{}, fmt.Errorf("[Getting staging image size] %v", err)
		}
		memHandle := VkDeviceMemory(newUnusedID(true, func(x uint64) bool {
			return GetState(p.sb.newState).DeviceMemories.Contains(VkDeviceMemory(x))
		}))
		// Since we cannot guess how much the driver will actually request of us,
		// overallocating by a factor of 2 should be enough.
		vkAllocateMemory(p.sb, dev.VulkanHandle, VkDeviceSize(stagingImgSize*2), uint32(memIndex), memHandle)
		mem := GetState(p.sb.newState).DeviceMemories.Get(memHandle)

		vkBindImageMemory(p.sb, dev.VulkanHandle, stagingImgHandle, memHandle, 0)
		stagingImgs = append(stagingImgs, stagingImg)
		stagingMems = append(stagingMems, mem)
		covered += stagingElementSize
	}
	return stagingImgs, stagingMems, nil
}

// Input attachment -> image render handler

type ipRenderJob struct {
	inputAttachmentImages []*ImageObject
	renderTarget          *ImageObject
	aspect                VkImageAspectFlagBits
	layer                 uint32
	level                 uint32
	finalLayout           VkImageLayout
}

type ipRenderPassInfo struct {
	numInputAttachments         int
	inputAttachmentImageFormat  VkFormat
	inputAttachmentImageSamples VkSampleCountFlagBits
	targetAspect                VkImageAspectFlagBits
	targetFormat                VkFormat
	targetSamples               VkSampleCountFlagBits
}

type ipPipelineInfo struct {
	fragShaderTicker imagePrimerShaderTicker
	pipelineLayout   VkPipelineLayout
	renderPassInfo   ipRenderPassInfo
}

type ipRenderHandler struct {
	sb *stateBuilder
	// descriptor set layouts indexed by different number of input attachment
	descriptorSetLayouts map[int]*DescriptorSetLayoutObject
	// pipeline layouts indexed by the number of input attachment in the only
	// descriptor set layout of the pipeline layout.
	pipelineLayouts map[int]*PipelineLayoutObject
	// pipelines indexed by the pipeline info.
	pipelines map[ipPipelineInfo]*GraphicsPipelineObject
	// shaders indexed by shader ticker
	shaders map[imagePrimerShaderTicker]*ShaderModuleObject
}

// interfaces to interact with image primer

func newImagePrimerRenderHandler(sb *stateBuilder) *ipRenderHandler {
	return &ipRenderHandler{
		sb:                   sb,
		descriptorSetLayouts: map[int]*DescriptorSetLayoutObject{},
		pipelineLayouts:      map[int]*PipelineLayoutObject{},
		pipelines:            map[ipPipelineInfo]*GraphicsPipelineObject{},
		shaders:              map[imagePrimerShaderTicker]*ShaderModuleObject{},
	}
}

func (h *ipRenderHandler) free() {
	for _, obj := range h.pipelines {
		h.sb.write(h.sb.cb.VkDestroyPipeline(obj.Device, obj.VulkanHandle, memory.Nullptr))
	}
	for _, obj := range h.pipelineLayouts {
		h.sb.write(h.sb.cb.VkDestroyPipelineLayout(obj.Device, obj.VulkanHandle, memory.Nullptr))
	}
	for _, obj := range h.descriptorSetLayouts {
		h.sb.write(h.sb.cb.VkDestroyDescriptorSetLayout(obj.Device, obj.VulkanHandle, memory.Nullptr))
	}
	for _, obj := range h.shaders {
		h.sb.write(h.sb.cb.VkDestroyShaderModule(obj.Device, obj.VulkanHandle, memory.Nullptr))
	}
}

func (h *ipRenderHandler) render(job *ipRenderJob, queue *QueueObject) error {
	if job.aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT {
		return fmt.Errorf("rendering to stencil aspect is not implemented")
	}

	outputBarrierAspect := VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT)
	outputPreRenderLayout := VkImageLayout_VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
	if job.aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT ||
		job.aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT {
		outputBarrierAspect = VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT | VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT)
		outputPreRenderLayout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
	}

	dev := job.renderTarget.Device
	descPool := h.createDescriptorPool(dev, len(job.inputAttachmentImages))
	if descPool != nil {
		defer h.sb.write(h.sb.cb.VkDestroyDescriptorPool(dev, descPool.VulkanHandle, memory.Nullptr))
	} else {
		return fmt.Errorf("failed to create descriptor pool for %v input attachments", len(job.inputAttachmentImages))
	}
	descSetLayout := h.getOrCreateDescriptorSetLayout(dev, len(job.inputAttachmentImages))
	descSet := h.allocDescriptorSet(dev, descPool.VulkanHandle, descSetLayout.VulkanHandle)
	if descSet != nil {
		defer func() {
			h.sb.write(h.sb.cb.VkFreeDescriptorSets(
				dev, descSet.DescriptorPool, uint32(1), NewVkDescriptorSetᶜᵖ(
					h.sb.MustAllocReadData(descSet.VulkanHandle).Ptr()), VkResult_VK_SUCCESS))
		}()
	} else {
		return fmt.Errorf("failed to allocate descriptorset with %v input attachments", len(job.inputAttachmentImages))
	}

	inputViews := []*ImageViewObject{}
	for _, input := range job.inputAttachmentImages {
		view := h.createImageView(dev, input, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, job.layer, job.level)
		inputViews = append(inputViews, view)
		if view != nil {
			defer h.sb.write(h.sb.cb.VkDestroyImageView(dev, view.VulkanHandle, memory.Nullptr))
		} else {
			return fmt.Errorf("failed to create image view for input attachment image: %v", input.VulkanHandle)
		}
	}
	outputView := h.createImageView(dev, job.renderTarget, job.aspect, job.layer, job.level)
	if outputView != nil {
		defer h.sb.write(h.sb.cb.VkDestroyImageView(dev, outputView.VulkanHandle, memory.Nullptr))
	} else {
		return fmt.Errorf("failed to create image view for rendering target image: %v", job.renderTarget.VulkanHandle)
	}

	imgInfoList := []VkDescriptorImageInfo{}
	for _, view := range inputViews {
		imgInfoList = append(imgInfoList, VkDescriptorImageInfo{
			VkSampler(0), view.VulkanHandle, VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
		})
	}

	writeDescriptorSet(h.sb, dev, descSet.VulkanHandle, 0, 0, VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT, imgInfoList, []VkDescriptorBufferInfo{}, []VkBufferView{})

	renderPassInfo := ipRenderPassInfo{
		numInputAttachments:         len(job.inputAttachmentImages),
		inputAttachmentImageFormat:  job.inputAttachmentImages[0].Info.Format,
		inputAttachmentImageSamples: job.inputAttachmentImages[0].Info.Samples,
		targetAspect:                job.aspect,
		targetFormat:                job.renderTarget.Info.Format,
		targetSamples:               job.renderTarget.Info.Samples,
	}
	renderPass := h.createRenderPass(dev, renderPassInfo, job.finalLayout)
	if renderPass != nil {
		defer h.sb.write(h.sb.cb.VkDestroyRenderPass(dev, renderPass.VulkanHandle, memory.Nullptr))
	} else {
		return fmt.Errorf("failed to create renderpass for rendering")
	}

	allViews := []VkImageView{}
	for _, view := range inputViews {
		allViews = append(allViews, view.VulkanHandle)
	}
	allViews = append(allViews, outputView.VulkanHandle)

	targetLevelSize := h.sb.levelSize(job.renderTarget.Info.Extent, job.renderTarget.Info.Format, job.level, job.aspect)

	framebuffer := h.createFramebuffer(dev, renderPass.VulkanHandle, allViews, uint32(targetLevelSize.width), uint32(targetLevelSize.height))
	if framebuffer != nil {
		defer h.sb.write(h.sb.cb.VkDestroyFramebuffer(dev, framebuffer.VulkanHandle, memory.Nullptr))
	} else {
		return fmt.Errorf("failed to create framebuffer for rendering")
	}

	pipelineLayout := h.getOrCreatePipelineLayout(dev, len(job.inputAttachmentImages))
	if pipelineLayout == nil {
		return fmt.Errorf("failed to get pipeline layout for the rendering")
	}

	fragShaderTicker := h.fragShaderTicker(job.renderTarget.Info.Format, job.aspect)
	if fragShaderTicker == ipRenderUnsupported {
		return fmt.Errorf("failed to get fragment shader code for rendering target in format: %v and aspect: %v", job.renderTarget.Info.Format, job.aspect)
	}
	pipelineInfo := ipPipelineInfo{
		fragShaderTicker: fragShaderTicker,
		pipelineLayout:   pipelineLayout.VulkanHandle,
		renderPassInfo:   renderPassInfo,
	}
	pipeline := h.getOrCreateGraphicsPipeline(dev, pipelineInfo, renderPass.VulkanHandle)
	if pipeline == nil {
		return fmt.Errorf("failed to get pipeline for the rendering")
	}

	var vc bytes.Buffer
	binary.Write(&vc, binary.LittleEndian, []float32{
		// positions, offset: 0 bytes
		1.0, 1.0, 0.0, -1.0, -1.0, 0.0, -1.0, 1.0, 0.0, 1.0, -1.0, 0.0,
	})
	vertexBuf, vertexBufMem := h.sb.allocAndFillScratchBuffer(h.sb.s.Devices.Get(dev), vc.Bytes(), VkBufferUsageFlagBits_VK_BUFFER_USAGE_VERTEX_BUFFER_BIT)
	defer h.sb.freeScratchBuffer(h.sb.s.Devices.Get(dev), vertexBuf, vertexBufMem)

	var ic bytes.Buffer
	binary.Write(&ic, binary.LittleEndian, []uint32{
		0, 1, 2, 0, 3, 1,
	})
	indexBuf, indexBufMem := h.sb.allocAndFillScratchBuffer(h.sb.s.Devices.Get(dev), ic.Bytes(), VkBufferUsageFlagBits_VK_BUFFER_USAGE_INDEX_BUFFER_BIT)
	defer h.sb.freeScratchBuffer(h.sb.s.Devices.Get(dev), indexBuf, indexBufMem)

	commandBuffer, commandPool := h.sb.getCommandBuffer(queue)

	inputBarriers := []VkImageMemoryBarrier{}
	for _, input := range job.inputAttachmentImages {
		inputBarriers = append(inputBarriers,
			VkImageMemoryBarrier{
				VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
				NewVoidᶜᵖ(memory.Nullptr),
				VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT - 1) | VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT),
				VkAccessFlags(VkAccessFlagBits_VK_ACCESS_INPUT_ATTACHMENT_READ_BIT),
				input.Info.Layout,
				VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
				queue.Family,
				queue.Family,
				input.VulkanHandle,
				VkImageSubresourceRange{
					VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT),
					0,
					input.Info.MipLevels,
					0,
					input.Info.ArrayLayers,
				},
			})
	}
	outputBarrier := VkImageMemoryBarrier{
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		NewVoidᶜᵖ(memory.Nullptr),
		VkAccessFlags(0),
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT),
		GetState(h.sb.newState).Images.Get(job.renderTarget.VulkanHandle).Info.Layout,
		outputPreRenderLayout,
		queue.Family,
		queue.Family,
		job.renderTarget.VulkanHandle,
		VkImageSubresourceRange{
			outputBarrierAspect,
			0,
			job.renderTarget.Info.MipLevels,
			0,
			job.renderTarget.Info.ArrayLayers,
		}}

	h.sb.write(h.sb.cb.VkCmdPipelineBarrier(
		commandBuffer,
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkDependencyFlags(0),
		uint32(0),
		memory.Nullptr,
		uint32(2),
		h.sb.MustAllocReadData(
			[]VkBufferMemoryBarrier{
				VkBufferMemoryBarrier{
					VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
					NewVoidᶜᵖ(memory.Nullptr),
					VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT - 1) | VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT),
					VkAccessFlags(VkAccessFlagBits_VK_ACCESS_VERTEX_ATTRIBUTE_READ_BIT),
					uint32(queue.Family),
					uint32(queue.Family),
					vertexBuf,
					0,
					VkDeviceSize(len(vc.Bytes())),
				},
				VkBufferMemoryBarrier{
					VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
					NewVoidᶜᵖ(memory.Nullptr),
					VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT - 1) | VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT),
					VkAccessFlags(VkAccessFlagBits_VK_ACCESS_INDEX_READ_BIT),
					uint32(queue.Family),
					uint32(queue.Family),
					indexBuf,
					0,
					VkDeviceSize(len(ic.Bytes())),
				},
			}).Ptr(),
		uint32(len(append(inputBarriers, outputBarrier))),
		h.sb.MustAllocReadData(append(inputBarriers, outputBarrier)).Ptr(),
	))

	h.sb.write(h.sb.cb.VkCmdBeginRenderPass(
		commandBuffer,
		h.sb.MustAllocReadData(
			VkRenderPassBeginInfo{
				VkStructureType_VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				renderPass.VulkanHandle,
				framebuffer.VulkanHandle,
				VkRect2D{
					VkOffset2D{int32(0), int32(0)},
					VkExtent2D{uint32(targetLevelSize.width), uint32(targetLevelSize.height)},
				},
				uint32(0),
				NewVkClearValueᶜᵖ(memory.Nullptr),
			}).Ptr(),
		VkSubpassContents(0),
	))

	h.sb.write(h.sb.cb.VkCmdBindPipeline(
		commandBuffer,
		VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS,
		pipeline.VulkanHandle,
	))

	// set dynamic states
	h.sb.write(h.sb.cb.VkCmdSetViewport(
		commandBuffer,
		uint32(0),
		uint32(1),
		NewVkViewportᶜᵖ(h.sb.MustAllocReadData(VkViewport{
			0.0, 0.0,
			float32(targetLevelSize.width), float32(targetLevelSize.height),
			0.0, 1.0,
		}).Ptr()),
	))
	h.sb.write(h.sb.cb.VkCmdSetScissor(
		commandBuffer,
		uint32(0),
		uint32(1),
		NewVkRect2Dᶜᵖ(h.sb.MustAllocReadData(VkRect2D{
			VkOffset2D{int32(0), int32(0)},
			VkExtent2D{uint32(targetLevelSize.width), uint32(targetLevelSize.height)},
		}).Ptr()),
	))

	if job.aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT {
		// TODO set stencil test dynamic state.
	}

	h.sb.write(h.sb.cb.VkCmdBindDescriptorSets(
		commandBuffer,
		VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS,
		pipelineLayout.VulkanHandle,
		0,
		1,
		h.sb.MustAllocReadData(descSet.VulkanHandle).Ptr(),
		0,
		NewU32ᶜᵖ(memory.Nullptr),
	))

	h.sb.write(h.sb.cb.VkCmdBindVertexBuffers(
		commandBuffer,
		0, 1,
		h.sb.MustAllocReadData(vertexBuf).Ptr(),
		h.sb.MustAllocReadData(VkDeviceSize(0)).Ptr(),
	))

	h.sb.write(h.sb.cb.VkCmdBindIndexBuffer(
		commandBuffer,
		indexBuf,
		VkDeviceSize(0),
		VkIndexType_VK_INDEX_TYPE_UINT32,
	))

	h.sb.write(h.sb.cb.VkCmdDrawIndexed(
		commandBuffer,
		6, 1, 0, 0, 0,
	))

	h.sb.write(h.sb.cb.VkCmdEndRenderPass(commandBuffer))

	h.sb.endSubmitAndDestroyCommandBuffer(queue, commandBuffer, commandPool)

	return nil
}

// internal functions for render handler

func (h *ipRenderHandler) createFramebuffer(dev VkDevice, renderPass VkRenderPass, imgViews []VkImageView, width, height uint32) *FramebufferObject {
	handle := VkFramebuffer(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).Framebuffers.Contains(VkFramebuffer(x))
	}))
	createInfo := VkFramebufferCreateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkFramebufferCreateFlags(0),
		renderPass,
		uint32(len(imgViews)),
		NewVkImageViewᶜᵖ(h.sb.MustAllocReadData(imgViews).Ptr()),
		width,
		height,
		1,
	}
	h.sb.write(h.sb.cb.VkCreateFramebuffer(
		dev,
		NewVkFramebufferCreateInfoᶜᵖ(h.sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	return GetState(h.sb.newState).Framebuffers.Get(handle)
}

func (h *ipRenderHandler) createImageView(dev VkDevice, img *ImageObject, aspect VkImageAspectFlagBits, layer, level uint32) *ImageViewObject {
	handle := VkImageView(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).ImageViews.Contains(VkImageView(x))
	}))
	h.sb.write(h.sb.cb.VkCreateImageView(
		dev,
		NewVkImageViewCreateInfoᶜᵖ(h.sb.MustAllocReadData(
			VkImageViewCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				VkImageViewCreateFlags(0),
				img.VulkanHandle,
				VkImageViewType_VK_IMAGE_VIEW_TYPE_2D,
				img.Info.Format,
				VkComponentMapping{
					VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
					VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
					VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
					VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
				},
				VkImageSubresourceRange{
					VkImageAspectFlags(aspect),
					level,
					1,
					layer,
					1,
				},
			}).Ptr()),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	return GetState(h.sb.newState).ImageViews.Get(handle)
}

func (h *ipRenderHandler) allocDescriptorSet(dev VkDevice, pool VkDescriptorPool, layout VkDescriptorSetLayout) *DescriptorSetObject {
	handle := VkDescriptorSet(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).DescriptorSets.Contains(VkDescriptorSet(x))
	}))
	h.sb.write(h.sb.cb.VkAllocateDescriptorSets(
		dev,
		h.sb.MustAllocReadData(VkDescriptorSetAllocateInfo{
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
			NewVoidᶜᵖ(memory.Nullptr),
			pool,
			1,
			NewVkDescriptorSetLayoutᶜᵖ(h.sb.MustAllocReadData(layout).Ptr()),
		}).Ptr(),
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	return GetState(h.sb.newState).DescriptorSets.Get(handle)
}

func (h *ipRenderHandler) createDescriptorPool(dev VkDevice, numInputAttachments int) *DescriptorPoolObject {
	handle := VkDescriptorPool(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).DescriptorPools.Contains(VkDescriptorPool(x))
	}))
	vkCreateDescriptorPool(h.sb, dev, VkDescriptorPoolCreateFlags(
		VkDescriptorPoolCreateFlagBits_VK_DESCRIPTOR_POOL_CREATE_FREE_DESCRIPTOR_SET_BIT),
		1, []VkDescriptorPoolSize{VkDescriptorPoolSize{
			VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT,
			uint32(numInputAttachments),
		}},
		handle)
	return GetState(h.sb.newState).DescriptorPools.Get(handle)
}

func (h *ipRenderHandler) createRenderPass(dev VkDevice, info ipRenderPassInfo, finalLayout VkImageLayout) *RenderPassObject {
	inputAttachmentRefs := make([]VkAttachmentReference, info.numInputAttachments)
	inputAttachmentDescs := make([]VkAttachmentDescription, info.numInputAttachments)
	for i := 0; i < info.numInputAttachments; i++ {
		inputAttachmentRefs[i] = VkAttachmentReference{
			uint32(i), VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
		}
		inputAttachmentDescs[i] = VkAttachmentDescription{
			VkAttachmentDescriptionFlags(0),
			info.inputAttachmentImageFormat,
			info.inputAttachmentImageSamples,
			VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD,
			VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE,
			VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE,
			VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE,
			VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
			VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
		}
	}
	outputAttachmentRef := VkAttachmentReference{
		uint32(info.numInputAttachments),
		// The layout will be set later according to the image apsect bits.
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
	}
	outputAttachmentDesc := VkAttachmentDescription{
		VkAttachmentDescriptionFlags(0),
		info.targetFormat,
		info.targetSamples,
		VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE,
		VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE,
		VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE,
		VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE,
		// The layout will be set later according to the image aspect bit.
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
		finalLayout,
	}
	subpassDesc := VkSubpassDescription{
		VkSubpassDescriptionFlags(0),
		VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS,
		uint32(info.numInputAttachments),
		NewVkAttachmentReferenceᶜᵖ(h.sb.MustAllocReadData(inputAttachmentRefs).Ptr()),
		uint32(0),
		// color/depthstencil attachments will be set later according to the
		// aspect bit.
		NewVkAttachmentReferenceᶜᵖ(memory.Nullptr),
		NewVkAttachmentReferenceᶜᵖ(memory.Nullptr),
		NewVkAttachmentReferenceᶜᵖ(memory.Nullptr),
		uint32(0),
		NewU32ᶜᵖ(memory.Nullptr),
	}
	switch info.targetAspect {
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
		outputAttachmentRef.Layout = VkImageLayout_VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
		outputAttachmentDesc.InitialLayout = VkImageLayout_VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
		subpassDesc.ColorAttachmentCount = uint32(1)
		subpassDesc.PColorAttachments = NewVkAttachmentReferenceᶜᵖ(h.sb.MustAllocReadData(outputAttachmentRef).Ptr())
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		outputAttachmentRef.Layout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
		outputAttachmentDesc.InitialLayout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
		subpassDesc.PDepthStencilAttachment = NewVkAttachmentReferenceᶜᵖ(h.sb.MustAllocReadData(outputAttachmentRef).Ptr())
	default:
		return nil
	}

	createInfo := VkRenderPassCreateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkRenderPassCreateFlags(0),
		uint32(info.numInputAttachments + 1),
		NewVkAttachmentDescriptionᶜᵖ(h.sb.MustAllocReadData(
			append(inputAttachmentDescs, outputAttachmentDesc),
		).Ptr()),
		uint32(1),
		NewVkSubpassDescriptionᶜᵖ(h.sb.MustAllocReadData(subpassDesc).Ptr()),
		uint32(0),
		NewVkSubpassDependencyᶜᵖ(memory.Nullptr),
	}

	handle := VkRenderPass(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).RenderPasses.Contains(VkRenderPass(x))
	}))

	h.sb.write(h.sb.cb.VkCreateRenderPass(
		dev,
		NewVkRenderPassCreateInfoᶜᵖ(h.sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))

	return GetState(h.sb.newState).RenderPasses.Get(handle)
}

func (h *ipRenderHandler) getOrCreateGraphicsPipeline(dev VkDevice, info ipPipelineInfo, renderPass VkRenderPass) *GraphicsPipelineObject {
	if p, ok := h.pipelines[info]; ok {
		return p
	}

	vertShader := h.getOrCreateShaderModule(dev, ipRenderVert)
	if vertShader == nil {
		return nil
	}
	fragShader := h.getOrCreateShaderModule(dev, info.fragShaderTicker)
	if fragShader == nil {
		return nil
	}

	depthTestEnable := VkBool32(0)
	depthWriteEnable := VkBool32(0)
	numColorAttachments := uint32(1)
	if info.renderPassInfo.targetAspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT {
		depthTestEnable = VkBool32(1)
		depthWriteEnable = VkBool32(1)
		numColorAttachments = uint32(0)
	}
	stencilTestEnable := VkBool32(0)
	dynamicStates := []VkDynamicState{
		VkDynamicState_VK_DYNAMIC_STATE_VIEWPORT,
		VkDynamicState_VK_DYNAMIC_STATE_SCISSOR,
	}
	if info.renderPassInfo.targetAspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT {
		stencilTestEnable = VkBool32(1)
		dynamicStates = append(dynamicStates,
			VkDynamicState_VK_DYNAMIC_STATE_STENCIL_WRITE_MASK,
			VkDynamicState_VK_DYNAMIC_STATE_STENCIL_REFERENCE,
		)
		numColorAttachments = uint32(0)
	}

	depethStencilState := VkPipelineDepthStencilStateCreateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkPipelineDepthStencilStateCreateFlags(0),
		depthTestEnable,
		depthWriteEnable,
		VkCompareOp_VK_COMPARE_OP_ALWAYS, // depthCompareOp
		VkBool32(0),                      // depthBoundsTestEnable
		stencilTestEnable,
		VkStencilOpState{
			VkStencilOp_VK_STENCIL_OP_KEEP,    // failOp
			VkStencilOp_VK_STENCIL_OP_REPLACE, // passOp
			VkStencilOp_VK_STENCIL_OP_REPLACE, // depthFailOp
			VkCompareOp_VK_COMPARE_OP_ALWAYS,  // compareOp
			0xFF, // compareMask
			// write mask and reference must be set dynamically
			0, // writeMask
			0, // reference
		}, // front
		VkStencilOpState{}, // back
		0.0, 0.0,
	}

	createInfo := VkGraphicsPipelineCreateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkPipelineCreateFlags(0),
		uint32(2),
		NewVkPipelineShaderStageCreateInfoᶜᵖ(h.sb.MustAllocReadData(
			[]VkPipelineShaderStageCreateInfo{
				VkPipelineShaderStageCreateInfo{
					VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
					NewVoidᶜᵖ(memory.Nullptr),
					VkPipelineShaderStageCreateFlags(0),
					VkShaderStageFlagBits_VK_SHADER_STAGE_VERTEX_BIT,
					vertShader.VulkanHandle,
					NewCharᶜᵖ(h.sb.MustAllocReadData("main").Ptr()),
					NewVkSpecializationInfoᶜᵖ(memory.Nullptr),
				},
				VkPipelineShaderStageCreateInfo{
					VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
					NewVoidᶜᵖ(memory.Nullptr),
					VkPipelineShaderStageCreateFlags(0),
					VkShaderStageFlagBits_VK_SHADER_STAGE_FRAGMENT_BIT,
					fragShader.VulkanHandle,
					NewCharᶜᵖ(h.sb.MustAllocReadData("main").Ptr()),
					NewVkSpecializationInfoᶜᵖ(memory.Nullptr),
				},
			}).Ptr()),
		NewVkPipelineVertexInputStateCreateInfoᶜᵖ(h.sb.MustAllocReadData(
			VkPipelineVertexInputStateCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				VkPipelineVertexInputStateCreateFlags(0),
				uint32(1),
				NewVkVertexInputBindingDescriptionᶜᵖ(h.sb.MustAllocReadData(
					[]VkVertexInputBindingDescription{
						VkVertexInputBindingDescription{0, 12, 0},
					}).Ptr()),
				uint32(1),
				NewVkVertexInputAttributeDescriptionᶜᵖ(h.sb.MustAllocReadData(
					[]VkVertexInputAttributeDescription{
						VkVertexInputAttributeDescription{0, 0, VkFormat_VK_FORMAT_R32G32B32_SFLOAT, 0},
					}).Ptr()),
			}).Ptr()),
		NewVkPipelineInputAssemblyStateCreateInfoᶜᵖ(h.sb.MustAllocReadData(
			VkPipelineInputAssemblyStateCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				VkPipelineInputAssemblyStateCreateFlags(0),
				VkPrimitiveTopology_VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
				VkBool32(0),
			}).Ptr()),
		NewVkPipelineTessellationStateCreateInfoᶜᵖ(memory.Nullptr),
		NewVkPipelineViewportStateCreateInfoᶜᵖ(h.sb.MustAllocReadData(
			VkPipelineViewportStateCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				VkPipelineViewportStateCreateFlags(0),
				uint32(1),
				// set viewport dynamically
				NewVkViewportᶜᵖ(memory.Nullptr),
				uint32(1),
				// set scissor dynamically
				NewVkRect2Dᶜᵖ(memory.Nullptr),
			}).Ptr()),
		NewVkPipelineRasterizationStateCreateInfoᶜᵖ(h.sb.MustAllocReadData(
			VkPipelineRasterizationStateCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				VkPipelineRasterizationStateCreateFlags(0),
				VkBool32(0),
				VkBool32(0),
				VkPolygonMode_VK_POLYGON_MODE_FILL,
				VkCullModeFlags(VkCullModeFlagBits_VK_CULL_MODE_BACK_BIT),
				VkFrontFace_VK_FRONT_FACE_COUNTER_CLOCKWISE,
				VkBool32(0),
				0.0, 0.0, 0.0, 1.0,
			}).Ptr()),
		NewVkPipelineMultisampleStateCreateInfoᶜᵖ(h.sb.MustAllocReadData(
			VkPipelineMultisampleStateCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				VkPipelineMultisampleStateCreateFlags(0),
				VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT,
				VkBool32(0),
				0.0,
				NewVkSampleMaskᶜᵖ(memory.Nullptr),
				VkBool32(0),
				VkBool32(0),
			}).Ptr()),
		NewVkPipelineDepthStencilStateCreateInfoᶜᵖ(h.sb.MustAllocReadData(depethStencilState).Ptr()),
		NewVkPipelineColorBlendStateCreateInfoᶜᵖ(h.sb.MustAllocReadData(
			VkPipelineColorBlendStateCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				VkPipelineColorBlendStateCreateFlags(0),
				VkBool32(0),
				VkLogicOp_VK_LOGIC_OP_CLEAR,
				numColorAttachments,
				// there is at most one color attachment
				NewVkPipelineColorBlendAttachmentStateᶜᵖ(h.sb.MustAllocReadData(
					VkPipelineColorBlendAttachmentState{
						VkBool32(0),
						VkBlendFactor_VK_BLEND_FACTOR_ZERO,
						VkBlendFactor_VK_BLEND_FACTOR_ONE,
						VkBlendOp_VK_BLEND_OP_ADD,
						VkBlendFactor_VK_BLEND_FACTOR_ZERO,
						VkBlendFactor_VK_BLEND_FACTOR_ONE,
						VkBlendOp_VK_BLEND_OP_ADD,
						VkColorComponentFlags(0xf),
					}).Ptr()),
				F32ː4ᵃ{0.0, 0.0, 0.0, 0.0},
			}).Ptr()),
		NewVkPipelineDynamicStateCreateInfoᶜᵖ(h.sb.MustAllocReadData(
			VkPipelineDynamicStateCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				VkPipelineDynamicStateCreateFlags(0),
				uint32(len(dynamicStates)),
				NewVkDynamicStateᶜᵖ(h.sb.MustAllocReadData(dynamicStates).Ptr()),
			}).Ptr()),
		info.pipelineLayout,
		renderPass,
		uint32(0),
		VkPipeline(0),
		int32(0),
	}

	handle := VkPipeline(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).GraphicsPipelines.Contains(VkPipeline(x))
	}))

	h.sb.write(h.sb.cb.VkCreateGraphicsPipelines(
		dev, VkPipelineCache(0), uint32(1),
		NewVkGraphicsPipelineCreateInfoᶜᵖ(h.sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr, h.sb.MustAllocWriteData(handle).Ptr(), VkResult_VK_SUCCESS,
	))

	h.pipelines[info] = GetState(h.sb.newState).GraphicsPipelines.Get(handle)
	return h.pipelines[info]
}

func (h *ipRenderHandler) getOrCreateShaderModule(dev VkDevice, ticker imagePrimerShaderTicker) *ShaderModuleObject {
	if m, ok := h.shaders[ticker]; ok {
		return m
	}
	handle := VkShaderModule(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).ShaderModules.Contains(VkShaderModule(x))
	}))
	code := shaderWords(ticker)
	if len(code) == 0 {
		return nil
	}
	vkCreateShaderModule(h.sb, dev, code, handle)
	h.shaders[ticker] = GetState(h.sb.newState).ShaderModules.Get(handle)
	return h.shaders[ticker]
}

func (h *ipRenderHandler) getOrCreatePipelineLayout(dev VkDevice, numInputAttachment int) *PipelineLayoutObject {
	if l, ok := h.pipelineLayouts[numInputAttachment]; ok {
		return l
	}
	handle := VkPipelineLayout(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).PipelineLayouts.Contains(VkPipelineLayout(x))
	}))
	descriptorSet := h.getOrCreateDescriptorSetLayout(dev, numInputAttachment)
	vkCreatePipelineLayout(h.sb, dev, []VkDescriptorSetLayout{descriptorSet.VulkanHandle}, []VkPushConstantRange{}, handle)
	h.pipelineLayouts[numInputAttachment] = GetState(h.sb.newState).PipelineLayouts.Get(handle)
	return h.pipelineLayouts[numInputAttachment]
}

func (h *ipRenderHandler) getOrCreateDescriptorSetLayout(dev VkDevice, numInputAttachment int) *DescriptorSetLayoutObject {
	if l, ok := h.descriptorSetLayouts[numInputAttachment]; ok {
		return l
	}

	handle := VkDescriptorSetLayout(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).DescriptorSetLayouts.Contains(VkDescriptorSetLayout(x))
	}))

	bindings := []VkDescriptorSetLayoutBinding{VkDescriptorSetLayoutBinding{
		0, VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT,
		uint32(numInputAttachment),
		VkShaderStageFlags(VkShaderStageFlagBits_VK_SHADER_STAGE_FRAGMENT_BIT),
		NewVkSamplerᶜᵖ(memory.Nullptr),
	}}
	vkCreateDescriptorSetLayout(h.sb, dev, bindings, handle)
	h.descriptorSetLayouts[numInputAttachment] = GetState(h.sb.newState).DescriptorSetLayouts.Get(handle)
	return h.descriptorSetLayouts[numInputAttachment]
}

// Buffer->Image copy session

// ipBufCopyJob describes how the data in the src image to be copied to dst
// images, i.e. which aspect of the src image should be copied to which aspect
// of which dst image, and the final layout of the dst images. Note that the
// source of the data is the state block of the source image (data owner), not
// the VkImage handle, so such a copy does not modify the state of the src image
type ipBufCopyJob struct {
	srcAspectsToDsts map[VkImageAspectFlagBits]*ipBufCopyDst
	finalLayout      VkImageLayout
	srcImg           *ImageObject
}

// ipBufCopyDst contains a list of dst images whose dst aspect will be written
// by a serial of image copy operations.
type ipBufCopyDst struct {
	dstImgs     []*ImageObject
	dstAspect   VkImageAspectFlagBits
	finalLayout VkImageLayout
}

func newImagePrimerBufCopyJob(srcImg *ImageObject, finalLayout VkImageLayout) *ipBufCopyJob {
	return &ipBufCopyJob{
		srcAspectsToDsts: map[VkImageAspectFlagBits]*ipBufCopyDst{},
		finalLayout:      finalLayout,
		srcImg:           srcImg,
	}
}

func (s *ipBufCopyJob) addDst(srcAspect, dstAspect VkImageAspectFlagBits, dstImgs ...*ImageObject) error {
	if s.srcAspectsToDsts[srcAspect] == nil {
		s.srcAspectsToDsts[srcAspect] = &ipBufCopyDst{
			dstImgs:   []*ImageObject{},
			dstAspect: dstAspect,
		}
	}
	if s.srcAspectsToDsts[srcAspect].dstAspect != dstAspect {
		return fmt.Errorf("new dstAspect:%v does not match with the existing one: %v", dstAspect, s.srcAspectsToDsts[srcAspect].dstAspect)
	}
	s.srcAspectsToDsts[srcAspect].dstImgs = append(s.srcAspectsToDsts[srcAspect].dstImgs, dstImgs...)
	return nil
}

type ipBufferCopySession struct {
	copies  map[*ImageObject][]VkBufferImageCopy
	content []uint8
	job     *ipBufCopyJob
	sb      *stateBuilder
}

// interfaces to interact with image primer

func newImagePrimerBufferCopySession(sb *stateBuilder, job *ipBufCopyJob) *ipBufferCopySession {
	h := &ipBufferCopySession{
		copies:  map[*ImageObject][]VkBufferImageCopy{},
		content: []uint8{},
		job:     job,
		sb:      sb,
	}
	for _, dst := range job.srcAspectsToDsts {
		for _, img := range dst.dstImgs {
			h.copies[img] = []VkBufferImageCopy{}
		}
	}
	return h
}

func (h *ipBufferCopySession) collectCopiesFromSubresourceRange(srcRng VkImageSubresourceRange) {
	offset := uint64(len(h.content))
	for _, aspect := range h.sb.imageAspectFlagBits(srcRng.AspectMask) {
		for i := uint32(0); i < srcRng.LevelCount; i++ {
			level := srcRng.BaseMipLevel + i
			levelSize := h.sb.levelSize(h.job.srcImg.Info.Extent, h.job.srcImg.Info.Format, level, aspect)
			extent := VkExtent3D{
				uint32(levelSize.width),
				uint32(levelSize.height),
				uint32(levelSize.depth),
			}
			for j := uint32(0); j < srcRng.LayerCount; j++ {
				layer := srcRng.BaseArrayLayer + j
				for dstIndex, dstImg := range h.job.srcAspectsToDsts[aspect].dstImgs {
					// dstIndex is reserved for handling wide channel image format
					// like R64G64B64A64
					// TODO: handle wide format
					_ = dstIndex
					data, bufImgCopy, err := h.getCopyAndData(dstImg, h.job.srcAspectsToDsts[aspect].dstAspect, h.job.srcImg, aspect, layer, level, VkOffset3D{0, 0, 0}, extent, offset)
					if err != nil {
						log.E(h.sb.ctx, "[Getting VkBufferImageCopy and raw data for priming data at image: %v, aspect: %v, layer: %v, level: %v] %v", h.job.srcImg.VulkanHandle, aspect, layer, level, err)
						continue
					}
					h.copies[dstImg] = append(h.copies[dstImg], bufImgCopy)
					h.content = append(h.content, data...)
					offset += uint64(len(data))
				}
			}
		}
	}
}

func (h *ipBufferCopySession) collectCopiesFromSparseImageBindings() {
	offset := uint64(len(h.content))
	for _, aspect := range h.sb.imageAspectFlagBits(h.job.srcImg.ImageAspect) {
		bindings, ok := (*h.job.srcImg.SparseImageMemoryBindings.Map)[uint32(aspect)]
		if !ok {
			log.E(h.sb.ctx, "Does not find sparse image binding for image: %v, aspect: %v", h.job.srcImg.VulkanHandle, aspect)
			continue
		}
		for layer, layerData := range *bindings.Layers.Map {
			for level, levelData := range *layerData.Levels.Map {
				for _, blockData := range *levelData.Blocks.Map {
					for dstIndex, dstImg := range h.job.srcAspectsToDsts[aspect].dstImgs {
						// dstIndex is reserved for handling wide channel image format
						// TODO: handle wide format
						_ = dstIndex
						data, bufImgCopy, err := h.getCopyAndData(dstImg, h.job.srcAspectsToDsts[aspect].dstAspect, h.job.srcImg, aspect, layer, level, blockData.Offset, blockData.Extent, offset)
						if err != nil {
							log.E(h.sb.ctx, "[Getting VkBufferImageCopy and raw data from sparse image binding at image: %v, aspect: %v, layer: %v, level: %v, offset: %v, extent: %v] %v", h.job.srcImg.VulkanHandle, aspect, layer, level, blockData.Offset, blockData.Extent, err)
							continue
						}
						h.copies[dstImg] = append(h.copies[dstImg], bufImgCopy)
						h.content = append(h.content, data...)
						offset += uint64(len(data))
					}
				}
			}
		}
	}
}

func (h *ipBufferCopySession) rolloutBufCopies(submissionQueue *QueueObject, dstImgsOldQueue *QueueObject) error {
	errMsg := "[Submit buf -> img copy commands]"
	if len(h.content) == 0 {
		return fmt.Errorf("%s no valid data to copy", errMsg)
	}

	scratchBuffer, scratchMemory := h.sb.allocAndFillScratchBuffer(h.sb.s.Devices.Get(h.job.srcImg.Device), h.content, VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT)
	defer h.sb.freeScratchBuffer(h.sb.s.Devices.Get(h.job.srcImg.Device), scratchBuffer, scratchMemory)

	commandBuffer, commandPool := h.sb.getCommandBuffer(submissionQueue)
	defer h.sb.endSubmitAndDestroyCommandBuffer(submissionQueue, commandBuffer, commandPool)

	oldQueueFamilyIndex := uint32(submissionQueue.Family)
	if dstImgsOldQueue != nil {
		oldQueueFamilyIndex = uint32(dstImgsOldQueue.Family)
	}
	dstImgBarriers := []VkImageMemoryBarrier{}
	for _, dst := range h.job.srcAspectsToDsts {
		for _, dstImg := range dst.dstImgs {
			barrier := VkImageMemoryBarrier{
				VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
				NewVoidᶜᵖ(memory.Nullptr),
				VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT - 1) | VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT),
				VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT - 1) | VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT),
				VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				oldQueueFamilyIndex,
				uint32(submissionQueue.Family),
				dstImg.VulkanHandle,
				VkImageSubresourceRange{
					VkImageAspectFlags(dst.dstAspect),
					0,
					dstImg.Info.MipLevels,
					0,
					dstImg.Info.ArrayLayers,
				},
			}
			dstImgBarriers = append(dstImgBarriers, barrier)
		}
	}

	h.sb.write(h.sb.cb.VkCmdPipelineBarrier(
		commandBuffer,
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkDependencyFlags(0),
		uint32(0),
		memory.Nullptr,
		uint32(1),
		h.sb.MustAllocReadData(
			VkBufferMemoryBarrier{
				VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
				NewVoidᶜᵖ(memory.Nullptr),
				VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT - 1) | VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT),
				VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT - 1) | VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT),
				uint32(submissionQueue.Family),
				uint32(submissionQueue.Family),
				scratchBuffer,
				0,
				VkDeviceSize(len(h.content)),
			}).Ptr(),
		uint32(len(dstImgBarriers)),
		h.sb.MustAllocReadData(dstImgBarriers).Ptr(),
	))

	for _, dst := range h.job.srcAspectsToDsts {
		for _, dstImg := range dst.dstImgs {
			h.sb.write(h.sb.cb.VkCmdCopyBufferToImage(
				commandBuffer,
				scratchBuffer,
				dstImg.VulkanHandle,
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				uint32(len(h.copies[dstImg])),
				h.sb.MustAllocReadData(h.copies[dstImg]).Ptr(),
			))
		}
	}

	for _, barrier := range dstImgBarriers {
		barrier.OldLayout = VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
		barrier.NewLayout = h.job.finalLayout
		barrier.SrcQueueFamilyIndex = uint32(submissionQueue.Family)
	}

	h.sb.write(h.sb.cb.VkCmdPipelineBarrier(
		commandBuffer,
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkDependencyFlags(0),
		uint32(0),
		memory.Nullptr,
		uint32(0),
		memory.Nullptr,
		uint32(len(dstImgBarriers)),
		h.sb.MustAllocReadData(dstImgBarriers).Ptr(),
	))

	return nil
}

// internal functions of ipBufferCopSessionr

func (h *ipBufferCopySession) getCopyAndData(dstImg *ImageObject, dstAspect VkImageAspectFlagBits, srcImg *ImageObject, srcAspect VkImageAspectFlagBits, layer, level uint32, blockOffset VkOffset3D, blockExtent VkExtent3D, bufDataOffset uint64) ([]uint8, VkBufferImageCopy, error) {
	var err error
	bufImgCopy := VkBufferImageCopy{
		VkDeviceSize(bufDataOffset), 0, 0,
		VkImageSubresourceLayers{
			VkImageAspectFlags(dstAspect),
			level, layer, 1},
		blockOffset, blockExtent,
	}
	srcImgDataOffset := uint64(h.sb.levelSize(VkExtent3D{
		uint32(blockOffset.X),
		uint32(blockOffset.Y),
		uint32(blockOffset.Z),
	}, srcImg.Info.Format, 0, srcAspect).levelSize)
	srcImgDataSizeInBytes := uint64(h.sb.levelSize(
		blockExtent,
		srcImg.Info.Format,
		0, srcAspect).levelSize)

	data := srcImg.Aspects.Get(srcAspect).Layers.Get(layer).Levels.Get(level).Data.Slice(srcImgDataOffset, srcImgDataOffset+srcImgDataSizeInBytes, h.sb.oldState.MemoryLayout).MustRead(h.sb.ctx, nil, h.sb.oldState, nil)

	unpacked := data
	if dstImg.Info.Format != srcImg.Info.Format {
		// dstImg format is different with the srcImage format, the dst image
		// should be a staging image.
		srcVkFmt := srcImg.Info.Format
		if srcImg.Info.Format == VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32 {
			data, err = ebgrDataToRGB32SFloat(data, blockExtent)
			if err != nil {
				return []uint8{}, bufImgCopy, fmt.Errorf("[Converting data in VK_FORMAT_E5B9G9R9_UFLOAT_PACK32 to VK_FORMAT_R32G32B32_SFLOAT] %v", err)
			}
			srcVkFmt = VkFormat_VK_FORMAT_R32G32B32_SFLOAT
		}
		var srcFmt *image.Format
		switch srcAspect {
		case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
			srcFmt, err = getImageFormatFromVulkanFormat(srcVkFmt)
			if err != nil {
				return []uint8{}, bufImgCopy, fmt.Errorf("[Getting image.Format for VkFormat: %v, aspect: %v] %v", srcVkFmt, srcAspect, err)
			}
		case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT:
			srcFmt, err = getDepthImageFormatFromVulkanFormat(srcVkFmt)
			if err != nil {
				return []uint8{}, bufImgCopy, fmt.Errorf("[Getting image.Format for VkFormat: %v, aspect: %v] %v", srcVkFmt, srcAspect, err)
			}
		case VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
			srcFmt, err = getImageFormatFromVulkanFormat(VkFormat_VK_FORMAT_S8_UINT)
			if err != nil {
				return []uint8{}, bufImgCopy, fmt.Errorf("[Getting image.Format for VkFormat: %v, aspect: %v] %v", srcVkFmt, srcAspect, err)
			}
		}
		dstFmt, err := getImageFormatFromVulkanFormat(dstImg.Info.Format)
		if err != nil {
			return []uint8{}, bufImgCopy, fmt.Errorf("[Getting image.Format for VkFormat: %v] %v", dstImg.Info.Format, err)
		}
		unpacked, err = unpackData(data, srcFmt, dstFmt)
		if err != nil {
			return []uint8{}, bufImgCopy, fmt.Errorf("[Unpacking data from format: %v to format: %v] %v", srcFmt, dstFmt, err)
		}

	} else if srcAspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT {
		// srcImg format is the same to the dstImage format, the data is ready to
		// be used directly, except when the src image is a dpeth 24 UNORM one.
		if (srcImg.Info.Format == VkFormat_VK_FORMAT_D24_UNORM_S8_UINT) ||
			(srcImg.Info.Format == VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32) {
			srcFmt, err := getDepthImageFormatFromVulkanFormat(srcImg.Info.Format)
			if err != nil {
				return []uint8{}, bufImgCopy, fmt.Errorf("[Getting image.Format for VkFormat: %v, aspect: %v] %v", srcImg.Info.Format, srcAspect, err)
			}
			dstFmt, err := getImageFormatFromVulkanFormat(VkFormat_VK_FORMAT_R32_UINT)
			if err != nil {
				return []uint8{}, bufImgCopy, fmt.Errorf("[Getting image.Format for VkFormat: VK_FORMAT_R32_UINT] %v", err)
			}
			unpacked, err = unpackData(data, srcFmt, dstFmt)
			if err != nil {
				return []uint8{}, bufImgCopy, fmt.Errorf("[Unpacking data from format: %v to format: %v] %v", srcFmt, dstFmt, err)
			}
		}
	}

	extendToMultipleOf8(&unpacked)

	dstLevelSize := h.sb.levelSize(blockExtent, dstImg.Info.Format, 0, dstAspect)
	if uint64(len(unpacked)) != dstLevelSize.alignedLevelSizeInBuf {
		return []uint8{}, bufImgCopy, fmt.Errorf("size of unpacked data does not match expectation, actual: %v, expected: %v, srcFmt: %v, dstFmt: %v", len(unpacked), dstLevelSize.alignedLevelSizeInBuf, srcImg.Info.Format, dstImg.Info.Format)
	}

	return unpacked, bufImgCopy, nil
}

// free functions

func extendToMultipleOf8(dataPtr *[]uint8) {
	l := uint64(len(*dataPtr))
	nl := nextMultipleOf8(l)
	zeros := make([]uint8, nl-l)
	*dataPtr = append(*dataPtr, zeros...)
}

func unpackData(data []uint8, srcFmt, dstFmt *image.Format) ([]uint8, error) {
	var err error
	if srcFmt.GetUncompressed() == nil {
		return []uint8{}, fmt.Errorf("compressed format: %v is not supported", srcFmt)
	}
	if dstFmt.GetUncompressed() == nil {
		return []uint8{}, fmt.Errorf("compressed format: %v is not supported", dstFmt)
	}
	sf := proto.Clone(srcFmt).(*image.Format).GetUncompressed().GetFormat()
	df := proto.Clone(dstFmt).(*image.Format).GetUncompressed().GetFormat()

	// The casting rule is described as below:
	// If the data layout is UNORM, unsigned extends the src data to uint32
	// If the data layout is SNORM, signed extends the src data to sint32
	// If the data layout is UINT, unsigned extends the src data to uint32
	// If the data layout is SINT, signed extends the src data to sint32
	// If the data layout is FLOAT, cast the src data to sfloat32
	// Note that, the staging image formats are always UINT32, the data within
	// the staging image should be encoded as float32, if the source data is
	// in float point type. The data will be bitcasted to float32 in the shader
	// when rendering to the state block image in the replay side.
	// If the source data is in normalized type, it will be treated as integer,
	// and will be normalized in the shader when rendering in the replay side.
	// Also, to keep data in SRGB untouched, the sampling curve of the source
	// format will be changed to linear.

	// Modify the src and dst format stream to follow the rule above.
	for _, sc := range sf.Components {
		if sc.Channel == stream.Channel_Depth || sc.Channel == stream.Channel_Stencil {
			sc.Channel = stream.Channel_Red
		}
		dc, _ := df.Component(sc.Channel)
		if dc == nil {
			return []uint8{}, fmt.Errorf("[Building src format: %v] unsuppored channel in source data format: %v", sf, sc.Channel)
		}
		sc.Sampling = stream.Linear
		if sc.GetDataType().GetInteger() != nil {
			dc.DataType = &stream.U32
			dc.Sampling = stream.Linear
			if sc.GetDataType().GetSigned() {
				dc.DataType = &stream.S32
			}
		} else if sc.GetDataType().GetFloat() != nil {
			dc.DataType = &stream.F32
			dc.Sampling = stream.Linear
		} else {
			return []uint8{}, fmt.Errorf("[Building dst format for: %v] %s", sf, "DataType other than stream.Integer and stream.Float are not handled.")
		}
	}

	converted, err := stream.Convert(df, sf, data)
	if err != nil {
		return []uint8{}, fmt.Errorf("[Converting data from %v to %v] %v", sf, df, err)
	}
	return converted, nil
}

func ebgrDataToRGB32SFloat(data []uint8, extent VkExtent3D) ([]uint8, error) {
	sf, err := getImageFormatFromVulkanFormat(VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32)
	if err != nil {
		return []uint8{}, err
	}
	df, err := getImageFormatFromVulkanFormat(VkFormat_VK_FORMAT_R32G32B32_SFLOAT)
	if err != nil {
		return []uint8{}, err
	}
	retData, err := image.Convert(data, int(extent.Width), int(extent.Height), int(extent.Depth), sf, df)
	if err != nil {
		return []uint8{}, err
	}
	return retData, nil
}

func denseBound(img *ImageObject) bool {
	return img.BoundMemory != nil
}

func sparseBound(img *ImageObject) bool {
	return len(*img.SparseImageMemoryBindings.Map) > 0 || len(*img.OpaqueSparseMemoryBindings.Map) > 0
}

func sparseResidency(img *ImageObject) bool {
	return ((uint32(img.Info.Flags) & uint32(VkImageCreateFlagBits_VK_IMAGE_CREATE_SPARSE_BINDING_BIT)) != 0) &&
		((uint32(img.Info.Flags) & uint32(VkImageCreateFlagBits_VK_IMAGE_CREATE_SPARSE_RESIDENCY_BIT)) != 0)
}

func vkCreateImage(sb *stateBuilder, dev VkDevice, info ImageInfo, handle VkImage) {
	pNext := NewVoidᶜᵖ(memory.Nullptr)
	if info.DedicatedAllocationNV != nil {
		pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
			VkDedicatedAllocationImageCreateInfoNV{
				VkStructureType_VK_STRUCTURE_TYPE_DEDICATED_ALLOCATION_IMAGE_CREATE_INFO_NV,
				NewVoidᶜᵖ(memory.Nullptr),
				info.DedicatedAllocationNV.DedicatedAllocation,
			},
		).Ptr())
	}
	sb.write(sb.cb.VkCreateImage(
		dev, sb.MustAllocReadData(
			VkImageCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
				pNext,
				info.Flags,
				info.ImageType,
				info.Format,
				info.Extent,
				info.MipLevels,
				info.ArrayLayers,
				info.Samples,
				info.Tiling,
				info.Usage,
				info.SharingMode,
				uint32(len(*info.QueueFamilyIndices.Map)),
				NewU32ᶜᵖ(sb.MustUnpackReadMap(*info.QueueFamilyIndices.Map).Ptr()),
				VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
			}).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func vkGetImageMemoryRequirements(sb *stateBuilder, dev VkDevice, handle VkImage, memReq *VkMemoryRequirements) {
	sb.write(sb.cb.VkGetImageMemoryRequirements(
		dev, handle, sb.MustAllocWriteData(*memReq).Ptr(),
	))
}

func vkAllocateMemory(sb *stateBuilder, dev VkDevice, size VkDeviceSize, memTypeIndex uint32, handle VkDeviceMemory) {
	sb.write(sb.cb.VkAllocateMemory(
		dev,
		NewVkMemoryAllocateInfoᶜᵖ(sb.MustAllocReadData(
			VkMemoryAllocateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				size,
				memTypeIndex,
			}).Ptr()),
		memory.Nullptr,
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func vkBindImageMemory(sb *stateBuilder, dev VkDevice, img VkImage, mem VkDeviceMemory, offset VkDeviceSize) {
	sb.write(sb.cb.VkBindImageMemory(
		dev, img, mem, offset, VkResult_VK_SUCCESS,
	))
}

func vkCreateDescriptorSetLayout(sb *stateBuilder, dev VkDevice, bindings []VkDescriptorSetLayoutBinding, handle VkDescriptorSetLayout) {
	sb.write(sb.cb.VkCreateDescriptorSetLayout(
		dev,
		sb.MustAllocReadData(VkDescriptorSetLayoutCreateInfo{
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO,
			NewVoidᶜᵖ(memory.Nullptr),
			VkDescriptorSetLayoutCreateFlags(0),
			uint32(len(bindings)),
			NewVkDescriptorSetLayoutBindingᶜᵖ(sb.MustAllocReadData(bindings).Ptr()),
		}).Ptr(),
		NewVoidᶜᵖ(memory.Nullptr),
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func vkCreatePipelineLayout(sb *stateBuilder, dev VkDevice, setLayouts []VkDescriptorSetLayout, pushConstantRanges []VkPushConstantRange, handle VkPipelineLayout) {
	createInfo := VkPipelineLayoutCreateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkPipelineLayoutCreateFlags(0),
		uint32(len(setLayouts)),
		NewVkDescriptorSetLayoutᶜᵖ(sb.MustAllocReadData(setLayouts).Ptr()),
		uint32(len(pushConstantRanges)),
		NewVkPushConstantRangeᶜᵖ(sb.MustAllocReadData(pushConstantRanges).Ptr()),
	}
	sb.write(sb.cb.VkCreatePipelineLayout(
		dev,
		NewVkPipelineLayoutCreateInfoᶜᵖ(sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func vkCreateShaderModule(sb *stateBuilder, dev VkDevice, code []uint32, handle VkShaderModule) {
	createInfo := VkShaderModuleCreateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkShaderModuleCreateFlags(0),
		memory.Size(len(code) * 4),
		NewU32ᶜᵖ(sb.MustAllocReadData(code).Ptr()),
	}
	sb.write(sb.cb.VkCreateShaderModule(
		dev,
		NewVkShaderModuleCreateInfoᶜᵖ(sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func vkCreateDescriptorPool(sb *stateBuilder, dev VkDevice, flags VkDescriptorPoolCreateFlags, maxSet uint32, poolSizes []VkDescriptorPoolSize, handle VkDescriptorPool) {
	sb.write(sb.cb.VkCreateDescriptorPool(
		dev,
		sb.MustAllocReadData(VkDescriptorPoolCreateInfo{
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
			NewVoidᶜᵖ(memory.Nullptr),
			flags,
			maxSet,
			uint32(len(poolSizes)),
			NewVkDescriptorPoolSizeᶜᵖ(sb.MustAllocReadData(poolSizes).Ptr()),
		}).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func writeDescriptorSet(sb *stateBuilder, dev VkDevice, descSet VkDescriptorSet, dstBinding, dstArrayElement uint32, descType VkDescriptorType, imgInfoList []VkDescriptorImageInfo, bufInfoList []VkDescriptorBufferInfo, texelBufInfoList []VkBufferView) {
	write := VkWriteDescriptorSet{
		VkStructureType_VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
		NewVoidᶜᵖ(memory.Nullptr),
		descSet,
		dstBinding,
		dstArrayElement,
		uint32(len(imgInfoList) + len(bufInfoList) + len(texelBufInfoList)),
		descType,
		NewVkDescriptorImageInfoᶜᵖ(sb.MustAllocReadData(imgInfoList).Ptr()),
		NewVkDescriptorBufferInfoᶜᵖ(sb.MustAllocReadData(bufInfoList).Ptr()),
		NewVkBufferViewᶜᵖ(sb.MustAllocReadData(texelBufInfoList).Ptr()),
	}

	sb.write(sb.cb.VkUpdateDescriptorSets(
		dev,
		uint32(1),
		NewVkWriteDescriptorSetᶜᵖ(sb.MustAllocReadData(write).Ptr()),
		uint32(0),
		memory.Nullptr,
	))
}
