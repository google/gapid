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

void mid_execution_generator::capture_physical_devices(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  serializer->insert_annotation("MecPhysicalDevices");
  for (auto& it : state_block->VkInstances) {
    std::map<uint32_t, VkPhysicalDevice> pd;
    for (auto i : state_block->VkPhysicalDevices) {
      if (i.second.second->instance == it.first) {
        GAPID2_ASSERT(pd.insert(std::make_pair(i.second.second->physical_device_idx,
                                               i.second.second->_handle))
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
    uint32_t ct = pds.size();

    serializer->vkEnumeratePhysicalDevices(
        it.first, &ct, nullptr);

    serializer->vkEnumeratePhysicalDevices(
        it.first, &ct, pds.data());

    auto enc = serializer->get_encoder(reinterpret_cast<uintptr_t>(it.first));
    if (enc) {
      for (size_t i = 0; i < ct; ++i) {
        VkPhysicalDeviceProperties properties;
        // Bypass serializing the call to GPDP
        bypass_caller->vkGetPhysicalDeviceProperties(pds[i], &properties);
        enc->encode<uint32_t>(properties.deviceID);
        enc->encode<uint32_t>(properties.vendorID);
        enc->encode<uint32_t>(properties.driverVersion);
      }
    }
  }
}

}  // namespace gapid2