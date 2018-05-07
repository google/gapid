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
	sh *ipStoreHandler
}

func newImagePrimer(sb *stateBuilder) *imagePrimer {
	p := &imagePrimer{
		sb: sb,
		rh: newImagePrimerRenderHandler(sb),
		sh: newImagePrimerStoreHandler(sb),
	}
	return p
}

const (
	stagingColorImageBufferFormat        = VkFormat_VK_FORMAT_R32G32B32A32_UINT
	stagingDepthStencilImageBufferFormat = VkFormat_VK_FORMAT_R32_UINT
)

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

func (p *imagePrimer) primeByImageStore(img *ImageObject, opaqueBoundRanges []VkImageSubresourceRange, queue *QueueObject, sparseBindingQueue *QueueObject) error {
	storeJobs := []*ipStoreJob{}
	for _, rng := range opaqueBoundRanges {
		walkImageSubresourceRange(p.sb, img, rng,
			func(aspect VkImageAspectFlagBits, layer, level uint32, levelSize byteSizeAndExtent) {
				storeJobs = append(storeJobs, &ipStoreJob{
					storeTarget:       img,
					aspect:            aspect,
					layer:             layer,
					level:             level,
					opaqueBlockOffset: VkOffset3D{0, 0, 0},
					opaqueBlockExtent: VkExtent3D{
						uint32(levelSize.width),
						uint32(levelSize.height),
						uint32(levelSize.depth),
					},
				})
			})
	}

	if sparseResidency(img) {
		walkSparseImageMemoryBindings(p.sb, img,
			func(aspect VkImageAspectFlagBits, layer, level uint32, blockData *SparseBoundImageBlockInfo) {
				storeJobs = append(storeJobs, &ipStoreJob{
					storeTarget:       img,
					aspect:            aspect,
					layer:             layer,
					level:             level,
					opaqueBlockOffset: blockData.Offset,
					opaqueBlockExtent: blockData.Extent,
				})
			})
	}

	p.sb.transitionImage(img, GetState(p.sb.newState).Images.Get(img.VulkanHandle).Info.Layout, VkImageLayout_VK_IMAGE_LAYOUT_GENERAL, sparseBindingQueue, queue)

	for _, job := range storeJobs {
		err := p.sh.store(job, queue)
		if err != nil {
			log.E(p.sb.ctx, "[Priming image: %v aspect: %v, layer: %v, level: %v, offset: %v, extent: %v data by imageStore] %v", job.storeTarget.VulkanHandle, job.aspect, job.layer, job.level, job.opaqueBlockOffset, job.opaqueBlockExtent, err)
		}
	}

	p.sb.transitionImage(img, VkImageLayout_VK_IMAGE_LAYOUT_GENERAL, img.Info.Layout, queue, queue)
	return nil
}

func (p *imagePrimer) free() {
	p.rh.free()
	p.sh.free()
}

// internal functions of image primer

