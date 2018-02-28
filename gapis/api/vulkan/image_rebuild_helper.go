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
	"context"
	"encoding/binary"
	"fmt"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
)

const (
	// Only uncompressed and unpacked formats shall be used for staging.
	colorStagingFormat = VkFormat_VK_FORMAT_R32G32B32A32_UINT
	depthStagingFormat = VkFormat_VK_FORMAT_R32_UINT
)

type imageRebuildHelper struct {
	sb                       *stateBuilder
	stagingImages            map[VkImage]*ImageObject
	tempMemories             map[VkDeviceMemory]*DeviceMemoryObject
	tempImageViews           map[VkImageView]*ImageViewObject
	tempBuffers              map[VkBuffer]*BufferObject
	tempDescriptorPools      map[VkDescriptorPool]*DescriptorPoolObject
	tempDescriptorSetLayouts map[VkDescriptorSetLayout]*DescriptorSetLayoutObject
	tempDescriptorSets       map[VkDescriptorSet]*DescriptorSetObject
	tempPipelineLayouts      map[VkPipelineLayout]*PipelineLayoutObject
	tempGfxPipelines         map[VkPipeline]*GraphicsPipelineObject
	tempRenderPasses         map[VkRenderPass]*RenderPassObject
	tempFramebuffers         map[VkFramebuffer]*FramebufferObject
	tempShaders              map[VkShaderModule]*ShaderModuleObject
}

func newImageRebuildHelper(sb *stateBuilder) *imageRebuildHelper {
	return &imageRebuildHelper{
		sb:                       sb,
		stagingImages:            map[VkImage]*ImageObject{},
		tempMemories:             map[VkDeviceMemory]*DeviceMemoryObject{},
		tempImageViews:           map[VkImageView]*ImageViewObject{},
		tempBuffers:              map[VkBuffer]*BufferObject{},
		tempDescriptorPools:      map[VkDescriptorPool]*DescriptorPoolObject{},
		tempDescriptorSetLayouts: map[VkDescriptorSetLayout]*DescriptorSetLayoutObject{},
		tempDescriptorSets:       map[VkDescriptorSet]*DescriptorSetObject{},
		tempPipelineLayouts:      map[VkPipelineLayout]*PipelineLayoutObject{},
		tempGfxPipelines:         map[VkPipeline]*GraphicsPipelineObject{},
		tempRenderPasses:         map[VkRenderPass]*RenderPassObject{},
		tempFramebuffers:         map[VkFramebuffer]*FramebufferObject{},
		tempShaders:              map[VkShaderModule]*ShaderModuleObject{},
	}
}

// allocateStagingImagesFor creates an array of staging images for the given
// image whose data to be recovered.
func (h *imageRebuildHelper) allocateStagingImagesFor(img *ImageObject) ([]*ImageObject, error) {
	stagingImgs := []*ImageObject{}
	empty := []*ImageObject{}
	imgTexelBlockSizeInBytes, err := texelBlockSizeInBytes(h.sb.ctx, h.sb.oldState, img.Info.Format)
	if err != nil {
		return empty, err
	}
	tbw, err := texelBlockWidth(h.sb.ctx, h.sb.oldState, img.Info.Format)
	if err != nil {
		return empty, err
	}
	tbh, err := texelBlockHeight(h.sb.ctx, h.sb.oldState, img.Info.Format)
	if err != nil {
		return empty, err
	}
	numPixelsInImgTexelBlock := tbw * tbh
	stagingFormat := stagingImageFormat(img)
	if stagingFormat == VkFormat_VK_FORMAT_UNDEFINED {
		return empty, fmt.Errorf("appropriate staging image format not found")
	}
	stagingImgPixelSizeInBytes, err := texelBlockSizeInBytes(h.sb.ctx, h.sb.oldState, stagingFormat)
	if err != nil {
		return empty, err
	}
	index := uint32(0)
	covered := uint32(0)
	for covered < imgTexelBlockSizeInBytes {
		// Create staging image handle
		handle := VkImage(newUnusedID(true, func(x uint64) bool {
			inState := h.sb.s.Images.Contains(VkImage(x))
			_, inHelper := h.stagingImages[VkImage(x)]
			return inState || inHelper
		}))
		info := img.Info
		info.Format = stagingFormat
		info.Usage = VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT | VkImageUsageFlagBits_VK_IMAGE_USAGE_INPUT_ATTACHMENT_BIT | VkImageUsageFlagBits_VK_IMAGE_USAGE_SAMPLED_BIT)
		h.vkCreateImage(img.Device, &info, handle)
		stagingImg := GetState(h.sb.newState).Images.Get(handle)
		h.stagingImages[handle] = stagingImg
		// Query the memory requirements so validation layers are happy
		memReq := VkMemoryRequirements{}
		h.vkGetImageMemoryRequirements(img.Device, handle, &memReq)
		mem, err := h.allocateTempMemoryForStagingImage(stagingImg, img)
		if err != nil {
			return empty, fmt.Errorf("[Allocating and binding memory for staging image] %v", err)
		}
		h.vkBindImageMemory(img.Device, handle, mem.VulkanHandle, 0)
		stagingImgs = append(stagingImgs, stagingImg)
		covered += numPixelsInImgTexelBlock * stagingImgPixelSizeInBytes
		index++
	}
	return stagingImgs, nil
}

func (h *imageRebuildHelper) allocateTempMemoryForStagingImage(stagingImg, origImg *ImageObject) (*DeviceMemoryObject, error) {
	// Get the allocation size and memory type index
	inferredSize, err := subInferImageSize(h.sb.ctx, nil, api.CmdNoID, nil, h.sb.oldState, GetState(h.sb.oldState), 0, nil, stagingImg)
	if err != nil {
		return nil, fmt.Errorf("[Inferring image size in bytes] %v", err)
	}
	devObj := h.sb.s.Devices.Get(origImg.Device)
	phyDevMemProps := h.sb.s.PhysicalDevices.Get(devObj.PhysicalDevice).MemoryProperties
	memTypeBits := origImg.MemoryRequirements.MemoryTypeBits

	index := memoryTypeIndexFor(memTypeBits, &phyDevMemProps, VkMemoryPropertyFlags(VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT))
	if index < 0 {
		// Fallback to select whatever type of memory available.
		index = memoryTypeIndexFor(memTypeBits, &phyDevMemProps, VkMemoryPropertyFlags(0))
	}
	if index < 0 {
		return nil, fmt.Errorf("cannot find an appropriate memory type index")
	}
	memoryTypeIndex := uint32(index)

	// Allocate memory
	handle := VkDeviceMemory(newUnusedID(true, func(x uint64) bool {
		inState := h.sb.s.DeviceMemories.Contains(VkDeviceMemory(x))
		_, inHelper := h.tempMemories[VkDeviceMemory(x)]
		return inState || inHelper
	}))

	h.sb.write(h.sb.cb.VkAllocateMemory(
		stagingImg.Device,
		NewVkMemoryAllocateInfoᶜᵖ(h.sb.MustAllocReadData(
			VkMemoryAllocateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				inferredSize * 2,
				// Since we cannot guess how much the driver will actually
				// request of us, overallocate by a factor of 2. This should
				// be enough.
				memoryTypeIndex,
			}).Ptr()),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))

	mem := GetState(h.sb.newState).DeviceMemories.Get(handle)
	h.tempMemories[handle] = mem
	return mem, nil
}

