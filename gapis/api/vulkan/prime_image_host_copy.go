// Copyright (C) 2019 Google Inc.
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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/memory"
)

const (
	ipHostCopyImageLayout = VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
)

// ipHostCopyRecipeSubAspectPiece describes a sub aspect region in the image
// whose data is to be primed.
type ipHostCopyRecipeSubAspectPiece struct {
	layer        uint32
	level        uint32
	offsetX      uint32
	offsetY      uint32
	offsetZ      uint32
	extentWidth  uint32
	extentHeight uint32
	extentDepth  uint32
}

// ipHostCopyRecipe describes how the shadow data in a specific aspect from the
// source image to be primed to a specifc aspect of a destination image.
type ipHostCopyRecipe struct {
	srcImageInOldState VkImage
	srcAspect          VkImageAspectFlagBits
	dstImageInNewState VkImage
	dstAspect          VkImageAspectFlagBits
	wordIndex          uint32
	subAspectPieces    []ipHostCopyRecipeSubAspectPiece
}

// ipHostCopyKitBuilder builds the kit used to generate commands to prime image
// data stored in the host side by bufer to image copy.
type ipHostCopyKitBuilder struct {
	dev           VkDevice
	scratchMemory *flushingMemory
}

func newImagePrimerHostCopyKitBuilder(sb *stateBuilder, dev VkDevice) *ipHostCopyKitBuilder {
	return &ipHostCopyKitBuilder{
		dev: dev,
		// use the scratch memory owned by the state builder. Another option is
		// to create a new flushing memory.
		scratchMemory: sb.scratchRes.GetMemory(sb, dev),
	}
}

// Free flushes the scratch memory used by all the kits built by this kit
// builder.
func (kb *ipHostCopyKitBuilder) Free(sb *stateBuilder) {
	// Do not free the scratch memory owned by the state builder, just flush it.
	kb.scratchMemory.Flush(sb)
}

// BuildHostCopyKits takes a list of host copy recipes, and returns a list of
// host copy kits for priming image data stored in the host side by buffer to
// image copy.
func (kb *ipHostCopyKitBuilder) BuildHostCopyKits(sb *stateBuilder, recipes ...ipHostCopyRecipe) ([]ipHostCopyKit, error) {
	kits := make([]ipHostCopyKit, len(recipes))
	for i := range kits {
		recipe := recipes[i]
		srcImgObj := GetState(sb.oldState).Images().Get(recipe.srcImageInOldState)
		dstImgObj := GetState(sb.newState).Images().Get(recipe.dstImageInNewState)
		kitPieces := make([]ipHostCopyKitPiece, 0, len(recipe.subAspectPieces))
		for _, subAspect := range recipe.subAspectPieces {
			piece, err := kb.buildHostCopyKitPiece(sb,
				dstImgObj, recipe.dstAspect, srcImgObj, recipe.srcAspect, subAspect)
			if err != nil {
				return kits, log.Errf(sb.ctx, err, "failed at building copy kit piece")
			}
			kitPieces = append(kitPieces, piece)
		}
		kits[i].pieces = kitPieces
		kits[i].dstImage = recipe.dstImageInNewState
		kits[i].name = debugMarkerName(
			fmt.Sprintf("Copy host data to img: %v", recipe.dstImageInNewState))
		kits[i].scratchMemory = kb.scratchMemory
	}
	return kits, nil
}

