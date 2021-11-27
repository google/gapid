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
#include <unordered_map>

namespace gapid2 {
template <BOOL HAS_DISPATCH>
struct HandleRunner {
  static const bool has_dispatch = HAS_DISPATCH;
  std::deque<uint64_t> tbd_handles;

  template <typename T>
  void register_handle(T* value, uint64_t ct) {
    if (!value) {
      return;
    }
    for (size_t i = 0; i < ct; ++i) {
      tbd_handles.push_back(reinterpret_cast<uint64_t>(value[i]));
    }
  }
  template <typename T>
  T cast_out(typename handle_type<HandleRunner, T>::type* t) {}
  template <typename T>
  void register_handle(T value, uint32_t* ct) {
    return register_handle(value, *ct);
  }

  template <typename P, typename T>
  void fixup_dispatch(P, T&) {
    return;
  }

  inline void register_handle_from_struct(
      VkPhysicalDeviceGroupProperties* pPhysicalDeviceGroupProperties,
      uint32_t* pPhysicalDeviceGroupCount) {
    if (pPhysicalDeviceGroupProperties) {
      for (uint32_t i = 0; i < *pPhysicalDeviceGroupCount; ++i) {
        register_handle<VkPhysicalDevice>(
            pPhysicalDeviceGroupProperties[i].physicalDevices,
            pPhysicalDeviceGroupProperties[i].physicalDeviceCount);
      }
    }
  }
#define PROCESS_HANDLE(Type)                                           \
  std::unordered_map<Type, Type##Wrapper<HandleRunner>*> Type##s_out_; \
                                                                       \
  Type##Wrapper<HandleRunner>* cast_from_vk(Type t) {                  \
    auto it = Type##s_out_.find(t);                                    \
    if (it == Type##s_out_.end()) {                                    \
      GAPID2_ERROR("Could not find " #Type);                           \
    }                                                                  \
    return it->second;                                                 \
  }                                                                    \
  template <>                                                          \
  Type cast_out<Type>(Type##Wrapper<HandleRunner> * t) {               \
    auto i = reinterpret_cast<Type>(tbd_handles.front());              \
    tbd_handles.pop_front();                                           \
    Type##s_out_[i] = t;                                               \
    return i;                                                          \
  }                                                                    \
  Type cast_in(Type t) {                                               \
    if (!t) {                                                          \
      return static_cast<Type>(VK_NULL_HANDLE);                        \
    }                                                                  \
    auto i = Type##s_out_[t];                                          \
    return i->_handle;                                                 \
  }
#include "handle_defines.inl"
#undef PROCESS_HANDLE
};
}  // namespace gapid2
