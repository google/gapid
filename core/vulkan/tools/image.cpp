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

#include "image.h"

#define STB_IMAGE_WRITE_IMPLEMENTATION
#include "stb_image_write.h"

namespace {

// A function of type 'stbi_write_func' that writes |size| bytes of |data| to a
// stream passed via |context| as a std::ostream pointer.
void WriteDataToStream(void* context, void* data, int size) {
  std::ostream* stream = static_cast<std::ostream*>(context);
  stream->write(static_cast<char*>(data), size);
}

}  // namespace

namespace vk_tools {

bool WritePng(std::ostream* stream, uint8_t* image_data, size_t size,
              uint32_t width, uint32_t height, VkFormat image_format) {
  switch (image_format) {
    case VK_FORMAT_B8G8R8A8_UNORM:
    case VK_FORMAT_B8G8R8A8_UINT: {
      if (width * height * 4u != size) {
        return false;
      }
      // Convert the image data from BGRA to RGBA
      auto data = image_data;
      for (uint32_t y = 0; y < height; y++) {
        uint32_t* row = reinterpret_cast<uint32_t*>(data);
        for (uint32_t x = 0; x < width; x++) {
          uint8_t* bgra = reinterpret_cast<uint8_t*>(row);
          uint8_t b = *bgra;
          *bgra = *(bgra + 2);
          *(bgra + 2) = b;
          row++;
        }
        data += width * 4;
      }
    }
      // fall through

    case VK_FORMAT_R8G8B8A8_UNORM:
    case VK_FORMAT_R8G8B8A8_UINT:
      if (width * height * 4u != size) {
        return false;
      }
      stbi_write_png_to_func(&WriteDataToStream, stream, width, height, 4,
                             image_data, width * 4);
      return true;

    default:
      break;
  }
  return false;
}

}  // namespace vk_tools
