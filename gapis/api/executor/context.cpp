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

#include "core/cc/assert.h"
#include "core/cc/log.h"
#include "core/memory/arena/cc/arena.h"

#if 1
#include <cstdio> // printf debug
#define DEBUG_PRINT(fmt, ...) fprintf(stderr, fmt "\n", ##__VA_ARGS__)
#else
#define DEBUG_PRINT(...)
#endif

using core::Arena;

extern "C" {

#include "context.h"

context* create_context(uint32_t id, globals* globals) {
    auto ctx = new context;
    gapil_init_context(ctx);
    ctx->id = id;
    ctx->globals = globals;
    return ctx;
}

void destroy_context(context* ctx) {
    gapil_term_context(ctx);
    delete ctx;
}

void init_context(context* ctx, TInit* func) {
    func(ctx);
}

uint32_t call(context* ctx, void* args, TFunc* func) {
    return func(ctx, args);
}

}  // extern "C"