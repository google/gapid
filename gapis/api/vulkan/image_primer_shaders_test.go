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
	"github.com/google/gapid/core/log"
)

func TestVertexShader(t *testing.T) {
	ctx := log.Testing(t)
	_, err := ipVertexShaderSpirv()
	assert.For(ctx, "err").ThatError(err).Succeeded()
}

func TestFragmentShader(t *testing.T) {
	for _, info := range []struct {
		format VkFormat
		aspect VkImageAspectFlagBits
	}{
		{VkFormat_VK_FORMAT_R8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8B8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8B8A8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B8G8R8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B8G8R8A8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16B16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16B16A16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R32G32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R32G32B32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R32G32B32A32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8B8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8B8A8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B8G8R8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B8G8R8A8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16B16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16B16A16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R32G32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R32G32B32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R32G32B32A32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A2R10G10B10_SINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A2B10G10R10_SINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8B8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8B8A8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B8G8R8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B8G8R8A8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8B8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8B8A8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B8G8R8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B8G8R8A8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16B16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16B16A16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R4G4_UNORM_PACK8, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R4G4B4A4_UNORM_PACK16, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B4G4R4A4_UNORM_PACK16, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R5G6B5_UNORM_PACK16, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B5G6R5_UNORM_PACK16, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R5G5B5A1_UNORM_PACK16, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B5G5R5A1_UNORM_PACK16, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A1R5G5B5_UNORM_PACK16, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8B8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R8G8B8A8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B8G8R8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B8G8R8A8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16B16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16B16A16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A2R10G10B10_SNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_A2B10G10R10_SNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16B16_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R32G32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R32G32B32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_B10G11R11_UFLOAT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},
		{VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT},

		{VkFormat_VK_FORMAT_D16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT},
		{VkFormat_VK_FORMAT_D16_UNORM_S8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT},
		{VkFormat_VK_FORMAT_D24_UNORM_S8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT},
		{VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT},
		{VkFormat_VK_FORMAT_D32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT},
		{VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT},

		{VkFormat_VK_FORMAT_S8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT},
		{VkFormat_VK_FORMAT_D16_UNORM_S8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT},
		{VkFormat_VK_FORMAT_D24_UNORM_S8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT},
		{VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT},
	} {
		ctx := log.Testing(t)
		_, err := ipFragmentShaderSpirv(info.format, info.aspect)
		assert.For(ctx, "err").ThatError(err).Succeeded()
	}
}

func TestComputeShader(t *testing.T) {
	for _, info := range []struct {
		format    VkFormat
		aspect    VkImageAspectFlagBits
		imageType VkImageType
	}{
		{VkFormat_VK_FORMAT_R8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8G8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16G16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R32G32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8G8B8A8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_B8G8R8A8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16G16B16A16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R32G32B32A32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8G8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16G16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R32G32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8G8B8A8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_B8G8R8A8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16G16B16A16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R32G32B32A32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8G8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8G8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16G16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8G8B8A8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_B8G8R8A8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8G8B8A8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_B8G8R8A8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16G16B16A16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8G8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16G16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R8G8B8A8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_B8G8R8A8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16G16B16A16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R32G32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},
		{VkFormat_VK_FORMAT_B10G11R11_UFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_1D},

		{VkFormat_VK_FORMAT_R8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8G8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16G16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R32G32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8G8B8A8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_B8G8R8A8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16G16B16A16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R32G32B32A32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8G8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16G16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R32G32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8G8B8A8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_B8G8R8A8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16G16B16A16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R32G32B32A32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8G8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8G8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16G16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8G8B8A8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_B8G8R8A8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8G8B8A8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_B8G8R8A8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16G16B16A16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8G8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16G16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R8G8B8A8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_B8G8R8A8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16G16B16A16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R32G32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},
		{VkFormat_VK_FORMAT_B10G11R11_UFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_2D},

		{VkFormat_VK_FORMAT_R8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8G8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16G16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R32G32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8G8B8A8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_B8G8R8A8_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16G16B16A16_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R32G32B32A32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8G8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16G16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R32G32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8G8B8A8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_B8G8R8A8_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16G16B16A16_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R32G32B32A32_SINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8G8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8G8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16G16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8G8B8A8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_B8G8R8A8_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8G8B8A8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_B8G8R8A8_SRGB, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16G16B16A16_UNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8G8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16G16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R8G8B8A8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_B8G8R8A8_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16G16B16A16_SNORM, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R32G32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
		{VkFormat_VK_FORMAT_B10G11R11_UFLOAT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT, VkImageType_VK_IMAGE_TYPE_3D},
	} {
		ctx := log.Testing(t)
		_, err := ipComputeShaderSpirv(info.format, info.aspect, info.imageType)
		assert.For(ctx, "err").ThatError(err).Succeeded()
	}
}
