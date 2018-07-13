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

#ifndef __GAPIL_RUNTIME_CLONER_H__
#define __GAPIL_RUNTIME_CLONER_H__

#include "gapil/runtime/cc/runtime.h"

#ifdef __cplusplus
extern "C" {
#endif  // __cplusplus

#ifndef DECL_GAPIL_CLONER_CB
#define DECL_GAPIL_CLONER_CB(RETURN, NAME, ...) RETURN NAME(__VA_ARGS__)
#endif

// gapil_create_clone_tracker returns an opaque handle to an object that tracks
// seen objects so that clones preserve cyclic dependencies.
DECL_GAPIL_CLONER_CB(void*, gapil_create_clone_tracker, arena* arena);

// gapil_destroy_clone_tracker deletes the tracker created by
// gapil_create_clone_tracker.
DECL_GAPIL_CLONER_CB(void, gapil_destroy_clone_tracker, void* tracker);

// gapil_clone_tracker_lookup returns a pointer to the previously cloned object,
// or nullptr if this object has not been cloned before.
DECL_GAPIL_CLONER_CB(void*, gapil_clone_tracker_lookup, void* tracker,
                     void* object);

// gapil_clone_tracker_track associates the original object to its cloned
// version.
DECL_GAPIL_CLONER_CB(void, gapil_clone_tracker_track, void* tracker,
                     void* original, void* cloned);

#undef DECL_GAPIL_CLONER_CB

#ifdef __cplusplus
}  // extern "C"
#endif  // __cplusplus

#endif  // __GAPIL_RUNTIME_CLONER_H__