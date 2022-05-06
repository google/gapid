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

#include "instance.h"

#include "mid_execution_generator.h"
#include "state_block.h"
#include "queue.h"
#include "utils.h"

namespace gapid2 {

void mid_execution_generator::capture_queues(const state_block* state_block, noop_serializer* serializer) const {
  for (auto& it : state_block->VkQueues) {
    VkQueue queue = it.first;
    if (it.second->get_info_2()) {
      serializer->vkGetDeviceQueue2(
          it.second->device,
          it.second->get_info_2(),
          &queue);
    } else {
      serializer->vkGetDeviceQueue(
        it.second->device,
        it.second->queue_family_index,
        it.second->queue_index,
        &queue
      );
    }
  }
}

}  // namespace gapid2