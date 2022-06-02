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
  mObserver->on_slice_encoded([&](const gapil::Pool* pool) {
    if (pool != nullptr && mSeenPools.count(pool->id()) == 0) {
      mSeenPools.insert(pool->id());

      memory::Observation observation;
      observation.set_pool(pool->id());
      observation.set_base(0);
      sendData(&observation, true, pool->buffer(), pool->size());
    }
  });
}

gapil::Pool* StateSerializer::createPool(
    uint64_t pool_size,
    std::function<void(memory::Observation*)> init_observation) {
  auto arena = mSpy->arena();
  auto pool = arena->create<gapil::Pool>(arena, mSpy->nextPoolID(), pool_size);

  mSeenPools.insert(pool->id());

  memory::Observation observation;
  observation.set_pool(pool->id());
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
