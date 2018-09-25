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

#include "asm.h"
#include "dataex.h"
#include "replay.h"

#include "core/cc/log.h"
#include "core/memory/arena/cc/arena.h"
#include "gapil/runtime/cc/buffer.inc"
#include "gapir/replay_service/vm.h"

#define __STDC_FORMAT_MACROS
#include <inttypes.h>

#define ENABLE_DEBUG 0
#define ENABLE_DEBUG_INST 0

#if ENABLE_DEBUG
#define DEBUG_PRINT(...) GAPID_WARNING(__VA_ARGS__)
#else
#define DEBUG_PRINT(...)
#endif

#if ENABLE_DEBUG_INST
#define DEBUG_PRINT_INST(...) GAPID_WARNING(__VA_ARGS__)
#else
#define DEBUG_PRINT_INST(...)
#endif

#define ASM_VAL_FMT "[0x%" PRIx64 ", %s]"
#define ASM_VAL_ARGS(val) val.data, asm_type_str(val.data_type)

using namespace gapir::vm;
using namespace gapil::runtime::replay;

// Ensure that replay.cpp is not stripped from final executable.
// See GAPIL_REPLAY_FORCE_LINK in replay.cpp for more info.
extern int GAPIL_REPLAY_FORCE_LINK;
void dont_strip_gapil_replay_cpp() { GAPIL_REPLAY_FORCE_LINK = 1; }

namespace {

// Absolute pointer value used when an unrecognised pointer is encountered, that
// cannot be remapped to a sensible location. In these situations we pass a
// pointer that should cause an access violation if it is dereferenced.
// We opt to not use 0x00 as this is often overloaded to mean something else.
// Must match value used in cc/gapir/memory_manager.h
const uint64_t unobservedPointer = 0xBADF00D;

// These functions are only used in debug
#if ENABLE_DEBUG_INST
const char* bool_str(bool b) { return b ? "true" : "false"; }

const char* asm_type_str(gapil_replay_asm_type ty) {
  switch (ty) {
    case GAPIL_REPLAY_ASM_TYPE_BOOL:
      return "BOOL";
    case GAPIL_REPLAY_ASM_TYPE_INT8:
      return "INT8";
    case GAPIL_REPLAY_ASM_TYPE_INT16:
      return "INT16";
    case GAPIL_REPLAY_ASM_TYPE_INT32:
      return "INT32";
    case GAPIL_REPLAY_ASM_TYPE_INT64:
      return "INT64";
    case GAPIL_REPLAY_ASM_TYPE_UINT8:
      return "UINT8";
    case GAPIL_REPLAY_ASM_TYPE_UINT16:
      return "UINT16";
    case GAPIL_REPLAY_ASM_TYPE_UINT32:
      return "UINT32";
    case GAPIL_REPLAY_ASM_TYPE_UINT64:
      return "UINT64";
    case GAPIL_REPLAY_ASM_TYPE_FLOAT:
      return "FLOAT";
    case GAPIL_REPLAY_ASM_TYPE_DOUBLE:
      return "DOUBLE";
    case GAPIL_REPLAY_ASM_TYPE_ABSOLUTE_POINTER:
      return "ABSOLUTE_POINTER";
    case GAPIL_REPLAY_ASM_TYPE_CONSTANT_POINTER:
      return "CONSTANT_POINTER";
    case GAPIL_REPLAY_ASM_TYPE_VOLATILE_POINTER:
      return "VOLATILE_POINTER";
    case GAPIL_REPLAY_ASM_TYPE_OBSERVED_POINTER_NAMESPACE_0:
      return "OBSERVED_POINTER_NAMESPACE_0";
  }
  switch ((unsigned int)ty) {
    case GAPIL_REPLAY_ASM_TYPE_OBSERVED_POINTER_NAMESPACE_0 + 1:
      return "OBSERVED_POINTER_NAMESPACE_1";
    case GAPIL_REPLAY_ASM_TYPE_OBSERVED_POINTER_NAMESPACE_0 + 2:
      return "OBSERVED_POINTER_NAMESPACE_2";
    case GAPIL_REPLAY_ASM_TYPE_OBSERVED_POINTER_NAMESPACE_0 + 3:
      return "OBSERVED_POINTER_NAMESPACE_3";
  }
  return "<unknown>";
}
#endif  // #if ENABLE_DEBUG_INST

template <typename T>
const core::Range<T> to_range(const buffer& buf) {
  return core::Range<T>(reinterpret_cast<T*>(buf.data), buf.size / sizeof(T));
}

uint32_t set_bit(uint32_t bits, uint32_t idx, bool v) {
  return v ? bits | (1 << idx) : bits & ~(1 << idx);
}

class Builder {
 public:
  Builder(::arena* a, gapil_replay_data* d);

