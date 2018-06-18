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

using core::Arena;

namespace {

void* default_pointer_remapper(context* ctx, uintptr_t pointer,
                               uint64_t length) {
  return reinterpret_cast<void*>(pointer);
}

void default_code_locator(context* ctx, char** file, uint32_t* line) {}

static gapil_pointer_remapper* pointer_remapper = &default_pointer_remapper;
static gapil_get_code_location* code_locator = &default_code_locator;

}  // anonymous namespace

extern "C" {

// sets the pointer remapper callback used to remap serialized pointers to a
// pointer in an allocated buffer.
void gapil_set_pointer_remapper(gapil_pointer_remapper* cb) {
  pointer_remapper = cb != nullptr ? cb : &default_pointer_remapper;
}

// sets the callback used to fetch the file and line location for the current
// source location within the .api file.
void gapil_set_code_locator(gapil_get_code_location* cb) {
  code_locator = cb != nullptr ? cb : &default_code_locator;
}

void gapil_logf(context* ctx, uint8_t severity, uint8_t* fmt, ...) {
  // core/log/severity.go is in reverse order to log.h! :(
  severity = 5 - severity;
  if (GAPID_SHOULD_LOG(severity)) {
    va_list args;
    va_start(args, fmt);
    char* file = nullptr;
    uint32_t line = 0;
    code_locator(ctx, &file, &line);
#if TARGET_OS == GAPID_OS_ANDROID
    char buf[2048];
    snprintf(buf, sizeof(buf), "[%s:%u] %s", file ? file : "<unknown>", line,
             fmt);
    __android_log_vprint(severity, "GAPID", buf, args);
#else
    ::core::Logger::instance().vlogf(severity, file ? file : "<unknown>", line,
                                     reinterpret_cast<const char*>(fmt), args);
#endif  // TARGET_OS
    if (file != nullptr) {
      free(file);
    }
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

void gapil_copy_slice(context* ctx, slice* dst, slice* src) {
  DEBUG_PRINT(
      "gapil_copy_slice(ctx: %p,\n"
      "    dst: [pool: %p, root: %p, base: %p, size: 0x%" PRIx64
      "],\n"
      "    src: [pool: %p, root: %p, base: %p, size: 0x%" PRIx64 "])",
      ctx, dst->pool, dst->root, dst->base, dst->size, src->pool, src->root,
      src->base, src->size);

  auto size = std::min(dst->size, src->size);
  memcpy(dst->base, src->base, size);
}

void gapil_pointer_to_slice(context* ctx, uintptr_t ptr, uint64_t offset,
                            uint64_t size, uint64_t count, slice* out) {
  DEBUG_PRINT("gapil_pointer_to_slice(ptr: 0x%" PRIx64 ", offset: 0x%" PRIx64
              ", size: 0x%" PRIx64 ", count: 0x%" PRIx64 ")",
              ptr, offset, size, count);

  auto end = ptr + offset + size;
  auto root = reinterpret_cast<uint8_t*>(pointer_remapper(ctx, ptr, end - ptr));
  auto base = root + offset;

  out->pool = nullptr;  // application pool
  out->root = root;
  out->base = base;
  out->size = size;
  out->count = count;
}

string* gapil_pointer_to_string(context* ctx, uintptr_t ptr) {
  DEBUG_PRINT("gapil_pointer_to_string(ptr: 0x%" PRIx64 ")", ptr);

  auto data = reinterpret_cast<char*>(pointer_remapper(ctx, ptr, 1));
  auto len = strlen(data);

  return gapil_make_string(ctx->arena, len, data);
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

string* gapil_slice_to_string(context* ctx, slice* slice) {
  DEBUG_PRINT("gapil_slice_to_string(base: %p, size: 0x%" PRIx64 ", pool: %p)",
              slice->base, slice->size, slice->pool);

  return gapil_make_string(ctx->arena, slice->size, slice->base);
}

void gapil_string_to_slice(context* ctx, string* str, slice* out) {
  DEBUG_PRINT("gapil_string_to_slice(str: '%s')", str->data);

  auto pool = gapil_make_pool(ctx, str->length);

  memcpy(pool->buffer, str->data, str->length);

  out->pool = pool;
  out->base = pool->buffer;
  out->root = pool->buffer;
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

}  // extern "C"
