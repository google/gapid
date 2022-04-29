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
#include <cstring>
#include <vector>

#include "common.h"

namespace gapid2 {
struct decoder {
  decoder() {
    memory_blocks_.push_back(block{4096, (char*)malloc(4096), 4096});
  }

  decoder(std::vector<block> data) : decoder() { data_ = std::move(data); }

  void* get_memory(size_t _sz) {
    if (_sz == 0) {
      return nullptr;
    }
    _sz = (_sz + 7) & (~7);  // Make sure we are at least 8-byte aligned
    auto* b = &memory_blocks_[data_offset];
    if (b->left < _sz) {
      ++data_offset;
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

  void read(void* ptr, size_t length) {
    auto b = &data_[read_offset];
    GAPID2_ASSERT(data_[read_offset].size >= (read_head + length),
                  "Out of data");
    if (ptr) {
      memcpy(ptr, b->data + read_head, length);
    }
    read_head += length;
    if (read_head == b->size - b->left) {
      read_head = 0;
      read_offset++;
    }
  }

  template <typename T, typename V>
  void decode(V* _t) {
    T t;
    read(&t, sizeof(T));
    *_t = static_cast<V>(t);
  }

  template <typename T>
  T decode() {
    T t;
    read(&t, sizeof(T));
    return t;
  }

  template <typename T>
  void decode_primitive_array(T* _t, size_t len) {
    read(_t, sizeof(T) * len);
  }

  template <typename T>
  void drop_primitive_array(size_t len) {
    read(nullptr, sizeof(T) * len);
  }

  uint64_t data_left() {
    uint64_t dl = 0;
    for (size_t i = read_offset; i < data_.size(); ++i) {
      if (i == read_offset) {
        dl += (data_[i].size - data_[i].left) - read_head;
      } else {
        dl += data_[i].size - data_[i].left;
      }
    }
    return dl;
  }

  bool has_data_left() {
    for (size_t i = read_offset; i < data_.size(); ++i) {
      if ((data_[i].size - data_[i].left) - read_head) {
        return true;
      }
    }
    return false;
  }

  std::vector<block> memory_blocks_;
  std::vector<block> data_;
  size_t data_offset = 0;
  size_t read_offset = 0;
  size_t read_head = 0;
};
}  // namespace gapid2