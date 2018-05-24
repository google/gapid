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

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif // __cplusplus

typedef struct arena_t   arena;
typedef struct pool_t    pool;
typedef struct globals_t globals;
typedef struct string_t  string;

#define GAPIL_ERR_SUCCESS 0
#define GAPIL_ERR_ABORTED 1

#define GAPIL_MAP_ELEMENT_EMPTY 0
#define GAPIL_MAP_ELEMENT_FULL  1
#define GAPIL_MAP_ELEMENT_USED  2

#define GAPIL_MAP_GROW_MULTIPLIER 2
#define GAPIL_MIN_MAP_SIZE        16
#define GAPIL_MAP_MAX_CAPACITY    0.8f

// context contains information about the environment in which a function is
// executing.
typedef struct context_t {
	uint32_t    id;           // the context identifier. Can be treated as user-data.
	uint32_t    location;     // the API source location.
	uint32_t*   next_pool_id; // the identifier of the next pool to be created.
	globals*    globals;      // a pointer to the global state.
	arena*      arena;        // the memory arena used for allocations.
} context;

// pool describes the underlying buffer that may be used by one or more slices.
typedef struct pool_t {
	uint32_t ref_count; // number of owners of this pool.
	uint32_t id;        // unique identifier of this pool.
	uint64_t size;      // total size of the pool in bytes.
	arena*   arena;     // arena that owns the allocation of this pool and its buffer.
	void*    buffer;    // nullptr for application pool
} pool;

// slice is the data of a gapil slice type (elty foo[]).
typedef struct slice_t {
	pool*    pool;  // the underlying pool. nullptr represents the application pool.
	void*    root;  // original pointer this slice derives from.
	void*    base;  // address of first element.
	uint64_t size;  // size in bytes of the slice.
	uint64_t count; // total number of elements in the slice.
} slice;

// string is the shared data of a gapil string type.
// A string is a pointer to this struct.
typedef struct string_t {
	uint32_t ref_count; // number of owners of this string.
	arena*   arena;     // arena that owns this string allocation.
	uint64_t length;    // size in bytes of this string (including null-terminator).
	uint8_t  data[1];   // the null-terminated string bytes.
} string;

// map is the shared data of a gapil map type.
// A map is a pointer to this struct.
typedef struct map_t {
	uint32_t ref_count; // number of owners of this map.
	arena*   arena;     // arena that owns this map allocation and its elements buffer.
	uint64_t count;     // number of elements in the map.
	uint64_t capacity;  // size of the elements buffer.
	void*    elements;  // pointer to the elements buffer.
} map;

// ref is the shared data of a gapil ref!T type.
// A ref is a pointer to this struct.
typedef struct ref_t {
	uint32_t ref_count; // number of owners of this ref.
	arena*   arena;     // arena that owns this ref allocation.
	/* T */             // referenced object immediately follows.
} ref;

typedef uint8_t GAPIL_BOOL;

#define GAPIL_FALSE 0
#define GAPIL_TRUE  1

////////////////////////////////////////////////////////////////////////////////
// Functions to be implemented by the user of the runtime                     //
////////////////////////////////////////////////////////////////////////////////

// callback to map the serialized pointer to a pointer in an allocated buffer.
typedef void* gapil_pointer_remapper(context* ctx, uintptr_t pointer, uint64_t length);

// assigns to file and line the current source location within the .api file.
// if there is no current source location then file and line are unassigned.
typedef void gapil_get_code_location(context* ctx, char** file, uint32_t* line);

////////////////////////////////////////////////////////////////////////////////
// Runtime API implemented by the compiler                                    //
////////////////////////////////////////////////////////////////////////////////

void gapil_string_reference(string*);
void gapil_string_release(string*);

void gapil_slice_reference(slice);
void gapil_slice_release(slice);

////////////////////////////////////////////////////////////////////////////////
// Runtime API implemented in runtime.cpp                                     //
////////////////////////////////////////////////////////////////////////////////

