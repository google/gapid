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

#include "core/cc/assert.h"

namespace gapir {

ResourceCache::ResourceCache(std::unique_ptr<ResourceProvider> fallbackProvider) :
        mFallbackProvider(std::move(fallbackProvider)) {}

bool ResourceCache::get(const Resource*         resources,
                        size_t                  count,
                        const ServerConnection& server,
                        void*                   target,
                        size_t                  size) {
    uint8_t* dst = reinterpret_cast<uint8_t*>(target);
    Batch batch(dst, size);
    for (size_t i = 0; i < count; i++) {
        const Resource& resource = resources[i];
        if (size < resource.size) {
            return false; // Not enough space
        }
        // Try fetching the resource from the cache.
        if (getCache(resource, dst)) {
            // In cache. Flush the pending requests.
            if (!batch.flush(*this, server)) {
                return false; // Failed to get resources from fallback provider.
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
    return batch.flush(*this, server);
}

ResourceCache::Batch::Batch(void* target, size_t size)
    : mTarget(reinterpret_cast<uint8_t*>(target))
    , mSize(0)
    , mSpace(size) {

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

bool ResourceCache::Batch::flush(ResourceCache& cache, const ServerConnection& server) {
    uint8_t* ptr = mTarget;
    mTarget = nullptr; // nullptr is used for detecting append-after-flush.
    size_t count = mResources.size();
    if (count == 0) {
        return true;
    }
    if (!cache.mFallbackProvider->get(mResources.data(), count, server, ptr, mSize)) {
        return false;
    }
    for (auto resource : mResources) {
        cache.putCache(resource, ptr);
        ptr += resource.size;
    }
    return true;
}

} // namespace gapir
