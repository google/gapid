// Copyright (C) 2017 Google Inc.
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

package executor

// #include "gapil/runtime/cc/runtime.h"
//
// typedef context* (TCreateContext) (arena*);
// typedef void     (TDestroyContext) (context*);
// typedef uint32_t (TFunc) (void* ctx);
//
// context* create_context(TCreateContext* func, arena* a) { return func(a); }
// void destroy_context(TDestroyContext* func, context* ctx) { func(ctx); }
// uint32_t call(context* ctx, TFunc* func) { return func(ctx); }
//
// // Implemented below.
// void* remap_pointer(context* ctx, uintptr_t pointer, uint64_t length);
// void  get_code_location(context* ctx, char** file, uint32_t* line);
//
// void set_callbacks() {
//   gapil_set_pointer_remapper(&remap_pointer);
//   gapil_set_code_locator(&get_code_location);
// }
import "C"
