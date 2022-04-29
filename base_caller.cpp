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

#include "base_caller.h"

#include "common.h"

namespace gapid2 {

void base_caller::on_instance_created(const VkInstanceCreateInfo* create_info, VkInstance* val, uint32_t count) {
  if (!val) {
    return;
  }
  instance_lock_.lock();
  for (uint32_t i = 0; i < count; ++i) {
    instance_functions_.insert(std::make_pair(val[i], std::make_unique<instance_functions>(val[i], vkGetInstanceProcAddr_)));
  }
  instance_lock_.unlock();
}

void base_caller::on_physicaldevice_created(VkInstance instance, VkPhysicalDevice* val, uint32_t count) {
  if (!val) {
    return;
  }
  instance_lock_.lock_shared();
  physicaldevice_lock_.lock();
  for (uint32_t i = 0; i < count; ++i) {
    physicaldevice_functions_.insert(std::make_pair(val[i], instance_functions_[instance].get()));
  }
  physicaldevice_lock_.unlock();
  instance_lock_.unlock_shared();
}

void base_caller::on_device_created(VkPhysicalDevice physical_device, VkDevice* val, uint32_t count) {
  if (!val) {
    return;
  }
  instance_lock_.lock_shared();
  physicaldevice_lock_.lock_shared();
  device_lock_.lock();
  for (uint32_t i = 0; i < count; ++i) {
    auto phys_dev_fns = physicaldevice_functions_[physical_device];
    auto inst = std::find_if(instance_functions_.begin(),
                             instance_functions_.end(),
                             [phys_dev_fns](const std::pair<const VkInstance, std::unique_ptr<gapid2::instance_functions>>& it) {
                               return it.second.get() == phys_dev_fns;
                             });
    GAPID2_ASSERT(inst != instance_functions_.end(), "Cannot find instance that created this physical device");
    auto gdpa = reinterpret_cast<PFN_vkGetDeviceProcAddr>(phys_dev_fns->vkGetInstanceProcAddr_(inst->first, "vkGetDeviceProcAddr"));
    if (!gdpa) {
      gdpa = vkGetDeviceProcAddr_;
    }
    device_functions_.insert(std::make_pair(val[i], std::make_unique<device_functions>(val[i], gdpa)));
  }
  device_lock_.unlock();
  physicaldevice_lock_.unlock_shared();
  instance_lock_.unlock_shared();
}

void base_caller::on_queue_created(VkDevice device, VkQueue* val, uint32_t count) {
  if (!val) {
    return;
  }
  device_lock_.lock_shared();
  queue_lock_.lock();
  for (uint32_t i = 0; i < count; ++i) {
    queue_functions_.insert(std::make_pair(val[i], device_functions_[device].get()));
  }
  queue_lock_.unlock();
  device_lock_.unlock_shared();
}

void base_caller::on_commandbuffer_created(VkDevice device, VkCommandBuffer* val, uint32_t count) {
  if (!val) {
    return;
  }
  device_lock_.lock_shared();
  commandbuffer_lock_.lock();
  for (uint32_t i = 0; i < count; ++i) {
    commandbuffer_functions_.insert(std::make_pair(val[i], device_functions_[device].get()));
  }
  commandbuffer_lock_.unlock();
  device_lock_.unlock_shared();
}

void base_caller::on_instance_destroyed(const VkInstance* val, uint32_t count) {
  if (!val) {
    return;
  }
  instance_lock_.lock();
  for (size_t i = 0; i < count; ++i) {
    physicaldevice_lock_.lock();
    auto inst_fns = instance_functions_[val[i]].get();
    for (auto it = physicaldevice_functions_.begin(); it != physicaldevice_functions_.end();) {
      if (it->second == inst_fns) {
        it = physicaldevice_functions_.erase(it);
        continue;
      }
      it++;
    }
    physicaldevice_lock_.unlock();
    instance_functions_.erase(val[i]);
  }
  instance_lock_.unlock();
}

void base_caller::on_physicaldevice_destroyed(const VkPhysicalDevice* val, uint32_t count) {
  if (!val) {
    return;
  }
  physicaldevice_lock_.lock();
  for (size_t i = 0; i < count; ++i) {
    physicaldevice_functions_.erase(val[i]);
  }
  physicaldevice_lock_.unlock();
}

void base_caller::on_device_destroyed(const VkDevice* val, uint32_t count) {
  if (!val) {
    return;
  }
  device_lock_.lock();
  for (size_t i = 0; i < count; ++i) {
    queue_lock_.lock();
    auto dev_fns = device_functions_[val[i]].get();
    for (auto it = queue_functions_.begin(); it != queue_functions_.end();) {
      if (it->second == dev_fns) {
        it = queue_functions_.erase(it);
        continue;
      }
      it++;
    }
    queue_lock_.unlock();
    device_functions_.erase(val[i]);
  }
  device_lock_.unlock();
}

void base_caller::on_queue_destroyed(const VkQueue* val, uint32_t count) {
  if (!val) {
    return;
  }
  queue_lock_.lock();
  for (size_t i = 0; i < count; ++i) {
    queue_functions_.erase(val[i]);
  }
  queue_lock_.unlock();
}

void base_caller::on_commandbuffer_destroyed(const VkCommandBuffer* val, uint32_t count) {
  if (!val) {
    return;
  }
  commandbuffer_lock_.lock();
  for (size_t i = 0; i < count; ++i) {
    commandbuffer_functions_.erase(val[i]);
  }
  commandbuffer_lock_.unlock();
}

}  // namespace gapid2