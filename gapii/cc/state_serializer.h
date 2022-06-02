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

#ifndef GAPII_STATE_SERIALIZER_H
#define GAPII_STATE_SERIALIZER_H

#include "gapii/cc/call_observer.h"
#include "gapii/cc/spy_base.h"
#include "gapil/runtime/cc/pool.h"
#include "gapil/runtime/cc/slice.h"
#include "gapis/memory/memory_pb/memory.pb.h"

#include <functional>

namespace gapii {

class StateSerializer {
 public:
  StateSerializer(SpyBase* spy, uint8_t api, CallObserver* observer)
      : mSpy(spy), mApi(api), mObserver(observer) {}

  ~StateSerializer();

  // Serializes the given state to the underlying CallObserver.
  template <typename T, typename = CallObserver::enable_if_encodable<T>>
  inline void encodeState(
      const T& state, std::function<void(StateSerializer*)> serialize_buffers);

  // Creates and returns a new slice backed by a new virtual pool of the given
  // size. A memory observation is attached to the underlying CallObserver. If
  // init_observation is non-null, it is called to construct the observation,
  // otherwhise an empty observation is used.
  template <typename T>
  inline void encodeBuffer(
      uint64_t pool_size, gapil::Slice<T>* dest,
      std::function<void(memory::Observation*)> init_observation);

  // Encodes the given data using the given observation. If sendObservation is
  // true, the observation itself is also encoded on the underlying
  // CallObserver, otherwhise the observation is simply updated  with the data
  // resource pointers.
  inline void sendData(memory::Observation* observation, bool sendObservation,
                       const void* data, size_t size);

 private:
  void prepareForState(std::function<void(StateSerializer*)> serialize_buffers);
  gapil::Pool* createPool(
      uint64_t pool_size,
      std::function<void(memory::Observation*)> init_observation);

  SpyBase* mSpy;
  uint8_t mApi;
  CallObserver* mObserver;
  std::unordered_set<uint32_t> mSeenPools;
  std::vector<std::function<void()>> mCleanup;
  int64_t mEmptyIndex = -1;
};

template <typename T, typename /* = CallObserver::enable_if_encodable<T> */>
inline void StateSerializer::encodeState(
    const T& state, std::function<void(StateSerializer*)> serialize_buffers) {
  prepareForState(serialize_buffers);
  mObserver->encode(state);
  mObserver->on_slice_encoded(nullptr);
}

template <typename T>
inline void StateSerializer::encodeBuffer(
    uint64_t pool_size, gapil::Slice<T>* dest,
    std::function<void(memory::Observation*)> init_observation) {
  *dest =
      gapil::Slice<T>::create(createPool(pool_size, init_observation), false);
  mCleanup.push_back(
      std::function<void()>([dest]() { *dest = gapil::Slice<T>(); }));
}

inline void StateSerializer::sendData(memory::Observation* observation,
                                      bool sendObservation, const void* data,
                                      size_t size) {
  auto index = mSpy->sendResource(mApi, data, size);
  observation->set_size(size);
  observation->set_res_index(index);

  if (sendObservation) {
    mObserver->encode_message(observation);
  }
}

inline StateSerializer::~StateSerializer() {
  for (auto& c : mCleanup) {
    c();
  }
}

}  // namespace gapii

#endif  // GAPII_STATE_SERIALIZER_H