func (kb *ipHostCopyKitBuilder) buildHostCopyKitPiece(
	sb *stateBuilder, dstImgObj ImageObjectʳ, dstAspect VkImageAspectFlagBits,
	srcImgObj ImageObjectʳ, srcAspect VkImageAspectFlagBits, subAspect ipHostCopyRecipeSubAspectPiece) (ipHostCopyKitPiece, error) {
	var err error
	srcVkFmt := srcImgObj.Info().Fmt()
	dstVkFmt := dstImgObj.Info().Fmt()
	kitPiece := ipHostCopyKitPiece{
		aspect:       dstAspect,
		layer:        subAspect.layer,
		level:        subAspect.level,
		offsetX:      subAspect.offsetX,
		offsetY:      subAspect.offsetY,
		offsetZ:      subAspect.offsetZ,
		extentWidth:  subAspect.extentWidth,
		extentHeight: subAspect.extentHeight,
		extentDepth:  subAspect.extentDepth,
	}
	srcImgLevel := srcImgObj.Aspects().Get(srcAspect).Layers().Get(
		subAspect.layer).Levels().Get(subAspect.level)
	unpackedData := []uint8{}
	if srcVkFmt != dstVkFmt {
		// dstImg format is different with the srcImage format, the dst image
		// should be a staging image.
		data := srcImgLevel.Data().MustRead(sb.ctx, nil, sb.oldState, nil)
		if srcVkFmt == VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32 {
			data, srcVkFmt, err = ebgrDataToRGB32SFloat(data,
				NewVkExtent3D(sb.ta,
					subAspect.extentWidth,
					subAspect.extentHeight,
					subAspect.extentDepth,
				),
			)
			if err != nil {
				return kitPiece, log.Errf(sb.ctx, err, "[Converting data in VK_FORMAT_E5B9G9R9_UFLOAT_PACK32 to VK_FORMAT_R32G32B32_SFLOAT]")
			}
		}
		unpackedData, _, err = unpackDataForPriming(sb.ctx, data, srcVkFmt, srcAspect)
		if err != nil {
			return kitPiece, log.Errf(sb.ctx, err, "[Unpacking data from format: %v aspect: %v]", srcVkFmt, srcAspect)
		}
	} else if srcAspect == VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT {
		// srcImg format is the same to the dstImage format, the data is ready to
		// be used directly, except when the src image is a dpeth 24 UNORM one.
		if (srcVkFmt == VkFormat_VK_FORMAT_D24_UNORM_S8_UINT) ||
			(srcVkFmt == VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32) {
			data := srcImgLevel.Data().MustRead(sb.ctx, nil, sb.oldState, nil)
			// data, err := src.data.Bytes(sb.ctx)
			if err != nil {
				return kitPiece, log.Errf(sb.ctx, err, "failed at resolving the source data")
			}
			unpackedData, _, err = unpackDataForPriming(sb.ctx, data, srcVkFmt, srcAspect)
			if err != nil {
				return kitPiece, log.Errf(sb.ctx, err, "[Unpacking data from format: %v aspect: %v]", srcVkFmt, srcAspect)
			}
		}
	}
	if len(unpackedData) != 0 {
		extendToMultipleOf8(&unpackedData)
		kitPiece.data = newHashedDataFromBytes(sb.ctx, unpackedData)
	} else if srcImgLevel.Data().Size()%8 != 0 {
		data := srcImgLevel.Data().MustRead(sb.ctx, nil, sb.oldState, nil)
		extendToMultipleOf8(&data)
		kitPiece.data = newHashedDataFromBytes(sb.ctx, data)
	} else {
		kitPiece.data = newHashedDataFromSlice(sb.ctx, sb.oldState, srcImgLevel.Data())
	}
	if err := checkHostCopyPieceDataSize(sb, dstVkFmt, dstAspect, kitPiece); err != nil {
		return kitPiece, log.Errf(sb.ctx, err, "failed at checking unpacked data size")
	}
	return kitPiece, nil
}

// ipHostCopyKitPiece describe a subresource of the priming target image and the
// data to be primed to that region.
type ipHostCopyKitPiece struct {
	aspect       VkImageAspectFlagBits
	layer        uint32
	level        uint32
	offsetX      uint32
	offsetY      uint32
	offsetZ      uint32
	extentWidth  uint32
	extentHeight uint32
	extentDepth  uint32
	data         hashedData
}

