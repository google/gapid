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

#include "gapil/runtime/cc/runtime.h"

void applyReads(gapil_context*);
void applyWrites(gapil_context*);
void* resolvePoolData(gapil_context*, gapil_pool*, uint64_t ptr,
                      gapil_data_access, uint64_t size);
void callExtern(gapil_context*, uint8_t* name, void* args, void* res);
void copySlice(gapil_context*, gapil_slice* dst, gapil_slice* src);
void cstringToSlice(gapil_context*, uint64_t ptr, gapil_slice* out);
void storeInDatabase(gapil_context*, void* ptr, uint64_t size, uint8_t* id_out);
gapil_pool* makePool(gapil_context*, uint64_t size);
void freePool(gapil_pool*);

void cloneSlice(gapil_context*, gapil_slice* dst, gapil_slice* src);