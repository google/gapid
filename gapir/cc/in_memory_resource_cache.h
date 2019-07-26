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

#ifndef GAPIR_IN_MEMORY_RESOURCE_CACHE_H
#define GAPIR_IN_MEMORY_RESOURCE_CACHE_H

#include "memory_allocator.h"

#include "replay_service.h"
#include "resource_cache.h"

#include "core/cc/assert.h"

#include <functional>
#include <memory>
#include <unordered_map>

namespace gapir {

// Fixed size in-memory resource cache. It uses a ring buffer to store the cache
// and starts invalidating cache entries from the oldest to the newest when more
// space is required.
class InMemoryResourceCache : public ResourceCache {
 public:
  // Creates a new in-memory cache with the given fallback provider and base
  // address. The initial cache size is 0 byte.
  static std::unique_ptr<InMemoryResourceCache> create(
      std::shared_ptr<MemoryAllocator> allocator, size_t memoryLimit);

  InMemoryResourceCache(std::shared_ptr<MemoryAllocator> allocator,
                        size_t memoryLimit);
  ~InMemoryResourceCache();

  // ResourceCache interface implementation
  virtual bool putCache(const Resource& res,
                        const void* resData) override final;
  virtual bool hasCache(const Resource& res) override final;
  virtual bool loadCache(const Resource& res, void* target) override final;
  virtual size_t totalCacheSize() const override final;
  virtual size_t unusedSize() const override final;
  virtual bool resize(size_t newSize) override final;
  virtual void dump(FILE* file) override final;

  void clear();

 protected:
  std::map<unsigned int,
           std::pair<Resource, MemoryAllocator::Handle> >::iterator
  findCache(const Resource& res);
  bool evictLeastRecentlyUsed(size_t bytes = 0);

  bool loadCacheMiss(const Resource& res, void* target);

 private:
  std::shared_ptr<MemoryAllocator> mAllocator;

  std::unordered_map<ResourceId, unsigned int> mResourceIndex;
  std::map<unsigned int, std::pair<Resource, MemoryAllocator::Handle> >
      mResources;

  size_t mMemoryLimit;
  size_t mMemoryUse;

  unsigned int mIDGenerator;

  unsigned int mCacheHits = 0;
  unsigned int mCacheAccesses = 0;
};

}  // namespace gapir

#endif  // GAPIR_IN_MEMORY_RESOURCE_CACHE_H
