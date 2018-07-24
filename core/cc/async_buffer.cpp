/*
 * Copyright (C) 2018 Google Inc.
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

#include "async_buffer.h"
#include "assert.h"

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif  // __STDC_FORMAT_MACROS
#include <inttypes.h>

namespace core {

AsyncBuffer::AsyncBuffer(std::shared_ptr<StreamWriter> out, size_t buffer_size)
    : size_(buffer_size),
      free_(buffer_size),
      read_head_(0),
      write_head_(0),
      out_(std::move(out)) {
  GAPID_ASSERT(buffer_size > 0);
  buffer_ = new uint8_t[buffer_size];
  thread_ = new std::thread([this]() { worker(); });
}

AsyncBuffer::~AsyncBuffer() {
  flush();
  free_.close();
  written_.close();
  thread_->join();
  delete[] buffer_;
}

uint64_t AsyncBuffer::write(const void* data, uint64_t size) {
  auto bytes = reinterpret_cast<const uint8_t*>(data);
  size_t remaining = size;
  size_t written = 0;
  while (remaining > size_) {
    // Split large writes into ring-buffer sized chunks.
    written += write_chunk(bytes, size_);
    bytes += size_;
    remaining -= size_;
  }

  written += write_chunk(bytes, remaining);
  return written;
}

void AsyncBuffer::flush() { free_.wait_until(size_); }

uint64_t AsyncBuffer::write_chunk(const void* data, uint64_t size) {
  GAPID_ASSERT(size <= size_);

  auto bytes = reinterpret_cast<const uint8_t*>(data);
  size_t remaining = size;

  if (!free_.acquire(size)) {
    GAPID_FATAL("Attempting to write on a destructed AsyncBuffer");
  }

  if (write_head_ + remaining >= size_) {
    // write wraps buffer end/start
    size_t count = size_ - write_head_;
    memcpy(&buffer_[write_head_], bytes, count);
    write_head_ = 0;
    bytes += count;
    remaining -= count;
  }
  if (remaining > 0) {
    memcpy(&buffer_[write_head_], bytes, remaining);
    write_head_ += remaining;
  }

  if (!written_.release(size)) {
    GAPID_FATAL("Attempting to write on a destructed AsyncBuffer");
  }

  return size;
}

void AsyncBuffer::worker() {
  while (true) {
    int count = written_.acquire_all();
    if (count < 0) {
      return;  // Signal that AsyncBuffer is destructing.
    }

    size_t remaining = static_cast<size_t>(count);
    while (remaining > 0) {
      size_t chunk = std::min(remaining, size_ - read_head_);
      size_t count = out_->write(&buffer_[read_head_], chunk);
      GAPID_ASSERT(count <= chunk);
      remaining -= count;
      read_head_ += count;
      if (read_head_ == size_) {
        read_head_ = 0;
      }
    }

    if (!free_.release(count)) {
      return;  // Signal that AsyncBuffer is destructing.
    }
  }
}

}  // namespace core
