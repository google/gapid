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

void mid_execution_generator::capture_synchronization(const state_block* state_block, noop_serializer* serializer) const {
  for (auto& it : state_block->VkSemaphores) {
    VkSemaphoreWrapper* sem = it.second;
    VkSemaphore semaphore = it.first;
    serializer->vkCreateSemaphore(sem->device, sem->get_create_info(), nullptr, &semaphore);
  }
  for (auto& it : state_block->VkFences) {
    VkFenceWrapper* f = it.second;
    VkFence fence = it.first;
    serializer->vkCreateFence(f->device, f->get_create_info(), nullptr, &fence);
  }
  for (auto& it : state_block->VkEvents) {
    VkEventWrapper* evt = it.second;
    VkEvent event = it.first;
    serializer->vkCreateEvent(evt->device, evt->get_create_info(), nullptr, &event);
  }
#pragma TODO(awoloszyn, Get the actual synchronization states here)
#pragma TODO(awoloszyn, Also we should set up any pending things on any queues)
}

}  // namespace gapid2