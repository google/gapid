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

#include "state_block.h"

#include "buffer.h"
#include "buffer_view.h"
#include "command_buffer.h"
#include "command_pool.h"
#include "descriptor_pool.h"
#include "descriptor_set.h"
#include "descriptor_set_layout.h"
#include "descriptor_update_template.h"
#include "device.h"
#include "device_memory.h"
#include "event.h"
#include "fence.h"
#include "framebuffer.h"
#include "image.h"
#include "image_view.h"
#include "instance.h"
#include "physical_device.h"
#include "pipeline.h"
#include "pipeline_cache.h"
#include "pipeline_layout.h"
#include "query_pool.h"
#include "queue.h"
#include "render_pass.h"
#include "sampler.h"
#include "sampler_ycbcr_conversion.h"
#include "semaphore.h"
#include "shader_module.h"
#include "surface.h"
#include "swapchain.h"

namespace gapid2 {
state_block::~state_block() {
}

#define PROCESS_HANDLE(Type)                                               \
  std::shared_ptr<Type##Wrapper> state_block::create(Type t) {             \
    Type##mut.lock_shared();                                               \
    auto it = Type##s.find(t);                                             \
    if (it != Type##s.end()) {                                             \
      Type##mut.unlock_shared();                                           \
      return nullptr;                                                      \
    }                                                                      \
    Type##mut.unlock_shared();                                             \
    Type##mut.lock();                                                      \
    auto ret = std::make_shared<Type##Wrapper>(t);                         \
    Type##s[t] = std::make_pair(1, ret);                                   \
    Type##mut.unlock();                                                    \
    return ret;                                                            \
  }                                                                        \
  std::shared_ptr<Type##Wrapper> state_block::get_or_create(Type t) {      \
    Type##mut.lock_shared();                                               \
    auto it = Type##s.find(t);                                             \
    if (it != Type##s.end()) {                                             \
      auto i = it->second.second;                                          \
      Type##mut.unlock_shared();                                           \
      return i;                                                            \
    }                                                                      \
    Type##mut.unlock_shared();                                             \
    auto ret = std::make_shared<Type##Wrapper>(t);                         \
    Type##mut.lock();                                                      \
    Type##s[t] = std::make_pair(1, ret);                                   \
    Type##mut.unlock();                                                    \
    return ret;                                                            \
  }                                                                        \
  std::shared_ptr<Type##Wrapper> state_block::get(Type t) {                \
    Type##mut.lock_shared();                                               \
    auto it = Type##s.find(t);                                             \
    if (it != Type##s.end()) {                                             \
      auto i = it->second.second;                                          \
      Type##mut.unlock_shared();                                           \
      return i;                                                            \
    }                                                                      \
    Type##mut.unlock_shared();                                             \
    return nullptr;                                                        \
  }                                                                        \
  const Type##Wrapper* state_block::get(Type t) const {                    \
    Type##mut.lock_shared();                                               \
    auto it = Type##s.find(t);                                             \
    if (it != Type##s.end()) {                                             \
      auto i = it->second.second;                                          \
      Type##mut.unlock_shared();                                           \
      return i.get();                                                      \
    }                                                                      \
    Type##mut.unlock_shared();                                             \
    return nullptr;                                                        \
  }                                                                        \
  bool state_block::erase(Type t) {                                        \
    Type##mut.lock();                                                      \
    auto it = Type##s.find(t);                                             \
    if (it == Type##s.end()) {                                             \
      Type##mut.unlock();                                                  \
      return false;                                                        \
    }                                                                      \
    if (!--it->second.first) {                                             \
      it->second.second->invalidate();                                     \
      Type##s.erase(it);                                                   \
    }                                                                      \
    Type##mut.unlock();                                                    \
    return true;                                                           \
  }                                                                        \
  void state_block::erase_if(std::function<bool(Type##Wrapper * w)> fun) { \
    Type##mut.lock();                                                      \
    for (auto it = Type##s.begin(); it != Type##s.end();) {                \
      if (fun(it->second.second.get())) {                                  \
        if (0 == --it->second.first) {                                     \
          it->second.second->invalidate();                                 \
          it = Type##s.erase(it);                                          \
          continue;                                                        \
        }                                                                  \
      }                                                                    \
      it++;                                                                \
    }                                                                      \
    Type##mut.unlock();                                                    \
  }

#include "handle_defines.inl"
#undef PROCESS_HANDLE

}  // namespace gapid2