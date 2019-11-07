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

#ifndef __PERFETTO_PROTO_STRUCTS_H__
#define __PERFETTO_PROTO_STRUCTS_H__

#include <deque>
#include <string>

#include "core/cc/log.h"
#include "protos/perfetto/trace/gpu/vulkan_memory_event.pbzero.h"

namespace core {

// Data structs for protos/perfetto/trace/gpu/vulkan_memory_event.proto

enum VulkanMemoryEventAnnotationValType {
  kInt = 1,
  kString = 2,
};

struct VulkanMemoryEventAnnotation {
  VulkanMemoryEventAnnotation(std::string key, int64_t value) {
    this->key = key;
    this->value_type = VulkanMemoryEventAnnotationValType::kInt;
    this->int_value = value;
  };
  VulkanMemoryEventAnnotation(std::string key, const std::string& value) {
    this->key = key;
    this->value_type = VulkanMemoryEventAnnotationValType::kString;
    this->string_value = value;
  };

  std::string key;
  VulkanMemoryEventAnnotationValType value_type;
  int64_t int_value;
  std::string string_value;
};

struct VulkanMemoryEvent {
  VulkanMemoryEvent()
      : has_device(false),
        has_device_memory(false),
        has_heap(false),
        has_object_handle(false),
        has_memory_address(false),
        has_memory_size(false) {}

  // Mandatory fields
  perfetto::protos::pbzero::VulkanMemoryEvent_Source source;
  perfetto::protos::pbzero::VulkanMemoryEvent_Type type;
  uint64_t timestamp;

  // Optional fields
  bool has_device;
  bool has_device_memory;
  bool has_heap;
  bool has_object_handle;
  bool has_memory_address;
  bool has_memory_size;

  uint64_t device;
  uint64_t device_memory;
  uint32_t heap;
  std::string function_name;
  uint64_t object_handle;
  uint64_t memory_address;
  uint64_t memory_size;
  std::deque<VulkanMemoryEventAnnotation> annotations;
};

}  // namespace core

#endif  // __PERFETTO_PROTO_STRUCTS_H__
