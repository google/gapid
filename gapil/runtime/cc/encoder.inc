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

// Private functions used by the generated encoder code.

#ifndef __GAPIL_RUNTIME_ENCODER_INC__
#define __GAPIL_RUNTIME_ENCODER_INC__

#include "gapil/runtime/cc/encoder.h"

namespace {

// buffer is a structure used to hold a variable size byte array.
// buffer is used internally by the compiler to write out variable length data.
typedef struct buffer_t {
  arena* arena;        // the arena that owns the buffer data.
  uint8_t* data;       // buffer data.
  uint32_t capacity;   // total capacity of the buffer.
  uint32_t size;       // current size of the buffer.
  uint32_t alignment;  // min alignment in bytes of the data allocation.
} buffer;

// gapil_buffer_init initializes a buffer allocating space with the given
// initial capacity.
void gapil_buffer_init(context* ctx, buffer* buf, uint32_t capacity) {
  buf->data = static_cast<uint8_t*>(gapil_alloc(ctx->arena, capacity, 1));
  buf->arena = ctx->arena;
  buf->capacity = capacity;
  buf->size = 0;
  buf->alignment = 16;
}

// gapil_buffer_append appends the given data to the buffer.
void gapil_buffer_append(buffer* buf, uint32_t size, const void* data) {
  uint32_t new_size = buf->size + size;
  if (new_size > buf->capacity) {
    uint32_t new_capacity = 2 * new_size;
    uint8_t* new_data = static_cast<uint8_t*>(
        gapil_realloc(buf->arena, buf->data, new_capacity, 1));
    buf->data = new_data;
    buf->capacity = new_capacity;
    memcpy(&new_data[buf->size], data, size);
  } else {
    memcpy(&buf->data[buf->size], data, size);
  }
  buf->size = new_size;
}

// write_var_int writes an integer in the proto varint wire format.
void write_var_int(buffer* buf, uint64_t v) {
  char bytes[10];
  uint32_t size = 0;
  while (v >= 0x80) {
    bytes[size++] = static_cast<uint8_t>(v | 0x80);
    v >>= 7;
  }
  bytes[size++] = static_cast<uint8_t>(v);
  gapil_buffer_append(buf, size, bytes);
}

// write_zig_zag writes an integer in the proto zig-zag wire format.
void write_zig_zag(buffer* buf, int64_t v) {
  write_var_int(buf, (v << 1) ^ (v >> 63));
}

}  // namespace

#endif  // __GAPIL_RUNTIME_ENCODER_INC__