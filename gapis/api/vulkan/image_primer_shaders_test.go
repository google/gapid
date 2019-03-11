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
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
)

func TestVertexShader(t *testing.T) {
	ctx := log.Testing(t)
	_, err := ipRenderVertexShaderSpirv()
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
		var err error
		switch info.aspect {
		case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
			_, err = ipRenderColorShaderSpirv(info.format)
		case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT:
			_, err = ipRenderDepthShaderSpirv(info.format)
		case VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
			_, err = ipRenderStencilShaderSpirv()
		default:
			err = fmt.Errorf("Unsupported aspect")
		}
		assert.For(ctx, "err").ThatError(err).Succeeded()
	}
}

func TestComputeShader(t *testing.T) {
	formats := []VkFormat{
		VkFormat_VK_FORMAT_R8_UINT,
		VkFormat_VK_FORMAT_R16_UINT,
		VkFormat_VK_FORMAT_R32_UINT,
		VkFormat_VK_FORMAT_R8G8_UINT,
		VkFormat_VK_FORMAT_R16G16_UINT,
		VkFormat_VK_FORMAT_R32G32_UINT,
		VkFormat_VK_FORMAT_R8G8B8A8_UINT,
		VkFormat_VK_FORMAT_B8G8R8A8_UINT,
		VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32,
		VkFormat_VK_FORMAT_R16G16B16A16_UINT,
		VkFormat_VK_FORMAT_R32G32B32A32_UINT,
		VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32,
		VkFormat_VK_FORMAT_R8_SINT,
		VkFormat_VK_FORMAT_R16_SINT,
		VkFormat_VK_FORMAT_R32_SINT,
		VkFormat_VK_FORMAT_R8G8_SINT,
		VkFormat_VK_FORMAT_R16G16_SINT,
		VkFormat_VK_FORMAT_R32G32_SINT,
		VkFormat_VK_FORMAT_R8G8B8A8_SINT,
		VkFormat_VK_FORMAT_B8G8R8A8_SINT,
		VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32,
		VkFormat_VK_FORMAT_R16G16B16A16_SINT,
		VkFormat_VK_FORMAT_R32G32B32A32_SINT,
		VkFormat_VK_FORMAT_R8_UNORM,
		VkFormat_VK_FORMAT_R8_SRGB,
		VkFormat_VK_FORMAT_R16_UNORM,
		VkFormat_VK_FORMAT_R8G8_UNORM,
		VkFormat_VK_FORMAT_R8G8_SRGB,
		VkFormat_VK_FORMAT_R16G16_UNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_UNORM,
		VkFormat_VK_FORMAT_B8G8R8A8_UNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_SRGB,
		VkFormat_VK_FORMAT_B8G8R8A8_SRGB,
		VkFormat_VK_FORMAT_R16G16B16A16_UNORM,
		VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32,
		VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32,
		VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32,
		VkFormat_VK_FORMAT_R8_SNORM,
		VkFormat_VK_FORMAT_R16_SNORM,
		VkFormat_VK_FORMAT_R8G8_SNORM,
		VkFormat_VK_FORMAT_R16G16_SNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_SNORM,
		VkFormat_VK_FORMAT_B8G8R8A8_SNORM,
		VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32,
		VkFormat_VK_FORMAT_R16G16B16A16_SNORM,
	}
	imageTypes := []VkImageType{
		VkImageType_VK_IMAGE_TYPE_1D,
		VkImageType_VK_IMAGE_TYPE_2D,
		VkImageType_VK_IMAGE_TYPE_3D,
	}

	ctx := log.Testing(t)

	// same input output format
	for _, f := range formats {
		for _, ty := range imageTypes {
			_, err := ipComputeShaderSpirv(
				f, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
				f, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
				ty)
			assert.For(ctx, "err").ThatError(err).Succeeded()
		}
	}

	// from R32G32B32A32_UINT to anything supported
	for _, f := range formats {
		for _, ty := range imageTypes {
			_, err := ipComputeShaderSpirv(
				f, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
				VkFormat_VK_FORMAT_R32G32B32A32_UINT, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
				ty)
			assert.For(ctx, "err").ThatError(err).Succeeded()
		}
	}
}

