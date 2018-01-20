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

#include <stddef.h>
#include <stdlib.h>

#if 1
#include <cstdio> // printf debug
#define DEBUG_PRINT(fmt, ...) fprintf(stderr, fmt "\n", ##__VA_ARGS__)
#else
#define DEBUG_PRINT(...)
#endif

using core::Arena;

extern "C" {

#include "context.h"
#include "memory.h"

// Declared in env.go
void* gapil_remap_pointer(context* ctx, uint64_t pointer, uint64_t length);
void  gapil_get_code_location(context* ctx, char** file, uint32_t* line);

exec_context* create_context(uint32_t id, globals* globals, arena* a) {
    auto app_pool = new pool;
    app_pool->ref_count = 1;
    app_pool->buffer = nullptr;

    auto empty_string = new string;
    empty_string->ref_count = 1;
    empty_string->length = 0;
    empty_string->data[0] = 0;

    Arena* arena = reinterpret_cast<Arena*>(a);
    exec_context* ec = new exec_context();
    ec->ctx.id = id;
    ec->ctx.location = 0xffffffff;
    ec->ctx.globals = globals;
    ec->ctx.app_pool = app_pool;
    ec->ctx.empty_string = empty_string;
    ec->arena = a;
    return ec;
}

void destroy_context(exec_context* ec) {
    delete ec->ctx.empty_string;
    delete ec->ctx.app_pool;
    delete ec;
}

void init_context(exec_context* ec, TInit* func) {
    func(ec);
}

uint32_t call(exec_context* ec, void* args, TFunc* func) {
    return func(ec, args);
}

void gapil_logf(context* ctx, uint8_t severity, uint8_t* fmt, ...) {
    severity = 5 - severity; // core/log/severity.go is in reverse order to log.h! :(
    if (::core::Logger::level() >= severity) {
        va_list args;
        va_start(args, fmt);
        char* file = nullptr;
        uint32_t line = 0;
        gapil_get_code_location(ctx, &file, &line);
        ::core::Logger::instance().vlogf(severity, file, line, reinterpret_cast<const char*>(fmt), args);
        if (file != nullptr) {
            free(file);
        }
        va_end(args);
    }
}

void* gapil_alloc(context* ctx, uint64_t size, uint64_t align) {
    auto ec = reinterpret_cast<exec_context*>(ctx);

    Arena* arena = reinterpret_cast<Arena*>(ec->arena);
    void* ptr = arena->allocate(size, align);
    DEBUG_PRINT("gapil_alloc(size: 0x%llx, align: 0x%llx) -> 0x%p", size, align, ptr);
    return ptr;
}

void* gapil_realloc(context* ctx, void* ptr, uint64_t size, uint64_t align) {
    auto ec = reinterpret_cast<exec_context*>(ctx);

    Arena* arena = reinterpret_cast<Arena*>(ec->arena);
    void* retptr = arena->reallocate(ptr, size, align);
    DEBUG_PRINT("gapil_realloc(ptr: %p, 0x%llx, align: 0x%llx) -> 0x%p", ptr, size, align, retptr);
    return retptr;
}

void gapil_free(context* ctx, void* ptr) {
    DEBUG_PRINT("gapil_free(ptr: %p)", ptr);
    auto ec = reinterpret_cast<exec_context*>(ctx);

    Arena* arena = reinterpret_cast<Arena*>(ec->arena);
    arena->free(ptr);
}

pool* gapil_make_pool(context* ctx, uint64_t size) {
    auto ec = reinterpret_cast<exec_context*>(ctx);
    Arena* arena = reinterpret_cast<Arena*>(ec->arena);

    auto buffer = arena->allocate(size, 16);
    memset(buffer, 0, size);

    auto pool = arena->create<pool_t>();
    pool->ref_count = 1;
    pool->buffer = buffer;

    DEBUG_PRINT("gapil_make_pool(size: %llu) -> [pool: %p, buffer: %p]", size, pool, buffer);
    return pool;
}

void gapil_free_pool(context* ctx, pool* pool) {
    DEBUG_PRINT("gapil_free_pool(pool: %p)", pool);
    auto ec = reinterpret_cast<exec_context*>(ctx);
    Arena* arena = reinterpret_cast<Arena*>(ec->arena);

    if (pool == ctx->app_pool) {
        // TODO: Panic?
        return;
    }

    arena->free(pool->buffer);
    arena->destroy(pool);
}

// TODO: Change this to gapil_make_pool for symetry?
void gapil_make_slice(context* ctx, uint64_t size, slice* out) {
    DEBUG_PRINT("gapil_make_slice(size: 0x%llx)", size);
    auto ec = reinterpret_cast<exec_context*>(ctx);
    Arena* arena = reinterpret_cast<Arena*>(ec->arena);

    auto pool = gapil_make_pool(ctx, size);

    *out = slice{pool, pool->buffer, pool->buffer, size};
}

void gapil_copy_slice(context* ctx, slice* dst, slice* src) {
    DEBUG_PRINT("gapil_copy_slice(ctx: %p,\n"
                "                 dst: [pool: %p, root: %p, base: %p, size: 0x%llx],\n"
                "                 src: [pool: %p, root: %p, base: %p, size: 0x%llx])",
            ctx,
            dst->pool, dst->root, dst->base, dst->size,
            src->pool, src->root, src->base, src->size);

    auto size = std::min(dst->size, src->size);
    memcpy(dst->base, src->base, size);
}

void gapil_pointer_to_slice(context* ctx, uint64_t ptr, uint64_t offset, uint64_t size, slice* out) {
    DEBUG_PRINT("gapil_pointer_to_slice(ptr: 0x%llx, offset: 0x%llx, size: 0x%llx)", ptr, offset, size);
    auto ec = reinterpret_cast<exec_context*>(ctx);

    auto end = ptr + offset + size;
    auto root = reinterpret_cast<uint8_t*>(gapil_remap_pointer(ctx, ptr, end - ptr));
    auto base = root + offset;

    ctx->app_pool->ref_count++;

    out->pool = ctx->app_pool;
    out->root = root;
    out->base = base;
    out->size = size;
}

string* gapil_pointer_to_string(context* ctx, uint64_t ptr) {
    DEBUG_PRINT("gapil_pointer_to_string(ptr: 0x%llx)", ptr);
    auto ec = reinterpret_cast<exec_context*>(ctx);

    auto data = reinterpret_cast<char*>(gapil_remap_pointer(ctx, ptr, 1));
    auto len = strlen(data);

    return gapil_make_string(ctx, len, data);
}

string* gapil_make_string(context* ctx, uint64_t length, void* data) {
    auto ec = reinterpret_cast<exec_context*>(ctx);
    Arena* arena = reinterpret_cast<Arena*>(ec->arena);

    auto str = reinterpret_cast<string_t*>(arena->allocate(sizeof(string_t) + length + 1, 1));
    str->ref_count = 1;
    str->length = length;

    if (data != nullptr) {
        memcpy(str->data, data, length);
        str->data[length] = 0;
    } else {
        memset(str->data, 0, length + 1);
    }

    return str;
}

void gapil_free_string(context* ctx, string* str) {
    DEBUG_PRINT("gapil_free_string(ref_count: %d, len: %llu, str: '%s' (%p))",
        str->ref_count, str->length, str->data, str->data);

    GAPID_ASSERT_MSG(str != ctx->empty_string,
        "Attempting to free the global empty string. "
        "This suggests asymmetrical reference/release logic.");

    auto ec = reinterpret_cast<exec_context*>(ctx);
    Arena* arena = reinterpret_cast<Arena*>(ec->arena);

    arena->free(str);
}

string* gapil_slice_to_string(context* ctx, slice* slice) {
    DEBUG_PRINT("gapil_slice_to_string(base: %p, size: 0x%llx, pool: %p)", slice->base, slice->size, slice->pool);
    auto ec = reinterpret_cast<exec_context*>(ctx);

    return gapil_make_string(ctx, slice->size, slice->base);
}

void gapil_string_to_slice(context* ctx, string* str, slice* out) {
    DEBUG_PRINT("gapil_string_to_slice(str: '%s')", str->data);
    auto ec = reinterpret_cast<exec_context*>(ctx);

    gapil_make_slice(ctx, str->length, out);
    memcpy(out->base, str->data, str->length);
}

string* gapil_string_concat(context* ctx, string* a, string* b) {
    DEBUG_PRINT("gapil_string_concat(a: '%s', b: '%s')", a->data, b->data);
    auto ec = reinterpret_cast<exec_context*>(ctx);
    Arena* arena = reinterpret_cast<Arena*>(ec->arena);

    auto str = gapil_make_string(ctx, a->length + b->length, nullptr);
    memcpy(str->data, a->data, a->length);
    memcpy(str->data + a->length, b->data, b->length);
    return str;
}

int32_t gapil_string_compare(context* ctx, string* a, string* b) {
    DEBUG_PRINT("gapil_string_compare(a: '%s', b: '%s')", a->data, b->data);
    if (a == b) {
        return 0;
    }
    return strncmp(
        reinterpret_cast<const char*>(a->data),
        reinterpret_cast<const char*>(b->data),
        std::max(a->length, b->length)
    );
}

}  // extern "C"