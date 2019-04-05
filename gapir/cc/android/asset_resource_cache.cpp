/*
 * Copyright (C) 2019 Google Inc.
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

#include "asset_resource_cache.h"
#include "gapir/cc/replay_service.h"

#include "core/cc/log.h"

#include <android/asset_manager.h>
#include <errno.h>
#include <string.h>
#include <unistd.h>

namespace gapir {

namespace {

const char *kAssetPathResourcesIndex = "replay_export/resources.index";
const char *kAssetPathResourcesData = "replay_export/resources.data";

// asset_read bails out if the read fails. Otherwise, it returns true unless EOF
// is reached.
bool asset_read(AAsset *asset, void *buf, size_t count) {
  int ret = AAsset_read(asset, buf, count);
  if (ret < 0) {
    GAPID_FATAL("Error on asset read");
  }
  if (ret == 0) {
    return false;
  }
  if (ret != count) {
    GAPID_FATAL("Asset read only %d bytes out of %zu bytes required", ret,
                count);
  }
  return true;
}

}  // namespace

std::unique_ptr<ResourceCache> AssetResourceCache::create(
    AAssetManager *assetManager) {
  return std::unique_ptr<ResourceCache>(new AssetResourceCache(assetManager));
}

AssetResourceCache::AssetResourceCache(AAssetManager *assetManager) {
  mAssetManager = assetManager;
  AAsset *asset_resource_index = AAssetManager_open(
      mAssetManager, kAssetPathResourcesIndex, AASSET_MODE_STREAMING);

  // Load the archive index in memory.
  for (;;) {
    uint32_t idSize;
    if (!asset_read(asset_resource_index, &idSize, sizeof(idSize))) break;
    std::string id(idSize, 0);
    uint64_t offset;
    uint32_t size;
    if (!asset_read(asset_resource_index, &id.front(), idSize) ||
        !asset_read(asset_resource_index, &offset, sizeof(offset)) ||
        !asset_read(asset_resource_index, &size, sizeof(size))) {
      break;
    }
    mRecords.emplace(id, AssetRecord{offset, size});
  }

  AAsset_close(asset_resource_index);

  // Open the resource data file descriptor
  AAsset *asset_resource_data = AAssetManager_open(
      mAssetManager, kAssetPathResourcesData, AASSET_MODE_STREAMING);
  off64_t length;
  mResourceDataFd = AAsset_openFileDescriptor64(asset_resource_data,
                                                &mResourceDataStart, &length);
  if (mResourceDataFd < 0) {
    GAPID_FATAL(
        "AssetResourceCache::AssetResourceCache() cannot open resource "
        "data asset as a file descriptor (due to compressed asset?)");
  }
  AAsset_close(asset_resource_data);
}

AssetResourceCache::~AssetResourceCache() {
  if (mResourceDataFd >= 0) {
    close(mResourceDataFd);
  }
}

bool AssetResourceCache::putCache(const Resource &resource, const void *data) {
  // AssetResourceCache is read-only, putCache always fails.
  return false;
}

bool AssetResourceCache::hasCache(const Resource &resource) {
  return (mRecords.find(resource.id) != mRecords.end());
}

bool AssetResourceCache::loadCache(const Resource &resource, void *data) {
  std::unordered_map<std::string, AssetRecord>::const_iterator it;
  it = mRecords.find(resource.id);
  if (it == mRecords.end()) {
    GAPID_FATAL("AssetResourceCache::loadCache() cannot find resource: %s",
                resource.id.c_str());
  }

  AssetRecord record = it->second;

  off64_t offset = mResourceDataStart + record.offset;
  off64_t ret = lseek64(mResourceDataFd, offset, SEEK_SET);
  if (ret == (off64_t)-1) {
    GAPID_FATAL("AssetResourceCache::loadCache() lseek64() failed");
  }

  size_t left_to_read = record.size;
  char *p = (char *)data;

  while (left_to_read > 0) {
    ssize_t read_this_time = read(mResourceDataFd, p, left_to_read);

    if (read_this_time == (ssize_t)-1) {
      char *errmsg = strerror(errno);
      GAPID_FATAL(
          "AssetResourceCache::loadCache() read() failed, errno: %d, strerror: "
          "%s",
          errno, errmsg);
    }
    if (read_this_time > left_to_read) {
      GAPID_FATAL(
          "AssetResourceCache::loadCache() read() returned"
          "more (%zu) than what is was asked for (%zu)",
          read_this_time, left_to_read);
    }
    left_to_read -= (size_t)read_this_time;
    p += read_this_time;
  }

  return true;
}

}  // namespace gapir
