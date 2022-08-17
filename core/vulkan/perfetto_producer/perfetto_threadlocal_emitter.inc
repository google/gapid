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

#ifndef __INCLUDING_PERFETTO_THREADLOCAL_EMITTER_INC__
#error \
    "It is invalid to include this file directly. Include the header instead."
#endif
#include <mutex>
#include <sstream>
#include <string>
#include "core/cc/process_name.h"
#include "core/cc/thread.h"
#include "core/cc/timer.h"
#include "core/vulkan/perfetto_producer/perfetto_data_source.h"
#include "perfetto/base/time.h"

#include "perfetto/tracing/core/data_source_config.h"
#include "protos/perfetto/trace/gpu/vulkan_api_event.pbzero.h"
#include "protos/perfetto/trace/gpu/vulkan_memory_event.pbzero.h"
#include "protos/perfetto/trace/interned_data/interned_data.pbzero.h"
#include "protos/perfetto/trace/profiling/profile_common.pbzero.h"
#include "protos/perfetto/trace/trace_packet.pbzero.h"
#include "protos/perfetto/trace/track_event/debug_annotation.pbzero.h"
#include "protos/perfetto/trace/track_event/thread_descriptor.pbzero.h"
#include "protos/perfetto/trace/track_event/track_event.pbzero.h"

// Note these implementations should match map.go

