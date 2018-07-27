/*
 * Copyright (C) 2018 Google Inc.
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

#ifndef GAPII_POOL_H
#define GAPII_POOL_H

#include "gapil/runtime/cc/runtime.h"

namespace gapii {

class SpyBase;

struct Pool : public gapil_pool {
  SpyBase* spy;
  uint64_t size;
  uint8_t* buffer;
  arena_t* arena;  // the arena associated with the pool.
};

}  // namespace gapii

#endif  // GAPII_POOL_H