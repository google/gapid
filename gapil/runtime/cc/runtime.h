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
typedef struct globals_t globals;
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
  uint32_t id;         // the context identifier. Can be treated as user-data.
  globals* globals;    // a pointer to the global state.
  arena* arena;        // the memory arena used for allocations.
  uint64_t thread;     // the identifier of the currently executing thread.
  void* cmd_args;      // the arguments to the currently executing command.
  uint64_t cmd_id;     // the current command identifier.
  uint64_t cmd_idx;    // the index of the current command being executed.
  uint64_t cmd_flags;  // extra info for the current command being executed.
  uint32_t* next_pool_id;  // the identifier of the next pool to be created.
  // additional data used by compiler plugins goes here.
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

typedef struct gapil_rtti {
  uint32_t kind;             // kind of the type.
  uint32_t api_index;        // api index to which the type belongs.
  uint32_t type_index;       // index of the type within the api.
  gapil_char* type_name;     // name of the type.
  void (*reference)(void*);  // increment the reference count for the type.
  void (*release)(void*);    // decrement the reference count for the type.
} gapil_rtti;

typedef struct gapil_any_t {
  uint32_t ref_count;  // number of owners of this any.
  arena* arena;        // arena that owns this any allocation.
  gapil_rtti* rtti;    // type information of the value.
  void* value;  // pointer to the value. For boxed value-types, this should be
                // allocated as part of the any (it should not be freed
                // separately).
} gapil_any;

typedef struct gapil_msg_arg_t {
  gapil_char* name;  // argument identifier.
  gapil_any* value;  // argument value.
} gapil_msg_arg;

typedef struct gapil_msg_t {
  // number of owners of this message.
  uint32_t ref_count;

  // arena that owns this message allocation.
  arena* arena;

  // message identifier string.
  gapil_char* identifier;

  // array of key-value pairs. Terminated with a null-null.
  gapil_msg_arg* args;
} gapil_msg;

// gapil_api_module holds the functions produced by a compilation for a single
// API.
typedef struct gapil_api_module_t {
  // offset in bytes of the API's globals from context.globals.
  uint64_t globals_offset;

  // size in bytes of the API's globals.
  uint64_t globals_size;

  // number of functions in this module.
  uint64_t num_cmds;

  // array of functions generated for all the commands.
  uint32_t (**cmds)(void* ctx);

} gapil_api_module;

// gapil_symbol is a pair of name and address.
typedef struct gapil_symbol_t {
  const char* name;
  const void* addr;
} gapil_symbol;

// gapil_module holds the functions produced by a compilation.
typedef struct gapil_module_t {
  // creates an initializes a new context with the given arena.
  context* (*create_context)(arena* arena);

  // destroys the context created by create_context.
  void (*destroy_context)(context*);

  // Size in bytes of all globals.
  uint64_t globals_size;

  // number of APIs in this module.
  uint32_t num_apis;

  // array of all the APIs.
  gapil_api_module* apis;

  // number of symbols in this module.
  uint32_t num_symbols;

  // array of all the symbols.
  gapil_symbol* symbols;

} gapil_module;

typedef uint8_t GAPIL_BOOL;

#define GAPIL_FALSE 0
#define GAPIL_TRUE 1

#ifndef DECL_GAPIL_CB
#define DECL_GAPIL_CB(RETURN, NAME, ...) RETURN NAME(__VA_ARGS__)
#endif

////////////////////////////////////////////////////////////////////////////////
// Functions to be implemented by the user of the runtime                     //
////////////////////////////////////////////////////////////////////////////////

typedef enum gapil_data_access_t {
  GAPIL_READ = 0x1,
  GAPIL_WRITE = 0x2,
} gapil_data_access;

typedef struct gapil_runtime_callbacks_t {
  // applys the read observations tagged to the current command into the memory
  // model.
  void (*apply_reads)(context*);

  // applys the write observations tagged to the current command into the memory
  // model.
  void (*apply_writes)(context*);

  // Returns a pointer to the pool's data starting at pointer for size bytes.
  void* (*resolve_pool_data)(context*, pool*, uint64_t ptr, gapil_data_access,
                             uint64_t* size);

  // call_extern calls the extern with the given name and arguments. If the
  // extern has a return type, then the result should be be stored to res.
  void (*call_extern)(context*, uint8_t* name, void* args, void* res);

  // stores the buffer at ptr of the given size into the database.
  // Writes the 20-byte database identifier of the stored data to id.
  void (*store_in_database)(context* ctx, void* ptr, uint64_t size,
                            uint8_t* id_out);

} gapil_runtime_callbacks;

void gapil_set_runtime_callbacks(gapil_runtime_callbacks*);