func (p *imagePrimer) allocStagingImages(img *ImageObject, aspect VkImageAspectFlagBits) ([]*ImageObject, []*DeviceMemoryObject, error) {
	stagingImgs := []*ImageObject{}
	stagingMems := []*DeviceMemoryObject{}

	srcElementAndTexelInfo, err := subGetElementAndTexelBlockSize(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.oldState, GetState(p.sb.oldState), 0, nil, img.Info.Fmt)
	if err != nil {
		return []*ImageObject{}, []*DeviceMemoryObject{}, fmt.Errorf("[Getting element size and texel block info] %v", err)
	}
	if srcElementAndTexelInfo.TexelBlockSize.Width != uint32(1) || srcElementAndTexelInfo.TexelBlockSize.Height != uint32(1) {
		// compressed formats are not supported
		return []*ImageObject{}, []*DeviceMemoryObject{}, fmt.Errorf("allocating staging images for compressed format images is not supported")
	}
	srcElementSize := srcElementAndTexelInfo.ElementSize
	if aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT {
		srcElementSize, err = subGetDepthElementSize(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.oldState, GetState(p.sb.oldState), 0, nil, img.Info.Fmt, false)
		if err != nil {
			return []*ImageObject{}, []*DeviceMemoryObject{}, fmt.Errorf("[Getting element size for depth aspect] %v", err)
		}
	} else if aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT {
		srcElementSize = uint32(1)
	}

	stagingImgFormat := VkFormat_VK_FORMAT_UNDEFINED
	switch aspect {
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
		stagingImgFormat = stagingColorImageBufferFormat
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		stagingImgFormat = stagingDepthStencilImageBufferFormat
	}
	if stagingImgFormat == VkFormat_VK_FORMAT_UNDEFINED {
		return []*ImageObject{}, []*DeviceMemoryObject{}, fmt.Errorf("unsupported aspect: %v", aspect)
	}
	stagingElementInfo, _ := subGetElementAndTexelBlockSize(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.oldState, GetState(p.sb.oldState), 0, nil, stagingImgFormat)
	stagingElementSize := stagingElementInfo.ElementSize

	stagingInfo := img.Info
	stagingInfo.DedicatedAllocationNV = nil
	stagingInfo.Fmt = stagingImgFormat
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

// In-shader image store handler
type ipStoreHandler struct {
	sb              *stateBuilder
	descSetLayouts  map[VkDevice]VkDescriptorSetLayout
	pipelineLayouts map[VkDevice]VkPipelineLayout
	pipelines       map[ipStoreShaderInfo]*ComputePipelineObject
	shaders         map[ipStoreShaderInfo]*ShaderModuleObject
}

type ipStoreJob struct {
	storeTarget       *ImageObject
	aspect            VkImageAspectFlagBits
	layer             uint32
	level             uint32
	opaqueBlockOffset VkOffset3D
	opaqueBlockExtent VkExtent3D
}

type ipStoreShaderInfo struct {
	dev     VkDevice
	fmt     VkFormat
	aspect  VkImageAspectFlagBits
	imgType VkImageType
}

const (
	ipStoreStorageImageBinding       = 0
	ipStoreUniformTexelBufferBinding = 1
	ipStoreUniformBufferBinding      = 2
	specMaxTexelBufferElements       = 65536
	specMaxComputeGlobalGroupCountX  = 65536
	specMaxComputeLocalGroupSizeX    = 128
)

// ipStoreTexelBufferStoreInfo contains the information to initiate one or more
// vkCmdDispatch calls to store the staging data in the texel buffer to the
// image. It is guaranteed the raw data can be saved in one texel buffer.
type ipStoreTexelBufferStoreInfo struct {
	dev VkDevice
	// About the staging data to be hold by a texel buffer
	data                []uint8
	dataFormat          VkFormat
	dataElementSize     uint32
	offsetInOpaqueBlock uint32
	// parent store job
	job *ipStoreJob
	// descriptor related
	descSet  VkDescriptorSet
	pipeline *ComputePipelineObject
}

// ipStoreDispatchInfo contains the information to submit a dispatch to store
// staging data to the target image. It is guaranteed that each pixel of the
// target image will be processed by a local invocation.
type ipStoreDispatchInfo struct {
	dev VkDevice
	// About the staging data for dispatch
	data                []uint8
	dataFormat          VkFormat
	dataElementSize     uint32
	offsetInOpaqueBlock uint32
	// Parent store job
	job *ipStoreJob
	// About descriptor set
	descSet        VkDescriptorSet
	pipelineLayout VkPipelineLayout
	pipeline       VkPipeline
}

// Interfaces of image store handler to interact with image primer

func newImagePrimerStoreHandler(sb *stateBuilder) *ipStoreHandler {
	return &ipStoreHandler{
		sb:              sb,
		descSetLayouts:  map[VkDevice]VkDescriptorSetLayout{},
		pipelineLayouts: map[VkDevice]VkPipelineLayout{},
		pipelines:       map[ipStoreShaderInfo]*ComputePipelineObject{},
		shaders:         map[ipStoreShaderInfo]*ShaderModuleObject{},
	}
}

func (h *ipStoreHandler) store(job *ipStoreJob, queue *QueueObject) error {
	var err error

	dev := job.storeTarget.Device

	phyDev := h.sb.s.Devices.Get(dev).PhysicalDevice
	maxTexelBufferElements := uint32(specMaxTexelBufferElements)
	if h.sb.s.PhysicalDevices.Get(phyDev).PhysicalDeviceProperties.Limits.MaxTexelBufferElements > specMaxTexelBufferElements {
		maxTexelBufferElements = h.sb.s.PhysicalDevices.Get(phyDev).PhysicalDeviceProperties.Limits.MaxTexelBufferElements
	}

	// create descriptor pool
	descPool := VkDescriptorPool(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).DescriptorPools.Contains(VkDescriptorPool(x))
	}))
	descPoolSizes := []VkDescriptorPoolSize{
		// for target image
		VkDescriptorPoolSize{
			VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
			uint32(1),
		},
		// for image data
		VkDescriptorPoolSize{
			VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER,
			uint32(1),
		},
		// for image dimension info
		VkDescriptorPoolSize{
			VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
			uint32(1),
		},
	}
	vkCreateDescriptorPool(h.sb, dev, VkDescriptorPoolCreateFlags(
		VkDescriptorPoolCreateFlagBits_VK_DESCRIPTOR_POOL_CREATE_FREE_DESCRIPTOR_SET_BIT),
		1, descPoolSizes, descPool)
	defer h.sb.write(h.sb.cb.VkDestroyDescriptorPool(dev, descPool, memory.Nullptr))

	// create descriptor set layout
	if _, ok := h.descSetLayouts[dev]; !ok {
		descSetLayoutHandle := VkDescriptorSetLayout(newUnusedID(true, func(x uint64) bool {
			return GetState(h.sb.newState).DescriptorSetLayouts.Contains(VkDescriptorSetLayout(x))
		}))
		bindings := []VkDescriptorSetLayoutBinding{
			VkDescriptorSetLayoutBinding{
				ipStoreStorageImageBinding,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
				uint32(1),
				VkShaderStageFlags(VkShaderStageFlagBits_VK_SHADER_STAGE_COMPUTE_BIT),
				NewVkSamplerᶜᵖ(memory.Nullptr),
			},
			VkDescriptorSetLayoutBinding{
				ipStoreUniformTexelBufferBinding,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER,
				uint32(1),
				VkShaderStageFlags(VkShaderStageFlagBits_VK_SHADER_STAGE_COMPUTE_BIT),
				NewVkSamplerᶜᵖ(memory.Nullptr),
			},
			VkDescriptorSetLayoutBinding{
				ipStoreUniformBufferBinding,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
				uint32(1),
				VkShaderStageFlags(VkShaderStageFlagBits_VK_SHADER_STAGE_COMPUTE_BIT),
				NewVkSamplerᶜᵖ(memory.Nullptr),
			},
		}
		vkCreateDescriptorSetLayout(h.sb, dev, bindings, descSetLayoutHandle)
		h.descSetLayouts[dev] = descSetLayoutHandle
	}

	// allocate descriptor set
	descSet := VkDescriptorSet(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).DescriptorSets.Contains(VkDescriptorSet(x))
	}))
	vkAllocateDescriptorSet(h.sb, dev, descPool, h.descSetLayouts[dev], descSet)
	defer func() {
		h.sb.write(h.sb.cb.VkFreeDescriptorSets(dev, descPool, uint32(1),
			NewVkDescriptorSetᶜᵖ(h.sb.MustAllocReadData(descSet).Ptr()), VkResult_VK_SUCCESS))
	}()

	// Create compute pipeline
	if _, ok := h.pipelineLayouts[dev]; !ok {
		pipelineLayoutHandle := VkPipelineLayout(newUnusedID(true, func(x uint64) bool {
			return GetState(h.sb.newState).PipelineLayouts.Contains(VkPipelineLayout(x))
		}))
		vkCreatePipelineLayout(h.sb, dev, []VkDescriptorSetLayout{h.descSetLayouts[dev]},
			[]VkPushConstantRange{}, pipelineLayoutHandle)
		h.pipelineLayouts[dev] = pipelineLayoutHandle
	}

	compShaderInfo := ipStoreShaderInfo{
		dev:     dev,
		fmt:     job.storeTarget.Info.Fmt,
		aspect:  job.aspect,
		imgType: job.storeTarget.Info.ImageType,
	}
	pipeline, err := h.getOrCreateComputePipeline(compShaderInfo)
	if err != nil {
		return fmt.Errorf("[Getting compute pipeline] %v", err)
	}

	// Prepare the raw image data
	opaqueDataOffset := uint64(h.sb.levelSize(VkExtent3D{
		uint32(job.opaqueBlockOffset.X),
		uint32(job.opaqueBlockOffset.Y),
		uint32(job.opaqueBlockOffset.Z),
	}, job.storeTarget.Info.Fmt, 0, job.aspect).levelSize)
	opaqueDataSizeInBytes := uint64(h.sb.levelSize(
		job.opaqueBlockExtent,
		job.storeTarget.Info.Fmt,
		0, job.aspect).levelSize)
	data := job.storeTarget.Aspects.
		Get(job.aspect).Layers.Get(job.layer).Levels.Get(job.level).Data.
		Slice(opaqueDataOffset, opaqueDataSizeInBytes).
		MustRead(h.sb.ctx, nil, h.sb.oldState, nil)

	srcVkFmt := job.storeTarget.Info.Fmt
	if srcVkFmt == VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32 {
		data, srcVkFmt, err = ebgrDataToRGB32SFloat(data, job.opaqueBlockExtent)
		if err != nil {
			return fmt.Errorf("[Converting data in VK_FORMAT_E5B9G9R9_UFLOAT_PACK32 to VK_FORMAT_R32G32B32_SFLOAT] %v", err)
		}
	}
	var unpackedFmt VkFormat
	unpacked, unpackedFmt, err := unpackDataForPriming(data, srcVkFmt, job.aspect)
	if err != nil {
		return fmt.Errorf("[Unpacking data from format: %v aspect: %v] %v", srcVkFmt, job.aspect, err)
	}

	unpackedElementAndTexelBlockSize, _ := subGetElementAndTexelBlockSize(
		h.sb.ctx, nil, api.CmdNoID, nil, h.sb.oldState, nil, 0, nil, unpackedFmt)
	unpackedElementSize := unpackedElementAndTexelBlockSize.ElementSize

	// Using multiple texel buffers if necessary
	texelBufferStart := uint32(0)
	for texelBufferStart < uint32(len(unpacked)) {
		texelBufferEnd := texelBufferStart + maxTexelBufferElements*unpackedElementSize
		if texelBufferEnd > uint32(len(unpacked)) {
			texelBufferEnd = uint32(len(unpacked))
		}
		texelOffset := texelBufferStart / unpackedElementSize
		texelBufferStoreInfo := ipStoreTexelBufferStoreInfo{
			dev:                 dev,
			data:                unpacked[texelBufferStart:texelBufferEnd],
			dataFormat:          unpackedFmt,
			dataElementSize:     unpackedElementSize,
			offsetInOpaqueBlock: texelOffset,
			job:                 job,
			descSet:             descSet,
			pipeline:            pipeline,
		}
		h.storeThroughTexelBuffer(texelBufferStoreInfo, queue)
		texelBufferStart = texelBufferEnd
	}
	return nil
}

func (h *ipStoreHandler) free() {
	for _, p := range h.pipelines {
		h.sb.write(h.sb.cb.VkDestroyPipeline(p.Device, p.VulkanHandle, memory.Nullptr))
	}
	for _, m := range h.shaders {
		h.sb.write(h.sb.cb.VkDestroyShaderModule(m.Device, m.VulkanHandle, memory.Nullptr))
	}
	for dev, l := range h.pipelineLayouts {
		h.sb.write(h.sb.cb.VkDestroyPipelineLayout(dev, l, memory.Nullptr))
	}
	for dev, l := range h.descSetLayouts {
		h.sb.write(h.sb.cb.VkDestroyDescriptorSetLayout(dev, l, memory.Nullptr))
	}
}

// Internal functions of image store handler

