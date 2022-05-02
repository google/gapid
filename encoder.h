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
#include <deque>
#include <functional>

#include "common.h"

namespace gapid2 {
struct encoder {
#define INITIAL_SIZE 4096
  encoder() {
    data_.push_back(
        block{INITIAL_SIZE, (char*)malloc(INITIAL_SIZE), INITIAL_SIZE});
    current_ = &data_[0];
  }

  void ensure_large_enough(const size_t _sz) {
    if (current_->left < _sz) {
      ++data_offset;
      if (data_offset >= data_.size() || data_[data_offset].size < _sz) {
        auto sz = std::max<size_t>(_sz, INITIAL_SIZE);
        char* c = (char*)malloc(sz);
        data_.push_back(block{sz, c, sz});
        // If we couldnt fit into the next bucket, make a new bucket,
        // and put it here
        if (data_offset != data_.size() - 1) {
          std::swap(data_[data_offset], data_[data_.size() - 1]);
        }
      }
      data_[data_offset].left = data_[data_offset].size;
      current_ = &data_[data_offset];
    }
  }

  void write(const void* ptr, size_t length) {
    auto b = current_;
    memcpy(b->data + (b->size - b->left), ptr, length);
    b->left -= length;
  }

  template <typename T, typename V>
  void encode(const V& _t) {
    ensure_large_enough(sizeof(T));
    T t = static_cast<const T>(_t);
    write(&t, sizeof(t));
  }

  template <typename T>
  void encode_primitive_array(const T* _t, size_t len) {
    ensure_large_enough(sizeof(T) * len);
    write(_t, sizeof(T) * len);
  }

  void reset() {
    for (size_t i = 0; i <= data_offset; ++i) {
      data_[i].left = data_[i].size;
    }
    data_offset = 0;
    current_ = &data_[data_offset];
  }

  std::vector<block> data_;
  block* current_;
  size_t data_offset = 0;
};

class encoder_handle {
 public:
  encoder_handle(encoder* _encoder, std::function<void()> on_return)
      : _encoder(_encoder), _on_return(on_return) {}
  explicit encoder_handle(encoder* _encoder) : _encoder(_encoder) {}
  encoder_handle(encoder_handle&& _other) = default;
  ~encoder_handle() {
    if (_on_return) {
      _on_return();
    }
  }
  encoder* operator*() { return _encoder; }
  encoder* operator->() { return _encoder; }
  operator encoder*() { return _encoder; }
  operator bool() const { return _encoder != nullptr; }
  encoder* _encoder;
  std::function<void()> _on_return;
};
}  // namespace gapid2