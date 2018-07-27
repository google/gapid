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

#include "gapis/api/gles/ctypes.h"

#include <stdint.h>

// This is very hot code. Regardless of build settings, make this fast.
#pragma GCC optimize("O3")

void externIndexLimits(gapil_context* ctx, IndexLimits_args* args,
                       IndexLimits_res* out) {
  gapil_slice s = args->indices;
  void* data = gapil_resolve_pool_data(ctx, s.pool, s.root, GAPIL_READ, s.size);
  switch (args->sizeof_index) {
#define IMPL_CASE(SIZE, TYPE)            \
  case SIZE: {                           \
    TYPE* indices = (TYPE*)data;         \
    size_t count = s.size / SIZE;        \
    TYPE min = (TYPE)0xffffffffffffffff; \
    TYPE max = (TYPE)0x0000000000000000; \
    for (size_t i = 0; i < count; i++) { \
      TYPE val = indices[i];             \
      min = (val < min) ? val : min;     \
      max = (val > max) ? val : max;     \
    }                                    \
    out->first = min;                    \
    out->count = max - min;              \
    break;                               \
  }
    IMPL_CASE(1, uint8_t)
    IMPL_CASE(2, uint16_t)
    IMPL_CASE(4, uint32_t)
    IMPL_CASE(8, uint64_t)
    default:
      gapil_logf(GAPIL_LOG_LEVEL_FATAL, 0, 0, "Unhandled index size %d",
                 (int)args->sizeof_index);
  }
}