func (h *ipStoreHandler) dispatchAndSubmit(info ipStoreDispatchInfo, queue *QueueObject) error {
	// check the number of texel
	if uint32(len(info.data))%info.dataElementSize != 0 {
		return fmt.Errorf("len(data): %v is not multiple times of the element size: %v", len(info.data), info.dataElementSize)
	}
	maxDispatchTexelCount := uint64(specMaxComputeGlobalGroupCountX * specMaxComputeLocalGroupSizeX)
	texelCount := uint64(uint32(len(info.data)) / info.dataElementSize)
	if texelCount > maxDispatchTexelCount {
		return fmt.Errorf("number of texels: %v exceeds the limit: %v", uint32(len(info.data))/info.dataElementSize, maxDispatchTexelCount)
	}

	// data buffer and buffer view
	dataBuf, dataMem := h.sb.allocAndFillScratchBuffer(
		h.sb.s.Devices.Get(info.dev), info.data,
		VkBufferUsageFlagBits_VK_BUFFER_USAGE_UNIFORM_TEXEL_BUFFER_BIT)
	defer h.sb.freeScratchBuffer(h.sb.s.Devices.Get(info.dev), dataBuf, dataMem)
	dataBufView := VkBufferView(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).BufferViews.Contains(VkBufferView(x))
	}))
	h.sb.write(h.sb.cb.VkCreateBufferView(
		info.dev,
		h.sb.MustAllocReadData(
			VkBufferViewCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_BUFFER_VIEW_CREATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				VkBufferViewCreateFlags(0),
				dataBuf,
				info.dataFormat,
				VkDeviceSize(0),
				VkDeviceSize(0xFFFFFFFFFFFFFFFF),
			}).Ptr(),
		memory.Nullptr,
		h.sb.MustAllocWriteData(dataBufView).Ptr(),
		VkResult_VK_SUCCESS,
	))
	defer h.sb.write(h.sb.cb.VkDestroyBufferView(info.dev, dataBufView, memory.Nullptr))

	// image view
	imgView := VkImageView(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).ImageViews.Contains(VkImageView(x))
	}))
	var imgViewType VkImageViewType
	switch info.job.storeTarget.Info.ImageType {
	case VkImageType_VK_IMAGE_TYPE_1D:
		imgViewType = VkImageViewType_VK_IMAGE_VIEW_TYPE_1D
	case VkImageType_VK_IMAGE_TYPE_2D:
		imgViewType = VkImageViewType_VK_IMAGE_VIEW_TYPE_2D
	case VkImageType_VK_IMAGE_TYPE_3D:
		imgViewType = VkImageViewType_VK_IMAGE_VIEW_TYPE_3D
	}
	h.sb.write(h.sb.cb.VkCreateImageView(
		info.dev,
		NewVkImageViewCreateInfoᶜᵖ(h.sb.MustAllocReadData(
			VkImageViewCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				VkImageViewCreateFlags(0),
				info.job.storeTarget.VulkanHandle,
				imgViewType,
				info.job.storeTarget.Info.Fmt,
				VkComponentMapping{
					VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
					VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
					VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
					VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY,
				},
				VkImageSubresourceRange{
					VkImageAspectFlags(info.job.aspect),
					info.job.level,
					1,
					info.job.layer,
					1,
				},
			}).Ptr()),
		memory.Nullptr,
		h.sb.MustAllocWriteData(imgView).Ptr(),
		VkResult_VK_SUCCESS,
	))
	defer h.sb.write(h.sb.cb.VkDestroyImageView(info.dev, imgView, memory.Nullptr))

	// metadata buffer
	// For an N dimensional image, metadata buffer contains:
	//	uint32 opaque_block_offsets[N];
	//	uint32 opaque_block_extent[N];
	//  uint32 dispatch_offset_in_block;
	//	uint32 texel_count;
	metadata := []uint32{}
	switch info.job.storeTarget.Info.ImageType {
	case VkImageType_VK_IMAGE_TYPE_1D:
		metadata = append(metadata,
			uint32(info.job.opaqueBlockOffset.X),
			uint32(info.job.opaqueBlockExtent.Width),
		)
	case VkImageType_VK_IMAGE_TYPE_2D:
		metadata = append(metadata,
			uint32(info.job.opaqueBlockOffset.X),
			uint32(info.job.opaqueBlockOffset.Y),
			uint32(info.job.opaqueBlockExtent.Width),
			uint32(info.job.opaqueBlockExtent.Height),
		)
	case VkImageType_VK_IMAGE_TYPE_3D:
		metadata = append(metadata,
			uint32(info.job.opaqueBlockOffset.X),
			uint32(info.job.opaqueBlockOffset.Y),
			uint32(info.job.opaqueBlockOffset.Z),
			uint32(info.job.opaqueBlockExtent.Width),
			uint32(info.job.opaqueBlockExtent.Height),
			uint32(info.job.opaqueBlockExtent.Depth),
		)
	}
	metadata = append(metadata, uint32(info.offsetInOpaqueBlock), uint32(texelCount))
	var db bytes.Buffer
	binary.Write(&db, binary.LittleEndian, metadata)
	metadataBuf, metadataMem := h.sb.allocAndFillScratchBuffer(
		h.sb.s.Devices.Get(info.dev), db.Bytes(),
		VkBufferUsageFlagBits_VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT)
	defer h.sb.freeScratchBuffer(h.sb.s.Devices.Get(info.dev), metadataBuf, metadataMem)

	writeDescriptorSet(h.sb, info.dev, info.descSet, ipStoreStorageImageBinding, 0, VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE, []VkDescriptorImageInfo{
		VkDescriptorImageInfo{
			VkSampler(0),
			imgView,
			VkImageLayout_VK_IMAGE_LAYOUT_GENERAL,
		},
	}, []VkDescriptorBufferInfo{}, []VkBufferView{})
	writeDescriptorSet(h.sb, info.dev, info.descSet, ipStoreUniformBufferBinding, 0,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
		[]VkDescriptorImageInfo{},
		[]VkDescriptorBufferInfo{VkDescriptorBufferInfo{
			metadataBuf, VkDeviceSize(0), VkDeviceSize(0xFFFFFFFFFFFFFFFF),
		}},
		[]VkBufferView{},
	)
	writeDescriptorSet(h.sb, info.dev, info.descSet, ipStoreUniformTexelBufferBinding, 0,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER,
		[]VkDescriptorImageInfo{},
		[]VkDescriptorBufferInfo{},
		[]VkBufferView{dataBufView},
	)

	// commands
	commandBuffer, commandPool := h.sb.getCommandBuffer(queue)
	h.sb.write(h.sb.cb.VkCmdBindPipeline(
		commandBuffer,
		VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_COMPUTE,
		info.pipeline,
	))
	h.sb.write(h.sb.cb.VkCmdBindDescriptorSets(
		commandBuffer,
		VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_COMPUTE,
		info.pipelineLayout,
		0, 1, h.sb.MustAllocReadData(info.descSet).Ptr(),
		0, NewU32ᶜᵖ(memory.Nullptr),
	))
	groupCount := uint32(roundUp(uint64(texelCount), specMaxComputeLocalGroupSizeX))
	h.sb.write(h.sb.cb.VkCmdDispatch(commandBuffer, groupCount, 1, 1))
	h.sb.endSubmitAndDestroyCommandBuffer(queue, commandBuffer, commandPool)

	return nil
}

func (h *ipStoreHandler) storeThroughTexelBuffer(info ipStoreTexelBufferStoreInfo, queue *QueueObject) {
	dispatchStart := uint32(0)
	maxDispatchTexelCount := specMaxComputeGlobalGroupCountX * specMaxComputeLocalGroupSizeX
	for dispatchStart < uint32(len(info.data)) {
		dispatchEnd := dispatchStart + uint32(maxDispatchTexelCount)*info.dataElementSize
		if dispatchEnd > uint32(len(info.data)) {
			dispatchEnd = uint32(len(info.data))
		}

		dispatchInfo := ipStoreDispatchInfo{
			dev:                 info.dev,
			data:                info.data[dispatchStart:dispatchEnd],
			dataFormat:          info.dataFormat,
			dataElementSize:     info.dataElementSize,
			offsetInOpaqueBlock: info.offsetInOpaqueBlock + dispatchStart/info.dataElementSize,
			job:                 info.job,
			descSet:             info.descSet,
			pipelineLayout:      info.pipeline.PipelineLayout.VulkanHandle,
			pipeline:            info.pipeline.VulkanHandle,
		}
		err := h.dispatchAndSubmit(dispatchInfo, queue)
		if err != nil {
			log.E(h.sb.ctx, "[Priming storage image: %v by imageStore] %v", info.job.storeTarget.VulkanHandle)
		}
		dispatchStart = dispatchEnd
	}
}

