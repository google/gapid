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

extern "C" {

void gapil_logf(uint8_t severity, uint8_t* file, uint32_t line, uint8_t* fmt,
                ...) {
  // core/log/severity.go is in reverse order to log.h! :(
  severity = 5 - severity;
  if (GAPID_SHOULD_LOG(severity)) {
    va_list args;
    va_start(args, fmt);
    auto f =
        (file != nullptr) ? reinterpret_cast<const char*>(file) : "<unknown>";

    ::core::Logger::instance().vlogf(severity, f, line,
                                     reinterpret_cast<const char*>(fmt), args);
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

int32_t gapil_string_compare(string* a, string* b) {
  DEBUG_PRINT("gapil_string_compare(a: '%s', b: '%s')", a->data, b->data);
  if (a == b) {
    return 0;
  }
  return strncmp(reinterpret_cast<const char*>(a->data),
                 reinterpret_cast<const char*>(b->data),
                 std::max(a->length, b->length));
}

}  // extern "C"
