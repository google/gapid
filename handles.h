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

#pragma once
#include <atomic>
#include <deque>
#include <mutex>
#include <unordered_set>

#include "common.h"
#include "handle_templates.h"

namespace gapid2 {
template <typename HandleUpdater, typename T>
struct handle_type {
  const bool is_handle = false;
};

struct dummy {};

struct base_handle {
  ~base_handle() {
    invalidate();
    std::unordered_set<base_handle*> i;
    {
      std::unique_lock<std::mutex> l(invalidation_mutex);
      i = std::move(invalidations_by);
    }

    for (auto& x : i) {
      x->no_longer_invalidates(this);
    }
  }

  bool invalidated = false;

  void invalidates(base_handle* other) {
    {
      std::unique_lock<std::mutex> l(invalidation_mutex);
      GAPID2_ASSERT(!invalidated, "Trying to use an invalid handle");
      invalidations.insert(other);
    }
    other->invalidated_by(this);
  }

  void no_longer_invalidates(base_handle* other) {
    std::unique_lock<std::mutex> l(invalidation_mutex);
    invalidations.erase(other);
    other->no_longer_invalidated_by(this);
  }

  void invalidate() {
    invalidated = true;
    std::unordered_set<base_handle*> i;

    {
      std::unique_lock<std::mutex> l(invalidation_mutex);
      i = std::move(invalidations);
    }

    for (auto& x : i) {
      x->no_longer_invalidated_by(this);
      x->invalidate();
    }
  }

  void reset_invalidations() {
    std::unique_lock<std::mutex> l(invalidation_mutex);
    invalidated = false;
    invalidations.clear();
  }

 private:
  void invalidated_by(base_handle* other) {
    std::unique_lock<std::mutex> l(invalidation_mutex);
    invalidations_by.insert(other);
  }
  void no_longer_invalidated_by(base_handle* other) {
    std::unique_lock<std::mutex> l(invalidation_mutex);
    invalidations_by.erase(other);
  }
  std::mutex invalidation_mutex;
  std::unordered_set<base_handle*> invalidations;
  std::unordered_set<base_handle*> invalidations_by;
};

template <typename T, typename DISPATCH = dummy>
struct handle_base : public base_handle {
  handle_base(T t) : _handle(t) {}
  using base_type = T;
  DISPATCH* dispatch = nullptr;
  T _handle;
};

struct HandleWrapperUpdater {
  std::deque<uint64_t> tbd_handles;

  template <typename T>
  void register_handle(T* value, uint64_t ct) {}

  template <typename T>
  void register_handle(T value, uint32_t* ct) {}

  inline void register_handle_from_struct(VkPhysicalDeviceGroupProperties*,
                                          uint32_t*) {}

  static const bool has_dispatch = true;
  template <typename P, typename T>
  typename std::enable_if_t<needs_dispatch_fixup<T>::val> fixup_dispatch(P p,
                                                                         T& t) {
    reinterpret_cast<void**>(t)[0] = reinterpret_cast<void**>(p)[0];
  }

  template <typename P, typename T>
  typename std::enable_if_t<!needs_dispatch_fixup<T>::val> fixup_dispatch(
      P p,
      T& t) {
    return;
  }

  template <typename T>
  typename handle_type<HandleWrapperUpdater, T>::type* cast_from_vk(T t) {
    return reinterpret_cast<
        typename handle_type<HandleWrapperUpdater, T>::type*>(t);
  }

  template <typename T>
  typename T::base_type cast_to_vk(T* t) {
    if (!t) {
      return 0;
    }
    return reinterpret_cast<typename T::base_type>(t->_handle);
  }

  template <typename T>
  T cast_in(T t) {
    return cast_to_vk(cast_from_vk(t));
  }

  template <typename T>
  T cast_out(typename handle_type<HandleWrapperUpdater, T>::type* t) {
    return reinterpret_cast<T>(reinterpret_cast<uintptr_t>(t));
  }
};

}  // namespace gapid2
