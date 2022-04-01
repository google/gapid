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

#ifndef __GAPIL_RUNTIME_BUFFER_H__
#define __GAPIL_RUNTIME_BUFFER_H__

#include "runtime.h"

namespace gapil {

// Buffer is a dynamic sized byte array.
class Buffer {
 public:
  // Reader is a buffer stream reader.
  class Reader {
   public:
    inline Reader(buffer* buf);

    // read copies sizeof(T) bytes from the current read position to out.
    // If there is enough data left in the buffer to read a full T then true is
    // returned and the read position is incremented by sizeof(T) bytes,
    // otherwise false is returned and the read position is unaltered.
    template <typename T>
    inline bool read(T* out);

   private:
    buffer* buf_;
    uint64_t offset_;
  };

  // Buffer constructs a new buffer with the given capacity and minimum
  // alignment.
  inline Buffer(arena* arena, uint64_t capacity = 16, uint64_t alignment = 16);

  // If release_ownership() has not been called then the destructor frees the
  // buffer. If release_ownership() has been called, then the destructor does
  // nothing.
  inline ~Buffer();

  // reader returns a reader for this Buffer.
  inline Reader reader();

  // set_size changes the size of the buffer, increasing the capacity if
  // necessary.
  inline void set_size(size_t size);

  // appends the bytes of T to the end of the buffer, reallocating if the buffer
  // capacity has been exceeded.
  template <typename T>
  inline void append(const T& data);

  // Writes the bytes of data to the offset in the buffer.
  // Returns true if there was enough space from offset to write a full T,
  // otherwise no write occurs and false is returned.
  template <typename T>
  inline bool write(uint64_t offset, const T& data);

  // release_ownership returns the underlying buffer, and prevents the buffer
  // from being automatically freed when this Buffer is destructed.
  inline buffer release_ownership();

 private:
  Buffer() = delete;
  Buffer(const Buffer&) = delete;
  Buffer(Buffer&&) = delete;
  Buffer& operator=(const Buffer&) = delete;

  buffer buf_;
  bool owns_buf_;
};

Buffer::Buffer(arena* arena, uint64_t capacity, uint64_t alignment)
    : buf_{0}, owns_buf_(true) {
  gapil_create_buffer(arena, capacity, alignment, &buf_);
}

Buffer::~Buffer() {
  if (owns_buf_) {
    gapil_destroy_buffer(&buf_);
  }
}

Buffer::Reader::Reader(buffer* buf) : buf_(buf), offset_(0) {}

template <typename T>
bool Buffer::Reader::read(T* out) {
  if (offset_ + sizeof(T) > buf_->size) {
    return false;
  }
  memcpy(out, buf_->data + offset_, sizeof(T));
  offset_ += sizeof(T);
  return true;
}

void Buffer::set_size(size_t size) {
  if (size > buf_.capacity) {
    gapil_realloc(buf_.arena, buf_.data, size, buf_.alignment);
    buf_.capacity = size;
  }
  buf_.size = size;
}

template <typename T>
void Buffer::append(const T& data) {
  gapil_append_buffer(&buf_, &data, sizeof(data));
}

template <typename T>
bool Buffer::write(uint64_t offset, const T& data) {
  if (offset + sizeof(T) > buf_.size) {
    return false;
  }
  memcpy(buf_.data + offset, &data, sizeof(T));
  return true;
}

buffer Buffer::release_ownership() {
  owns_buf_ = false;
  return buf_;
}

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_BUFFER_H__