func (h *ipStoreHandler) getOrCreateComputePipeline(info ipStoreShaderInfo) (*ComputePipelineObject, error) {
	if p, ok := h.pipelines[info]; ok {
		return p, nil
	}

	compShader, err := h.getOrCreateShaderModule(info)
	// TODO: report to report view if the image is a depth/stencil image.
	if err != nil {
		return nil, fmt.Errorf("[Getting compute shader module] %v", err)
	}

	if _, ok := h.pipelineLayouts[info.dev]; !ok {
		return nil, fmt.Errorf("pipeline layout not found")
	}

	handle := VkPipeline(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).ComputePipelines.Contains(VkPipeline(x))
	}))

	createInfo := VkComputePipelineCreateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkPipelineCreateFlags(0),
		VkPipelineShaderStageCreateInfo{
			VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
			NewVoidᶜᵖ(memory.Nullptr),
			VkPipelineShaderStageCreateFlags(0),
			VkShaderStageFlagBits_VK_SHADER_STAGE_COMPUTE_BIT,
			compShader.VulkanHandle,
			NewCharᶜᵖ(h.sb.MustAllocReadData("main").Ptr()),
			NewVkSpecializationInfoᶜᵖ(memory.Nullptr),
		},
		h.pipelineLayouts[info.dev],
		VkPipeline(0),
		int32(0),
	}
	h.sb.write(h.sb.cb.VkCreateComputePipelines(
		info.dev, VkPipelineCache(0), uint32(1),
		h.sb.MustAllocReadData(createInfo).Ptr(),
		memory.Nullptr, h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	h.pipelines[info] = GetState(h.sb.newState).ComputePipelines.Get(handle)
	return h.pipelines[info], nil
}

func (h *ipStoreHandler) getOrCreateShaderModule(info ipStoreShaderInfo) (*ShaderModuleObject, error) {
	if m, ok := h.shaders[info]; ok {
		return m, nil
	}
	handle := VkShaderModule(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).ShaderModules.Contains(VkShaderModule(x))
	}))
	code, err := ipComputeShaderSpirv(info.fmt, info.aspect, info.imgType)
	if err != nil {
		return nil, fmt.Errorf("[Generating SPIR-V for: %v] %v", info, err)
	}
	if len(code) == 0 {
		return nil, fmt.Errorf("no SPIR-V code generated")
	}
	vkCreateShaderModule(h.sb, info.dev, code, handle)
	h.shaders[info] = GetState(h.sb.newState).ShaderModules.Get(handle)
	return h.shaders[info], nil
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

const (
	ipRenderInputAttachmentBinding = 0
	ipRenderUniformBufferBinding   = 1
)

type ipRenderDescriptorSetInfo struct {
	dev                 VkDevice
	numInputAttachments int
	numUniformBuffers   int
}

type ipRenderPassInfo struct {
	dev                         VkDevice
	numInputAttachments         int
	inputAttachmentImageFormat  VkFormat
	inputAttachmentImageSamples VkSampleCountFlagBits
	targetAspect                VkImageAspectFlagBits
	targetFormat                VkFormat
	targetSamples               VkSampleCountFlagBits
}

type ipRenderShaderInfo struct {
	dev      VkDevice
	isVertex bool
	format   VkFormat
	aspect   VkImageAspectFlagBits
}

type ipGfxPipelineInfo struct {
	fragShaderInfo ipRenderShaderInfo
	pipelineLayout VkPipelineLayout
	renderPassInfo ipRenderPassInfo
}

type ipRenderHandler struct {
	sb *stateBuilder
	// descriptor set layouts indexed by different number of input attachment
	descriptorSetLayouts map[ipRenderDescriptorSetInfo]*DescriptorSetLayoutObject
	// pipeline layouts indexed by the number of input attachment in the only
	// descriptor set layout of the pipeline layout.
	pipelineLayouts map[ipRenderDescriptorSetInfo]*PipelineLayoutObject
	// pipelines indexed by the pipeline info.
	pipelines map[ipGfxPipelineInfo]*GraphicsPipelineObject
	// shader modules indexed by the shader info.
	shaders map[ipRenderShaderInfo]*ShaderModuleObject
}

// Interfaces of render handler to interact with image primer

func newImagePrimerRenderHandler(sb *stateBuilder) *ipRenderHandler {
	return &ipRenderHandler{
		sb:                   sb,
		descriptorSetLayouts: map[ipRenderDescriptorSetInfo]*DescriptorSetLayoutObject{},
		pipelineLayouts:      map[ipRenderDescriptorSetInfo]*PipelineLayoutObject{},
		pipelines:            map[ipGfxPipelineInfo]*GraphicsPipelineObject{},
		shaders:              map[ipRenderShaderInfo]*ShaderModuleObject{},
	}
}

func (h *ipRenderHandler) free() {
	for _, obj := range h.pipelines {
		h.sb.write(h.sb.cb.VkDestroyPipeline(obj.Device, obj.VulkanHandle, memory.Nullptr))
	}
	for _, obj := range h.shaders {
		h.sb.write(h.sb.cb.VkDestroyShaderModule(obj.Device, obj.VulkanHandle, memory.Nullptr))
	}
	for _, obj := range h.pipelineLayouts {
		h.sb.write(h.sb.cb.VkDestroyPipelineLayout(obj.Device, obj.VulkanHandle, memory.Nullptr))
	}
	for _, obj := range h.descriptorSetLayouts {
		h.sb.write(h.sb.cb.VkDestroyDescriptorSetLayout(obj.Device, obj.VulkanHandle, memory.Nullptr))
	}
}

