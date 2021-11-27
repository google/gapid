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
#include <vulkan.h>
#include "device.h"
#include "null_cloner.h"

namespace gapid2 {
template <typename HandleUpdater>
struct VkBufferViewWrapper : handle_base<VkBufferView> {
  VkBufferViewWrapper(HandleUpdater*, VkDevice, VkBufferView buffer_view)
      : handle_base<VkBufferView>(buffer_view) {}

  void set_create_info(const VkBufferViewCreateInfo* pCreateInfo) {
    create_info = mem.get_typed_memory<VkBufferViewCreateInfo>(1);
    clone<NullCloner>(&cloner, pCreateInfo[0], create_info[0], &mem);
  }

  VkBufferViewCreateInfo* create_info = nullptr;
  NullCloner cloner;
  temporary_allocator mem;
};
}  // namespace gapid2