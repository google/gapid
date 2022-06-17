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

#include "event.h"
#include "fence.h"
#include "mid_execution_generator.h"
#include "semaphore.h"
#include "state_block.h"
#include "utils.h"

namespace gapid2 {

void mid_execution_generator::capture_synchronization(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  serializer->insert_annotation("MecSemaphores");
  for (auto& it : state_block->VkSemaphores) {
    VkSemaphoreWrapper* sem = it.second.second;
    VkSemaphore semaphore = it.first;
    serializer->vkCreateSemaphore(sem->device, sem->get_create_info(), nullptr, &semaphore);
    auto tp = get_pNext<VkSemaphoreTypeCreateInfo>(sem->get_create_info());
    if (tp && tp->semaphoreType == VK_SEMAPHORE_TYPE_TIMELINE) {
      GAPID2_ASSERT(false, "Timeline semaphores not quite ready yet");
    }
    if (sem->value) {
      auto q = get_queue_for_device(state_block, sem->device);
      GAPID2_ASSERT(q, "Cannot find queue for device .. how?");
      VkSubmitInfo sub_info{
          .sType = VK_STRUCTURE_TYPE_SUBMIT_INFO,
          .pNext = nullptr,
          .waitSemaphoreCount = 0,
          .pWaitSemaphores = nullptr,
          .commandBufferCount = 0,
          .pCommandBuffers = nullptr,
          .signalSemaphoreCount = 1,
          .pSignalSemaphores = &semaphore};
      serializer->vkQueueSubmit(q, 1, &sub_info, 0);
    }
  }
  serializer->insert_annotation("MecFences");
  for (auto& it : state_block->VkFences) {
    VkFenceWrapper* f = it.second.second;
    VkFence fence = it.first;
    auto status = bypass_caller->vkGetFenceStatus(f->device, fence);
    VkFenceCreateInfo ci = *f->get_create_info();
    ci.flags = status == VK_SUCCESS ? VK_FENCE_CREATE_SIGNALED_BIT : 0;
    serializer->vkCreateFence(f->device, &ci, nullptr, &fence);
  }
  serializer->insert_annotation("MecEvents");
  for (auto& it : state_block->VkEvents) {
    VkEventWrapper* evt = it.second.second;
    VkEvent event = it.first;
    VkEventCreateInfo eci = *evt->get_create_info();

    serializer->vkCreateEvent(evt->device, &eci, nullptr, &event);

    auto status = bypass_caller->vkGetEventStatus(evt->device, event);
    if (status == VK_EVENT_SET) {
      bypass_caller->vkSetEvent(evt->device, event);
    }
  }
#pragma TODO(awoloszyn, We should set up any pending things on any queues)
}

}  // namespace gapid2