func checkHostCopyPieceDataSize(sb *stateBuilder, format VkFormat, aspect VkImageAspectFlagBits, p ipHostCopyKitPiece) error {
	levelSize := sb.levelSize(
		NewVkExtent3D(sb.ta, p.extentWidth, p.extentHeight, p.extentDepth),
		format, 0, aspect,
	)
	if p.data.size != levelSize.alignedLevelSizeInBuf {
		return fmt.Errorf("size of data does not match with expectation, actual: %v, expected: %v, format: %v, aspect: %v", p.data.size,
			levelSize.alignedLevelSizeInBuf, format, aspect)
	}
	return nil
}

// ipHostCopyKit constains all the necessary information to roll out command
// buffer commands to map host data to buffer then use buffer to image copy to
// copy data to the dstImage.
type ipHostCopyKit struct {
	name          debugMarkerName
	dstImage      VkImage
	pieces        []ipHostCopyKitPiece
	scratchMemory *flushingMemory
}

// BuildHostCopyCommands generates a queue comamnd batch, which when being
// committed to a queue command handler, will create a scratch buffer, map the
// data that scratch buffer, then record command buffer commands to copy the
// data from the buffer to that target image specified in this host copy kit.
func (kit ipHostCopyKit) BuildHostCopyCommands(sb *stateBuilder) *queueCommandBatch {
	cmdBatch := newQueueCommandBatch(kit.name.String())
	dataOffsetPieces := []hashedDataAndOffset{}
	copies := []VkBufferImageCopy{}
	bufferOffset := uint64(0)
	for _, p := range kit.pieces {
		copy := NewVkBufferImageCopy(sb.ta,
			VkDeviceSize(bufferOffset), // bufferOffset
			0,                          // bufferRowLength
			0,                          // bufferImageHeight
			NewVkImageSubresourceLayers(sb.ta, // imageSubresourceLayers
				VkImageAspectFlags(p.aspect),
				p.level, p.layer, 1,
			),
			NewVkOffset3D(sb.ta, int32(p.offsetX), int32(p.offsetY), int32(p.offsetZ)),
			NewVkExtent3D(sb.ta, p.extentWidth, p.extentHeight, p.extentDepth),
		)
		copies = append(copies, copy)
		dataWithOffset := newHashedDataAndOffset(p.data, bufferOffset)
		dataOffsetPieces = append(dataOffsetPieces, dataWithOffset)
		bufferOffset += p.data.size
		bufferOffset = nextMultipleOf(bufferOffset, 8)
	}
	bufferSize := bufferOffset
	dev := GetState(sb.newState).Images().Get(kit.dstImage).Device()
	scratchBuf := cmdBatch.NewScratchBuffer(sb, kit.name, kit.scratchMemory, dev,
		VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT),
		dataOffsetPieces...,
	)
	cmdBatch.RecordCommandsOnCommit(func(commandBuffer VkCommandBuffer) {
		sb.write(sb.cb.VkCmdPipelineBarrier(
			commandBuffer,
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_ALL_COMMANDS_BIT),
			VkDependencyFlags(0),
			uint32(0),
			memory.Nullptr,
			uint32(1),
			sb.MustAllocReadData(
				NewVkBufferMemoryBarrier(sb.ta,
					VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, // sType
					0, // pNext
					VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // srcAccessMask
					VkAccessFlags((VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT-1)|VkAccessFlagBits_VK_ACCESS_MEMORY_WRITE_BIT), // dstAccessMask
					queueFamilyIgnore,        // srcQueueFamilyIndex
					queueFamilyIgnore,        // dstQueueFamilyIndex
					scratchBuf,               // buffer
					0,                        // offset
					VkDeviceSize(bufferSize), // size
				)).Ptr(),
			uint32(0),
			memory.Nullptr,
		))
		sb.write(sb.cb.VkCmdCopyBufferToImage(
			commandBuffer,
			scratchBuf,
			kit.dstImage,
			VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
			uint32(len(copies)),
			sb.MustAllocReadData(copies).Ptr(),
		))
	})
	return cmdBatch
}
