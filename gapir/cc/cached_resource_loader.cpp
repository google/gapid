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

#include "cached_resource_loader.h"

#include <utility>
#include <vector>

#include "core/cc/assert.h"
#include "resource.h"

namespace gapir {

bool CachedResourceLoader::loadBatch(const ResourceLoadingBatch& bat) {
  if (bat.size() == 0) {
    return true;
  }
  auto res = fetch(bat.resources().data(), bat.resources().size());
  if (res == nullptr) {
    return false;
  }
  if (res->size() != bat.size()) {
    return false;
  }
  size_t readSize = 0;
  for (const auto& r : bat.resources()) {
    mCache->putCache(r,
                     reinterpret_cast<const uint8_t*>(res->data()) + readSize);
    readSize += r.getSize();
  }
  const uint8_t* src = reinterpret_cast<const uint8_t*>(res->data());
  for (const auto& dsp : bat.dstsAndSizes()) {
    memcpy(dsp.first, src, dsp.second);
    src += dsp.second;
  }
  return true;
}

bool CachedResourceLoader::load(const Resource* resources, size_t count,
                                void* target, size_t targetSize) {
  size_t totalSize = 0;
  for (size_t i = 0; i < count; i++) {
    totalSize += resources[i].getSize();
  }
  if (targetSize < totalSize) {
    return false;  // Not enough space
  }

  ResourceLoadingBatch batch;
  uint8_t* dst = reinterpret_cast<uint8_t*>(target);
  for (size_t i = 0; i < count; i++) {
    const auto& r = resources[i];
    // Check cache first
    if (mCache->loadCache(r, static_cast<void*>(dst))) {
      dst += r.getSize();
      continue;
    }
    // Not in cache, batch for fetching.
    if (!batch.append(r, dst)) {
      // Current batch full, flush the batch first.
      if (!loadBatch(batch)) {
        // batch loading failed.
        return false;
      }
      batch.clear();
      if (!batch.append(r, dst)) {
        // failed append in empty batch, should not happen
        return false;
      }
    }
    dst += r.getSize();
  }
  if (batch.size() != 0) {
    if (!loadBatch(batch)) {
      return false;
    }
  }
  return true;
}

}  // namespace gapir
