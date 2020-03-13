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

#ifndef __CORE_VULKAN_VK_API_TIMING_LAYER_CC_VK_API_EMITTER_H__
#define __CORE_VULKAN_VK_API_TIMING_LAYER_CC_VK_API_EMITTER_H__

#include <unordered_map>
#include "core/vulkan/perfetto_producer/perfetto_data_source.h"
#include "core/vulkan/perfetto_producer/perfetto_threadlocal_emitter.h"

namespace vk_api {

template <typename T>
class VkApiEmitter : ThreadlocalEmitterBase {
 public:
  VkApiEmitter();
  ~VkApiEmitter();

  void StartTracing() override;
  void SetupTracing(
      const typename perfetto::DataSourceBase::SetupArgs&) override{};
  void StopTracing() override;
  void EmitDebugUtilsObjectName(uint64_t vk_device, int32_t object_type,
                                uint64_t handle, const char* name);

 private:
  void EmitDebugPacket(uint64_t vk_device, int32_t object_type, uint64_t handle,
                       const char* name);

  class DebugMarker {
   public:
    DebugMarker() = default;
    DebugMarker(uint64_t vk_device, int32_t object_type, uint64_t handle,
                std::string name)
        : vk_device_(vk_device),
          object_type_(object_type),
          handle_(handle),
          name_(name){};

    uint64_t vk_device_;
    int32_t object_type_;
    uint64_t handle_;
    std::string name_;
  };

  struct PairHash {
    inline size_t operator()(const std::pair<int32_t, uint64_t>& val) const {
      return val.first ^ val.second;
    }
  };

  std::unordered_map<std::pair<int32_t, uint64_t>, DebugMarker, PairHash>
      debug_markers_;
};

namespace tracing {

template <typename T>
VkApiEmitter<T>& Emit() {
  thread_local VkApiEmitter<T> emitter{};
  return emitter;
}
}  // namespace tracing

struct VkApiTypeTraits {
  static constexpr const char* producer_name = "VulkanAPI";
};

using VkApiProducer = VkApiEmitter<VkApiTypeTraits>;
auto const VkApiEmit = &vk_api::tracing::Emit<VkApiTypeTraits>;
}  // namespace vk_api

#define __INCLUDING_VK_API_EMITTER_INC__
#include "core/vulkan/vk_debug_marker_layer/cc/vk_api_emitter.inc"
#undef __INCLUDING_VK_API_EMITTER_INC__

PERFETTO_DECLARE_DATA_SOURCE_STATIC_MEMBERS(vk_api::VkApiProducer);
#endif