  void layout_volatile_memory();
  void generate_opcodes();
  void build_resources();

 private:
  // Various bit-masks used by this class.
  // Many opcodes can fit values into the opcode itself.
  // These masks are used to determine which values fit.
  static const uint64_t mask19 = 0x7ffff;
  static const uint64_t mask20 = 0xfffff;
  static const uint64_t mask26 = 0x3ffffff;
  static const uint64_t mask45 = 0x1fffffffffff;
  static const uint64_t mask46 = 0x3fffffffffff;
  static const uint64_t mask52 = 0xfffffffffffff;

  gapil_replay_asm_value remap(gapil_replay_asm_value v);

  void push(gapil_replay_asm_value val);
  void load(gapil_replay_asm_value val, gapil_replay_asm_type ty);
  void store(gapil_replay_asm_value dst);

  inline void pushi(uint32_t ty, uint32_t v) {
    opcodes_.append(packCYZ(Opcode::PUSH_I, ty, v));
  }

  inline void extend(uint32_t v) { CX(Opcode::EXTEND, v); }

  inline void C(Opcode c) { opcodes_.append(packC(c)); }
  inline void CX(Opcode c, uint32_t x) { opcodes_.append(packCX(c, x)); }
  inline void CYZ(Opcode c, uint32_t y, uint32_t z) {
    opcodes_.append(packCYZ(c, y, z));
  }

  // clang-format off
	//     ▏60       ▏50       ▏40       ▏30       ▏20       ▏10
	// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●● mask19
	// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●● mask20
	// ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●●●●●●●● mask26
	// ○○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●● mask45
	// ○○○○○○○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●● mask46
	// ○○○○○○○○○○○○●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●● mask52
	//                                            ▕      PUSHI 20     ▕
	//                                      ▕         EXTEND 26       ▕
  // clang-format on

  // clang-format off
// ┏━━┯━━┯━━┯━━┯━━┯━━┳━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┓
// ┃c │c │c │c │c │c ┃0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 │0 ┃
// ┃ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃₂₅│₂₄│₂₃│₂₂│₂₁│₂₀│₁₉│₁₈│₁₇│₁₆│₁₅│₁₄│₁₃│₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆│ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃
// ┡━━┿━━┿━━┿━━┿━━┿━━╇━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┩
// │₃₁│₃₀│₂₉│₂₈│₂₇│₂₆│₂₅│₂₄│₂₃│₂₂│₂₁│₂₀│₁₉│₁₈│₁₇│₁₆│₁₅│₁₄│₁₃│₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆│ ₅│ ₄│ ₃│ ₂│ ₁│ ₀│
// └──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
  // clang-format on
  inline uint32_t packC(Opcode c) {
    if (uint32_t(c) >= 0x3f) {
      GAPID_FATAL("c exceeds 6 bits (0x%x)", uint32_t(c));
    }
    return uint32_t(c) << 26;
  }

