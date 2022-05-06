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

#include "physical_device.h"

#include <map>

#include "mid_execution_generator.h"
#include "state_block.h"

namespace gapid2 {

void mid_execution_generator::capture_physical_devices(const state_block* state_block, noop_serializer* serializer) const {
  for (auto& it : state_block->VkInstances) {
    std::map<uint32_t, VkPhysicalDevice> pd;
    for (auto i : state_block->VkPhysicalDevices) {
      if (i.second->instance == it.first) {
        GAPID2_ASSERT(pd.insert(std::make_pair(i.second->physical_device_idx,
                                               i.second->_handle))
                              .second == true,
                      "Same device used twice for the same instance");
      }
    }

    std::vector<VkPhysicalDevice> pds;
    pds.reserve(pd.size());
    for (auto& i : pd) {
      pds.push_back(i.second);
    }
    GAPID2_ASSERT(pds.size() == pd.size(),
                  "Lost a physical device");
    uint32_t i = pds.size();

    serializer->vkEnumeratePhysicalDevices(
        it.first, &i, nullptr);

    serializer->vkEnumeratePhysicalDevices(
        it.first, &i, pds.data());
  }
}

}  // namespace gapid2