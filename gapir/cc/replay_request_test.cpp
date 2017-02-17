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

#include "memory_manager.h"
#include "mock_resource_provider.h"
#include "replay_request.h"
#include "server_connection.h"
#include "server_listener.h"
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

const uint32_t MEMORY_SIZE = 4096;
const std::string replayId = "ABCDE";

}  // anonymous namespace

TEST(ReplayRequestTestStatic, Create) {
    uint32_t stackSize = 128;
    uint32_t volatileMemorySize = 1024;
    std::vector<uint8_t> constantMemory =
        {'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H'};
    std::vector<Resource> resources{{"ZYX", 16}, {"1234", 32}};
    std::vector<uint32_t> instructionList{0, 1, 2};

    auto replayData = createReplayData(stackSize, volatileMemorySize, constantMemory, resources,
                                       instructionList);

    auto connection = new core::test::MockConnection();
    std::unique_ptr<StrictMock<MockResourceProvider>> resourceProvider(
            new StrictMock<MockResourceProvider>());
    pushString(&connection->in, replayId);
    pushUint32(&connection->in, replayData.size());

    Resource res(replayId, replayData.size());
    EXPECT_CALL(*resourceProvider, get(_, _, _, _, replayData.size()))
            .With(Args<0, 1>(ElementsAre(res)))
            .WillOnce(DoAll(WithArg<3>(SetVoidPointee(replayData)), Return(true)));

    std::vector<uint32_t> memorySizes = {MEMORY_SIZE};
    std::unique_ptr<MemoryManager> memoryManager(new MemoryManager(memorySizes));

    auto gazerConnection = ServerConnection::create(std::unique_ptr<core::Connection>(connection));
    auto replayRequest =
            ReplayRequest::create(*gazerConnection, resourceProvider.get(), memoryManager.get());

    EXPECT_THAT(gazerConnection, NotNull());
    EXPECT_THAT(replayRequest, NotNull());

    EXPECT_EQ(stackSize, replayRequest->getStackSize());
    EXPECT_EQ(volatileMemorySize, replayRequest->getVolatileMemorySize());
    EXPECT_EQ(resources, replayRequest->getResources());
    EXPECT_THAT(constantMemory,
        ElementsAreArray((uint8_t*)(replayRequest->getConstantMemory().first), replayRequest->getConstantMemory().second));
    EXPECT_THAT(instructionList,
        ElementsAreArray(replayRequest->getInstructionList().first, replayRequest->getInstructionList().second));
}

TEST(ReplayRequestTestStatic, CreateErrorGet) {
    uint32_t replayLength = 255;
    auto connection = new core::test::MockConnection();
    std::unique_ptr<StrictMock<MockResourceProvider>> resourceProvider(
            new StrictMock<MockResourceProvider>());

    pushString(&connection->in, replayId);
    pushUint32(&connection->in, replayLength);

    // Get replay request from resource provider fail
    Resource res(replayId, replayLength);
    EXPECT_CALL(*resourceProvider, get(_, _, _, _, replayLength))
        .With(Args<0, 1>(ElementsAre(res)))
        .WillOnce(Return(0));

    std::vector<uint32_t> memorySizes = {MEMORY_SIZE};
    std::unique_ptr<MemoryManager> memoryManager(new MemoryManager(memorySizes));

    auto gazerConnection = ServerConnection::create(std::unique_ptr<core::Connection>(connection));
    auto replayRequest =
            ReplayRequest::create(*gazerConnection, resourceProvider.get(), memoryManager.get());

    EXPECT_THAT(gazerConnection, NotNull());
    EXPECT_EQ(nullptr, replayRequest);
}

}  // namespace test
}  // namespace gapir
