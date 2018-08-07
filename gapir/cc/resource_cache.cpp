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

#include "resource_cache.h"

#include <memory>

#include "core/cc/assert.h"
#include "replay_connection.h"

namespace gapir {

ResourceCache::ResourceCache(std::unique_ptr<ResourceProvider> fallbackProvider)
    : mFallbackProvider(std::move(fallbackProvider)) {}

bool ResourceCache::get(const Resource* resources, size_t count,
                        ReplayConnection* conn, void* target, size_t size) {
  uint8_t* dst = reinterpret_cast<uint8_t*>(target);
  Batch batch(dst, size);
  for (size_t i = 0; i < count; i++) {
    const Resource& resource = resources[i];
    if (size < resource.size) {
      return false;  // Not enough space
    }
    // Try fetching the resource from the cache.
    if (getCache(resource, dst)) {
      // In cache. Flush the pending requests.
      // Note: This implementation can result in many round trips to the
      // GAPIS, because whenever a cache hit happens, all the pending
      // resources accumulated before this resource must be fetched and
      // loaded prior to the cached resource to be loaded. The original
      // design was based around the idea that we could load the resource
      // directly into the destination buffer without temporary copies.
      // As gRPC forces us to have temporary copies, this implementation
      // should be changed to have a single fetch.
      // TODO: Update this batching logic to reduce the number of resource
      // fetching calls.
      if (!batch.flush(*this, conn)) {
        return false;  // Failed to get resources from fallback provider.
      }
      batch = Batch(dst, size);
    } else {
      // Not in cache.
      // Add this to the batch we need to request from the fallback provider.
      batch.append(resource);
    }
    dst += resource.size;
    size -= resource.size;
  }
  return batch.flush(*this, conn);
}

ResourceCache::Batch::Batch(void* target, size_t size)
    : mTarget(reinterpret_cast<uint8_t*>(target)), mSize(0), mSpace(size) {
  GAPID_ASSERT(target != nullptr);
}

bool ResourceCache::Batch::append(const Resource& resource) {
  if (mTarget == nullptr) {
    GAPID_FATAL("Cannot append after flush.");
  }
  if (mSpace < resource.size) {
    return false;
  }
  mResources.push_back(resource);
  mSize += resource.size;
  mSpace -= resource.size;
  return true;
}

bool ResourceCache::Batch::flush(ResourceCache& cache, ReplayConnection* conn) {
  uint8_t* ptr = mTarget;
  mTarget = nullptr;  // nullptr is used for detecting append-after-flush.
  size_t count = mResources.size();
  if (count == 0) {
    return true;
  }
  size_t one_req_limit = 100 * 1024 * 1024;  // limit 100MB
  size_t i = 0;
  size_t j = 0;
  size_t one_req_size = 0;
  while (i < count) {
    if (((i != j) && (one_req_size + mResources[j].size > one_req_limit)) ||
        (j == mResources.size())) {
      if (!cache.mFallbackProvider->get(&mResources[i], j - i, conn, ptr,
                                        one_req_size)) {
        return false;
      }
      while (i < j) {
        cache.putCache(mResources[i], ptr);
        ptr += mResources[i].size;
        i++;
      }
      one_req_size = 0;
    }
    if (j < mResources.size()) {
      one_req_size += mResources[j].size;
      j++;
    }
  }
  return true;
}

}  // namespace gapir