// unpackDataForCopy unpacks the data of an image format to another image
// format. The first encountered error will be returned if there are any. In the
// case of error, the unpacked data will be filled with zero values, so that the
// returned unpacked data should have expected number of bytes, if the dst image
// format is known.
func (h *imageRebuildHelper) unpackDataForCopy(src []uint8, srcFmt VkFormat, dstImgIndex uint32, dstFmt VkFormat, extent VkExtent3D) ([]uint8, error) {
	var firstErr error
	// Format unchanged, no need to unpack.
	if srcFmt == dstFmt {
		return src, nil
	}
	// Allocate for the unpacked data
	e := h.sb.levelSize(extent, dstFmt, 0)
	dst := make([]uint8, e.alignedLevelSize)
	srcBW, err := texelBlockWidth(h.sb.ctx, h.sb.oldState, srcFmt)
	if err != nil {
		return dst, err
	}
	srcBH, err := texelBlockHeight(h.sb.ctx, h.sb.oldState, srcFmt)
	if err != nil {
		return dst, err
	}
	srcBS, err := texelBlockSizeInBytes(h.sb.ctx, h.sb.oldState, srcFmt)
	if err != nil {
		return dst, err
	}
	widthInSrcBlocks := roundUpTo(extent.Width, srcBW)
	heightInSrcBlocks := roundUpTo(extent.Height, srcBH)
	depthSrcBlockPitch := widthInSrcBlocks * heightInSrcBlocks
	heightSrcBlockPitch := widthInSrcBlocks

	dstPixelSize, err := texelBlockSizeInBytes(h.sb.ctx, h.sb.oldState, dstFmt)
	if err != nil {
		return dst, err
	}
	depthDstPitch := extent.Width * extent.Height * dstPixelSize
	heightDstPitch := extent.Width * dstPixelSize

	for z := uint32(0); z < extent.Depth; z++ {
		for by := uint32(0); by < heightInSrcBlocks; by++ {
			for bx := uint32(0); bx < widthInSrcBlocks; bx++ {
				blockN := z*depthSrcBlockPitch + by*heightSrcBlockPitch + bx
				srcOffset := blockN * srcBS
				blockData := src[srcOffset : srcOffset+srcBS]
				for dy := uint32(0); dy < srcBH; dy++ {
					dstOffset := (by*srcBH+dy)*heightDstPitch + z*depthDstPitch
					for dx := uint32(0); dx < srcBW; dx++ {
						dstOffset += (bx*srcBW + dx) * dstPixelSize
						if dstOffset+dstPixelSize <= uint32(len(dst)) {
							unpacked, err := h.unpackTexelBlock(blockData, srcFmt, dstImgIndex, dx, dy)
							if err != nil && firstErr == nil {
								firstErr = err
							}
							copy(dst[dstOffset:dstOffset+dstPixelSize], unpacked)
						}
					}
				}
			}
		}
	}
	return dst, firstErr
}

