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

}  // extern "C"
