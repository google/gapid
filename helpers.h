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
#include <vulkan.h>
#include <cassert>
#include "handles.h"
#include "temporary_allocator.h"
#define _QUOTE(x) #x
#define QUOTE(x) _QUOTE(x)
#define __FILE__LINE__ __FILE__ ":" QUOTE(__LINE__) " : "

#define FIXME(user, ...)                                        \
  message(__FILE__LINE__                                        \
          "\n"                                                  \
          " ------------------------------------------------\n" \
          "|  FIXME(" #user ") :  " #__VA_ARGS__                \
          "\n"                                                  \
          " -------------------------------------------------\n")

#include "buffer.h"
#include "buffer_view.h"
#include "command_buffer.h"
#include "command_pool.h"
#include "common.h"
#include "descriptor_pool.h"
#include "descriptor_set.h"
#include "descriptor_set_layout.h"
#include "descriptor_update_template.h"
#include "device.h"
#include "device_memory.h"
#include "event.h"
#include "fence.h"
#include "framebuffer.h"
#include "handles.h"
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

#define PROCESS_HANDLE(Type)                   \
  template <typename HandleUpdater>            \
  struct handle_type<HandleUpdater, Type> {    \
    using type = Type##Wrapper<HandleUpdater>; \
    const bool is_handle = true;               \
  };
#include "handle_defines.inl"
#undef PROCESS_HANDLE

template <typename HandleUpdater, typename T, typename... Args>
T* clone_struct(HandleUpdater* _updater,
                const T* t,
                const size_t _num,
                temporary_allocator* mem,
                Args... args) {
  if (!t || !_num) {
    return nullptr;
  }
  T* nt = mem->get_typed_memory<T>(_num);
  for (size_t i = 0; i < _num; ++i) {
    clone(_updater, t[i], nt[i], mem, std::forward<Args>(args)...);
  }
  return nt;
}
template <typename HandleUpdater, typename T, typename... Args>
T* clone_struct(HandleUpdater* _updater,
                const T* t,
                uint32_t* _num,
                temporary_allocator* mem,
                Args... args) {
  return clone_struct(_updater, t, static_cast<size_t>(*_num), mem,
                      std::forward<Args>(args)...);
}
template <typename HandleUpdater, typename T, typename... Args>
T* clone_handle(HandleUpdater* _updater,
                const T* t,
                const size_t _num,
                temporary_allocator* mem) {
  T* nt = mem->get_typed_memory<T>(_num);

  for (size_t i = 0; i < _num; ++i) {
    nt[i] = _updater->cast_in(t[i]);
  }
  return nt;
}

template <typename HandleUpdater, typename T, typename... Args>
T* clone_handle(HandleUpdater* _updater,
                const T* t,
                const uint32_t* _num,
                temporary_allocator* mem) {
  return clone_handle(_updater, t, static_cast<size_t>(*_num), mem);
}

template <typename HandleUpdater, typename P, typename T, typename RT>
void create_handle(HandleUpdater* _updater, P p, T* t, size_t num) {
  if (!t) {
    return;
  }
  auto p_ptr = _updater->cast_from_vk(p);

  for (size_t i = 0; i < num; ++i) {
    _updater->fixup_dispatch<P, T>(p, t[i]);
    auto ti = reinterpret_cast<typename handle_type<HandleUpdater, T>::type*>(
        p_ptr->get_and_increment_child(t[i]));
    if (ti) {
      t[i] = _updater->template cast_out<T>(ti);
      continue;
    }
    auto ni = new RT(_updater, p, t[i]);
    p_ptr->add_child(t[i], ni);
    t[i] = _updater->template cast_out<T>(ni);
  }
}

template <typename HandleUpdater, typename P, typename T, typename RT>
void create_handle(HandleUpdater* _updater, P p, T* t, uint32_t* num) {
  if (!t) {
    return;
  }
  return create_handle<HandleUpdater, P, T, RT>(_updater, p, t,
                                                static_cast<size_t>(*num));
}

template <typename HandleUpdater, typename RT>
void create_instance(HandleUpdater* _updater, VkInstance* i) {
  auto ni = new RT(*i);
  *i = _updater->template cast_out<VkInstance>(ni);
}

template <typename HandleUpdater>
inline void create_handle_from_struct(
    HandleUpdater* _updater,
    VkInstance instance,
    VkPhysicalDeviceGroupProperties* pPhysicalDeviceGroupProperties,
    uint32_t* pPhysicalDeviceGroupCount) {
  if (pPhysicalDeviceGroupProperties) {
    for (uint32_t i = 0; i < *pPhysicalDeviceGroupCount; ++i) {
      create_handle<HandleUpdater, VkInstance, VkPhysicalDevice,
                    VkPhysicalDeviceWrapper<HandleUpdater>>(
          _updater, instance, pPhysicalDeviceGroupProperties[i].physicalDevices,
          pPhysicalDeviceGroupProperties[i].physicalDeviceCount);
    }
  }
}
class encoder;
class decoder;

}  // namespace gapid2