// unpackTexelBlock takes an slice of the src data which is encoded in the src
// format, unpacks each channel of the data to a 4-byte Uint32 or Float32 and
// keeps the value unchanged. If the source image channel is wider than 32 bit,
// [dstImgIndex * 32 : (dstImgIndex+1)*32) bits will be unpacked. If the block
// is compressed, dx, dy will be used to identify the specific pixel in the block
// whose data should be unpacked.
func (h *imageRebuildHelper) unpackTexelBlock(src []uint8, srcFmt VkFormat, dstImgIndex uint32, dx, dy uint32) ([]uint8, error) {
	var buf bytes.Buffer
	var firstUnpackingErr error
	writeInt := func(startBit, oneOverEndBit uint32, sign signedness) {
		var u uint32
		u, err := unpackIntToUint32(src, startBit, oneOverEndBit, sign)
		if err != nil && firstUnpackingErr == nil {
			firstUnpackingErr = err
		}
		binary.Write(&buf, binary.LittleEndian, u)
	}
	writeFloat := func(fracStart, oneOverFracEnd, expStart, oneOverExpEnd uint32, sign signedness, signBit uint32) {
		var f float32
		f, err := unpackFloatToFloat32(src, fracStart, oneOverFracEnd, expStart, oneOverExpEnd, sign, signBit)
		if err != nil && firstUnpackingErr == nil {
			firstUnpackingErr = err
		}
		binary.Write(&buf, binary.LittleEndian, f)
	}
	// Always write in order: R, G, B, A
	switch srcFmt {
	case VkFormat_VK_FORMAT_R4G4_UNORM_PACK8:
		writeInt(4, 8, unsigned)
		writeInt(0, 4, unsigned)
	case VkFormat_VK_FORMAT_R4G4B4A4_UNORM_PACK16:
		writeInt(12, 16, unsigned)
		writeInt(8, 12, unsigned)
		writeInt(4, 8, unsigned)
		writeInt(0, 4, unsigned)
	case VkFormat_VK_FORMAT_B4G4R4A4_UNORM_PACK16:
		writeInt(4, 8, unsigned)
		writeInt(8, 12, unsigned)
		writeInt(12, 16, unsigned)
		writeInt(0, 4, unsigned)
	case VkFormat_VK_FORMAT_R5G6B5_UNORM_PACK16:
		writeInt(11, 16, unsigned)
		writeInt(5, 11, unsigned)
		writeInt(0, 5, unsigned)
	case VkFormat_VK_FORMAT_B5G6R5_UNORM_PACK16:
		writeInt(0, 5, unsigned)
		writeInt(5, 11, unsigned)
		writeInt(11, 16, unsigned)
	case VkFormat_VK_FORMAT_R5G5B5A1_UNORM_PACK16:
		writeInt(11, 16, unsigned)
		writeInt(6, 11, unsigned)
		writeInt(1, 6, unsigned)
		writeInt(0, 1, unsigned)
	case VkFormat_VK_FORMAT_B5G5R5A1_UNORM_PACK16:
		writeInt(1, 6, unsigned)
		writeInt(6, 11, unsigned)
		writeInt(11, 16, unsigned)
		writeInt(0, 1, unsigned)
	case VkFormat_VK_FORMAT_A1R5G5B5_UNORM_PACK16:
		writeInt(10, 15, unsigned)
		writeInt(5, 10, unsigned)
		writeInt(0, 5, unsigned)
		writeInt(15, 16, unsigned)
	case VkFormat_VK_FORMAT_R8_UNORM,
		VkFormat_VK_FORMAT_R8_UINT:
		writeInt(0, 8, unsigned)
	case VkFormat_VK_FORMAT_R8_SNORM,
		VkFormat_VK_FORMAT_R8_SINT:
		writeInt(0, 8, signed)

	case VkFormat_VK_FORMAT_R8G8_UNORM,
		VkFormat_VK_FORMAT_R8G8_UINT:
		writeInt(0, 8, unsigned)
		writeInt(8, 16, unsigned)
	case VkFormat_VK_FORMAT_R8G8_SNORM,
		VkFormat_VK_FORMAT_R8G8_SINT:
		writeInt(0, 8, signed)
		writeInt(8, 16, signed)

	case VkFormat_VK_FORMAT_R8G8B8_UNORM,
		VkFormat_VK_FORMAT_R8G8B8_UINT:
		writeInt(0, 8, unsigned)
		writeInt(8, 16, unsigned)
		writeInt(16, 24, unsigned)
	case VkFormat_VK_FORMAT_R8G8B8_SNORM,
		VkFormat_VK_FORMAT_R8G8B8_SINT:
		writeInt(0, 8, signed)
		writeInt(8, 16, signed)
		writeInt(16, 24, signed)

	case VkFormat_VK_FORMAT_B8G8R8_UNORM,
		VkFormat_VK_FORMAT_B8G8R8_UINT:
		writeInt(16, 24, unsigned)
		writeInt(8, 16, unsigned)
		writeInt(0, 8, unsigned)
	case VkFormat_VK_FORMAT_B8G8R8_SNORM,
		VkFormat_VK_FORMAT_B8G8R8_SINT:
		writeInt(16, 24, signed)
		writeInt(8, 16, signed)
		writeInt(0, 8, signed)

	case VkFormat_VK_FORMAT_R8G8B8A8_UNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_UINT,
		VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32,
		VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32:
		writeInt(0, 8, unsigned)
		writeInt(8, 16, unsigned)
		writeInt(16, 24, unsigned)
		writeInt(24, 32, unsigned)
	case VkFormat_VK_FORMAT_R8G8B8A8_SNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_SINT,
		VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32,
		VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32:
		writeInt(0, 8, signed)
		writeInt(8, 16, signed)
		writeInt(16, 24, signed)
		writeInt(24, 32, signed)

	case VkFormat_VK_FORMAT_B8G8R8A8_UNORM,
		VkFormat_VK_FORMAT_B8G8R8A8_UINT:
		writeInt(16, 24, unsigned)
		writeInt(8, 16, unsigned)
		writeInt(0, 8, unsigned)
		writeInt(24, 32, unsigned)
	case VkFormat_VK_FORMAT_B8G8R8A8_SNORM,
		VkFormat_VK_FORMAT_B8G8R8A8_SINT:
		writeInt(16, 24, signed)
		writeInt(8, 16, signed)
		writeInt(0, 8, signed)
		writeInt(24, 32, signed)

	case VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32,
		VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32:
		writeInt(20, 30, unsigned)
		writeInt(10, 20, unsigned)
		writeInt(0, 10, unsigned)
		writeInt(30, 32, unsigned)
	case VkFormat_VK_FORMAT_A2R10G10B10_SNORM_PACK32,
		VkFormat_VK_FORMAT_A2R10G10B10_SINT_PACK32:
		writeInt(20, 30, signed)
		writeInt(10, 20, signed)
		writeInt(0, 10, signed)
		writeInt(30, 32, signed)

	case VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32:
		writeInt(0, 10, unsigned)
		writeInt(10, 20, unsigned)
		writeInt(20, 30, unsigned)
		writeInt(30, 32, unsigned)
	case VkFormat_VK_FORMAT_A2B10G10R10_SNORM_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_SINT_PACK32:
		writeInt(0, 10, signed)
		writeInt(10, 20, signed)
		writeInt(20, 30, signed)
		writeInt(30, 32, signed)

	case VkFormat_VK_FORMAT_R16_UNORM,
		VkFormat_VK_FORMAT_R16_UINT:
		writeInt(0, 16, unsigned)
	case VkFormat_VK_FORMAT_R16_SNORM,
		VkFormat_VK_FORMAT_R16_SINT:
		writeInt(0, 16, signed)
	case VkFormat_VK_FORMAT_R16_SFLOAT:
		writeFloat(0, 10, 10, 15, signed, 15)

	case VkFormat_VK_FORMAT_R16G16_UNORM,
		VkFormat_VK_FORMAT_R16G16_UINT:
		writeInt(0, 16, unsigned)
		writeInt(16, 32, unsigned)
	case VkFormat_VK_FORMAT_R16G16_SNORM,
		VkFormat_VK_FORMAT_R16G16_SINT:
		writeInt(0, 16, signed)
		writeInt(16, 32, signed)
	case VkFormat_VK_FORMAT_R16G16_SFLOAT:
		writeFloat(0, 10, 10, 15, signed, 15)
		writeFloat(16, 26, 26, 31, signed, 31)

	case VkFormat_VK_FORMAT_R16G16B16_UNORM,
		VkFormat_VK_FORMAT_R16G16B16_UINT:
		writeInt(0, 16, unsigned)
		writeInt(16, 32, unsigned)
		writeInt(32, 48, unsigned)
	case VkFormat_VK_FORMAT_R16G16B16_SNORM,
		VkFormat_VK_FORMAT_R16G16B16_SINT:
		writeInt(0, 16, signed)
		writeInt(16, 32, signed)
		writeInt(32, 48, signed)
	case VkFormat_VK_FORMAT_R16G16B16_SFLOAT:
		writeFloat(0, 10, 10, 15, signed, 15)
		writeFloat(16, 26, 26, 31, signed, 31)
		writeFloat(32, 42, 42, 47, signed, 47)

	case VkFormat_VK_FORMAT_R16G16B16A16_UNORM,
		VkFormat_VK_FORMAT_R16G16B16A16_UINT:
		writeInt(0, 16, unsigned)
		writeInt(16, 32, unsigned)
		writeInt(32, 48, unsigned)
		writeInt(48, 64, unsigned)
	case VkFormat_VK_FORMAT_R16G16B16A16_SNORM,
		VkFormat_VK_FORMAT_R16G16B16A16_SINT:
		writeInt(0, 16, signed)
		writeInt(16, 32, signed)
		writeInt(32, 48, signed)
		writeInt(48, 64, signed)
	// TODO: rgba16_sfloat

	case VkFormat_VK_FORMAT_R32_UINT:
		writeInt(0, 32, unsigned)
	case VkFormat_VK_FORMAT_R32_SINT:
		writeInt(0, 32, signed)
	// TODO: r32 sfloat

	case VkFormat_VK_FORMAT_R32G32_UINT:
		writeInt(0, 32, unsigned)
		writeInt(32, 64, unsigned)
	case VkFormat_VK_FORMAT_R32G32_SINT:
		writeInt(0, 32, signed)
		writeInt(32, 64, signed)
	// TODO: rg32 sfloat

	case VkFormat_VK_FORMAT_R32G32B32_UINT:
		writeInt(0, 32, unsigned)
		writeInt(32, 64, unsigned)
		writeInt(64, 96, unsigned)
	case VkFormat_VK_FORMAT_R32G32B32_SINT:
		writeInt(0, 32, signed)
		writeInt(32, 64, signed)
		writeInt(64, 96, signed)
	// TODO: rgb32 sfloat

	case VkFormat_VK_FORMAT_R32G32B32A32_UINT:
		writeInt(0, 32, unsigned)
		writeInt(32, 64, unsigned)
		writeInt(64, 96, unsigned)
		writeInt(96, 128, unsigned)
	case VkFormat_VK_FORMAT_R32G32B32A32_SINT:
		writeInt(0, 32, signed)
		writeInt(32, 64, signed)
		writeInt(64, 96, signed)
		writeInt(96, 128, signed)
	// TODO: rgba32 sfloat

	case VkFormat_VK_FORMAT_R64_UINT:
		writeInt(0+dstImgIndex*32, 64+dstImgIndex*32, unsigned)
	case VkFormat_VK_FORMAT_R64_SINT:
		writeInt(0+dstImgIndex*32, 64+dstImgIndex*32, signed)
	// TODO: r64 sfloat

	case VkFormat_VK_FORMAT_R64G64_UINT:
		writeInt(0+dstImgIndex*32, 64+dstImgIndex*32, unsigned)
		writeInt(64+dstImgIndex*32, 128+dstImgIndex*32, unsigned)
	case VkFormat_VK_FORMAT_R64G64_SINT:
		writeInt(0+dstImgIndex*32, 64+dstImgIndex*32, signed)
		writeInt(64+dstImgIndex*32, 128+dstImgIndex*32, signed)
	// TODO: rg64 sfloat

	case VkFormat_VK_FORMAT_R64G64B64_UINT:
		writeInt(0+dstImgIndex*32, 64+dstImgIndex*32, unsigned)
		writeInt(64+dstImgIndex*32, 128+dstImgIndex*32, unsigned)
		writeInt(128+dstImgIndex*32, 192+dstImgIndex*32, unsigned)
	case VkFormat_VK_FORMAT_R64G64B64_SINT:
		writeInt(0+dstImgIndex*32, 64+dstImgIndex*32, signed)
		writeInt(64+dstImgIndex*32, 128+dstImgIndex*32, signed)
		writeInt(128+dstImgIndex*32, 192+dstImgIndex*32, signed)
	// TODO: rgb64 sfloat

	case VkFormat_VK_FORMAT_R64G64B64A64_UINT:
		writeInt(0+dstImgIndex*32, 64+dstImgIndex*32, unsigned)
		writeInt(64+dstImgIndex*32, 128+dstImgIndex*32, unsigned)
		writeInt(128+dstImgIndex*32, 192+dstImgIndex*32, unsigned)
		writeInt(192+dstImgIndex*32, 256+dstImgIndex*32, unsigned)
	case VkFormat_VK_FORMAT_R64G64B64A64_SINT:
		writeInt(0+dstImgIndex*32, 64+dstImgIndex*32, signed)
		writeInt(64+dstImgIndex*32, 128+dstImgIndex*32, signed)
		writeInt(128+dstImgIndex*32, 192+dstImgIndex*32, signed)
		writeInt(192+dstImgIndex*32, 256+dstImgIndex*32, signed)
	// TODO: rgba64 sfloat

	case VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32:
		writeFloat(0, 9, 27, 32, unsigned, 0)
		writeFloat(9, 18, 27, 32, unsigned, 0)
		writeFloat(18, 27, 27, 32, unsigned, 0)

	case VkFormat_VK_FORMAT_D16_UNORM:
		writeInt(0, 16, unsigned)

	default:
		return []uint8{}, fmt.Errorf("unsupported format: %v", srcFmt)
	}
	if firstUnpackingErr != nil {
		return []uint8{}, firstUnpackingErr
	}
	return buf.Bytes(), nil
}

