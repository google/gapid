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
#include "handle_templates.h"

namespace gapid2 {
template <typename HandleUpdater, typename T>
struct handle_type {
  const bool is_handle = false;
};

struct dummy {};
template <typename T, typename DISPATCH = dummy>
struct handle_base {
  handle_base(T t) : _handle(t) {}
  using base_type = T;
  DISPATCH* dispatch = nullptr;
  T _handle;
};

struct HandleWrapperUpdater {
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
