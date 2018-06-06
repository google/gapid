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
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
)

func TestUnpackData(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)

	// Expect valid unpacked data with a 2x2x1 image
	valid := func(src []uint8, srcFmt, dstFmt VkFormat, aspect VkImageAspectFlagBits, expected []uint8) {
		var sf *image.Format
		switch aspect {
		case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
			sf, _ = getImageFormatFromVulkanFormat(srcFmt)
		case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT:
			sf, _ = getDepthImageFormatFromVulkanFormat(srcFmt)
		case VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
			sf, _ = getImageFormatFromVulkanFormat(VkFormat_VK_FORMAT_S8_UINT)
		}
		df, _ := getImageFormatFromVulkanFormat(dstFmt)
		r, err := unpackData(ctx, src, sf, df)

		if assert.For("srcFmt %v dstFmt %v", srcFmt, dstFmt).ThatError(err).Succeeded() {
			assert.For("srcFmt %v dstFmt %v", srcFmt, dstFmt).ThatSlice(r).Equals(expected)
		}
	}

	// uint type
	valid([]uint8{
		0xAB,
		0xCD,
		0xEF,
		0x12,
	}, VkFormat_VK_FORMAT_R8_UINT,
		VkFormat_VK_FORMAT_R32_UINT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
		[]uint8{
			0xAB, 0x00, 0x00, 0x00,
			0xCD, 0x00, 0x00, 0x00,
			0xEF, 0x00, 0x00, 0x00,
			0x12, 0x00, 0x00, 0x00,
		})

	// unorm type
	valid([]uint8{
		0xAB,
		0xCD,
		0xEF,
		0x12,
	}, VkFormat_VK_FORMAT_R8_UNORM,
		VkFormat_VK_FORMAT_R32_UINT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
		[]uint8{
			0xAB, 0x00, 0x00, 0x00,
			0xCD, 0x00, 0x00, 0x00,
			0xEF, 0x00, 0x00, 0x00,
			0x12, 0x00, 0x00, 0x00,
		})

	// sint type
	valid([]uint8{
		0xAB, // 10101011
		0xCD, // 11001101
		0xEF, // 11101111
		0x12, // 00010010
	}, VkFormat_VK_FORMAT_R8_SINT,
		VkFormat_VK_FORMAT_R32_UINT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
		[]uint8{
			0xAB, 0xFF, 0xFF, 0xFF,
			0xCD, 0xFF, 0xFF, 0xFF,
			0xEF, 0xFF, 0xFF, 0xFF,
			0x12, 0x00, 0x00, 0x00,
		})

	// snorm type
	valid([]uint8{
		0xAB, // 10101011
		0xCD, // 11001101
		0xEF, // 11101111
		0x12, // 00010010
	}, VkFormat_VK_FORMAT_R8_SNORM,
		VkFormat_VK_FORMAT_R32_UINT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
		[]uint8{
			0xAB, 0xFF, 0xFF, 0xFF,
			0xCD, 0xFF, 0xFF, 0xFF,
			0xEF, 0xFF, 0xFF, 0xFF,
			0x12, 0x00, 0x00, 0x00,
		})

	// f16 type
	valid([]uint8{
		0x00, 0x3D, // 1.25
		0xC0, 0x45, // 5.75
		0xB8, 0x57, // 123.5
		0x00, 0xC1, // -2.5
	}, VkFormat_VK_FORMAT_R16_SFLOAT,
		VkFormat_VK_FORMAT_R32_UINT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
		[]uint8{
			0x00, 0x00, 0xA0, 0x3F,
			0x00, 0x00, 0xB8, 0x40,
			0x00, 0x00, 0xF7, 0x42,
			0x00, 0x00, 0x20, 0xC0,
		})

	// f32 type
	valid([]uint8{
		0xDA, 0x0F, 0x49, 0x40, // 3.1415926
		0xDA, 0x0F, 0x49, 0xC0, // -3.1415926
		0xC2, 0xF3, 0x8E, 0x4D, // 299792458
		0xC2, 0xF3, 0x8E, 0xCD, // -299792458
	}, VkFormat_VK_FORMAT_R32_SFLOAT,
		VkFormat_VK_FORMAT_R32_UINT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
		[]uint8{
			0xDA, 0x0F, 0x49, 0x40,
			0xDA, 0x0F, 0x49, 0xC0,
			0xC2, 0xF3, 0x8E, 0x4D,
			0xC2, 0xF3, 0x8E, 0xCD,
		})
}