func (h *ipRenderHandler) render(job *ipRenderJob, queue *QueueObject) error {
	var outputBarrierAspect VkImageAspectFlags
	switch job.aspect {
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
		outputBarrierAspect = VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT)
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		switch job.renderTarget.Info.Fmt {
		case VkFormat_VK_FORMAT_D16_UNORM_S8_UINT,
			VkFormat_VK_FORMAT_D24_UNORM_S8_UINT,
			VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
			outputBarrierAspect = VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT |
				VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT)
		default:
			outputBarrierAspect = VkImageAspectFlags(job.aspect)
		}
	default:
		return fmt.Errorf("unsupported aspect: %v", job.aspect)
	}

	var outputPreRenderLayout VkImageLayout
	switch job.aspect {
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
		outputPreRenderLayout = VkImageLayout_VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		outputPreRenderLayout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
	default:
		return fmt.Errorf("unsupported aspect: %v", job.aspect)
	}

	dev := job.renderTarget.Device

	descSetInfo := ipRenderDescriptorSetInfo{
		dev:                 dev,
		numInputAttachments: len(job.inputAttachmentImages),
	}
	if job.aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT {
		// If the render target aspect is stencil, an uniform buffer is required
		// to store the stencil bit index value.
		descSetInfo.numUniformBuffers = 1
	}
	descPool := h.createDescriptorPool(descSetInfo)
	if descPool != nil {
		defer h.sb.write(h.sb.cb.VkDestroyDescriptorPool(dev, descPool.VulkanHandle, memory.Nullptr))
	} else {
		return fmt.Errorf("failed to create descriptor pool for %v input attachments", len(job.inputAttachmentImages))
	}
	descSetLayout := h.getOrCreateDescriptorSetLayout(descSetInfo)
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
		// TODO: support rendering to 3D images if maintenance1 is enabled.
		if input.Info.ImageType == VkImageType_VK_IMAGE_TYPE_3D {
			return fmt.Errorf("rendering to 3D images are not supported yet")
		}
		view := h.createImageView(dev, input, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, job.layer, job.level)
		inputViews = append(inputViews, view)
		if view != nil {
			defer h.sb.write(h.sb.cb.VkDestroyImageView(dev, view.VulkanHandle, memory.Nullptr))
		} else {
			return fmt.Errorf("failed to create image view for input attachment image: %v", input.VulkanHandle)
		}
	}
	// TODO: support rendering to 3D images if maintenance1 is enabled.
	if job.renderTarget.Info.ImageType == VkImageType_VK_IMAGE_TYPE_3D {
		return fmt.Errorf("rendering to 3D images are not supported yet")
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

	writeDescriptorSet(h.sb, dev, descSet.VulkanHandle, ipRenderInputAttachmentBinding, 0, VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT, imgInfoList, []VkDescriptorBufferInfo{}, []VkBufferView{})

	var stencilIndexBuf VkBuffer
	var stencilIndexMem VkDeviceMemory
	if job.aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT {
		// write the uniform buffer for rendering stencil value.
		stencilBitIndices := []uint32{0}
		var sbic bytes.Buffer
		binary.Write(&sbic, binary.LittleEndian, stencilBitIndices)
		stencilIndexBuf, stencilIndexMem = h.sb.allocAndFillScratchBuffer(h.sb.s.Devices.Get(dev), sbic.Bytes(), VkBufferUsageFlagBits_VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT|VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT)
		defer h.sb.freeScratchBuffer(h.sb.s.Devices.Get(dev), stencilIndexBuf, stencilIndexMem)

		bufInfoList := []VkDescriptorBufferInfo{
			VkDescriptorBufferInfo{stencilIndexBuf, 0, 4},
		}

		writeDescriptorSet(h.sb, dev, descSet.VulkanHandle, ipRenderUniformBufferBinding, 0, VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, []VkDescriptorImageInfo{}, bufInfoList, []VkBufferView{})
	}

	renderPassInfo := ipRenderPassInfo{
		dev:                         dev,
		numInputAttachments:         len(job.inputAttachmentImages),
		inputAttachmentImageFormat:  job.inputAttachmentImages[0].Info.Fmt,
		inputAttachmentImageSamples: job.inputAttachmentImages[0].Info.Samples,
		targetAspect:                job.aspect,
		targetFormat:                job.renderTarget.Info.Fmt,
		targetSamples:               job.renderTarget.Info.Samples,
	}
	renderPass := h.createRenderPass(renderPassInfo, job.finalLayout)
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

	targetLevelSize := h.sb.levelSize(job.renderTarget.Info.Extent, job.renderTarget.Info.Fmt, job.level, job.aspect)

	framebuffer := h.createFramebuffer(dev, renderPass.VulkanHandle, allViews,
		uint32(targetLevelSize.width), uint32(targetLevelSize.height))
	if framebuffer != nil {
		defer h.sb.write(h.sb.cb.VkDestroyFramebuffer(dev, framebuffer.VulkanHandle, memory.Nullptr))
	} else {
		return fmt.Errorf("failed to create framebuffer for rendering")
	}

	pipelineLayout := h.getOrCreatePipelineLayout(descSetInfo)
	if pipelineLayout == nil {
		return fmt.Errorf("failed to get pipeline layout for the rendering")
	}

	pipelineInfo := ipGfxPipelineInfo{
		fragShaderInfo: ipRenderShaderInfo{
			dev:      dev,
			isVertex: false,
			format:   job.renderTarget.Info.Fmt,
			aspect:   job.aspect,
		},
		pipelineLayout: pipelineLayout.VulkanHandle,
		renderPassInfo: renderPassInfo,
	}
	pipeline, err := h.getOrCreateGraphicsPipeline(pipelineInfo, renderPass.VulkanHandle)
	if err != nil {
		return fmt.Errorf("[Getting graphics pipeline] %v", err)
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
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT),
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
	bufBarriers := []VkBufferMemoryBarrier{
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
	}

	h.sb.write(h.sb.cb.VkCmdPipelineBarrier(
		commandBuffer,
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
		VkDependencyFlags(0),
		uint32(0),
		memory.Nullptr,
		uint32(len(bufBarriers)),
		h.sb.MustAllocReadData(bufBarriers).Ptr(),
		uint32(len(append(inputBarriers, outputBarrier))),
		h.sb.MustAllocReadData(append(inputBarriers, outputBarrier)).Ptr(),
	))

	switch job.aspect {
	// render color or depth aspect
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT:
		drawInfo := ipRenderDrawInfo{
			commandBuffer:    commandBuffer,
			renderPass:       renderPass,
			framebuffer:      framebuffer,
			descSet:          descSet,
			pipelineLayout:   pipelineLayout,
			pipeline:         pipeline,
			vertexBuf:        vertexBuf,
			indexBuf:         indexBuf,
			aspect:           job.aspect,
			width:            uint32(targetLevelSize.width),
			height:           uint32(targetLevelSize.height),
			stencilWriteMask: 0,
			stencilReference: 0,
			clearStencil:     false,
		}
		h.beginRenderPassAndDraw(drawInfo)

	// render stencil aspect
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		// render the i'th bit of all pixels.
		for i := uint32(0); i < uint32(8); i++ {
			h.sb.write(h.sb.cb.VkCmdPipelineBarrier(
				commandBuffer,
				VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
				VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
				VkDependencyFlags(0),
				uint32(0),
				memory.Nullptr,
				uint32(1),
				h.sb.MustAllocReadData([]VkBufferMemoryBarrier{
					VkBufferMemoryBarrier{
						VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
						NewVoidᶜᵖ(memory.Nullptr),
						VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT - 1) | VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT),
						VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT),
						uint32(queue.Family),
						uint32(queue.Family),
						stencilIndexBuf,
						0,
						VkDeviceSize(0xFFFFFFFFFFFFFFFF),
					}}).Ptr(),
				uint32(1),
				h.sb.MustAllocReadData([]VkImageMemoryBarrier{
					VkImageMemoryBarrier{
						VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
						NewVoidᶜᵖ(memory.Nullptr),
						VkAccessFlags(VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT),
						VkAccessFlags(VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT),
						VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
						VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
						queue.Family,
						queue.Family,
						job.renderTarget.VulkanHandle,
						VkImageSubresourceRange{
							outputBarrierAspect,
							0,
							job.renderTarget.Info.MipLevels,
							0,
							job.renderTarget.Info.ArrayLayers,
						}},
				}).Ptr(),
			))
			h.sb.write(h.sb.cb.VkCmdUpdateBuffer(
				commandBuffer,
				stencilIndexBuf,
				0, 4, NewVoidᶜᵖ(h.sb.MustAllocReadData([]uint32{i}).Ptr()),
			))
			h.sb.write(h.sb.cb.VkCmdPipelineBarrier(
				commandBuffer,
				VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
				VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
				VkDependencyFlags(0),
				uint32(0),
				memory.Nullptr,
				uint32(1),
				h.sb.MustAllocReadData([]VkBufferMemoryBarrier{
					VkBufferMemoryBarrier{
						VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
						NewVoidᶜᵖ(memory.Nullptr),
						VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_WRITE_BIT),
						VkAccessFlags(VkAccessFlagBits_VK_ACCESS_UNIFORM_READ_BIT),
						uint32(queue.Family),
						uint32(queue.Family),
						stencilIndexBuf,
						0,
						VkDeviceSize(0xFFFFFFFFFFFFFFFF),
					}}).Ptr(),
				uint32(0),
				memory.Nullptr,
			))
			drawInfo := ipRenderDrawInfo{
				commandBuffer:    commandBuffer,
				renderPass:       renderPass,
				framebuffer:      framebuffer,
				descSet:          descSet,
				pipelineLayout:   pipelineLayout,
				pipeline:         pipeline,
				vertexBuf:        vertexBuf,
				indexBuf:         indexBuf,
				aspect:           VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT,
				width:            uint32(targetLevelSize.width),
				height:           uint32(targetLevelSize.height),
				stencilWriteMask: 0x1 << i,
				stencilReference: 0x1 << i,
				clearStencil:     false,
			}
			if i == uint32(0) {
				drawInfo.clearStencil = true
			}
			h.beginRenderPassAndDraw(drawInfo)
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
			uint32(1),
			h.sb.MustAllocReadData([]VkImageMemoryBarrier{
				VkImageMemoryBarrier{
					VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
					NewVoidᶜᵖ(memory.Nullptr),
					VkAccessFlags(VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT),
					VkAccessFlags(VkAccessFlagBits_VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT),
					VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
					job.finalLayout,
					queue.Family,
					queue.Family,
					job.renderTarget.VulkanHandle,
					VkImageSubresourceRange{
						outputBarrierAspect,
						0,
						job.renderTarget.Info.MipLevels,
						0,
						job.renderTarget.Info.ArrayLayers,
					}},
			}).Ptr(),
		))
	default:
		return fmt.Errorf("invalid aspect: %v to render", job.aspect)
	}

	h.sb.endSubmitAndDestroyCommandBuffer(queue, commandBuffer, commandPool)
	return nil
}