  // clang-format off
// ┏━━┯━━┯━━┯━━┯━━┯━━┳━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┓
// ┃c │c │c │c │c │c ┃x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x │x ┃
// ┃ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃₂₅│₂₄│₂₃│₂₂│₂₁│₂₀│₁₉│₁₈│₁₇│₁₆│₁₅│₁₄│₁₃│₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆│ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃
// ┡━━┿━━┿━━┿━━┿━━┿━━╇━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┩
// │₃₁│₃₀│₂₉│₂₈│₂₇│₂₆│₂₅│₂₄│₂₃│₂₂│₂₁│₂₀│₁₉│₁₈│₁₇│₁₆│₁₅│₁₄│₁₃│₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆│ ₅│ ₄│ ₃│ ₂│ ₁│ ₀│
// └──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
  // clang-format on
  inline uint32_t packCX(Opcode c, uint32_t x) {
    if (x > 0x3ffffff) {
      GAPID_FATAL("x exceeds 26 bits (0x%x)", x);
    }
    return packC(c) | x;
  }

  // clang-format off
// ┏━━┯━━┯━━┯━━┯━━┯━━┳━━┯━━┯━━┯━━┯━━┯━━┳━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┯━━┓
// ┃c │c │c │c │c │c ┃y │y │y │y │y │y ┃z │z │z │z │z │z │z │z │z │z │z │z │z │z │z │z │z │z │z │z ┃
// ┃ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃₁₉│₁₈│₁₇│₁₆│₁₅│₁₄│₁₃│₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆│ ₅│ ₄│ ₃│ ₂│ ₁│ ₀┃
// ┡━━┿━━┿━━┿━━┿━━┿━━╇━━┿━━┿━━┿━━┿━━┿━━╇━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┿━━┩
// │₃₁│₃₀│₂₉│₂₈│₂₇│₂₆│₂₅│₂₄│₂₃│₂₂│₂₁│₂₀│₁₉│₁₈│₁₇│₁₆│₁₅│₁₄│₁₃│₁₂│₁₁│₁₀│ ₉│ ₈│ ₇│ ₆│ ₅│ ₄│ ₃│ ₂│ ₁│ ₀│
// └──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
  // clang-format on
  inline uint32_t packCYZ(Opcode c, uint32_t y, uint32_t z) {
    if (y > 0x3f) {
      GAPID_FATAL("y exceeds 6 bits (0x%x)", y);
    }
    if (z > 0xfffff) {
      GAPID_FATAL("z exceeds 20 bits (0x%x)", z);
    }
    return packC(c) | (y << 20) | z;
  }

