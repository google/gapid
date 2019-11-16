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
	"encoding/binary"
	"fmt"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/image/astc"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/stream/fmts"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/shadertools"
)

var _ api.Resource = ImageObjectʳ{}

func (t ImageObjectʳ) IsResource() bool {
	// Since there are no good differentiating features for what is a "texture" and what
	// image is used for other things, we treat only images that can be used as SAMPLED
	// or STORAGE as resources. This may change in the future when we start doing
	// replays to get back gpu-generated image data.
	isTexture := 0 != (uint32(t.Info().Usage()) & uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_SAMPLED_BIT|
		VkImageUsageFlagBits_VK_IMAGE_USAGE_STORAGE_BIT))
	return t.VulkanHandle() != 0 && isTexture
}

// ResourceHandle returns the UI identity for the resource.
func (t ImageObjectʳ) ResourceHandle() string {
	return fmt.Sprintf("Image<%d>", t.VulkanHandle())
}

// ResourceLabel returns an optional debug label for the resource.
func (t ImageObjectʳ) ResourceLabel() string {
	if t.DebugInfo().IsNil() {
		return ""
	}
	if t.DebugInfo().ObjectName() != "" {
		return t.DebugInfo().ObjectName()
	}
	return fmt.Sprintf("<%d:%v>", t.DebugInfo().TagName(), t.DebugInfo().Tag())
}

// Order returns an integer used to sort the resources for presentation.
func (t ImageObjectʳ) Order() uint64 {
	return uint64(t.VulkanHandle())
}

