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

#include "device.h"
#include "handles.h"
#include "temporary_allocator.h"

namespace gapid2 {
class state_block;

struct VkDeviceMemoryWrapper : handle_base<VkDeviceMemory> {
  VkDeviceMemoryWrapper(VkDeviceMemory fence)
      : handle_base<VkDeviceMemory>(fence) {}

  void set_allocate_info(state_block* state_block_, const VkMemoryAllocateInfo* pAllocateInfo);

  VkMemoryAllocateInfo* allocate_info = nullptr;
  temporary_allocator mem;

  VkDeviceSize _mapped_size;
  VkDeviceSize _mapped_offset;
  char* _mapped_location = nullptr;
  bool _is_coherent = false;
  VkDeviceSize _size = 0;
  ;
};

struct memory_binding {
  VkDeviceMemory memory;
  VkDeviceSize offset;
  VkDeviceSize size;
};
}  // namespace gapid2
