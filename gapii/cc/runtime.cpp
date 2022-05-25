/*
 * Copyright (C) 2017 Google Inc.
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

#include "core/memory/arena/cc/arena.h"

#include "gapil/runtime/cc/encoder.h"
#include "gapil/runtime/cc/runtime.h"

#include "gapii/cc/call_observer.h"
#include "gapii/cc/pack_encoder.h"

#if 0
#define DEBUG_PRINT(...) GAPID_DEBUG(__VA_ARGS__)
#else
#define DEBUG_PRINT(...)
#endif

extern "C" {

int64_t gapil_encode_type(context* ctx, const char* name, uint32_t desc_size,
                          const void* desc) {
  DEBUG_PRINT("gapil_encode_type(%p, %s, %d, %p)", ctx, name, desc_size, desc);
  auto cb = static_cast<gapii::CallObserver*>(ctx);
  auto e = cb->encoder();
  auto res = e->type(name, desc_size, desc);
  auto id = static_cast<int64_t>(res.first);
  auto isnew = res.second;
  return isnew ? id : -id;
}

void* gapil_encode_object(context* ctx, uint8_t is_group, uint32_t type,
                          uint32_t data_size, void* data) {
  DEBUG_PRINT("gapil_encode_object(%p, %s, %d, %d, %p)", ctx,
              is_group ? "true" : "false", type, data_size, data);
  auto cb = static_cast<gapii::CallObserver*>(ctx);
  auto e = cb->encoder();
  if (is_group) {
    return e->group(type, data_size, data);
  }
  e->object(type, data_size, data);
  return nullptr;
}

void gapil_slice_encoded(context* ctx, const void* slice) {
  DEBUG_PRINT("gapil_on_encode_slice(%p, %p)", ctx, slice);
  auto cb = static_cast<gapii::CallObserver*>(ctx);
  cb->slice_encoded(reinterpret_cast<const slice_t*>(slice));
}

int64_t gapil_encode_backref(context* ctx, const void* object) {
  auto cb = static_cast<gapii::CallObserver*>(ctx);
  auto res = cb->reference_id(object);
  auto id = static_cast<int64_t>(res.first);
  auto isnew = res.second;
  DEBUG_PRINT("gapil_encode_backref(%p, %p) -> new: %s id: %d", ctx, object,
              isnew ? "true" : "false", int(id));
  return isnew ? id : -id;
}

}  // extern "C"
