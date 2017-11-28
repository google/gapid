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

#include "pool.h"

#include "core/cc/log.h"

#define __STDC_FORMAT_MACROS
#include <inttypes.h>

#include <cstdlib>

namespace gapii {

std::shared_ptr<Pool> Pool::create(uint32_t id, uint64_t size) {
    return std::shared_ptr<Pool>(new Pool(id, size));
}

std::shared_ptr<Pool> Pool::create_virtual(uint32_t id, uint64_t size) {
  return std::shared_ptr<Pool>(new Pool(Pool::virtual_pool{}, id, size));
}

Pool::Pool(uint32_t id, uint64_t size) : mId(id), mData(calloc(size,1)), mSize(size), mIsVirtual(false) {
  if (mData == nullptr) {
    GAPID_FATAL("Out of memory allocating 0x%" PRIx64 " bytes", size);
  }
}

Pool::Pool(virtual_pool, uint32_t id, uint64_t size) : mId(id), mData(nullptr), mSize(size), mIsVirtual(true) {
}

Pool::~Pool() {
    if (mData) {
      free(mData);
    }
}

}  // namespace gapii

