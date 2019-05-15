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
	"github.com/google/gapid/core/math/u32"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

// minimizeTextures returns a transform that sets the size of textures to 1x1x1
func minimizeTextures(ctx context.Context) transform.Transformer {
	ctx = log.Enter(ctx, "Minimize textures")
	return transform.Transform("Minimize textures", func(ctx context.Context,
		id api.CmdID, cmd api.Cmd, out transform.Writer) {

		const newTexWidth = 1
		const newTexHeight = 1
		const newTexDepth = 1
		const regionCount = 1

		s := out.State()
		cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}
		cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
		switch cmd := cmd.(type) {
		case *VkCreateImage:
			imageCreateInfo := cmd.PCreateInfo().MustRead(ctx, cmd, s, nil)
			if 0 != (imageCreateInfo.Usage() & VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)) {
				out.MutateAndWrite(ctx, id, cmd)
				break
			}

			imageCreateInfo.SetExtent(NewVkExtent3D(s.Arena, newTexWidth, newTexHeight, newTexDepth))

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
			out.MutateAndWrite(ctx, id, newCmd)
		case *VkCmdCopyBufferToImage:
			bufferImageCopy := cmd.PRegions().MustRead(ctx, cmd, s, nil)
			bufferImageCopy.SetImageOffset(NewVkOffset3D(s.Arena, 0, 0, 0))
			imageExtent := bufferImageCopy.ImageExtent()
			width := u32.Min(imageExtent.Width(), newTexWidth)
			height := u32.Min(imageExtent.Height(), newTexHeight)
			depth := u32.Min(imageExtent.Depth(), newTexDepth)

			bufferImageCopy.SetImageExtent(NewVkExtent3D(s.Arena, width, height, depth))
			bufferImageCopyData := s.AllocDataOrPanic(ctx, bufferImageCopy)
			defer bufferImageCopyData.Free()

			newCmd := cb.VkCmdCopyBufferToImage(
				cmd.commandBuffer,
				cmd.srcBuffer,
				cmd.dstImage,
				cmd.dstImageLayout,
				regionCount,
				bufferImageCopyData.Ptr(),
			).AddRead(bufferImageCopyData.Data())

			out.MutateAndWrite(ctx, id, newCmd)
		case *VkCmdCopyImageToBuffer:
			bufferImageCopy := cmd.PRegions().MustRead(ctx, cmd, s, nil)
			bufferImageCopy.SetImageOffset(NewVkOffset3D(s.Arena, 0, 0, 0))
			imageExtent := bufferImageCopy.ImageExtent()
			width := u32.Min(imageExtent.Width(), newTexWidth)
			height := u32.Min(imageExtent.Height(), newTexHeight)
			depth := u32.Min(imageExtent.Depth(), newTexDepth)

			bufferImageCopy.SetImageExtent(NewVkExtent3D(s.Arena, width, height, depth))
			bufferImageCopyData := s.AllocDataOrPanic(ctx, bufferImageCopy)
			defer bufferImageCopyData.Free()

			newCmd := cb.VkCmdCopyImageToBuffer(
				cmd.commandBuffer,
				cmd.srcImage,
				cmd.srcImageLayout,
				cmd.dstBuffer,
				regionCount,
				bufferImageCopyData.Ptr(),
			).AddRead(bufferImageCopyData.Data())

			out.MutateAndWrite(ctx, id, newCmd)
		default:
			out.MutateAndWrite(ctx, id, cmd)
		}
	})
}
