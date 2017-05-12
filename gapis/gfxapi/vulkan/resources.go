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
	"fmt"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/image/astc"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/stream/fmts"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/shadertools"
)

func (t *ImageObject) IsResource() bool {
	// Since there are no good differentiating features for what is a "texture" and what
	// image is used for other things, we treat only images that can be used as SAMPLED
	// or STORAGE as resources. This may change in the future when we start doing
	// replays to get back gpu-generated image data.
	is_texture := 0 != (uint32(t.Info.Usage) & uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_SAMPLED_BIT|
		VkImageUsageFlagBits_VK_IMAGE_USAGE_STORAGE_BIT))
	return t.VulkanHandle != 0 && is_texture
}

// ResourceHandle returns the UI identity for the resource.
func (t *ImageObject) ResourceHandle() string {
	return fmt.Sprintf("Image<%d>", t.VulkanHandle)
}

// ResourceLabel returns an optional debug label for the resource.
func (t *ImageObject) ResourceLabel() string {
	return ""
}

// Order returns an integer used to sort the resources for presentation.
func (t *ImageObject) Order() uint64 {
	return uint64(t.VulkanHandle)
}

// ResourceType returns the type of this resource.
func (t *ImageObject) ResourceType(ctx context.Context) gfxapi.ResourceType {
	if uint32(t.Info.Flags)&uint32(VkImageCreateFlagBits_VK_IMAGE_CREATE_CUBE_COMPATIBLE_BIT) != 0 {
		return gfxapi.ResourceType_CubemapResource
	} else {
		return gfxapi.ResourceType_Texture2DResource
	}
}

type unsupportedVulkanFormatError struct {
	Format VkFormat
}

func (e *unsupportedVulkanFormatError) Error() string {
	return fmt.Sprintf("Unsupported Vulkan format: %d", e.Format)
}

