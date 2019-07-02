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

#include "in_memory_resource_cache.h"
#include "replay_service.h"
#include "resource_loader.h"

#include "core/cc/assert.h"

#include <string.h>

#include <algorithm>
#include <sstream>
#include <string>
#include <utility>
#include <vector>

namespace gapir {

std::unique_ptr<InMemoryResourceCache> InMemoryResourceCache::create(
    size_t memoryLimit) {
  return std::unique_ptr<InMemoryResourceCache>(
      new InMemoryResourceCache(memoryLimit));
}

InMemoryResourceCache::InMemoryResourceCache(size_t memoryLimit)
    : mResourceIndex(),
      mResources(),
      mMemoryLimit(memoryLimit),
      mMemoryUse(0),
      mIDGenerator(0) {}

InMemoryResourceCache::~InMemoryResourceCache() {}

bool InMemoryResourceCache::putCache(const Resource& res, const void* resData) {
  if (res.getSize() > mMemoryLimit) {
    return false;
  }

  // If we need to evict anything to get this new entry to fit. Now's the time
  // to do it.
  resize(mMemoryLimit - res.getSize());

  // Get a new ID for this cache entry that is larger than any current ID.
  auto newID = mIDGenerator++;

  // Try to allocate some memory. If we get an allocation faulture, throw more
  // stuff out until we succeed. This might happen even if we passed the memory
  // limit check above, because we cannot control the other applications running
  // on our device and how much memory they might use.
  std::shared_ptr<char> newMemory = nullptr;
  while (newMemory == nullptr) {
    newMemory = std::shared_ptr<char>((new (std::nothrow) char[res.getSize()]));

    if (newMemory == nullptr) {
      assert(evictLeastRecentlyUsed());
    }
  }

  assert(mResourceIndex.find(res.getID()) == mResourceIndex.end());
  assert(mResources.find(newID) == mResources.end());

  // Enter the new allocation into our records.
  mResourceIndex[res.getID()] = newID;
  mResources[newID] = std::make_pair(res, newMemory);

  // Add the memory allocated to our record of how much we have "live"
  mMemoryUse += res.getSize();

  // Copy the bits into the cache.
  memcpy(newMemory.get(), resData, res.getSize());

  return true;
}

bool InMemoryResourceCache::hasCache(const Resource& res) {
  return findCache(res) != mResources.end();
}

bool InMemoryResourceCache::loadCache(const Resource& res, void* target) {
  mCacheAccesses++;

  // Do we have this thing in the cache? If we don't, then we need to trigger
  // loadCacheMiss()
  auto resRecord = findCache(res);
  if (resRecord == mResources.end()) {
    if ((mCacheAccesses - mCacheHits) % 1 == 0) {
      std::stringstream ss;
      ss << "Replay cache miss. " << mCacheHits << " cache hits in "
         << mCacheAccesses
         << " accesses: " << (float)mCacheHits / (float)mCacheAccesses * 100.f
         << " pc cache hit rate.";
      GAPID_DEBUG(ss.str().c_str());
    }

    // Get the data into the cache and return it.
    return loadCacheMiss(res, target);
  }

  // Copy the data out of the cache.
  if (target != nullptr) {
    memcpy(target, resRecord->second.second.get(),
           resRecord->second.first.getSize());
  }

  // Update the bookkeeping for LRU to reflect this access.
  auto newID = mIDGenerator++;
  mResourceIndex[resRecord->second.first.getID()] = newID;
  mResources[newID] = resRecord->second;
  mResources.erase(resRecord);

  // Note down the cache hit and return true
  mCacheHits++;
  return true;
}

size_t InMemoryResourceCache::totalCacheSize() const { return mMemoryLimit; }

size_t InMemoryResourceCache::unusedSize() const {
  return (size_t)(mMemoryLimit - mMemoryUse);
}

bool InMemoryResourceCache::resize(size_t newSize) {
  // Throw things out of the cache until we're below limit.
  while (newSize < mMemoryUse) {
    assert(evictLeastRecentlyUsed());
  }

  return true;
}

void InMemoryResourceCache::dump(FILE* file) { assert(false); }

void InMemoryResourceCache::clear() {
  for (auto&& resource : mResources) {
    resource.second.second = nullptr;
  }

  mResourceIndex.clear();
  mResources.clear();

  mMemoryUse = 0;
}

std::map<unsigned int, std::pair<Resource, std::shared_ptr<char> > >::iterator
InMemoryResourceCache::findCache(const Resource& res) {
  auto resIndex = mResourceIndex.find(res.getID());
  if (resIndex == mResourceIndex.end()) {
    return mResources.end();
  }

  auto resRecord = mResources.find(resIndex->second);
  return resRecord;
}

bool InMemoryResourceCache::evictLeastRecentlyUsed() {
  auto lru = mResources.begin();
  if (lru != mResources.end()) {
    lru->second.second = nullptr;
    mMemoryUse -= lru->second.first.getSize();

    mResourceIndex.erase(lru->second.first.getID());
    mResources.erase(lru);
  } else {
    return false;
  }

  return true;
}

bool InMemoryResourceCache::loadCacheMiss(const Resource& res, void* target) {
  // How much could we prefetch if we wanted to completely fill (100% eviction
  // rate) the cache?
  size_t possiblePrefetch =
      res.getSize() < totalCacheSize() ? (totalCacheSize() - res.getSize()) : 0;
  // Lets prefetch 10% of that maximum figure.
  size_t prefetch = possiblePrefetch / 10;

  // Try to anticipate the next few resources.
  std::vector<Resource> anticipated = anticipateNextResources(res, prefetch);

  // Don't forget the resource that kicked this cache miss off.
  anticipated.push_back(res);

  // Prefetch the anticipated resources first, so if anything gets evicted it's
  // not the resource that initiated the miss. Then finally fetch the resource
  // that caused the cache miss. This is last in the vector so it will not be
  // kicked by the LRU Policy.
  prefetchImpl(&anticipated[0], anticipated.size(), true);

  // Unless something went very wrong, the data should now be in cache.
  auto resRecord = findCache(res);
  if (resRecord == mResources.end()) {
    GAPID_ERROR(
        "Cache miss prefetch failed for resource %s. This is probably very "
        "bad.",
        res.getID().c_str());
    return false;
  }

  // Copy the data out of the cache.
  if (target != nullptr) {
    memcpy(target, resRecord->second.second.get(),
           resRecord->second.first.getSize());
  }

  // Return success.
  return true;
}

}  // namespace gapir