type signedness int

const (
	unsigned signedness = iota
	signed
)

// Always assume little endianness.
func unpackIntToUint32(src []uint8, start, oneOverEnd uint32, sign signedness) (uint32, error) {
	ret, err := extractBitsToUin32(src, start, oneOverEnd)
	if err != nil {
		return 0, err
	}
	if sign == unsigned {
		return ret, nil
	}
	ret = ret << (32 - (oneOverEnd - start))
	sret := int32(ret) >> (32 - (oneOverEnd - start))
	return uint32(sret), nil
}

// TODO: This is not correct!
func unpackFloatToFloat32(src []uint8, fracStart, oneOverFracEnd, expStart, oneOverExpEnd uint32, sign signedness, signBit uint32) (float32, error) {
	frac, err := extractBitsToUin32(src, fracStart, oneOverFracEnd)
	if err != nil {
		return 0, err
	}
	exp, err := extractBitsToUin32(src, expStart, oneOverExpEnd)
	if err != nil {
		return 0, err
	}
	var u uint32
	u += frac << (23 - (oneOverFracEnd - fracStart))
	u += exp << (31 - (oneOverExpEnd - expStart))
	if sign == signed {
		s, err := extractBitsToUin32(src, signBit, signBit+1)
		if err != nil {
			return 0, err
		}
		if s == uint32(1) {
			u += 0x1 << 31
		}
	}
	var buf bytes.Buffer
	err = binary.Write(&buf, binary.LittleEndian, u)
	if err != nil {
		return 0, err
	}
	var f float32
	err = binary.Read(&buf, binary.LittleEndian, &f)
	if err != nil {
		return 0, err
	}
	return f, nil
}

func extractBitsToUin32(src []uint8, start, oneOverEnd uint32) (uint32, error) {
	if oneOverEnd <= start {
		return 0, fmt.Errorf("'oneOverEnd' bit must be greater than 'start' bit")
	}
	if oneOverEnd-start > 32 {
		return 0, fmt.Errorf("cannot extract more than 32 bits")
	}
	if uint32(len(src)) <= (oneOverEnd-1)/8 {
		return 0, fmt.Errorf("not enough bytes given")
	}
	d := make([]uint8, 4)
	c := 0
	for i := start / 8; i <= (oneOverEnd-1)/8; i++ {
		d[c] = src[i]
		c++
	}
	raw := uint32(0)
	buf := bytes.NewBuffer(d)
	binary.Read(buf, binary.LittleEndian, &raw)
	shift := start % 8
	mask := uint32(0x1<<(oneOverEnd-start/8*8)) - 1
	return uint32((raw & mask) >> shift), nil
}

// stagingImageFormat returns the format of the staging image for the given
// image object.
func stagingImageFormat(img *ImageObject) VkFormat {
	switch img.ImageAspect {
	case VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT):
		// color
		return colorStagingFormat
	case VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT):
		// depth
		return depthStagingFormat
	case VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT) | VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT):
		// depth+stencil
		break
	default:
		break
	}
	return VkFormat_VK_FORMAT_UNDEFINED
}

func texelBlockSizeInBytes(ctx context.Context, s *api.GlobalState, fmt VkFormat) (uint32, error) {
	elementAndTexelSizeInfo, err := subGetElementAndTexelBlockSize(ctx, nil, api.CmdNoID, nil, s, GetState(s), 0, nil, fmt)
	if err != nil {
		return 0, err
	}
	w := elementAndTexelSizeInfo.TexelBlockSize.Width
	h := elementAndTexelSizeInfo.TexelBlockSize.Height
	return elementAndTexelSizeInfo.ElementSize * w * h, nil
}

func texelBlockWidth(ctx context.Context, s *api.GlobalState, fmt VkFormat) (uint32, error) {
	elementAndTexelSizeInfo, err := subGetElementAndTexelBlockSize(ctx, nil, api.CmdNoID, nil, s, GetState(s), 0, nil, fmt)
	if err != nil {
		return 0, err
	}
	return elementAndTexelSizeInfo.TexelBlockSize.Width, nil
}

func texelBlockHeight(ctx context.Context, s *api.GlobalState, fmt VkFormat) (uint32, error) {
	elementAndTexelSizeInfo, err := subGetElementAndTexelBlockSize(ctx, nil, api.CmdNoID, nil, s, GetState(s), 0, nil, fmt)
	if err != nil {
		return 0, err
	}
	return elementAndTexelSizeInfo.TexelBlockSize.Height, nil
}

