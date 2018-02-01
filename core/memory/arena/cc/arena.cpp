/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "arena.h"

#include "core/cc/target.h"
#include "core/cc/assert.h"

#include <stdlib.h>

#if TARGET_OS == GAPID_OS_OSX
#include <malloc/malloc.h> // malloc_size
#else
#include <malloc.h> // malloc_size,malloc_usable_size, dlmalloc_usable_size
#endif

namespace {

size_t alloc_size(void* ptr) {
    WINDOWS_ONLY(return _msize(ptr));
    OSX_ONLY(return malloc_size(ptr));
    LINUX_ONLY(return malloc_usable_size(ptr));
    ANDROID_ONLY(return dlmalloc_usable_size(ptr));
    return 0;
}

}  // anonymous namespace

namespace core {

Arena::Arena() {}

Arena::~Arena() {
    for (void* ptr : allocations) {
        ::free(ptr);
    }
    allocations.clear();
}

void* Arena::allocate(uint32_t size, uint32_t align) {
    void* ptr = malloc(size); // TODO: alignment
    allocations.insert(ptr);
    return ptr;
}

void* Arena::reallocate(void* ptr, uint32_t size, uint32_t align) {
    GAPID_ASSERT_MSG(this->owns(ptr), "ptr: %p", ptr);
    void* newptr = realloc(ptr, size); // TODO: alignment
    allocations.erase(ptr);
    allocations.insert(newptr);
    return newptr;
}

void Arena::free(void* ptr) {
    GAPID_ASSERT_MSG(this->owns(ptr), "ptr: %p", ptr);
    allocations.erase(ptr);
    ::free(ptr);
}

bool Arena::owns(void* ptr) {
    return allocations.count(ptr) == 1;
}

void Arena::stats(size_t* num_allocations, size_t* num_bytes_allocated) const {
    size_t bytes = 0;
    for (void* ptr : allocations) {
        bytes += alloc_size(ptr);
    }
    *num_allocations = allocations.size();
    *num_bytes_allocated = bytes;
}

}  // namespace core

extern "C" {

arena* arena_create() {
    return reinterpret_cast<arena*>(new core::Arena());
}

void arena_destroy(arena* a) {
    delete reinterpret_cast<core::Arena*>(a);
}

void* arena_alloc(arena* a, uint32_t size, uint32_t align) {
    return reinterpret_cast<core::Arena*>(a)->allocate(size, align);
}

void* arena_realloc(arena* a, void* ptr, uint32_t size, uint32_t align) {
    return reinterpret_cast<core::Arena*>(a)->reallocate(ptr, size, align);
}

void arena_free(arena* a, void* ptr) {
    reinterpret_cast<core::Arena*>(a)->free(ptr);
}

// arena_stats returns statistics of the current state of the arena.
void arena_stats(arena* a, size_t* num_allocations, size_t* num_bytes_allocated) {
    reinterpret_cast<core::Arena*>(a)->stats(num_allocations, num_bytes_allocated);
}

} // extern "C"

