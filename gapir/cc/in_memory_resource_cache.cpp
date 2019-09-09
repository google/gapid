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

#define unused(x) ((void)sizeof(x))

namespace gapir {

std::unique_ptr<InMemoryResourceCache> InMemoryResourceCache::create(
    std::shared_ptr<MemoryAllocator> allocator, size_t memoryLimit) {
  return std::unique_ptr<InMemoryResourceCache>(
      new InMemoryResourceCache(allocator, memoryLimit));
}

InMemoryResourceCache::InMemoryResourceCache(
    std::shared_ptr<MemoryAllocator> allocator, size_t memoryLimit)
    : mAllocator(allocator),
      mResourceIndex(),
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

  // Try to allocate some memory. If we get an allocation failure, throw more
  // stuff out until we succeed. This might happen even if we passed the memory
  // limit check above, because we cannot control the other applications running
  // on our device and how much memory they might use. It may also fail
  // for reasons of fragmentation in the allocator.
  MemoryAllocator::Handle newMemory;
  while (newMemory == nullptr) {
    newMemory = mAllocator->allocatePurgable(res.getSize());

    if (newMemory == nullptr) {
      // Throwing out only as much data as is required to fit the new data in
      // cache is maximally efficient for cache hit rate, but also puts
      // the memory allocator under extreme pressure due to fragmentation.
      // See http://go/GapirCustomAllocator for more details on why
      // we discard half the cache's contents here.
      bool evictedSomething = evictLeastRecentlyUsed(mMemoryUse / 2);
      assert(evictedSomething);
      unused(evictedSomething);
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
  memcpy(&newMemory[0], resData, res.getSize());

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
      GAPID_INFO(ss.str().c_str());
    }

    // Get the data into the cache and return it.
    return loadCacheMiss(res, target);
  }

  // Copy the data out of the cache.
  if (target != nullptr) {
    // If the allocator purged this data, then we need to delete the cache
    // record and treat this load like a cache miss.
    if (resRecord->second.second == nullptr) {
      mAllocator->releaseAllocation(resRecord->second.second);
      mMemoryUse -= resRecord->second.first.getSize();

      mResourceIndex.erase(resRecord->second.first.getID());
      mResources.erase(resRecord);

      // Get the data into the cache and return it.
      return loadCacheMiss(res, target);
    }

    memcpy(target, &resRecord->second.second[0],
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
  if (newSize < mMemoryUse) {
    assert(evictLeastRecentlyUsed(mMemoryUse - newSize));
  }

  return true;
}

void InMemoryResourceCache::dump(FILE* out) {
  for (auto res = mResources.begin(); res != mResources.end(); ++res) {
    fprintf(out, (res == mResources.begin()) ? "┏━━━━━━━━━━━━━━━━"
                                             : "┳━━━━━━━━━━━━━━━━");
  }
  fprintf(out, "┓\n");

  for (auto res = mResources.begin(); res != mResources.end(); ++res) {
    fprintf(out, "┃ addr: %6zu ",
            (res->second.second == nullptr ? (char*)nullptr
                                           : (char*)&res->second.second[0]) -
                (char*)nullptr);
  }
  fprintf(out, "┃\n");

  for (auto res = mResources.begin(); res != mResources.end(); ++res) {
    fprintf(out, "┃ size:   %6zu ", (size_t)res->second.first.getSize());
  }
  fprintf(out, "┃\n");

  for (auto res = mResources.begin(); res != mResources.end(); ++res) {
    fprintf(out, (res == mResources.begin()) ? "┃ head           "
                                             : "┃                ");
  }
  fprintf(out, "┃\n");

  for (auto res = mResources.begin(); res != mResources.end(); ++res) {
    fprintf(out, (res == mResources.begin()) ? "┗━━━━━━━━━━━━━━━━"
                                             : "┻━━━━━━━━━━━━━━━━");
  }
  fprintf(out, "┛\n");
}

void InMemoryResourceCache::clear() {
  for (auto&& resource : mResources) {
    mAllocator->releaseAllocation(resource.second.second);
  }

  mResourceIndex.clear();
  mResources.clear();

  mMemoryUse = 0;
}

std::map<unsigned int, std::pair<Resource, MemoryAllocator::Handle> >::iterator
InMemoryResourceCache::findCache(const Resource& res) {
  auto resIndex = mResourceIndex.find(res.getID());
  if (resIndex == mResourceIndex.end()) {
    return mResources.end();
  }

  auto resRecord = mResources.find(resIndex->second);
  return resRecord;
}

bool InMemoryResourceCache::evictLeastRecentlyUsed(size_t bytes) {
  size_t bytesReleased = 0;
  auto lru = mResources.begin();

  if (lru == mResources.end()) {
    return false;
  }

  while (lru != mResources.end() &&
         (bytesReleased < bytes || (bytes == 0 && bytesReleased == 0))) {
    mAllocator->releaseAllocation(lru->second.second);
    mMemoryUse -= lru->second.first.getSize();
    bytesReleased += lru->second.first.getSize();

    mResourceIndex.erase(lru->second.first.getID());
    lru = mResources.erase(lru);
  }

  GAPID_DEBUG(
      "evictLeastRecentlyUsed evicted %zu bytes (wanted to release %zu)",
      bytesReleased, bytes);

  return true;
}

bool InMemoryResourceCache::loadCacheMiss(const Resource& res, void* target) {
  size_t tcs =
      mAllocator->getTotalSize() - mAllocator->getTotalStaticDataUsage();
  // How much could we prefetch if we wanted to completely fill (100% eviction
  // rate) the cache?
  size_t possiblePrefetch = res.getSize() < tcs ? (tcs - res.getSize()) : 0;
  // Lets prefetch 10% of that maximum figure. This is a heuristic.
  // Larger fractions of tcs will be more efficient at bulk loading resources
  // but also more costly in the case of miss-predicts on resource
  // anticipation. Also, larger fractions will cause fewer but larger pauses
  // in replay while resource data is fetched. More smaller pauses may be
  // preferable for performance work. Feel free to tune this heuristic at
  // a later date.
  size_t prefetch = possiblePrefetch / 10;

  // Try to anticipate the next few resources.
  std::vector<Resource> anticipated = anticipateNextResources(res, prefetch);

  // Don't forget the resource that kicked this cache miss off.
  anticipated.push_back(res);

  // Prefetch the anticipated resources first, so if anything gets evicted it's
  // not the resource that initiated the miss. Then finally fetch the resource
  // that caused the cache miss. This is last in the vector so it will not be
  // kicked by the LRU Policy.
  prefetchImpl(anticipated);

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
    if (resRecord->second.second == nullptr) {
      GAPID_ERROR(
          "Cache miss prefetch returned nullptr for resource %s. This is "
          "probably very "
          "bad.",
          res.getID().c_str());
      return false;
    }

    memcpy(target, &resRecord->second.second[0],
           resRecord->second.first.getSize());
  }

  // Return success.
  return true;
}

}  // namespace gapir
