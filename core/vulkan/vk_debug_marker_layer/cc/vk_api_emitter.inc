/*
 * Copyright (C) 2019 Google Inc.
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
 *
 */

#include "core/vulkan/perfetto_producer/perfetto_data_source.h"
#include "perfetto/tracing/core/data_source_config.h"
#include "perfetto/base/time.h"

namespace vk_api {

template <typename T>
VkApiEmitter<T>::VkApiEmitter() {
  core::PerfettoProducer<T>::Get().RegisterEmitter(this);
}

template <typename T>
VkApiEmitter<T>::~VkApiEmitter() {
  core::PerfettoProducer<T>::Get().UnregisterEmitter(this);
}

template <typename T>
void VkApiEmitter<T>::StartTracing() {
  for (auto entry : debug_markers_) {
    auto marker = entry.second;
    EmitDebugPacket(marker.vk_device_, marker.object_type_, marker.handle_, marker.name_.c_str());
  }
}

template <typename T>
void VkApiEmitter<T>::StopTracing() {
  core::PerfettoProducer<T>::Trace(
      [](typename core::PerfettoProducer<T>::TraceContext ctx) {
        ctx.Flush();
      });
}

template <typename T>
void VkApiEmitter<T>::EmitDebugUtilsObjectName(uint64_t vk_device, int32_t object_type,
                                               uint64_t handle, const char* name) {
  debug_markers_[std::make_pair(object_type, handle)] = DebugMarker(vk_device, object_type, handle, name);
  EmitDebugPacket(vk_device, object_type, handle, name);
}

template <typename T>
void VkApiEmitter<T>::EmitDebugPacket(uint64_t vk_device, int32_t object_type, uint64_t handle,
                                      const char* name) {
  core::PerfettoProducer<T>::Trace(
      [this, vk_device, object_type, handle, name](
          typename core::PerfettoProducer<T>::TraceContext ctx) {
        auto data_source = ctx.GetDataSourceLocked();
        uint64_t time = perfetto::base::GetBootTimeNs().count();
        auto packet = ctx.NewTracePacket();
        packet->set_timestamp(time);
        auto event = packet->set_vulkan_api_event()->set_vk_debug_utils_object_name();
        event->set_vk_device(vk_device);
        event->set_object_type(object_type);
        event->set_object(handle);
        event->set_object_name(name);
      });
}

}
