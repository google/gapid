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

// Implemented in env.cpp
context* create_context(TCreateContext* func, arena* a);
void destroy_context(TDestroyContext* func, context* ctx);
uint32_t call(context* ctx, TFunc* func);
void set_callbacks();

// Implemented in env.go
void* pool_data_resolver(context*, pool*, uint64_t ptr, gapil_data_access,
                         uint64_t* len);
void database_storer(context* ctx, void* ptr, uint64_t size, uint8_t* id_out);

#ifdef __cplusplus
}  // extern "C"
#endif  // __cplusplus
