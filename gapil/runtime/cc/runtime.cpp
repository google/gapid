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

#include "runtime.h"

#include "core/cc/assert.h"
#include "core/cc/log.h"
#include "core/memory/arena/cc/arena.h"

#include <stdarg.h>
#include <stddef.h>
#include <stdlib.h>

#if TARGET_OS == GAPID_OS_ANDROID
// for snprintf
#include <cstdio>
#endif

#define __STDC_FORMAT_MACROS
#include <inttypes.h>

#include <cstring>

#if 0
#define DEBUG_PRINT(...) GAPID_DEBUG(__VA_ARGS__)
#else
#define DEBUG_PRINT(...)
#endif

#define SLICE_FMT \
  "[pool: %p, root: 0x%" PRIx64 ", base: 0x%" PRIx64 ", size: 0x%" PRIx64 "]"
#define SLICE_ARGS(sli) sli->pool, sli->root, sli->base, sli->size

using core::Arena;

namespace {

void* default_pool_data_resolver(context*, pool* pool, uint64_t ptr,
                                 gapil_data_access, uint64_t* len) {
  if (pool != nullptr) {
    auto pool_base =
        static_cast<uint64_t>(reinterpret_cast<uintptr_t>(pool->buffer));
    if (ptr > pool->size) {
      GAPID_FATAL("ptr (0x%" PRIx64
                  ") is greater than the pool size (0x%" PRIx64 ")",
                  ptr, pool->size);
    }
    if (len != nullptr) {
      *len = pool->size - ptr;
    }
    return reinterpret_cast<void*>(pool_base + ptr);
  }
  if (len != nullptr) {
    *len = (~uint64_t(0)) - ptr;
  }
  return reinterpret_cast<void*>(ptr);
}

static gapil_pool_data_resolver* pool_data_resolver =
    &default_pool_data_resolver;
static gapil_database_storer* database_storer = nullptr;

}  // anonymous namespace

