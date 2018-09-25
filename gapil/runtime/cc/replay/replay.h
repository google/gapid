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

#ifndef __GAPIL_RUNTIME_REPLAY_H__
#define __GAPIL_RUNTIME_REPLAY_H__

#include "gapil/runtime/cc/runtime.h"

#ifdef __cplusplus
extern "C" {
#endif  // __cplusplus

typedef struct gapil_replay_resource_info {
  uint8_t id[20];
  uint32_t size;
} gapil_replay_resource_info_t;

typedef struct gapil_replay_data_t {
  // instructions currently being built, or opcodes post build.
  buffer stream;

  // buffer of gapil_replay_resource_info representing all the resources used by
  // the replay.
  buffer resources;

  // buffer of constant data used by the replay.
  buffer constants;

  // function used to emit the call of the current command
  void (*call)(context*);

  // additional data referenced by replay.cpp.
  void* data_ex;

  // Alignment of a pointer for the replay device.
  // TODO: Remove. This is only here to match old replay implementation.
  uint32_t pointer_alignment;
} gapil_replay_data;

////////////////////////////////////////////////////////////////////////////////
// Runtime API implemented in replay.cpp                                      //
////////////////////////////////////////////////////////////////////////////////

// TODO
void gapil_replay_build(context* ctx, gapil_replay_data* data);

// gapil_replay_remap_func is a function that can be used to return a remapping
// key for the given remapped value at ptr.
typedef uint64_t gapil_replay_remap_func(context* ctx, void* ptr);

// gapil_replay_register_remap_func registers the given remapping function for
// the API type.
void gapil_replay_register_remap_func(const char* api, const char* type,
                                      gapil_replay_remap_func* func);

#ifndef DECL_GAPIL_REPLAY_FUNC
#define DECL_GAPIL_REPLAY_FUNC(RETURN, NAME, ...) RETURN NAME(__VA_ARGS__)
#endif

// gapil_replay_init_data initializes the gapil_replay_data structure.
// Note there are fields that the compiler will initialize itself.
DECL_GAPIL_REPLAY_FUNC(void, gapil_replay_init_data, context* ctx,
                       gapil_replay_data* data);

// gapil_replay_term_data frees fields of the gapil_replay_data structure
// that were initialized by gapil_replay_init_data.
DECL_GAPIL_REPLAY_FUNC(void, gapil_replay_term_data, context* ctx,
                       gapil_replay_data* data);

// gapil_replay_allocate_memory allocates size bytes from the volatile memory
// address space with the guaranteed minimum alignment.
// This memory remains allocated for the entire duration of the replay.
DECL_GAPIL_REPLAY_FUNC(uint64_t, gapil_replay_allocate_memory, context* ctx,
                       gapil_replay_data* data, uint64_t size,
                       uint64_t alignment);

// gapil_replay_reserve_memory reserves the given capture memory range for
// replay. start is the address of the first byte in the memory range to
// reserve.
// size is the number of bytes to reserve.
// ns is the address namespace.
// min_alignment is the minimum expected alignment in bytes for this block when
// recreated for replay.
DECL_GAPIL_REPLAY_FUNC(void, gapil_replay_reserve_memory, context* ctx,
                       gapil_replay_data* data, slice* sli, uint32_t ns,
                       uint32_t min_alignment);

// gapil_replay_add_resource is called whenever a memory range needs to be
// encoded as a resource. The resource identifier is returned.
DECL_GAPIL_REPLAY_FUNC(uint32_t, gapil_replay_add_resource, context* ctx,
                       gapil_replay_data* data, slice* slice);

// gapil_replay_add_constant adds data to the constants buffer, returning the
// address of the constant in the constant address space.
// Constants are deduplicated.
DECL_GAPIL_REPLAY_FUNC(uint32_t, gapil_replay_add_constant, context* ctx,
                       gapil_replay_data* data, void* buf, uint32_t size,
                       uint32_t alignment);

// gapil_replay_get_remap_func is called to lookup the remapping function for a
// given API type.
DECL_GAPIL_REPLAY_FUNC(gapil_replay_remap_func*, gapil_replay_get_remap_func,
                       char* api, char* type);

// gapil_replay_add_remapping is called to register a remapped value address by
// key.
DECL_GAPIL_REPLAY_FUNC(void, gapil_replay_add_remapping, context* ctx,
                       gapil_replay_data* data, uint64_t addr, uint64_t key);

// gapil_replay_lookup_remapping is called to lookup a remapped value address
// that was previously registered with gapil_replay_add_remapping. Returns the
// volatile address if the key is found, otherwise ~0.
DECL_GAPIL_REPLAY_FUNC(uint64_t, gapil_replay_lookup_remapping, context* ctx,
                       gapil_replay_data* data, uint64_t key);

#undef DECL_GAPIL_REPLAY_FUNC

#ifdef __cplusplus
}  // extern "C"
#endif  // __cplusplus

#endif  // __GAPIL_RUNTIME_REPLAY_H__