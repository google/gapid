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

#ifndef __GAPIL_RUNTIME_H__
#define __GAPIL_RUNTIME_H__

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif  // __cplusplus

typedef struct arena_t arena;
typedef struct pool_t pool;

#define GAPIL_MAP_ELEMENT_EMPTY 0
#define GAPIL_MAP_ELEMENT_FULL 1
#define GAPIL_MAP_ELEMENT_USED 2

#define GAPIL_MAP_GROW_MULTIPLIER 4
#define GAPIL_MIN_MAP_SIZE 32
#define GAPIL_MAP_MAX_CAPACITY 0.8f

// context contains information about the environment in which a function is
// executing.
typedef struct context_t {
  arena* arena;            // the memory arena used for allocations.
  uint32_t* next_pool_id;  // the identifier of the next pool to be created.
} context;

// pool describes the underlying buffer that may be used by one or more slices.
typedef struct pool_t {
  uint32_t ref_count;  // number of owners of this pool.
  uint32_t id;         // unique identifier of this pool.
  uint64_t size;       // total size of the pool in bytes.
  arena* arena;  // arena that owns the allocation of this pool and its buffer.
  void* buffer;  // nullptr for application pool
} pool;

////////////////////////////////////////////////////////////////////////////////
// Runtime API implemented in runtime.cpp                                     //
////////////////////////////////////////////////////////////////////////////////

// allocates a new slice and underlying pool with the given size.
pool* gapil_make_pool(context*, uint64_t size);

// frees a pool previously allocated with gapil_make_pool.
void gapil_free_pool(pool*);

#ifdef __cplusplus
}  // extern "C"
#endif  // __cplusplus

#endif  // __GAPIL_RUNTIME_H__
