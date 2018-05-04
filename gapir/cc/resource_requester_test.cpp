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

#include "resource_provider.h"
#include "resource_requester.h"
#include "test_utilities.h"
#include "replay_connection.h"
#include "mock_replay_connection.h"

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

class ResourceRequesterTest : public Test {
protected:
    virtual void SetUp() {
        mResourceProvider = ResourceRequester::create();
        mConn.reset(new MockReplayConnection());
    }

    std::unique_ptr<MockReplayConnection> mConn;
    std::unique_ptr<ResourceProvider> mResourceProvider;
    std::vector<uint8_t> mBuffer;
};

}  // anonymous namespace

TEST_F(ResourceRequesterTest, SingleGet) {
    std::vector<uint8_t> payload = {'X', 'Y', 'Z'};
    mBuffer.resize(payload.size());

    auto res = createResources(payload);
    EXPECT_CALL(*mConn, mockedGetResources(NotNull()))
        .WillOnce(Invoke([&res](ReplayConnection::ResourceRequest* req)
                             -> std::unique_ptr<ReplayConnection::Resources> {
          auto p = std::unique_ptr<replay_service::ResourceRequest>(
              req->release_to_proto());
          EXPECT_EQ(p->expected_total_size(), A.size);
          EXPECT_EQ(p->ids_size(), 1);
          EXPECT_EQ(p->ids(0), A.id);
          return std::move(res);
        }));
    EXPECT_TRUE(mResourceProvider->get(&A, 1, mConn.get(), mBuffer.data(), 3));
    EXPECT_THAT(mBuffer, ElementsAreArray(payload));

}

TEST_F(ResourceRequesterTest, MultiGet) {
    std::vector<uint8_t> payload = {'X', 'Y', 'Z', '1', '2', '3', '4', '5'};
    mBuffer.resize(payload.size());

    auto res = createResources(payload);
    EXPECT_CALL(*mConn, mockedGetResources(NotNull()))
        .WillOnce(Invoke([&res](ReplayConnection::ResourceRequest* req)
                             -> std::unique_ptr<ReplayConnection::Resources> {
          auto p = std::unique_ptr<replay_service::ResourceRequest>(
              req->release_to_proto());
          EXPECT_EQ(p->expected_total_size(), A.size + B.size);
          EXPECT_EQ(p->ids_size(), 2);
          EXPECT_EQ(p->ids(0), A.id);
          EXPECT_EQ(p->ids(1), B.id);
          return std::move(res);
        }));

    Resource resReq[] = {A, B};
    EXPECT_TRUE(mResourceProvider->get(resReq, 2, mConn.get(), mBuffer.data(), mBuffer.size()));
    EXPECT_THAT(mBuffer, ElementsAreArray(payload));
}
}  // namespace gapir
}  // namespace test
