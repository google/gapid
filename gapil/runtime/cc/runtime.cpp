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
#include <cstdio>  // for snprintf
#endif

#define __STDC_FORMAT_MACROS
#include <inttypes.h>

#include <cstring>

#if 0
#define DEBUG_PRINT(...) GAPID_WARNING(__VA_ARGS__)
#else
#define DEBUG_PRINT(...)
#endif

#define SLICE_FMT                                                         \
  "[pool: %p, root: 0x%" PRIx64 ", base: 0x%" PRIx64 ", size: 0x%" PRIx64 \
  ", count: 0x%" PRIx64 "]"
#define SLICE_ARGS(sli) sli->pool, sli->root, sli->base, sli->size, sli->count

using core::Arena;

namespace {

gapil_runtime_callbacks callbacks = {0};

}  // anonymous namespace

extern "C" {

void gapil_abort() { throw uint32_t(GAPIL_ERR_ABORTED); }

void gapil_set_runtime_callbacks(gapil_runtime_callbacks* cbs) {
  callbacks = *cbs;
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

void gapil_any_reference(gapil_any* a) {
  if (a != nullptr) {
    GAPID_ASSERT_MSG(a->ref_count > 0, "Attempting to reference released any");
    a->ref_count++;
  }
}

void gapil_any_release(gapil_any* a) {
  if (a != nullptr) {
    GAPID_ASSERT_MSG(a->ref_count > 0, "Attempting to release released any");
    a->ref_count--;
    if (a->ref_count == 0) {
      if (a->rtti->release != nullptr) {
        a->rtti->release(a->value);
      }
      reinterpret_cast<Arena*>(a->arena)->free(a);
    }
  }
}

void gapil_msg_reference(gapil_msg* m) {
  if (m != nullptr) {
    GAPID_ASSERT_MSG(m->ref_count > 0, "Attempting to reference released msg");
    m->ref_count++;
  }
}

void gapil_msg_release(gapil_msg* m) {
  if (m != nullptr) {
    GAPID_ASSERT_MSG(m->ref_count > 0, "Attempting to release released msg");
    m->ref_count--;
    if (m->ref_count == 0) {
      auto args = m->args;
      while (args->name != nullptr) {
        gapil_any_release(args->value);
        args++;
      }
      auto a = reinterpret_cast<Arena*>(m->arena);
      a->free(m->args);
      a->free(m);
    }
  }
}

void gapil_string_reference(gapil_string* s) {
  GAPID_ASSERT_MSG(s != nullptr, "gapil_string_reference passed null string");
  GAPID_ASSERT_MSG(s->ref_count > 0, "Attempting to reference released string");
  s->ref_count++;
}

void gapil_string_release(gapil_string* s) {
  GAPID_ASSERT_MSG(s != nullptr, "gapil_string_release passed null string");
  GAPID_ASSERT_MSG(s->ref_count > 0, "Attempting to release released string");
  s->ref_count--;
  if (s->ref_count == 0) {
    Arena* a = reinterpret_cast<Arena*>(s->arena);
    a->free(s);
  }
}

void gapil_slice_reference(gapil_slice* s) {
  GAPID_ASSERT_MSG(s != nullptr, "gapil_slice_reference passed null slice");
  if (auto p = s->pool) {
    GAPID_ASSERT_MSG(p->ref_count > 0, "Attempting to reference released pool");
    p->ref_count++;
  }
}

void gapil_slice_release(gapil_slice* s) {
  GAPID_ASSERT_MSG(s != nullptr, "gapil_slice_release passed null slice");
  if (auto p = s->pool) {
    GAPID_ASSERT_MSG(p->ref_count > 0, "Attempting to release released string");
    p->ref_count--;
    if (p->ref_count == 0) {
      gapil_free_pool(p);
    }
  }
}

void* gapil_alloc(arena_t* a, uint64_t size, uint64_t align) {
  GAPID_ASSERT_MSG(a != nullptr, "arena is null");
  Arena* arena = reinterpret_cast<Arena*>(a);
  void* ptr = arena->allocate(size, align);
  memset(ptr, 0, size);
  DEBUG_PRINT("gapil_alloc(size: 0x%" PRIx64 ", align: 0x%" PRIx64 ") -> %p",
              size, align, ptr);
  return ptr;
}

void* gapil_realloc(arena_t* a, void* ptr, uint64_t size, uint64_t align) {
  GAPID_ASSERT_MSG(a != nullptr, "arena is null");
  Arena* arena = reinterpret_cast<Arena*>(a);
  void* retptr = arena->reallocate(ptr, size, align);
  DEBUG_PRINT("gapil_realloc(ptr: %p, 0x%" PRIx64 ", align: 0x%" PRIx64
              ") -> %p",
              ptr, size, align, retptr);
  return retptr;
}

void gapil_free(arena_t* a, void* ptr) {
  DEBUG_PRINT("gapil_free(ptr: %p)", ptr);
  GAPID_ASSERT_MSG(a != nullptr, "arena is null");
  Arena* arena = reinterpret_cast<Arena*>(a);
  arena->free(ptr);
}

void gapil_create_buffer(arena* a, uint64_t capacity, uint64_t alignment,
                         gapil_buffer* buf) {
  DEBUG_PRINT("gapil_create_buffer(capacity: %" PRId64 ", alignment: %" PRId64
              ")",
              capacity, alignment);
  GAPID_ASSERT_MSG(a != nullptr, "arena is null");
  Arena* arena = reinterpret_cast<Arena*>(a);
  buf->arena = a;
  buf->data = (uint8_t*)arena->allocate(capacity, alignment);
  buf->size = 0;
  buf->capacity = capacity;
  buf->alignment = alignment;
}

void gapil_destroy_buffer(gapil_buffer* buf) {
  DEBUG_PRINT("gapil_destroy_buffer()");
  Arena* arena = reinterpret_cast<Arena*>(buf->arena);
  arena->free(buf->data);
  buf->capacity = 0;
  buf->size = 0;
}

void gapil_append_buffer(gapil_buffer* buf, const void* data, uint64_t size) {
  DEBUG_PRINT("gapil_append_buffer(data: %p, size: %" PRId64 ")", data, size);
  auto new_size = buf->size + size;
  if (new_size > buf->capacity) {
    Arena* arena = reinterpret_cast<Arena*>(buf->arena);
    buf->capacity = std::max<uint32_t>(buf->size + size, buf->capacity * 2);
    buf->data =
        (uint8_t*)arena->reallocate(buf->data, buf->capacity, buf->alignment);
  }
  memcpy(buf->data + buf->size, data, size);
  buf->size = new_size;
}

void* gapil_slice_data(gapil_context* ctx, gapil_slice* sli,
                       gapil_data_access access) {
  auto ptr =
      gapil_resolve_pool_data(ctx, sli->pool, sli->base, access, sli->size);
  DEBUG_PRINT("gapil_slice_data(" SLICE_FMT ", %d) -> %p", SLICE_ARGS(sli),
              access, ptr);
  return ptr;
}

gapil_string* gapil_make_string(arena* a, uint64_t length, void* data) {
  GAPID_ASSERT_MSG(a != nullptr, "arena is null");
  Arena* arena = reinterpret_cast<Arena*>(a);

  auto str = reinterpret_cast<gapil_string_t*>(
      arena->allocate(sizeof(gapil_string_t) + length + 1, 1));
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

void gapil_free_string(gapil_string* str) {
  DEBUG_PRINT("gapil_free_string(str: %p, ref_count: %" PRIu32 ", len: %" PRIu64
              ", str: '%s' (%p))",
              str, str->ref_count, str->length, str->data, str->data);

  Arena* arena = reinterpret_cast<Arena*>(str->arena);
  arena->free(str);
}

gapil_string* gapil_slice_to_string(gapil_context* ctx, gapil_slice* sli) {
  DEBUG_PRINT("gapil_slice_to_string(" SLICE_FMT ")", SLICE_ARGS(sli));
  auto ptr = gapil_slice_data(ctx, sli, GAPIL_READ);
  // Trim null terminator from the string.
  if (sli->size > 0 && ((uint8_t*)ptr)[sli->size - 1] == 0) {
    sli->size--;
  }
  return gapil_make_string(ctx->arena, sli->size, ptr);
}

void gapil_string_to_slice(gapil_context* ctx, gapil_string* str,
                           gapil_slice* out) {
  DEBUG_PRINT("gapil_string_to_slice(str: '%s')", str->data);

  auto pool = gapil_make_pool(ctx, str->length);
  auto buf = reinterpret_cast<uint8_t*>(
      gapil_resolve_pool_data(ctx, pool, 0, GAPIL_WRITE, str->length + 1));
  memcpy(buf, str->data, str->length);
  str->data[str->length] = 0;  // Null-terminate

  out->pool = pool;
  out->base = 0;
  out->root = 0;
  out->size = str->length;
  out->count = str->length;
}

gapil_string* gapil_string_concat(gapil_string* a, gapil_string* b) {
  DEBUG_PRINT("gapil_string_concat(a: '%s', b: '%s')", a->data, b->data);
  GAPID_ASSERT(a->ref_count > 0);
  GAPID_ASSERT(b->ref_count > 0);

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

int32_t gapil_string_compare(gapil_string* a, gapil_string* b) {
  DEBUG_PRINT("gapil_string_compare(a: '%s', b: '%s')", a->data, b->data);
  if (a == b) {
    return 0;
  }
  return strncmp(reinterpret_cast<const char*>(a->data),
                 reinterpret_cast<const char*>(b->data),
                 std::max(a->length, b->length));
}

void gapil_apply_reads(gapil_context* ctx) {
  DEBUG_PRINT("gapil_apply_reads(ctx: %p)", ctx);
  GAPID_ASSERT(callbacks.apply_reads != nullptr);
  callbacks.apply_reads(ctx);
}

void gapil_apply_writes(gapil_context* ctx) {
  DEBUG_PRINT("gapil_apply_writes(ctx: %p)", ctx);
  GAPID_ASSERT(callbacks.apply_writes != nullptr);
  callbacks.apply_writes(ctx);
}

void* gapil_resolve_pool_data(gapil_context* ctx, gapil_pool* pool,
                              uint64_t ptr, gapil_data_access access,
                              uint64_t size) {
  DEBUG_PRINT("gapil_resolve_pool_data(ctx: %p, pool: %p, ptr: 0x%" PRIx64
              ", access: %d, size: 0x%" PRIx64 ")",
              ctx, pool, ptr, access, size);
  GAPID_ASSERT(callbacks.resolve_pool_data != nullptr);
  return callbacks.resolve_pool_data(ctx, pool, ptr, access, size);
}

void gapil_copy_slice(gapil_context* ctx, gapil_slice* dst, gapil_slice* src) {
  DEBUG_PRINT(
      "gapil_copy_slice(ctx: %p,\n"
      "    dst: " SLICE_FMT
      ",\n"
      "    src: " SLICE_FMT ")",
      ctx, SLICE_ARGS(dst), SLICE_ARGS(src));

  GAPID_ASSERT(callbacks.copy_slice != nullptr);
  return callbacks.copy_slice(ctx, dst, src);
}

void gapil_cstring_to_slice(gapil_context* ctx, uint64_t ptr,
                            gapil_slice* out) {
  DEBUG_PRINT("gapil_cstring_to_slice(ctx: %p, ptr: 0x%" PRIx64 ", out: %p)",
              ctx, ptr, out);

  GAPID_ASSERT(callbacks.cstring_to_slice != nullptr);
  return callbacks.cstring_to_slice(ctx, ptr, out);
}

void gapil_store_in_database(gapil_context* ctx, void* ptr, uint64_t size,
                             uint8_t* id_out) {
  DEBUG_PRINT("gapil_store_in_database(ctx: %p, ptr: %p, size: 0x%" PRIx64
              ", id_out:  %p)",
              ctx, ptr, size, id_out);
  GAPID_ASSERT(callbacks.store_in_database != nullptr);
  callbacks.store_in_database(ctx, ptr, size, id_out);
}

gapil_pool* gapil_make_pool(gapil_context* ctx, uint64_t size) {
  DEBUG_PRINT("gapil_make_pool(ctx: %p, size: %" PRIu64 ")", ctx, size);
  GAPID_ASSERT(callbacks.make_pool != nullptr);
  return callbacks.make_pool(ctx, size);
}

void gapil_free_pool(gapil_pool* pool) {
  DEBUG_PRINT("gapil_free_pool(pool: %p)", pool);
  GAPID_ASSERT(callbacks.free_pool != nullptr);
  callbacks.free_pool(pool);
}

void gapil_call_extern(gapil_context* ctx, uint8_t* name, void* args,
                       void* res) {
  DEBUG_PRINT("gapil_call_extern(ctx: %p, name: %s, args: %p, res: %p)", ctx,
              name, args, res);
  GAPID_ASSERT(callbacks.call_extern != nullptr);
  callbacks.call_extern(ctx, name, args, res);
}

}  // extern "C"
