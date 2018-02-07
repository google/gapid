// Copyright (C) 2017 The Android Open Source Project
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

#ifndef CORE_ARENA_H
#define CORE_ARENA_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus

#include <unordered_set>

namespace core {

// Arena is a memory allocator that owns each of the allocations made by it.
// If there are any outstanding allocations when the Arena is destructed then
// these allocations are automatically freed.
class Arena {
public:
    Arena();
    ~Arena();

    // allocates a contiguous block of memory of at least the requested size and
    // alignment.
    void* allocate(uint32_t size, uint32_t align);

    // reallocates a block of memory previously allocated by this arena.
    // Data held in the previous allocation will be copied to the reallocated
    // address, but data may be trimmed if the new size is smaller than the
    // previous allocation.
    void* reallocate(void* ptr, uint32_t size, uint32_t align);

    // free releases the memory previously allocated by this arena.
    // Once the memory is freed, it must not be used.
    void free(void* ptr);

    // owns returns true if ptr is owned by this arena.
    bool owns(void* ptr);

    // create constructs and returns a pointer to a new T.
    template<typename T>
    inline T* create();

    // destroy destructs an object constructed with create<T>().
    template<typename T>
    inline void destroy(T* ptr);

    // returns the total number of allocations owned by this arena.
    size_t num_allocations() const;

    // returns the total number of bytes allocated by this arena.
    size_t num_bytes_allocated() const;

private:
    std::unordered_set<void*> allocations;
};

template<typename T>
inline T* Arena::create() {
    auto buf = allocate(sizeof(T), alignof(T));
    return new(buf) T;
}

template<typename T>
inline void Arena::destroy(T* ptr) {
    ptr->~T();
    free(ptr);
}


}  // namespace core

extern "C" {
#endif

// C handle for an arena.
typedef struct arena_t arena;

// arena_create constructs and returns a new arena.
arena* arena_create();

// arena_destroy destructs the specified arena, freeing all allocations
// made by that arena. Once destroyed, you must not use the arena.
void arena_destroy(arena* arena);

// arena_alloc creates a memory allocation in the specified arena of the
// given size and alignment.
void* arena_alloc(arena* arena, uint32_t size, uint32_t align);

// arena_realloc reallocates the memory at ptr to the new size and
// alignment. ptr must have been allocated from arena.
void* arena_realloc(arena* arena, void* ptr, uint32_t size, uint32_t align);

// arena_free deallocates the memory at ptr. ptr must have been allocated
// from arena.
void arena_free(arena* arena, void* ptr);

// arena_stats returns statistics of the current state of the arena.
void arena_stats(arena* arena, size_t* num_allocations, size_t* num_bytes_allocated);

#ifdef __cplusplus
} // extern "C"
#endif

#endif //  CORE_ARENA_H
