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
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

// minimizeTextures returns a transform that sets the size of textures to 1x1
func minimizeTextures(ctx context.Context) transform.Transformer {
	ctx = log.Enter(ctx, "Minimize textures")
	return transform.Transform("Minimize textures", func(ctx context.Context,
		id api.CmdID, cmd api.Cmd, out transform.Writer) error {

		const newTexWidth = 1
		const newTexHeight = 1

		s := out.State()
		a := s.Arena
		l := s.MemoryLayout
		cb := CommandBuilder{Thread: cmd.Thread(), Arena: a}
		cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
		switch cmd := cmd.(type) {

		case *VkCreateImage:
			imageCreateInfo := cmd.PCreateInfo().MustRead(ctx, cmd, s, nil)
			if 0 != (imageCreateInfo.Usage() & VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)) {
				return out.MutateAndWrite(ctx, id, cmd)
			}

			imageCreateInfo.SetExtent(NewVkExtent3D(a, newTexWidth, newTexHeight, imageCreateInfo.Extent().Depth()))
			imageCreateInfo.SetMipLevels(1)

			imageCreateInfoData := s.AllocDataOrPanic(ctx, imageCreateInfo)
			defer imageCreateInfoData.Free()

			newCmd := cb.VkCreateImage(
				cmd.Device(),
				imageCreateInfoData.Ptr(),
				cmd.PAllocator(),
				cmd.PImage(),
				VkResult_VK_SUCCESS,
			).AddRead(imageCreateInfoData.Data())

			for _, w := range cmd.Extras().Observations().Writes {
				newCmd.AddWrite(w.Range, w.ID)
			}
			return out.MutateAndWrite(ctx, id, newCmd)

		case *VkCmdCopyBufferToImage:
			bufferImageCopy := cmd.PRegions().Slice(0, 1, l).MustRead(ctx, cmd, s, nil)[0]
			imageSubresource := bufferImageCopy.ImageSubresource()
			imageSubresource.SetMipLevel(0)
			bufferImageCopy.SetImageSubresource(imageSubresource)
			bufferImageCopy.SetImageExtent(NewVkExtent3D(a, newTexWidth, newTexHeight, bufferImageCopy.ImageExtent().Depth()))
			bufferImageCopy.SetImageOffset(NewVkOffset3D(a, 0, 0, bufferImageCopy.ImageOffset().Z()))

			bufferImageCopyData := s.AllocDataOrPanic(ctx, bufferImageCopy)
			defer bufferImageCopyData.Free()

			newCmd := cb.VkCmdCopyBufferToImage(
				cmd.commandBuffer,
				cmd.srcBuffer,
				cmd.dstImage,
				cmd.dstImageLayout,
				1, // regionCount
				bufferImageCopyData.Ptr(),
			).AddRead(bufferImageCopyData.Data())

			return out.MutateAndWrite(ctx, id, newCmd)

		case *VkCmdCopyImageToBuffer:
			bufferImageCopy := cmd.PRegions().Slice(0, 1, l).MustRead(ctx, cmd, s, nil)[0]
			imageSubresource := bufferImageCopy.ImageSubresource()
			imageSubresource.SetMipLevel(0)
			bufferImageCopy.SetImageSubresource(imageSubresource)
			bufferImageCopy.SetImageExtent(NewVkExtent3D(a, newTexWidth, newTexHeight, bufferImageCopy.ImageExtent().Depth()))
			bufferImageCopy.SetImageOffset(NewVkOffset3D(a, 0, 0, bufferImageCopy.ImageOffset().Z()))

			bufferImageCopyData := s.AllocDataOrPanic(ctx, bufferImageCopy)
			defer bufferImageCopyData.Free()

			newCmd := cb.VkCmdCopyImageToBuffer(
				cmd.commandBuffer,
				cmd.srcImage,
				cmd.srcImageLayout,
				cmd.dstBuffer,
				1, // regionCount
				bufferImageCopyData.Ptr(),
			).AddRead(bufferImageCopyData.Data())

			return out.MutateAndWrite(ctx, id, newCmd)

		case *VkCmdBlitImage:
			oldImageBlit := cmd.PRegions().MustRead(ctx, cmd, s, nil)
			srcSubresource := oldImageBlit.SrcSubresource()
			srcSubresource.SetMipLevel(0)
			dstSubresource := oldImageBlit.DstSubresource()
			dstSubresource.SetMipLevel(0)

			newImageBlit := NewVkImageBlit(a,
				srcSubresource,
				NewVkOffset3Dː2ᵃ(a, // srcOffsets (Bounds)
					NewVkOffset3D(a, 0, 0, oldImageBlit.SrcOffsets().Get(0).Z()),
					NewVkOffset3D(a, newTexWidth, newTexHeight, oldImageBlit.SrcOffsets().Get(1).Z()),
				),
				dstSubresource,
				NewVkOffset3Dː2ᵃ(a, // dstOffsets (Bounds)
					NewVkOffset3D(a, 0, 0, oldImageBlit.DstOffsets().Get(0).Z()),
					NewVkOffset3D(a, newTexWidth, newTexHeight, oldImageBlit.DstOffsets().Get(1).Z()),
				),
			)
			imageBlitData := s.AllocDataOrPanic(ctx, newImageBlit)
			defer imageBlitData.Free()

			newCmd := cb.VkCmdBlitImage(
				cmd.commandBuffer,
				cmd.srcImage,
				cmd.srcImageLayout,
				cmd.dstImage,
				cmd.dstImageLayout,
				1, // regionCount
				imageBlitData.Ptr(),
				cmd.filter,
			).AddRead(imageBlitData.Data())

			return out.MutateAndWrite(ctx, id, newCmd)

		case *VkCreateImageView:
			imageViewCreateInfo := cmd.PCreateInfo().MustRead(ctx, cmd, s, nil)
			subresourceRange := imageViewCreateInfo.SubresourceRange()
			subresourceRange.SetBaseMipLevel(0)
			subresourceRange.SetLevelCount(1)

			imageViewCreateInfo.SetSubresourceRange(subresourceRange)
			imageViewCreateInfoData := s.AllocDataOrPanic(ctx, imageViewCreateInfo)
			defer imageViewCreateInfoData.Free()

			newCmd := cb.VkCreateImageView(
				cmd.device,
				imageViewCreateInfoData.Ptr(),
				cmd.PAllocator(),
				cmd.PView(),
				VkResult_VK_SUCCESS,
			).AddRead(imageViewCreateInfoData.Data())

			for _, w := range cmd.Extras().Observations().Writes {
				newCmd.AddWrite(w.Range, w.ID)
			}

			return out.MutateAndWrite(ctx, id, newCmd)

		case *VkCmdClearColorImage:
			subresourceRanges := cmd.PRanges().MustRead(ctx, cmd, s, nil)
			subresourceRanges.SetBaseMipLevel(0)
			subresourceRanges.SetLevelCount(1)
			subresourceRangesData := s.AllocDataOrPanic(ctx, subresourceRanges)
			defer subresourceRangesData.Free()

			newCmd := cb.VkCmdClearColorImage(
				cmd.commandBuffer,
				cmd.image,
				cmd.imageLayout,
				cmd.pColor,
				1, // rangeCount
				subresourceRangesData.Ptr(),
			).AddRead(subresourceRangesData.Data())

			return out.MutateAndWrite(ctx, id, newCmd)

		default:
			return out.MutateAndWrite(ctx, id, cmd)
		}
	})
}
