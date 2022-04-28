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

#pragma once

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include "handles.h"
#include "instance.h"
#include "temporary_allocator.h"

#define REGISTER_CHILD_TYPE(type)                                   \
 public:                                                            \
  type get_and_increment_child(type t) {                            \
    std::unique_lock l(child_mutex);                                \
    auto it = __##type##s.find(t);                                  \
    if (it == __##type##s.end()) {                                  \
      return VK_NULL_HANDLE;                                        \
    }                                                               \
    it->second.second++;                                            \
    return it->second.first;                                        \
  }                                                                 \
  void add_child(type t, void* _t) {                                \
    std::unique_lock l(child_mutex);                                \
    __##type##s[t] = std::make_pair(reinterpret_cast<type>(_t), 1); \
  }                                                                 \
                                                                    \
 private:                                                           \
  std::unordered_map<type, std::pair<type, uint32_t>> __##type##s;

namespace gapid2 {
struct VkPhysicalDeviceWrapper : handle_base<VkPhysicalDevice, void> {
  VkPhysicalDeviceWrapper(
      VkPhysicalDevice physical_device)
      : handle_base<VkPhysicalDevice, void>(physical_device) {
  }

  std::mutex child_mutex;
  REGISTER_CHILD_TYPE(VkDevice);
};
}  // namespace gapid2

#undef REGISTER_CHILD_TYPE