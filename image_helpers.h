#pragma once

/*
 * Copyright (C) 2022 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include "common.h"

namespace gapid2 {

struct texel_block_size {
  uint32_t width;
  uint32_t height;
};

struct element_and_block_size {
  uint32_t element_size;
  texel_block_size texel_block_size;
};

inline uint32_t
get_depth_element_size(VkFormat format, bool inBuffer) {
  const uint32_t d16_size = 2;
  const uint32_t d24_size = 3;
  const uint32_t d32_size = 4;

  switch (format) {
    case VK_FORMAT_D16_UNORM:
    case VK_FORMAT_D16_UNORM_S8_UINT:
      return d16_size;
    case VK_FORMAT_D32_SFLOAT:
    case VK_FORMAT_D32_SFLOAT_S8_UINT:
      return d32_size;
    case VK_FORMAT_X8_D24_UNORM_PACK32:
    case VK_FORMAT_D24_UNORM_S8_UINT:
      if (inBuffer) {
        return d32_size;
      } else {
        return d24_size;
      }
    default:
      return 0;
  }
}

// This should roughly correspond to "Compatible formats" in the Vulkan spec
inline element_and_block_size get_element_and_block_size(VkFormat format) {
  switch (format) {
    case VK_FORMAT_R4G4_UNORM_PACK8:
    case VK_FORMAT_R8_UNORM:
    case VK_FORMAT_R8_SNORM:
    case VK_FORMAT_R8_USCALED:
    case VK_FORMAT_R8_SSCALED:
    case VK_FORMAT_R8_UINT:
    case VK_FORMAT_R8_SINT:
    case VK_FORMAT_R8_SRGB:
      return element_and_block_size{1, texel_block_size{1, 1}};
    case VK_FORMAT_R4G4B4A4_UNORM_PACK16:
    case VK_FORMAT_B4G4R4A4_UNORM_PACK16:
    case VK_FORMAT_R5G6B5_UNORM_PACK16:
    case VK_FORMAT_B5G6R5_UNORM_PACK16:
    case VK_FORMAT_R5G5B5A1_UNORM_PACK16:
    case VK_FORMAT_B5G5R5A1_UNORM_PACK16:
    case VK_FORMAT_A1R5G5B5_UNORM_PACK16:
    case VK_FORMAT_R8G8_UNORM:
    case VK_FORMAT_R8G8_SNORM:
    case VK_FORMAT_R8G8_USCALED:
    case VK_FORMAT_R8G8_SSCALED:
    case VK_FORMAT_R8G8_UINT:
    case VK_FORMAT_R8G8_SINT:
    case VK_FORMAT_R8G8_SRGB:
    case VK_FORMAT_R16_UNORM:
    case VK_FORMAT_R16_SNORM:
    case VK_FORMAT_R16_USCALED:
    case VK_FORMAT_R16_SSCALED:
    case VK_FORMAT_R16_UINT:
    case VK_FORMAT_R16_SINT:
    case VK_FORMAT_R16_SFLOAT:
      return element_and_block_size{
          2, texel_block_size{1, 1}};
    case VK_FORMAT_R8G8B8_UNORM:
    case VK_FORMAT_R8G8B8_SNORM:
    case VK_FORMAT_R8G8B8_USCALED:
    case VK_FORMAT_R8G8B8_SSCALED:
    case VK_FORMAT_R8G8B8_UINT:
    case VK_FORMAT_R8G8B8_SINT:
    case VK_FORMAT_R8G8B8_SRGB:
    case VK_FORMAT_B8G8R8_UNORM:
    case VK_FORMAT_B8G8R8_SNORM:
    case VK_FORMAT_B8G8R8_USCALED:
    case VK_FORMAT_B8G8R8_SSCALED:
    case VK_FORMAT_B8G8R8_UINT:
    case VK_FORMAT_B8G8R8_SINT:
    case VK_FORMAT_B8G8R8_SRGB:
      return element_and_block_size{
          3, texel_block_size{1, 1}};
    case VK_FORMAT_R8G8B8A8_UNORM:
    case VK_FORMAT_R8G8B8A8_SNORM:
    case VK_FORMAT_R8G8B8A8_USCALED:
    case VK_FORMAT_R8G8B8A8_SSCALED:
    case VK_FORMAT_R8G8B8A8_UINT:
    case VK_FORMAT_R8G8B8A8_SINT:
    case VK_FORMAT_R8G8B8A8_SRGB:
    case VK_FORMAT_B8G8R8A8_UNORM:
    case VK_FORMAT_B8G8R8A8_SNORM:
    case VK_FORMAT_B8G8R8A8_USCALED:
    case VK_FORMAT_B8G8R8A8_SSCALED:
    case VK_FORMAT_B8G8R8A8_UINT:
    case VK_FORMAT_B8G8R8A8_SINT:
    case VK_FORMAT_B8G8R8A8_SRGB:
    case VK_FORMAT_A8B8G8R8_UNORM_PACK32:
    case VK_FORMAT_A8B8G8R8_SNORM_PACK32:
    case VK_FORMAT_A8B8G8R8_USCALED_PACK32:
    case VK_FORMAT_A8B8G8R8_SSCALED_PACK32:
    case VK_FORMAT_A8B8G8R8_UINT_PACK32:
    case VK_FORMAT_A8B8G8R8_SINT_PACK32:
    case VK_FORMAT_A8B8G8R8_SRGB_PACK32:
    case VK_FORMAT_A2R10G10B10_UNORM_PACK32:
    case VK_FORMAT_A2R10G10B10_SNORM_PACK32:
    case VK_FORMAT_A2R10G10B10_USCALED_PACK32:
    case VK_FORMAT_A2R10G10B10_SSCALED_PACK32:
    case VK_FORMAT_A2R10G10B10_UINT_PACK32:
    case VK_FORMAT_A2R10G10B10_SINT_PACK32:
    case VK_FORMAT_A2B10G10R10_UNORM_PACK32:
    case VK_FORMAT_A2B10G10R10_SNORM_PACK32:
    case VK_FORMAT_A2B10G10R10_USCALED_PACK32:
    case VK_FORMAT_A2B10G10R10_SSCALED_PACK32:
    case VK_FORMAT_A2B10G10R10_UINT_PACK32:
    case VK_FORMAT_A2B10G10R10_SINT_PACK32:
    case VK_FORMAT_R16G16_UNORM:
    case VK_FORMAT_R16G16_SNORM:
    case VK_FORMAT_R16G16_USCALED:
    case VK_FORMAT_R16G16_SSCALED:
    case VK_FORMAT_R16G16_UINT:
    case VK_FORMAT_R16G16_SINT:
    case VK_FORMAT_R16G16_SFLOAT:
    case VK_FORMAT_R32_UINT:
    case VK_FORMAT_R32_SINT:
    case VK_FORMAT_R32_SFLOAT:
    case VK_FORMAT_B10G11R11_UFLOAT_PACK32:
    case VK_FORMAT_E5B9G9R9_UFLOAT_PACK32:
      return element_and_block_size{
          4, texel_block_size{1, 1}};
    case VK_FORMAT_R16G16B16_UNORM:
    case VK_FORMAT_R16G16B16_SNORM:
    case VK_FORMAT_R16G16B16_USCALED:
    case VK_FORMAT_R16G16B16_SSCALED:
    case VK_FORMAT_R16G16B16_UINT:
    case VK_FORMAT_R16G16B16_SINT:
    case VK_FORMAT_R16G16B16_SFLOAT:
      return element_and_block_size{
          6, texel_block_size{1, 1}};
    case VK_FORMAT_R16G16B16A16_UNORM:
    case VK_FORMAT_R16G16B16A16_SNORM:
    case VK_FORMAT_R16G16B16A16_USCALED:
    case VK_FORMAT_R16G16B16A16_SSCALED:
    case VK_FORMAT_R16G16B16A16_UINT:
    case VK_FORMAT_R16G16B16A16_SINT:
    case VK_FORMAT_R16G16B16A16_SFLOAT:
    case VK_FORMAT_R32G32_UINT:
    case VK_FORMAT_R32G32_SINT:
    case VK_FORMAT_R32G32_SFLOAT:
    case VK_FORMAT_R64_UINT:
    case VK_FORMAT_R64_SINT:
    case VK_FORMAT_R64_SFLOAT:
      return element_and_block_size{
          8, texel_block_size{1, 1}};
    case VK_FORMAT_R32G32B32_UINT:
    case VK_FORMAT_R32G32B32_SINT:
    case VK_FORMAT_R32G32B32_SFLOAT:
      return element_and_block_size{
          12, texel_block_size{1, 1}};
    case VK_FORMAT_R32G32B32A32_UINT:
    case VK_FORMAT_R32G32B32A32_SINT:
    case VK_FORMAT_R32G32B32A32_SFLOAT:
    case VK_FORMAT_R64G64_UINT:
    case VK_FORMAT_R64G64_SINT:
    case VK_FORMAT_R64G64_SFLOAT:
      return element_and_block_size{
          16, texel_block_size{1, 1}};
    case VK_FORMAT_R64G64B64_UINT:
    case VK_FORMAT_R64G64B64_SINT:
    case VK_FORMAT_R64G64B64_SFLOAT:
      return element_and_block_size{
          24, texel_block_size{1, 1}};
    case VK_FORMAT_R64G64B64A64_UINT:
    case VK_FORMAT_R64G64B64A64_SINT:
    case VK_FORMAT_R64G64B64A64_SFLOAT:
      return element_and_block_size{
          32, texel_block_size{1, 1}};
    case VK_FORMAT_BC1_RGB_UNORM_BLOCK:
    case VK_FORMAT_BC1_RGB_SRGB_BLOCK:
    case VK_FORMAT_BC1_RGBA_UNORM_BLOCK:
    case VK_FORMAT_BC1_RGBA_SRGB_BLOCK:
      return element_and_block_size{
          8, texel_block_size{4, 4}};
    case VK_FORMAT_BC2_UNORM_BLOCK:
    case VK_FORMAT_BC2_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{4, 4}};
    case VK_FORMAT_BC3_UNORM_BLOCK:
    case VK_FORMAT_BC3_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{4, 4}};
    case VK_FORMAT_BC4_UNORM_BLOCK:
    case VK_FORMAT_BC4_SNORM_BLOCK:
      return element_and_block_size{
          8, texel_block_size{4, 4}};
    case VK_FORMAT_BC5_UNORM_BLOCK:
    case VK_FORMAT_BC5_SNORM_BLOCK:
      return element_and_block_size{
          16, texel_block_size{4, 4}};
    case VK_FORMAT_BC6H_UFLOAT_BLOCK:
    case VK_FORMAT_BC6H_SFLOAT_BLOCK:
      return element_and_block_size{
          16, texel_block_size{4, 4}};
    case VK_FORMAT_BC7_UNORM_BLOCK:
    case VK_FORMAT_BC7_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{4, 4}};
    case VK_FORMAT_ETC2_R8G8B8_UNORM_BLOCK:
    case VK_FORMAT_ETC2_R8G8B8_SRGB_BLOCK:
    case VK_FORMAT_ETC2_R8G8B8A1_UNORM_BLOCK:
    case VK_FORMAT_ETC2_R8G8B8A1_SRGB_BLOCK:
      return element_and_block_size{
          8, texel_block_size{4, 4}};
    case VK_FORMAT_ETC2_R8G8B8A8_UNORM_BLOCK:
    case VK_FORMAT_ETC2_R8G8B8A8_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{4, 4}};
    case VK_FORMAT_EAC_R11_UNORM_BLOCK:
    case VK_FORMAT_EAC_R11_SNORM_BLOCK:
      return element_and_block_size{
          8, texel_block_size{4, 4}};
    case VK_FORMAT_EAC_R11G11_UNORM_BLOCK:
    case VK_FORMAT_EAC_R11G11_SNORM_BLOCK:
      return element_and_block_size{
          16, texel_block_size{4, 4}};
    case VK_FORMAT_ASTC_4x4_UNORM_BLOCK:
    case VK_FORMAT_ASTC_4x4_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{4, 4}};
    case VK_FORMAT_ASTC_5x4_UNORM_BLOCK:
    case VK_FORMAT_ASTC_5x4_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{5, 4}};
    case VK_FORMAT_ASTC_5x5_UNORM_BLOCK:
    case VK_FORMAT_ASTC_5x5_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{5, 5}};
    case VK_FORMAT_ASTC_6x5_UNORM_BLOCK:
    case VK_FORMAT_ASTC_6x5_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{6, 5}};
    case VK_FORMAT_ASTC_6x6_UNORM_BLOCK:
    case VK_FORMAT_ASTC_6x6_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{6, 6}};
    case VK_FORMAT_ASTC_8x5_UNORM_BLOCK:
    case VK_FORMAT_ASTC_8x5_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{8, 5}};
    case VK_FORMAT_ASTC_8x6_UNORM_BLOCK:
    case VK_FORMAT_ASTC_8x6_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{8, 6}};
    case VK_FORMAT_ASTC_8x8_UNORM_BLOCK:
    case VK_FORMAT_ASTC_8x8_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{8, 8}};
    case VK_FORMAT_ASTC_10x5_UNORM_BLOCK:
    case VK_FORMAT_ASTC_10x5_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{10, 5}};
    case VK_FORMAT_ASTC_10x6_UNORM_BLOCK:
    case VK_FORMAT_ASTC_10x6_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{10, 6}};
    case VK_FORMAT_ASTC_10x8_UNORM_BLOCK:
    case VK_FORMAT_ASTC_10x8_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{10, 8}};
    case VK_FORMAT_ASTC_10x10_UNORM_BLOCK:
    case VK_FORMAT_ASTC_10x10_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{10, 10}};
    case VK_FORMAT_ASTC_12x10_UNORM_BLOCK:
    case VK_FORMAT_ASTC_12x10_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{12, 10}};
    case VK_FORMAT_ASTC_12x12_UNORM_BLOCK:
    case VK_FORMAT_ASTC_12x12_SRGB_BLOCK:
      return element_and_block_size{
          16, texel_block_size{12, 12}};
    case VK_FORMAT_D16_UNORM:
      return element_and_block_size{
          2, texel_block_size{1, 1}};
    case VK_FORMAT_X8_D24_UNORM_PACK32:
      return element_and_block_size{
          4, texel_block_size{1, 1}};
    case VK_FORMAT_D32_SFLOAT:
      return element_and_block_size{
          4, texel_block_size{1, 1}};
    case VK_FORMAT_S8_UINT:
      return element_and_block_size{
          1, texel_block_size{1, 1}};
    case VK_FORMAT_D16_UNORM_S8_UINT:
      return element_and_block_size{
          3, texel_block_size{1, 1}};
    case VK_FORMAT_D24_UNORM_S8_UINT:
      return element_and_block_size{
          3, texel_block_size{1, 1}};
    case VK_FORMAT_D32_SFLOAT_S8_UINT:
      return element_and_block_size{
          5, texel_block_size{1, 1}};
    case VK_FORMAT_G8B8G8R8_422_UNORM:
    case VK_FORMAT_B8G8R8G8_422_UNORM:
      return element_and_block_size{
          4, texel_block_size{1, 1}};
    case VK_FORMAT_R10X6_UNORM_PACK16:
      return element_and_block_size{
          2, texel_block_size{1, 1}};
    case VK_FORMAT_R10X6G10X6_UNORM_2PACK16:
      return element_and_block_size{
          4, texel_block_size{1, 1}};
    case VK_FORMAT_R10X6G10X6B10X6A10X6_UNORM_4PACK16:
    case VK_FORMAT_G10X6B10X6G10X6R10X6_422_UNORM_4PACK16:
    case VK_FORMAT_B10X6G10X6R10X6G10X6_422_UNORM_4PACK16:
      return element_and_block_size{
          8, texel_block_size{1, 1}};
    case VK_FORMAT_R12X4_UNORM_PACK16:
    case VK_FORMAT_R12X4G12X4_UNORM_2PACK16:
      return element_and_block_size{
          2, texel_block_size{1, 1}};
    case VK_FORMAT_G16B16G16R16_422_UNORM:
    case VK_FORMAT_B16G16R16G16_422_UNORM:
      return element_and_block_size{
          8, texel_block_size{1, 1}};
    case VK_FORMAT_R12X4G12X4B12X4A12X4_UNORM_4PACK16:
    case VK_FORMAT_G12X4B12X4G12X4R12X4_422_UNORM_4PACK16:
    case VK_FORMAT_B12X4G12X4R12X4G12X4_422_UNORM_4PACK16:
      return element_and_block_size{
          8, texel_block_size{1, 1}};
    default:
      GAPID2_ERROR("Unhandled texture format");
      return element_and_block_size();
  }
}

inline VkImageAspectFlags get_aspects(VkFormat format) {
  switch (format) {
    case VK_FORMAT_D16_UNORM:
    case VK_FORMAT_D32_SFLOAT:
    case VK_FORMAT_X8_D24_UNORM_PACK32:
      return static_cast<VkImageAspectFlags>(VK_IMAGE_ASPECT_DEPTH_BIT);
    case VK_FORMAT_S8_UINT:
    case VK_FORMAT_D16_UNORM_S8_UINT:
    case VK_FORMAT_D24_UNORM_S8_UINT:
    case VK_FORMAT_D32_SFLOAT_S8_UINT:
      return static_cast<VkImageAspectFlags>(VK_IMAGE_ASPECT_DEPTH_BIT |
                                             VK_IMAGE_ASPECT_STENCIL_BIT);
    case VK_FORMAT_G8_B8R8_2PLANE_420_UNORM:
    case VK_FORMAT_G8_B8R8_2PLANE_422_UNORM:
    case VK_FORMAT_G10X6_B10X6R10X6_2PLANE_420_UNORM_3PACK16:
    case VK_FORMAT_G10X6_B10X6R10X6_2PLANE_422_UNORM_3PACK16:
    case VK_FORMAT_G12X4_B12X4R12X4_2PLANE_420_UNORM_3PACK16:
    case VK_FORMAT_G12X4_B12X4R12X4_2PLANE_422_UNORM_3PACK16:
    case VK_FORMAT_G16_B16R16_2PLANE_420_UNORM:
    case VK_FORMAT_G16_B16R16_2PLANE_422_UNORM:
      return static_cast<VkImageAspectFlags>(
          VK_IMAGE_ASPECT_PLANE_0_BIT |
          VK_IMAGE_ASPECT_PLANE_1_BIT);
    case VK_FORMAT_G8_B8_R8_3PLANE_420_UNORM:
    case VK_FORMAT_G8_B8_R8_3PLANE_422_UNORM:
    case VK_FORMAT_G8_B8_R8_3PLANE_444_UNORM:
    case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_420_UNORM_3PACK16:
    case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_422_UNORM_3PACK16:
    case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_444_UNORM_3PACK16:
    case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_420_UNORM_3PACK16:
    case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_422_UNORM_3PACK16:
    case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_444_UNORM_3PACK16:
    case VK_FORMAT_G16_B16_R16_3PLANE_420_UNORM:
    case VK_FORMAT_G16_B16_R16_3PLANE_422_UNORM:
    case VK_FORMAT_G16_B16_R16_3PLANE_444_UNORM:
      return static_cast<VkImageAspectFlags>(
          VK_IMAGE_ASPECT_PLANE_0_BIT |
          VK_IMAGE_ASPECT_PLANE_1_BIT |
          VK_IMAGE_ASPECT_PLANE_2_BIT);
    default:
      return static_cast<VkImageAspectFlags>(VK_IMAGE_ASPECT_COLOR_BIT);
  }
}

inline VkImageAspectFlags is_multi_planar_color(VkFormat format) {
  switch (format) {
    case VK_FORMAT_G8_B8R8_2PLANE_420_UNORM:
    case VK_FORMAT_G8_B8R8_2PLANE_422_UNORM:
    case VK_FORMAT_G10X6_B10X6R10X6_2PLANE_420_UNORM_3PACK16:
    case VK_FORMAT_G10X6_B10X6R10X6_2PLANE_422_UNORM_3PACK16:
    case VK_FORMAT_G12X4_B12X4R12X4_2PLANE_420_UNORM_3PACK16:
    case VK_FORMAT_G12X4_B12X4R12X4_2PLANE_422_UNORM_3PACK16:
    case VK_FORMAT_G16_B16R16_2PLANE_420_UNORM:
    case VK_FORMAT_G16_B16R16_2PLANE_422_UNORM:
    case VK_FORMAT_G8_B8_R8_3PLANE_420_UNORM:
    case VK_FORMAT_G8_B8_R8_3PLANE_422_UNORM:
    case VK_FORMAT_G8_B8_R8_3PLANE_444_UNORM:
    case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_420_UNORM_3PACK16:
    case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_422_UNORM_3PACK16:
    case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_444_UNORM_3PACK16:
    case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_420_UNORM_3PACK16:
    case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_422_UNORM_3PACK16:
    case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_444_UNORM_3PACK16:
    case VK_FORMAT_G16_B16_R16_3PLANE_420_UNORM:
    case VK_FORMAT_G16_B16_R16_3PLANE_422_UNORM:
    case VK_FORMAT_G16_B16_R16_3PLANE_444_UNORM:
      return true;
    default:
      return false;
  }
}

inline element_and_block_size get_element_and_block_size_for_aspect(VkFormat format, VkImageAspectFlagBits aspect) {
  auto original_aspect = get_element_and_block_size(format);
  auto des = get_depth_element_size(format, false);
  switch (aspect) {
    default:
      GAPID2_ERROR("Unknown image aspect");
      return original_aspect;
    case VK_IMAGE_ASPECT_COLOR_BIT:
      return original_aspect;
    case VK_IMAGE_ASPECT_DEPTH_BIT:
      return element_and_block_size{des, texel_block_size{1, 1}};
    case VK_IMAGE_ASPECT_STENCIL_BIT:
      return element_and_block_size{1, texel_block_size{1, 1}};
    case VK_IMAGE_ASPECT_PLANE_0_BIT:
      switch (format) {
        case VK_FORMAT_G8_B8_R8_3PLANE_420_UNORM:
        case VK_FORMAT_G8_B8R8_2PLANE_420_UNORM:
        case VK_FORMAT_G8_B8_R8_3PLANE_422_UNORM:
        case VK_FORMAT_G8_B8R8_2PLANE_422_UNORM:
        case VK_FORMAT_G8_B8_R8_3PLANE_444_UNORM:
          return element_and_block_size{
              1, texel_block_size{1, 1}};
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_420_UNORM_3PACK16:
        case VK_FORMAT_G10X6_B10X6R10X6_2PLANE_420_UNORM_3PACK16:
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_422_UNORM_3PACK16:
        case VK_FORMAT_G10X6_B10X6R10X6_2PLANE_422_UNORM_3PACK16:
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_444_UNORM_3PACK16:
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_420_UNORM_3PACK16:
        case VK_FORMAT_G12X4_B12X4R12X4_2PLANE_420_UNORM_3PACK16:
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_422_UNORM_3PACK16:
        case VK_FORMAT_G12X4_B12X4R12X4_2PLANE_422_UNORM_3PACK16:
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_444_UNORM_3PACK16:
        case VK_FORMAT_G16_B16_R16_3PLANE_420_UNORM:
        case VK_FORMAT_G16_B16R16_2PLANE_420_UNORM:
        case VK_FORMAT_G16_B16_R16_3PLANE_422_UNORM:
        case VK_FORMAT_G16_B16R16_2PLANE_422_UNORM:
        case VK_FORMAT_G16_B16_R16_3PLANE_444_UNORM:
          return element_and_block_size{
              2, texel_block_size{1, 1}};
        default:
          return original_aspect;
      }
    case VK_IMAGE_ASPECT_PLANE_1_BIT:
      switch (format) {
        case VK_FORMAT_G8_B8_R8_3PLANE_420_UNORM:
          return element_and_block_size{1, texel_block_size{1, 1}};
        case VK_FORMAT_G8_B8R8_2PLANE_420_UNORM:
          return element_and_block_size{2, texel_block_size{1, 1}};
        case VK_FORMAT_G8_B8_R8_3PLANE_422_UNORM:
          return element_and_block_size{1, texel_block_size{1, 1}};
        case VK_FORMAT_G8_B8R8_2PLANE_422_UNORM:
          return element_and_block_size{2, texel_block_size{1, 1}};
        case VK_FORMAT_G8_B8_R8_3PLANE_444_UNORM:
          return element_and_block_size{1, texel_block_size{1, 1}};
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_420_UNORM_3PACK16:
          return element_and_block_size{2, texel_block_size{1, 1}};
        case VK_FORMAT_G10X6_B10X6R10X6_2PLANE_420_UNORM_3PACK16:
          return element_and_block_size{4, texel_block_size{1, 1}};
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_422_UNORM_3PACK16:
          return element_and_block_size{2, texel_block_size{1, 1}};
        case VK_FORMAT_G10X6_B10X6R10X6_2PLANE_422_UNORM_3PACK16:
          return element_and_block_size{4, texel_block_size{1, 1}};
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_444_UNORM_3PACK16:
          return element_and_block_size{2, texel_block_size{1, 1}};
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_420_UNORM_3PACK16:
          return element_and_block_size{2, texel_block_size{1, 1}};
        case VK_FORMAT_G12X4_B12X4R12X4_2PLANE_420_UNORM_3PACK16:
          return element_and_block_size{4, texel_block_size{1, 1}};
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_422_UNORM_3PACK16:
          return element_and_block_size{2, texel_block_size{1, 1}};
        case VK_FORMAT_G12X4_B12X4R12X4_2PLANE_422_UNORM_3PACK16:
          return element_and_block_size{4, texel_block_size{1, 1}};
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_444_UNORM_3PACK16:
          return element_and_block_size{2, texel_block_size{1, 1}};
        case VK_FORMAT_G16_B16_R16_3PLANE_420_UNORM:
          return element_and_block_size{2, texel_block_size{1, 1}};
        case VK_FORMAT_G16_B16R16_2PLANE_420_UNORM:
          return element_and_block_size{4, texel_block_size{1, 1}};
        case VK_FORMAT_G16_B16_R16_3PLANE_422_UNORM:
          return element_and_block_size{2, texel_block_size{1, 1}};
        case VK_FORMAT_G16_B16R16_2PLANE_422_UNORM:
          return element_and_block_size{4, texel_block_size{1, 1}};
        case VK_FORMAT_G16_B16_R16_3PLANE_444_UNORM:
          return element_and_block_size{2, texel_block_size{1, 1}};
        default:
          GAPID2_ERROR("Unhandled multiplane format");
      }
    case VK_IMAGE_ASPECT_PLANE_2_BIT:
      switch (format) {
        case VK_FORMAT_G8_B8_R8_3PLANE_420_UNORM:
        case VK_FORMAT_G8_B8_R8_3PLANE_422_UNORM:
        case VK_FORMAT_G8_B8_R8_3PLANE_444_UNORM:
          return element_and_block_size{1, texel_block_size{1, 1}};
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_420_UNORM_3PACK16:
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_422_UNORM_3PACK16:
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_444_UNORM_3PACK16:
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_420_UNORM_3PACK16:
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_422_UNORM_3PACK16:
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_444_UNORM_3PACK16:
          return element_and_block_size{2, texel_block_size{1, 1}};
        case VK_FORMAT_G16_B16_R16_3PLANE_420_UNORM:
        case VK_FORMAT_G16_B16_R16_3PLANE_422_UNORM:
        case VK_FORMAT_G16_B16_R16_3PLANE_444_UNORM:
          return element_and_block_size{1, texel_block_size{1, 1}};
        default:
          GAPID2_ERROR("Unhandled multiplane format");
          return element_and_block_size();
      }
  }
}

inline texel_block_size get_aspect_size_divisor(VkFormat format, VkImageAspectFlagBits aspect) {
  switch (aspect) {
    case VK_IMAGE_ASPECT_PLANE_0_BIT:
      return texel_block_size{1, 1};
    case VK_IMAGE_ASPECT_PLANE_1_BIT:
      switch (format) {
        case VK_FORMAT_G8_B8_R8_3PLANE_420_UNORM:
          return texel_block_size{2, 2};
        case VK_FORMAT_G8_B8R8_2PLANE_420_UNORM:
          return texel_block_size{2, 2};
        case VK_FORMAT_G8_B8_R8_3PLANE_422_UNORM:
          return texel_block_size{2, 1};
        case VK_FORMAT_G8_B8R8_2PLANE_422_UNORM:
          return texel_block_size{2, 1};
        case VK_FORMAT_G8_B8_R8_3PLANE_444_UNORM:
          return texel_block_size{1, 1};
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_420_UNORM_3PACK16:
          return texel_block_size{2, 2};
        case VK_FORMAT_G10X6_B10X6R10X6_2PLANE_420_UNORM_3PACK16:
          return texel_block_size{2, 2};
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_422_UNORM_3PACK16:
          return texel_block_size{2, 1};
        case VK_FORMAT_G10X6_B10X6R10X6_2PLANE_422_UNORM_3PACK16:
          return texel_block_size{2, 1};
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_444_UNORM_3PACK16:
          return texel_block_size{1, 1};
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_420_UNORM_3PACK16:
          return texel_block_size{2, 2};
        case VK_FORMAT_G12X4_B12X4R12X4_2PLANE_420_UNORM_3PACK16:
          return texel_block_size{2, 2};
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_422_UNORM_3PACK16:
          return texel_block_size{2, 1};
        case VK_FORMAT_G12X4_B12X4R12X4_2PLANE_422_UNORM_3PACK16:
          return texel_block_size{2, 1};
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_444_UNORM_3PACK16:
          return texel_block_size{1, 1};
        case VK_FORMAT_G16_B16_R16_3PLANE_420_UNORM:
          return texel_block_size{2, 2};
        case VK_FORMAT_G16_B16R16_2PLANE_420_UNORM:
          return texel_block_size{2, 2};
        case VK_FORMAT_G16_B16_R16_3PLANE_422_UNORM:
          return texel_block_size{2, 1};
        case VK_FORMAT_G16_B16R16_2PLANE_422_UNORM:
          return texel_block_size{2, 1};
        case VK_FORMAT_G16_B16_R16_3PLANE_444_UNORM:
          return texel_block_size{1, 1};
        default:
          GAPID2_ERROR("Unhandled multiplane format");
      }
    case VK_IMAGE_ASPECT_PLANE_2_BIT:
      switch (format) {
        case VK_FORMAT_G8_B8_R8_3PLANE_420_UNORM:
          return texel_block_size{2, 2};
        case VK_FORMAT_G8_B8_R8_3PLANE_422_UNORM:
          return texel_block_size{2, 1};
        case VK_FORMAT_G8_B8_R8_3PLANE_444_UNORM:
          return texel_block_size{1, 1};
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_420_UNORM_3PACK16:
          return texel_block_size{2, 2};
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_422_UNORM_3PACK16:
          return texel_block_size{2, 1};
        case VK_FORMAT_G10X6_B10X6_R10X6_3PLANE_444_UNORM_3PACK16:
          return texel_block_size{1, 1};
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_420_UNORM_3PACK16:
          return texel_block_size{2, 2};
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_422_UNORM_3PACK16:
          return texel_block_size{2, 1};
        case VK_FORMAT_G12X4_B12X4_R12X4_3PLANE_444_UNORM_3PACK16:
          return texel_block_size{1, 1};
        case VK_FORMAT_G16_B16_R16_3PLANE_420_UNORM:
          return texel_block_size{2, 2};
        case VK_FORMAT_G16_B16_R16_3PLANE_422_UNORM:
          return texel_block_size{2, 1};
        case VK_FORMAT_G16_B16_R16_3PLANE_444_UNORM:
          return texel_block_size{1, 1};
      }
    default:
      return texel_block_size{1, 1};
  }
}

inline uint32_t get_mip_size(uint32_t original, uint32_t level) {
  auto value = original / (1 << level);
  if (!value) {
    return original ? 1 : 0;
  }

  return value;
}

enum class data_type : uint8_t {
  uint,
  sint,
  unorm,
  snorm,
  uscaled,
  sscaled,
  sfloat,
  srgb,
  shared_exponent_mantissa,
  shared_exponent_exponent,
  ufloat  // boo ufloat
};

enum class e_channel_name : uint8_t {
  r,
  g,
  b,
  a,
  e,
  d,
  s,
  none
};

struct channel_info {
  uint8_t nbits;
  data_type type;
  e_channel_name name = e_channel_name::none;
};

static const uint8_t kMaxChannels = 4;

struct image_layout {
  uint8_t n_channels;
  uint8_t stride_bits;
  channel_info channels[kMaxChannels];
};

// This should roughly correspond to "Compatible formats" in the Vulkan spec
const inline image_layout* get_buffer_layout_for_aspect(VkFormat format, VkImageAspectFlagBits aspect) {
  if (aspect == VK_IMAGE_ASPECT_DEPTH_BIT) {
    switch (format) {
      case VK_FORMAT_D16_UNORM:
      case VK_FORMAT_D16_UNORM_S8_UINT: {
        static image_layout x{1, 16, {{16, data_type::unorm, e_channel_name::d}}};
        return &x;
      }
      case VK_FORMAT_D32_SFLOAT:
      case VK_FORMAT_D32_SFLOAT_S8_UINT: {
        static image_layout x{1, 32, {{32, data_type::sfloat, e_channel_name::d}}};
        return &x;
      }
      case VK_FORMAT_X8_D24_UNORM_PACK32:
      case VK_FORMAT_D24_UNORM_S8_UINT: {
        static image_layout x{1, 32, {{24, data_type::unorm, e_channel_name::d}}};
        return &x;
      }
    }
    GAPID2_ERROR("Invalid image format for depth");
  } else if (aspect == VK_IMAGE_ASPECT_STENCIL_BIT) {
    switch (format) {
      case VK_FORMAT_S8_UINT:
      case VK_FORMAT_D16_UNORM_S8_UINT:
      case VK_FORMAT_D24_UNORM_S8_UINT:
      case VK_FORMAT_D32_SFLOAT_S8_UINT: {
        static image_layout x(1, 8, {{8, data_type::uint, e_channel_name::s}});
        return &x;
      }
    }
    GAPID2_ERROR("Invalid image format for stencil");
  }

  switch (format) {
    case VK_FORMAT_R4G4_UNORM_PACK8: {
      static image_layout x{2, 8, {{4, data_type::unorm, e_channel_name::r}, {4, data_type::unorm, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R8_UNORM: {
      static image_layout x{1, 8, {{8, data_type::unorm, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R8_SNORM: {
      static image_layout x{1, 8, {{8, data_type::snorm, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R8_USCALED: {
      static image_layout x{1, 8, {{8, data_type::uscaled, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R8_SSCALED: {
      static image_layout x{1, 8, {{8, data_type::sscaled, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R8_UINT: {
      static image_layout x{1, 8, {{8, data_type::uint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R8_SINT: {
      static image_layout x{1, 8, {{8, data_type::sint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R8_SRGB: {
      static image_layout x{1, 8, {{8, data_type::srgb, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R4G4B4A4_UNORM_PACK16: {
      static image_layout x{4, 16, {{4, data_type::unorm, e_channel_name::r}, {4, data_type::unorm, e_channel_name::g}, {4, data_type::unorm, e_channel_name::b}, {4, data_type::unorm, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_B4G4R4A4_UNORM_PACK16: {
      static image_layout x{4, 16, {{4, data_type::unorm, e_channel_name::b}, {4, data_type::unorm, e_channel_name::g}, {4, data_type::unorm, e_channel_name::r}, {4, data_type::unorm, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R5G6B5_UNORM_PACK16: {
      static image_layout x{3, 16, {{5, data_type::unorm, e_channel_name::r}, {6, data_type::unorm, e_channel_name::g}, {5, data_type::unorm, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_B5G6R5_UNORM_PACK16: {
      static image_layout x{3, 16, {{5, data_type::unorm, e_channel_name::b}, {6, data_type::unorm, e_channel_name::g}, {5, data_type::unorm, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R5G5B5A1_UNORM_PACK16: {
      static image_layout x(4, 16, {{5, data_type::unorm, e_channel_name::r}, {5, data_type::unorm, e_channel_name::g}, {5, data_type::unorm, e_channel_name::b}, {1, data_type::unorm, e_channel_name::a}});
      return &x;
    }
    case VK_FORMAT_B5G5R5A1_UNORM_PACK16: {
      static image_layout x(4, 16, {{5, data_type::unorm, e_channel_name::b}, {5, data_type::unorm, e_channel_name::g}, {5, data_type::unorm, e_channel_name::r}, {1, data_type::unorm, e_channel_name::a}});
      return &x;
    }
    case VK_FORMAT_A1R5G5B5_UNORM_PACK16: {
      static image_layout x(4, 16, {{1, data_type::unorm, e_channel_name::a}, {5, data_type::unorm, e_channel_name::r}, {5, data_type::unorm, e_channel_name::g}, {5, data_type::unorm, e_channel_name::b}});
      return &x;
    }
    case VK_FORMAT_R8G8_UNORM: {
      static image_layout x{2, 16, {{8, data_type::unorm, e_channel_name::r}, {8, data_type::unorm, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R8G8_SNORM: {
      static image_layout x{2, 16, {{8, data_type::snorm, e_channel_name::r}, {8, data_type::snorm, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R8G8_USCALED: {
      static image_layout x{2, 16, {{8, data_type::uscaled, e_channel_name::r}, {8, data_type::uscaled, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R8G8_SSCALED: {
      static image_layout x{2, 16, {{8, data_type::sscaled, e_channel_name::r}, {8, data_type::sscaled, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R8G8_UINT: {
      static image_layout x{2, 16, {{8, data_type::uint, e_channel_name::r}, {8, data_type::uint, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R8G8_SINT: {
      static image_layout x{2, 16, {{8, data_type::sint, e_channel_name::r}, {8, data_type::sint, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R8G8_SRGB: {
      static image_layout x{2, 16, {{8, data_type::srgb, e_channel_name::r}, {8, data_type::srgb, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R16_UNORM: {
      static image_layout x{1, 16, {{16, data_type::unorm, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R16_SNORM: {
      static image_layout x{1, 16, {{16, data_type::snorm, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R16_USCALED: {
      static image_layout x{1, 16, {{16, data_type::uscaled, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R16_SSCALED: {
      static image_layout x{1, 16, {{16, data_type::sscaled, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R16_UINT: {
      static image_layout x{1, 16, {{16, data_type::uint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R16_SINT: {
      static image_layout x{1, 16, {{16, data_type::sint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R16_SFLOAT: {
      static image_layout x{1, 16, {{16, data_type::sfloat, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8_UNORM: {
      static image_layout x{3, 24, {{8, data_type::unorm, e_channel_name::r}, {8, data_type::unorm, e_channel_name::g}, {8, data_type::unorm, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8_SNORM: {
      static image_layout x{3, 24, {{8, data_type::snorm, e_channel_name::r}, {8, data_type::snorm, e_channel_name::g}, {8, data_type::snorm, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8_USCALED: {
      static image_layout x{3, 24, {{8, data_type::uscaled, e_channel_name::r}, {8, data_type::uscaled, e_channel_name::g}, {8, data_type::uscaled, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8_SSCALED: {
      static image_layout x{3, 24, {{8, data_type::sscaled, e_channel_name::r}, {8, data_type::sscaled, e_channel_name::g}, {8, data_type::sscaled, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8_UINT: {
      static image_layout x{3, 24, {{8, data_type::uint, e_channel_name::r}, {8, data_type::uint, e_channel_name::g}, {8, data_type::uint, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8_SINT: {
      static image_layout x{3, 24, {{8, data_type::sint, e_channel_name::r}, {8, data_type::sint, e_channel_name::g}, {8, data_type::sint, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8_SRGB: {
      static image_layout x{3, 24, {{8, data_type::srgb, e_channel_name::r}, {8, data_type::srgb, e_channel_name::g}, {8, data_type::srgb, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8_UNORM: {
      static image_layout x{3, 24, {{8, data_type::unorm, e_channel_name::b}, {8, data_type::unorm, e_channel_name::g}, {8, data_type::unorm, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8_SNORM: {
      static image_layout x{3, 24, {{8, data_type::snorm, e_channel_name::b}, {8, data_type::snorm, e_channel_name::g}, {8, data_type::snorm, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8_USCALED: {
      static image_layout x{3, 24, {{8, data_type::uscaled, e_channel_name::b}, {8, data_type::uscaled, e_channel_name::g}, {8, data_type::uscaled, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8_SSCALED: {
      static image_layout x{3, 24, {{8, data_type::sscaled, e_channel_name::b}, {8, data_type::sscaled, e_channel_name::g}, {8, data_type::sscaled, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8_UINT: {
      static image_layout x{3, 24, {{8, data_type::uint, e_channel_name::b}, {8, data_type::uint, e_channel_name::g}, {8, data_type::uint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8_SINT: {
      static image_layout x{3, 24, {{8, data_type::sint, e_channel_name::b}, {8, data_type::sint, e_channel_name::g}, {8, data_type::sint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8_SRGB: {
      static image_layout x{3, 24, {{8, data_type::srgb, e_channel_name::b}, {8, data_type::srgb, e_channel_name::g}, {8, data_type::srgb, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8A8_UNORM: {
      static image_layout x{4, 32, {{8, data_type::unorm, e_channel_name::r}, {8, data_type::unorm, e_channel_name::g}, {8, data_type::unorm, e_channel_name::b}, {8, data_type::unorm, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8A8_SNORM: {
      static image_layout x{4, 32, {{8, data_type::snorm, e_channel_name::r}, {8, data_type::snorm, e_channel_name::g}, {8, data_type::snorm, e_channel_name::b}, {8, data_type::snorm, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8A8_USCALED: {
      static image_layout x{4, 32, {{8, data_type::uscaled, e_channel_name::r}, {8, data_type::uscaled, e_channel_name::g}, {8, data_type::uscaled, e_channel_name::b}, {8, data_type::uscaled, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8A8_SSCALED: {
      static image_layout x{4, 32, {{8, data_type::sscaled, e_channel_name::r}, {8, data_type::sscaled, e_channel_name::g}, {8, data_type::sscaled, e_channel_name::b}, {8, data_type::sscaled, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8A8_UINT: {
      static image_layout x{4, 32, {{8, data_type::uint, e_channel_name::r}, {8, data_type::uint, e_channel_name::g}, {8, data_type::uint, e_channel_name::b}, {8, data_type::uint, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8A8_SINT: {
      static image_layout x{4, 32, {{8, data_type::sint, e_channel_name::r}, {8, data_type::sint, e_channel_name::g}, {8, data_type::sint, e_channel_name::b}, {8, data_type::sint, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R8G8B8A8_SRGB: {
      static image_layout x{4, 32, {{8, data_type::srgb, e_channel_name::r}, {8, data_type::srgb, e_channel_name::g}, {8, data_type::srgb, e_channel_name::b}, {8, data_type::srgb, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8A8_UNORM: {
      static image_layout x{4, 32, {{8, data_type::unorm, e_channel_name::b}, {8, data_type::unorm, e_channel_name::g}, {8, data_type::unorm, e_channel_name::r}, {8, data_type::unorm, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8A8_SNORM: {
      static image_layout x{4, 32, {{8, data_type::snorm, e_channel_name::b}, {8, data_type::snorm, e_channel_name::g}, {8, data_type::snorm, e_channel_name::r}, {8, data_type::snorm, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8A8_USCALED: {
      static image_layout x{4, 32, {{8, data_type::uscaled, e_channel_name::b}, {8, data_type::uscaled, e_channel_name::g}, {8, data_type::uscaled, e_channel_name::r}, {8, data_type::uscaled, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8A8_SSCALED: {
      static image_layout x{4, 32, {{8, data_type::sscaled, e_channel_name::b}, {8, data_type::sscaled, e_channel_name::g}, {8, data_type::sscaled, e_channel_name::r}, {8, data_type::sscaled, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8A8_UINT: {
      static image_layout x{4, 32, {{8, data_type::uint, e_channel_name::b}, {8, data_type::uint, e_channel_name::g}, {8, data_type::uint, e_channel_name::r}, {8, data_type::uint, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8A8_SINT: {
      static image_layout x{4, 32, {{8, data_type::sint, e_channel_name::b}, {8, data_type::sint, e_channel_name::g}, {8, data_type::sint, e_channel_name::r}, {8, data_type::sint, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_B8G8R8A8_SRGB: {
      static image_layout x{4, 32, {{8, data_type::srgb, e_channel_name::b}, {8, data_type::srgb, e_channel_name::g}, {8, data_type::srgb, e_channel_name::r}, {8, data_type::srgb, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_A8B8G8R8_UNORM_PACK32: {
      static image_layout x{4, 32, {{8, data_type::unorm, e_channel_name::a}, {8, data_type::unorm, e_channel_name::b}, {8, data_type::unorm, e_channel_name::g}, {8, data_type::unorm, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_A8B8G8R8_SNORM_PACK32: {
      static image_layout x{4, 32, {{8, data_type::snorm, e_channel_name::a}, {8, data_type::snorm, e_channel_name::b}, {8, data_type::snorm, e_channel_name::g}, {8, data_type::snorm, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_A8B8G8R8_USCALED_PACK32: {
      static image_layout x{4, 32, {{8, data_type::uscaled, e_channel_name::a}, {8, data_type::uscaled, e_channel_name::b}, {8, data_type::uscaled, e_channel_name::g}, {8, data_type::uscaled, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_A8B8G8R8_SSCALED_PACK32: {
      static image_layout x{4, 32, {{8, data_type::sscaled, e_channel_name::a}, {8, data_type::sscaled, e_channel_name::b}, {8, data_type::sscaled, e_channel_name::g}, {8, data_type::sscaled, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_A8B8G8R8_UINT_PACK32: {
      static image_layout x{4, 32, {{8, data_type::uint, e_channel_name::a}, {8, data_type::uint, e_channel_name::b}, {8, data_type::uint, e_channel_name::g}, {8, data_type::uint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_A8B8G8R8_SINT_PACK32: {
      static image_layout x{4, 32, {{8, data_type::sint, e_channel_name::a}, {8, data_type::sint, e_channel_name::b}, {8, data_type::sint, e_channel_name::g}, {8, data_type::sint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_A8B8G8R8_SRGB_PACK32: {
      static image_layout x{4, 32, {{8, data_type::srgb, e_channel_name::a}, {8, data_type::srgb, e_channel_name::b}, {8, data_type::srgb, e_channel_name::g}, {8, data_type::srgb, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_A2R10G10B10_UNORM_PACK32: {
      static image_layout x{4, 32, {{2, data_type::unorm, e_channel_name::a}, {10, data_type::unorm, e_channel_name::r}, {10, data_type::unorm, e_channel_name::g}, {10, data_type::unorm, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_A2R10G10B10_SNORM_PACK32: {
      static image_layout x{4, 32, {{2, data_type::snorm, e_channel_name::a}, {10, data_type::snorm, e_channel_name::r}, {10, data_type::snorm, e_channel_name::g}, {10, data_type::snorm, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_A2R10G10B10_USCALED_PACK32: {
      static image_layout x{4, 32, {{2, data_type::uscaled, e_channel_name::a}, {10, data_type::uscaled, e_channel_name::r}, {10, data_type::uscaled, e_channel_name::g}, {10, data_type::uscaled, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_A2R10G10B10_SSCALED_PACK32: {
      static image_layout x{4, 32, {{2, data_type::sscaled, e_channel_name::a}, {10, data_type::sscaled, e_channel_name::r}, {10, data_type::sscaled, e_channel_name::g}, {10, data_type::sscaled, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_A2R10G10B10_UINT_PACK32: {
      static image_layout x{4, 32, {{2, data_type::uint, e_channel_name::a}, {10, data_type::uint, e_channel_name::r}, {10, data_type::uint, e_channel_name::g}, {10, data_type::uint, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_A2R10G10B10_SINT_PACK32: {
      static image_layout x{4, 32, {{2, data_type::sint, e_channel_name::a}, {10, data_type::sint, e_channel_name::r}, {10, data_type::sint, e_channel_name::g}, {10, data_type::sint, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_A2B10G10R10_UNORM_PACK32: {
      static image_layout x{4, 32, {{2, data_type::unorm, e_channel_name::a}, {10, data_type::unorm, e_channel_name::b}, {10, data_type::unorm, e_channel_name::g}, {10, data_type::unorm, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_A2B10G10R10_SNORM_PACK32: {
      static image_layout x{4, 32, {{2, data_type::snorm, e_channel_name::a}, {10, data_type::snorm, e_channel_name::b}, {10, data_type::snorm, e_channel_name::g}, {10, data_type::snorm, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_A2B10G10R10_USCALED_PACK32: {
      static image_layout x{4, 32, {{2, data_type::uscaled, e_channel_name::a}, {10, data_type::uscaled, e_channel_name::b}, {10, data_type::uscaled, e_channel_name::g}, {10, data_type::uscaled, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_A2B10G10R10_SSCALED_PACK32: {
      static image_layout x{4, 32, {{2, data_type::sscaled, e_channel_name::a}, {10, data_type::sscaled, e_channel_name::b}, {10, data_type::sscaled, e_channel_name::g}, {10, data_type::sscaled, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_A2B10G10R10_UINT_PACK32: {
      static image_layout x{4, 32, {{2, data_type::uint, e_channel_name::a}, {10, data_type::uint, e_channel_name::b}, {10, data_type::uint, e_channel_name::g}, {10, data_type::uint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_A2B10G10R10_SINT_PACK32: {
      static image_layout x{4, 32, {{2, data_type::sint, e_channel_name::a}, {10, data_type::sint, e_channel_name::b}, {10, data_type::sint, e_channel_name::g}, {10, data_type::sint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R16G16_UNORM: {
      static image_layout x{2, 32, {{16, data_type::unorm, e_channel_name::r}, {16, data_type::unorm, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R16G16_SNORM: {
      static image_layout x{2, 32, {{16, data_type::snorm, e_channel_name::r}, {16, data_type::snorm, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R16G16_USCALED: {
      static image_layout x{2, 32, {{16, data_type::uscaled, e_channel_name::r}, {16, data_type::uscaled, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R16G16_SSCALED: {
      static image_layout x{2, 32, {{16, data_type::sscaled, e_channel_name::r}, {16, data_type::sscaled, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R16G16_UINT: {
      static image_layout x{2, 32, {{16, data_type::uint, e_channel_name::r}, {16, data_type::uint, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R16G16_SINT: {
      static image_layout x{2, 32, {{16, data_type::sint, e_channel_name::r}, {16, data_type::sint, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R16G16_SFLOAT: {
      static image_layout x{2, 32, {{16, data_type::sfloat, e_channel_name::r}, {16, data_type::sfloat, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R32_UINT: {
      static image_layout x{1, 32, {{32, data_type::uint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R32_SINT: {
      static image_layout x{1, 32, {{32, data_type::sint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R32_SFLOAT: {
      static image_layout x{1, 32, {{32, data_type::sfloat, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_B10G11R11_UFLOAT_PACK32: {
      static image_layout x{3, 32, {{10, data_type::ufloat, e_channel_name::b}, {11, data_type::ufloat, e_channel_name::g}, {11, data_type::ufloat, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_E5B9G9R9_UFLOAT_PACK32: {
      static image_layout x{4, 32, {{5, data_type::shared_exponent_exponent, e_channel_name::e}, {9, data_type::shared_exponent_mantissa, e_channel_name::b}, {9, data_type::shared_exponent_mantissa, e_channel_name::g}, {9, data_type::shared_exponent_mantissa, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16_UNORM: {
      static image_layout x{3, 48, {{16, data_type::unorm, e_channel_name::r}, {16, data_type::unorm, e_channel_name::g}, {16, data_type::unorm, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16_SNORM: {
      static image_layout x{3, 48, {{16, data_type::snorm, e_channel_name::r}, {16, data_type::snorm, e_channel_name::g}, {16, data_type::snorm, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16_USCALED: {
      static image_layout x{3, 48, {{16, data_type::uscaled, e_channel_name::r}, {16, data_type::uscaled, e_channel_name::g}, {16, data_type::uscaled, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16_SSCALED: {
      static image_layout x{3, 48, {{16, data_type::sscaled, e_channel_name::r}, {16, data_type::sscaled, e_channel_name::g}, {16, data_type::sscaled, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16_UINT: {
      static image_layout x{3, 48, {{16, data_type::uint, e_channel_name::r}, {16, data_type::uint, e_channel_name::g}, {16, data_type::uint, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16_SINT: {
      static image_layout x{3, 48, {{16, data_type::sint, e_channel_name::r}, {16, data_type::sint, e_channel_name::g}, {16, data_type::sint, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16_SFLOAT: {
      static image_layout x{3, 48, {{16, data_type::sfloat, e_channel_name::r}, {16, data_type::sfloat, e_channel_name::g}, {16, data_type::sfloat, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16A16_UNORM: {
      static image_layout x{4, 64, {{16, data_type::unorm, e_channel_name::r}, {16, data_type::unorm, e_channel_name::g}, {16, data_type::unorm, e_channel_name::b}, {16, data_type::unorm, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16A16_SNORM: {
      static image_layout x{4, 64, {{16, data_type::snorm, e_channel_name::r}, {16, data_type::snorm, e_channel_name::g}, {16, data_type::snorm, e_channel_name::b}, {16, data_type::snorm, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16A16_USCALED: {
      static image_layout x{4, 64, {{16, data_type::uscaled, e_channel_name::r}, {16, data_type::uscaled, e_channel_name::g}, {16, data_type::uscaled, e_channel_name::b}, {16, data_type::uscaled, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16A16_SSCALED: {
      static image_layout x{4, 64, {{16, data_type::sscaled, e_channel_name::r}, {16, data_type::sscaled, e_channel_name::g}, {16, data_type::sscaled, e_channel_name::b}, {16, data_type::sscaled, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16A16_UINT: {
      static image_layout x{4, 64, {{16, data_type::uint, e_channel_name::r}, {16, data_type::uint, e_channel_name::g}, {16, data_type::uint, e_channel_name::b}, {16, data_type::uint, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16A16_SINT: {
      static image_layout x{4, 64, {{16, data_type::sint, e_channel_name::r}, {16, data_type::sint, e_channel_name::g}, {16, data_type::sint, e_channel_name::b}, {16, data_type::sint, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R16G16B16A16_SFLOAT: {
      static image_layout x{4, 64, {{16, data_type::sfloat, e_channel_name::r}, {16, data_type::sfloat, e_channel_name::g}, {16, data_type::sfloat, e_channel_name::b}, {16, data_type::sfloat, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R32G32_UINT: {
      static image_layout x{2, 64, {{32, data_type::uint, e_channel_name::r}, {32, data_type::uint, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R32G32_SINT: {
      static image_layout x{2, 64, {{32, data_type::sint, e_channel_name::r}, {32, data_type::sint, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R32G32_SFLOAT: {
      static image_layout x{2, 64, {{32, data_type::sfloat, e_channel_name::r}, {32, data_type::sfloat, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R64_UINT: {
      static image_layout x{1, 64, {{64, data_type::uint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R64_SINT: {
      static image_layout x{1, 64, {{64, data_type::sint, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R64_SFLOAT: {
      static image_layout x{1, 64, {{64, data_type::sfloat, e_channel_name::r}}};
      return &x;
    }
    case VK_FORMAT_R32G32B32_UINT: {
      static image_layout x{3, 96, {{32, data_type::uint, e_channel_name::r}, {32, data_type::uint, e_channel_name::g}, {32, data_type::uint, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R32G32B32_SINT: {
      static image_layout x{3, 96, {{32, data_type::sint, e_channel_name::r}, {32, data_type::sint, e_channel_name::g}, {32, data_type::sint, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R32G32B32_SFLOAT: {
      static image_layout x{3, 96, {{32, data_type::sfloat, e_channel_name::r}, {32, data_type::sfloat, e_channel_name::g}, {32, data_type::sfloat, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R32G32B32A32_UINT: {
      static image_layout x{3, 128, {{32, data_type::uint, e_channel_name::r}, {32, data_type::uint, e_channel_name::g}, {32, data_type::uint, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R32G32B32A32_SINT: {
      static image_layout x{3, 128, {{32, data_type::sint, e_channel_name::r}, {32, data_type::sint, e_channel_name::g}, {32, data_type::sint, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R32G32B32A32_SFLOAT: {
      static image_layout x{3, 128, {{32, data_type::sfloat, e_channel_name::r}, {32, data_type::sfloat, e_channel_name::g}, {32, data_type::sfloat, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R64G64_UINT: {
      static image_layout x{2, 128, {{64, data_type::uint, e_channel_name::r}, {64, data_type::uint, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R64G64_SINT: {
      static image_layout x{2, 128, {{64, data_type::sint, e_channel_name::r}, {64, data_type::sint, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R64G64_SFLOAT: {
      static image_layout x{2, 128, {{64, data_type::sfloat, e_channel_name::r}, {64, data_type::sfloat, e_channel_name::g}}};
      return &x;
    }
    case VK_FORMAT_R64G64B64_UINT: {
      static image_layout x{3, 192, {{64, data_type::uint, e_channel_name::r}, {64, data_type::uint, e_channel_name::g}, {64, data_type::uint, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R64G64B64_SINT: {
      static image_layout x{3, 192, {{64, data_type::sint, e_channel_name::r}, {64, data_type::sint, e_channel_name::g}, {64, data_type::sint, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R64G64B64_SFLOAT: {
      static image_layout x{3, 192, {{64, data_type::sfloat, e_channel_name::r}, {64, data_type::sfloat, e_channel_name::g}, {64, data_type::sfloat, e_channel_name::b}}};
      return &x;
    }
    case VK_FORMAT_R64G64B64A64_UINT: {
      static image_layout x{4, 256, {{64, data_type::uint, e_channel_name::r}, {64, data_type::uint, e_channel_name::g}, {64, data_type::uint, e_channel_name::b}, {64, data_type::uint, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R64G64B64A64_SINT: {
      static image_layout x{4, 256, {{64, data_type::sint, e_channel_name::r}, {64, data_type::sint, e_channel_name::g}, {64, data_type::sint, e_channel_name::b}, {64, data_type::sint, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_R64G64B64A64_SFLOAT: {
      static image_layout x{4, 256, {{64, data_type::sfloat, e_channel_name::r}, {64, data_type::sfloat, e_channel_name::g}, {64, data_type::sfloat, e_channel_name::b}, {64, data_type::sfloat, e_channel_name::a}}};
      return &x;
    }
    case VK_FORMAT_BC1_RGB_UNORM_BLOCK:
    case VK_FORMAT_BC1_RGB_SRGB_BLOCK:
    case VK_FORMAT_BC1_RGBA_UNORM_BLOCK:
    case VK_FORMAT_BC1_RGBA_SRGB_BLOCK:
    case VK_FORMAT_BC2_UNORM_BLOCK:
    case VK_FORMAT_BC2_SRGB_BLOCK:
    case VK_FORMAT_BC3_UNORM_BLOCK:
    case VK_FORMAT_BC3_SRGB_BLOCK:
    case VK_FORMAT_BC4_UNORM_BLOCK:
    case VK_FORMAT_BC4_SNORM_BLOCK:
    case VK_FORMAT_BC5_UNORM_BLOCK:
    case VK_FORMAT_BC5_SNORM_BLOCK:
    case VK_FORMAT_BC6H_UFLOAT_BLOCK:
    case VK_FORMAT_BC6H_SFLOAT_BLOCK:
    case VK_FORMAT_BC7_UNORM_BLOCK:
    case VK_FORMAT_BC7_SRGB_BLOCK:
    case VK_FORMAT_ETC2_R8G8B8_UNORM_BLOCK:
    case VK_FORMAT_ETC2_R8G8B8_SRGB_BLOCK:
    case VK_FORMAT_ETC2_R8G8B8A1_UNORM_BLOCK:
    case VK_FORMAT_ETC2_R8G8B8A1_SRGB_BLOCK:
    case VK_FORMAT_ETC2_R8G8B8A8_UNORM_BLOCK:
    case VK_FORMAT_ETC2_R8G8B8A8_SRGB_BLOCK:
    case VK_FORMAT_EAC_R11_UNORM_BLOCK:
    case VK_FORMAT_EAC_R11_SNORM_BLOCK:
    case VK_FORMAT_EAC_R11G11_UNORM_BLOCK:
    case VK_FORMAT_EAC_R11G11_SNORM_BLOCK:
    case VK_FORMAT_ASTC_4x4_UNORM_BLOCK:
    case VK_FORMAT_ASTC_4x4_SRGB_BLOCK:
    case VK_FORMAT_ASTC_5x4_UNORM_BLOCK:
    case VK_FORMAT_ASTC_5x4_SRGB_BLOCK:
    case VK_FORMAT_ASTC_5x5_UNORM_BLOCK:
    case VK_FORMAT_ASTC_5x5_SRGB_BLOCK:
    case VK_FORMAT_ASTC_6x5_UNORM_BLOCK:
    case VK_FORMAT_ASTC_6x5_SRGB_BLOCK:
    case VK_FORMAT_ASTC_6x6_UNORM_BLOCK:
    case VK_FORMAT_ASTC_6x6_SRGB_BLOCK:
    case VK_FORMAT_ASTC_8x5_UNORM_BLOCK:
    case VK_FORMAT_ASTC_8x5_SRGB_BLOCK:
    case VK_FORMAT_ASTC_8x6_UNORM_BLOCK:
    case VK_FORMAT_ASTC_8x6_SRGB_BLOCK:
    case VK_FORMAT_ASTC_8x8_UNORM_BLOCK:
    case VK_FORMAT_ASTC_8x8_SRGB_BLOCK:
    case VK_FORMAT_ASTC_10x5_UNORM_BLOCK:
    case VK_FORMAT_ASTC_10x5_SRGB_BLOCK:
    case VK_FORMAT_ASTC_10x6_UNORM_BLOCK:
    case VK_FORMAT_ASTC_10x6_SRGB_BLOCK:
    case VK_FORMAT_ASTC_10x8_UNORM_BLOCK:
    case VK_FORMAT_ASTC_10x8_SRGB_BLOCK:
    case VK_FORMAT_ASTC_10x10_UNORM_BLOCK:
    case VK_FORMAT_ASTC_10x10_SRGB_BLOCK:
    case VK_FORMAT_ASTC_12x10_UNORM_BLOCK:
    case VK_FORMAT_ASTC_12x10_SRGB_BLOCK:
    case VK_FORMAT_ASTC_12x12_UNORM_BLOCK:
    case VK_FORMAT_ASTC_12x12_SRGB_BLOCK:
      GAPID2_ERROR("Block based formats do not have buffer layouts");
    case VK_FORMAT_D16_UNORM:
    case VK_FORMAT_X8_D24_UNORM_PACK32:
    case VK_FORMAT_D32_SFLOAT:
    case VK_FORMAT_S8_UINT:
    case VK_FORMAT_D16_UNORM_S8_UINT:
    case VK_FORMAT_D24_UNORM_S8_UINT:
    case VK_FORMAT_D32_SFLOAT_S8_UINT:
      GAPID2_ERROR("Depth formats do not have a non-depth aspects");
    case VK_FORMAT_G8B8G8R8_422_UNORM:
    case VK_FORMAT_B8G8R8G8_422_UNORM:
    case VK_FORMAT_R10X6_UNORM_PACK16:
    case VK_FORMAT_R10X6G10X6_UNORM_2PACK16:
    case VK_FORMAT_R10X6G10X6B10X6A10X6_UNORM_4PACK16:
    case VK_FORMAT_G10X6B10X6G10X6R10X6_422_UNORM_4PACK16:
    case VK_FORMAT_B10X6G10X6R10X6G10X6_422_UNORM_4PACK16:
    case VK_FORMAT_R12X4_UNORM_PACK16:
    case VK_FORMAT_R12X4G12X4_UNORM_2PACK16:
    case VK_FORMAT_G16B16G16R16_422_UNORM:
    case VK_FORMAT_B16G16R16G16_422_UNORM:
    case VK_FORMAT_R12X4G12X4B12X4A12X4_UNORM_4PACK16:
    case VK_FORMAT_G12X4B12X4G12X4R12X4_422_UNORM_4PACK16:
    case VK_FORMAT_B12X4G12X4R12X4G12X4_422_UNORM_4PACK16:
      GAPID2_ERROR("Unimplemented mutli-planar images");
    default:
      GAPID2_ERROR("Unhandled texture format");
  }

  return nullptr;
}

}  // namespace gapid2
