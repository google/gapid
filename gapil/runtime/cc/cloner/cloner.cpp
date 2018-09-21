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

#include "cloner.h"

#include "core/cc/log.h"
#include "core/memory/arena/cc/arena.h"
#include "gapil/runtime/cc/map.inc"

#if 0
#define DEBUG_PRINT(...) GAPID_INFO(__VA_ARGS__)
#else
#define DEBUG_PRINT(...)
#endif

namespace {

struct tracker {
  tracker(core::Arena* a) : arena(a), map(a) {}
  core::Arena* arena;
  gapil::Map<void*, void*, false> map;
};

}  // anonymous namespace

extern "C" {

void* gapil_create_clone_tracker(arena* arena) {
  auto a = reinterpret_cast<core::Arena*>(arena);
  auto out = a->create<tracker>(a);
  DEBUG_PRINT("gapil_create_clone_tracker(arena: %p) -> %p", arena, out);
  return out;
}

// gapil_destroy_clone_tracker deletes the tracker created by
// gapil_create_clone_tracker.
void gapil_destroy_clone_tracker(void* ct) {
  DEBUG_PRINT("gapil_destroy_clone_tracker(tracker: %p)", ct);
  auto t = reinterpret_cast<tracker*>(ct);
  t->arena->destroy(t);
}

// gapil_clone_tracker_lookup returns a pointer to the previously cloned object,
// or nullptr if this object has not been cloned before.
void* gapil_clone_tracker_lookup(void* t, void* object) {
  DEBUG_PRINT("tracker count: %d", reinterpret_cast<tracker*>(t)->map.count());
  auto out = reinterpret_cast<tracker*>(t)->map.findOrZero(object);
  DEBUG_PRINT("gapil_clone_tracker_lookup(tracker: %p, object: %p) -> %p", t,
              object, out);
  return out;
}

// gapil_clone_tracker_track associates the original object to its cloned
// version.
void gapil_clone_tracker_track(void* t, void* original, void* cloned) {
  DEBUG_PRINT(
      "gapil_clone_tracker_track(tracker: %p, original: %p, cloned: %p)", t,
      original, cloned);
  reinterpret_cast<tracker*>(t)->map[original] = cloned;
}

}  // extern "C"