namespace core {

template <typename T>
ThreadlocalEmitter<T>::ThreadlocalEmitter()
    : interned_names_(&arena_),
      interned_annotation_names_(&arena_),
      interned_categories_(&arena_),
      interned_function_names_(&arena_),
      interned_vulkan_annotation_keys_(&arena_),
      enabled_categories_(&arena_),
      last_reset_timestamp_(0),
      reset_period_ms_(SEQUENCE_RESET_PERIOD_MS),
      reset_(false) {
  thread_name_ = Thread::current().get_name();
  process_name_ = core::get_process_name();
  thread_id_ = Thread::current().id();
  process_id_ = core::get_process_id();
  PerfettoProducer<T>::Get().RegisterEmitter(this);
}

template <typename T>
ThreadlocalEmitter<T>::~ThreadlocalEmitter() {
  PerfettoProducer<T>::Get().UnregisterEmitter(this);
}

template <typename T>
uint64_t ThreadlocalEmitter<T>::InternName(
    const char* name,
    typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
    perfetto::protos::pbzero::InternedData** interned_data) {
  uint64_t val = interned_names_.findOrZero(name);
  if (!val) {
    val = interned_names_.count() + 1;
    interned_names_[name] = val;
    if (!*interned_data) {
      *interned_data = packet->set_interned_data();
    }
    perfetto::protos::pbzero::EventName* event_name =
        (*interned_data)->add_event_names();
    event_name->set_iid(val);
    event_name->set_name(name);
    return interned_names_.count();
  }
  return val;
}

template <typename T>
uint64_t ThreadlocalEmitter<T>::InternAnnotationName(
    const char* name,
    typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
    perfetto::protos::pbzero::InternedData** interned_data) {
  uint64_t val = interned_annotation_names_.findOrZero(name);
  if (!val) {
    val = interned_annotation_names_.count() + 1;
    interned_annotation_names_[name] = val;
    if (!*interned_data) {
      *interned_data = packet->set_interned_data();
    }
    auto* event_name = (*interned_data)->add_debug_annotation_names();
    event_name->set_iid(val);
    event_name->set_name(name);
    return interned_annotation_names_.count();
  }
  return val;
}

template <typename T>
uint64_t ThreadlocalEmitter<T>::InternCategory(
    const char* name,
    typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
    perfetto::protos::pbzero::InternedData** interned_data) {
  uint64_t val = interned_categories_.findOrZero(name);
  if (!val) {
    val = interned_categories_.count() + 1;
    interned_categories_[name] = val;
    if (!*interned_data) {
      *interned_data = packet->set_interned_data();
    }
    auto* event_name = (*interned_data)->add_event_categories();
    event_name->set_iid(val);
    event_name->set_name(name);
    return interned_categories_.count();
  }
  return val;
}

template <typename T>
uint64_t ThreadlocalEmitter<T>::InternFunctionName(
    const char* name,
    typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
    perfetto::protos::pbzero::InternedData** interned_data) {
  uint64_t val = interned_function_names_.findOrZero(name);
  if (!val) {
    val = interned_function_names_.count() + 1;
    interned_function_names_[name] = val;
    if (!*interned_data) {
      *interned_data = packet->set_interned_data();
    }
    auto* function_name = (*interned_data)->add_function_names();
    function_name->set_iid(val);
    function_name->set_str(reinterpret_cast<const uint8_t*>(name),
                           strlen(name));
    return interned_function_names_.count();
  }
  return val;
}

template <typename T>
uint64_t ThreadlocalEmitter<T>::InternVulkanAnnotationKey(
    const char* name,
    typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
    perfetto::protos::pbzero::InternedData** interned_data) {
  uint64_t val = interned_vulkan_annotation_keys_.findOrZero(name);
  if (!val) {
    val = interned_vulkan_annotation_keys_.count() + 1;
    interned_vulkan_annotation_keys_[name] = val;
    if (!*interned_data) {
      *interned_data = packet->set_interned_data();
    }
    auto* annotation_key = (*interned_data)->add_vulkan_memory_keys();
    annotation_key->set_iid(val);
    annotation_key->set_str(reinterpret_cast<const uint8_t*>(name),
                            strlen(name));
    return interned_vulkan_annotation_keys_.count();
  }
  return val;
}

template <typename T>
void ThreadlocalEmitter<T>::ResetIfNecessary() {
  if (reset_) {
    interned_annotation_names_.clear();
    interned_names_.clear();
    interned_function_names_.clear();
    interned_vulkan_annotation_keys_.clear();
    emitted_thread_data_ = false;
    emitted_process_data_ = false;
    reset_ = false;
  }
}

template <typename T>
void ThreadlocalEmitter<T>::EmitThreadData() {
  if (emitted_thread_data_) {
    return;
  }
  uint64_t time = perfetto::base::GetBootTimeNs().count();
  PerfettoProducer<T>::Trace(
      [this, time](typename PerfettoProducer<T>::TraceContext ctx) {
        auto packet = ctx.NewTracePacket();
        packet->set_timestamp(time);
        packet->set_incremental_state_cleared(true);
        perfetto::protos::pbzero::ThreadDescriptor* thread_descriptor =
            packet->set_thread_descriptor();
        thread_descriptor->set_pid(process_id_);
        thread_descriptor->set_tid(thread_id_);
        thread_descriptor->set_thread_name(thread_name_.c_str());
      });

  EmitProcessData();
  emitted_thread_data_ = true;
}

template <typename T>
void ThreadlocalEmitter<T>::EmitProcessData() {
  if (emitted_process_data_) {
    return;
  }
  uint64_t time = perfetto::base::GetBootTimeNs().count();
  PerfettoProducer<T>::Trace(
      [this, time](typename PerfettoProducer<T>::TraceContext ctx) {
        auto packet = ctx.NewTracePacket();
        packet->set_timestamp(time);
        perfetto::protos::pbzero::InternedData* interned_data = nullptr;
        uint64_t name_iid = InternName("process_name", packet, &interned_data);
        uint64_t annotation_name_iid =
            InternAnnotationName("name", packet, &interned_data);

        uint64_t category_iid = InternCategory("cat", packet, &interned_data);

        auto* track_event = packet->set_track_event();
        track_event->add_category_iids(category_iid);

        auto* debug_annotation = track_event->add_debug_annotations();
        debug_annotation->set_name_iid(annotation_name_iid);
        debug_annotation->set_string_value(process_name_.c_str());

        auto* legacy_event = track_event->set_legacy_event();
        legacy_event->set_name_iid(name_iid);
        legacy_event->set_phase('M');
      });
  emitted_process_data_ = true;
}

template <typename T>
void ThreadlocalEmitter<T>::SetupTracing(
    const typename perfetto::DataSourceBase::SetupArgs& a) {
  auto config = a.config;
  if (config) {
    enabled_categories_.clear();
    std::istringstream f(config->legacy_config());
    std::string s;
    enabled_categories_[""] = 1;
    while (std::getline(f, s, ':')) {
      enabled_categories_[s] = 1;
    }
  } else {
    enabled_categories_.clear();
    enabled_categories_[""] = 1;
  }
}

template <typename T>
void ThreadlocalEmitter<T>::StartEvent(const char* category, const char* name) {
  if (!enabled_) {
    return;
  }
  if (category && !enabled_categories_.contains(category)) {
    return;
  }
  ResetIfNecessary();
  EmitThreadData();
  uint64_t time = perfetto::base::GetBootTimeNs().count();

  PerfettoProducer<T>::Trace(
      [this, name, time,
       category](typename PerfettoProducer<T>::TraceContext ctx) {
        auto packet = ctx.NewTracePacket();
        packet->set_timestamp(time);
        perfetto::protos::pbzero::InternedData* interned_data = nullptr;
        uint64_t name_iid = InternName(name, packet, &interned_data);

        // TODO: proper categories
        uint64_t category_iid =
            InternCategory(category, packet, &interned_data);

        auto track_event = packet->set_track_event();
        track_event->add_category_iids(category_iid);

        auto legacy_event = track_event->set_legacy_event();
        legacy_event->set_name_iid(name_iid);
        legacy_event->set_phase('B');
      });
}

template <typename T>
void ThreadlocalEmitter<T>::EndEvent(const char* category) {
  if (!enabled_) {
    return;
  }
  if (category && !enabled_categories_.contains(category)) {
    return;
  }
  uint64_t time = perfetto::base::GetBootTimeNs().count();
  PerfettoProducer<T>::Trace(
      [this, time, category](typename PerfettoProducer<T>::TraceContext ctx) {
        auto packet = ctx.NewTracePacket();
        packet->set_timestamp(time);

        perfetto::protos::pbzero::InternedData* interned_data = nullptr;
        uint64_t category_iid =
            InternCategory(category, packet, &interned_data);

        auto track_event = packet->set_track_event();
        track_event->add_category_iids(category_iid);

        auto legacy_event = track_event->set_legacy_event();
        legacy_event->set_phase('E');
      });
}

template <typename T>
void ThreadlocalEmitter<T>::EmitVulkanMemoryUsageEvent(
    const VulkanMemoryEvent* event) {
  if (!enabled_) {
    return;
  }
  // Check if we need to reset the sequence incermental data (interned strings,
  // in this case).
  bool is_reset = false;
  if (!last_reset_timestamp_ && (event->timestamp - last_reset_timestamp_) >
                                    (SEQUENCE_RESET_PERIOD_MS * 1000000)) {
    reset_ = true;
    is_reset = true;
    last_reset_timestamp_ = event->timestamp;
  }

  ResetIfNecessary();
  EmitProcessData();
  EmitThreadData();

  PerfettoProducer<T>::Trace(
      [this, event, is_reset](typename PerfettoProducer<T>::TraceContext ctx) {
        auto packet = ctx.NewTracePacket();
        packet->set_timestamp(event->timestamp);
        if (is_reset) {
          packet->set_sequence_flags(
              perfetto::protos::pbzero::TracePacket_SequenceFlags::
                  SEQ_INCREMENTAL_STATE_CLEARED);
        } else {
          packet->set_sequence_flags(
              perfetto::protos::pbzero::TracePacket_SequenceFlags::
                  SEQ_NEEDS_INCREMENTAL_STATE);
        }

        perfetto::protos::pbzero::InternedData* interned_data = nullptr;
        std::unordered_map<const char*, uint64_t> interned_strings;

        if (event->function_name.length() > 0) {
          interned_strings[event->function_name.c_str()] = InternFunctionName(
              event->function_name.c_str(), packet, &interned_data);
        }
        for (const auto& annotation : event->annotations) {
          interned_strings[annotation.key.c_str()] = InternVulkanAnnotationKey(
              annotation.key.c_str(), packet, &interned_data);
          if (annotation.value_type ==
              VulkanMemoryEventAnnotationValType::kString) {
            interned_strings[annotation.string_value.c_str()] =
                InternVulkanAnnotationKey(annotation.string_value.c_str(),
                                          packet, &interned_data);
          }
        }

        auto memory_event = packet->set_vulkan_memory_event();
        memory_event->set_source(event->source);
        memory_event->set_operation(event->operation);
        memory_event->set_timestamp(event->timestamp);
        memory_event->set_pid(process_id_);

        if (event->has_device) memory_event->set_device(event->device);
        if (event->has_device_memory)
          memory_event->set_device_memory(event->device_memory);
        if (event->has_heap) memory_event->set_heap(event->heap);
        if (event->has_memory_type)
          memory_event->set_memory_type(event->memory_type);
        if (event->function_name.length() > 0) {
          memory_event->set_caller_iid(
              interned_strings[event->function_name.c_str()]);
        }
        if (event->has_object_handle)
          memory_event->set_object_handle(event->object_handle);
        if (event->has_allocation_scope)
          memory_event->set_allocation_scope(event->allocation_scope);
        if (event->has_memory_address)
          memory_event->set_memory_address(event->memory_address);
        if (event->has_memory_size)
          memory_event->set_memory_size(event->memory_size);

        for (const auto& annotation : event->annotations) {
          auto* vulkan_annotation = memory_event->add_annotations();
          uint64_t key_iid = interned_strings[annotation.key.c_str()];
          vulkan_annotation->set_key_iid(key_iid);
          switch (annotation.value_type) {
            case VulkanMemoryEventAnnotationValType::kInt:
              vulkan_annotation->set_int_value(annotation.int_value);
              break;
            case VulkanMemoryEventAnnotationValType::kString:
              uint64_t string_iid =
                  interned_strings[annotation.string_value.c_str()];
              vulkan_annotation->set_string_iid(string_iid);
          }
        }
      });
}

}  // namespace core

