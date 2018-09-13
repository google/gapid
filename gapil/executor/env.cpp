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

#include "env.h"

extern "C" {

context* create_context(TCreateContext* func, arena* a) { return func(a); }

void destroy_context(TDestroyContext* func, context* ctx) { func(ctx); }

uint32_t call(context* ctx, TFunc* fptr) {
  try {
    fptr(ctx);
    return 0;
  } catch (uint32_t err) {
    return err;
  }
}

void set_callbacks(callbacks* go_cbs) {
  gapil_runtime_callbacks cbs = {0};
  cbs.apply_reads =
      reinterpret_cast<decltype(cbs.apply_reads)>(go_cbs->apply_reads),
  cbs.apply_writes =
      reinterpret_cast<decltype(cbs.apply_writes)>(go_cbs->apply_writes),
  cbs.resolve_pool_data = reinterpret_cast<decltype(cbs.resolve_pool_data)>(
      go_cbs->resolve_pool_data),
  cbs.store_in_database = reinterpret_cast<decltype(cbs.store_in_database)>(
      go_cbs->store_in_database),
  gapil_set_runtime_callbacks(&cbs);
}

}  // extern "C"
