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

#include <gmock/gmock.h>
#include <gtest/gtest.h>
#include <string.h>

#include <vector>

namespace gapir {
namespace test {
namespace {

const std::vector<uint8_t> input = {0, 1, 2, 3, 4, 5};

class PostBufferTest : public ::testing::Test {
 protected:
  virtual void SetUp() {
    mPostBuffer.reset();
    mOutput.clear();
    mPostsCounter = 0;
  }

  void setupPostBuffer(uint32_t bufferSize, bool callbackShouldSucceed) {
    mPostBuffer.reset(new PostBuffer(
        bufferSize, [&, callbackShouldSucceed](
                        std::unique_ptr<ReplayService::Posts> posts) {
          mPostsCounter++;
          size_t piece_count = posts->piece_count();
          for (uint32_t i = 0; i < piece_count; i++) {
            size_t piece_size = posts->piece_size(i);
            mOutput.resize(mOutput.size() + piece_size);
            memcpy(&mOutput[mOutput.size() - piece_size], posts->piece_data(i),
                   piece_size);
          }
          return callbackShouldSucceed;
        }));
  }

  std::unique_ptr<PostBuffer> mPostBuffer;
  std::vector<uint8_t> mOutput;
  uint32_t mPostsCounter;
};

}  // anonymous namespace

TEST_F(PostBufferTest, ZeroSizedBuffer) {
  // No buffering, callback always succeeds.
  setupPostBuffer(0, true);

  // Push should immediately call the post callback as there's no buffering.
  EXPECT_TRUE(mPostBuffer->push(&input.front(), input.size()));
  EXPECT_EQ(input, mOutput);
  const uint32_t postsCounterBeforeFlush = mPostsCounter;

  // Flush should be a no-op if there's no buffering.
  EXPECT_TRUE(mPostBuffer->flush());
  EXPECT_EQ(postsCounterBeforeFlush, mPostsCounter);
}

TEST_F(PostBufferTest, PushSmallPacketsThenFlush) {
  // Buffer much larger than the whole input, callback always succeeds.
  setupPostBuffer(input.size() * 4, true);

  for (size_t i = 0; i < input.size(); ++i) {
    EXPECT_TRUE(mPostBuffer->push(&input[i], 1));
  }
  EXPECT_TRUE(mPostBuffer->flush());
  EXPECT_EQ(input, mOutput);
}

TEST_F(PostBufferTest, PushLargePacketsThenFlush) {
  // Buffer size smaller than each packet pushed.
  setupPostBuffer(1, true);

  ASSERT_EQ(0, input.size() % 2);  // Make sure we're not being silly.
  for (size_t i = 0; i < input.size(); i += 2) {
    EXPECT_TRUE(mPostBuffer->push(&input[i], 2));
  }

  // Each packet larger than the buffer should trigger a separate post.
  EXPECT_EQ(input.size() / 2, mPostsCounter);
  EXPECT_EQ(input, mOutput);
  const uint32_t postsCounterBeforeFlush = mPostsCounter;

  // Flush should be a no-op as none of the packets did fit in the buffer.
  EXPECT_TRUE(mPostBuffer->flush());
  EXPECT_EQ(postsCounterBeforeFlush, mPostsCounter);
}

TEST_F(PostBufferTest, PushMixSizedPacketsThenFlush) {
  // Buffer size respectively smaller, equal and larger than pushed packets.
  setupPostBuffer(2, true);

  ASSERT_EQ(1 + 2 + 3, input.size());  // Make sure we're not being silly.
  EXPECT_TRUE(mPostBuffer->push(&input[0], 1));
  EXPECT_TRUE(mPostBuffer->push(&input[1], 2));
  EXPECT_TRUE(mPostBuffer->push(&input[3], 3));
  EXPECT_TRUE(mPostBuffer->flush());

  EXPECT_EQ(input, mOutput);
}

TEST_F(PostBufferTest, FlushOnDestruction) {
  // Buffer much larger than the whole input, callback always succeeds.
  setupPostBuffer(input.size() * 4, true);

  EXPECT_TRUE(mPostBuffer->push(&input.front(), input.size()));
  // Note: While the semantics are not explicit about it, we don't expect
  // the PostBuffer to be flushed after only 1/4 of its capacity has been
  // pushed to. If this happens to be wrong, remove the following check.
  EXPECT_EQ(0, mPostsCounter);

  // Check that the packets gets posted on PostBuffer destruction.
  mPostBuffer.reset();
  EXPECT_EQ(input, mOutput);
}

TEST_F(PostBufferTest, ReportCallbackErrors) {
  // No buffering, callback always failing.
  setupPostBuffer(0, false);

  // At least one of these commands should call and report a callback error.
  bool pushSuccess = mPostBuffer->push(&input.front(), input.size());
  bool flushSuccess = mPostBuffer->flush();
  EXPECT_LT(0, mPostsCounter);
  EXPECT_FALSE(pushSuccess && flushSuccess);
}

}  // namespace test
}  // namespace gapir
