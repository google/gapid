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

#include "gapil/runtime/cc/runtime.h"

#ifdef __cplusplus
extern "C" {
#endif  // __cplusplus

typedef context*(TCreateContext)(arena*);
typedef void(TDestroyContext)(context*);
typedef uint32_t(TFunc)(void* ctx);
typedef void gapil_extern(context*, void* args, void* res);

// Implemented in env.cpp
context* create_context(TCreateContext* func, arena* a);
void destroy_context(TDestroyContext* func, context* ctx);
uint32_t call(context* ctx, TFunc* func);

void register_c_extern(const char* name, gapil_extern* fn);

typedef struct callbacks_t {
  void* apply_reads;
  void* apply_writes;
  void* resolve_pool_data;
  void* call_extern;
  void* store_in_database;
} callbacks;

void set_callbacks(callbacks*);

#ifdef __cplusplus
}  // extern "C"
#endif  // __cplusplus