func TestComputeCopyShader(t *testing.T) {
	formats := []VkFormat{
		VkFormat_VK_FORMAT_R8_UINT,
		VkFormat_VK_FORMAT_R16_UINT,
		VkFormat_VK_FORMAT_R32_UINT,
		VkFormat_VK_FORMAT_R8G8_UINT,
		VkFormat_VK_FORMAT_R16G16_UINT,
		VkFormat_VK_FORMAT_R32G32_UINT,
		VkFormat_VK_FORMAT_R8G8B8A8_UINT,
		VkFormat_VK_FORMAT_B8G8R8A8_UINT,
		VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32,
		VkFormat_VK_FORMAT_R16G16B16A16_UINT,
		VkFormat_VK_FORMAT_R32G32B32A32_UINT,
		VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32,
		VkFormat_VK_FORMAT_R8_SINT,
		VkFormat_VK_FORMAT_R16_SINT,
		VkFormat_VK_FORMAT_R32_SINT,
		VkFormat_VK_FORMAT_R8G8_SINT,
		VkFormat_VK_FORMAT_R16G16_SINT,
		VkFormat_VK_FORMAT_R32G32_SINT,
		VkFormat_VK_FORMAT_R8G8B8A8_SINT,
		VkFormat_VK_FORMAT_B8G8R8A8_SINT,
		VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32,
		VkFormat_VK_FORMAT_R16G16B16A16_SINT,
		VkFormat_VK_FORMAT_R32G32B32A32_SINT,
		VkFormat_VK_FORMAT_R8_UNORM,
		VkFormat_VK_FORMAT_R8_SRGB,
		VkFormat_VK_FORMAT_R16_UNORM,
		VkFormat_VK_FORMAT_R8G8_UNORM,
		VkFormat_VK_FORMAT_R8G8_SRGB,
		VkFormat_VK_FORMAT_R16G16_UNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_UNORM,
		VkFormat_VK_FORMAT_B8G8R8A8_UNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_SRGB,
		VkFormat_VK_FORMAT_B8G8R8A8_SRGB,
		VkFormat_VK_FORMAT_R16G16B16A16_UNORM,
		VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32,
		VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32,
		VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32,
		VkFormat_VK_FORMAT_R8_SNORM,
		VkFormat_VK_FORMAT_R16_SNORM,
		VkFormat_VK_FORMAT_R8G8_SNORM,
		VkFormat_VK_FORMAT_R16G16_SNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_SNORM,
		VkFormat_VK_FORMAT_B8G8R8A8_SNORM,
		VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32,
		VkFormat_VK_FORMAT_R16G16B16A16_SNORM,
	}
	imageTypes := []VkImageType{
		VkImageType_VK_IMAGE_TYPE_1D,
		VkImageType_VK_IMAGE_TYPE_2D,
		VkImageType_VK_IMAGE_TYPE_3D,
	}

	ctx := log.Testing(t)

	// same input output format
	for _, f := range formats {
		for _, ty := range imageTypes {
			_, err := ipComputeCopySpirv(
				f, VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
				ty)
			assert.For(ctx, "err").ThatError(err).Succeeded()
		}
	}
}

func TestRenderByCopyShader(t *testing.T) {
	formats := []VkFormat{
		VkFormat_VK_FORMAT_R8_UINT,
		VkFormat_VK_FORMAT_R16_UINT,
		VkFormat_VK_FORMAT_R32_UINT,
		VkFormat_VK_FORMAT_R8G8_UINT,
		VkFormat_VK_FORMAT_R16G16_UINT,
		VkFormat_VK_FORMAT_R32G32_UINT,
		VkFormat_VK_FORMAT_R8G8B8A8_UINT,
		VkFormat_VK_FORMAT_B8G8R8A8_UINT,
		VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32,
		VkFormat_VK_FORMAT_R16G16B16A16_UINT,
		VkFormat_VK_FORMAT_R32G32B32A32_UINT,
		VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32,
		VkFormat_VK_FORMAT_R8_SINT,
		VkFormat_VK_FORMAT_R16_SINT,
		VkFormat_VK_FORMAT_R32_SINT,
		VkFormat_VK_FORMAT_R8G8_SINT,
		VkFormat_VK_FORMAT_R16G16_SINT,
		VkFormat_VK_FORMAT_R32G32_SINT,
		VkFormat_VK_FORMAT_R8G8B8A8_SINT,
		VkFormat_VK_FORMAT_B8G8R8A8_SINT,
		VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32,
		VkFormat_VK_FORMAT_R16G16B16A16_SINT,
		VkFormat_VK_FORMAT_R32G32B32A32_SINT,
		VkFormat_VK_FORMAT_R8_UNORM,
		VkFormat_VK_FORMAT_R8_SRGB,
		VkFormat_VK_FORMAT_R16_UNORM,
		VkFormat_VK_FORMAT_R8G8_UNORM,
		VkFormat_VK_FORMAT_R8G8_SRGB,
		VkFormat_VK_FORMAT_R16G16_UNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_UNORM,
		VkFormat_VK_FORMAT_B8G8R8A8_UNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_SRGB,
		VkFormat_VK_FORMAT_B8G8R8A8_SRGB,
		VkFormat_VK_FORMAT_R16G16B16A16_UNORM,
		VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32,
		VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32,
		VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32,
		VkFormat_VK_FORMAT_R8_SNORM,
		VkFormat_VK_FORMAT_R16_SNORM,
		VkFormat_VK_FORMAT_R8G8_SNORM,
		VkFormat_VK_FORMAT_R16G16_SNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_SNORM,
		VkFormat_VK_FORMAT_B8G8R8A8_SNORM,
		VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32,
		VkFormat_VK_FORMAT_R16G16B16A16_SNORM,
	}

	ctx := log.Testing(t)

	// same input output format
	for _, f := range formats {
		_, err := ipCopyByRenderShaderSpirv(f)
		assert.For(ctx, "err").ThatError(err).Succeeded()
	}
}
func TestRenderStencilByCopyShader(t *testing.T) {
	ctx := log.Testing(t)

	_, err := ipRenderStencilShaderSpirv()
	assert.For(ctx, "err").ThatError(err).Succeeded()
}
