// Copyright (C) 2017 Google Inc.
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

#ifndef GAPIS_API_EXECUTOR_CONTEXT_H
#define GAPIS_API_EXECUTOR_CONTEXT_H

#include "gapil/compiler/cc/builtins.h"

typedef struct arena_t arena;

typedef struct exec_context_t {
    context  ctx;
    arena*   arena;
} exec_context;

typedef void     (TInit) (void* ctx);
typedef uint32_t (TFunc) (void* ctx, void* args);

exec_context* create_context(uint32_t id, globals* globals, arena* a);
void          destroy_context(exec_context*);
void          init_context(exec_context* ctx, TInit* func);
uint32_t      call(exec_context* ctx, void* args, TFunc* func);
void*         alloc(exec_context* ctx, uint32_t size, uint32_t align);

#endif // GAPIS_API_EXECUTOR_CONTEXT_H