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

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/stream"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/shadertools"
)

// imagePrimer can create staging images and manages a series of image priming
// kit builders
type imagePrimer struct {
	sb               *stateBuilder
	hostCopyBuilders map[VkDevice]*ipHostCopyKitBuilder
	renderBuilders   map[VkDevice]*ipRenderKitBuilder
	storeBuilders    map[VkDevice]*ipStoreKitBuilder
}

func newImagePrimer(sb *stateBuilder) *imagePrimer {
	p := &imagePrimer{
		sb:               sb,
		hostCopyBuilders: map[VkDevice]*ipHostCopyKitBuilder{},
		renderBuilders:   map[VkDevice]*ipRenderKitBuilder{},
		storeBuilders:    map[VkDevice]*ipStoreKitBuilder{},
	}
	return p
}

const (
	stagingColorImageBufferFormat        = VkFormat_VK_FORMAT_R32G32B32A32_UINT
	stagingDepthStencilImageBufferFormat = VkFormat_VK_FORMAT_R32_UINT
)

func (p *imagePrimer) Free() {
	{
		keys := make([]VkDevice, 0, len(p.renderBuilders))
		for k := range p.renderBuilders {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		for _, i := range keys {
			b := p.renderBuilders[i]
			b.Free(p.sb)
			delete(p.renderBuilders, i)
		}
	}
	{
		keys := make([]VkDevice, 0, len(p.hostCopyBuilders))
		for k := range p.hostCopyBuilders {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		for _, i := range keys {
			b := p.hostCopyBuilders[i]
			b.Free(p.sb)
			delete(p.hostCopyBuilders, i)
		}
	}
	{
		keys := make([]VkDevice, 0, len(p.storeBuilders))
		for k := range p.storeBuilders {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		for _, i := range keys {
			b := p.storeBuilders[i]
			b.Free(p.sb)
			delete(p.storeBuilders, i)
		}
	}
}

func (p *imagePrimer) GetHostCopyKitBuilder(dev VkDevice) *ipHostCopyKitBuilder {
	if _, ok := p.hostCopyBuilders[dev]; !ok {
		p.hostCopyBuilders[dev] = newImagePrimerHostCopyKitBuilder(p.sb, dev)
	}
	return p.hostCopyBuilders[dev]
}

func (p *imagePrimer) GetRenderKitBuilder(dev VkDevice) *ipRenderKitBuilder {
	if _, ok := p.renderBuilders[dev]; !ok {
		p.renderBuilders[dev] = newImagePrimerRenderKitBuilder(p.sb, dev)
	}
	return p.renderBuilders[dev]
}

func (p *imagePrimer) GetStoreKitBuilder(dev VkDevice) *ipStoreKitBuilder {
	if _, ok := p.storeBuilders[dev]; !ok {
		p.storeBuilders[dev] = newImagePrimerStoreKitBuilder(p.sb, dev)
	}
	return p.storeBuilders[dev]
}

// internal functions of image primer

// createImageAndBindMemory creates an image with the give image info and device
// handle in the new state of the state builder of the current image primer,
// allocates memory for the created image based on the given memory type index,
// binds the memory with the new image, returns the created image object and the
// new device memory object in the new state of the state builder of the current
// image primer, and an error if any error occur.
func (p *imagePrimer) createImageAndBindMemory(dev VkDevice, info ImageInfo, memTypeIndex int) (ImageObjectʳ, DeviceMemoryObjectʳ, error) {
	imgHandle := VkImage(newUnusedID(true, func(x uint64) bool {
		return GetState(p.sb.newState).Images().Contains(VkImage(x))
	}))
	vkCreateImage(p.sb, dev, info, imgHandle)
	img := GetState(p.sb.newState).Images().Get(imgHandle)
	// Query the memory requirements so validation layers are happy
	vkGetImageMemoryRequirements(p.sb, dev, imgHandle, MakeVkMemoryRequirements())

	imgSize, err := subInferImageSize(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.newState, GetState(p.sb.newState), 0, nil, nil, img)
	if err != nil {
		return ImageObjectʳ{}, DeviceMemoryObjectʳ{}, log.Errf(p.sb.ctx, err, "[Getting image size]")
	}
	memHandle := VkDeviceMemory(newUnusedID(true, func(x uint64) bool {
		return GetState(p.sb.newState).DeviceMemories().Contains(VkDeviceMemory(x)) ||
			GetState(p.sb.oldState).DeviceMemories().Contains(VkDeviceMemory(x))
	}))
	// Since we cannot guess how much the driver will actually request of us,
	// overallocating by a factor of 2 should be enough.
	// TODO: Insert opcodes to determine the allocation size dynamically on the
	// replay side.
	allocSize := VkDeviceSize(imgSize * 2)
	if allocSize < VkDeviceSize(65536*info.Extent().Depth()) {
		allocSize = VkDeviceSize(65536 * info.Extent().Depth())
	}
	if allocSize < VkDeviceSize(256*1024) {
		allocSize = VkDeviceSize(256 * 1024)
	}
	vkAllocateMemory(p.sb, dev, allocSize, uint32(memTypeIndex), memHandle)
	mem := GetState(p.sb.newState).DeviceMemories().Get(memHandle)

	vkBindImageMemory(p.sb, dev, imgHandle, memHandle, 0)
	return img, mem, nil
}

// CreateSameStagingImage creates an image with the same image info (except
// initial layout) as the given image along with the given initial layout, and
// create backing memory for the new image and bind the image with the created
// memory (sparse binding not supported). Returns the created image object in
// the new state of the stateBuilder in the image primer, a function to destroy
// the new created image and backing memory, and an error.
func (p *imagePrimer) CreateSameStagingImage(img ImageObjectʳ) (ImageObjectʳ, func(), error) {
	dev := p.sb.s.Devices().Get(img.Device())
	phyDevMemProps := p.sb.s.PhysicalDevices().Get(dev.PhysicalDevice()).MemoryProperties()
	// TODO: Handle multi-planar images
	memInfo, _ := subGetImagePlaneMemoryInfo(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.oldState, GetState(p.sb.oldState), 0, nil, nil, img, VkImageAspectFlagBits(0))
	memTypeBits := memInfo.MemoryRequirements().MemoryTypeBits()
	memIndex := memoryTypeIndexFor(memTypeBits, phyDevMemProps, VkMemoryPropertyFlags(VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT))
	if memIndex < 0 {
		// fallback to use whatever type of memory available
		memIndex = memoryTypeIndexFor(memTypeBits, phyDevMemProps, VkMemoryPropertyFlags(0))
	}
	if memIndex < 0 {
		return ImageObjectʳ{}, func() {}, log.Errf(p.sb.ctx, fmt.Errorf("can't find an appropriate memory type index"), "[Creatig staging image same as image: %v]", img.VulkanHandle())
	}

	stagingImg, stagingImgMem, err := p.createImageAndBindMemory(img.Device(), img.Info(), memIndex)
	if err != nil {
		return ImageObjectʳ{}, func() {}, log.Errf(p.sb.ctx, err, "[Creating staging image same as image: %v]", img.VulkanHandle())
	}
	return stagingImg, func() {
		p.sb.write(p.sb.cb.VkDestroyImage(stagingImg.Device(), stagingImg.VulkanHandle(), memory.Nullptr))
		p.sb.write(p.sb.cb.VkFreeMemory(stagingImgMem.Device(), stagingImgMem.VulkanHandle(), memory.Nullptr))
	}, nil
}

// Create32BitUintColorStagingImagesForAspect creates stagining images with format
// RGBA32_UINT for the given image's specific, allocated backing memory for the
// new created images and bind memory for them, returns the created image
// objects in the new state of the state builder of the current image primer, a
// function to destroy the created image and backing memories, and an error in
// case of any error occur.
func (p *imagePrimer) Create32BitUintColorStagingImagesForAspect(img ImageObjectʳ, aspect VkImageAspectFlagBits, usages VkImageUsageFlags) ([]ImageObjectʳ, func(), error) {
	stagingImgs := []ImageObjectʳ{}
	stagingMems := []DeviceMemoryObjectʳ{}

	srcElementAndTexelInfo, err := subGetElementAndTexelBlockSize(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.oldState, GetState(p.sb.oldState), 0, nil, nil, img.Info().Fmt())
	if err != nil {
		return []ImageObjectʳ{}, func() {}, log.Errf(p.sb.ctx, err, "[Getting element size and texel block info]")
	}
	if srcElementAndTexelInfo.TexelBlockSize().Width() != 1 || srcElementAndTexelInfo.TexelBlockSize().Height() != 1 {
		// compressed formats are not supported
		return []ImageObjectʳ{}, func() {}, log.Errf(p.sb.ctx, err, "allocating staging images for compressed format images is not supported")
	}
	srcElementSize := srcElementAndTexelInfo.ElementSize()
	if aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT {
		srcElementSize, err = subGetDepthElementSize(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.oldState, GetState(p.sb.oldState), 0, nil, nil, img.Info().Fmt(), false)
		if err != nil {
			return []ImageObjectʳ{}, func() {}, log.Errf(p.sb.ctx, err, "[Getting element size for depth aspect] %v", err)
		}
	} else if aspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT {
		srcElementSize = 1
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
		return []ImageObjectʳ{}, func() {}, log.Errf(p.sb.ctx, nil, "unsupported aspect: %v", aspect)
	}
	stagingElementInfo, _ := subGetElementAndTexelBlockSize(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.oldState, GetState(p.sb.oldState), 0, nil, nil, stagingImgFormat)
	stagingElementSize := stagingElementInfo.ElementSize()

	stagingInfo := img.Info().Clone(api.CloneContext{})
	stagingInfo.SetDedicatedAllocationNV(NilDedicatedAllocationBufferImageCreateInfoNVʳ)
	stagingInfo.SetFmt(stagingImgFormat)
	stagingInfo.SetUsage(usages)

	dev := p.sb.s.Devices().Get(img.Device())
	phyDevMemProps := p.sb.s.PhysicalDevices().Get(dev.PhysicalDevice()).MemoryProperties()
	// TODO: Handle multi-planar images
	memInfo, _ := subGetImagePlaneMemoryInfo(p.sb.ctx, nil, api.CmdNoID, nil, p.sb.oldState, GetState(p.sb.oldState), 0, nil, nil, img, VkImageAspectFlagBits(0))
	memTypeBits := memInfo.MemoryRequirements().MemoryTypeBits()
	memIndex := memoryTypeIndexFor(memTypeBits, phyDevMemProps, VkMemoryPropertyFlags(VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT))
	if memIndex < 0 {
		// fallback to use whatever type of memory available
		memIndex = memoryTypeIndexFor(memTypeBits, phyDevMemProps, VkMemoryPropertyFlags(0))
	}
	if memIndex < 0 {
		return []ImageObjectʳ{}, func() {}, log.Errf(p.sb.ctx, nil, "can't find an appropriate memory type index")
	}

	covered := uint32(0)
	for covered < srcElementSize {
		stagingImg, mem, err := p.createImageAndBindMemory(dev.VulkanHandle(), stagingInfo, memIndex)
		if err != nil {
			return []ImageObjectʳ{}, func() {}, log.Errf(p.sb.ctx, err, "[Creating 32 bit wide staging images for image: %v, aspect: %v, usages: %v]", img.VulkanHandle(), aspect, usages)
		}
		stagingImgs = append(stagingImgs, stagingImg)
		stagingMems = append(stagingMems, mem)
		covered += stagingElementSize
	}

	free := func() {
		for _, img := range stagingImgs {
			p.sb.write(p.sb.cb.VkDestroyImage(img.Device(), img.VulkanHandle(), memory.Nullptr))
		}
		for _, mem := range stagingMems {
			p.sb.write(p.sb.cb.VkFreeMemory(mem.Device(), mem.VulkanHandle(), memory.Nullptr))
		}
	}
	return stagingImgs, free, nil
}

// ipLayoutInfo contains the layout info index by the image subresource.
type ipLayoutInfo interface {
	layoutOf(aspect VkImageAspectFlagBits, layer, level uint32) VkImageLayout
}

type ipLayoutInfoFromImage struct {
	img ImageObjectʳ
}

func (i *ipLayoutInfoFromImage) layoutOf(aspect VkImageAspectFlagBits, layer, level uint32) VkImageLayout {
	if _, ok := i.img.Aspects().Lookup(aspect); !ok {
		return VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED
	}
	if _, ok := i.img.Aspects().Get(aspect).Layers().Lookup(layer); !ok {
		return VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED
	}
	if _, ok := i.img.Aspects().Get(aspect).Layers().Get(layer).Levels().Lookup(level); !ok {
		return VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED
	}
	return i.img.Aspects().Get(aspect).Layers().Get(layer).Levels().Get(level).Layout()
}

// sameLayoutsOfImage creates an ipLayoutInfo, which when being queried, always
// returns the layout of the corresponding subresource in given image object.
func sameLayoutsOfImage(img ImageObjectʳ) ipLayoutInfo {
	return &ipLayoutInfoFromImage{img: img}
}

type ipLayoutInfoFromLayout struct {
	layout VkImageLayout
}

func (i *ipLayoutInfoFromLayout) layoutOf(aspect VkImageAspectFlagBits, layer, level uint32) VkImageLayout {
	return i.layout
}

// useSpecifiedLayout creates an ipLayoutInfo, which when being queried, always
// returns the given layout, for any given image subresource.
func useSpecifiedLayout(layout VkImageLayout) ipLayoutInfo {
	return &ipLayoutInfoFromLayout{layout: layout}
}

// ipImageSubresourceLayoutTransitionBarrier returns a VkImageMemoryBarrier
// built for layout transition of the subresource of the given image object
// specified with aspect bit, layer and level.
func ipImageSubresourceLayoutTransitionBarrier(sb *stateBuilder, imgObj ImageObjectʳ, aspect VkImageAspectFlagBits, layer, level uint32, oldLayout, newLayout VkImageLayout) VkImageMemoryBarrier {
	return NewVkImageMemoryBarrier(
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
		0, // pNext
		VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // srcAccessMask
		VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // dstAccessMask
		oldLayout,             // oldLayout
		newLayout,             // newLayout
		^uint32(0),            // srcQueueFamilyIndex
		^uint32(0),            // dstQueueFamilyIndex
		imgObj.VulkanHandle(), // image
		NewVkImageSubresourceRange(
			ipImageBarrierAspectFlags(aspect, imgObj.Info().Fmt()),
			level,
			1,
			layer,
			1,
		), // subresourceRange
	)
}

// ipImageLayoutTransitionBarriers returns a list of VkImageMemoryBarrier to
// transition all the subresources of the given image object, from the
// oldLayouts to newLayouts.
func ipImageLayoutTransitionBarriers(sb *stateBuilder, imgObj ImageObjectʳ, oldLayouts, newLayouts ipLayoutInfo) []VkImageMemoryBarrier {
	barriers := []VkImageMemoryBarrier{}
	walkImageSubresourceRange(sb, imgObj, sb.imageWholeSubresourceRange(imgObj),
		func(aspect VkImageAspectFlagBits, layer, level uint32, unused byteSizeAndExtent) {
			oldLayout := oldLayouts.layoutOf(aspect, layer, level)
			newLayout := newLayouts.layoutOf(aspect, layer, level)
			if newLayout == VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED || newLayout == VkImageLayout_VK_IMAGE_LAYOUT_PREINITIALIZED || oldLayout == newLayout {
				return
			}
			imageBarrier := ipImageSubresourceLayoutTransitionBarrier(
				sb,
				imgObj,
				aspect,
				layer,
				level,
				oldLayout,
				newLayout,
			)
			for _, barrier := range barriers {
				if barrier.Equals(imageBarrier) {
					return
				}
			}
			barriers = append(barriers, imageBarrier)
		})
	return barriers
}

// ipRecordImageMemoryBarriers records a VkCmdPipelineBarrier with a list of
// VkImageMemoryBarrier to the command buffer of the given queue command
// handler.
func ipRecordImageMemoryBarriers(sb *stateBuilder, queueHandler *queueCommandHandler, barriers ...VkImageMemoryBarrier) error {
	err := queueHandler.RecordCommands(sb, "", func(commandBuffer VkCommandBuffer) {
		sb.write(sb.cb.VkCmdPipelineBarrier(
			commandBuffer,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkDependencyFlags(0),
			0,
			memory.Nullptr,
			0,
			memory.Nullptr,
			uint32(len(barriers)),
			sb.MustAllocReadData(barriers).Ptr(),
		))
	})
	if err != nil {
		return log.Errf(sb.ctx, err, "failed at recording command for image memory barriers")
	}
	return nil
}

// free functions

func ipImageBarrierAspectFlags(aspect VkImageAspectFlagBits, fmt VkFormat) VkImageAspectFlags {
	switch fmt {
	case VkFormat_VK_FORMAT_D16_UNORM_S8_UINT,
		VkFormat_VK_FORMAT_D24_UNORM_S8_UINT,
		VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
		aspect |= VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT |
			VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT
	}
	return VkImageAspectFlags(aspect)
}

func extendToMultipleOf8(dataPtr *[]uint8) {
	l := uint64(len(*dataPtr))
	nl := nextMultipleOf(l, 8)
	zeros := make([]uint8, nl-l)
	*dataPtr = append(*dataPtr, zeros...)
}

func unpackDataForPriming(ctx context.Context, data []uint8, srcFmt VkFormat, aspect VkImageAspectFlagBits) ([]uint8, VkFormat, error) {
	ctx = log.Enter(ctx, "unpackDataForPriming")
	var sf *image.Format
	var err error
	var dstFmt VkFormat
	switch aspect {
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
		sf, err = getImageFormatFromVulkanFormat(srcFmt)
		if err != nil {
			return []uint8{}, dstFmt, log.Errf(ctx, err, "[Getting image.Format for VkFormat: %v, aspect: %v]", srcFmt, aspect)
		}
		dstFmt = stagingColorImageBufferFormat

	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT:
		sf, err = getDepthImageFormatFromVulkanFormat(srcFmt)
		if err != nil {
			return []uint8{}, dstFmt, log.Errf(ctx, err, "[Getting image.Format for VkFormat: %v, aspect: %v]", srcFmt, aspect)
		}
		dstFmt = stagingDepthStencilImageBufferFormat

	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		sf, err = getImageFormatFromVulkanFormat(VkFormat_VK_FORMAT_S8_UINT)
		if err != nil {
			return []uint8{}, dstFmt, log.Errf(ctx, err, "[Getting image.Format for VkFormat: %v, aspect: %v]", srcFmt, aspect)
		}
		dstFmt = stagingDepthStencilImageBufferFormat

	default:
		return []uint8{}, dstFmt, log.Errf(ctx, nil, "unsupported aspect: %v", aspect)
	}

	df, err := getImageFormatFromVulkanFormat(dstFmt)
	if err != nil {
		return []uint8{}, dstFmt, log.Errf(ctx, err, "[Getting image.Format for VkFormat %v]", dstFmt)
	}
	unpacked, err := unpackData(ctx, data, sf, df)
	if err != nil {
		return []uint8{}, dstFmt, err
	}
	return unpacked, dstFmt, nil
}

func unpackData(ctx context.Context, data []uint8, srcFmt, dstFmt *image.Format) ([]uint8, error) {
	ctx = log.Enter(ctx, "unpackData")
	var err error
	if srcFmt.GetUncompressed() == nil {
		return []uint8{}, log.Errf(ctx, nil, "compressed format: %v is not supported", srcFmt)
	}
	if dstFmt.GetUncompressed() == nil {
		return []uint8{}, log.Errf(ctx, nil, "compressed format: %v is not supported", dstFmt)
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
			return []uint8{}, log.Errf(ctx, nil, "[Building src format: %v] unsuppored channel in source data format: %v", sf, sc.Channel)
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
			return []uint8{}, log.Errf(ctx, nil, "[Building dst format for: %v] %s", sf, "DataType other than stream.Integer and stream.Float are not handled.")
		}
	}

	converted, err := stream.Convert(df, sf, data)
	if err != nil {
		return []uint8{}, log.Errf(ctx, err, "[Converting data from %v to %v]", sf, df)
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
	retData, err := image.Convert(data, int(extent.Width()), int(extent.Height()), int(extent.Depth()), sf, df)
	if err != nil {
		return []uint8{}, dstFmt, err
	}
	return retData, dstFmt, nil
}

func isDenseBound(img ImageObjectʳ) bool {
	return img.PlaneMemoryInfo().Len() > 0 && func() bool {
		for _, m := range img.PlaneMemoryInfo().All() {
			if m.BoundMemory().IsNil() {
				return false
			}
		}
		return true
	}()
}

func isSparseBound(img ImageObjectʳ) bool {
	return (img.SparseImageMemoryBindings().Len() > 0 || img.OpaqueSparseMemoryBindings().Len() > 0) && ((uint64(img.Info().Flags()) & uint64(VkImageCreateFlagBits_VK_IMAGE_CREATE_SPARSE_BINDING_BIT)) != 0)
}

func isSparseResidency(img ImageObjectʳ) bool {
	return isSparseBound(img) &&
		((uint32(img.Info().Flags()) & uint32(VkImageCreateFlagBits_VK_IMAGE_CREATE_SPARSE_RESIDENCY_BIT)) != 0)
}

func vkCreateImage(sb *stateBuilder, dev VkDevice, info ImageInfo, handle VkImage) {
	pNext := NewVoidᶜᵖ(memory.Nullptr)
	if !info.DedicatedAllocationNV().IsNil() {
		pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
			NewVkDedicatedAllocationImageCreateInfoNV(
				VkStructureType_VK_STRUCTURE_TYPE_DEDICATED_ALLOCATION_IMAGE_CREATE_INFO_NV, // sType
				0, // pNext
				info.DedicatedAllocationNV().DedicatedAllocation(), // dedicatedAllocation
			),
		).Ptr())
	}

	if !info.ViewFormatList().IsNil() {
		pNext = NewVoidᶜᵖ(sb.MustAllocReadData(
			NewVkImageFormatListCreateInfoKHR(
				VkStructureType_VK_STRUCTURE_TYPE_IMAGE_FORMAT_LIST_CREATE_INFO_KHR, // sType
				pNext, // pNext
				uint32(info.ViewFormatList().ViewFormats().Len()),                                    // viewFormatCount
				NewVkFormatᶜᵖ(sb.MustUnpackReadMap(info.ViewFormatList().ViewFormats().All()).Ptr()), // pViewFormats
			),
		).Ptr())
	}

	create := sb.cb.VkCreateImage(
		dev, sb.MustAllocReadData(
			NewVkImageCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO, // sType
				pNext,                                   // pNext
				info.Flags(),                            // flags
				info.ImageType(),                        // imageType
				info.Fmt(),                              // format
				info.Extent(),                           // extent
				info.MipLevels(),                        // mipLevels
				info.ArrayLayers(),                      // arrayLayers
				info.Samples(),                          // samples
				info.Tiling(),                           // tiling
				info.Usage(),                            // usage
				info.SharingMode(),                      // sharingMode
				uint32(info.QueueFamilyIndices().Len()), // queueFamilyIndexCount
				NewU32ᶜᵖ(sb.MustUnpackReadMap(info.QueueFamilyIndices().All()).Ptr()), // pQueueFamilyIndices
				info.InitialLayout(), // initialLayout
			)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	)

	if sb.s.Images().Contains(handle) {
		obj := sb.s.Images().Get(handle)
		fetchedReq := MakeFetchedImageMemoryRequirements()
		for p, pmi := range obj.PlaneMemoryInfo().All() {
			fetchedReq.PlaneBitsToMemoryRequirements().Add(p, pmi.MemoryRequirements())
		}
		for b, sparseReq := range obj.SparseMemoryRequirements().All() {
			fetchedReq.AspectBitsToSparseMemoryRequirements().Add(b, sparseReq)
		}
		create.Extras().Add(fetchedReq)
	}

	sb.write(create)
}

func vkGetImageMemoryRequirements(sb *stateBuilder, dev VkDevice, handle VkImage, memReq VkMemoryRequirements) {
	sb.write(sb.cb.VkGetImageMemoryRequirements(
		dev, handle, sb.MustAllocWriteData(memReq).Ptr(),
	))
}

func vkAllocateMemory(sb *stateBuilder, dev VkDevice, size VkDeviceSize, memTypeIndex uint32, handle VkDeviceMemory) {
	sb.write(sb.cb.VkAllocateMemory(
		dev,
		NewVkMemoryAllocateInfoᶜᵖ(sb.MustAllocReadData(
			NewVkMemoryAllocateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, // sType
				0,            // pNext
				size,         // allocationSize
				memTypeIndex, // memoryTypeIndex
			)).Ptr()),
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
		sb.MustAllocReadData(NewVkDescriptorSetLayoutCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO, // sType
			0,                     // pNext
			0,                     // flags
			uint32(len(bindings)), // bindingCount
			NewVkDescriptorSetLayoutBindingᶜᵖ(sb.MustAllocReadData(bindings).Ptr()), // pBindings
		)).Ptr(),
		NewVoidᶜᵖ(memory.Nullptr),
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func vkAllocateDescriptorSet(sb *stateBuilder, dev VkDevice, pool VkDescriptorPool, layout VkDescriptorSetLayout, handle VkDescriptorSet) {
	sb.write(sb.cb.VkAllocateDescriptorSets(
		dev,
		sb.MustAllocReadData(NewVkDescriptorSetAllocateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO, // sType
			0,    // pNext
			pool, // descriptorPool
			1,    // descriptorSetCount
			NewVkDescriptorSetLayoutᶜᵖ(sb.MustAllocReadData(layout).Ptr()), // pSetLayouts
		)).Ptr(),
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func vkCreatePipelineLayout(sb *stateBuilder, dev VkDevice, setLayouts []VkDescriptorSetLayout, pushConstantRanges []VkPushConstantRange, handle VkPipelineLayout) {
	createInfo := NewVkPipelineLayoutCreateInfo(
		VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO, // sType
		0,                       // pNext
		0,                       // flags
		uint32(len(setLayouts)), // setLayoutCount
		NewVkDescriptorSetLayoutᶜᵖ(sb.MustAllocReadData(setLayouts).Ptr()),       // pSetLayouts
		uint32(len(pushConstantRanges)),                                          // pushConstantRangeCount
		NewVkPushConstantRangeᶜᵖ(sb.MustAllocReadData(pushConstantRanges).Ptr()), // pPushConstantRanges
	)
	sb.write(sb.cb.VkCreatePipelineLayout(
		dev,
		NewVkPipelineLayoutCreateInfoᶜᵖ(sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func vkCreateShaderModule(sb *stateBuilder, dev VkDevice, code []uint32, handle VkShaderModule) {
	createInfo := NewVkShaderModuleCreateInfo(
		VkStructureType_VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO, // sType
		0,                        // pNext
		0,                        // flags
		memory.Size(len(code)*4), // codeSize
		NewU32ᶜᵖ(sb.MustAllocReadData(code).Ptr()), // pCode
	)

	descriptors, err := shadertools.ParseAllDescriptorSets(code)
	u := MakeDescriptorInfo()
	dsc := u.Descriptors()
	if err != nil {
		log.E(sb.ctx, "Could not parse SPIR-V")
	} else {
		for name, desc := range descriptors {
			d := NewU32ːDescriptorUsageᵐ()
			for _, set := range desc {
				for _, binding := range set {
					d.Add(uint32(d.Len()),
						NewDescriptorUsage(

							binding.Set,
							binding.Binding,
							binding.DescriptorCount))
				}
			}
			dsc.Add(name, d)
		}
	}
	csb := sb.cb.VkCreateShaderModule(
		dev,
		NewVkShaderModuleCreateInfoᶜᵖ(sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	)
	csb.Extras().Add(u)
	sb.write(csb)
}

func vkCreateDescriptorPool(sb *stateBuilder, dev VkDevice, flags VkDescriptorPoolCreateFlags, maxSet uint32, poolSizes []VkDescriptorPoolSize, handle VkDescriptorPool) {
	sb.write(sb.cb.VkCreateDescriptorPool(
		dev,
		sb.MustAllocReadData(NewVkDescriptorPoolCreateInfo(
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO, // sType
			0,                      // pNext
			flags,                  // flags
			maxSet,                 // maxSets
			uint32(len(poolSizes)), // poolSizeCount
			NewVkDescriptorPoolSizeᶜᵖ(sb.MustAllocReadData(poolSizes).Ptr()), // pPoolSizes
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func writeDescriptorSet(sb *stateBuilder, dev VkDevice, descSet VkDescriptorSet, dstBinding, dstArrayElement uint32, descType VkDescriptorType, imgInfoList []VkDescriptorImageInfo, bufInfoList []VkDescriptorBufferInfo, texelBufInfoList []VkBufferView) {
	write := NewVkWriteDescriptorSet(
		VkStructureType_VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, // sType
		0,               // pNext
		descSet,         // dstSet
		dstBinding,      // dstBinding
		dstArrayElement, // dstArrayElement
		uint32(len(imgInfoList)+len(bufInfoList)+len(texelBufInfoList)), // descriptorCount
		descType, // descriptorType
		NewVkDescriptorImageInfoᶜᵖ(sb.MustAllocReadData(imgInfoList).Ptr()),  // pImageInfo
		NewVkDescriptorBufferInfoᶜᵖ(sb.MustAllocReadData(bufInfoList).Ptr()), // pBufferInfo
		NewVkBufferViewᶜᵖ(sb.MustAllocReadData(texelBufInfoList).Ptr()),      // pTexelBufferView
	)

	sb.write(sb.cb.VkUpdateDescriptorSets(
		dev,
		1,
		NewVkWriteDescriptorSetᶜᵖ(sb.MustAllocReadData(write).Ptr()),
		0,
		memory.Nullptr,
	))
}

func walkImageSubresourceRange(sb *stateBuilder, img ImageObjectʳ, rng VkImageSubresourceRange, f func(aspect VkImageAspectFlagBits, layer, level uint32, levelSize byteSizeAndExtent)) {
	layerCount, _ := subImageSubresourceLayerCount(sb.ctx, nil, api.CmdNoID, nil, sb.oldState, nil, 0, nil, nil, img, rng)
	levelCount, _ := subImageSubresourceLevelCount(sb.ctx, nil, api.CmdNoID, nil, sb.oldState, nil, 0, nil, nil, img, rng)
	for _, aspect := range sb.imageAspectFlagBits(img, rng.AspectMask()) {
		for i := uint32(0); i < levelCount; i++ {
			level := rng.BaseMipLevel() + i
			divisor, _ := subGetAspectSizeDivisor(sb.ctx, nil, api.CmdNoID, nil, sb.oldState, nil, 0, nil, nil, img.Info().Fmt(), aspect)
			levelSize := sb.levelSize(img.Info().Extent(), img.Info().Fmt(), level, aspect)
			levelSize.width /= uint64(divisor.Width())
			levelSize.height /= uint64(divisor.Height())
			for j := uint32(0); j < layerCount; j++ {
				layer := rng.BaseArrayLayer() + j
				f(aspect, layer, level, levelSize)
			}
		}
	}
}

func walkSparseImageMemoryBindings(sb *stateBuilder, img ImageObjectʳ, f func(aspect VkImageAspectFlagBits, layer, level uint32, blockData SparseBoundImageBlockInfoʳ)) {
	for _, aspect := range img.SparseImageMemoryBindings().Keys() {
		aspectData := img.SparseImageMemoryBindings().Get(aspect)
		for _, layer := range aspectData.Layers().Keys() {
			layerData := aspectData.Layers().Get(layer)
			for _, level := range layerData.Levels().Keys() {
				levelData := layerData.Levels().Get(level)
				for _, block := range levelData.Blocks().Keys() {
					blockData := levelData.Blocks().Get(block)
					f(VkImageAspectFlagBits(aspect), layer, level, blockData)
				}
			}
		}
	}
}

func roundUp(dividend, divisor uint64) uint64 {
	return (dividend + divisor - 1) / divisor
}

// debugReportObjectType takes a Vulkan handle and returns the
// VkDebugReportObjectTypeEXT for that handle.
func debugReportObjectType(object interface{}) VkDebugReportObjectTypeEXT {
	switch object.(type) {
	case VkInstance:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_INSTANCE_EXT
	case VkPhysicalDevice:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_PHYSICAL_DEVICE_EXT
	case VkDevice:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_DEVICE_EXT
	case VkQueue:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_QUEUE_EXT
	case VkSemaphore:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_SEMAPHORE_EXT
	case VkCommandBuffer:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_COMMAND_BUFFER_EXT
	case VkFence:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_FENCE_EXT
	case VkDeviceMemory:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_DEVICE_MEMORY_EXT
	case VkBuffer:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_BUFFER_EXT
	case VkImage:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_IMAGE_EXT
	case VkEvent:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_EVENT_EXT
	case VkQueryPool:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_QUERY_POOL_EXT
	case VkBufferView:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_BUFFER_VIEW_EXT
	case VkImageView:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_IMAGE_VIEW_EXT
	case VkShaderModule:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_SHADER_MODULE_EXT
	case VkPipelineCache:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_PIPELINE_CACHE_EXT
	case VkPipelineLayout:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_PIPELINE_LAYOUT_EXT
	case VkRenderPass:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_RENDER_PASS_EXT
	case VkPipeline:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_PIPELINE_EXT
	case VkDescriptorSetLayout:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_DESCRIPTOR_SET_LAYOUT_EXT
	case VkSampler:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_SAMPLER_EXT
	case VkDescriptorPool:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_DESCRIPTOR_POOL_EXT
	case VkDescriptorSet:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_DESCRIPTOR_SET_EXT
	case VkFramebuffer:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_FRAMEBUFFER_EXT
	case VkCommandPool:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_COMMAND_POOL_EXT
	case VkSurfaceKHR:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_SURFACE_KHR_EXT
	case VkSwapchainKHR:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_SWAPCHAIN_KHR_EXT
	case VkDebugReportCallbackEXT:
		return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_DEBUG_REPORT_EXT
	}
	return VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_UNKNOWN_EXT
}

// attachDebugMarkerName writes a VkDebugMarkerSetObjectNameEXT command to
// specify a debug marker name to the given Vulkan handle object.
func attachDebugMarkerName(sb *stateBuilder, nm debugMarkerName, dev VkDevice, object interface{}) error {
	objectType := debugReportObjectType(object)
	if objectType == VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_UNKNOWN_EXT {
		return fmt.Errorf("unknown object type")
	}
	objectValue, ok := object.(uint64)
	if !ok {
		return fmt.Errorf("object: %v can not be cast to uint64", object)
	}
	sb.write(sb.cb.VkDebugMarkerSetObjectNameEXT(
		dev,
		NewVkDebugMarkerObjectNameInfoEXTᵖ(sb.MustAllocReadData(
			NewVkDebugMarkerObjectNameInfoEXT(
				VkStructureType_VK_STRUCTURE_TYPE_DEBUG_MARKER_OBJECT_NAME_INFO_EXT, // sType
				0,           // pNext
				objectType,  // objectType
				objectValue, // object
				NewCharᶜᵖ(sb.MustAllocReadData(nm.String()).Ptr()), // pObjectName
			)).Ptr(),
		),
		VkResult_VK_SUCCESS,
	))
	return nil
}

// ipDescriptorSetLayoutBindingInfo describes a binding for descriptor set
// binding used to create descriptor set layout.
type ipDescriptorSetLayoutBindingInfo struct {
	descriptorType VkDescriptorType
	count          uint32
	stages         VkShaderStageFlags
}

// ipDescriptorSetLayoutInfo contains the descriptor set binding used to create
// descriptor set layout for image priming.
type ipDescriptorSetLayoutInfo struct {
	bindings map[uint32]ipDescriptorSetLayoutBindingInfo
}

func ipCreateDescriptorSetLayout(sb *stateBuilder, nm debugMarkerName, dev VkDevice, info ipDescriptorSetLayoutInfo) VkDescriptorSetLayout {
	bindings := []VkDescriptorSetLayoutBinding{}
	for b, bInfo := range info.bindings {
		bindings = append(bindings,
			NewVkDescriptorSetLayoutBinding(
				b,                              // binding
				bInfo.descriptorType,           // descriptorType
				bInfo.count,                    // descriptorCount
				bInfo.stages,                   // stageFlags
				NewVkSamplerᶜᵖ(memory.Nullptr), // pImmutableSamplers
			),
		)
	}
	handle := VkDescriptorSetLayout(newUnusedID(true, func(x uint64) bool {
		return GetState(sb.newState).DescriptorSetLayouts().Contains(VkDescriptorSetLayout(x)) || GetState(sb.oldState).DescriptorSetLayouts().Contains(VkDescriptorSetLayout(x))
	}))
	vkCreateDescriptorSetLayout(sb, dev, bindings, handle)
	if len(nm) > 0 {
		attachDebugMarkerName(sb, nm, dev, handle)
	}
	return handle
}

func ipCreatePipelineLayout(sb *stateBuilder, nm debugMarkerName, dev VkDevice,
	descSetLayouts []VkDescriptorSetLayout, pushConstantStages VkShaderStageFlags,
	pushConstantSize uint32) VkPipelineLayout {
	handle := VkPipelineLayout(newUnusedID(true, func(x uint64) bool {
		return GetState(sb.newState).PipelineLayouts().Contains(VkPipelineLayout(x)) ||
			GetState(sb.oldState).PipelineLayouts().Contains(VkPipelineLayout(x))
	}))
	vkCreatePipelineLayout(sb, dev,
		descSetLayouts,
		[]VkPushConstantRange{NewVkPushConstantRange(
			pushConstantStages, // stageFlags
			0,                  // offset
			pushConstantSize,   // size
		)},
		handle,
	)
	if len(nm) > 0 {
		attachDebugMarkerName(sb, nm, dev, handle)
	}
	return handle
}

// ipShaderModuleInfo contains all the information used to select a image
// priming shader.
type ipShaderModuleInfo struct {
	stage        VkShaderStageFlagBits
	inputFormat  VkFormat
	inputAspect  VkImageAspectFlagBits
	outputFormat VkFormat
	outputAspect VkImageAspectFlagBits
	outputType   VkImageType
}

func ipCreateShaderModule(sb *stateBuilder, nm debugMarkerName, dev VkDevice, info ipShaderModuleInfo) (VkShaderModule, error) {
	var err error
	code := []uint32{}
	switch info.stage {
	case VkShaderStageFlagBits_VK_SHADER_STAGE_VERTEX_BIT:
		code, err = ipRenderVertexShaderSpirv()
	case VkShaderStageFlagBits_VK_SHADER_STAGE_FRAGMENT_BIT:
		switch info.outputAspect {
		case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT:
			code, err = ipRenderDepthShaderSpirv(info.outputFormat)
		case VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
			code, err = ipRenderStencilShaderSpirv()
		case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
			// Prime from host image data, the staging image is always in format of VkFormat_VK_FORMAT_R32G32B32A32_UINT
			if info.inputFormat == stagingColorImageBufferFormat {
				code, err = ipRenderColorShaderSpirv(info.outputFormat)
			} else { // Otherwise, the input and output format should be the same.
				code, err = ipCopyByRenderShaderSpirv(info.outputFormat)
			}
		default:
			return VkShaderModule(0), fmt.Errorf("Unsupported output aspect: %v for stage: %v", info.outputAspect, info.stage)
		}
	case VkShaderStageFlagBits_VK_SHADER_STAGE_COMPUTE_BIT:
		code, err = ipComputeShaderSpirv(info.outputFormat, info.outputAspect, info.inputFormat, info.inputAspect, info.outputType)
	default:
		return VkShaderModule(0), fmt.Errorf("Unsupported stage: %v", info.stage)
	}
	if err != nil {
		return VkShaderModule(0), log.Errf(sb.ctx, err, "failed at getting SPIR-V code")
	}
	if len(code) == 0 {
		return VkShaderModule(0), fmt.Errorf("Empty SPIR-V code")
	}

	handle := VkShaderModule(newUnusedID(true, func(x uint64) bool {
		return GetState(sb.newState).ShaderModules().Contains(VkShaderModule(x)) ||
			GetState(sb.oldState).ShaderModules().Contains(VkShaderModule(x))
	}))
	vkCreateShaderModule(sb, dev, code, handle)
	if len(nm) > 0 {
		attachDebugMarkerName(sb, nm, dev, handle)
	}
	return handle, nil
}

// ipImageView contains all the information about the image subresource to
// create a image view for image priming
type ipImageViewInfo struct {
	image  VkImage
	aspect VkImageAspectFlagBits
	layer  uint32
	level  uint32
}

func ipCreateImageView(sb *stateBuilder, nm debugMarkerName, dev VkDevice, info ipImageViewInfo) VkImageView {
	imgObj := GetState(sb.newState).Images().Get(info.image)
	viewType := VkImageViewType_VK_IMAGE_VIEW_TYPE_2D
	if imgObj.Info().ImageType() == VkImageType_VK_IMAGE_TYPE_3D {
		viewType = VkImageViewType_VK_IMAGE_VIEW_TYPE_3D
	} else if imgObj.Info().ImageType() == VkImageType_VK_IMAGE_TYPE_1D {
		viewType = VkImageViewType_VK_IMAGE_VIEW_TYPE_1D
	}
	handle := VkImageView(newUnusedID(true, func(x uint64) bool {
		return GetState(sb.newState).ImageViews().Contains(VkImageView(x)) ||
			GetState(sb.oldState).ImageViews().Contains(VkImageView(x))
	}))
	sb.write(sb.cb.VkCreateImageView(
		dev,
		NewVkImageViewCreateInfoᶜᵖ(sb.MustAllocReadData(
			NewVkImageViewCreateInfo(
				VkStructureType_VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO, // sType
				0,          // pNext
				0,          // flags
				info.image, // image
				viewType,   // viewType
				GetState(sb.newState).Images().Get(info.image).Info().Fmt(), // format
				NewVkComponentMapping( // components
					VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY, // r
					VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY, // g
					VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY, // b
					VkComponentSwizzle_VK_COMPONENT_SWIZZLE_IDENTITY, // a
				),
				NewVkImageSubresourceRange( // subresourceRange
					VkImageAspectFlags(info.aspect), // aspectMask
					info.level,                      // baseMipLevel
					1,                               // levelCount
					info.layer,                      // baseArrayLayer
					1,                               // layerCount
				),
			)).Ptr()),
		memory.Nullptr,
		sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	if len(nm) > 0 {
		attachDebugMarkerName(sb, nm, dev, handle)
	}
	return handle
}
