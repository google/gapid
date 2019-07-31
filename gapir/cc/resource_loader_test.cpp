/*
 * Copyright (C) 2018 Google Inc.
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

#include "resource_loader.h"
#include "mock_replay_service.h"
#include "replay_service.h"
#include "test_utilities.h"

#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include <memory>
#include <string>
#include <vector>

#include "gapir/replay_service/service.pb.h"

using namespace ::testing;

namespace gapir {
namespace test {
namespace {

const Resource A("A", 3);
const Resource B("B", 5);

class PassThroughResourceLoaderTest : public Test {
 protected:
  virtual void SetUp() {
    mSrv.reset(new MockReplayService());
    mResourceLoader = PassThroughResourceLoader::create(mSrv.get());
  }

  std::unique_ptr<MockReplayService> mSrv;
  std::unique_ptr<ResourceLoader> mResourceLoader;
  std::vector<uint8_t> mBuffer;
};

}  // anonymous namespace

TEST_F(PassThroughResourceLoaderTest, SingleGet) {
  std::vector<uint8_t> payload = {'X', 'Y', 'Z'};
  mBuffer.resize(payload.size());

  auto resData = createResources(payload);
  EXPECT_CALL(*mSrv, getResources(Pointee(Eq(A)), 1))
      .WillOnce(Invoke([&resData](const Resource* res, size_t resCount)
                           -> std::unique_ptr<ReplayService::Resources> {
        EXPECT_EQ(resCount, 1);
        EXPECT_EQ(res->getID(), A.getID());
        return std::move(resData);
      }));
  EXPECT_TRUE(mResourceLoader->load(&A, 1, mBuffer.data(), 3));
  EXPECT_THAT(mBuffer, ElementsAreArray(payload));
}

TEST_F(PassThroughResourceLoaderTest, MultiGet) {
  std::vector<uint8_t> payload = {'X', 'Y', 'Z', '1', '2', '3', '4', '5'};
  mBuffer.resize(payload.size());

  Resource resReq[] = {A, B};
  auto resData = createResources(payload);
  EXPECT_CALL(*mSrv, getResources(resReq, 2))
      .WillOnce(Invoke([&resData](const Resource* res, size_t resCount)
                           -> std::unique_ptr<ReplayService::Resources> {
        EXPECT_EQ(resCount, 2);
        EXPECT_EQ(res[0].getID(), A.getID());
        EXPECT_EQ(res[1].getID(), B.getID());
        return std::move(resData);
      }));

  EXPECT_TRUE(mResourceLoader->load(resReq, 2, mBuffer.data(), mBuffer.size()));
  EXPECT_THAT(mBuffer, ElementsAreArray(payload));
}
}  // namespace test
}  // namespace gapir
