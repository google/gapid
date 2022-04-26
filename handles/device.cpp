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

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include <memory>
#include <mutex>

#include "forwards.h"
#include "physical_device.h"
#include "struct_clone.h"

namespace gapid2 {

void VkDeviceWrapper::set_create_info(state_block* state_block_, const VkDeviceCreateInfo* pCreateInfo) {
  create_info = mem.get_typed_memory<VkDeviceCreateInfo>(1);
  clone(
      state_block_, pCreateInfo[0], create_info[0], &mem,
      _VkDeviceCreateInfo_VkPhysicalDeviceFeatures2_VkPhysicalDeviceShaderSubgroupExtendedTypesFeatures_shaderSubgroupExtendedTypes_valid,
      _VkDeviceCreateInfo_VkPhysicalDeviceShaderSubgroupExtendedTypesFeatures_shaderSubgroupExtendedTypes_valid);
}

}  // namespace gapid2

#undef REGISTER_CHILD_TYPE