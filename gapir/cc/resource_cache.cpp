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
size_t ResourceCache::prefetch(const Resource* res, size_t count,
                               ResourceLoader* fetcher) {
  size_t res_sum = 0;
  std::vector<Resource> uncached;
  uncached.reserve(count);
  size_t alreadyCached = 0;

  for (size_t i = 0; i < count; i++) {
    const auto& r = res[i];
    if (hasCache(r)) {
      alreadyCached++;
      continue;
    }
    if (res_sum + r.size > totalCacheSize()) {
      break;
    }
    uncached.push_back(r);
    res_sum += r.size;
  }
  GAPID_INFO(
      "Prefetching %zu new uncached resources (%zu / %zu resources will be in "
      "cache after prefetch)...",
      uncached.size(), uncached.size() + alreadyCached, count);
  ResourceLoadingBatch bat;
  auto fetchBatch = [&bat, fetcher, this]() {
    auto fetched =
        fetcher->fetch(bat.resources().data(), bat.resources().size());
    size_t put_sum = 0;
    for (size_t i = 0; i < bat.resources().size(); i++) {
      putCache(bat.resources().at(i),
               reinterpret_cast<const uint8_t*>(fetched->data()) + put_sum);
      put_sum += bat.resources().at(i).size;
    }
    bat.clear();
  };

  for (auto& r : uncached) {
    if (!bat.append(r, nullptr)) {
      fetchBatch();
      bat.append(r, nullptr);
    }
  }
  if (bat.size() > 0) {
    fetchBatch();
  }
  return uncached.size();
}
}  // namespace gapir