  arena* arena_;
  gapil_replay_data* data_;
  gapil::Buffer opcodes_;
  std::unordered_map<DataEx::Namespace, std::vector<DataEx::VolatileAddr>>
      reserved_base_offsets_;
};

Builder::Builder(arena* arena, gapil_replay_data* data)
    : arena_(arena), data_(data), opcodes_(arena) {}

void Builder::layout_volatile_memory() {
  DEBUG_PRINT("Builder::layout_volatile_memory()");

  auto ex = reinterpret_cast<DataEx*>(data_->data_ex);

  StackAllocator<uint64_t> volatile_mem;

  // Allocate the memory allocated by gapil_replay_allocate_memory.
  volatile_mem.alloc(ex->allocated.size(), ex->allocated.alignment());

  // Allocate all the blocks reserved by gapil_replay_reserve_memory.
  std::vector<DataEx::Namespace> namespaces;
  namespaces.reserve(ex->reserved.size());
  for (auto it : ex->reserved) {
    namespaces.push_back(it.first);
  }
  std::sort(namespaces.begin(), namespaces.end());

  for (auto ns : namespaces) {
    const auto& reserved = ex->reserved[ns];
    auto& offsets = reserved_base_offsets_[ns];
    offsets.reserve(reserved.count());
    for (auto block : reserved) {
      auto size = block.mEnd - block.mStart;
      auto alignment = block.mAlignment;

      // TODO: Remove. This is only here to match old implementation.
      alignment = data_->pointer_alignment;

      auto addr = volatile_mem.alloc(size, alignment);
      DEBUG_PRINT("%.3d: Block NS %d [0x%x - 0x%x] (align %d): 0x%x",
                  offsets.size(), ns, block.mStart, block.mEnd - 1, alignment,
                  addr);
      offsets.push_back(addr);
    }
  }
}

void Builder::generate_opcodes() {
  DEBUG_PRINT("Builder::generate_opcodes()");

  gapil::Buffer::Reader reader(&data_->stream);

  while (true) {
    uint8_t ty;
    if (!reader.read(&ty)) {
      break;
    }

    switch (gapil_replay_asm_inst(ty)) {
      case GAPIL_REPLAY_ASM_INST_CALL: {
        gapil_replay_asm_call inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST(
              "GAPIL_REPLAY_ASM_INST_CALL(push_return: %s, api_index: %" PRId8
              ", function_id: %" PRId16 ")",
              bool_str(inst.push_return), inst.api_index, inst.function_id);
          auto packed = (uint32_t(inst.api_index & 0xf) << 16) |
                        uint32_t(inst.function_id);
          packed = set_bit(packed, 24, inst.push_return);
          CX(Opcode::CALL, packed);
        }
        break;
      }
      case GAPIL_REPLAY_ASM_INST_PUSH: {
        gapil_replay_asm_push inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST("GAPIL_REPLAY_ASM_INST_PUSH(value: " ASM_VAL_FMT ")",
                           ASM_VAL_ARGS(inst.value));
          push(remap(inst.value));
        }
        break;
      }
      case GAPIL_REPLAY_ASM_INST_POP: {
        gapil_replay_asm_pop inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST("GAPIL_REPLAY_ASM_INST_POP(count: %" PRIu32 ")",
                           inst.count);
          CX(Opcode::POP, inst.count);
        }
        break;
      }
      case GAPIL_REPLAY_ASM_INST_COPY: {
        gapil_replay_asm_copy inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST("GAPIL_REPLAY_ASM_INST_COPY(count: %" PRIu32 ")",
                           inst.count);
          CX(Opcode::COPY, inst.count);
        }
        break;
      }
      case GAPIL_REPLAY_ASM_INST_CLONE: {
        gapil_replay_asm_clone inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST("GAPIL_REPLAY_ASM_INST_CLONE(n: %" PRIu32 ")",
                           inst.n);
          CX(Opcode::CLONE, inst.n);
        }
        break;
      }
      case GAPIL_REPLAY_ASM_INST_LOAD: {
        gapil_replay_asm_load inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST(
              "GAPIL_REPLAY_ASM_INST_LOAD(type: %s, src: " ASM_VAL_FMT ")",
              asm_type_str(inst.data_type), ASM_VAL_ARGS(inst.source));
          load(remap(inst.source), inst.data_type);
        }
        break;
      }
      case GAPIL_REPLAY_ASM_INST_STORE: {
        gapil_replay_asm_store inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST("GAPIL_REPLAY_ASM_INST_STORE(dst: " ASM_VAL_FMT ")",
                           ASM_VAL_ARGS(inst.dst));
          store(remap(inst.dst));
        }
        break;
      }
      case GAPIL_REPLAY_ASM_INST_STRCPY: {
        gapil_replay_asm_strcpy inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST("GAPIL_REPLAY_ASM_INST_STRCPY(max_count: %" PRIu64
                           ")",
                           inst.max_count);
          CX(Opcode::STRCPY, inst.max_count);
        }
        break;
      }
      case GAPIL_REPLAY_ASM_INST_RESOURCE: {
        gapil_replay_asm_resource inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST("GAPIL_REPLAY_ASM_INST_RESOURCE(index: %" PRIu32
                           ", dst: " ASM_VAL_FMT ")",
                           inst.index, ASM_VAL_ARGS(inst.dest));
          push(remap(inst.dest));
          CX(Opcode::RESOURCE, inst.index);
        }
        break;
      }
      case GAPIL_REPLAY_ASM_INST_POST: {
        gapil_replay_asm_post inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST("GAPIL_REPLAY_ASM_INST_POST(src: " ASM_VAL_FMT
                           ", size: 0x%" PRIx64 ")",
                           ASM_VAL_ARGS(inst.source), inst.size);
          push(remap(inst.source));
          C(Opcode::POST);
        }
        break;
      }
      case GAPIL_REPLAY_ASM_INST_ADD: {
        gapil_replay_asm_add inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST("GAPIL_REPLAY_ASM_INST_ADD(count: %" PRIu32 ")",
                           inst.count);
          CX(Opcode::RESOURCE, inst.count);
        }
        break;
      }
      case GAPIL_REPLAY_ASM_INST_LABEL: {
        gapil_replay_asm_label inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST("GAPIL_REPLAY_ASM_INST_LABEL(count: %" PRIu32 ")",
                           inst.value);
          CX(Opcode::LABEL, inst.value);
        }
        break;
      }
      case GAPIL_REPLAY_ASM_INST_SWITCHTHREAD: {
        gapil_replay_asm_switchthread inst;
        if (reader.read(&inst)) {
          DEBUG_PRINT_INST("GAPIL_REPLAY_ASM_INST_SWITCHTHREAD(index: %" PRIu32
                           ")",
                           inst.index);
          CX(Opcode::SWITCH_THREAD, inst.index);
        }
        break;
      }
    }
  }

  // Free the instructions as they are now no longer needed.
  gapil_free(arena_, data_->stream.data);

  // The stream is now a stream of opcodes.
  data_->stream = opcodes_.release_ownership();
}