// Internal functions for render handler

type ipRenderDrawInfo struct {
	commandBuffer    VkCommandBuffer
	renderPass       *RenderPassObject
	framebuffer      *FramebufferObject
	descSet          *DescriptorSetObject
	pipelineLayout   *PipelineLayoutObject
	pipeline         *GraphicsPipelineObject
	vertexBuf        VkBuffer
	indexBuf         VkBuffer
	aspect           VkImageAspectFlagBits
	width            uint32
	height           uint32
	stencilWriteMask uint32
	stencilReference uint32
	clearStencil     bool
}

func (h *ipRenderHandler) beginRenderPassAndDraw(info ipRenderDrawInfo) {
	h.sb.write(h.sb.cb.VkCmdBeginRenderPass(
		info.commandBuffer,
		h.sb.MustAllocReadData(
			VkRenderPassBeginInfo{
				VkStructureType_VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				info.renderPass.VulkanHandle,
				info.framebuffer.VulkanHandle,
				VkRect2D{
					VkOffset2D{int32(0), int32(0)},
					VkExtent2D{info.width, info.height},
				},
				uint32(0),
				NewVkClearValueᶜᵖ(memory.Nullptr),
			}).Ptr(),
		VkSubpassContents(0),
	))

	if info.clearStencil {
		h.sb.write(h.sb.cb.VkCmdClearAttachments(
			info.commandBuffer,
			uint32(1),
			h.sb.MustAllocReadData([]VkClearAttachment{
				VkClearAttachment{
					VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT),
					uint32(0),
					VkClearValue{VkClearColorValue{[4]uint32{0, 0, 0, 0}}},
				},
			}).Ptr(),
			uint32(1),
			h.sb.MustAllocReadData([]VkClearRect{
				VkClearRect{
					VkRect2D{
						VkOffset2D{int32(0), int32(0)},
						VkExtent2D{info.width, info.height},
					}, // rect
					// the baseArrayLayer counts from the base layer of the
					// attachment image view.
					uint32(0), // baseArrayLayer
					uint32(1), // layerCount
				},
			}).Ptr(),
		))
	}

	h.sb.write(h.sb.cb.VkCmdBindPipeline(
		info.commandBuffer,
		VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS,
		info.pipeline.VulkanHandle,
	))
	h.sb.write(h.sb.cb.VkCmdBindVertexBuffers(
		info.commandBuffer,
		0, 1,
		h.sb.MustAllocReadData(info.vertexBuf).Ptr(),
		h.sb.MustAllocReadData(VkDeviceSize(0)).Ptr(),
	))
	h.sb.write(h.sb.cb.VkCmdBindIndexBuffer(
		info.commandBuffer,
		info.indexBuf,
		VkDeviceSize(0),
		VkIndexType_VK_INDEX_TYPE_UINT32,
	))
	h.sb.write(h.sb.cb.VkCmdSetViewport(
		info.commandBuffer,
		uint32(0),
		uint32(1),
		NewVkViewportᶜᵖ(h.sb.MustAllocReadData(VkViewport{
			0.0, 0.0,
			float32(info.width), float32(info.height),
			0.0, 1.0,
		}).Ptr()),
	))
	h.sb.write(h.sb.cb.VkCmdSetScissor(
		info.commandBuffer,
		uint32(0),
		uint32(1),
		NewVkRect2Dᶜᵖ(h.sb.MustAllocReadData(VkRect2D{
			VkOffset2D{int32(0), int32(0)},
			VkExtent2D{info.width, info.height},
		}).Ptr()),
	))
	if info.aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT {
		h.sb.write(h.sb.cb.VkCmdSetStencilWriteMask(
			info.commandBuffer,
			VkStencilFaceFlags(VkStencilFaceFlagBits_VK_STENCIL_FRONT_AND_BACK),
			info.stencilWriteMask,
		))
		h.sb.write(h.sb.cb.VkCmdSetStencilReference(
			info.commandBuffer,
			VkStencilFaceFlags(VkStencilFaceFlagBits_VK_STENCIL_FRONT_AND_BACK),
			info.stencilReference,
		))
	}
	h.sb.write(h.sb.cb.VkCmdBindDescriptorSets(
		info.commandBuffer,
		VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS,
		info.pipelineLayout.VulkanHandle,
		0,
		1,
		h.sb.MustAllocReadData(info.descSet.VulkanHandle).Ptr(),
		0,
		NewU32ᶜᵖ(memory.Nullptr),
	))
	h.sb.write(h.sb.cb.VkCmdDrawIndexed(
		info.commandBuffer,
		6, 1, 0, 0, 0,
	))
	h.sb.write(h.sb.cb.VkCmdEndRenderPass(info.commandBuffer))
}

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
				img.Info.Fmt,
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
	vkAllocateDescriptorSet(h.sb, dev, pool, layout, handle)
	return GetState(h.sb.newState).DescriptorSets.Get(handle)
}

func (h *ipRenderHandler) createDescriptorPool(descSetInfo ipRenderDescriptorSetInfo) *DescriptorPoolObject {
	handle := VkDescriptorPool(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).DescriptorPools.Contains(VkDescriptorPool(x))
	}))

	poolSizes := []VkDescriptorPoolSize{}
	if descSetInfo.numInputAttachments != 0 {
		poolSizes = append(poolSizes, VkDescriptorPoolSize{
			VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT,
			uint32(descSetInfo.numInputAttachments),
		})
	}
	if descSetInfo.numUniformBuffers != 0 {
		poolSizes = append(poolSizes, VkDescriptorPoolSize{
			VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
			uint32(descSetInfo.numUniformBuffers),
		})
	}

	vkCreateDescriptorPool(h.sb, descSetInfo.dev, VkDescriptorPoolCreateFlags(
		VkDescriptorPoolCreateFlagBits_VK_DESCRIPTOR_POOL_CREATE_FREE_DESCRIPTOR_SET_BIT),
		1, poolSizes, handle)
	return GetState(h.sb.newState).DescriptorPools.Get(handle)
}

func (h *ipRenderHandler) createRenderPass(info ipRenderPassInfo, finalLayout VkImageLayout) *RenderPassObject {
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
		// The layout will be set later according to the image aspect bits.
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
	}
	outputAttachmentDesc := VkAttachmentDescription{
		VkAttachmentDescriptionFlags(0),
		info.targetFormat,
		info.targetSamples,
		VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE,
		VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE,
		// Keep the stencil aspect data. When rendering color or depth aspect,
		// stencil test will be disabled so stencil data won't be modified.
		VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD,
		VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE,
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
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT:
		outputAttachmentRef.Layout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
		outputAttachmentDesc.InitialLayout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
		subpassDesc.PDepthStencilAttachment = NewVkAttachmentReferenceᶜᵖ(h.sb.MustAllocReadData(outputAttachmentRef).Ptr())
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		outputAttachmentRef.Layout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
		outputAttachmentDesc.InitialLayout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
		// Rendering stencil data requires running the renderpass multiple times,
		// so do not change the image layout at the end of the renderpass
		outputAttachmentDesc.FinalLayout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
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
		info.dev,
		NewVkRenderPassCreateInfoᶜᵖ(h.sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))

	return GetState(h.sb.newState).RenderPasses.Get(handle)
}