extern "C" {

void gapil_set_pool_data_resolver(gapil_pool_data_resolver* cb) {
  pool_data_resolver = cb != nullptr ? cb : &default_pool_data_resolver;
}

void gapil_set_database_storer(gapil_database_storer* cb) {
  database_storer = cb;
}

void gapil_logf(uint8_t severity, uint8_t* file, uint32_t line, uint8_t* fmt,
                ...) {
  // core/log/severity.go is in reverse order to log.h! :(
  severity = 5 - severity;
  if (GAPID_SHOULD_LOG(severity)) {
    va_list args;
    va_start(args, fmt);
    auto f =
        (file != nullptr) ? reinterpret_cast<const char*>(file) : "<unknown>";
#if TARGET_OS == GAPID_OS_ANDROID
    char buf[2048];
    snprintf(buf, sizeof(buf), "[%s:%" PRIu32 "] %s", f, line, fmt);
    __android_log_vprint(severity, "GAPID", buf, args);
#else
    ::core::Logger::instance().vlogf(severity, f, line,
                                     reinterpret_cast<const char*>(fmt), args);
#endif  // TARGET_OS
    va_end(args);
  }
}

void* gapil_alloc(arena_t* a, uint64_t size, uint64_t align) {
  Arena* arena = reinterpret_cast<Arena*>(a);
  void* ptr = arena->allocate(size, align);
  DEBUG_PRINT("gapil_alloc(size: 0x%" PRIx64 ", align: 0x%" PRIx64 ") -> %p",
              size, align, ptr);
  return ptr;
}

void* gapil_realloc(arena_t* a, void* ptr, uint64_t size, uint64_t align) {
  Arena* arena = reinterpret_cast<Arena*>(a);
  void* retptr = arena->reallocate(ptr, size, align);
  DEBUG_PRINT("gapil_realloc(ptr: %p, 0x%" PRIx64 ", align: 0x%" PRIx64
              ") -> %p",
              ptr, size, align, retptr);
  return retptr;
}

void gapil_free(arena_t* a, void* ptr) {
  DEBUG_PRINT("gapil_free(ptr: %p)", ptr);

  Arena* arena = reinterpret_cast<Arena*>(a);
  arena->free(ptr);
}

void gapil_create_buffer(arena* a, uint64_t capacity, uint64_t alignment,
                         buffer* buf) {
  DEBUG_PRINT("gapil_create_buffer(capacity: %" PRId64 ", alignment: %" PRId64
              ")",
              capacity, alignment);
  Arena* arena = reinterpret_cast<Arena*>(a);
  buf->data = (uint8_t*)arena->allocate(capacity, alignment);
  buf->size = 0;
  buf->capacity = capacity;
}

void gapil_destroy_buffer(arena* a, buffer* buf) {
  DEBUG_PRINT("gapil_destroy_buffer()");
  Arena* arena = reinterpret_cast<Arena*>(a);
  arena->free(buf->data);
  buf->capacity = 0;
  buf->size = 0;
}

void gapil_append_buffer(arena* a, buffer* buf, const void* data, uint64_t size,
                         uint64_t alignment) {
  DEBUG_PRINT("gapil_append_buffer(data: %p, size: " PRId64
              ", alignment: %" PRId64 ")",
              data, size, alignment);
  if (buf->size + size > buf->capacity) {
    Arena* arena = reinterpret_cast<Arena*>(a);
    buf->capacity *= 2;
    buf->data =
        (uint8_t*)arena->reallocate(buf->data, buf->capacity, alignment);
  }
  memcpy(buf->data + buf->size, data, size);
  buf->size = buf->size + size;
}

pool* gapil_make_pool(context* ctx, uint64_t size) {
  Arena* arena = reinterpret_cast<Arena*>(ctx->arena);

  void* buffer = arena->allocate(size, 16);
  memset(buffer, 0, size);

  auto pool = arena->create<pool_t>();
  pool->arena = ctx->arena;
  pool->id = (*ctx->next_pool_id)++;
  pool->size = size;
  pool->ref_count = 1;
  pool->buffer = buffer;

  DEBUG_PRINT("gapil_make_pool(size: 0x%" PRIx64 ") -> [pool: %p, buffer: %p]",
              size, pool, buffer);
  return pool;
}

void gapil_free_pool(pool* pool) {
  DEBUG_PRINT("gapil_free_pool(pool: %p)", pool);

  if (pool == nullptr) {  // Application pool.
    // TODO: Panic?
    return;
  }

  Arena* arena = reinterpret_cast<Arena*>(pool->arena);
  arena->free(pool->buffer);
  arena->destroy(pool);
}

void* gapil_slice_data(context* ctx, slice* sli, gapil_data_access access) {
  uint64_t bufSize = 0;
  auto ptr = pool_data_resolver(ctx, sli->pool, sli->base, access, &bufSize);
  GAPID_ASSERT_MSG(sli->size <= bufSize,
                   "gapil_slice_data(" SLICE_FMT
                   ", %d) overflows underlying buffer",
                   SLICE_ARGS(sli), access);

  DEBUG_PRINT("gapil_slice_data(" SLICE_FMT ", %d) -> %p", SLICE_ARGS(sli),
              access, ptr);
  return ptr;
}

void gapil_copy_slice(context* ctx, slice* dst, slice* src) {
  DEBUG_PRINT(
      "gapil_copy_slice(ctx: %p,\n"
      "    dst: " SLICE_FMT
      ",\n"
      "    src: " SLICE_FMT ")",
      ctx, SLICE_ARGS(dst), SLICE_ARGS(src));

  uint64_t size = std::min(dst->size, src->size);

  uint64_t dstBufLen = 0;
  auto dstPtr =
      pool_data_resolver(ctx, dst->pool, dst->base, GAPIL_WRITE, &dstBufLen);
  GAPID_ASSERT_MSG(size <= dstBufLen, "gapil_copy_slice overflows dst buffer");

  uint64_t srcBufLen = 0;
  auto srcPtr =
      pool_data_resolver(ctx, src->pool, src->base, GAPIL_READ, &srcBufLen);
  GAPID_ASSERT_MSG(size <= srcBufLen, "gapil_copy_slice overflows src buffer");

  memcpy(dstPtr, srcPtr, size);
}

void gapil_cstring_to_slice(context* ctx, uintptr_t ptr, slice* out) {
  DEBUG_PRINT("gapil_cstring_to_slice(ptr: 0x%" PRIx64 ")", ptr);

  pool* pool = nullptr;  // application pool

  uint64_t bufSize = 0;
  auto data = reinterpret_cast<char*>(
      pool_data_resolver(ctx, pool, ptr, GAPIL_READ, &bufSize));

  uint64_t len = 0;
  for (; len < bufSize; len++) {
    if (data[len] == 0) {
      len++;  // Include null-terminator in the slice.
      break;
    }
  }

  slice s = {0};
  s.pool = pool;  // application pool
  s.root = ptr;
  s.base = ptr;
  s.size = len;
  s.count = len;
  *out = s;
  return;
}

string* gapil_make_string(arena* a, uint64_t length, void* data) {
  Arena* arena = reinterpret_cast<Arena*>(a);

  auto str = reinterpret_cast<string_t*>(
      arena->allocate(sizeof(string_t) + length + 1, 1));
  str->arena = a;
  str->ref_count = 1;
  str->length = length;

  if (data != nullptr) {
    memcpy(str->data, data, length);
    str->data[length] = 0;
  } else {
    memset(str->data, 0, length + 1);
  }

  DEBUG_PRINT("gapil_make_string(arena: %p, length: %" PRIu64
              ", data: '%s') -> %p",
              a, length, data, str);

  return str;
}

void gapil_free_string(string* str) {
  DEBUG_PRINT("gapil_free_string(str: %p, ref_count: %" PRIu32 ", len: %" PRIu64
              ", str: '%s' (%p))",
              str, str->ref_count, str->length, str->data, str->data);

  Arena* arena = reinterpret_cast<Arena*>(str->arena);
  arena->free(str);
}

string* gapil_slice_to_string(context* ctx, slice* sli) {
  DEBUG_PRINT("gapil_slice_to_string(" SLICE_FMT ")", SLICE_ARGS(sli));
  auto ptr = gapil_slice_data(ctx, sli, GAPIL_READ);
  // Trim null terminator from the string.
  if (sli->size > 0 && ((uint8_t*)ptr)[sli->size - 1] == 0) {
    sli->size--;
  }
  return gapil_make_string(ctx->arena, sli->size, ptr);
}

void gapil_string_to_slice(context* ctx, string* str, slice* out) {
  DEBUG_PRINT("gapil_string_to_slice(str: '%s')", str->data);

  auto pool = gapil_make_pool(ctx, str->length);

  memcpy(pool->buffer, str->data, str->length);

  out->pool = pool;
  out->base = 0;
  out->root = 0;
  out->size = str->length;
  out->count = str->length;
}

string* gapil_string_concat(string* a, string* b) {
  DEBUG_PRINT("gapil_string_concat(a: '%s', b: '%s')", a->data, b->data);

  if (a->length == 0) {
    b->ref_count++;
    return b;
  }
  if (b->length == 0) {
    a->ref_count++;
    return a;
  }

  GAPID_ASSERT_MSG(a->arena != nullptr,
                   "string concat using string with no arena");
  GAPID_ASSERT_MSG(b->arena != nullptr,
                   "string concat using string with no arena");

  auto str = gapil_make_string(a->arena, a->length + b->length, nullptr);
  memcpy(str->data, a->data, a->length);
  memcpy(str->data + a->length, b->data, b->length);
  return str;
}

int32_t gapil_string_compare(string* a, string* b) {
  DEBUG_PRINT("gapil_string_compare(a: '%s', b: '%s')", a->data, b->data);
  if (a == b) {
    return 0;
  }
  return strncmp(reinterpret_cast<const char*>(a->data),
                 reinterpret_cast<const char*>(b->data),
                 std::max(a->length, b->length));
}

void gapil_store_in_database(context* ctx, void* ptr, uint64_t size,
                             uint8_t* id_out) {
  GAPID_ASSERT_MSG(database_storer != nullptr, "No database storer set");
  database_storer(ctx, ptr, size, id_out);
}

}  // extern "C"
