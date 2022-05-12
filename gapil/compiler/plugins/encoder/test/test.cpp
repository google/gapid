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

#include "test.h"

#include "core/memory/arena/cc/arena.h"

#include "gapil/runtime/cc/map.inc"
#include "gapil/runtime/cc/ref.inc"
#include "gapil/runtime/cc/runtime.h"
#include "gapil/runtime/cc/string.h"

namespace {

template <typename T>
void create_map(arena* arena, map** p) {
  auto a = reinterpret_cast<core::Arena*>(arena);
  auto m = reinterpret_cast<T*>(p);
  new (m) T(a);
}

template <typename T>
T* create_ref(arena* arena, ref** p) {
  auto a = reinterpret_cast<core::Arena*>(arena);
  auto ref = new (p) gapil::Ref<T>();
  *ref = gapil::Ref<T>::create(a);
  return ref->get();
}

}  // namespace

extern "C" {

void create_map_u32(arena* arena, map** p) {
  create_map<gapil::Map<uint32_t, uint32_t, false> >(arena, p);
}

void insert_map_u32(map* m, uint32_t k, uint32_t v) {
  auto& map = *reinterpret_cast<gapil::Map<uint32_t, uint32_t, false>*>(&m);
  map[k] = v;
}

void create_map_string(arena* arena, map** p) {
  create_map<gapil::Map<string_t, string_t, false> >(arena, p);
}

void insert_map_string(map* m, const char* k, const char* v) {
  auto& map =
      *reinterpret_cast<gapil::Map<gapil::String, gapil::String, false>*>(&m);
  map[gapil::String(map.arena(), k)] = gapil::String(map.arena(), v);
}

basic_types* create_basic_types_ref(arena* a, ref** p) {
  return create_ref<basic_types>(a, p);
}

inner_class* create_inner_class_ref(arena* a, ref** p) {
  return create_ref<inner_class>(a, p);
}

context* create_context(arena* arena) {
  context* ctx = (context_t*)gapil_alloc(arena, sizeof(context_t), 8);
  ctx->arena = arena;
  return ctx;
}

void destroy_context(context* ctx) { gapil_free(ctx->arena, ctx); }

}  // extern "C"
