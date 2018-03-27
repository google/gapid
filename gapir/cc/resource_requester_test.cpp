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

const Resource A("A", 3);
const Resource B("B", 5);

class ResourceRequesterTest : public Test {
protected:
    virtual void SetUp() {
        mConnection = new core::test::MockConnection();
        mServer = createServerConnection(mConnection, "", 0);
        mResourceProvider = ResourceRequester::create();
    }

    core::test::MockConnection* mConnection;
    std::unique_ptr<ServerConnection> mServer;
    std::unique_ptr<ResourceProvider> mResourceProvider;
    std::vector<uint8_t> mBuffer;
};

}  // anonymous namespace

TEST_F(ResourceRequesterTest, SingleGet) {
    std::vector<uint8_t> payload = {'X', 'Y', 'Z'};
    mBuffer.resize(payload.size());
    std::vector<uint8_t> expected;
    pushUint8(&expected, ServerConnection::MESSAGE_TYPE_GET);
    pushUint32(&expected, 1);
    pushUint32(&expected, 3);
    pushUint32(&expected, 0);
    pushString(&expected, "A");

    pushBytes(&mConnection->in, payload);

    EXPECT_TRUE(mResourceProvider->get(&A, 1, *mServer, mBuffer.data(), 3));
    EXPECT_THAT(mBuffer, ElementsAreArray(payload));
    EXPECT_EQ(mConnection->out, expected);
}

TEST_F(ResourceRequesterTest, MultiGet) {
    std::vector<uint8_t> payload = {'X', 'Y', 'Z', '1', '2', '3', '4', '5'};
    mBuffer.resize(payload.size());
    std::vector<uint8_t> expected;
    pushUint8(&expected, ServerConnection::MESSAGE_TYPE_GET);
    pushUint32(&expected, 2);
    pushUint32(&expected, 8);
    pushUint32(&expected, 0);
    pushString(&expected, "A");
    pushString(&expected, "B");

    pushBytes(&mConnection->in, payload);

    Resource res[] = {A, B};
    EXPECT_TRUE(mResourceProvider->get(res, 2, *mServer, mBuffer.data(), mBuffer.size()));
    EXPECT_THAT(mBuffer, ElementsAreArray(payload));
    EXPECT_EQ(mConnection->out, expected);
}
}  // namespace gapir
}  // namespace test