void Builder::build_resources() {
  auto ex = reinterpret_cast<DataEx*>(data_->data_ex);
  auto count = ex->resources.size();
  auto size = count * sizeof(gapil_replay_resource_info);
  gapil::Buffer resources(arena_, size);
  resources.set_size(size);
  for (auto it : ex->resources) {
    gapil_replay_resource_info info;
    memcpy(info.id, it.first, sizeof(info.id));
    info.size = it.second.size;
    resources.write(it.second.index * sizeof(gapil_replay_resource_info), info);
  }
  data_->resources = resources.release_ownership();
}

gapil_replay_asm_value Builder::remap(gapil_replay_asm_value v) {
  auto ex = reinterpret_cast<DataEx*>(data_->data_ex);

  if (v.data_type >= GAPIL_REPLAY_ASM_TYPE_OBSERVED_POINTER_NAMESPACE_0) {
    auto ns = DataEx::Namespace(
        v.data_type - GAPIL_REPLAY_ASM_TYPE_OBSERVED_POINTER_NAMESPACE_0);
    const auto& reserved = ex->reserved[ns];
    auto idx = reserved.index_of(v.data);

    if (idx < 0) {
      GAPID_WARNING("Pointer 0x%" PRIx64 "@%d not reserved", v.data, ns);
      return gapil_replay_asm_value{unobservedPointer,
                                    GAPIL_REPLAY_ASM_TYPE_ABSOLUTE_POINTER};
    } else {
      const auto& offsets = reserved_base_offsets_[ns];
      auto remapped = offsets[idx] + v.data - reserved[idx].mStart;
      return gapil_replay_asm_value{remapped,
                                    GAPIL_REPLAY_ASM_TYPE_VOLATILE_POINTER};
    }
  }
  return v;
}

