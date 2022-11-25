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

#include "device.h"

#include "mid_execution_generator.h"
#include "state_block.h"
namespace gapid2 {

void mid_execution_generator::capture_devices(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  serializer->insert_annotation("MecDevices");
  for (auto& it : state_block->VkDevices) {
    auto dev = it.second.second;
    VkDevice device = it.first;
    serializer->vkCreateDevice(
        dev->get_physical_device(),
        dev->get_create_info(),
        nullptr,
        &device);
  }
}

}  // namespace gapid2