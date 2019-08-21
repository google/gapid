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

#ifndef GAPIR_ON_DISK_RESOURCE_CACHE_H
#define GAPIR_ON_DISK_RESOURCE_CACHE_H

#include "replay_service.h"
#include "resource_cache.h"
#include "resource_loader.h"

#include "core/cc/archive.h"

#include <limits>
#include <memory>
#include <string>

#if TARGET_OS == GAPID_OS_LINUX || TARGET_OS == GAPID_OS_OSX
#include <unistd.h>
#endif

namespace gapir {

// Unlimited size cache on disk for resources
class OnDiskResourceCache : public ResourceCache {
 public:
  // Creates new disk cache with the specified base path. If the base path is
  // not readable or it can't be created then returns the fall back provider.
  static std::unique_ptr<ResourceCache> create(const std::string& path,
                                               bool cleanUp);
  virtual ~OnDiskResourceCache() {
#if TARGET_OS == GAPID_OS_LINUX || TARGET_OS == GAPID_OS_OSX
    if (mCleanUp) {
      unlink(mArchive.dataFilePath().c_str());
      unlink(mArchive.indexFilePath().c_str());
    }
#endif
  }

  // ResourceCache interface implementation
  virtual bool putCache(const Resource& res, const void* resData) override;
  virtual bool hasCache(const Resource& res) override;
  virtual bool loadCache(const Resource& res, void* target) override;

  // Unlimited size for on-disk cache.
  virtual size_t totalCacheSize() const override {
    return std::numeric_limits<size_t>::max();
  }
  virtual size_t unusedSize() const override {
    return std::numeric_limits<size_t>::max();
  }
  // Do not support resize.
  virtual bool resize(size_t newSize) override { return true; };

 private:
  OnDiskResourceCache(const std::string& path, bool cleanUp);

  // Disk-backed archive holding the cached resources.
  core::Archive mArchive;

  // Delete archive files when this On-disk cache is out of scope.
  bool mCleanUp;
};

}  // namespace gapir

#endif  // GAPIR_ON_DISK_RESOURCE_CACHE_H
