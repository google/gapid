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

#include <handles.h>
#include <vk_layer.h>
#include <vulkan.h>
#include <memory>
#include <mutex>
#include <unordered_map>
#include "instance_functions.h"
#include "null_cloner.h"
#include "struct_clone.h"
#include "bind_first.h"

#define REGISTER_CHILD_TYPE(type)           \
 public:                                    \
  void* get_and_increment_child(type t) {   \
    std::unique_lock l(child_mutex);        \
    auto it = __##type##s.find(t);          \
    if (it == __##type##s.end()) {          \
      return VK_NULL_HANDLE;                \
    }                                       \
    it->second.second++;                    \
    return it->second.first;                \
  }                                         \
  void add_child(type t, void* _t) {        \
    std::unique_lock l(child_mutex);        \
    __##type##s[t] = std::make_pair(_t, 1); \
  }                                         \
                                            \
 private:                                   \
  std::unordered_map<type, std::pair<void*, uint32_t>> __##type##s;

namespace gapid2 {
template <typename HandleUpdater>
struct VkInstanceWrapper : handle_base<VkInstance, void> {
  VkInstanceWrapper(VkInstance instance)
      : handle_base<VkInstance, void>(instance) {
    if (HandleUpdater::has_dispatch) {
      dispatch = reinterpret_cast<void**>(instance)[0];
    }
  }

  void set_instance_data(PFN_vkGetInstanceProcAddr get_instance_proc_addr,
                         PFN_vkSetInstanceLoaderData _vkSetInstanceLoaderData) {
    vkSetInstanceLoaderData = _vkSetInstanceLoaderData;
    _functions =
        std::make_unique<InstanceFunctions>(_handle, get_instance_proc_addr);
  }

  std::unique_ptr<InstanceFunctions> _functions;
  PFN_vkSetInstanceLoaderData vkSetInstanceLoaderData;

  void set_create_info(const VkInstanceCreateInfo* pCreateInfo) {
    create_info = mem.get_typed_memory<VkInstanceCreateInfo>(1);
    clone<NullCloner>(&cloner, pCreateInfo[0], create_info[0], &mem);
  }

  REGISTER_CHILD_TYPE(VkDevice);
  REGISTER_CHILD_TYPE(VkPhysicalDevice);
  REGISTER_CHILD_TYPE(VkSurfaceKHR);

 private:
  std::mutex child_mutex;

  VkInstanceCreateInfo* create_info = nullptr;
  NullCloner cloner;
  temporary_allocator mem;
};
}  // namespace gapid2

#undef REGISTER_CHILD_TYPE