func (h *ipRenderHandler) getOrCreateShaderModule(info ipRenderShaderInfo) (*ShaderModuleObject, error) {
	if m, ok := h.shaders[info]; ok {
		return m, nil
	}
	handle := VkShaderModule(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).ShaderModules.Contains(VkShaderModule(x))
	}))
	var err error
	code := []uint32{}
	if info.isVertex {
		code, err = ipVertexShaderSpirv()
	} else {
		code, err = ipFragmentShaderSpirv(info.format, info.aspect)
	}
	if err != nil {
		return nil, fmt.Errorf("[Generating shader SPIR-V for: %v] %v", info, err)
	}
	if len(code) == 0 {
		return nil, fmt.Errorf("no SPIR-V code generated")
	}
	vkCreateShaderModule(h.sb, info.dev, code, handle)
	h.shaders[info] = GetState(h.sb.newState).ShaderModules.Get(handle)
	return h.shaders[info], nil
}

func (h *ipRenderHandler) getOrCreateGraphicsPipeline(info ipGfxPipelineInfo, renderPass VkRenderPass) (*GraphicsPipelineObject, error) {
	if p, ok := h.pipelines[info]; ok {
		return p, nil
	}

	vertInfo := ipRenderShaderInfo{dev: info.renderPassInfo.dev, isVertex: true}
	vertShader, err := h.getOrCreateShaderModule(vertInfo)
	if err != nil {
		return nil, fmt.Errorf("[Getting vertex shader module] %v", err)
	}
	fragShader, err := h.getOrCreateShaderModule(info.fragShaderInfo)
	if err != nil {
		return nil, fmt.Errorf("[Getting fragment shader module] %v", err)
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
			0, // compareMask
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
		info.renderPassInfo.dev, VkPipelineCache(0), uint32(1),
		NewVkGraphicsPipelineCreateInfoᶜᵖ(h.sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr, h.sb.MustAllocWriteData(handle).Ptr(), VkResult_VK_SUCCESS,
	))

	h.pipelines[info] = GetState(h.sb.newState).GraphicsPipelines.Get(handle)
	return h.pipelines[info], nil
}

func (h *ipRenderHandler) getOrCreatePipelineLayout(descSetInfo ipRenderDescriptorSetInfo) *PipelineLayoutObject {
	if l, ok := h.pipelineLayouts[descSetInfo]; ok {
		return l
	}
	handle := VkPipelineLayout(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).PipelineLayouts.Contains(VkPipelineLayout(x))
	}))
	descriptorSet := h.getOrCreateDescriptorSetLayout(descSetInfo)
	vkCreatePipelineLayout(h.sb, descSetInfo.dev, []VkDescriptorSetLayout{descriptorSet.VulkanHandle}, []VkPushConstantRange{}, handle)
	h.pipelineLayouts[descSetInfo] = GetState(h.sb.newState).PipelineLayouts.Get(handle)
	return h.pipelineLayouts[descSetInfo]
}

func (h *ipRenderHandler) getOrCreateDescriptorSetLayout(descSetInfo ipRenderDescriptorSetInfo) *DescriptorSetLayoutObject {
	if l, ok := h.descriptorSetLayouts[descSetInfo]; ok {
		return l
	}

	handle := VkDescriptorSetLayout(newUnusedID(true, func(x uint64) bool {
		return GetState(h.sb.newState).DescriptorSetLayouts.Contains(VkDescriptorSetLayout(x))
	}))

	bindings := []VkDescriptorSetLayoutBinding{}
	if descSetInfo.numInputAttachments != 0 {
		bindings = append(bindings, VkDescriptorSetLayoutBinding{
			ipRenderInputAttachmentBinding,
			VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT,
			uint32(descSetInfo.numInputAttachments),
			VkShaderStageFlags(VkShaderStageFlagBits_VK_SHADER_STAGE_FRAGMENT_BIT),
			NewVkSamplerᶜᵖ(memory.Nullptr),
		})
	}
	if descSetInfo.numUniformBuffers != 0 {
		bindings = append(bindings, VkDescriptorSetLayoutBinding{
			ipRenderUniformBufferBinding,
			VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
			uint32(descSetInfo.numUniformBuffers),
			VkShaderStageFlags(VkShaderStageFlagBits_VK_SHADER_STAGE_FRAGMENT_BIT),
			NewVkSamplerᶜᵖ(memory.Nullptr),
		})
	}

	vkCreateDescriptorSetLayout(h.sb, descSetInfo.dev, bindings, handle)
	h.descriptorSetLayouts[descSetInfo] = GetState(h.sb.newState).DescriptorSetLayouts.Get(handle)
	return h.descriptorSetLayouts[descSetInfo]
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
	walkImageSubresourceRange(h.sb, h.job.srcImg, srcRng,
		func(aspect VkImageAspectFlagBits, layer, level uint32, levelSize byteSizeAndExtent) {
			extent := VkExtent3D{
				uint32(levelSize.width),
				uint32(levelSize.height),
				uint32(levelSize.depth),
			}
			for dstIndex, dstImg := range h.job.srcAspectsToDsts[aspect].dstImgs {
				// dstIndex is reserved for handling wide channel image format
				// like R64G64B64A64
				// TODO: handle wide format
				_ = dstIndex
				data, bufImgCopy, err := h.getCopyAndData(
					dstImg, h.job.srcAspectsToDsts[aspect].dstAspect,
					h.job.srcImg, aspect, layer, level, VkOffset3D{0, 0, 0},
					extent, offset)
				if err != nil {
					log.E(h.sb.ctx, "[Getting VkBufferImageCopy and raw data for priming data at image: %v, aspect: %v, layer: %v, level: %v] %v", h.job.srcImg.VulkanHandle, aspect, layer, level, err)
					continue
				}
				h.copies[dstImg] = append(h.copies[dstImg], bufImgCopy)
				h.content = append(h.content, data...)
				offset += uint64(len(data))
			}
		})
}

