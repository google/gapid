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

#include "resource.h"

namespace gapir {
bool ResourceLoadingBatch::append(const Resource& res, uint8_t* dst) {
  size_t n = mDstsAndSizes.size();

  // Always includes the first resource.
  if (n == 0) {
    mResources.push_back(res);
    mDstsAndSizes.emplace_back(std::make_pair(dst, res.getSize()));
    mSize += res.getSize();
    return true;
  }

  // Returns false if exceeds the maximum size.
  if (mSize + res.getSize() > kMultipleResourcesSizeLimit) {
    return false;
  }

  // If the resource destination is contiguous to the last one, expend the
  // last chunk.
  if (dst == mDstsAndSizes[n - 1].first + mDstsAndSizes[n - 1].second) {
    mResources.push_back(res);
    mDstsAndSizes[n - 1].second += res.getSize();
    mSize += res.getSize();
    return true;
  }

  // The resource destination is not contigous to the last one, create
  // new chunk.
  mResources.push_back(res);
  mDstsAndSizes.emplace_back(std::make_pair(dst, res.getSize()));
  mSize += res.getSize();

  return true;
}

}  // namespace gapir
