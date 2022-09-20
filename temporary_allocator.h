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
#include <malloc.h>

#include <vector>

#include "common.h"

namespace gapid2 {

struct temporary_allocator {
  temporary_allocator() { memory_blocks_.push_back(block{4096, init, 4096}); }

  ~temporary_allocator() {
    for (auto& i : memory_blocks_) {
      if (i.data != init) {
        free(i.data);
      }
    }
  }

  void reset() {
    data_offset = 0;
    for (auto& i : memory_blocks_) {
      i.left = i.size;
    }
  }

  void* get_memory(size_t _sz) {
    _sz = (_sz + 7) & (~7);  // Make sure we are at least 8-byte aligned
    auto* b = &memory_blocks_[data_offset];
    if (b->left < _sz) {
      data_offset++;
      if (data_offset >= memory_blocks_.size() ||
          memory_blocks_[data_offset].size < _sz) {
        auto sz = std::max<size_t>(_sz, 4096);
        char* c = (char*)malloc(sz);
        memory_blocks_.push_back(block{sz, c, sz});
        if (data_offset != memory_blocks_.size() - 1) {
          std::swap(memory_blocks_[data_offset],
                    memory_blocks_[memory_blocks_.size() - 1]);
        }
      }
      memory_blocks_[data_offset].left = memory_blocks_[data_offset].size;
    }
    b = &memory_blocks_[data_offset];
    void* ptr = b->data + (b->size - b->left);
    b->left -= _sz;
    return ptr;
  }

  template <typename T>
  T* get_typed_memory(size_t count) {
    T* t = static_cast<T*>(get_memory(sizeof(T) * count));
    return t;
  }
  char init[4096];
  std::vector<block> memory_blocks_;
  size_t data_offset = 0;
};
}  // namespace gapid2