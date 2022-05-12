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
typedef struct string_t string;

typedef uint8_t gapil_char;

#define GAPIL_ERR_SUCCESS 0
#define GAPIL_ERR_ABORTED 1

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

// slice is the data of a gapil slice type (elty foo[]).
typedef struct slice_t {
  pool* pool;  // the underlying pool. nullptr represents the application pool.
  uint64_t root;   // original offset in bytes from pool base that this slice
                   // derives from.
  uint64_t base;   // offset in bytes from pool base of the first element.
  uint64_t size;   // size in bytes of the slice.
  uint64_t count;  // total number of elements in the slice.
} slice;

// string is the shared data of a gapil string type.
// A string is a pointer to this struct.
typedef struct string_t {
  uint32_t ref_count;  // number of owners of this string.
  arena* arena;        // arena that owns this string allocation.
  uint64_t length;  // size in bytes of this string (including null-terminator).
  uint8_t data[1];  // the null-terminated string bytes.
} string;

// map is the shared data of a gapil map type.
// A map is a pointer to this struct.
typedef struct map_t {
  uint32_t ref_count;  // number of owners of this map.
  arena* arena;  // arena that owns this map allocation and its elements buffer.
  uint64_t count;     // number of elements in the map.
  uint64_t capacity;  // size of the elements buffer.
  void* elements;     // pointer to the elements buffer.
} map;

// ref is the shared data of a gapil ref!T type.
// A ref is a pointer to this struct.
typedef struct ref_t {
  uint32_t ref_count;  // number of owners of this ref.
  arena* arena;        // arena that owns this ref allocation.
  /* T */              // referenced object immediately follows.
} ref;

// buffer is a structure used to hold a variable size byte array.
// buffer is used internally by the compiler to write out variable length data.
typedef struct buffer_t {
  arena* arena;        // the arena that owns the buffer data.
  uint8_t* data;       // buffer data.
  uint32_t capacity;   // total capacity of the buffer.
  uint32_t size;       // current size of the buffer.
  uint32_t alignment;  // min alignment in bytes of the data allocation.
} buffer;

#define GAPIL_KIND_BOOL 1
#define GAPIL_KIND_U8 2
#define GAPIL_KIND_S8 3
#define GAPIL_KIND_U16 4
#define GAPIL_KIND_S16 5
#define GAPIL_KIND_F32 6
#define GAPIL_KIND_U32 7
#define GAPIL_KIND_S32 8
#define GAPIL_KIND_F64 9
#define GAPIL_KIND_U64 10
#define GAPIL_KIND_S64 11
#define GAPIL_KIND_INT 12
#define GAPIL_KIND_UINT 13
#define GAPIL_KIND_SIZE 14
#define GAPIL_KIND_CHAR 15
#define GAPIL_KIND_ARRAY 16
#define GAPIL_KIND_CLASS 17
#define GAPIL_KIND_ENUM 18
#define GAPIL_KIND_MAP 19
#define GAPIL_KIND_POINTER 20
#define GAPIL_KIND_REFERENCE 21
#define GAPIL_KIND_SLICE 22
#define GAPIL_KIND_STRING 23

typedef uint8_t GAPIL_BOOL;

#define GAPIL_FALSE 0
#define GAPIL_TRUE 1

#ifndef DECL_GAPIL_CB
#define DECL_GAPIL_CB(RETURN, NAME, ...) RETURN NAME(__VA_ARGS__)
#endif

////////////////////////////////////////////////////////////////////////////////
// Runtime API implemented in runtime.cpp                                     //
////////////////////////////////////////////////////////////////////////////////

// allocates memory using the arena with the given size and alignment.
DECL_GAPIL_CB(void*, gapil_alloc, arena*, uint64_t size, uint64_t align);

// re-allocates memory previously allocated with the arena to a new size and
// alignment.
DECL_GAPIL_CB(void*, gapil_realloc, arena*, void* ptr, uint64_t size,
              uint64_t align);

// frees memory previously allocated with gapil_alloc or gapil_realloc.
DECL_GAPIL_CB(void, gapil_free, arena*, void* ptr);

// allocates a new slice and underlying pool with the given size.
DECL_GAPIL_CB(pool*, gapil_make_pool, context*, uint64_t size);

// frees a pool previously allocated with gapil_make_pool.
DECL_GAPIL_CB(void, gapil_free_pool, pool*);

// allocates a new string with the given data and length.
// length excludes a null-pointer.
DECL_GAPIL_CB(string*, gapil_make_string, arena*, uint64_t length, void* data);

// frees a string allocated with gapil_make_string.
DECL_GAPIL_CB(void, gapil_free_string, string*);

// compares two strings lexicographically, using the same rules as strcmp.
DECL_GAPIL_CB(int32_t, gapil_string_compare, string*, string*);

// logs a message to the current logger.
// fmt is a printf-style message.
DECL_GAPIL_CB(void, gapil_logf, uint8_t severity, uint8_t* file, uint32_t line,
              uint8_t* fmt, ...);

#undef DECL_GAPIL_CB

#ifdef __cplusplus
}  // extern "C"
#endif  // __cplusplus

#endif  // __GAPIL_RUNTIME_H__
