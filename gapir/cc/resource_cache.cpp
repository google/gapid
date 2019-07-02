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
#include "resource.h"
#include "resource_loader.h"

#include "core/cc/log.h"

#include <vector>

namespace gapir {

size_t ResourceCache::setPrefetch(const Resource* resources, size_t count,
                                  std::unique_ptr<ResourceLoader> fetcher) {
  mResources.clear();
  mResourceIterators.clear();

  mResources.reserve(count);

  for (unsigned int i = 0; i < count; ++i) {
    mResources.push_back(resources[i]);
    mResourceIterators[resources[i].getID()] = mResources.end() - 1;
  }

  mFetcher = std::move(fetcher);

  return prefetchImpl(resources, count);
}

std::vector<Resource> ResourceCache::anticipateNextResources(
    const Resource& resource, size_t bytesToFetch) {
  std::vector<Resource> expectedResources;

  size_t bytesSoFar = 0;

  auto resMapIter = mResourceIterators.find(resource.getID());
  if (resMapIter == mResourceIterators.end()) {
    return expectedResources;  // We don't know about this resource so we're
                               // blind. Return the empty vector.
  }

  std::vector<Resource>::iterator resIter = resMapIter->second;

  if (resIter != mResources.end()) {
    ++resIter;
  }

  for (unsigned int i = 0;
       resIter != mResources.end() && bytesSoFar < bytesToFetch; ++i) {
    expectedResources.push_back(*resIter);
    bytesSoFar += resIter->getSize();
    resIter++;
  }

  return expectedResources;
}

size_t ResourceCache::prefetchImpl(const Resource* resources, size_t count,
                                   bool allowEviction) {
  size_t bytesToFetch = 0;

  std::vector<Resource> uncachedResources;
  uncachedResources.reserve(count);

  size_t numResourcesAlreadyCached = 0;

  for (size_t i = 0; i < count; i++) {
    const auto& resource = resources[i];

    if (hasCache(resource)) {
      numResourcesAlreadyCached++;
      continue;
    }

    if (bytesToFetch + resource.getSize() >
        (allowEviction ? totalCacheSize() : unusedSize())) {
      break;
    }

    uncachedResources.push_back(resource);
    bytesToFetch += resource.getSize();
  }

  ResourceLoadingBatch bat;

  GAPID_INFO(
      "Prefetching %zu new uncached resources (%zu / %zu resources should be "
      "in "
      "cache after prefetch)...",
      uncachedResources.size(),
      uncachedResources.size() + numResourcesAlreadyCached, count);

  auto fetchBatch = [&bat, this]() {
    auto fetched =
        this->mFetcher->fetch(bat.resources().data(), bat.resources().size());

    size_t put_sum = 0;

    for (size_t i = 0; i < bat.resources().size(); i++) {
      putCache(bat.resources().at(i),
               reinterpret_cast<const uint8_t*>(fetched->data()) + put_sum);
      put_sum += bat.resources().at(i).getSize();
    }

    bat.clear();
  };

  for (auto& resource : uncachedResources) {
    if (!bat.append(resource, nullptr)) {
      fetchBatch();
      bat.append(resource, nullptr);
    }
  }

  if (bat.size() > 0) {
    fetchBatch();
  }

  GAPID_INFO("Prefetching complete.");

  return uncachedResources.size();
}

}  // namespace gapir
