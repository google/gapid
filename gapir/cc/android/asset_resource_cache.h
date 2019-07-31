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

#ifndef GAPIR_ASSET_RESOURCE_CACHE_H
#define GAPIR_ASSET_RESOURCE_CACHE_H

#include <android/asset_manager.h>

#include "gapir/cc/resource_cache.h"
#include "gapir/cc/resource_loader.h"

#include <string>

namespace gapir {

// AssetResourceCache is a read-only cache based on Android Assets
class AssetResourceCache : public ResourceCache {
 public:
  // Creates new asset cache.
  static std::unique_ptr<ResourceCache> create(AAssetManager* assetManager);

  ~AssetResourceCache();

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
  AssetResourceCache(AAssetManager* assetManager);

  struct AssetRecord {
    uint64_t offset;
    uint32_t size;
  };
  std::unordered_map<std::string, AssetRecord> mRecords;
  AAssetManager* mAssetManager;
  // File descriptor data to access resources
  int mResourceDataFd;
  off64_t mResourceDataStart;
};

}  // namespace gapir

#endif  // GAPIR_ASSET_RESOURCE_CACHE_H
