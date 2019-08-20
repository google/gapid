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

#include "replay_service.h"
#include "resource.h"
#include "resource_loader.h"

#include <memory>
#include <vector>

namespace gapir {

// ResourceCache is an abstract base class for caching resources.
class ResourceCache {
 public:
  enum class PrefetchMode { DEFERRED_PREFETCH, IMMEDIATE_PREFETCH };

 public:
  ResourceCache(PrefetchMode mode = PrefetchMode::DEFERRED_PREFETCH)
      : mPrefetchMode(mode) {}
  virtual ~ResourceCache() {}

  // putCache caches the given resource and its data. Returns true if caching
  // is done successfully, otherwise false.
  virtual bool putCache(const Resource& res, const void* resData) = 0;
  // hasCache returns true if the given resource has been cached, otherwise
  // false.
  virtual bool hasCache(const Resource& res) = 0;
  // loadCache loads the resource data to the given target location. Returns
  // true if loading is done successfully, otherwise false and do not load
  // anything to the target location.
  virtual bool loadCache(const Resource& res, void* target) = 0;
  // size returns the total size in bytes that can be used for caching.
  virtual size_t totalCacheSize() const = 0;
  // returns the unused capacity of the cache in bytes.
  virtual size_t unusedSize() const = 0;
  // resize adjust the total size in bytes that can be used for this cache.
  // this does not actually resize the cache upwards, it can only be used to
  // ensure the cache uses no more than newSize bytes on return. Please note the
  // cache may later resize upwards again.
  virtual bool resize(size_t newSize) = 0;
  // set the anticipated resources and access order, so that on cache misses,
  // the cache can fetch not only the missing resource, but also an anticipated
  // lookahead. also fills any free space in the cache with the first N
  // resources that fit in storage.
  void setPrefetch(const std::vector<Resource>& resources,
                   std::unique_ptr<ResourceLoader> fetcher);
  // debug print the internal state.
  virtual void dump(FILE*) {}

 protected:
  std::vector<Resource> anticipateNextResources(const Resource& resource,
                                                size_t bytesToFetch);

  virtual size_t prefetchImpl(const std::vector<Resource>& resources);

 private:
  PrefetchMode mPrefetchMode;

  std::vector<Resource> mResources;
  std::map<ResourceId, std::vector<Resource>::iterator> mResourceIterators;

  std::unique_ptr<ResourceLoader> mFetcher;
};

}  // namespace gapir

#endif  // GAPIR_RESOURCE_CACHE_H
