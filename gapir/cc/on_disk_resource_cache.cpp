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

#include "on_disk_resource_cache.h"
#include "replay_service.h"

#include "core/cc/log.h"

#include <errno.h>
#include <stdio.h>
#include <sys/stat.h>

#include <memory>
#include <string>
#include <utility>
#include <vector>

#if TARGET_OS == GAPID_OS_WINDOWS
#include <direct.h>
#define mkdir(path, mode) _mkdir(path)
#else
static const mode_t MKDIR_MODE = 0755;
#endif

namespace gapir {
namespace {

int mkdirAll(const std::string& path) {
  if (0 != mkdir(path.c_str(), MKDIR_MODE)) {
    switch (errno) {
      case ENOENT: {  // Non-existent parent(s).
        size_t pos = path.find_last_of("/\\");
        if (pos == std::string::npos) {
          return -1;
        }
        mkdirAll(path.substr(0, pos));
        return mkdir(path.c_str(), MKDIR_MODE);  // Retry.
      }
      case EEXIST:  // Already exists, return success.
        return 0;
      default:  // Something went wrong, return failure.
        return -1;
    }
  }
  return 0;
}

}  // anonymous namespace

std::unique_ptr<ResourceCache> OnDiskResourceCache::create(
    const std::string& path, bool cleanUp) {
  if (0 != mkdirAll(path)) {
    GAPID_WARNING(
        "Couldn't access/create cache directory; disabling disk cache.");
    return nullptr;  // Disk path was inaccessible.
  } else {
    std::string diskPath = path;
    if (diskPath.back() != PATH_DELIMITER) {
      diskPath.push_back(PATH_DELIMITER);
    }

    return std::unique_ptr<ResourceCache>(
        new OnDiskResourceCache(std::move(diskPath), cleanUp));
  }
}

OnDiskResourceCache::OnDiskResourceCache(const std::string& path, bool cleanUp)
    : ResourceCache(ResourceCache::PrefetchMode::IMMEDIATE_PREFETCH),
      mArchive(path + "resources"),
      mCleanUp(cleanUp) {}

bool OnDiskResourceCache::putCache(const Resource& resource, const void* data) {
  return mArchive.write(resource.getID(), data, resource.getSize());
}

bool OnDiskResourceCache::hasCache(const Resource& resource) {
  return mArchive.contains(resource.getID());
}

bool OnDiskResourceCache::loadCache(const Resource& resource, void* data) {
  return mArchive.read(resource.getID(), data, resource.getSize());
}

}  // namespace gapir
