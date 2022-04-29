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
#define PROCESS_HANDLE(Type)                                                         \
  std::unordered_map<Type, Type> Type##_map;                                         \
  void fix_handle(const Type* t) {                                                   \
    if (!*t) {                                                                       \
      return;                                                                        \
    }                                                                                \
    auto it = Type##_map.find(*t);                                                   \
    GAPID2_ASSERT(it != Type##_map.end(), "Cannot find handle to fix");              \
    *const_cast<Type*>(t) = it->second;                                              \
  }                                                                                  \
                                                                                     \
  std::unordered_map<Type*, Type> Type##_registered_handles_;                        \
  void register_handle(Type* t) {                                                    \
    if (t[0] == VK_NULL_HANDLE) {                                                    \
      t[0] = reinterpret_cast<Type>(next_unassigned_handle--);                       \
    }                                                                                \
    Type##_registered_handles_[t] = t[0];                                            \
  }                                                                                  \
  void process_handle(Type* t) {                                                     \
    auto it = Type##_registered_handles_.find(t);                                    \
    GAPID2_ASSERT(it != Type##_registered_handles_.end(), "Unknown handle address"); \
    Type##_map[it->second] = t[0];                                                   \
    t[0] = it->second;                                                               \
    Type##_registered_handles_.erase(it);                                            \
  }
#include "handle_defines.inl"
#undef PROCESS_HANDLE

  void ensure_clean() {
#define PROCESS_HANDLE(Type) \
  GAPID2_ASSERT(Type##_registered_handles_.empty(), "Unassigned handle");
#include "handle_defines.inl"
#undef PROCESS_HANDLE
  }

  uintptr_t next_unassigned_handle = -2;
};
}  // namespace gapid2