// ResourceType returns the type of this resource.
func (t ImageObjectʳ) ResourceType(ctx context.Context) api.ResourceType {
	return api.ResourceType_TextureResource
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
		return image.NewUncompressed("VK_FORMAT_R8G8B8_SRGB", fmts.SRGB_U8_NORM), nil
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
		return image.NewUncompressed("VK_FORMAT_R8G8B8A8_SRGB", fmts.SRGBA_U8_NORM), nil
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
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_UNORM_PACK32", fmts.RGBA_U8_NORM), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_SNORM_PACK32", fmts.RGBA_S8_NORM), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_USCALED_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_USCALED_PACK32", fmts.RGBA_U8), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_SSCALED_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_SSCALED_PACK32", fmts.RGBA_S8), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_UINT_PACK32", fmts.RGBA_U8), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_SINT_PACK32", fmts.RGBA_S8), nil
	case VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32:
		return image.NewUncompressed("VK_FORMAT_A8B8G8R8_SRGB_PACK32", fmts.RGBA_sRGBU8N_sRGBU8N_sRGBU8_NU8N), nil
	case VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2R10G10B10_UNORM_PACK32", fmts.BGRA_U10U10U10U2_NORM), nil
	case VkFormat_VK_FORMAT_A2R10G10B10_SNORM_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2R10G10B10_SNORM_PACK32", fmts.BGRA_S10S10S10S2_NORM), nil
	case VkFormat_VK_FORMAT_A2R10G10B10_USCALED_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2R10G10B10_USCALED_PACK32", fmts.RGBA_U10U10U10U2), nil
	case VkFormat_VK_FORMAT_A2R10G10B10_SSCALED_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2R10G10B10_SSCALED_PACK32", fmts.RGBA_S10S10S10S2), nil
	case VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2R10G10B10_UINT_PACK32", fmts.BGRA_U10U10U10U2), nil
	case VkFormat_VK_FORMAT_A2R10G10B10_SINT_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2R10G10B10_SINT_PACK32", fmts.BGRA_S10S10S10S2), nil
	case VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2B10G10R10_UNORM_PACK32", fmts.RGBA_U10U10U10U2_NORM), nil
	case VkFormat_VK_FORMAT_A2B10G10R10_SNORM_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2B10G10R10_SNORM_PACK32", fmts.RGBA_S10S10S10S2_NORM), nil
	case VkFormat_VK_FORMAT_A2B10G10R10_USCALED_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2B10G10R10_USCALED_PACK32", fmts.RGBA_U10U10U10U2), nil
	case VkFormat_VK_FORMAT_A2B10G10R10_SSCALED_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2B10G10R10_SSCALED_PACK32", fmts.RGBA_S10S10S10S2), nil
	case VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2B10G10R10_UINT_PACK32", fmts.RGBA_U10U10U10U2), nil
	case VkFormat_VK_FORMAT_A2B10G10R10_SINT_PACK32:
		return image.NewUncompressed("VK_FORMAT_A2B10G10R10_SINT_PACK32", fmts.RGBA_S10S10S10S2), nil
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
		return image.NewUncompressed("VK_FORMAT_B10G11R11_UFLOAT_PACK32", fmts.RGB_F11F11F10), nil
	case VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32:
		return image.NewUncompressed("VK_FORMAT_E5B9G9R9_UFLOAT_PACK32", fmts.RGBE_U9U9U9U5), nil
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
		return image.NewRGTC1_BC4_R_U8_NORM("VK_FORMAT_BC4_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC4_SNORM_BLOCK:
		return image.NewRGTC1_BC4_R_S8_NORM("VK_FORMAT_BC4_SNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC5_UNORM_BLOCK:
		return image.NewRGTC2_BC5_RG_U8_NORM("VK_FORMAT_BC5_UNORM_BLOCK"), nil
	case VkFormat_VK_FORMAT_BC5_SNORM_BLOCK:
		return image.NewRGTC2_BC5_RG_S8_NORM("VK_FORMAT_BC5_SNORM_BLOCK"), nil
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
	case VkFormat_VK_FORMAT_S8_UINT:
		return image.NewUncompressed("VK_FORMAT_S8_UINT", fmts.S_U8), nil
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

// Returns the corresponding stencil format for the given Vulkan format. If the given Vulkan
// format contains a depth field, returns a format which matches only with the tightly
// packed stencil field of the given Vulkan format.
func getStencilImageFormatFromVulkanFormat(vkfmt VkFormat) (*image.Format, error) {
	switch vkfmt {
	case VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
		// Only the stencil field is considered, and assume the data is tightly packed.
		return image.NewUncompressed("VK_FORMAT_D32_SFLOAT_S8_UINT", fmts.S_U8), nil
	case VkFormat_VK_FORMAT_D16_UNORM_S8_UINT:
		// Only the stencil field is considered, and assume the data is tightly packed.
		return image.NewUncompressed("VK_FORMAT_D16_UNORM_S8_UINT", fmts.S_U8), nil
	case VkFormat_VK_FORMAT_D24_UNORM_S8_UINT:
		// Only the stencil field is considered, and assume the data is tightly packed.
		return image.NewUncompressed("VK_FORMAT_D24_UNORM_S8_UINT", fmts.S_U8), nil
	case VkFormat_VK_FORMAT_S8_UINT:
		return image.NewUncompressed("VK_FORMAT_S8_UINT", fmts.S_U8), nil
	default:
		return nil, &unsupportedVulkanFormatError{Format: vkfmt}
	}
}

func setCubemapFace(img *image.Info, cubeMap *api.CubemapLevel, layerIndex uint32) (success bool) {
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

func (t ImageObjectʳ) imageInfo(ctx context.Context, s *api.GlobalState, vkFmt VkFormat, layer, level uint32) *image.Info {
	if t.Info().ArrayLayers() <= layer || t.Info().MipLevels() <= level {
		return nil
	}
	format, _ := getImageFormatFromVulkanFormat(vkFmt)

	switch VkImageAspectFlagBits(t.ImageAspect()) {
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		l := t.Aspects().Get(VkImageAspectFlagBits(t.ImageAspect())).Layers().Get(layer).Levels().Get(level)
		if l.Data().Size() == 0 {
			return nil
		}
		ll := l.LinearLayout()
		expectedSize := format.Size(int(l.Width()), int(l.Height()), int(l.Depth()))
		if ll.IsNil() || ll.Size() == VkDeviceSize(expectedSize) {
			return &image.Info{
				Format: format,
				Width:  l.Width(),
				Height: l.Height(),
				Depth:  l.Depth(),
				Bytes:  image.NewID(l.Data().ResourceID(ctx, s)),
			}
		}
		elementAndTexelBlockSize, err := subGetElementAndTexelBlockSize(ctx, nil, api.CmdNoID, nil, s, nil, 0, nil, nil, vkFmt)
		if err != nil {
			log.Errf(ctx, err, "[Trim linear image data for image: %v]", t.VulkanHandle())
			return nil
		}
		texelHeight := elementAndTexelBlockSize.TexelBlockSize().Height()
		heightInBlocks, _ := subRoundUpTo(ctx, nil, api.CmdNoID, nil, s, nil, 0, nil, nil, l.Height(), texelHeight)
		colorRawSize := uint64(format.Size(int(l.Width()), 1, 1))
		levelDataResID := l.Data().ResourceID(ctx, s)

		resolveColorData := func() ([]byte, error) {
			levelDataRaw, err := database.Resolve(ctx, levelDataResID)
			if err != nil {
				return []byte{}, log.Errf(ctx, err, "[Resolve color image level data failed, image: %v, layer: %v, level: %v]", t.VulkanHandle(), layer, level)
			}
			levelData, ok := levelDataRaw.([]byte)
			if !ok {
				return []byte{}, log.Errf(ctx, err, "[Resolve color image level data failed, image: %v, layer: %v, level: %v]", t.VulkanHandle(), layer, level)
			}
			levelDataMinimumLen := uint64(l.Depth()-uint32(1))*uint64(ll.DepthPitch()) + uint64(heightInBlocks-uint32(1))*uint64(ll.RowPitch()) + colorRawSize
			if uint64(len(levelData)) < levelDataMinimumLen {
				return []byte{}, log.Errf(ctx, nil, "[Not enough image level data to get color data, image: %v, layer: %v, level: %v]", t.VulkanHandle(), layer, level)
			}
			colorData := make([]uint8, 0, expectedSize)
			for z := uint64(0); z < uint64(l.Depth()); z++ {
				for y := uint64(0); y < uint64(heightInBlocks); y++ {
					offset := z*uint64(ll.DepthPitch()) + y*uint64(ll.RowPitch())
					colorData = append(colorData, levelData[offset:offset+colorRawSize]...)
				}
			}
			return colorData, nil
		}
		colorDataID, err := database.Store(ctx, resolveColorData)
		if err != nil {
			log.Errf(ctx, err, "[Trim linear image data for image: %v]", t.VulkanHandle())
			return nil
		}

		return &image.Info{
			Format: format,
			Width:  l.Width(),
			Height: l.Height(),
			Depth:  l.Depth(),
			Bytes:  image.NewID(colorDataID),
		}

	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT | VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		depthLevel := t.Aspects().Get(VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT).Layers().Get(layer).Levels().Get(level)
		depthDataResID := depthLevel.Data().ResourceID(ctx, s)
		stencilLevel := t.Aspects().Get(VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT).Layers().Get(layer).Levels().Get(level)
		stencilDataResID := stencilLevel.Data().ResourceID(ctx, s)

		var dStep, sStep int
		// Stencil data is always 1 byte wide
		sStep = 1
		switch t.Info().Fmt() {
		case VkFormat_VK_FORMAT_D16_UNORM_S8_UINT:
			dStep = 2
		case VkFormat_VK_FORMAT_D24_UNORM_S8_UINT:
			dStep = 3
		case VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
			dStep = 4
		default:
			log.E(ctx, "[Mergeing depth and stencil data] unsupported depth+stencil format: %v", t.Info().Fmt())
			return nil
		}

		if int(depthLevel.Data().Count())/dStep != int(stencilLevel.Data().Count())/sStep {
			log.E(ctx, "[Merging depth and stencil data] depth/stencil data does not match")
			return nil
		}

		resolveDepthStencilData := func() ([]byte, error) {
			depthDataRaw, err := database.Resolve(ctx, depthDataResID)
			if err != nil {
				return []byte{}, log.Errf(ctx, err, "[Resolve depth image level data failed, image: %v, layer: %v, level: %v]", t.VulkanHandle(), layer, level)
			}
			depthData, ok := depthDataRaw.([]byte)
			if !ok {
				return []byte{}, log.Errf(ctx, err, "[Resolve depth image level data failed, image: %v, layer: %v, level: %v]", t.VulkanHandle(), layer, level)
			}
			if uint64(len(depthData)) != depthLevel.Data().Count() {
				return []byte{}, log.Errf(ctx, nil, "[Incorrect length of depth iamge level data, image: %v, layer: %v, level: %v]", t.VulkanHandle(), layer, level)
			}
			stencilDataRaw, err := database.Resolve(ctx, stencilDataResID)
			if err != nil {
				return []byte{}, log.Errf(ctx, err, "[Resolve stencil image level data failed, image: %v, layer: %v, level: %v]", t.VulkanHandle(), layer, level)
			}
			stencilData, ok := stencilDataRaw.([]byte)
			if !ok {
				return []byte{}, log.Errf(ctx, err, "[Resolve stencil image level data failed, image: %v, layer: %v, level: %v]", t.VulkanHandle(), layer, level)
			}
			if uint64(len(stencilData)) != stencilLevel.Data().Count() {
				return []byte{}, log.Errf(ctx, nil, "[Incorect length of stencil image level data, image: %v, layer: %v, level: %v]", t.VulkanHandle(), layer, level)
			}

			dsData := make([]uint8, depthLevel.Data().Count()+stencilLevel.Data().Count())
			// This assumes there are no 'packed' depth+stencil format, so the order
			// is always depth first, stencil later.
			for i := 0; i < len(stencilData); i++ {
				dsO := i * (dStep + sStep)
				dO := i * dStep
				sO := i
				copy(dsData[dsO:dsO+dStep], depthData[dO:dO+dStep])
				copy(dsData[dsO+dStep:dsO+dStep+sStep], stencilData[sO:sO+sStep])
			}
			return dsData, nil
		}
		depthStencilDataID, err := database.Store(ctx, resolveDepthStencilData)
		if err != nil {
			log.Errf(ctx, err, "[Storing depth data resolving callback for image: %v]", t.VulkanHandle())
		}

		return &image.Info{
			Format: format,
			Width:  depthLevel.Width(),
			Height: depthLevel.Height(),
			Depth:  depthLevel.Depth(),
			Bytes:  image.NewID(depthStencilDataID),
		}
	}
	return nil
}

// ResourceData returns the resource data given the current state.
func (t ImageObjectʳ) ResourceData(ctx context.Context, s *api.GlobalState) (*api.ResourceData, error) {
	ctx = log.Enter(ctx, "ImageObject.ResourceData()")
	vkFmt := t.Info().Fmt()
	_, err := getImageFormatFromVulkanFormat(vkFmt)
	if err != nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
	}
	switch t.Info().ImageType() {
	case VkImageType_VK_IMAGE_TYPE_2D:
		// If this image has VK_IMAGE_CREATE_CUBE_COMPATIBLE_BIT set, it should have six layers to
		// represent a cubemap, and the image type must not be VK_IMAGE_TYPE_3D
		if uint32(t.Info().Flags())&uint32(VkImageCreateFlagBits_VK_IMAGE_CREATE_CUBE_COMPATIBLE_BIT) != 0 {
			// Cubemap
			cubeMapLevels := make([]*api.CubemapLevel, t.Info().MipLevels())
			for l := range cubeMapLevels {
				cubeMapLevels[l] = &api.CubemapLevel{}
			}
			for layer := uint32(0); layer < t.Info().ArrayLayers(); layer++ {
				for level := uint32(0); level < t.Info().MipLevels(); level++ {
					info := t.imageInfo(ctx, s, vkFmt, layer, level)
					if info == nil {
						return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
					}
					if !setCubemapFace(info, cubeMapLevels[level], layer) {
						return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
					}
				}
			}
			return api.NewResourceData(api.NewTexture(&api.Cubemap{Levels: cubeMapLevels})), nil
		}

		if t.Info().ArrayLayers() > 1 {
			// 2D texture array
			layers := make([]*api.Texture2D, int(t.Info().ArrayLayers()))

			for layer := uint32(0); layer < t.Info().ArrayLayers(); layer++ {
				levels := make([]*image.Info, t.Info().MipLevels())
				for level := uint32(0); level < t.Info().MipLevels(); level++ {
					info := t.imageInfo(ctx, s, vkFmt, layer, level)
					if info == nil {
						return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
					}
					levels[level] = info
				}
				layers[layer] = &api.Texture2D{Levels: levels}
			}
			return api.NewResourceData(api.NewTexture(&api.Texture2DArray{Layers: layers})), nil
		}

		// Single layer 2D texture
		levels := make([]*image.Info, t.Info().MipLevels())
		for level := uint32(0); level < t.Info().MipLevels(); level++ {
			info := t.imageInfo(ctx, s, vkFmt, 0, level)
			if info == nil {
				return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
			}
			levels[level] = info
		}
		return api.NewResourceData(api.NewTexture(&api.Texture2D{Levels: levels})), nil

	case VkImageType_VK_IMAGE_TYPE_3D:
		// 3D images can have only one layer
		levels := make([]*image.Info, t.Info().MipLevels())
		for level := uint32(0); level < t.Info().MipLevels(); level++ {
			info := t.imageInfo(ctx, s, vkFmt, 0, level)
			if info == nil {
				return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
			}
			levels[level] = info
		}
		return api.NewResourceData(api.NewTexture(&api.Texture3D{Levels: levels})), nil

	case VkImageType_VK_IMAGE_TYPE_1D:
		if t.Info().ArrayLayers() > uint32(1) {
			// 1D texture array
			layers := make([]*api.Texture1D, int(t.Info().ArrayLayers()))
			for layer := uint32(0); layer < t.Info().ArrayLayers(); layer++ {
				levels := make([]*image.Info, t.Info().MipLevels())
				for level := uint32(0); level < t.Info().MipLevels(); level++ {
					info := t.imageInfo(ctx, s, vkFmt, layer, level)
					if info == nil {
						return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
					}
					levels[level] = info
				}
				layers[layer] = &api.Texture1D{Levels: levels}
			}
			return api.NewResourceData(api.NewTexture(&api.Texture1DArray{Layers: layers})), nil
		}
		// Single layer 1D texture
		levels := make([]*image.Info, t.Info().MipLevels())
		for level := uint32(0); level < t.Info().MipLevels(); level++ {
			info := t.imageInfo(ctx, s, vkFmt, 0, level)
			if info == nil {
				return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
			}
			levels[level] = info
		}
		return api.NewResourceData(api.NewTexture(&api.Texture1D{Levels: levels})), nil

	default:
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNoTextureData(t.ResourceHandle())}
	}
}

func (t ImageObjectʳ) SetResourceData(
	ctx context.Context,
	at *path.Command,
	data *api.ResourceData,
	resources api.ResourceMap,
	edits api.ReplaceCallback,
	mutate api.MutateInitialState,
	r *path.ResolveConfig) error {

	return fmt.Errorf("SetResourceData is not supported for ImageObject")
}

// IsResource returns true if this instance should be considered as a resource.
func (s ShaderModuleObjectʳ) IsResource() bool {
	return true
}

// ResourceHandle returns the UI identity for the resource.
func (s ShaderModuleObjectʳ) ResourceHandle() string {
	return fmt.Sprintf("Shader<0x%x>", s.VulkanHandle())
}

// ResourceLabel returns an optional debug label for the resource.
func (s ShaderModuleObjectʳ) ResourceLabel() string {
	if s.DebugInfo().IsNil() {
		return ""
	}
	if s.DebugInfo().ObjectName() != "" {
		return s.DebugInfo().ObjectName()
	}
	return fmt.Sprintf("<%d:%v>", s.DebugInfo().TagName(), s.DebugInfo().Tag())
}

// Order returns an integer used to sort the resources for presentation.
func (s ShaderModuleObjectʳ) Order() uint64 {
	return uint64(s.VulkanHandle())
}

// ResourceType returns the type of this resource.
func (s ShaderModuleObjectʳ) ResourceType(ctx context.Context) api.ResourceType {
	return api.ResourceType_ShaderResource
}

// ResourceData returns the resource data given the current state.
func (s ShaderModuleObjectʳ) ResourceData(ctx context.Context, t *api.GlobalState) (*api.ResourceData, error) {
	ctx = log.Enter(ctx, "ShaderModuleObject.ResourceData()")
	words, err := s.Words().Read(ctx, nil, t, nil)
	if err != nil {
		return nil, fmt.Errorf("Could not get resource data %v", err)
	}
	source := shadertools.DisassembleSpirvBinary(words)
	return api.NewResourceData(&api.Shader{Type: api.ShaderType_Spirv, Source: source}), nil
}

func (shader ShaderModuleObjectʳ) SetResourceData(
	ctx context.Context,
	at *path.Command,
	data *api.ResourceData,
	resourceIDs api.ResourceMap,
	edits api.ReplaceCallback,
	mutate api.MutateInitialState,
	r *path.ResolveConfig) error {

	ctx = log.Enter(ctx, "ShaderModuleObject.SetResourceData()")

	cmdIdx := at.Indices[0]

	// Dirty. TODO: Make separate type for getting info for a single resource.
	resources, err := resolve.Resources(ctx, at.Capture, r)
	if err != nil {
		return err
	}
	resourceID := resourceIDs[shader]

	resource, err := resources.Find(shader.ResourceType(ctx), resourceID)
	if err != nil {
		return err
	}

	c, err := capture.ResolveGraphicsFromPath(ctx, at.Capture)
	if err != nil {
		return err
	}

	index := len(resource.Accesses) - 1
	for resource.Accesses[index].Indices[0] > cmdIdx && index >= 0 { // TODO: Subcommands
		index--
	}
	for j := index; j >= 0; j-- {
		i := resource.Accesses[j].Indices[0] // TODO: Subcommands
		if cmd, ok := c.Commands[i].(*VkCreateShaderModule); ok {
			edits(uint64(i), cmd.Replace(ctx, c, data))
			return nil
		}
	}
	return fmt.Errorf("No command to set data in")
}

func (cmd *VkCreateShaderModule) Replace(ctx context.Context, c *capture.GraphicsCapture, data *api.ResourceData) interface{} {
	ctx = log.Enter(ctx, "VkCreateShaderModule.Replace()")
	cb := CommandBuilder{Thread: cmd.Thread(), Arena: c.Arena} // TODO: We probably should have a new arena passed in here!
	state := c.NewState(ctx)
	cmd.Mutate(ctx, api.CmdNoID, state, nil, nil)

	shader := data.GetShader()
	var codeSlice []uint32
	var codeSize int
	if shader.GetType() == api.ShaderType_Spirv {
		assembledCode := shadertools.AssembleSpirvText(shader.Source)
		codeSlice = assembledCode
		codeSize = len(assembledCode) * 4
	} else {
		codeSize = len(shader.Source)
		if codeSize%4 != 0 {
			log.E(ctx, "Invalid SPIR-V, number of bytes is not a multiple of 4")
		}
		cs := []byte(shader.Source)
		codeSlice = make([]uint32, len(cs)/4)
		for i := range codeSlice {
			codeSlice[i] = uint32(binary.LittleEndian.Uint32(cs[i*4 : (i+1)*4]))
		}
	}

	if len(codeSlice) == 0 {
		log.E(ctx, "Failed at assembling new SPIR-V shader code. Shader module unchanged.")
		return cmd
	}

	code := state.AllocDataOrPanic(ctx, codeSlice)
	device := cmd.Device()
	pAlloc := memory.Pointer(cmd.PAllocator())
	pShaderModule := memory.Pointer(cmd.PShaderModule())
	result := cmd.Result()
	createInfo := cmd.PCreateInfo().MustRead(ctx, cmd, state, nil)

	createInfo.SetPCode(NewU32ᶜᵖ(code.Ptr()))
	createInfo.SetCodeSize(memory.Size(codeSize))
	newCreateInfo := state.AllocDataOrPanic(ctx, createInfo)
	newCmd := cb.VkCreateShaderModule(device, newCreateInfo.Ptr(), pAlloc, pShaderModule, result)

	descriptors, err := shadertools.ParseAllDescriptorSets(codeSlice)
	u := MakeDescriptorInfo(c.Arena)
	dsc := u.Descriptors()
	if err != nil {
		log.E(ctx, "Could not parse SPIR-V")
	} else {
		for name, desc := range descriptors {
			d := NewU32ːDescriptorUsageᵐ(c.Arena)
			for _, set := range desc {
				for _, binding := range set {
					d.Add(uint32(d.Len()),
						NewDescriptorUsage(
							c.Arena,
							binding.Set,
							binding.Binding,
							binding.DescriptorCount))
				}
			}
			dsc.Add(name, d)
		}
		// Add our new used descriptors
		newCmd.Extras().Add(u)
	}

	// Carry all non-observation extras through,
	// except for the Used descriptors
	for _, e := range cmd.Extras().All() {
		if _, ok := e.(*api.CmdObservations); !ok {
			if _, ok := e.(DescriptorInfo); !ok {
				newCmd.Extras().Add(e)
			}
		}
	}

	// Add observations
	newCmd.AddRead(newCreateInfo.Data()).AddRead(code.Data())

	for _, w := range cmd.Extras().Observations().Writes {
		newCmd.AddWrite(w.Range, w.ID)
	}
	return newCmd
}

var _ api.Resource = GraphicsPipelineObjectʳ{}

func (p GraphicsPipelineObjectʳ) IsResource() bool {
	return true
}

// ResourceHandle returns the UI identity for the resource.
func (p GraphicsPipelineObjectʳ) ResourceHandle() string {
	return fmt.Sprintf("GraphicsPipeline<%d>", p.VulkanHandle())
}

// ResourceLabel returns an optional debug label for the resource.
func (p GraphicsPipelineObjectʳ) ResourceLabel() string {
	if p.DebugInfo().IsNil() {
		return ""
	}
	if p.DebugInfo().ObjectName() != "" {
		return p.DebugInfo().ObjectName()
	}
	return fmt.Sprintf("<%d:%v>", p.DebugInfo().TagName(), p.DebugInfo().Tag())
}

// Order returns an integer used to sort the resources for presentation.
func (p GraphicsPipelineObjectʳ) Order() uint64 {
	return uint64(p.VulkanHandle())
}

// ResourceType returns the type of this resource.
func (p GraphicsPipelineObjectʳ) ResourceType(ctx context.Context) api.ResourceType {
	return api.ResourceType_PipelineResource
}

// ResourceData returns the resource data given the current state.
func (p GraphicsPipelineObjectʳ) ResourceData(ctx context.Context, s *api.GlobalState) (*api.ResourceData, error) {
	vkState := GetState(s)
	isBound := false
	// Use LastDrawInfos to get bound descriptor set data.
	// TODO: Ideally we could look at just a specific pipeline/descriptor
	// set pair.  Maybe we could modify mutate to track which what
	// descriptor sets were bound to particular pipelines.
	if !vkState.LastBoundQueue().IsNil() {
		ldi, ok := vkState.LastDrawInfos().Lookup(vkState.LastBoundQueue().VulkanHandle())
		if ok {
			if ldi.GraphicsPipeline() == p {
				isBound = true
			}
		}
	}

	stages := []*api.Stage{
		p.inputAssembly(),
		p.vertexShader(ctx, s),
		p.tessellationControlShader(ctx, s),
		p.tessellationEvulationShader(ctx, s),
		p.geometryShader(ctx, s),
		p.rasterizer(),
		p.fragmentShader(ctx, s),
		p.colorBlending(),
	}

	return &api.ResourceData{
		Data: &api.ResourceData_Pipeline{
			Pipeline: &api.Pipeline{
				API:          path.NewAPI(id.ID(ID)),
				PipelineType: api.Pipeline_GRAPHICS,
				DebugName:    "GRAPH",
				Stages:       stages,
				Bound:        isBound,
			},
		},
	}, nil
}

func (p GraphicsPipelineObjectʳ) inputAssembly() *api.Stage {
	bindings := p.VertexInputState().BindingDescriptions()

	bindingRows := make([]*api.Row, bindings.Len())
	for i, index := range bindings.Keys() {
		binding := bindings.Get(index)

		bindingRows[i] = &api.Row{
			RowValues: []*api.DataValue{
				api.CreatePoDDataValue("uint32_t ", binding.Binding()),
				api.CreatePoDDataValue("uint32_t ", binding.Stride()),
				api.CreateEnumDataValue("VkVertexInputRate", binding.InputRate()),
			},
		}
	}

	vertexBindingsTable := &api.Table{
		Headers: []string{"Binding", "Stride", "Vertex Input Rate"},
		Rows:    bindingRows,
	}

	attributes := p.VertexInputState().AttributeDescriptions()

	attributeRows := make([]*api.Row, attributes.Len())
	for i, index := range attributes.Keys() {
		attribute := attributes.Get(index)

		attributeRows[i] = &api.Row{
			RowValues: []*api.DataValue{
				api.CreatePoDDataValue("uint32_t", attribute.Location()),
				api.CreatePoDDataValue("uint32_t", attribute.Binding()),
				api.CreateEnumDataValue("VkFormat", attribute.Fmt()),
				api.CreatePoDDataValue("uint32_t", attribute.Offset()),
			},
		}
	}

	vertexAttributesTable := &api.Table{
		Headers: []string{"Location", "Binding", "Format", "Offset"},
		Rows:    attributeRows,
	}

	assemblyList := &api.KeyValuePairList{}
	assemblyList = assemblyList.AppendKeyValuePair("Topology", api.CreateEnumDataValue("VkPrimitiveTopology", p.InputAssemblyState().Topology()))
	assemblyList = assemblyList.AppendKeyValuePair("Primitive Restart Enabled", api.CreatePoDDataValue("VkBool32",
		p.InputAssemblyState().PrimitiveRestartEnable() != 0))

	dataGroups := []*api.DataGroup{
		&api.DataGroup{
			GroupName: "Vertex Bindings",
			Data:      &api.DataGroup_Table{vertexBindingsTable},
		},

		&api.DataGroup{
			GroupName: "Vertex Attributes",
			Data:      &api.DataGroup_Table{vertexAttributesTable},
		},

		&api.DataGroup{
			GroupName: "Input Assembly State",
			Data:      &api.DataGroup_KeyValues{assemblyList},
		},
	}

	return &api.Stage{
		StageName: "Input Assembly",
		DebugName: "IA",
		Enabled:   true,
		Groups:    dataGroups,
	}
}

func (p GraphicsPipelineObjectʳ) vertexShader(ctx context.Context, s *api.GlobalState) *api.Stage {
	stages := p.Stages()
	for _, index := range stages.Keys() {
		stage := stages.Get(index)

		if stage.Stage() == VkShaderStageFlagBits_VK_SHADER_STAGE_VERTEX_BIT {
			module := stage.Module()

			words, _ := module.Words().Read(ctx, nil, s, nil)
			source := shadertools.DisassembleSpirvBinary(words)
			shader := &api.Shader{Type: api.ShaderType_Spirv, Source: source}

			dataGroups := []*api.DataGroup{
				&api.DataGroup{
					GroupName: "Shader Code",
					Data:      &api.DataGroup_Shader{shader},
				},
			}

			return &api.Stage{
				StageName: "Vertex Shader",
				DebugName: "VS",
				Enabled:   true,
				Groups:    dataGroups,
			}
		}
	}

	// For vertex shader, technically shouldn't happen
	return &api.Stage{
		StageName: "Vertex Shader",
		DebugName: "VS",
		Enabled:   false,
	}
}

func (p GraphicsPipelineObjectʳ) tessellationControlShader(ctx context.Context, s *api.GlobalState) *api.Stage {
	stages := p.Stages()
	for _, index := range stages.Keys() {
		stage := stages.Get(index)

		if stage.Stage() == VkShaderStageFlagBits_VK_SHADER_STAGE_TESSELLATION_CONTROL_BIT {
			module := stage.Module()

			words, _ := module.Words().Read(ctx, nil, s, nil)
			source := shadertools.DisassembleSpirvBinary(words)
			shader := &api.Shader{Type: api.ShaderType_Spirv, Source: source}

			dataGroups := []*api.DataGroup{
				&api.DataGroup{
					GroupName: "Shader Code",
					Data:      &api.DataGroup_Shader{shader},
				},
			}

			return &api.Stage{
				StageName: "Tessellation Control Shader",
				DebugName: "TCS",
				Enabled:   true,
				Groups:    dataGroups,
			}
		}
	}

	return &api.Stage{
		StageName: "Tessellation Control Shader",
		DebugName: "TCS",
		Enabled:   false,
	}
}

func (p GraphicsPipelineObjectʳ) tessellationEvulationShader(ctx context.Context, s *api.GlobalState) *api.Stage {
	stages := p.Stages()
	for _, index := range stages.Keys() {
		stage := stages.Get(index)

		if stage.Stage() == VkShaderStageFlagBits_VK_SHADER_STAGE_TESSELLATION_EVALUATION_BIT {
			stage := stages.Get(index)

			module := stage.Module()

			words, _ := module.Words().Read(ctx, nil, s, nil)
			source := shadertools.DisassembleSpirvBinary(words)
			shader := &api.Shader{Type: api.ShaderType_Spirv, Source: source}

			dataGroups := []*api.DataGroup{
				&api.DataGroup{
					GroupName: "Shader Code",
					Data:      &api.DataGroup_Shader{shader},
				},
			}

			return &api.Stage{
				StageName: "Tessellation Evaluation Shader",
				DebugName: "TES",
				Enabled:   true,
				Groups:    dataGroups,
			}
		}
	}

	return &api.Stage{
		StageName: "Tessellation Evaluation Shader",
		DebugName: "TES",
		Enabled:   false,
	}
}

func (p GraphicsPipelineObjectʳ) geometryShader(ctx context.Context, s *api.GlobalState) *api.Stage {
	stages := p.Stages()
	for _, index := range stages.Keys() {
		stage := stages.Get(index)

		if stage.Stage() == VkShaderStageFlagBits_VK_SHADER_STAGE_GEOMETRY_BIT {
			stage := stages.Get(index)

			module := stage.Module()

			words, _ := module.Words().Read(ctx, nil, s, nil)
			source := shadertools.DisassembleSpirvBinary(words)
			shader := &api.Shader{Type: api.ShaderType_Spirv, Source: source}

			dataGroups := []*api.DataGroup{
				&api.DataGroup{
					GroupName: "Shader Code",
					Data:      &api.DataGroup_Shader{shader},
				},
			}

			return &api.Stage{
				StageName: "Geometry Shader",
				DebugName: "GS",
				Enabled:   true,
				Groups:    dataGroups,
			}
		}
	}

	return &api.Stage{
		StageName: "Geometry Shader",
		DebugName: "GS",
		Enabled:   false,
	}
}

func (p GraphicsPipelineObjectʳ) rasterizer() *api.Stage {
	rasterState := p.RasterizationState()
	rasterList := &api.KeyValuePairList{}
	rasterList = rasterList.AppendKeyValuePair("Depth Clamp Enabled", api.CreatePoDDataValue("VkBool32", rasterState.DepthClampEnable() != 0))
	rasterList = rasterList.AppendKeyValuePair("Rasterizer Discard", api.CreatePoDDataValue("VkBool32", rasterState.RasterizerDiscardEnable() != 0))
	rasterList = rasterList.AppendKeyValuePair("Polygon Mode", api.CreateEnumDataValue("VkPolygonMode", rasterState.PolygonMode()))
	rasterList = rasterList.AppendKeyValuePair("Cull Mode", api.CreateBitfieldDataValue("VkCullModeFlags", rasterState.CullMode(), VkCullModeFlagBitsConstants(), API{}))
	rasterList = rasterList.AppendKeyValuePair("Front Face", api.CreateEnumDataValue("VkFrontFace", rasterState.FrontFace()))
	rasterList = rasterList.AppendKeyValuePair("Depth Bias Enabled", api.CreatePoDDataValue("VkBool32", rasterState.DepthBiasEnable() != 0))
	rasterList = rasterList.AppendKeyValuePair("Depth Bias Constant Factor", api.CreatePoDDataValue("f32", rasterState.DepthBiasConstantFactor()))
	rasterList = rasterList.AppendKeyValuePair("Depth Bias Clamp", api.CreatePoDDataValue("f32", rasterState.DepthBiasClamp()))
	rasterList = rasterList.AppendKeyValuePair("Depth Bias Slope Factor", api.CreatePoDDataValue("f32", rasterState.DepthBiasSlopeFactor()))
	rasterList = rasterList.AppendKeyValuePair("Line Width", api.CreatePoDDataValue("f32", rasterState.LineWidth()))

	multiState := p.MultisampleState()
	multiList := &api.KeyValuePairList{}
	multiList = multiList.AppendKeyValuePair("Sample Count", api.CreateBitfieldDataValue("VkSampleCountFlagBits", multiState.RasterizationSamples(), VkSampleCountFlagBitsConstants(), API{}))

	// // For now, only display the first element of the sample mask array. There's rarely more in practice.
	mask := uint64(0xFFFFFFFF)
	if multiState.SampleMask().Len() > 0 {
		mask = uint64(multiState.SampleMask().Get(multiState.SampleMask().Keys()[0]))
	}
	multiList = multiList.AppendKeyValuePair("Sample Mask", api.CreatePoDDataValue("VkSampleMask", fmt.Sprintf("%X", mask)))

	multiList = multiList.AppendKeyValuePair("Sample Shading Enabled", api.CreatePoDDataValue("VkBool32", multiState.SampleShadingEnable() != 0))
	multiList = multiList.AppendKeyValuePair("Min Sample Shading", api.CreatePoDDataValue("f32", multiState.MinSampleShading()))
	multiList = multiList.AppendKeyValuePair("Alpha to Coverage", api.CreatePoDDataValue("VkBool32", multiState.AlphaToCoverageEnable() != 0))
	multiList = multiList.AppendKeyValuePair("Alpha to One", api.CreatePoDDataValue("VkBool32", multiState.AlphaToOneEnable() != 0))

	viewports := p.ViewportState().Viewports()
	viewportRows := make([]*api.Row, viewports.Len())
	for i, index := range viewports.Keys() {
		viewport := viewports.Get(index)

		viewportRows[i] = &api.Row{
			RowValues: []*api.DataValue{
				api.CreatePoDDataValue("f32", viewport.X()),
				api.CreatePoDDataValue("f32", viewport.Y()),
				api.CreatePoDDataValue("f32", viewport.Width()),
				api.CreatePoDDataValue("f32", viewport.Height()),
				api.CreatePoDDataValue("f32", viewport.MinDepth()),
				api.CreatePoDDataValue("f32", viewport.MaxDepth()),
			},
		}
	}

	viewportTable := &api.Table{
		Headers: []string{"X", "Y", "Width", "Height", "Min Depth", "Max Depth"},
		Rows:    viewportRows,
	}

	scissors := p.ViewportState().Scissors()
	scissorRows := make([]*api.Row, scissors.Len())
	for i, index := range scissors.Keys() {
		scissor := scissors.Get(index)

		scissorRows[i] = &api.Row{
			RowValues: []*api.DataValue{
				api.CreatePoDDataValue("s32", scissor.Offset().X()),
				api.CreatePoDDataValue("s32", scissor.Offset().Y()),
				api.CreatePoDDataValue("u32", scissor.Extent().Width()),
				api.CreatePoDDataValue("u32", scissor.Extent().Height()),
			},
		}
	}

	scissorTable := &api.Table{
		Headers: []string{"X", "Y", "Width", "Height"},
		Rows:    scissorRows,
	}

	dataGroups := []*api.DataGroup{
		&api.DataGroup{
			GroupName: "Rasterization State",
			Data:      &api.DataGroup_KeyValues{rasterList},
		},

		&api.DataGroup{
			GroupName: "Multisample State",
			Data:      &api.DataGroup_KeyValues{multiList},
		},

		&api.DataGroup{
			GroupName: "Viewports",
			Data:      &api.DataGroup_Table{viewportTable},
		},

		&api.DataGroup{
			GroupName: "Scissors",
			Data:      &api.DataGroup_Table{scissorTable},
		},
	}

	return &api.Stage{
		StageName: "Rasterizer",
		DebugName: "RAST",
		Enabled:   true,
		Groups:    dataGroups,
	}
}

func (p GraphicsPipelineObjectʳ) fragmentShader(ctx context.Context, s *api.GlobalState) *api.Stage {
	stages := p.Stages()
	for _, index := range stages.Keys() {
		stage := stages.Get(index)

		if stage.Stage() == VkShaderStageFlagBits_VK_SHADER_STAGE_FRAGMENT_BIT {
			stage := stages.Get(index)

			module := stage.Module()

			words, _ := module.Words().Read(ctx, nil, s, nil)
			source := shadertools.DisassembleSpirvBinary(words)
			shader := &api.Shader{Type: api.ShaderType_Spirv, Source: source}

			dataGroups := []*api.DataGroup{
				&api.DataGroup{
					GroupName: "Shader Code",
					Data:      &api.DataGroup_Shader{shader},
				},
			}

			return &api.Stage{
				StageName: "Fragment Shader",
				DebugName: "FS",
				Enabled:   true,
				Groups:    dataGroups,
			}
		}
	}

	return &api.Stage{
		StageName: "Fragment Shader",
		DebugName: "FS",
		Enabled:   false,
	}
}

func (p GraphicsPipelineObjectʳ) colorBlending() *api.Stage {
	depthData := p.DepthState()
	depthList := &api.KeyValuePairList{}
	depthList = depthList.AppendKeyValuePair("Test Enabled", api.CreatePoDDataValue("VkBool32", depthData.DepthTestEnable() != 0))
	depthList = depthList.AppendKeyValuePair("Write Enabled", api.CreatePoDDataValue("VkBool32", depthData.DepthWriteEnable() != 0))
	depthList = depthList.AppendKeyValuePair("Function", api.CreateEnumDataValue("VkCompareOp", depthData.DepthCompareOp()))
	depthList = depthList.AppendKeyValuePair("Bounds Test Enabled", api.CreatePoDDataValue("VkBool32", depthData.DepthBoundsTestEnable() != 0))

	stencilRows := []*api.Row{}

	if depthData.StencilTestEnable() != 0 {
		frontStencil := depthData.Front()

		stencilRows = append(stencilRows, &api.Row{
			RowValues: []*api.DataValue{
				api.CreatePoDDataValue("string", "Front"),
				api.CreateEnumDataValue("VkStencilOp", frontStencil.FailOp()),
				api.CreateEnumDataValue("VkStencilOp", frontStencil.PassOp()),
				api.CreateEnumDataValue("VkStencilOp", frontStencil.DepthFailOp()),
				api.CreateEnumDataValue("VkCompareOp", frontStencil.CompareOp()),
				api.CreatePoDDataValue("uint32_t", fmt.Sprintf("%X", frontStencil.CompareMask())),
				api.CreatePoDDataValue("uint32_t", fmt.Sprintf("%X", frontStencil.WriteMask())),
				api.CreatePoDDataValue("uint32_t", frontStencil.Reference()),
			},
		})

		backStencil := depthData.Back()

		stencilRows = append(stencilRows, &api.Row{
			RowValues: []*api.DataValue{
				api.CreatePoDDataValue("string", "Back"),
				api.CreateEnumDataValue("VkStencilOp", backStencil.FailOp()),
				api.CreateEnumDataValue("VkStencilOp", backStencil.PassOp()),
				api.CreateEnumDataValue("VkStencilOp", backStencil.DepthFailOp()),
				api.CreateEnumDataValue("VkCompareOp", backStencil.CompareOp()),
				api.CreatePoDDataValue("uint32_t", fmt.Sprintf("%X", backStencil.CompareMask())),
				api.CreatePoDDataValue("uint32_t", fmt.Sprintf("%X", backStencil.WriteMask())),
				api.CreatePoDDataValue("uint32_t", backStencil.Reference()),
			},
		})

	}

	stencilTable := &api.Table{
		Headers: []string{"Face", "Fail Op", "Pass Op", "Depth Fail Op", "Func", "Compare Mask", "Write Mask", "Ref"},
		Rows:    stencilRows,
	}

	blendData := p.ColorBlendState()
	blendList := &api.KeyValuePairList{}
	blendList = blendList.AppendKeyValuePair("Logic Op Enabled", api.CreatePoDDataValue("VkBool32", blendData.LogicOpEnable() != 0))
	blendList = blendList.AppendKeyValuePair("Logic Op", api.CreateEnumDataValue("VkLogicOp", blendData.LogicOp()))
	blendList = blendList.AppendKeyValuePair("Blend Constants", api.CreatePoDDataValue("float[4]", blendData.BlendConstants().GetArrayValues()))

	targets := blendData.Attachments()
	targetRows := make([]*api.Row, targets.Len())
	for i, index := range targets.Keys() {
		target := targets.Get(index)

		targetRows[i] = &api.Row{
			RowValues: []*api.DataValue{
				api.CreatePoDDataValue("VkBool32", target.BlendEnable() != 0),
				api.CreateEnumDataValue("VkBlendFactor", target.SrcColorBlendFactor()),
				api.CreateEnumDataValue("VkBlendFactor", target.DstColorBlendFactor()),
				api.CreateEnumDataValue("VkBlendOp", target.ColorBlendOp()),
				api.CreateEnumDataValue("VkBlendFactor", target.SrcAlphaBlendFactor()),
				api.CreateEnumDataValue("VkBlendFactor", target.DstAlphaBlendFactor()),
				api.CreateEnumDataValue("VkBlendOp", target.AlphaBlendOp()),
				api.CreateBitfieldDataValue("VkColorComponentFlagBits", target.ColorWriteMask(), VkColorComponentFlagBitsConstants(), API{}),
			},
		}
	}

	targetTable := &api.Table{
		Headers: []string{"Enabled", "Color Src", "Color Dst", "Color Op", "Alpha Src", "Alpha Dst", "Alpha Op", "Color Write Mask"},
		Rows:    targetRows,
	}

	renderAttachments := p.RenderPass().AttachmentDescriptions()
	attachmentRows := make([]*api.Row, renderAttachments.Len())
	for i, index := range renderAttachments.Keys() {
		attach := renderAttachments.Get(index)

		attachmentRows[i] = &api.Row{
			RowValues: []*api.DataValue{
				api.CreateEnumDataValue("VkFormat", attach.Fmt()),
				api.CreateBitfieldDataValue("VkSampleCountFlagBits", attach.Samples(), VkSampleCountFlagBitsConstants(), API{}),
				api.CreateEnumDataValue("VkAttachmentLoadOp", attach.LoadOp()),
				api.CreateEnumDataValue("VkAttachmentStoreOp", attach.StoreOp()),
				api.CreateEnumDataValue("VkAttachmentLoadOp", attach.StencilLoadOp()),
				api.CreateEnumDataValue("VkAttachmentStoreOp", attach.StencilStoreOp()),
				api.CreateEnumDataValue("VkImageLayout", attach.InitialLayout()),
				api.CreateEnumDataValue("VkImageLayout", attach.FinalLayout()),
			},
		}
	}

	attachTable := &api.Table{
		Headers: []string{"Format", "Samples", "Load Op", "Store Op", "Stencil Load Op", "Stencil Store Op", "Init Layout", "Final Layout"},
		Rows:    attachmentRows,
	}

	dataGroups := []*api.DataGroup{
		&api.DataGroup{
			GroupName: "Render Pass Attachments",
			Data:      &api.DataGroup_Table{attachTable},
		},

		&api.DataGroup{
			GroupName: "Target Blends",
			Data:      &api.DataGroup_Table{targetTable},
		},

		&api.DataGroup{
			GroupName: "Blend State",
			Data:      &api.DataGroup_KeyValues{blendList},
		},

		&api.DataGroup{
			GroupName: "Depth State",
			Data:      &api.DataGroup_KeyValues{depthList},
		},

		&api.DataGroup{
			GroupName: "Stencil State",
			Data:      &api.DataGroup_Table{stencilTable},
		},
	}

	return &api.Stage{
		StageName: "Color Blending",
		DebugName: "BLEND",
		Enabled:   true,
		Groups:    dataGroups,
	}
}

// SetResourceData sets resource data in a new capture.
func (p GraphicsPipelineObjectʳ) SetResourceData(
	context.Context,
	*path.Command,
	*api.ResourceData,
	api.ResourceMap,
	api.ReplaceCallback,
	api.MutateInitialState,
	*path.ResolveConfig) error {
	return fmt.Errorf("SetResourceData is not supported on GraphicsPipeline")
}

var _ api.Resource = ComputePipelineObjectʳ{}

func (p ComputePipelineObjectʳ) IsResource() bool {
	return true
}

// ResourceHandle returns the UI identity for the resource.
func (p ComputePipelineObjectʳ) ResourceHandle() string {
	return fmt.Sprintf("ComputePipeline<%d>", p.VulkanHandle())
}

// ResourceLabel returns an optional debug label for the resource.
func (p ComputePipelineObjectʳ) ResourceLabel() string {
	if p.DebugInfo().IsNil() {
		return ""
	}
	if p.DebugInfo().ObjectName() != "" {
		return p.DebugInfo().ObjectName()
	}
	return fmt.Sprintf("<%d:%v>", p.DebugInfo().TagName(), p.DebugInfo().Tag())
}

// Order returns an integer used to sort the resources for presentation.
func (p ComputePipelineObjectʳ) Order() uint64 {
	return uint64(p.VulkanHandle())
}

// ResourceType returns the type of this resource.
func (p ComputePipelineObjectʳ) ResourceType(ctx context.Context) api.ResourceType {
	return api.ResourceType_PipelineResource
}

// ResourceData returns the resource data given the current state.
func (p ComputePipelineObjectʳ) ResourceData(ctx context.Context, s *api.GlobalState) (*api.ResourceData, error) {
	vkState := GetState(s)
	isBound := false
	// Use LastComputeInfos to get bound descriptor set data.
	// TODO: Ideally we could look at just a specific pipeline/descriptor
	// set pair.  Maybe we could modify mutate to track which what
	// descriptor sets were bound to particular pipelines.
	if !vkState.LastBoundQueue().IsNil() {
		lci, ok := vkState.LastComputeInfos().Lookup(vkState.LastBoundQueue().VulkanHandle())
		if ok {
			if lci.ComputePipeline() == p {
				isBound = true
			}
		}
	}

	stage := p.Stage()
	module := stage.Module()

	words, _ := module.Words().Read(ctx, nil, s, nil)
	source := shadertools.DisassembleSpirvBinary(words)
	shader := &api.Shader{Type: api.ShaderType_Spirv, Source: source}

	stages := []*api.Stage{
		&api.Stage{
			StageName: "Compute Shader",
			DebugName: "CS",
			Enabled:   true,
			Groups: []*api.DataGroup{
				&api.DataGroup{
					GroupName: "Shader Code",
					Data:      &api.DataGroup_Shader{shader},
				},
			},
		},
	}

	return &api.ResourceData{
		Data: &api.ResourceData_Pipeline{
			Pipeline: &api.Pipeline{
				API:          path.NewAPI(id.ID(ID)),
				PipelineType: api.Pipeline_COMPUTE,
				DebugName:    "COMP",
				Stages:       stages,
				Bound:        isBound,
			},
		},
	}, nil
}

// SetResourceData sets resource data in a new capture.
func (p ComputePipelineObjectʳ) SetResourceData(
	context.Context,
	*path.Command,
	*api.ResourceData,
	api.ResourceMap,
	api.ReplaceCallback,
	api.MutateInitialState,
	*path.ResolveConfig) error {
	return fmt.Errorf("SetResourceData is not supported on ComputePipeline")
}

func stageType(vkStage VkShaderStageFlagBits) (string, error) {
	switch vkStage {
	case VkShaderStageFlagBits_VK_SHADER_STAGE_VERTEX_BIT:
		return "Vertex", nil
	case VkShaderStageFlagBits_VK_SHADER_STAGE_TESSELLATION_CONTROL_BIT:
		return "Tessellation Control", nil
	case VkShaderStageFlagBits_VK_SHADER_STAGE_TESSELLATION_EVALUATION_BIT:
		return "Tessellation Evaluation", nil
	case VkShaderStageFlagBits_VK_SHADER_STAGE_GEOMETRY_BIT:
		return "Geometry", nil
	case VkShaderStageFlagBits_VK_SHADER_STAGE_FRAGMENT_BIT:
		return "Fragment", nil
	case VkShaderStageFlagBits_VK_SHADER_STAGE_COMPUTE_BIT:
		return "Compute", nil
	default:
		return "", fmt.Errorf("Invalid Vulkan stage: %v", vkStage)
	}
}

func typeMatch(sourceType uint32, matchType uint32) bool {
	// Some pipeline layout descriptor types can match multiple SPIR-V
	// types, the types we get from shadertools.ParseDescriptorSets don't
	// have to match exactly in all cases.
	source := VkDescriptorType(sourceType)
	match := VkDescriptorType(matchType)

	valid := []VkDescriptorType{match}
	switch source {
	case VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLER,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE:
		valid = append(valid, VkDescriptorType_VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER)
	case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER:
		valid = append(valid, VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC)
	case VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
		valid = append(valid, VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC)
	}

	for _, val := range valid {
		if match == val {
			return true
		}
	}
	return false
}

// Bindings are bound by (set, binding) pairs, and are unique for a pipeline layout
type bindIdx struct {
	set     uint32
	binding uint32
}

func (a *bindIdx) Less(b *bindIdx) bool {
	if a.set != b.set {
		return a.set < b.set
	}
	return a.binding < b.binding
}
