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

#include "image.h"
#include "staging_resource_manager.h"

namespace gapid2 {
class command_serializer;
class image_copier {
 public:
  image_copier(staging_resource_manager* mgr,
               const state_block* sb) : m_resource_manager(mgr),
                                        m_state_block(sb) {}
  bool get_image_content(
      const VkImageWrapper* image,
      uint32_t array_layer,
      uint32_t mip_level,
      command_serializer* next_serializer,
      transform_base* bypass_caller,
      VkOffset3D offset,
      VkExtent3D extent,
      VkImageAspectFlagBits aspect);

  void convert_data_to_rgba32(const char* data, VkDeviceSize data_size,
                              const VkImageWrapper* src_image,
                              VkExtent3D extent,
                              VkImageAspectFlagBits aspect,
                              std::vector<char>* out_data);

 private:
  staging_resource_manager* m_resource_manager;
  const state_block* m_state_block;
};

}  // namespace gapid2