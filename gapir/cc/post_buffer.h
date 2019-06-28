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

#ifndef GAPIR_POSTBUFFER_H
#define GAPIR_POSTBUFFER_H

#include <cstdint>
#include <functional>
#include <memory>

#include "replay_service.h"

namespace gapir {

// PostBuffer provides a delayed-processed buffer for tasks like pushing data to
// the server, etc. This serves as an optimisation to batch many small postbacks
// into fewer, larger batches.
class PostBuffer {
 public:
  typedef std::function<bool(std::unique_ptr<ReplayService::Posts>)>
      PostBufferCallback;

  // Constructs a PostBuffer with the specified maximum capacity and function to
  // invoke when the PostBuffer wants to flush the buffer to the server.
  PostBuffer(uint32_t desiredCapacity, PostBufferCallback callback);
  ~PostBuffer();

  // Push data to the buffer. If the buffer does not have enough space to buffer
  // the data, then the contents of the PushBuffer will be flushed.
  bool push(const void* address, uint32_t count);

  // Forcefully flush the PostBuffer. If the PostBuffer is empty then calling
  // this function does nothing.
  bool flush();

  void resetCount() { mTotalPostCount = 0; }

 private:
  // The PostBuffer's internal buffer.
  std::unique_ptr<ReplayService::Posts> mPosts;

  // The total number of posts ever processed by this post buffer.
  uint64_t mTotalPostCount;

  // The maximum capacity of the internal buffer.
  const uint32_t mCapacity;

  // The flush callback function.
  const PostBufferCallback mCallback;

  // The offset in mBuffer for the next write.
  uint32_t mOffset;
};

}  // namespace gapir

#endif  // GAPIR_POSTBUFFER_H
