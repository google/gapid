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

void ResourceCache::setPrefetch(const std::vector<Resource>& resources,
                                std::unique_ptr<ResourceLoader> fetcher) {
  mResources.clear();
  mResourceIterators.clear();

  mResources.reserve(resources.size());

  for (unsigned int i = 0; i < resources.size(); ++i) {
    mResources.push_back(resources[i]);
    mResourceIterators[resources[i].getID()] = mResources.end() - 1;
  }

  mFetcher = std::move(fetcher);

  if (mPrefetchMode == PrefetchMode::IMMEDIATE_PREFETCH &&
      resources.size() > 0) {
    auto fetch = anticipateNextResources(resources[0], unusedSize());
    if (resources.size() > 0) {
      prefetchImpl(resources);
    }
  }
}

std::vector<Resource> ResourceCache::anticipateNextResources(
    const Resource& resource, size_t bytesToFetch) {
  std::vector<Resource> expectedResources;

  size_t bytesSoFar = 0;

  auto resMapIter = mResourceIterators.find(resource.getID());
  if (resMapIter == mResourceIterators.end()) {
    // We don't know about this resource. Return the empty vector.
    return expectedResources;
  }

  auto resIter = resMapIter->second;

  if (resIter != mResources.end()) {
    ++resIter;
  }

  // mResources is not a perfect chronological ordering of the resource
  // access pattern of the replay. It is sorted by resource "first use
  // order". That is A, B, C, D, E, C, F will reduce to A, B, C, D, E, F.
  // This may be the origin of cache prefetch mispredictions if a resource
  // provokes a cache miss on a second or subsequent use in a given replay.
  // In empirical measurements this has not proved to be a significant problem
  // yet, so I've kept things simple and not added extra complexity to deal
  // with the issue.

  // The following loop also returns anticipated resources without concern
  // for whether they are already in the cache. If some of the resources
  // returned ARE already in the cache, then the total bytes fetched by
  // a call to prefetchImpl() will be less than bytesToFetch.
  // This could be compensated for by doing an in-cache check here
  // but would also result in further look-ahead and accordingly greater cost
  // in the case of a cache mispredict. This compromise works well in my
  // measurements so I'm going to keep it simple for now.

  for (unsigned int i = 0;
       resIter != mResources.end() && bytesSoFar < bytesToFetch; ++i) {
    expectedResources.push_back(*resIter);
    bytesSoFar += resIter->getSize();
    resIter++;
  }

  return expectedResources;
}

size_t ResourceCache::prefetchImpl(const std::vector<Resource>& resources) {
  std::vector<Resource> uncachedResources;
  uncachedResources.reserve(resources.size());

  size_t numResourcesAlreadyCached = 0;

  for (size_t i = 0; i < resources.size(); i++) {
    const auto& resource = resources[i];

    if (hasCache(resource)) {
      numResourcesAlreadyCached++;
      continue;
    }

    uncachedResources.push_back(resource);
  }

  ResourceLoadingBatch bat;

  GAPID_INFO(
      "Prefetching %zu new uncached resources (%zu / %zu resources should be "
      "in "
      "cache after prefetch)...",
      uncachedResources.size(),
      uncachedResources.size() + numResourcesAlreadyCached, resources.size());

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
