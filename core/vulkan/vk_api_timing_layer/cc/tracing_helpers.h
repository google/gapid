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

#ifndef __CORE_VULKAN_VK_API_TIMING_LAYER_CC_TRACING_HELPERS_H__
#define __CORE_VULKAN_VK_API_TIMING_LAYER_CC_TRACING_HELPERS_H__

#include "core/vulkan/perfetto_producer/perfetto_data_source.h"
#include "core/vulkan/perfetto_producer/perfetto_threadlocal_emitter.h"

namespace api_timing {

struct TimingTypeTraits {
  static constexpr const char* producer_name = "VulkanCPUTiming";
};

using Producer = core::PerfettoProducer<TimingTypeTraits>;
auto const Emit = &core::tracing::Emit<TimingTypeTraits>;
}  // namespace api_timing

PERFETTO_DECLARE_DATA_SOURCE_STATIC_MEMBERS(api_timing::Producer);
#endif
