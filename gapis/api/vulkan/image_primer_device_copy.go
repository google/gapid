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

import "fmt"

const (
	ipDeviceCopySrcImageLayout = VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL
	ipDeviceCopyDstImageLayout = VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
)

// ipDeviceCopyKit describes an image priming operation with a source and a
// destination image and a debug marker name. The source image and the
// destination image, when used for priming, must be in
// VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL and VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
// layout respectively.
type ipDeviceCopyKit struct {
	name     debugMarkerName
	srcImage VkImage
	dstImage VkImage
}

// ipBuildDeviceCopyKit takes a source image and a destination image then
// generates a device copy kit. The source image and the destination must have
// same format, same counts of mip levels, array layers, same extents, and same
// image type.
func ipBuildDeviceCopyKit(sb *stateBuilder, srcImage, dstImage VkImage) (ipDeviceCopyKit, error) {
	srcObj := GetState(sb.newState).Images().Get(srcImage)
	dstObj := GetState(sb.newState).Images().Get(dstImage)

	if srcObj.Info().Fmt() != dstObj.Info().Fmt() {
		return ipDeviceCopyKit{}, fmt.Errorf("src image format: %v does not match with dst image format: %v", srcObj.Info().Fmt(), dstObj.Info().Fmt())
	}
	if srcObj.Info().ImageType() != dstObj.Info().ImageType() {
		return ipDeviceCopyKit{}, fmt.Errorf("src image type: %v does not match with dst image type: %v", srcObj.Info().ImageType(), dstObj.Info().ImageType())
	}
	if srcObj.Info().MipLevels() != dstObj.Info().MipLevels() {
		return ipDeviceCopyKit{}, fmt.Errorf("src image num of miplevels: %v does not match with dst image: %v", srcObj.Info().MipLevels(), dstObj.Info().MipLevels())
	}
	if srcObj.Info().ArrayLayers() != dstObj.Info().ArrayLayers() {
		return ipDeviceCopyKit{}, fmt.Errorf("src image num of arraylayers: %v does not match with dst image: %v", srcObj.Info().ArrayLayers(), dstObj.Info().ArrayLayers())
	}
	if srcObj.Info().Samples() != dstObj.Info().Samples() {
		return ipDeviceCopyKit{}, fmt.Errorf("src image sample counts: %v does not match with dst image sample count: %v", srcObj.Info().Samples(), dstObj.Info().Samples())
	}
	if srcObj.Info().Extent().Width() != dstObj.Info().Extent().Width() {
		return ipDeviceCopyKit{}, fmt.Errorf("src image extent width: %v does not match with dst image: %v", srcObj.Info().Extent().Width(), dstObj.Info().Extent().Width())
	}
	if srcObj.Info().Extent().Height() != dstObj.Info().Extent().Height() {
		return ipDeviceCopyKit{}, fmt.Errorf("src image extent height: %v does not match with dst image: %v", srcObj.Info().Extent().Height(), dstObj.Info().Extent().Height())
	}
	if srcObj.Info().Extent().Depth() != dstObj.Info().Extent().Depth() {
		return ipDeviceCopyKit{}, fmt.Errorf("src image extent depth: %v does not match with dst image: %v", srcObj.Info().Extent().Depth(), dstObj.Info().Extent().Depth())
	}
	return ipDeviceCopyKit{
		name:     debugMarkerName(fmt.Sprintf("Copy device data from img: %v to img: %v", srcImage, dstImage)),
		srcImage: srcImage,
		dstImage: dstImage,
	}, nil
}

// BuildDeviceCopyCommands generates a queue command batch which when being
// committed to a queue command handler, will do an image -> image copy
// operation to prime the image data from the source image to the destination
// image. When the image is to be used by the generated queue command batch,
// the source image must be in the VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL layout
// and the destination image must be in the VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
// layout.
func (kit ipDeviceCopyKit) BuildDeviceCopyCommands(sb *stateBuilder) *queueCommandBatch {
	srcObj := GetState(sb.newState).Images().Get(kit.srcImage)
	copies := []VkImageCopy{}
	if isSparseResidency(srcObj) {
		walkSparseImageMemoryBindings(sb, srcObj, func(aspect VkImageAspectFlagBits, layer, level uint32, blockData SparseBoundImageBlockInfoʳ) {
			copies = append(copies, NewVkImageCopy(
				NewVkImageSubresourceLayers(
					VkImageAspectFlags(aspect),
					level,
					layer,
					uint32(1),
				), // srcSubresource
				NewVkOffset3D(
					blockData.Offset().X(),
					blockData.Offset().Y(),
					blockData.Offset().Z(),
				), // srcOffset
				NewVkImageSubresourceLayers(
					VkImageAspectFlags(aspect),
					level,
					layer,
					uint32(1),
				), // dstSubresource
				NewVkOffset3D(
					blockData.Offset().X(),
					blockData.Offset().Y(),
					blockData.Offset().Z(),
				), // dstOffset
				NewVkExtent3D(
					blockData.Extent().Width(),
					blockData.Extent().Height(),
					blockData.Extent().Depth(),
				), // extent
			))
		})
	} else {
		walkImageSubresourceRange(sb, srcObj, sb.imageWholeSubresourceRange(srcObj),
			func(aspect VkImageAspectFlagBits, layer, level uint32, levelSize byteSizeAndExtent) {
				copies = append(copies, NewVkImageCopy(
					NewVkImageSubresourceLayers(
						VkImageAspectFlags(aspect),
						level,
						layer,
						uint32(1),
					), // srcSubresource
					MakeVkOffset3D(), // srcOffset
					NewVkImageSubresourceLayers(
						VkImageAspectFlags(aspect),
						level,
						layer,
						uint32(1),
					), // dstSubresource
					MakeVkOffset3D(), // dstOffset
					NewVkExtent3D(
						uint32(levelSize.width),
						uint32(levelSize.height),
						uint32(levelSize.depth),
					), // extent
				))
			})
	}
	cmdBatch := newQueueCommandBatch(kit.name.String())
	cmdBatch.RecordCommandsOnCommit(func(commandBuffer VkCommandBuffer) {
		sb.write(sb.cb.VkCmdCopyImage(
			commandBuffer,
			kit.srcImage,
			ipDeviceCopySrcImageLayout,
			kit.dstImage,
			ipDeviceCopyDstImageLayout,
			uint32(len(copies)),
			sb.MustAllocReadData(copies).Ptr(),
		))
	})
	return cmdBatch
}
