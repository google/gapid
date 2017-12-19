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
#include "server_listener.h"
#include "test_utilities.h"

#include "core/cc/mock_connection.h"

#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include <memory>

using ::testing::_;
using ::testing::DoAll;
using ::testing::IsNull;
using ::testing::NotNull;
using ::testing::Return;
using ::testing::ReturnArg;
using ::testing::StrictMock;
using ::testing::WithArg;

namespace gapir {
namespace test {
namespace {

const uint64_t MAX_MEMORY_SIZE = 1024;

class ServerListenerTest : public ::testing::Test {
protected:
    virtual void SetUp() {
      mConnection = std::unique_ptr<core::test::MockConnection>(
          new core::test::MockConnection());
      mServerListener.reset(
          new ServerListener(mConnection.get(), MAX_MEMORY_SIZE));
    }

    inline void pushValidReplayRequest(std::vector<uint8_t>* buf) {
        pushUint8(buf, ServerListener::REPLAY_REQUEST);
        pushString(buf, "");
        pushUint32(buf, 0);
    }
    std::unique_ptr<core::test::MockConnection> mConnection;
    std::unique_ptr<ServerListener> mServerListener;
};

}  // anonymous namespace

TEST_F(ServerListenerTest, AcceptConnection) {
    auto clientConnection = new core::test::MockConnection();
    mConnection->connections.push(clientConnection);
    pushValidReplayRequest(&clientConnection->in);
    EXPECT_THAT(mServerListener->acceptConnection(core::Connection::NO_TIMEOUT, nullptr), NotNull());
}

TEST_F(ServerListenerTest, AcceptConnectionErrorAccept) {
    EXPECT_THAT(mServerListener->acceptConnection(core::Connection::NO_TIMEOUT, nullptr), IsNull());
}

TEST_F(ServerListenerTest, AcceptConnectionErrorServerConnection) {
    std::string replayId = "Replay2";
    auto clientConnection1 = new core::test::MockConnection();
    auto clientConnection2 = new core::test::MockConnection();
    mConnection->connections.push(clientConnection1);
    mConnection->connections.push(clientConnection2);
    pushUint8(&clientConnection1->in, ServerListener::REPLAY_REQUEST);
    pushUint8(&clientConnection1->in, '1');
    pushUint8(&clientConnection2->in, ServerListener::REPLAY_REQUEST);
    pushString(&clientConnection2->in, replayId);
    pushUint32(&clientConnection2->in, 0);
    EXPECT_THAT(mServerListener->acceptConnection(core::Connection::NO_TIMEOUT, nullptr), NotNull());
    //TODO: check we actually got connection 2
}

TEST_F(ServerListenerTest, AcceptConnectionMissingAuthTokenHeader) {
    auto clientConnection = new core::test::MockConnection();
    mConnection->connections.push(clientConnection);
    pushValidReplayRequest(&clientConnection->in);
    EXPECT_THAT(mServerListener->acceptConnection(core::Connection::NO_TIMEOUT, "secrets"), IsNull());
}

TEST_F(ServerListenerTest, AcceptConnectionBadAuthTokenHeader) {
    auto clientConnection = new core::test::MockConnection();
    mConnection->connections.push(clientConnection);
    pushUint8(&clientConnection->in, 'B');
    pushUint8(&clientConnection->in, 'A');
    pushUint8(&clientConnection->in, 'D');
    pushUint8(&clientConnection->in, ' ');
    pushValidReplayRequest(&clientConnection->in);
    EXPECT_THAT(mServerListener->acceptConnection(core::Connection::NO_TIMEOUT, "secrets"), IsNull());
}

TEST_F(ServerListenerTest, AcceptConnectionBadAuthToken) {
    auto clientConnection = new core::test::MockConnection();
    mConnection->connections.push(clientConnection);
    pushUint8(&clientConnection->in, 'A');
    pushUint8(&clientConnection->in, 'U');
    pushUint8(&clientConnection->in, 'T');
    pushUint8(&clientConnection->in, 'H');
    pushString(&clientConnection->in, "wrong");
    pushValidReplayRequest(&clientConnection->in);
    EXPECT_THAT(mServerListener->acceptConnection(core::Connection::NO_TIMEOUT, "secrets"), IsNull());
}

TEST_F(ServerListenerTest, AcceptConnectionCorrectAuthToken) {
    auto clientConnection = new core::test::MockConnection();
    mConnection->connections.push(clientConnection);
    pushUint8(&clientConnection->in, 'A');
    pushUint8(&clientConnection->in, 'U');
    pushUint8(&clientConnection->in, 'T');
    pushUint8(&clientConnection->in, 'H');
    pushString(&clientConnection->in, "secrets");
    pushValidReplayRequest(&clientConnection->in);
    EXPECT_THAT(mServerListener->acceptConnection(core::Connection::NO_TIMEOUT, "secrets"), NotNull());
}


}  // namespace test
}  // namespace gapir
