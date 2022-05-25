/*
 * Copyright (C) 2018 Google Inc.
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

#include "state_serializer.h"
#include "core/cc/timer.h"

namespace gapii {

void StateSerializer::prepareForState(
    std::function<void(StateSerializer*)> serialize_buffers) {
  capture::GlobalState global;
  mObserver->enter(&global);

  serialize_buffers(this);
  mObserver->on_slice_encoded([&](const slice_t* slice) {
    auto p = slice->pool;
    if (p != nullptr && mSeenPools.count(p->id) == 0) {
      mSeenPools.insert(p->id);

      memory::Observation observation;
      observation.set_pool(p->id);
      observation.set_base(0);
      sendData(&observation, true, p->buffer, p->size);
    }
  });
}

pool_t* StateSerializer::createPool(
    uint64_t pool_size,
    std::function<void(memory::Observation*)> init_observation) {
  auto arena = mSpy->arena();
  auto pool = arena->create<pool_t>();
  pool->arena = reinterpret_cast<arena_t*>(arena);
  pool->id = (*mObserver->next_pool_id)++;
  pool->size = pool_size;
  pool->ref_count = 1;
  pool->buffer = nullptr;

  mSeenPools.insert(pool->id);

  memory::Observation observation;
  observation.set_pool(pool->id);
  observation.set_base(0);
  if (init_observation != nullptr) {
    init_observation(&observation);
  } else {
    if (mEmptyIndex < 0) {
      char empty = 0;
      mEmptyIndex = mSpy->sendResource(mApi, &empty, 0);
    }

    observation.set_size(0);
    observation.set_res_index(mEmptyIndex);
  }
  mObserver->encode_message(&observation);
  return pool;
}

}  // namespace gapii
