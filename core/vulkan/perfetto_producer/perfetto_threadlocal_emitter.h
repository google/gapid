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

#ifndef __PERFETTO_THREADLOCAL_EMITTER_H__
#define __PERFETTO_THREADLOCAL_EMITTER_H__
#include <atomic>
#include "core/memory/arena/cc/arena.h"
#include "core/vulkan/perfetto_producer/perfetto_proto_structs.h"
#include "core/vulkan/perfetto_producer/threadlocal_emitter_base.h"
#include "gapil/runtime/cc/map.h"

namespace gapil {

template <>
struct hash<std::string, void> {
  uint64_t operator()(const std::string& t) {
    return std::hash<std::string>()(t);
    return 0;
  }
};
}  // namespace gapil

namespace core {

const uint32_t SEQUENCE_RESET_PERIOD_MS = 2000;

template <typename T>
class ThreadlocalEmitter : ThreadlocalEmitterBase {
 public:
  ThreadlocalEmitter();
  ~ThreadlocalEmitter();

  void StartTracing() override {
    reset_ = true;
    enabled_ = true;
  }
  void SetupTracing(
      const typename perfetto::DataSourceBase::SetupArgs&) override;
  void StopTracing() override { enabled_ = false; }
  bool Enabled() { return enabled_; }
  bool CategoryEnabled(const char* category) {
    return enabled_categories_.empty() ||
           enabled_categories_.contains(category);
  }
  void StartEvent(const char* catagory, const char* name);
  void EndEvent(const char* category);
  void EmitVulkanMemoryUsageEvent(const VulkanMemoryEvent* vulkan_memory_event);

 private:
  void ResetIfNecessary();
  void EmitThreadData();
  void EmitProcessData();

  uint64_t InternName(
      const char* name,
      typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
      perfetto::protos::pbzero::InternedData** interned_data);
  uint64_t InternAnnotationName(
      const char* name,
      typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
      perfetto::protos::pbzero::InternedData** interned_data);
  uint64_t InternCategory(
      const char* name,
      typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
      perfetto::protos::pbzero::InternedData** interned_data);
  uint64_t InternFunctionName(
      const char* name,
      typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
      perfetto::protos::pbzero::InternedData** interned_data);
  uint64_t InternVulkanAnnotationKey(
      const char* name,
      typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
      perfetto::protos::pbzero::InternedData** interned_data);

  std::string thread_name_;
  std::string process_name_;
  uint64_t thread_id_;
  uint64_t process_id_;

  core::Arena arena_;
  gapil::Map<std::string, uint64_t, false> interned_names_;
  gapil::Map<std::string, uint64_t, false> interned_annotation_names_;
  gapil::Map<std::string, uint64_t, false> interned_categories_;
  gapil::Map<std::string, uint64_t, false> interned_function_names_;
  gapil::Map<std::string, uint64_t, false> interned_vulkan_annotation_keys_;
  gapil::Map<std::string, uint64_t, false> enabled_categories_;
  bool emitted_thread_data_ = false;
  bool emitted_process_data_ = false;
  uint64_t last_reset_timestamp_;
  uint64_t reset_period_ms_;
  std::atomic_bool reset_;
  std::atomic_bool enabled_;
};

namespace tracing {
template <typename T>
ThreadlocalEmitter<T>& Emit() {
  thread_local ThreadlocalEmitter<T> emitter{};
  return emitter;
}
}  // namespace tracing

}  // namespace core

#define __INCLUDING_PERFETTO_THREADLOCAL_EMITTER_INC__
#include "core/vulkan/perfetto_producer/perfetto_threadlocal_emitter.inc"
#undef __INCLUDING_PERFETTO_THREADLOCAL_EMITTER_INC__

#endif  // __PERFETTO_THREADLOCAL_EMITTER_H__