// initializes the context.
void gapil_init_context(context* ctx);

// termintates the context.
void gapil_term_context(context* ctx);

// sets the pointer remapper callback used to remap serialized pointers to a
// pointer in an allocated buffer.
void gapil_set_pointer_remapper(gapil_pointer_remapper*);

// sets the callback used to fetch the file and line location for the current
// source location within the .api file.
void gapil_set_code_locator(gapil_get_code_location*);


#ifndef DECL_GAPIL_CB
#define DECL_GAPIL_CB(RETURN, NAME, ...) RETURN NAME(__VA_ARGS__)
#endif

// allocates a buffer using the arena with the given size and alignment.
DECL_GAPIL_CB(void*, gapil_alloc, arena*, uint64_t size, uint64_t align);

// re-allocates a buffer previously allocated with the arena to a new size and
// alignment.
DECL_GAPIL_CB(void*, gapil_realloc, arena*, void* ptr, uint64_t size, uint64_t align);

// frees a buffer previously allocated with gapil_alloc or gapil_realloc.
DECL_GAPIL_CB(void, gapil_free, arena*, void* ptr);

// allocates a new slice and underlying pool with the given size.
DECL_GAPIL_CB(pool*, gapil_make_pool, context* ctx, uint64_t size);

// allocates a new slice and underlying pool filled with data from the given
// base pointer, offset and size.
DECL_GAPIL_CB(void, gapil_pointer_to_slice, context* ctx, uintptr_t ptr, uint64_t offset, uint64_t size, uint64_t count, slice* out);

// frees a pool previously allocated with gapil_make_pool, gapil_string_to_slice
// or gapil_pointer_to_slice.
DECL_GAPIL_CB(void, gapil_free_pool, pool*);

// copies N bytes of data from src to dst, where N is min(dst.size, src.size).
DECL_GAPIL_CB(void, gapil_copy_slice, context* ctx, slice* dst, slice* src);

// allocates a new slice and underlying pool filled with the data of string.
DECL_GAPIL_CB(void, gapil_string_to_slice, context* ctx, string* string, slice* out);

// allocates a new string with the given data and length.
// length excludes a null-pointer.
DECL_GAPIL_CB(string*, gapil_make_string, arena*, uint64_t length, void* data);

// allocates a new string a copy of the null-terminated string at the given
// serialized pointer.
DECL_GAPIL_CB(string*, gapil_pointer_to_string, context* ctx, uintptr_t ptr);

// frees a string allocated with gapil_make_string, gapil_pointer_to_string,
// gapil_string_concat or gapil_slice_to_string.
DECL_GAPIL_CB(void, gapil_free_string, string*);

// allocates a new string filled with the data of slice.
DECL_GAPIL_CB(string*, gapil_slice_to_string, context* ctx, slice* slice);

// allocates a new string containing the concatenated data of the two strings.
DECL_GAPIL_CB(string*, gapil_string_concat, string*, string*);

// compares two strings lexicographically, using the same rules as strcmp.
DECL_GAPIL_CB(int32_t, gapil_string_compare, string*, string*);

// applys the read observations tagged to the current command into the memory
// model.
DECL_GAPIL_CB(void, gapil_apply_reads, context* ctx);

// applys the write observations tagged to the current command into the memory
// model.
DECL_GAPIL_CB(void, gapil_apply_writes, context* ctx);

// calls an extern function with the given name and pointer to arguments.
// If the extern returns a value, this is placed in res.
DECL_GAPIL_CB(void, gapil_call_extern, context* ctx, string* name, void* args, void* res);

// logs a message to the current logger.
// fmt is a printf-style message.
DECL_GAPIL_CB(void, gapil_logf, context* ctx, uint8_t severity, uint8_t* fmt, ...);

#undef DECL_GAPIL_CB

#ifdef __cplusplus
} // extern "C"
#endif // __cplusplus

#endif  // __GAPIL_RUNTIME_H__