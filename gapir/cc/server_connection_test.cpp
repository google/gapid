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

#include "server_connection.h"
#include "test_utilities.h"

#include "core/cc/mock_connection.h"

#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include <memory>
#include <string>
#include <vector>

using namespace ::testing;

namespace gapir {
namespace test {
namespace {

const ResourceId A("A");
const ResourceId B("B");
const ResourceId AB[] = {A, B};

const std::string replayId = "ABCDE";

class ServerConnectionTest : public ::testing::Test {
 protected:
  virtual void SetUp() {
    mConnection = new core::test::MockConnection();
    mServerConnection = createServerConnection(mConnection, replayId, 0);
  }

  core::test::MockConnection* mConnection;
  std::unique_ptr<ServerConnection> mServerConnection;
  std::vector<uint8_t> mBuffer;
};
}  // anonymous namespace

TEST(ServerConnectionTestStatic, Create) {
  auto connection = new core::test::MockConnection();
  uint32_t replayLength = 0x56003412;
  pushString(&connection->in, replayId);
  pushUint32(&connection->in, replayLength);

  auto svrConnection =
      ServerConnection::create(std::unique_ptr<core::Connection>(connection));

  EXPECT_THAT(svrConnection, NotNull());
  EXPECT_EQ(replayId, svrConnection->replayId());
  EXPECT_EQ(replayLength, svrConnection->replayLength());
}

TEST(ServerConnectionTestStatic, CreateErrorReadReplayId) {
  auto connection = new core::test::MockConnection();
  pushUint8(&connection->in, 'A');
  // Replay id read failed
  auto svrConnection =
      ServerConnection::create(std::unique_ptr<core::Connection>(connection));

  EXPECT_THAT(svrConnection, IsNull());
}

TEST_F(ServerConnectionTest, Get) {
  std::vector<uint8_t> resourceContent{1, 2, 3};
  mBuffer.resize(resourceContent.size());

  std::vector<uint8_t> expected;
  pushUint8(&expected, ServerConnection::MESSAGE_TYPE_GET);
  pushUint32(&expected, 2);
  pushUint32(&expected, 3);
  pushUint32(&expected, 0);
  pushString(&expected, "A");
  pushString(&expected, "B");

  pushBytes(&mConnection->in, resourceContent);

  EXPECT_TRUE(
      mServerConnection->getResources(AB, 2, mBuffer.data(), mBuffer.size()));
  EXPECT_THAT(mBuffer, ElementsAreArray(resourceContent));
  EXPECT_EQ(mConnection->out, expected);
}

TEST_F(ServerConnectionTest, GetErrorMessageType) {
  mBuffer.resize(3);
  mConnection->out_limit = 0;
  EXPECT_FALSE(
      mServerConnection->getResources(AB, 2, mBuffer.data(), mBuffer.size()));
}

TEST_F(ServerConnectionTest, GetErrorCount) {
  mBuffer.resize(3);
  mConnection->out_limit = 1;
  EXPECT_FALSE(
      mServerConnection->getResources(AB, 2, mBuffer.data(), mBuffer.size()));
}

TEST_F(ServerConnectionTest, GetErrorId) {
  mBuffer.resize(3);
  mConnection->out_limit = 3;
  EXPECT_FALSE(
      mServerConnection->getResources(AB, 2, mBuffer.data(), mBuffer.size()));
}

TEST_F(ServerConnectionTest, GetErrorContent) {
  mBuffer.resize(3);
  mConnection->out_limit = 6;
  EXPECT_FALSE(
      mServerConnection->getResources(AB, 2, mBuffer.data(), mBuffer.size()));
}

TEST_F(ServerConnectionTest, Post) {
  std::vector<uint8_t> postData{1, 2, 3};

  std::vector<uint8_t> expected;
  pushUint8(&expected, ServerConnection::MESSAGE_TYPE_POST);
  pushUint32(&expected, 3);
  pushBytes(&expected, postData);

  EXPECT_TRUE(mServerConnection->post(&postData.front(), postData.size()));
  EXPECT_EQ(mConnection->out, expected);
}

TEST_F(ServerConnectionTest, PostErrorMessageType) {
  std::vector<uint8_t> postData{1, 2, 3};
  mConnection->out_limit = 0;
  EXPECT_FALSE(mServerConnection->post(&postData.front(), postData.size()));
}

TEST_F(ServerConnectionTest, PostErrorSize) {
  std::vector<uint8_t> postData{1, 2, 3};
  mConnection->out_limit = 1;
  EXPECT_FALSE(mServerConnection->post(&postData.front(), postData.size()));
}

TEST_F(ServerConnectionTest, PostErrorData) {
  std::vector<uint8_t> postData{1, 2, 3};
  mConnection->out_limit = 4;
  EXPECT_FALSE(mServerConnection->post(&postData.front(), postData.size()));
}

}  // namespace test
}  // namespace gapir