func getImageFormatFromVulkanFormat(vkfmt VkFormat) (*image.Format, error) {
	switch vkfmt {
	case VkFormat_VK_FORMAT_R4G4_UNORM_PACK8:
		return image.NewUncompressed("VK_FORMAT_R4G4_UNORM_PACK8", fmts.RG_U4_NORM), nil
	case VkFormat_VK_FORMAT_R8_UNORM:
		return image.NewUncompressed("VK_FORMAT_R8_UNORM", fmts.R_U8_NORM), nil
	case VkFormat_VK_FORMAT_R8_SNORM:
		return image.NewUncompressed("VK_FORMAT_R8_SNORM", fmts.R_S8_NORM), nil
	case VkFormat_VK_FORMAT_R8_USCALED:
		return image.NewUncompressed("VK_FORMAT_R8_USCALED", fmts.R_U8), nil
	case VkFormat_VK_FORMAT_R8_SSCALED:
		return image.NewUncompressed("VK_FORMAT_R8_SSCALED", fmts.R_S8), nil
	case VkFormat_VK_FORMAT_R8_UINT:
		return image.NewUncompressed("VK_FORMAT_R8_UINT", fmts.R_U8), nil
	case VkFormat_VK_FORMAT_R8_SINT:
		return image.NewUncompressed("VK_FORMAT_R8_SINT", fmts.R_S8), nil
	case VkFormat_VK_FORMAT_R8_SRGB:
		return image.NewUncompressed("VK_FORMAT_R8_SRGB", fmts.R_U8_NORM_sRGB), nil
	case VkFormat_VK_FORMAT_R4G4B4A4_UNORM_PACK16:
		return image.NewUncompressed("VK_FORMAT_R4G4B4A4_UNORM_PACK16", fmts.RGBA_U4_NORM), nil
	case VkFormat_VK_FORMAT_B4G4R4A4_UNORM_PACK16:
		return image.NewUncompressed("VK_FORMAT_B4G4R4A4_UNORM_PACK16", fmts.BGRA_U4_NORM), nil
	case VkFormat_VK_FORMAT_R5G6B5_UNORM_PACK16:
		return image.NewUncompressed("VK_FORMAT_R5G6B5_UNORM_PACK16", fmts.RGB_U5U6U5_NORM), nil
	case VkFormat_VK_FORMAT_B5G6R5_UNORM_PACK16:
		return image.NewUncompressed("VK_FORMAT_B5G6R5_UNORM_PACK16", fmts.BGR_U5U6U5_NORM), nil
	case VkFormat_VK_FORMAT_R5G5B5A1_UNORM_PACK16:
		return image.NewUncompressed("VK_FORMAT_R5G5B5A1_UNORM_PACK16", fmts.RGBA_U5U5U5U1_NORM), nil
	case VkFormat_VK_FORMAT_B5G5R5A1_UNORM_PACK16:
		return image.NewUncompressed("VK_FORMAT_B5G5R5A1_UNORM_PACK16", fmts.BGRA_U5U5U5U1_NORM), nil
	case VkFormat_VK_FORMAT_A1R5G5B5_UNORM_PACK16:
		return image.NewUncompressed("VK_FORMAT_A1R5G5B5_UNORM_PACK16", fmts.ARGB_U1U5U5U5_NORM), nil
	case VkFormat_VK_FORMAT_R8G8_UNORM:
		return image.NewUncompressed("VK_FORMAT_R8G8_UNORM", fmts.RG_U8_NORM), nil
	case VkFormat_VK_FORMAT_R8G8_SNORM:
		return image.NewUncompressed("VK_FORMAT_R8G8_SNORM", fmts.RG_S8_NORM), nil
	case VkFormat_VK_FORMAT_R8G8_USCALED:
		return image.NewUncompressed("VK_FORMAT_R8G8_USCALED", fmts.RG_U8), nil
	case VkFormat_VK_FORMAT_R8G8_SSCALED:
		return image.NewUncompressed("VK_FORMAT_R8G8_SSCALED", fmts.RG_S8), nil
	case VkFormat_VK_FORMAT_R8G8_UINT:
		return image.NewUncompressed("VK_FORMAT_R8G8_UINT", fmts.RG_U8), nil
	case VkFormat_VK_FORMAT_R8G8_SINT:
		return image.NewUncompressed("VK_FORMAT_R8G8_SINT", fmts.RG_S8), nil
	case VkFormat_VK_FORMAT_R8G8_SRGB:
		return image.NewUncompressed("VK_FORMAT_R8G8_SRGB", fmts.RG_U8_NORM_sRGB), nil
	case VkFormat_VK_FORMAT_R16_UNORM:
		return image.NewUncompressed("VK_FORMAT_R16_UNORM", fmts.R_U16_NORM), nil
	case VkFormat_VK_FORMAT_R16_SNORM:
		return image.NewUncompressed("VK_FORMAT_R16_SNORM", fmts.R_S16_NORM), nil
	case VkFormat_VK_FORMAT_R16_USCALED:
		return image.NewUncompressed("VK_FORMAT_R16_USCALED", fmts.R_U16), nil
	case VkFormat_VK_FORMAT_R16_SSCALED:
		return image.NewUncompressed("VK_FORMAT_R16_USCALED", fmts.R_S16), nil
	case VkFormat_VK_FORMAT_R16_UINT:
		return image.NewUncompressed("VK_FORMAT_R16_UINT", fmts.R_U16), nil
	case VkFormat_VK_FORMAT_R16_SINT:
		return image.NewUncompressed("VK_FORMAT_R16_SINT", fmts.R_S16), nil
	case VkFormat_VK_FORMAT_R16_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R16_SFLOAT", fmts.R_F16), nil
	case VkFormat_VK_FORMAT_R8G8B8_UNORM:
		return image.NewUncompressed("VK_FORMAT_R8G8B8_UNORM", fmts.RGB_U8_NORM), nil
	case VkFormat_VK_FORMAT_R8G8B8_SNORM:
		return image.NewUncompressed("VK_FORMAT_R8G8B8_SNORM", fmts.RGB_S8_NORM), nil
	case VkFormat_VK_FORMAT_R8G8B8_USCALED:
		return image.NewUncompressed("VK_FORMAT_R8G8B8_USCALED", fmts.RGB_U8), nil
	case VkFormat_VK_FORMAT_R8G8B8_SSCALED:
		return image.NewUncompressed("VK_FORMAT_R8G8B8_SSCALED", fmts.RGB_S8), nil
	case VkFormat_VK_FORMAT_R8G8B8_UINT:
		return image.NewUncompressed("VK_FORMAT_R8G8B8_UINT", fmts.RGB_U8), nil
	case VkFormat_VK_FORMAT_R8G8B8_SINT:
		return image.NewUncompressed("VK_FORMAT_R8G8B8_SINT", fmts.RGB_S8), nil
	case VkFormat_VK_FORMAT_R8G8B8_SRGB:
		return image.NewUncompressed("VK_FORMAT_R8G8B8_SRGB", fmts.RGB_U8_NORM_sRGB), nil
	case VkFormat_VK_FORMAT_B8G8R8_UNORM:
		return image.NewUncompressed("VK_FORMAT_B8G8R8_UNORM", fmts.BGR_U8_NORM), nil
	case VkFormat_VK_FORMAT_B8G8R8_SNORM:
		return image.NewUncompressed("VK_FORMAT_B8G8R8_SNORM", fmts.BGR_S8_NORM), nil
	case VkFormat_VK_FORMAT_B8G8R8_USCALED:
		return image.NewUncompressed("VK_FORMAT_B8G8R8_SNORM", fmts.BGR_U8), nil
	case VkFormat_VK_FORMAT_B8G8R8_SSCALED:
		return image.NewUncompressed("VK_FORMAT_B8G8R8_SSCALED", fmts.BGR_S8), nil
	case VkFormat_VK_FORMAT_B8G8R8_UINT:
		return image.NewUncompressed("VK_FORMAT_B8G8R8_UINT", fmts.BGR_U8), nil
	case VkFormat_VK_FORMAT_B8G8R8_SINT:
		return image.NewUncompressed("VK_FORMAT_B8G8R8_SINT", fmts.BGR_S8), nil
	case VkFormat_VK_FORMAT_B8G8R8_SRGB:
		return image.NewUncompressed("VK_FORMAT_B8G8R8_SRGB", fmts.BGR_U8_NORM_sRGB), nil
	case VkFormat_VK_FORMAT_R8G8B8A8_UNORM:
		return image.NewUncompressed("VK_FORMAT_R8G8B8A8_UNORM", fmts.RGBA_U8_NORM), nil
	case VkFormat_VK_FORMAT_R8G8B8A8_SNORM:
		return image.NewUncompressed("VK_FORMAT_R8G8B8A8_UNORM", fmts.RGBA_S8_NORM), nil
	case VkFormat_VK_FORMAT_R8G8B8A8_USCALED:
		return image.NewUncompressed("VK_FORMAT_R8G8B8A8_USCALED", fmts.RGBA_U8), nil
	case VkFormat_VK_FORMAT_R8G8B8A8_SSCALED:
		return image.NewUncompressed("VK_FORMAT_R8G8B8A8_USCALED", fmts.RGBA_S8), nil
	case VkFormat_VK_FORMAT_R8G8B8A8_UINT:
		return image.NewUncompressed("VK_FORMAT_R8G8B8A8_UINT", fmts.RGBA_U8), nil
	case VkFormat_VK_FORMAT_R8G8B8A8_SINT:
		return image.NewUncompressed("VK_FORMAT_R8G8B8A8_SINT", fmts.RGBA_S8), nil
	case VkFormat_VK_FORMAT_R8G8B8A8_SRGB:
		return image.NewUncompressed("VK_FORMAT_R8G8B8A8_SRGB", fmts.RGBA_N_sRGBU8N_sRGBU8N_sRGBU8NU8), nil
	case VkFormat_VK_FORMAT_B8G8R8A8_UNORM:
		return image.NewUncompressed("VK_FORMAT_B8G8R8A8_UNORM", fmts.BGRA_U8_NORM), nil
	case VkFormat_VK_FORMAT_B8G8R8A8_SNORM:
		return image.NewUncompressed("VK_FORMAT_B8G8R8A8_SNORM", fmts.BGRA_S8_NORM), nil
	case VkFormat_VK_FORMAT_B8G8R8A8_USCALED:
		return image.NewUncompressed("VK_FORMAT_B8G8R8A8_USCALED", fmts.BGRA_U8), nil
	case VkFormat_VK_FORMAT_B8G8R8A8_SSCALED:
		return image.NewUncompressed("VK_FORMAT_B8G8R8A8_USCALED", fmts.BGRA_S8), nil
	case VkFormat_VK_FORMAT_B8G8R8A8_UINT:
		return image.NewUncompressed("VK_FORMAT_B8G8R8A8_UINT", fmts.BGRA_U8), nil
	case VkFormat_VK_FORMAT_B8G8R8A8_SINT:
		return image.NewUncompressed("VK_FORMAT_B8G8R8A8_SINT", fmts.BGRA_S8), nil
	case VkFormat_VK_FORMAT_B8G8R8A8_SRGB:
		return image.NewUncompressed("VK_FORMAT_B8G8R8A8_SRGB", fmts.BGRA_N_sRGBU8N_sRGBU8N_sRGBU8NU8), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_UNORM_PACK32", fmts.ABGR_U8_NORM), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_SNORM_PACK32", fmts.ABGR_S8_NORM), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_USCALED_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_USCALED_PACK32", fmts.ABGR_U8), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_SSCALED_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_SSCALED_PACK32", fmts.ABGR_S8), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_UINT_PACK32", fmts.ABGR_U8), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_SINT_PACK32", fmts.ABGR_S8), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_SRGB_PACK32", fmts.ABGR_NU8N_sRGBU8N_sRGBU8N_sRGBU8), nil
	case VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2R10G10B10_UNORM_PACK32", fmts.ARGB_U2U10U10U10_NORM), nil
	case VkFormat_VK_FORMAT_A2R10G10B10_SNORM_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2R10G10B10_SNORM_PACK32", fmts.ARGB_S2S10S10S10_NORM), nil
	case VkFormat_VK_FORMAT_A2R10G10B10_USCALED_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2R10G10B10_USCALED_PACK32", fmts.ARGB_U2U10U10U10), nil
	case VkFormat_VK_FORMAT_A2R10G10B10_SSCALED_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2R10G10B10_SSCALED_PACK32", fmts.ARGB_S2S10S10S10), nil
	case VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2R10G10B10_UINT_PACK32", fmts.ARGB_U2U10U10U10), nil
	case VkFormat_VK_FORMAT_A2R10G10B10_SINT_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2R10G10B10_SINT_PACK32", fmts.ARGB_S2S10S10S10), nil
	case VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2B10G10R10_UNORM_PACK32", fmts.ABGR_U2U10U10U10_NORM), nil
	case VkFormat_VK_FORMAT_A2B10G10R10_SNORM_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2B10G10R10_SNORM_PACK32", fmts.ABGR_S2S10S10S10_NORM), nil
	case VkFormat_VK_FORMAT_A2B10G10R10_USCALED_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2B10G10R10_USCALED_PACK32", fmts.ABGR_U2U10U10U10), nil
	case VkFormat_VK_FORMAT_A2B10G10R10_SSCALED_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2B10G10R10_SSCALED_PACK32", fmts.ABGR_S2S10S10S10), nil
	case VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2B10G10R10_UINT_PACK32", fmts.ABGR_U2U10U10U10), nil
	case VkFormat_VK_FORMAT_A2B10G10R10_SINT_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2B10G10R10_SINT_PACK32", fmts.ABGR_S2S10S10S10), nil
	case VkFormat_VK_FORMAT_R16G16_UNORM:
		return image.NewUncompressed("VK_FORMAT_R16G16_UNORM", fmts.RG_U16_NORM), nil
	case VkFormat_VK_FORMAT_R16G16_SNORM:
		return image.NewUncompressed("VK_FORMAT_R16G16_SNORM", fmts.RG_S16_NORM), nil
	case VkFormat_VK_FORMAT_R16G16_USCALED:
		return image.NewUncompressed("VK_FORMAT_R16G16_USCALED", fmts.RG_U16), nil
	case VkFormat_VK_FORMAT_R16G16_SSCALED:
		return image.NewUncompressed("VK_FORMAT_R16G16_SSCALED", fmts.RG_S16), nil
	case VkFormat_VK_FORMAT_R16G16_UINT:
		return image.NewUncompressed("VK_FORMAT_R16G16_UINT", fmts.RG_U16), nil
	case VkFormat_VK_FORMAT_R16G16_SINT:
		return image.NewUncompressed("VK_FORMAT_R16G16_SINT", fmts.RG_S16), nil
	case VkFormat_VK_FORMAT_R16G16_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R16G16_SFLOAT", fmts.RG_F16), nil
	case VkFormat_VK_FORMAT_R32_UINT:
		return image.NewUncompressed("VK_FORMAT_R32_UINT", fmts.R_U32), nil
	case VkFormat_VK_FORMAT_R32_SINT:
		return image.NewUncompressed("VK_FORMAT_R32_SINT", fmts.R_S32), nil
	case VkFormat_VK_FORMAT_R32_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R32_SINT", fmts.R_F32), nil
	case VkFormat_VK_FORMAT_B10G11R11_UFLOAT_PACK32:
		return image.NewUncompressed("VK_FORMAT_B10G11R11_UFLOAT_PACK32", fmts.BGR_F10F11F11), nil
	case VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	case VkFormat_VK_FORMAT_R16G16B16_UNORM:
		return image.NewUncompressed("VK_FORMAT_R16G16B16_UNORM", fmts.RGB_U16_NORM), nil
	case VkFormat_VK_FORMAT_R16G16B16_SNORM:
		return image.NewUncompressed("VK_FORMAT_R16G16B16_SNORM", fmts.RGB_S16_NORM), nil
	case VkFormat_VK_FORMAT_R16G16B16_USCALED:
		return image.NewUncompressed("VK_FORMAT_R16G16B16_USCALED", fmts.RGB_U16), nil
	case VkFormat_VK_FORMAT_R16G16B16_SSCALED:
		return image.NewUncompressed("VK_FORMAT_R16G16B16_SSCALED", fmts.RGB_S16), nil
	case VkFormat_VK_FORMAT_R16G16B16_UINT:
		return image.NewUncompressed("VK_FORMAT_R16G16B16_UINT", fmts.RGB_U16), nil
	case VkFormat_VK_FORMAT_R16G16B16_SINT:
		return image.NewUncompressed("VK_FORMAT_R16G16B16_SINT", fmts.RGB_S16), nil
	case VkFormat_VK_FORMAT_R16G16B16_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R16G16B16_SFLOAT", fmts.RGB_F32), nil
	case VkFormat_VK_FORMAT_R16G16B16A16_UNORM:
		return image.NewUncompressed("VK_FORMAT_R16G16B16A16_UNORM", fmts.RGBA_U16_NORM), nil
	case VkFormat_VK_FORMAT_R16G16B16A16_SNORM:
		return image.NewUncompressed("VK_FORMAT_R16G16B16A16_UNORM", fmts.RGBA_S16_NORM), nil
	case VkFormat_VK_FORMAT_R16G16B16A16_USCALED:
		return image.NewUncompressed("VK_FORMAT_R16G16B16A16_USCALED", fmts.RGBA_U16), nil
	case VkFormat_VK_FORMAT_R16G16B16A16_SSCALED:
		return image.NewUncompressed("VK_FORMAT_R16G16B16A16_USCALED", fmts.RGBA_S16), nil
	case VkFormat_VK_FORMAT_R16G16B16A16_UINT:
		return image.NewUncompressed("VK_FORMAT_R16G16B16A16_UINT", fmts.RGBA_U16), nil
	case VkFormat_VK_FORMAT_R16G16B16A16_SINT:
		return image.NewUncompressed("VK_FORMAT_R16G16B16A16_SINT", fmts.RGBA_S16), nil
	case VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R16G16B16A16_SFLOAT", fmts.RGBA_F16), nil
	case VkFormat_VK_FORMAT_R32G32_UINT:
		return image.NewUncompressed("VK_FORMAT_R32G32_UINT", fmts.RG_U32), nil
	case VkFormat_VK_FORMAT_R32G32_SINT:
		return image.NewUncompressed("VK_FORMAT_R32G32_SINT", fmts.RG_S32), nil
	case VkFormat_VK_FORMAT_R32G32_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R32G32_SFLOAT", fmts.RG_F32), nil
	case VkFormat_VK_FORMAT_R64_UINT:
		return image.NewUncompressed("VK_FORMAT_R64_UINT", fmts.R_U64), nil
	case VkFormat_VK_FORMAT_R64_SINT:
		return image.NewUncompressed("VK_FORMAT_R64_SINT", fmts.R_S64), nil
	case VkFormat_VK_FORMAT_R64_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R64_SFLOAT", fmts.R_F64), nil
	case VkFormat_VK_FORMAT_R32G32B32_UINT:
		return image.NewUncompressed("VK_FORMAT_R32G32B32_UINT", fmts.RGB_U32), nil
	case VkFormat_VK_FORMAT_R32G32B32_SINT:
		return image.NewUncompressed("VK_FORMAT_R32G32B32_SINT", fmts.RGB_S32), nil
	case VkFormat_VK_FORMAT_R32G32B32_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R32G32B32_SFLOAT", fmts.RGB_F32), nil
	case VkFormat_VK_FORMAT_R32G32B32A32_UINT:
		return image.NewUncompressed("VK_FORMAT_R32G32B32A32_UINT", fmts.RGBA_U32), nil
	case VkFormat_VK_FORMAT_R32G32B32A32_SINT:
		return image.NewUncompressed("VK_FORMAT_R32G32B32A32_SINT", fmts.RGBA_S32), nil
	case VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R32G32B32A32_SFLOAT", fmts.RGBA_F32), nil
	case VkFormat_VK_FORMAT_R64G64_UINT:
		return image.NewUncompressed("VK_FORMAT_R64G64_UINT", fmts.RG_U64), nil
	case VkFormat_VK_FORMAT_R64G64_SINT:
		return image.NewUncompressed("VK_FORMAT_R64G64_SINT", fmts.RG_S64), nil
	case VkFormat_VK_FORMAT_R64G64_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R64G64_SFLOAT", fmts.RG_F64), nil
	case VkFormat_VK_FORMAT_R64G64B64_UINT:
		return image.NewUncompressed("VK_FORMAT_R64G64B64_UINT", fmts.RGB_U64), nil
	case VkFormat_VK_FORMAT_R64G64B64_SINT:
		return image.NewUncompressed("VK_FORMAT_R64G64B64_SINT", fmts.RGB_S64), nil
	case VkFormat_VK_FORMAT_R64G64B64_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R64G64B64_SFLOAT", fmts.RGB_F64), nil
	case VkFormat_VK_FORMAT_R64G64B64A64_UINT:
		return image.NewUncompressed("VK_FORMAT_R64G64B64A64_UINT", fmts.RGBA_U64), nil
	case VkFormat_VK_FORMAT_R64G64B64A64_SINT:
		return image.NewUncompressed("VK_FORMAT_R64G64B64A64_SINT", fmts.RGBA_S64), nil
	case VkFormat_VK_FORMAT_R64G64B64A64_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_R64G64B64A64_SFLOAT", fmts.RGBA_F64), nil
	case VkFormat_VK_FORMAT_BC1_RGB_UNORM_BLOCK:
		return image.NewS3_DXT1_RGB("VK_FORMAT_BC1_RGB_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC1_RGB_SRGB_BLOCK:
		return image.NewS3_DXT1_RGB("VK_FORMAT_BC1_RGB_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC1_RGBA_UNORM_BLOCK:
		return image.NewS3_DXT1_RGBA("VK_FORMAT_BC1_RGBA_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC1_RGBA_SRGB_BLOCK:
		return image.NewS3_DXT1_RGBA("VK_FORMAT_BC1_RGBA_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC2_UNORM_BLOCK:
		return image.NewS3_DXT3_RGBA("VK_FORMAT_BC2_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC2_SRGB_BLOCK:
		return image.NewS3_DXT3_RGBA("VK_FORMAT_BC2_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC3_UNORM_BLOCK:
		return image.NewS3_DXT5_RGBA("VK_FORMAT_BC3_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC3_SRGB_BLOCK:
		return image.NewS3_DXT5_RGBA("VK_FORMAT_BC3_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC4_UNORM_BLOCK:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	case VkFormat_VK_FORMAT_BC4_SNORM_BLOCK:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	case VkFormat_VK_FORMAT_BC5_UNORM_BLOCK:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	case VkFormat_VK_FORMAT_BC5_SNORM_BLOCK:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	case VkFormat_VK_FORMAT_BC6H_UFLOAT_BLOCK:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	case VkFormat_VK_FORMAT_BC6H_SFLOAT_BLOCK:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	case VkFormat_VK_FORMAT_BC7_UNORM_BLOCK:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	case VkFormat_VK_FORMAT_BC7_SRGB_BLOCK:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	case VkFormat_VK_FORMAT_ETC2_R8G8B8_UNORM_BLOCK:
		return image.NewETC2_RGB_U8_NORM("VK_FORMAT_ETC2_R8G8B8_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ETC2_R8G8B8_SRGB_BLOCK:
		return image.NewETC2_RGB_U8_NORM("VK_FORMAT_ETC2_R8G8B8_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ETC2_R8G8B8A1_UNORM_BLOCK:
		return image.NewETC2_RGBA_U8U8U8U1_NORM("VK_FORMAT_ETC2_R8G8B8A1_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ETC2_R8G8B8A1_SRGB_BLOCK:
		return image.NewETC2_RGBA_U8U8U8U1_NORM("VK_FORMAT_ETC2_R8G8B8A1_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ETC2_R8G8B8A8_UNORM_BLOCK:
		return image.NewETC2_SRGBA_U8_NORM("VK_FORMAT_ETC2_R8G8B8A8_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ETC2_R8G8B8A8_SRGB_BLOCK:
		return image.NewETC2_SRGBA_U8_NORM("VK_FORMAT_ETC2_R8G8B8A8_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_EAC_R11_UNORM_BLOCK:
		return image.NewETC2_R_U11_NORM("VK_FORMAT_EAC_R11_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_EAC_R11_SNORM_BLOCK:
		return image.NewETC2_R_S11_NORM("VK_FORMAT_EAC_R11_SNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_EAC_R11G11_UNORM_BLOCK:
		return image.NewETC2_RG_U11_NORM("VK_FORMAT_EAC_R11G11_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_EAC_R11G11_SNORM_BLOCK:
		return image.NewETC2_RG_S11_NORM("VK_FORMAT_EAC_R11G11_SNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_4x4_UNORM_BLOCK:
		return astc.NewRGBA_4x4("VK_FORMAT_ASTC_4x4_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_4x4_SRGB_BLOCK:
		return astc.NewRGBA_4x4("VK_FORMAT_ASTC_4x4_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_5x4_UNORM_BLOCK:
		return astc.NewRGBA_5x4("VK_FORMAT_ASTC_5x4_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_5x4_SRGB_BLOCK:
		return astc.NewRGBA_5x4("VK_FORMAT_ASTC_5x4_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_5x5_UNORM_BLOCK:
		return astc.NewRGBA_5x5("VK_FORMAT_ASTC_5x5_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_5x5_SRGB_BLOCK:
		return astc.NewRGBA_5x5("VK_FORMAT_ASTC_5x5_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_6x5_UNORM_BLOCK:
		return astc.NewRGBA_6x5("VK_FORMAT_ASTC_6x5_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_6x5_SRGB_BLOCK:
		return astc.NewRGBA_6x5("VK_FORMAT_ASTC_6x5_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_6x6_UNORM_BLOCK:
		return astc.NewRGBA_6x6("VK_FORMAT_ASTC_6x6_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_6x6_SRGB_BLOCK:
		return astc.NewRGBA_6x6("VK_FORMAT_ASTC_6x6_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_8x5_UNORM_BLOCK:
		return astc.NewRGBA_8x5("VK_FORMAT_ASTC_8x5_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_8x5_SRGB_BLOCK:
		return astc.NewRGBA_8x5("VK_FORMAT_ASTC_8x5_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_8x6_UNORM_BLOCK:
		return astc.NewRGBA_8x6("VK_FORMAT_ASTC_8x6_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_8x6_SRGB_BLOCK:
		return astc.NewRGBA_8x6("VK_FORMAT_ASTC_8x6_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_8x8_UNORM_BLOCK:
		return astc.NewRGBA_8x8("VK_FORMAT_ASTC_8x8_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_8x8_SRGB_BLOCK:
		return astc.NewRGBA_8x8("VK_FORMAT_ASTC_8x8_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_10x5_UNORM_BLOCK:
		return astc.NewRGBA_10x5("VK_FORMAT_ASTC_10x5_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_10x5_SRGB_BLOCK:
		return astc.NewRGBA_10x5("VK_FORMAT_ASTC_10x5_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_10x6_UNORM_BLOCK:
		return astc.NewRGBA_10x6("VK_FORMAT_ASTC_10x6_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_10x6_SRGB_BLOCK:
		return astc.NewRGBA_10x6("VK_FORMAT_ASTC_10x6_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_10x8_UNORM_BLOCK:
		return astc.NewRGBA_10x8("VK_FORMAT_ASTC_10x8_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_10x8_SRGB_BLOCK:
		return astc.NewRGBA_10x8("VK_FORMAT_ASTC_10x8_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_10x10_UNORM_BLOCK:
		return astc.NewRGBA_10x10("VK_FORMAT_ASTC_10x10_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_10x10_SRGB_BLOCK:
		return astc.NewRGBA_10x10("VK_FORMAT_ASTC_10x10_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_12x10_UNORM_BLOCK:
		return astc.NewRGBA_12x10("VK_FORMAT_ASTC_12x10_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_12x10_SRGB_BLOCK:
		return astc.NewRGBA_12x10("VK_FORMAT_ASTC_12x10_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_12x12_UNORM_BLOCK:
		return astc.NewRGBA_12x12("VK_FORMAT_ASTC_12x12_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_ASTC_12x12_SRGB_BLOCK:
		return astc.NewRGBA_12x12("VK_FORMAT_ASTC_12x12_SRGB_BLOCK"), nil
	case VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
		return image.NewUncompressed("VK_FORMAT_D32_SFLOAT_S8_UINT", fmts.DS_F32U8), nil
	case VkFormat_VK_FORMAT_D32_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_D32_SFLOAT", fmts.D_F32), nil
	case VkFormat_VK_FORMAT_D16_UNORM:
		return image.NewUncompressed("VK_FORMAT_D16_UNORM", fmts.D_U16_NORM), nil
	case VkFormat_VK_FORMAT_D16_UNORM_S8_UINT:
		return image.NewUncompressed("VK_FORMAT_D16_UNORM_S8_UINT", fmts.DS_NU16S8), nil
	case VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32:
		return image.NewUncompressed("VK_FORMAT_X8_D24_UNORM_PACK32", fmts.ЖD_U8U24_NORM), nil
	case VkFormat_VK_FORMAT_D24_UNORM_S8_UINT:
		return image.NewUncompressed("VK_FORMAT_D24_UNORM_S8_UINT", fmts.DS_NU24S8), nil
	default:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	}
}

// Returns the corresponding depth format for the given Vulkan format. If the given Vulkan
// format contains a stencil field, returns a format which matches only with the tightly
// packed depth field of the given Vulkan format.
func getDepthImageFormatFromVulkanFormat(vkfmt VkFormat) (*image.Format, error) {
	switch vkfmt {
	case VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
		// Only the depth field is considered, and assume the data is tightly packed.
		return image.NewUncompressed("VK_FORMAT_D32_SFLOAT_S8_UINT", fmts.D_F32), nil
	case VkFormat_VK_FORMAT_D32_SFLOAT:
		return image.NewUncompressed("VK_FORMAT_D32_SFLOAT", fmts.D_F32), nil
	case VkFormat_VK_FORMAT_D16_UNORM:
		return image.NewUncompressed("VK_FORMAT_D16_UNORM", fmts.D_U16_NORM), nil
	case VkFormat_VK_FORMAT_D16_UNORM_S8_UINT:
		// Only the depth field is considered, and assume the data is tightly packed.
		return image.NewUncompressed("VK_FORMAT_D16_UNORM_S8_UINT", fmts.D_U16_NORM), nil
	case VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32:
		// Only the depth field is considered, and assume the data is tightly packed.
		return image.NewUncompressed("VK_FORMAT_X8_D24_UNORM_PACK32", fmts.D_U24_NORM), nil
	case VkFormat_VK_FORMAT_D24_UNORM_S8_UINT:
		// Only the depth field is considered, and assume the data is tightly packed.
		return image.NewUncompressed("VK_FORMAT_D24_UNORM_S8_UINT", fmts.D_U24_NORM), nil
	default:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	}
}

func setCubemapFace(img *image.Info2D, cubeMap *gfxapi.CubemapLevel, layerIndex uint32) (success bool) {
	if cubeMap == nil || img == nil {
		return false
	}
	switch layerIndex {
	case 0:
		cubeMap.PositiveX = img
	case 1:
		cubeMap.NegativeX = img
	case 2:
		cubeMap.PositiveY = img
	case 3:
		cubeMap.NegativeY = img
	case 4:
		cubeMap.PositiveZ = img
	case 5:
		cubeMap.NegativeZ = img
	default:
		return false
	}
	return true
}

// ResourceData returns the resource data given the current state.
func (t *ImageObject) ResourceData(ctx context.Context, s *gfxapi.State) (interface{}, error) {
	ctx = log.Enter(ctx, "ImageObject.Resource()")

	vkFmt := t.Info.Format
	format, err := getImageFormatFromVulkanFormat(vkFmt)
	if err != nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
	}
	switch t.Info.ImageType {
	case VkImageType_VK_IMAGE_TYPE_2D:
		// If this image has VK_IMAGE_CREATE_CUBE_COMPATIBLE_BIT set, it should have six layers to
		// represent a cubemap
		if uint32(t.Info.Flags)&uint32(VkImageCreateFlagBits_VK_IMAGE_CREATE_CUBE_COMPATIBLE_BIT) != 0 {
			// Cubemap
			cubeMapLevels := make([]*gfxapi.CubemapLevel, len(t.Layers[0].Levels))
			for l := range cubeMapLevels {
				cubeMapLevels[l] = &gfxapi.CubemapLevel{}
			}
			for layerIndex, imageLayer := range t.Layers {
				for levelIndex, imageLevel := range imageLayer.Levels {
					img := &image.Info2D{
						Format: format,
						Width:  imageLevel.Width,
						Height: imageLevel.Height,
						Data:   image.NewID(imageLevel.Data.ResourceID(ctx, s)),
					}
					if !setCubemapFace(img, cubeMapLevels[levelIndex], layerIndex) {
						return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
					}
				}
			}
			return &gfxapi.Cubemap{Levels: cubeMapLevels}, nil
		} else {
			levels := make([]*image.Info2D, len(t.Layers[0].Levels))
			for i, level := range t.Layers[0].Levels {
				levels[i] = &image.Info2D{
					Format: format,
					Width:  level.Width,
					Height: level.Height,
					Data:   image.NewID(level.Data.ResourceID(ctx, s)),
				}
			}
			return &gfxapi.Texture2D{Levels: levels}, nil
		}
	default:
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
	}
}

func (t *ImageObject) SetResourceData(ctx context.Context, at *path.Command,
	data interface{}, resources gfxapi.ResourceMap, edits gfxapi.ReplaceCallback) error {
	return fmt.Errorf("SetResourceData is not supported for ImageObject")
}

// IsResource returns true if this instance should be considered as a resource.
func (s *ShaderModuleObject) IsResource() bool {
	return true
}

// ResourceHandle returns the UI identity for the resource.
func (s *ShaderModuleObject) ResourceHandle() string {
	return fmt.Sprintf("Shader<0x%x>", s.VulkanHandle)
}

// ResourceLabel returns an optional debug label for the resource.
func (s *ShaderModuleObject) ResourceLabel() string {
	return ""
}

// Order returns an integer used to sort the resources for presentation.
func (s *ShaderModuleObject) Order() uint64 {
	return uint64(s.VulkanHandle)
}

// ResourceType returns the type of this resource.
func (s *ShaderModuleObject) ResourceType(ctx context.Context) gfxapi.ResourceType {
	return gfxapi.ResourceType_ShaderResource
}

// ResourceData returns the resource data given the current state.
func (s *ShaderModuleObject) ResourceData(ctx context.Context, t *gfxapi.State) (interface{}, error) {
	ctx = log.Enter(ctx, "Shader.ResourceData()")
	words := s.Words.Read(ctx, nil, t, nil)
	source := shadertools.DisassembleSpirvBinary(words)
	return &gfxapi.Shader{Type: gfxapi.ShaderType_Spirv, Source: source}, nil
}

func (shader *ShaderModuleObject) SetResourceData(
	ctx context.Context,
	at *path.Command,
	data interface{},
	resourceIDs gfxapi.ResourceMap,
	edits gfxapi.ReplaceCallback) error {

	ctx = log.Enter(ctx, "ShaderModuleObject.SetResourceData()")

	atomIdx := at.Index[0]
	if len(at.Index) > 1 {
		return fmt.Errorf("Subcommands currently not supported") // TODO: Subcommands
	}

	// Dirty. TODO: Make separate type for getting info for a single resource.
	resources, err := resolve.Resources(ctx, at.Capture)
	if err != nil {
		return err
	}
	resourceID := resourceIDs[shader]

	resource := resources.Find(shader.ResourceType(ctx), resourceID)
	if resource == nil {
		return fmt.Errorf("Couldn't find resource")
	}

	c, err := capture.ResolveFromPath(ctx, at.Capture)
	if err != nil {
		return err
	}

	index := len(resource.Accesses) - 1
	for resource.Accesses[index].Index[0] > atomIdx && index >= 0 { // TODO: Subcommands
		index--
	}
	for j := index; j >= 0; j-- {
		i := resource.Accesses[j].Index[0] // TODO: Subcommands
		if a, ok := c.Atoms[i].(*VkCreateShaderModule); ok {
			edits(uint64(i), a.Replace(ctx, data))
			return nil
		} else if a, ok := c.Atoms[i].(*RecreateShaderModule); ok {
			edits(uint64(i), a.Replace(ctx, data))
			return nil
		}
	}
	return fmt.Errorf("No atom to set data in")
}

func (a *VkCreateShaderModule) Replace(ctx context.Context, data interface{}) gfxapi.ResourceAtom {
	ctx = log.Enter(ctx, "VkCreateShaderModule.Replace()")
	state := capture.NewState(ctx)
	a.Mutate(ctx, state, nil)

	shader := data.(*gfxapi.Shader)
	codeSlice := shadertools.AssembleSpirvText(shader.Source)
	if codeSlice == nil {
		return nil
	}

	code := atom.Must(atom.AllocData(ctx, state, codeSlice))
	device := a.Device
	pAlloc := memory.Pointer(a.PAllocator)
	pShaderModule := memory.Pointer(a.PShaderModule)
	result := a.Result
	createInfo := a.PCreateInfo.Read(ctx, a, state, nil)

	createInfo.PCode = NewU32ᶜᵖ(code.Ptr())
	createInfo.CodeSize = uint64(len(codeSlice)) * 4
	// TODO(qining): The following is a hack to work around memory.Write().
	// In VkShaderModuleCreateInfo, CodeSize should be of type 'size', but
	// 'uint64' is used for now, and memory.Write() will always treat is as
	// a 8-byte type and causing padding issues so that won't encode the struct
	// correctly.
	// Possible solution: define another type 'size' and handle it correctly in
	// memory.Write().
	buf := &bytes.Buffer{}
	writer := endian.Writer(buf, state.MemoryLayout.GetEndian())
	VkShaderModuleCreateInfoEncodeRaw(state.MemoryLayout, writer, &createInfo)
	newCreateInfo := atom.Must(atom.AllocData(ctx, state, buf.Bytes()))
	newAtom := NewVkCreateShaderModule(device, newCreateInfo.Ptr(), pAlloc, pShaderModule, result)

	// Carry all non-observation extras through.
	for _, e := range a.Extras().All() {
		if _, ok := e.(*atom.Observations); !ok {
			newAtom.Extras().Add(e)
		}
	}

	// Add observations
	newAtom.AddRead(newCreateInfo.Data()).AddRead(code.Data())

	for _, w := range a.Extras().Observations().Writes {
		newAtom.AddWrite(w.Range, w.ID)
	}
	return newAtom
}

func (a *RecreateShaderModule) Replace(ctx context.Context, data interface{}) gfxapi.ResourceAtom {
	ctx = log.Enter(ctx, "RecreateShaderModule.Replace()")
	state := capture.NewState(ctx)
	a.Mutate(ctx, state, nil)

	shader := data.(*gfxapi.Shader)
	codeSlice := shadertools.AssembleSpirvText(shader.Source)
	if codeSlice == nil {
		return nil
	}

	code := atom.Must(atom.AllocData(ctx, state, codeSlice))
	device := a.Device
	pShaderModule := memory.Pointer(a.PShaderModule)
	createInfo := a.PCreateInfo.Read(ctx, a, state, nil)

	createInfo.PCode = NewU32ᶜᵖ(code.Ptr())
	createInfo.CodeSize = uint64(len(codeSlice)) * 4
	// TODO(qining): The following is a hack to work around memory.Write().
	// In VkShaderModuleCreateInfo, CodeSize should be of type 'size', but
	// 'uint64' is used for now, and memory.Write() will always treat is as
	// a 8-byte type and causing padding issues so that won't encode the struct
	// correctly.
	// Possible solution: define another type 'size' and handle it correctly in
	// memory.Write().
	buf := &bytes.Buffer{}
	writer := endian.Writer(buf, state.MemoryLayout.GetEndian())
	VkShaderModuleCreateInfoEncodeRaw(state.MemoryLayout, writer, &createInfo)
	newCreateInfo := atom.Must(atom.AllocData(ctx, state, buf.Bytes()))
	newAtom := NewRecreateShaderModule(device, newCreateInfo.Ptr(), pShaderModule)

	// Carry all non-observation extras through.
	for _, e := range a.Extras().All() {
		if _, ok := e.(*atom.Observations); !ok {
			newAtom.Extras().Add(e)
		}
	}

	// Add observations
	newAtom.AddRead(newCreateInfo.Data()).AddRead(code.Data())

	for _, w := range a.Extras().Observations().Writes {
		newAtom.AddWrite(w.Range, w.ID)
	}
	return newAtom
}
