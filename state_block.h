#pragma once
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

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include <functional>
#include <memory>
#include <shared_mutex>
#include <unordered_map>

#include "transform_base.h"

namespace gapid2 {
#define PROCESS_HANDLE(Type) \
  struct Type##Wrapper;
#include "handle_defines.inl"
#undef PROCESS_HANDLE

class state_block : public transform_base {
 public:
  ~state_block();

#define PROCESS_HANDLE(Type)                                                             \
  std::shared_ptr<Type##Wrapper> get_or_create(Type t);                                  \
  std::shared_ptr<Type##Wrapper> create(Type t);                                         \
  std::shared_ptr<Type##Wrapper> get(Type t);                                            \
  const Type##Wrapper* get(Type t) const;                                                \
  bool erase(Type t);                                                                    \
  void erase_if(std::function<bool(Type##Wrapper * w)> fun);                             \
  mutable std::shared_mutex Type##mut;                                                   \
  std::unordered_map<Type, std::pair<uint64_t, std::shared_ptr<Type##Wrapper>>> Type##s; \
  Type get_unused_##Type() const {                                                       \
    Type t;                                                                              \
    do {                                                                                 \
      t = reinterpret_cast<Type>(static_cast<uintptr_t>(std::rand()));                   \
    } while (Type##s.contains(t));                                                       \
    return t;                                                                            \
  }

#include "handle_defines.inl"
#undef PROCESS_HANDLE
};
}  // namespace gapid2