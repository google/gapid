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

#ifndef GAPIR_RESOURCE_H
#define GAPIR_RESOURCE_H

#include <stdint.h>

#include <string>
#include <vector>

namespace gapir {

typedef std::string ResourceId;

// Resource represent a requestable blob of data from the server.
struct Resource {
  Resource() : mID(), mSize(0) {}
  Resource(ResourceId id, uint32_t size) : mID(id), mSize(size) {}
  Resource(const Resource& other) : mID(other.mID), mSize(other.mSize) {}
  Resource(Resource&& other) : mID(other.mID), mSize(other.mSize) {}

  Resource& operator=(const Resource& other) = default;
  Resource& operator=(Resource&& other) = default;

  bool operator==(const Resource& other) const {
    return mID == other.mID && mSize == other.mSize;
  }

  uint32_t getSize() const { return mSize; }
  ResourceId getID() const { return mID; }

 private:
  ResourceId mID;
  uint32_t mSize;
};

// ResourceLoadingBatch is a helper class to group resources and their loading
// destinations. Contiguous resources will be grouped together for loading.
// TODO(qining): Drop or improve this class once we have fetch/load methods of
// ResourceLoader merged.
class ResourceLoadingBatch {
 public:
  // Limit the size of the resources when there are multiple
  // resources to be fetched in a ResourceLoadingBatch.
  static const size_t kMultipleResourcesSizeLimit = 100 * 1024 * 1024;

  ResourceLoadingBatch() : mResources(), mDstsAndSizes(), mSize(0) {}

  // Accessors.
  const std::vector<Resource>& resources() const { return mResources; }
  const std::vector<std::pair<uint8_t*, size_t> >& dstsAndSizes() const {
    return mDstsAndSizes;
  }
  size_t size() const { return mSize; }

  // Clear resets the ResourceLoadingBatch.
  void clear() {
    mResources.clear();
    mDstsAndSizes.clear();
    mSize = 0;
  }

  // Appends a resource to be fetched later with its loading destination. It is
  // fine to append resources with nullptr dst if the resources are to be
  // fetched only, never loaded.
  bool append(const Resource& res, uint8_t* dst);

 private:
  std::vector<Resource> mResources;
  std::vector<std::pair<uint8_t*, size_t> > mDstsAndSizes;
  size_t mSize;
};

}  // namespace gapir

#endif  // GAPIR_RESOURCE_H
