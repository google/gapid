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

#include "replay.h"
#include "dataex.h"

#include "core/cc/log.h"
#include "core/cc/range.h"
#include "core/memory/arena/cc/arena.h"

#include "gapir/replay_service/vm.h"

#define __STDC_FORMAT_MACROS
#include <inttypes.h>

#if 0
#define DEBUG_PRINT(...) GAPID_WARNING(__VA_ARGS__)
#else
#define DEBUG_PRINT(...)
#endif

#define SLICE_FMT \
  "[pool: %p, root: 0x%" PRIx64 ", base: 0x%" PRIx64 ", size: 0x%" PRIx64 "]"
#define SLICE_ARGS(sli) sli->pool, sli->root, sli->base, sli->size

using namespace gapir::vm;
using namespace gapil::runtime::replay;

// Ensure that replay.cpp is not stripped from the final executable.
// Typically the symbols defined by this compilation unit are only dynamically
// referenced by the JIT'd code, so the linker may decide to strip it from the
// final executable. This is a placeholder symbol referenced by builder.cpp
// (which is directly used by the GAPIS executable).
int GAPIL_REPLAY_FORCE_LINK = 0;

namespace {

std::unordered_map<std::string, gapil_replay_remap_func*> remap_funcs;

}  // anonymous namespace

extern "C" {

void gapil_replay_init_data(context* ctx, gapil_replay_data* data) {
  auto arena = reinterpret_cast<core::Arena*>(ctx->arena);
  data->data_ex = arena->create<DataEx>();
  data->resources = buffer{.arena = ctx->arena};
  data->constants = buffer{.arena = ctx->arena};
}

void gapil_replay_term_data(context* ctx, gapil_replay_data* data) {
  auto arena = reinterpret_cast<core::Arena*>(ctx->arena);
  arena->destroy(reinterpret_cast<DataEx*>(data->data_ex));
  gapil_destroy_buffer(&data->resources);
  gapil_destroy_buffer(&data->constants);
}

uint64_t gapil_replay_allocate_memory(context* ctx, gapil_replay_data* data,
                                      uint64_t size, uint64_t alignment) {
  auto ex = reinterpret_cast<DataEx*>(data->data_ex);
  auto res = ex->allocated.alloc(size, alignment);
  DEBUG_PRINT("gapil_replay_allocate_memory(size: 0x%" PRIx64
              ", alignment: 0x%" PRIx64 ") -> 0x%" PRIx64,
              size, alignment, res);
  return res;
}

void gapil_replay_reserve_memory(context* ctx, gapil_replay_data* data,
                                 slice* sli, uint32_t ns, uint32_t alignment) {
  DEBUG_PRINT("gapil_replay_reserve_memory(sli:" SLICE_FMT
              ", ns: %d, alignment: %d)",
              SLICE_ARGS(sli), ns, alignment);
  auto ex = reinterpret_cast<DataEx*>(data->data_ex);
  auto start = sli->root;
  auto end = sli->base + sli->size;
  auto& reserved = ex->reserved[ns];
  for (auto block : reserved.intersect(start, end)) {
    alignment = std::max(alignment, block.mAlignment);
  }
  reserved.merge(MemoryRange(start, end, alignment));
}

uint32_t gapil_replay_add_resource(context* ctx, gapil_replay_data* data,
                                   slice* sli) {
  DEBUG_PRINT("gapil_replay_add_resource(" SLICE_FMT ")", SLICE_ARGS(sli));
  auto ex = reinterpret_cast<DataEx*>(data->data_ex);

  auto ptr = gapil_slice_data(ctx, sli, GAPIL_READ);
  core::Id id;
  gapil_store_in_database(ctx, ptr, sli->size, id.data);

  auto it = ex->resources.find(id);
  if (it != ex->resources.end()) {
    return it->second.index;
  }

  DataEx::ResourceInfo info;
  info.index = ex->resources.size();
  info.size = sli->size;
  ex->resources[id] = info;
  return info.index;
}

uint32_t gapil_replay_add_constant(context* ctx, gapil_replay_data* data,
                                   void* buf, uint32_t size,
                                   uint32_t alignment) {
  DEBUG_PRINT("gapil_replay_add_constant(buf: %p, size: %" PRIu32 ")", buf,
              size);
  auto ex = reinterpret_cast<DataEx*>(data->data_ex);

  // TODO: Remove. This is only here to match old implementation.
  alignment = std::max(alignment, data->pointer_alignment);

  // try to find an existing constant in the constants buffer.
  auto id = core::Id::Hash(buf, size);
  auto it = ex->constant_offsets.find(id);
  if (it != ex->constant_offsets.end()) {
    return it->second;
  }

  auto& consts = data->constants;

  // grow the constants buffer to fit the data with the specified alignment.
  auto offset = align<size_t>(consts.size, alignment);
  auto new_size = offset + size;
  if (new_size > consts.capacity) {
    auto arena = reinterpret_cast<core::Arena*>(consts.arena);
    consts.capacity = std::max<uint32_t>(new_size, consts.capacity * 2);
    consts.data = (uint8_t*)arena->reallocate(consts.data, consts.capacity,
                                              consts.alignment);
  }
  // clear the alignment padding to 0
  memset(consts.data + consts.size, 0, offset - consts.size);
  consts.size = new_size;

  // append the data.
  memcpy(consts.data + offset, buf, size);

  // add the id to the constants_offsets map so that this data can be shared.
  ex->constant_offsets[id] = offset;

  return offset;
}

gapil_replay_remap_func* gapil_replay_get_remap_func(char* api, char* type) {
  DEBUG_PRINT("gapil_replay_get_remap_func('%s')", type);
  auto name = std::string(api) + "." + type;
  auto it = remap_funcs.find(name);
  if (it == remap_funcs.end()) {
    GAPID_FATAL("No replay remapping function registered for type '%s'", type);
  }
  return it->second;
}

void gapil_replay_register_remap_func(const char* api, const char* type,
                                      gapil_replay_remap_func* func) {
  DEBUG_PRINT("gapil_replay_register_remap_func('%s', %p)", type, func);
  auto name = std::string(api) + "." + type;
  remap_funcs[name] = func;
}

void gapil_replay_add_remapping(context* ctx, gapil_replay_data* data,
                                uint64_t addr, uint64_t key) {
  DEBUG_PRINT("gapil_replay_add_remapping(addr: 0x%" PRIx64 ", key: 0x%" PRIx64
              ")",
              addr, key);
  auto ex = reinterpret_cast<DataEx*>(data->data_ex);
  ex->remappings[key] = addr;
}

uint64_t gapil_replay_lookup_remapping(context* ctx, gapil_replay_data* data,
                                       uint64_t key) {
  auto ex = reinterpret_cast<DataEx*>(data->data_ex);
  auto it = ex->remappings.find(key);
  if (it == ex->remappings.end()) {
    DEBUG_PRINT(
        "gapil_replay_lookup_remapping(key: 0x%" PRIx64 ") -> not found", key);
    return ~(uint64_t)0;
  }
  DEBUG_PRINT("gapil_replay_lookup_remapping(key: 0x%" PRIx64 ") -> %" PRIx64,
              key, it->second);
  return it->second;
}

}  // extern "C"
