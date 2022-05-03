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
#include <unordered_map>

#include "common.h"

namespace gapid2 {
class handle_fixer {
 public:
#define PROCESS_HANDLE(Type)                                                                                             \
  std::unordered_map<Type, Type> Type##_map;                                                                             \
  void fix_handle(const Type* t) {                                                                                       \
    if (!*t) {                                                                                                           \
      return;                                                                                                            \
    }                                                                                                                    \
    if constexpr (sizeof(Type) == sizeof(uint32_t)) {                                                                              \
      uint32_t p = *reinterpret_cast<const uint32_t*>(t);\
      fixed_handles_32.push_back(std::make_pair<void*, const uint32_t&>(const_cast<Type*>(t), p)); \
    } else {                                                    \
      uint64_t p = *reinterpret_cast<const uint64_t*>(t);\
      fixed_handles_64.push_back(std::make_pair<void*, const uint64_t&>(const_cast<Type*>(t), p)); \
    }                                                                                                                    \
                                                                                                                         \
    auto it = Type##_map.find(*t);                                                                                       \
    GAPID2_ASSERT(it != Type##_map.end(), "Cannot find handle to fix");                                                  \
    *const_cast<Type*>(t) = it->second;                                                                                  \
  }                                                                                                                      \
                                                                                                                         \
  std::unordered_map<Type*, Type>                                                                                        \
      Type##_registered_handles_;                                                                                        \
  void register_handle(Type* t) {                                                                                        \
    if (t[0] == VK_NULL_HANDLE) {                                                                                        \
      t[0] = reinterpret_cast<Type>(next_unassigned_handle--);                                                           \
    }                                                                                                                    \
    Type##_registered_handles_[t] = t[0];                                                                                \
  }                                                                                                                      \
  void process_handle(Type* t) {                                                                                         \
    auto it = Type##_registered_handles_.find(t);                                                                        \
    GAPID2_ASSERT(it != Type##_registered_handles_.end(), "Unknown handle address");                                     \
    Type##_map[it->second] = t[0];                                                                                       \
    t[0] = it->second;                                                                                                   \
    Type##_registered_handles_.erase(it);                                                                                \
  }
#include "handle_defines.inl"
#undef PROCESS_HANDLE

  std::vector<std::pair<void*, uint32_t>> fixed_handles_32;
  std::vector<std::pair<void*, uint64_t>> fixed_handles_64;
  void ensure_clean() {
#define PROCESS_HANDLE(Type) \
  GAPID2_ASSERT(Type##_registered_handles_.empty(), "Unassigned handle");
#include "handle_defines.inl"
#undef PROCESS_HANDLE
  }

  void undo_handles() {
    for (auto& x : fixed_handles_32) {
      *reinterpret_cast<uint32_t*>(x.first) = x.second;
    }
    fixed_handles_32.clear();
    for (auto& x : fixed_handles_64) {
      *reinterpret_cast<uint64_t*>(x.first) = x.second;
    }
    fixed_handles_64.clear();
  }

  uintptr_t next_unassigned_handle = -2;
};
}  // namespace gapid2