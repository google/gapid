/*
 * Copyright (C) 2019 Google Inc.
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

#ifndef VULKAN_TOOLS_IMAGE_H_
#define VULKAN_TOOLS_IMAGE_H_

#include <vulkan/vulkan.h>
#include <ostream>

namespace vk_tools {

// If |image_format| is a suitable format, the image in |image_data| is
// converted to a PNG, with the data for the PNG being sent to |stream|.
// If |image_format| is in an unsuitable format, the |image_data| might
// be converted in-place to a suitable format before being converted to
// a PNG, or might not be sent to the stream at all.
// Returns true if the data has been sent to the stream.
// Returns false otherwise, and also if |size| does not match the expected
// size from |width|, |height| and |image_format|.
bool WritePng(std::ostream* stream, uint8_t* image_data, size_t size,
              uint32_t width, uint32_t height, VkFormat image_format);

}  // namespace vk_tools

#endif  //  VULKAN_TOOLS_IMAGE_H_
