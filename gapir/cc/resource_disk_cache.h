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

#ifndef GAPIR_RESOURCE_DISK_CACHE_H
#define GAPIR_RESOURCE_DISK_CACHE_H

#include "replay_connection.h"
#include "resource_cache.h"

#include "core/cc/archive.h"

#include <memory>
#include <string>

namespace gapir {

// Unlimited size disk cache for resources
class ResourceDiskCache : public ResourceCache {
 public:
  // Creates new disk cache with the specified base path. If the base path is
  // not readable or it can't be created then returns the fall back provider.
  static std::unique_ptr<ResourceProvider> create(
      std::unique_ptr<ResourceProvider> fallbackProvider,
      const std::string& path);

  // Prefetches the specified resources, caching them to disk.
  void prefetch(const Resource* resources, size_t count, ReplayConnection* conn,
                void* temp, size_t tempSize) override;

 protected:
  void putCache(const Resource& resource, const void* data) override;
  bool getCache(const Resource& resource, void* data) override;

 private:
  ResourceDiskCache(std::unique_ptr<ResourceProvider> fallbackProvider,
                    const std::string& path);

  // Disk-backed archive holding the cached resources.
  core::Archive mArchive;
};

}  // namespace gapir

#endif  // GAPIR_RESOURCE_DISK_CACHE_H
