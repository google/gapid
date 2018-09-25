// Copyright (C) 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#ifndef __GAPIL_RUNTIME_REPLAY_DATAEX_H__
#define __GAPIL_RUNTIME_REPLAY_DATAEX_H__

#include "core/cc/id.h"
#include "core/cc/interval_list.h"

#include <unordered_map>
#include <vector>

namespace gapil {
namespace runtime {
namespace replay {

// align returns val aligned up the the next multiple of by.
template <typename T>
inline T align(T val, T by) {
  return ((val + by - 1) / by) * by;
}

template <typename T>
class StackAllocator {
 public:
  StackAllocator() : head(0), min_alignment(1) {}

  inline T alloc(T size, T alignment) {
    head = align(head, alignment);
    auto out = head;
    head += size;
    min_alignment = std::max(alignment, min_alignment);
    return out;
  }

  inline T size() const { return head; }
  inline T alignment() const { return min_alignment; }

 private:
  T head;
  T min_alignment;
};

class MemoryRange : public core::Interval<uint64_t> {
 public:
  MemoryRange(uint64_t start, uint64_t end, uint32_t alignment)
      : core::Interval<uint64_t>{start, end}, mAlignment(alignment) {}
  uint32_t mAlignment;
};

typedef core::CustomIntervalList<MemoryRange> MemoryRanges;

struct DataEx {
  typedef uint32_t Namespace;
  typedef uint32_t RemapKey;
  typedef uint32_t ResourceIndex;
  typedef uint64_t VolatileAddr;
  typedef uint64_t ConstantAddr;

  struct ResourceInfo {
    uint32_t index;
    uint32_t size;
  };

  StackAllocator<VolatileAddr> allocated;
  std::unordered_map<Namespace, MemoryRanges> reserved;
  std::unordered_map<core::Id, ResourceInfo> resources;
  std::unordered_map<RemapKey, VolatileAddr> remappings;
  std::unordered_map<core::Id, uint32_t> constant_offsets;
};

}  // namespace replay
}  // namespace runtime
}  // namespace gapil

#endif  // __GAPIL_RUNTIME_REPLAY_DATAEX_H__