void Builder::push(gapil_replay_asm_value val) {
  auto v = val.data;
  auto t = val.data_type;
  switch (t) {
    case GAPIL_REPLAY_ASM_TYPE_FLOAT: {
      pushi(GAPIL_REPLAY_ASM_TYPE_FLOAT, v >> 23);
      if ((v & 0x7fffff) != 0) {
        extend(v & 0x7fffff);
      }
      break;
    }
    case GAPIL_REPLAY_ASM_TYPE_DOUBLE: {
      pushi(t, v >> 52);
      v = v & mask52;
      if (v != 0) {
        extend(v >> 26);
        extend(v & mask26);
      }
      break;
    }
    case GAPIL_REPLAY_ASM_TYPE_INT8:
    case GAPIL_REPLAY_ASM_TYPE_INT16:
    case GAPIL_REPLAY_ASM_TYPE_INT32:
    case GAPIL_REPLAY_ASM_TYPE_INT64: {
      // Signed PUSHI types are sign-extended
      if ((v & ~mask19) == 0) {
        // ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
        //                                            ▕      PUSHI 20     ▕
        pushi(t, v);
      } else if ((v & ~mask19) == ~mask19) {
        // ●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
        //                                            ▕      PUSHI 20     ▕
        pushi(t, v & mask20);
      } else if ((v & ~mask45) == 0) {
        // ○○○○○○○○○○○○○○○○○○○◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
        //                  ▕      PUSHI 20     ▕         EXTEND 26       ▕
        pushi(t, v >> 26);
        extend(v & mask26);
      } else if ((v & ~mask45) == ~mask45) {
        // ●●●●●●●●●●●●●●●●●●●◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
        //                  ▕      PUSHI 20     ▕         EXTEND 26       ▕
        pushi(t, (v >> 26) & mask20);
        extend(v & mask26);
      } else {
        // ◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
        //▕  PUSHI 12 ▕         EXTEND 26       ▕         EXTEND 26       ▕
        pushi(t, v >> 52);
        extend((v >> 26) & mask26);
        extend(v & mask26);
      }
      break;
    }
    case GAPIL_REPLAY_ASM_TYPE_BOOL:
    case GAPIL_REPLAY_ASM_TYPE_UINT8:
    case GAPIL_REPLAY_ASM_TYPE_UINT16:
    case GAPIL_REPLAY_ASM_TYPE_UINT32:
    case GAPIL_REPLAY_ASM_TYPE_UINT64:
    case GAPIL_REPLAY_ASM_TYPE_ABSOLUTE_POINTER:
    case GAPIL_REPLAY_ASM_TYPE_CONSTANT_POINTER:
    case GAPIL_REPLAY_ASM_TYPE_VOLATILE_POINTER: {
      if ((v & ~mask20) == 0) {
        // ○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○○◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
        //                                            ▕      PUSHI 20     ▕
        pushi(t, v);
      } else if ((v & ~mask46) == 0) {
        // ○○○○○○○○○○○○○○○○○○◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
        //                  ▕      PUSHI 20     ▕         EXTEND 26       ▕
        pushi(t, v >> 26);
        extend(v & mask26);
      } else {
        // ◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒◒
        //▕  PUSHI 12 ▕         EXTEND 26       ▕         EXTEND 26       ▕
        pushi(t, v >> 52);
        extend((v >> 26) & mask26);
        extend(v & mask26);
      }
      break;
    }
    default:
      GAPID_FATAL("Cannot push value type %d", t);
  }
}

void Builder::load(gapil_replay_asm_value val, gapil_replay_asm_type ty) {
  if ((val.data & ~mask20) == 0) {
    switch (val.data_type) {
      case GAPIL_REPLAY_ASM_TYPE_CONSTANT_POINTER:
        CYZ(Opcode::LOAD_C, ty, val.data);
        return;
      case GAPIL_REPLAY_ASM_TYPE_VOLATILE_POINTER:
        CYZ(Opcode::LOAD_V, ty, val.data);
        return;
      default:
        break;
    }
  }
  push(val);
  CX(Opcode::LOAD, ty);
}

void Builder::store(gapil_replay_asm_value dst) {
  if ((dst.data & ~mask20) == 0 &&
      dst.data_type == GAPIL_REPLAY_ASM_TYPE_VOLATILE_POINTER) {
    CX(Opcode::STORE_V, dst.data);
  } else {
    push(dst);
    C(Opcode::STORE);
  }
}

}  // anonymous namespace

void gapil_replay_build(context* ctx, gapil_replay_data* data) {
  Builder builder(ctx->arena, data);
  builder.layout_volatile_memory();
  builder.generate_opcodes();
  builder.build_resources();
}