////////////////////////////////////////////////////////////////////////////////
// Runtime API implemented in runtime.cpp                                     //
////////////////////////////////////////////////////////////////////////////////

DECL_GAPIL_CB(void, gapil_any_reference, gapil_any*);
DECL_GAPIL_CB(void, gapil_any_release, gapil_any*);

DECL_GAPIL_CB(void, gapil_msg_reference, gapil_msg*);
DECL_GAPIL_CB(void, gapil_msg_release, gapil_msg*);

// allocates memory using the arena with the given size and alignment.
DECL_GAPIL_CB(void*, gapil_alloc, arena*, uint64_t size, uint64_t align);

// re-allocates memory previously allocated with the arena to a new size and
// alignment.
DECL_GAPIL_CB(void*, gapil_realloc, arena*, void* ptr, uint64_t size,
              uint64_t align);

// frees memory previously allocated with gapil_alloc or gapil_realloc.
DECL_GAPIL_CB(void, gapil_free, arena*, void* ptr);

// creates a buffer with the given alignment and capacity.
DECL_GAPIL_CB(void, gapil_create_buffer, arena*, uint64_t capacity,
              uint64_t alignment, buffer*);

// destroys a buffer previously created with gapil_create_buffer.
DECL_GAPIL_CB(void, gapil_destroy_buffer, buffer*);

// appends data to a buffer.
DECL_GAPIL_CB(void, gapil_append_buffer, buffer*, const void* data,
              uint64_t size);

// allocates a new slice and underlying pool with the given size.
DECL_GAPIL_CB(pool*, gapil_make_pool, context*, uint64_t size);

// frees a pool previously allocated with gapil_make_pool or
// gapil_string_to_slice.
DECL_GAPIL_CB(void, gapil_free_pool, pool*);

// returns a pointer to the underlying buffer data for the given slice,
// using gapil_data_resolver if it has been set.
DECL_GAPIL_CB(void*, gapil_slice_data, context*, slice*, gapil_data_access);

// copies N bytes of data from src to dst, where N is min(dst.size, src.size).
DECL_GAPIL_CB(void, gapil_copy_slice, context*, slice* dst, slice* src);

// allocates a new slice and underlying pool filled with the data of string.
DECL_GAPIL_CB(void, gapil_string_to_slice, context*, string* string,
              slice* out);

// allocates a new string with the given data and length.
// length excludes a null-pointer.
DECL_GAPIL_CB(string*, gapil_make_string, arena*, uint64_t length, void* data);

// outputs a slice spanning the bytes of the null-terminated string starting at
// ptr. The slice includes the null-terminator byte.
DECL_GAPIL_CB(void, gapil_cstring_to_slice, context*, uintptr_t ptr,
              slice* out);

// frees a string allocated with gapil_make_string, gapil_string_concat or
// gapil_slice_to_string.
DECL_GAPIL_CB(void, gapil_free_string, string*);

// allocates a new string filled with the data of slice.
DECL_GAPIL_CB(string*, gapil_slice_to_string, context*, slice* slice);

// allocates a new string containing the concatenated data of the two strings.
DECL_GAPIL_CB(string*, gapil_string_concat, string*, string*);

// compares two strings lexicographically, using the same rules as strcmp.
DECL_GAPIL_CB(int32_t, gapil_string_compare, string*, string*);

// logs a message to the current logger.
// fmt is a printf-style message.
DECL_GAPIL_CB(void, gapil_logf, uint8_t severity, uint8_t* file, uint32_t line,
              uint8_t* fmt, ...);

// applys the read observations tagged to the current command into the memory
// model.
DECL_GAPIL_CB(void, gapil_apply_reads, context*);

// applys the write observations tagged to the current command into the memory
// model.
DECL_GAPIL_CB(void, gapil_apply_writes, context*);

// Returns a pointer to access a pool's data.
// If len is non-nil, it will be assigned the maximum number of bytes from the
// returned pointer that can be accessed.
DECL_GAPIL_CB(void*, gapil_resolve_pool_data, context*, pool*, uint64_t ptr,
              gapil_data_access, uint64_t* size);

// stores the buffer at ptr of the given size into the database.
// Writes the 20-byte database identifier of the stored data to id.
DECL_GAPIL_CB(void, gapil_store_in_database, context* ctx, void* ptr,
              uint64_t size, uint8_t* id_out);

// gapil_call_extern calls the extern with the given name and arguments. If the
// extern has a return type, then the result should be be stored to res.
DECL_GAPIL_CB(void, gapil_call_extern, context*, uint8_t* name, void* args,
              void* res);

#undef DECL_GAPIL_CB

#ifdef __cplusplus
}  // extern "C"
#endif  // __cplusplus

#endif  // __GAPIL_RUNTIME_H__
