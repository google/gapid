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

#ifndef GAPIR_CACHED_RESOURCE_LOADER_H
#define GAPIR_CACHED_RESOURCE_LOADER_H

#include <memory>

#include "resource_cache.h"
#include "resource_loader.h"

namespace gapir {

// CachedResourceLoader implements the ResourceLoader interface. It drives a
// ResourceCache and requires a fallback loader for fetching uncached resources.
class CachedResourceLoader : public ResourceLoader {
 public:
  static std::unique_ptr<CachedResourceLoader> create(
      ResourceCache* cache, std::unique_ptr<ResourceLoader> fallbackLoader) {
    return std::unique_ptr<CachedResourceLoader>(
        new CachedResourceLoader(cache, std::move(fallbackLoader)));
  }
  virtual ~CachedResourceLoader() {}

  // Load first tires to find the resources in cache and loads from there if
  // they are found. If a resource is not found it uses its fallback loader to
  // fetch the resource data, loads the data to target and puts the data to its
  // cache. If the net size of all the resources exceeds targetSize, then false
  // is returned.
  virtual bool load(const Resource* resources, size_t count, void* target,
                    size_t targetSize) override;

  // Fetch defers the resources to the fallback loader to fetch the
  // ReplayService::Resource instance.
  virtual std::unique_ptr<ReplayService::Resources> fetch(
      const Resource* resources, size_t count) override {
    if (mFallbackLoader != nullptr) {
      return mFallbackLoader->fetch(resources, count);
    }
    return nullptr;
  }

  // Accessors
  ResourceCache* getCache() { return mCache; }
  ResourceLoader* getFallbackResourceLoader() { return mFallbackLoader.get(); }

 protected:
  // loadBatch fetches the resources in the Batch, put the fetched resources to
  // cache, then load the data to their corresponding destinations.
  bool loadBatch(const ResourceLoadingBatch& bat);

 private:
  CachedResourceLoader(ResourceCache* cache,
                       std::unique_ptr<ResourceLoader> fallbackLoader)
      : mCache(cache), mFallbackLoader(std::move(fallbackLoader)) {}

  ResourceCache* mCache;
  std::unique_ptr<ResourceLoader> mFallbackLoader;
};

}  // namespace gapir

#endif  // GAPIR_CACHED_RESOURCE_LOADER_H