func (h *ipBufferCopySession) collectCopiesFromSparseImageBindings() {
	offset := uint64(len(h.content))
	walkSparseImageMemoryBindings(h.sb, h.job.srcImg,
		func(aspect VkImageAspectFlagBits, layer, level uint32, blockData *SparseBoundImageBlockInfo) {
			for dstIndex, dstImg := range h.job.srcAspectsToDsts[aspect].dstImgs {
				// dstIndex is reserved for handling wide channel image format
				// TODO: handle wide format
				_ = dstIndex
				data, bufImgCopy, err := h.getCopyAndData(
					dstImg, h.job.srcAspectsToDsts[aspect].dstAspect,
					h.job.srcImg, aspect, layer, level, blockData.Offset,
					blockData.Extent, offset)
				if err != nil {
					log.E(h.sb.ctx, "[Getting VkBufferImageCopy and raw data from sparse image binding at image: %v, aspect: %v, layer: %v, level: %v, offset: %v, extent: %v] %v", h.job.srcImg.VulkanHandle, aspect, layer, level, blockData.Offset, blockData.Extent, err)
					continue
				}
				h.copies[dstImg] = append(h.copies[dstImg], bufImgCopy)
				h.content = append(h.content, data...)
				offset += uint64(len(data))
			}
		})
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
				GetState(h.sb.newState).Images.Get(dstImg.VulkanHandle).Info.Layout,
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

	for i := range dstImgBarriers {
		dstImgBarriers[i].OldLayout = VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
		dstImgBarriers[i].NewLayout = h.job.finalLayout
		dstImgBarriers[i].SrcQueueFamilyIndex = uint32(submissionQueue.Family)
		if h.job.srcImg.LastBoundQueue != nil {
			dstImgBarriers[i].DstQueueFamilyIndex = uint32(h.job.srcImg.LastBoundQueue.Family)
		}
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

func (h *ipBufferCopySession) getCopyAndData(dstImg *ImageObject, dstAspect VkImageAspectFlagBits, srcImg *ImageObject, srcAspect VkImageAspectFlagBits, layer, level uint32, opaqueBlockOffset VkOffset3D, opaqueBlockExtent VkExtent3D, bufDataOffset uint64) ([]uint8, VkBufferImageCopy, error) {
	var err error
	bufImgCopy := VkBufferImageCopy{
		VkDeviceSize(bufDataOffset), 0, 0,
		VkImageSubresourceLayers{
			VkImageAspectFlags(dstAspect),
			level, layer, 1},
		opaqueBlockOffset, opaqueBlockExtent,
	}
	srcImgDataOffset := uint64(h.sb.levelSize(VkExtent3D{
		uint32(opaqueBlockOffset.X),
		uint32(opaqueBlockOffset.Y),
		uint32(opaqueBlockOffset.Z),
	}, srcImg.Info.Fmt, 0, srcAspect).levelSize)
	srcImgDataSizeInBytes := uint64(h.sb.levelSize(
		opaqueBlockExtent,
		srcImg.Info.Fmt,
		0, srcAspect).levelSize)

	data := srcImg.Aspects.Get(srcAspect).Layers.Get(layer).Levels.Get(level).
		Data.Slice(srcImgDataOffset, srcImgDataOffset+srcImgDataSizeInBytes).
		MustRead(h.sb.ctx, nil, h.sb.oldState, nil)

	unpacked := data
	if dstImg.Info.Fmt != srcImg.Info.Fmt {
		// dstImg format is different with the srcImage format, the dst image
		// should be a staging image.
		srcVkFmt := srcImg.Info.Fmt
		if srcImg.Info.Fmt == VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32 {
			data, srcVkFmt, err = ebgrDataToRGB32SFloat(data, opaqueBlockExtent)
			if err != nil {
				return []uint8{}, bufImgCopy, fmt.Errorf("[Converting data in VK_FORMAT_E5B9G9R9_UFLOAT_PACK32 to VK_FORMAT_R32G32B32_SFLOAT] %v", err)
			}
		}
		unpacked, _, err = unpackDataForPriming(data, srcVkFmt, srcAspect)
		if err != nil {
			return []uint8{}, bufImgCopy, fmt.Errorf("[Unpacking data from format: %v aspect: %v] %v", srcVkFmt, srcAspect, err)
		}

	} else if srcAspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT {
		// srcImg format is the same to the dstImage format, the data is ready to
		// be used directly, except when the src image is a dpeth 24 UNORM one.
		if (srcImg.Info.Fmt == VkFormat_VK_FORMAT_D24_UNORM_S8_UINT) ||
			(srcImg.Info.Fmt == VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32) {
			unpacked, _, err = unpackDataForPriming(data, srcImg.Info.Fmt, srcAspect)
			if err != nil {
				return []uint8{}, bufImgCopy, fmt.Errorf("[Unpacking data from format: %v aspect: %v] %v", srcImg.Info.Fmt, srcAspect, err)
			}
		}
	}

	extendToMultipleOf8(&unpacked)

	dstLevelSize := h.sb.levelSize(opaqueBlockExtent, dstImg.Info.Fmt, 0, dstAspect)
	if uint64(len(unpacked)) != dstLevelSize.alignedLevelSizeInBuf {
		return []uint8{}, bufImgCopy, fmt.Errorf("size of unpacked data does not match expectation, actual: %v, expected: %v, srcFmt: %v, dstFmt: %v", len(unpacked), dstLevelSize.alignedLevelSizeInBuf, srcImg.Info.Fmt, dstImg.Info.Fmt)
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

func unpackDataForPriming(data []uint8, srcFmt VkFormat, aspect VkImageAspectFlagBits) ([]uint8, VkFormat, error) {
	var sf *image.Format
	var err error
	var dstFmt VkFormat
	switch aspect {
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
		sf, err = getImageFormatFromVulkanFormat(srcFmt)
		if err != nil {
			return []uint8{}, dstFmt, fmt.Errorf("[Getting image.Format for VkFormat: %v, aspect: %v] %v", srcFmt, aspect, err)
		}
		dstFmt = stagingColorImageBufferFormat

	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT:
		sf, err = getDepthImageFormatFromVulkanFormat(srcFmt)
		if err != nil {
			return []uint8{}, dstFmt, fmt.Errorf("[Getting image.Format for VkFormat: %v, aspect: %v] %v", srcFmt, aspect, err)
		}
		dstFmt = stagingDepthStencilImageBufferFormat

	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		sf, err = getImageFormatFromVulkanFormat(VkFormat_VK_FORMAT_S8_UINT)
		if err != nil {
			return []uint8{}, dstFmt, fmt.Errorf("[Getting image.Format for VkFormat: %v, aspect: %v] %v", srcFmt, aspect, err)
		}
		dstFmt = stagingDepthStencilImageBufferFormat

	default:
		return []uint8{}, dstFmt, fmt.Errorf("unsupported aspect: %v", aspect)
	}

	df, err := getImageFormatFromVulkanFormat(dstFmt)
	if err != nil {
		return []uint8{}, dstFmt, fmt.Errorf("[Getting image.Format for VkFormat %v] %v", dstFmt, err)
	}
	unpacked, err := unpackData(data, sf, df)
	if err != nil {
		return []uint8{}, dstFmt, err
	}
	return unpacked, dstFmt, nil
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

func ebgrDataToRGB32SFloat(data []uint8, extent VkExtent3D) ([]uint8, VkFormat, error) {
	dstFmt := VkFormat_VK_FORMAT_R32G32B32_SFLOAT
	sf, err := getImageFormatFromVulkanFormat(VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32)
	if err != nil {
		return []uint8{}, dstFmt, err
	}
	df, err := getImageFormatFromVulkanFormat(dstFmt)
	if err != nil {
		return []uint8{}, dstFmt, err
	}
	retData, err := image.Convert(data, int(extent.Width), int(extent.Height), int(extent.Depth), sf, df)
	if err != nil {
		return []uint8{}, dstFmt, err
	}
	return retData, dstFmt, nil
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
				info.Fmt,
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

func vkAllocateDescriptorSet(sb *stateBuilder, dev VkDevice, pool VkDescriptorPool, layout VkDescriptorSetLayout, handle VkDescriptorSet) {
	sb.write(sb.cb.VkAllocateDescriptorSets(
		dev,
		sb.MustAllocReadData(VkDescriptorSetAllocateInfo{
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
			NewVoidᶜᵖ(memory.Nullptr),
			pool,
			1,
			NewVkDescriptorSetLayoutᶜᵖ(sb.MustAllocReadData(layout).Ptr()),
		}).Ptr(),
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

func walkImageSubresourceRange(sb *stateBuilder, img *ImageObject, rng VkImageSubresourceRange, f func(aspect VkImageAspectFlagBits, layer, level uint32, levelSize byteSizeAndExtent)) {
	for _, aspect := range sb.imageAspectFlagBits(rng.AspectMask) {
		for i := uint32(0); i < rng.LevelCount; i++ {
			level := rng.BaseMipLevel + i
			levelSize := sb.levelSize(img.Info.Extent, img.Info.Fmt, level, aspect)
			for j := uint32(0); j < rng.LayerCount; j++ {
				layer := rng.BaseArrayLayer + j
				f(aspect, layer, level, levelSize)
			}
		}
	}
}

func walkSparseImageMemoryBindings(sb *stateBuilder, img *ImageObject, f func(aspect VkImageAspectFlagBits, layer, level uint32, blockData *SparseBoundImageBlockInfo)) {
	for aspect, aspectData := range *img.SparseImageMemoryBindings.Map {
		for layer, layerData := range *aspectData.Layers.Map {
			for level, levelData := range *layerData.Levels.Map {
				for _, blockData := range *levelData.Blocks.Map {
					f(VkImageAspectFlagBits(aspect), layer, level, blockData)
				}
			}
		}
	}
}

func roundUp(dividend, divisor uint64) uint64 {
	return (dividend + divisor - 1) / divisor
}
