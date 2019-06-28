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

#include "post_buffer.h"

#include <cstring>
#include <new>

#include "replay_service.h"

namespace gapir {

PostBuffer::PostBuffer(uint32_t desiredCapacity, PostBufferCallback callback)
    : mPosts(ReplayService::Posts::create()),
      mTotalPostCount(0),
      mCapacity(desiredCapacity),
      mCallback(callback),
      mOffset(0) {}

PostBuffer::~PostBuffer() { flush(); }

bool PostBuffer::push(const void* address, uint32_t count) {
  if (mOffset == 0 && (count > mCapacity / 2)) {
    // Large push into an empty buffer.
    // Write it out immediately instead of buffering to reduce time spent
    // copying large buffers around. This also handles the case where the count
    // is larger than the buffer capacity.
    auto onePost = ReplayService::Posts::create();
    onePost->append(mTotalPostCount, address, count);
    mTotalPostCount++;
    return mCallback(std::move(onePost));
  }

  if (mOffset + count <= mCapacity) {
    // Fits in the buffer. Copy.
    mPosts->append(mTotalPostCount, address, count);
    mTotalPostCount++;
    mOffset += count;
    return true;
  } else {
    // Not enough capacity to fit push data. Flush and try again.
    return flush() && push(address, count);
  }
}

bool PostBuffer::flush() {
  bool ok = true;

  if (mOffset > 0) {
    ok = mCallback(std::move(mPosts));
    mPosts = ReplayService::Posts::create();
    mOffset = 0;
  }
  return ok;
}

}  // namespace gapir