func (h *imageRebuildHelper) renderStagingImages(inputImgs []*ImageObject, outputImg *ImageObject, queue *QueueObject, layer, level uint32) error {
	descPool := h.createTempDescriptorPool(outputImg.Device, 1, []VkDescriptorPoolSize{
		VkDescriptorPoolSize{
			VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT,
			uint32(len(inputImgs)),
		},
	})
	descSetLayout := h.createTempDescriptorSetLayout(outputImg.Device, []VkDescriptorSetLayoutBinding{
		VkDescriptorSetLayoutBinding{
			0,
			VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT,
			uint32(len(inputImgs)),
			VkShaderStageFlags(VkShaderStageFlagBits_VK_SHADER_STAGE_FRAGMENT_BIT),
			NewVkSamplerᶜᵖ(memory.Nullptr),
		},
	})
	descSet := h.allocateTempDescriptorSet(descPool, descSetLayout)
	inputViews := []*ImageViewObject{}
	for _, input := range inputImgs {
		inputViews = append(inputViews, h.createTempImageView(input, layer, level))
	}
	outputView := h.createTempImageView(outputImg, layer, level)
	imgInfo := []VkDescriptorImageInfo{}
	for _, view := range inputViews {
		imgInfo = append(imgInfo, VkDescriptorImageInfo{
			VkSampler(0),
			view.VulkanHandle,
			VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
		})
	}
	h.writeDescriptorSet(descSet, 0, 0, VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT, imgInfo, []VkDescriptorBufferInfo{}, []VkBufferView{})
	renderpass, err := h.createTempRenderPassForPriming(inputImgs, outputImg)
	if err != nil {
		return fmt.Errorf("[Creating RenderPass] %v", err)
	}
	framebuffer, err := h.createTempFrameBufferForPriming(renderpass, append(inputViews, outputView), level)
	if err != nil {
		return fmt.Errorf("[Creating Framebuffer] %v", err)
	}
	pipelineLayout := h.createTempPipelineLayoutForPriming(descSetLayout)
	vertShader := h.createTempVertShaderModuleForPriming(outputImg.Device)
	fragShader := h.createTempFragShaderModuleForPriming(inputImgs, outputImg)
	e := h.sb.levelSize(outputImg.Info.Extent, outputImg.Info.Format, level)
	viewport := VkViewport{
		0.0, 0.0,
		float32(e.width), float32(e.height),
		0.0, 1.0,
	}
	scissor := VkRect2D{
		VkOffset2D{int32(0), int32(0)},
		VkExtent2D{uint32(e.width), uint32(e.height)},
	}
	gfxPipeline := h.createTempGfxPipelineForPriming(vertShader, fragShader, pipelineLayout, renderpass, viewport, scissor)

	var vertexBufContent bytes.Buffer
	binary.Write(&vertexBufContent, binary.LittleEndian, []float32{
		// positions, offset: 0 bytes
		1.0, 1.0, 0.0, -1.0, -1.0, 0.0, -1.0, 1.0, 0.0, 1.0, -1.0, 0.0,
		// uv, offset: 48 bytes
		1.0, 1.0, 0.0, 0.0, 0.0, 1.0, 1.0, 0.0,
		// normals, offset : 80 bytes
		0.0, 0.0, 1.0, 0.0, 0.0, 1.0, 0.0, 0.0, 1.0, 0.0, 0.0, 1.0,
	})
	vertexBuf, vertexBufMem := h.sb.allocAndFillScratchBuffer(h.sb.s.Devices.Get(outputImg.Device), vertexBufContent.Bytes())
	h.tempBuffers[vertexBuf] = GetState(h.sb.newState).Buffers.Get(vertexBuf)
	h.tempMemories[vertexBufMem] = GetState(h.sb.newState).DeviceMemories.Get(vertexBufMem)

	var indexBufContent bytes.Buffer
	binary.Write(&indexBufContent, binary.LittleEndian, []uint32{
		0, 1, 2, 0, 3, 1,
	})
	indexBuf, indexBufMem := h.sb.allocAndFillScratchBuffer(h.sb.s.Devices.Get(outputImg.Device), indexBufContent.Bytes())
	h.tempBuffers[indexBuf] = GetState(h.sb.newState).Buffers.Get(indexBuf)
	h.tempMemories[indexBufMem] = GetState(h.sb.newState).DeviceMemories.Get(indexBufMem)

	commandBuffer, commandPool := h.sb.getCommandBuffer(queue)

	imgBarriers := []VkImageMemoryBarrier{}
	for _, img := range inputImgs {
		imgBarriers = append(imgBarriers,
			VkImageMemoryBarrier{
				VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
				NewVoidᶜᵖ(memory.Nullptr),
				VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT - 1) | VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT),
				VkAccessFlags(VkAccessFlagBits_VK_ACCESS_INPUT_ATTACHMENT_READ_BIT),
				VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
				queue.Family,
				queue.Family,
				img.VulkanHandle,
				VkImageSubresourceRange{
					img.ImageAspect,
					0,
					img.Info.MipLevels,
					0,
					img.Info.ArrayLayers,
				},
			})
	}
	outputBarrier := VkImageMemoryBarrier{
		VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
		NewVoidᶜᵖ(memory.Nullptr),
		VkAccessFlags(0),
		VkAccessFlags(VkAccessFlagBits_VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT),
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
		queue.Family,
		queue.Family,
		outputImg.VulkanHandle,
		VkImageSubresourceRange{
			outputImg.ImageAspect,
			0,
			outputImg.Info.MipLevels,
			0,
			outputImg.Info.ArrayLayers,
		}}

	switch outputImg.ImageAspect {
	case VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT):
		outputBarrier.NewLayout = VkImageLayout_VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
	case VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT):
		outputBarrier.NewLayout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
	case VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT) | VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT):
		outputBarrier.NewLayout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
	default:
		return fmt.Errorf("unsupported image aspect flags: %v", outputImg.ImageAspect)
	}

	imgBarriers = append(imgBarriers, outputBarrier)

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
					VkDeviceSize(len(vertexBufContent.Bytes())),
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
					VkDeviceSize(len(indexBufContent.Bytes())),
				},
			}).Ptr(),
		uint32(len(imgBarriers)),
		h.sb.MustAllocReadData(imgBarriers).Ptr(),
	))

	h.sb.write(h.sb.cb.VkCmdBeginRenderPass(
		commandBuffer,
		h.sb.MustAllocReadData(
			VkRenderPassBeginInfo{
				VkStructureType_VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				renderpass.VulkanHandle,
				framebuffer.VulkanHandle,
				VkRect2D{
					VkOffset2D{int32(0), int32(0)},
					VkExtent2D{outputImg.Info.Extent.Width, outputImg.Info.Extent.Height},
				},
				uint32(0),
				NewVkClearValueᶜᵖ(memory.Nullptr),
			}).Ptr(),
		VkSubpassContents(0),
	))

	h.sb.write(h.sb.cb.VkCmdBindPipeline(
		commandBuffer,
		VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS,
		gfxPipeline.VulkanHandle,
	))

	h.sb.write(h.sb.cb.VkCmdBindDescriptorSets(
		commandBuffer,
		VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS,
		pipelineLayout.VulkanHandle,
		0,
		1,
		h.sb.MustAllocReadData([]VkDescriptorSet{descSet.VulkanHandle}).Ptr(),
		0,
		NewU32ᶜᵖ(memory.Nullptr),
	))

	h.sb.write(h.sb.cb.VkCmdBindVertexBuffers(
		commandBuffer,
		0, 3,
		h.sb.MustAllocReadData(
			[]VkBuffer{
				vertexBuf,
				vertexBuf,
				vertexBuf,
			}).Ptr(),
		h.sb.MustAllocReadData(
			[]VkDeviceSize{
				VkDeviceSize(0),
				VkDeviceSize(48),
				VkDeviceSize(80),
			}).Ptr(),
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

func (h *imageRebuildHelper) createTempDescriptorPool(dev VkDevice, maxSet uint32, poolSizes []VkDescriptorPoolSize) *DescriptorPoolObject {
	handle := VkDescriptorPool(newUnusedID(true, func(x uint64) bool {
		inState := h.sb.s.DescriptorPools.Contains(VkDescriptorPool(x))
		_, inHelper := h.tempDescriptorPools[VkDescriptorPool(x)]
		return inState || inHelper
	}))
	h.sb.write(h.sb.cb.VkCreateDescriptorPool(
		dev,
		h.sb.MustAllocReadData(VkDescriptorPoolCreateInfo{
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
			NewVoidᶜᵖ(memory.Nullptr),
			VkDescriptorPoolCreateFlags(VkDescriptorPoolCreateFlagBits_VK_DESCRIPTOR_POOL_CREATE_FREE_DESCRIPTOR_SET_BIT),
			maxSet,
			uint32(len(poolSizes)),
			NewVkDescriptorPoolSizeᶜᵖ(h.sb.MustAllocReadData(poolSizes).Ptr()),
		}).Ptr(),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	h.tempDescriptorPools[handle] = GetState(h.sb.newState).DescriptorPools.Get(handle)
	return h.tempDescriptorPools[handle]
}

func (h *imageRebuildHelper) createTempDescriptorSetLayout(dev VkDevice, bindings []VkDescriptorSetLayoutBinding) *DescriptorSetLayoutObject {
	handle := VkDescriptorSetLayout(newUnusedID(true, func(x uint64) bool {
		inState := h.sb.s.DescriptorSetLayouts.Contains(VkDescriptorSetLayout(x))
		_, inHelper := h.tempDescriptorSetLayouts[VkDescriptorSetLayout(x)]
		return inState || inHelper
	}))
	h.sb.write(h.sb.cb.VkCreateDescriptorSetLayout(
		dev,
		h.sb.MustAllocReadData(VkDescriptorSetLayoutCreateInfo{
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO,
			NewVoidᶜᵖ(memory.Nullptr),
			VkDescriptorSetLayoutCreateFlags(0),
			uint32(len(bindings)),
			NewVkDescriptorSetLayoutBindingᶜᵖ(h.sb.MustAllocReadData(bindings).Ptr()),
		}).Ptr(),
		NewVoidᶜᵖ(memory.Nullptr),
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	h.tempDescriptorSetLayouts[handle] = GetState(h.sb.newState).DescriptorSetLayouts.Get(handle)
	return h.tempDescriptorSetLayouts[handle]
}

func (h *imageRebuildHelper) allocateTempDescriptorSet(pool *DescriptorPoolObject, layout *DescriptorSetLayoutObject) *DescriptorSetObject {
	handle := VkDescriptorSet(newUnusedID(true, func(x uint64) bool {
		inState := h.sb.s.DescriptorSets.Contains(VkDescriptorSet(x))
		_, inHelper := h.tempDescriptorSets[VkDescriptorSet(x)]
		return inState || inHelper
	}))
	h.sb.write(h.sb.cb.VkAllocateDescriptorSets(
		pool.Device,
		h.sb.MustAllocReadData(VkDescriptorSetAllocateInfo{
			VkStructureType_VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
			NewVoidᶜᵖ(memory.Nullptr),
			pool.VulkanHandle,
			1,
			NewVkDescriptorSetLayoutᶜᵖ(h.sb.MustAllocReadData(layout.VulkanHandle).Ptr()),
		}).Ptr(),
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	h.tempDescriptorSets[handle] = GetState(h.sb.newState).DescriptorSets.Get(handle)
	return h.tempDescriptorSets[handle]
}

func (h *imageRebuildHelper) createTempRenderPassForPriming(stagingImgs []*ImageObject, dstImg *ImageObject) (*RenderPassObject, error) {
	inputAttachmentRefs := []VkAttachmentReference{}
	inputAttachmentDescs := []VkAttachmentDescription{}
	for i, img := range stagingImgs {
		inputAttachmentRefs = append(inputAttachmentRefs,
			VkAttachmentReference{
				uint32(i), VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
			})
		inputAttachmentDescs = append(inputAttachmentDescs,
			VkAttachmentDescription{
				VkAttachmentDescriptionFlags(0),
				img.Info.Format,
				img.Info.Samples,
				VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD,
				VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE,
				VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE,
				VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE,
				VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
				VkImageLayout_VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
			})
	}
	outputAttachmentRef := VkAttachmentReference{
		uint32(len(stagingImgs)), VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
	}
	outputAttachmentDesc := VkAttachmentDescription{
		VkAttachmentDescriptionFlags(0),
		dstImg.Info.Format,
		dstImg.Info.Samples,
		VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE,
		VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE,
		VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE,
		VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE,
		VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
		dstImg.Info.Layout,
	}
	subpassDesc := VkSubpassDescription{
		VkSubpassDescriptionFlags(0),
		VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS,
		uint32(len(stagingImgs)),
		NewVkAttachmentReferenceᶜᵖ(h.sb.MustAllocReadData(inputAttachmentRefs).Ptr()),
		uint32(0),
		NewVkAttachmentReferenceᶜᵖ(memory.Nullptr),
		NewVkAttachmentReferenceᶜᵖ(memory.Nullptr),
		NewVkAttachmentReferenceᶜᵖ(memory.Nullptr),
		uint32(0),
		NewU32ᶜᵖ(memory.Nullptr),
	}
	switch dstImg.ImageAspect {
	case VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT):
		outputAttachmentRef.Layout = VkImageLayout_VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
		outputAttachmentDesc.InitialLayout = VkImageLayout_VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
		subpassDesc.ColorAttachmentCount = uint32(1)
		subpassDesc.PColorAttachments = NewVkAttachmentReferenceᶜᵖ(h.sb.MustAllocReadData(outputAttachmentRef).Ptr())
	case VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT):
		outputAttachmentRef.Layout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
		outputAttachmentDesc.InitialLayout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
		subpassDesc.PDepthStencilAttachment = NewVkAttachmentReferenceᶜᵖ(h.sb.MustAllocReadData(outputAttachmentRef).Ptr())
	case VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT) | VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT):
		outputAttachmentRef.Layout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
		outputAttachmentDesc.InitialLayout = VkImageLayout_VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
		subpassDesc.PDepthStencilAttachment = NewVkAttachmentReferenceᶜᵖ(h.sb.MustAllocReadData(outputAttachmentRef).Ptr())
	default:
		return nil, fmt.Errorf("unsupported image aspect flags: %v", dstImg.ImageAspect)
	}

	createInfo := VkRenderPassCreateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkRenderPassCreateFlags(0),
		uint32(len(inputAttachmentDescs) + 1),
		NewVkAttachmentDescriptionᶜᵖ(h.sb.MustAllocReadData(
			append(inputAttachmentDescs, outputAttachmentDesc),
		).Ptr()),
		uint32(1),
		NewVkSubpassDescriptionᶜᵖ(h.sb.MustAllocReadData(subpassDesc).Ptr()),
		uint32(0),
		NewVkSubpassDependencyᶜᵖ(memory.Nullptr),
	}

	handle := VkRenderPass(newUnusedID(true, func(x uint64) bool {
		inState := h.sb.s.RenderPasses.Contains(VkRenderPass(x))
		_, inHelper := h.tempRenderPasses[VkRenderPass(x)]
		return inState || inHelper
	}))

	h.sb.write(h.sb.cb.VkCreateRenderPass(
		dstImg.Device,
		NewVkRenderPassCreateInfoᶜᵖ(h.sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))

	h.tempRenderPasses[handle] = GetState(h.sb.newState).RenderPasses.Get(handle)
	return h.tempRenderPasses[handle], nil
}

func (h *imageRebuildHelper) createTempFrameBufferForPriming(renderpass *RenderPassObject, imgViews []*ImageViewObject, level uint32) (*FramebufferObject, error) {
	if len(imgViews) < 2 {
		return nil, fmt.Errorf("requires at least two image views, %d are given", len(imgViews))
	}
	e := h.sb.levelSize(imgViews[0].Image.Info.Extent, imgViews[0].Image.Info.Format, level)
	attachments := []VkImageView{}
	for _, v := range imgViews {
		attachments = append(attachments, v.VulkanHandle)
	}
	createInfo := VkFramebufferCreateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkFramebufferCreateFlags(0),
		renderpass.VulkanHandle,
		uint32(len(imgViews)),
		NewVkImageViewᶜᵖ(h.sb.MustAllocReadData(attachments).Ptr()),
		uint32(e.width),
		uint32(e.height),
		1,
	}

	handle := VkFramebuffer(newUnusedID(true, func(x uint64) bool {
		inState := h.sb.s.Framebuffers.Contains(VkFramebuffer(x))
		_, inHelper := h.tempFramebuffers[VkFramebuffer(x)]
		return inState || inHelper
	}))

	h.sb.write(h.sb.cb.VkCreateFramebuffer(
		imgViews[0].Image.Device,
		NewVkFramebufferCreateInfoᶜᵖ(h.sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))

	h.tempFramebuffers[handle] = GetState(h.sb.newState).Framebuffers.Get(handle)
	return h.tempFramebuffers[handle], nil
}

func (h *imageRebuildHelper) createTempImageView(img *ImageObject, layer, level uint32) *ImageViewObject {
	handle := VkImageView(newUnusedID(true, func(x uint64) bool {
		inState := h.sb.s.ImageViews.Contains(VkImageView(x))
		_, inHelper := h.tempImageViews[VkImageView(x)]
		return inState || inHelper
	}))
	h.sb.write(h.sb.cb.VkCreateImageView(
		img.Device,
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
					img.ImageAspect,
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
	h.tempImageViews[handle] = GetState(h.sb.newState).ImageViews.Get(handle)
	return h.tempImageViews[handle]
}

func (h *imageRebuildHelper) writeDescriptorSet(descriptorSet *DescriptorSetObject, dstBinding, dstArrayElement uint32, descType VkDescriptorType, imgInfo []VkDescriptorImageInfo, bufInfo []VkDescriptorBufferInfo, texelBufInfo []VkBufferView) {
	write := VkWriteDescriptorSet{
		VkStructureType_VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
		NewVoidᶜᵖ(memory.Nullptr),
		descriptorSet.VulkanHandle,
		dstBinding,
		dstArrayElement,
		uint32(len(imgInfo) + len(bufInfo) + len(texelBufInfo)),
		descType,
		NewVkDescriptorImageInfoᶜᵖ(h.sb.MustAllocReadData(imgInfo).Ptr()),
		NewVkDescriptorBufferInfoᶜᵖ(h.sb.MustAllocReadData(bufInfo).Ptr()),
		NewVkBufferViewᶜᵖ(h.sb.MustAllocReadData(texelBufInfo).Ptr()),
	}

	h.sb.write(h.sb.cb.VkUpdateDescriptorSets(
		descriptorSet.Device,
		uint32(1),
		NewVkWriteDescriptorSetᶜᵖ(h.sb.MustAllocReadData(write).Ptr()),
		uint32(0),
		memory.Nullptr,
	))
}

func (h *imageRebuildHelper) createTempPipelineLayoutForPriming(descSetLayout *DescriptorSetLayoutObject) *PipelineLayoutObject {
	handle := VkPipelineLayout(newUnusedID(true, func(x uint64) bool {
		inState := h.sb.s.PipelineLayouts.Contains(VkPipelineLayout(x))
		_, inHelper := h.tempPipelineLayouts[VkPipelineLayout(x)]
		return inState || inHelper
	}))
	createInfo := VkPipelineLayoutCreateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkPipelineLayoutCreateFlags(0),
		uint32(1),
		NewVkDescriptorSetLayoutᶜᵖ(h.sb.MustAllocReadData(descSetLayout.VulkanHandle).Ptr()),
		uint32(0),
		NewVkPushConstantRangeᶜᵖ(memory.Nullptr),
	}
	h.sb.write(h.sb.cb.VkCreatePipelineLayout(
		descSetLayout.Device,
		NewVkPipelineLayoutCreateInfoᶜᵖ(h.sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	h.tempPipelineLayouts[handle] = GetState(h.sb.newState).PipelineLayouts.Get(handle)
	return h.tempPipelineLayouts[handle]
}

func (h *imageRebuildHelper) createTempVertShaderModuleForPriming(dev VkDevice) *ShaderModuleObject {
	handle := VkShaderModule(newUnusedID(true, func(x uint64) bool {
		inState := h.sb.s.ShaderModules.Contains(VkShaderModule(x))
		_, inHelper := h.tempShaders[VkShaderModule(x)]
		return inState || inHelper
	}))

	spv := primingVertSpv()

	createInfo := VkShaderModuleCreateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkShaderModuleCreateFlags(0),
		memory.Size(len(spv) * 4),
		NewU32ᶜᵖ(h.sb.MustAllocReadData(spv).Ptr()),
	}
	h.sb.write(h.sb.cb.VkCreateShaderModule(
		dev,
		NewVkShaderModuleCreateInfoᶜᵖ(h.sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	h.tempShaders[handle] = GetState(h.sb.newState).ShaderModules.Get(handle)
	return h.tempShaders[handle]
}

func (h *imageRebuildHelper) createTempFragShaderModuleForPriming(stagingImgs []*ImageObject, dstImg *ImageObject) *ShaderModuleObject {
	handle := VkShaderModule(newUnusedID(true, func(x uint64) bool {
		inState := h.sb.s.ShaderModules.Contains(VkShaderModule(x))
		_, inHelper := h.tempShaders[VkShaderModule(x)]
		return inState || inHelper
	}))

	spv := primingFragSpv(dstImg.Info.Format)

	createInfo := VkShaderModuleCreateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkShaderModuleCreateFlags(0),
		memory.Size(len(spv) * 4),
		NewU32ᶜᵖ(h.sb.MustAllocReadData(spv).Ptr()),
	}
	h.sb.write(h.sb.cb.VkCreateShaderModule(
		dstImg.Device,
		NewVkShaderModuleCreateInfoᶜᵖ(h.sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
	h.tempShaders[handle] = GetState(h.sb.newState).ShaderModules.Get(handle)
	return h.tempShaders[handle]
}

func (h *imageRebuildHelper) createTempGfxPipelineForPriming(vertShader, fragShader *ShaderModuleObject, pipelineLayout *PipelineLayoutObject, renderpass *RenderPassObject, viewport VkViewport, scissor VkRect2D) *GraphicsPipelineObject {
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
				uint32(3),
				NewVkVertexInputBindingDescriptionᶜᵖ(h.sb.MustAllocReadData(
					[]VkVertexInputBindingDescription{
						VkVertexInputBindingDescription{0, 12, 0},
						VkVertexInputBindingDescription{1, 8, 0},
						VkVertexInputBindingDescription{2, 12, 0},
					}).Ptr()),
				uint32(3),
				NewVkVertexInputAttributeDescriptionᶜᵖ(h.sb.MustAllocReadData(
					[]VkVertexInputAttributeDescription{
						VkVertexInputAttributeDescription{0, 0, VkFormat_VK_FORMAT_R32G32B32_SFLOAT, 0},
						VkVertexInputAttributeDescription{1, 1, VkFormat_VK_FORMAT_R32G32_SFLOAT, 0},
						VkVertexInputAttributeDescription{2, 2, VkFormat_VK_FORMAT_R32G32B32_SFLOAT, 0},
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
				NewVkViewportᶜᵖ(h.sb.MustAllocReadData(viewport).Ptr()),
				uint32(1),
				NewVkRect2Dᶜᵖ(h.sb.MustAllocReadData(scissor).Ptr()),
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
		NewVkPipelineDepthStencilStateCreateInfoᶜᵖ(h.sb.MustAllocReadData(
			VkPipelineDepthStencilStateCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				VkPipelineDepthStencilStateCreateFlags(0),
				VkBool32(1),
				VkBool32(1),
				VkCompareOp_VK_COMPARE_OP_ALWAYS,
				VkBool32(0),
				VkBool32(0),
				VkStencilOpState{},
				VkStencilOpState{},
				0.0, 0.0,
			}).Ptr()),
		NewVkPipelineColorBlendStateCreateInfoᶜᵖ(h.sb.MustAllocReadData(
			VkPipelineColorBlendStateCreateInfo{
				VkStructureType_VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO,
				NewVoidᶜᵖ(memory.Nullptr),
				VkPipelineColorBlendStateCreateFlags(0),
				VkBool32(0),
				VkLogicOp_VK_LOGIC_OP_CLEAR,
				uint32(len(*(*renderpass.SubpassDescriptions.Map)[0].ColorAttachments.Map)),
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
		NewVkPipelineDynamicStateCreateInfoᶜᵖ(memory.Nullptr),
		pipelineLayout.VulkanHandle,
		renderpass.VulkanHandle,
		uint32(0),
		VkPipeline(0),
		int32(0),
	}

	handle := VkPipeline(newUnusedID(true, func(x uint64) bool {
		inState := h.sb.s.GraphicsPipelines.Contains(VkPipeline(x))
		_, inHelper := h.tempGfxPipelines[VkPipeline(x)]
		return inState || inHelper
	}))

	h.sb.write(h.sb.cb.VkCreateGraphicsPipelines(
		renderpass.Device,
		VkPipelineCache(0),
		uint32(1),
		NewVkGraphicsPipelineCreateInfoᶜᵖ(h.sb.MustAllocReadData(createInfo).Ptr()),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))

	h.tempGfxPipelines[handle] = GetState(h.sb.newState).GraphicsPipelines.Get(handle)
	return h.tempGfxPipelines[handle]
}

func (h *imageRebuildHelper) freeTempObjects() {
	for handle, obj := range h.tempGfxPipelines {
		h.sb.write(h.sb.cb.VkDestroyPipeline(obj.Device, handle, memory.Nullptr))
	}
	for handle, obj := range h.tempPipelineLayouts {
		h.sb.write(h.sb.cb.VkDestroyPipelineLayout(obj.Device, handle, memory.Nullptr))
	}
	for handle, obj := range h.tempRenderPasses {
		h.sb.write(h.sb.cb.VkDestroyRenderPass(obj.Device, handle, memory.Nullptr))
	}
	for handle, obj := range h.tempFramebuffers {
		h.sb.write(h.sb.cb.VkDestroyFramebuffer(obj.Device, handle, memory.Nullptr))
	}
	for handle, obj := range h.tempShaders {
		h.sb.write(h.sb.cb.VkDestroyShaderModule(obj.Device, handle, memory.Nullptr))
	}
	for handle, obj := range h.tempDescriptorSets {
		h.sb.write(h.sb.cb.VkFreeDescriptorSets(obj.Device, obj.DescriptorPool, uint32(1), NewVkDescriptorSetᶜᵖ(h.sb.MustAllocReadData(handle).Ptr()), VkResult_VK_SUCCESS))
	}
	for handle, obj := range h.tempDescriptorSetLayouts {
		h.sb.write(h.sb.cb.VkDestroyDescriptorSetLayout(obj.Device, handle, memory.Nullptr))
	}
	for handle, obj := range h.tempDescriptorPools {
		h.sb.write(h.sb.cb.VkDestroyDescriptorPool(obj.Device, handle, memory.Nullptr))
	}
	for handle, obj := range h.tempBuffers {
		h.sb.write(h.sb.cb.VkDestroyBuffer(obj.Device, handle, memory.Nullptr))
	}
	for handle, obj := range h.tempImageViews {
		h.sb.write(h.sb.cb.VkDestroyImageView(obj.Device, handle, memory.Nullptr))
	}
	for handle, obj := range h.stagingImages {
		h.sb.write(h.sb.cb.VkDestroyImage(obj.Device, handle, memory.Nullptr))
	}
	for handle, obj := range h.tempMemories {
		h.sb.write(h.sb.cb.VkFreeMemory(obj.Device, handle, memory.Nullptr))
	}
}

func (h *imageRebuildHelper) vkCreateImage(dev VkDevice, info *ImageInfo, handle VkImage) {
	pNext := NewVoidᶜᵖ(memory.Nullptr)
	if info.DedicatedAllocationNV != nil {
		pNext = NewVoidᶜᵖ(h.sb.MustAllocReadData(
			VkDedicatedAllocationImageCreateInfoNV{
				VkStructureType_VK_STRUCTURE_TYPE_DEDICATED_ALLOCATION_IMAGE_CREATE_INFO_NV,
				NewVoidᶜᵖ(memory.Nullptr),
				info.DedicatedAllocationNV.DedicatedAllocation,
			},
		).Ptr())
	}
	h.sb.write(h.sb.cb.VkCreateImage(
		dev, h.sb.MustAllocReadData(
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
				NewU32ᶜᵖ(h.sb.MustUnpackReadMap(*info.QueueFamilyIndices.Map).Ptr()),
				VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,
			}).Ptr(),
		memory.Nullptr,
		h.sb.MustAllocWriteData(handle).Ptr(),
		VkResult_VK_SUCCESS,
	))
}

func (h *imageRebuildHelper) vkGetImageMemoryRequirements(dev VkDevice, handle VkImage, memReq *VkMemoryRequirements) {
	h.sb.write(h.sb.cb.VkGetImageMemoryRequirements(
		dev, handle, h.sb.MustAllocWriteData(*memReq).Ptr(),
	))
}

func (h *imageRebuildHelper) vkBindImageMemory(dev VkDevice, handle VkImage, mem VkDeviceMemory, offset VkDeviceSize) {
	h.sb.write(h.sb.cb.VkBindImageMemory(
		dev, handle, mem, offset, VkResult_VK_SUCCESS,
	))
}

func roundUpTo(dividend, divisor uint32) uint32 {
	return (dividend + divisor - 1) / divisor
}
