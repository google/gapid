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

#include "gapil/runtime/cc/runtime.h"

typedef void     (TInit) (void* ctx);
typedef uint32_t (TFunc) (void* ctx, void* args);

context*  create_context(uint32_t id, globals* globals);
void      destroy_context(context*);
void      init_context(context* ctx, TInit* func);
uint32_t  call(context* ctx, void* args, TFunc* func);
void*     alloc(context* ctx, uint32_t size, uint32_t align);

#endif // GAPIS_API_EXECUTOR_CONTEXT_H