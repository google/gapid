// Copyright (C) 2022 Google Inc.
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

#ifndef __GAPIL_RUNTIME_POOL_H__
#define __GAPIL_RUNTIME_POOL_H__

#include <cstring>

#include "core/memory/arena/cc/arena.h"

namespace gapil {

class Pool {
 public:
  Pool(core::Arena* arena, uint32_t id, uint64_t size)
      : id_(id), size_(size), arena_(arena), buffer_(nullptr), ref_count_(1) {}

  inline void allocate() {
    buffer_ = arena_->allocate(size_, 16);
    memset(buffer_, 0, size_);
  }

  inline void reference() { ref_count_++; }

  inline void release() {
    ref_count_--;
    if (ref_count_ == 0) {
      arena_->free(buffer_);
      arena_->destroy(this);
    }
  }

  inline uint32_t id() const { return id_; }

  inline uint64_t size() const { return size_; }

  inline uint64_t base() const { return reinterpret_cast<uint64_t>(buffer_); }

  inline const void* buffer() const { return buffer_; }

 private:
  uint32_t id_;         // unique identifier of this pool.
  uint64_t size_;       // total size of the pool in bytes.
  core::Arena* arena_;  // arena that owns the this pool and buffer.
  void* buffer_;        // data in this pool.
  uint32_t ref_count_;  // number of owners of this pool.
};

}  // namespace gapil

#endif  // GAPIL_POOL_H