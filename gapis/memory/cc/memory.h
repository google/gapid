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

#ifndef GAPIS_MEMORY_POOL_H
#define GAPIS_MEMORY_POOL_H

#include "gapil/runtime/cc/runtime.h"

#ifdef __cplusplus
extern "C" {
#endif

struct memory;

memory* memory_create(arena*);
void memory_destroy(memory*);

void* memory_read(memory*, gapil_slice* sli);
void memory_write(memory*, gapil_slice* sli, const void* data);
void memory_copy(memory*, gapil_slice* dst, gapil_slice* src);

#ifdef __cplusplus
}  // extern "C"
#endif

#endif  // GAPIS_MEMORY_POOL_H