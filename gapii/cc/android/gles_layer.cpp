/*
 * Copyright (C) 2019 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "gapii/cc/gles_exports.h"
#include "gapii/cc/spy.h"

#include "core/cc/get_gles_proc_address.h"

namespace {

typedef void* (*PFNEGLGETNEXTLAYERPROCADDRESS)(void*, const char*);

PFNEGLGETNEXTLAYERPROCADDRESS g_next_layer_proc_addr = nullptr;
void* g_layer_id = nullptr;

void* get_next_gles_proc_address(const char* name) {
  return g_next_layer_proc_addr(g_layer_id, name);
}

}  // anonymous namespace

extern "C" {

void AndroidGLESLayer_Initialize(
    void* layer_id, PFNEGLGETNEXTLAYERPROCADDRESS get_next_layer_proc_address) {
  GAPID_INFO("GLES Layer: InitializeLayer(%p, %p)", layer_id,
             get_next_layer_proc_address);
  g_layer_id = layer_id;
  g_next_layer_proc_addr = get_next_layer_proc_address;
  core::GetGlesProcAddress = &get_next_gles_proc_address;
}

void* AndroidGLESLayer_GetProcAddress(const char* name, void* next) {
  GAPID_DEBUG("GLES Layer: GetProcAddress(%s, %p)", name, next);
  for (int i = 0; gapii::kGLESExports[i].mName != NULL; ++i) {
    if (strcmp(name, gapii::kGLESExports[i].mName) == 0) {
      return gapii::kGLESExports[i].mFunc;
    }
  }
  GAPID_WARNING("Unhandled GLES function '%s'", name);
  return next;
}

}  // extern "C"
