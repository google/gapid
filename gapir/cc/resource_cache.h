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

#ifndef GAPIR_RESOURCE_CACHE_H
#define GAPIR_RESOURCE_CACHE_H

#include "resource_provider.h"

#include <memory>
#include <vector>

#include "replay_connection.h"

namespace gapir {

// ResourceCache is an abstract base class for caching resource providers.
// TODO: simplify the caching logic as we adopted gRPC.
class ResourceCache : public ResourceProvider {
public:
    ResourceCache(std::unique_ptr<ResourceProvider> fallbackProvider);

    // Loads count resources from the provider and writes them, in-order, to target.
    // If the net size of all the resources exceeds size, then false is returned.
    bool get(const Resource* resources, size_t count, ReplayConnection* conn,
             void* target, size_t size) override;

protected:
    virtual void putCache(const Resource& resource, const void* data) = 0;
    virtual bool getCache(const Resource& resource, void* data) = 0;

    // Fall back resource provider for the cases when the requested resource is not in the cache.
    std::unique_ptr<ResourceProvider> mFallbackProvider;

    // Batch is a helper class for accumulating resources to request on the fallback provider.
    class Batch {
    public:
        // Constructor.
        // target is the pointer to the memory to hold the fetched resources on calling flush.
        // size is the size in bytes of the buffer at target.
        Batch(void* target, size_t size);

        // append adds the resource to the batch.
        // Returns true if the resource fits in the remaining buffer space, otherwise false.
        bool append(const Resource& resource);

        // flush requests the resources appended to the Batch from the fallback provider.
        // The requested resources are added to the cache using putCache.
        // Once called, the batch must not be used again.
        // Returns true if all resources were fetched, otherwise false.
        bool flush(ResourceCache& cache, ReplayConnection* conn);
    private:
        std::vector<Resource> mResources; // Batch of resources to request.
        uint8_t* mTarget;                 // Target address of resource data.
        size_t mSize;                     // Target buffer size.
        size_t mSpace;                    // mSize - size of all resources in batch.
    };
};

} // namespace gapir

#endif  // GAPIR_RESOURCE_CACHE_H
