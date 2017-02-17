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

#include "base_type.h"
#include "context.h"
#include "interpreter.h"
#include "memory_manager.h"
#include "mock_resource_provider.h"
#include "resource_provider.h"
#include "test_utilities.h"

#include "core/cc/mock_connection.h"

#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include <memory>
#include <vector>

using namespace ::testing;

namespace gapir {
namespace test {
namespace {

const uint32_t MEMORY_SIZE = 4096;
const ResourceId REPLAY_ID = "replay-id";
const Resource A("A", 4);

class ContextTest : public ::testing::Test {
protected:
    virtual void SetUp() {
        std::vector<uint32_t> memorySizes = {MEMORY_SIZE};
        mMemoryManager.reset(new MemoryManager(memorySizes));
        mResourceProvider.reset(new StrictMock<MockResourceProvider>());
    }

    std::unique_ptr<MemoryManager> mMemoryManager;
    std::unique_ptr<StrictMock<MockResourceProvider>> mResourceProvider;
};

// expectLoadReplay verifies that the a request is made for the replay with id REPLAY_ID.
void expectLoadReplay(MockResourceProvider* resourceProvider,
                      const std::vector<uint8_t>& replayData) {
    Resource res(REPLAY_ID, replayData.size());
    EXPECT_CALL(*resourceProvider, get(_, _, _, _, replayData.size()))
            .With(Args<0, 1>(ElementsAre(res)))
            .WillOnce(DoAll(WithArg<3>(SetVoidPointee(replayData)), Return(true)));
}

}  // anonymous namespace

TEST_F(ContextTest, Create) {
    auto replayData = createReplayData(0, 0, {}, {}, {});

    expectLoadReplay(mResourceProvider.get(), replayData);
    auto connection = createServerConnection(REPLAY_ID, replayData.size());
    auto context = Context::create(*connection, mResourceProvider.get(), mMemoryManager.get());

    EXPECT_THAT(context, NotNull());
}

TEST_F(ContextTest, CreateErrorReplayRequest) {
    // Failed to load
    Resource res(REPLAY_ID, 10);
    EXPECT_CALL(*mResourceProvider, get(_, _, _, _, 10))
            .With(Args<0, 1>(ElementsAre(res)))
            .WillOnce(Return(false));

    auto server = createServerConnection(REPLAY_ID, 10);
    auto context = Context::create(*server, mResourceProvider.get(), mMemoryManager.get());

    EXPECT_THAT(context, IsNull());
}

TEST_F(ContextTest, CreateErrorVolatileMemory) {
    auto replayData = createReplayData(0, MEMORY_SIZE, {}, {}, {});

    Resource res(REPLAY_ID, replayData.size());
    EXPECT_CALL(*mResourceProvider, get(_, _, _, _, replayData.size()))
            .With(Args<0, 1>(ElementsAre(res)))
            .WillOnce(DoAll(WithArg<3>(SetVoidPointee(replayData)), Return(true)));

    auto server = createServerConnection(REPLAY_ID, replayData.size());
    auto context = Context::create(*server, mResourceProvider.get(), mMemoryManager.get());

    EXPECT_THAT(context, IsNull());
}

TEST_F(ContextTest, LoadResource) {
    auto replayData = createReplayData(
            128, 1024, {}, {A},
            {instruction(Interpreter::InstructionCode::PUSH_I, BaseType::VolatilePointer, 0),
             instruction(Interpreter::InstructionCode::RESOURCE, 0)});

    std::vector<uint8_t> resourceA{1, 2, 3, 4};

    expectLoadReplay(mResourceProvider.get(), replayData);

    EXPECT_CALL(*mResourceProvider, get(Pointee(Eq(A)), 1, _, _, 4))
            .WillOnce(DoAll(WithArg<3>(SetVoidPointee(resourceA)), Return(true)));

    auto server = createServerConnection(REPLAY_ID, replayData.size());
    auto context = Context::create(*server, mResourceProvider.get(), mMemoryManager.get());

    EXPECT_THAT(context, NotNull());
    EXPECT_TRUE(context->interpret());
    auto res = (uint8_t*)mMemoryManager->volatileToAbsolute(0);
    EXPECT_THAT(resourceA, ElementsAreArray(res, resourceA.size()));
}

TEST_F(ContextTest, LoadResourcePopFailed) {
    auto replayData = createReplayData(128, 1024, {}, {A},
                                       {instruction(Interpreter::InstructionCode::RESOURCE, 0)});

    expectLoadReplay(mResourceProvider.get(), replayData);
    auto server = createServerConnection(REPLAY_ID, replayData.size());
    auto context = Context::create(*server, mResourceProvider.get(), mMemoryManager.get());

    EXPECT_THAT(context, NotNull());
    EXPECT_FALSE(context->interpret());
}

TEST_F(ContextTest, LoadResourceGetFailed) {
    auto replayData = createReplayData(
            128, 1024, {}, {A},
            {instruction(Interpreter::InstructionCode::PUSH_I, BaseType::VolatilePointer, 0),
             instruction(Interpreter::InstructionCode::RESOURCE, 0)});

    expectLoadReplay(mResourceProvider.get(), replayData);

    EXPECT_CALL(*mResourceProvider, get(Pointee(Eq(A)), 1, _, _, 4)).WillOnce(Return(false));

    auto server = createServerConnection(REPLAY_ID, replayData.size());
    auto context = Context::create(*server, mResourceProvider.get(), mMemoryManager.get());

    EXPECT_THAT(context, NotNull());
    EXPECT_FALSE(context->interpret());
}

TEST_F(ContextTest, PostData) {
    auto replayData = createReplayData(
            128, 1024, {0, 1, 2, 3, 4, 5, 6, 7}, {},
            {instruction(Interpreter::InstructionCode::PUSH_I, BaseType::ConstantPointer, 1),
             instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint32, 6),
             instruction(Interpreter::InstructionCode::POST)});
    std::vector<uint8_t> expected;
    pushUint8(&expected, ServerConnection::MESSAGE_TYPE_POST);
    pushUint32(&expected, 6);
    pushBytes(&expected, {1, 2, 3, 4, 5, 6});

    auto connection = new core::test::MockConnection();
    expectLoadReplay(mResourceProvider.get(), replayData);
    auto server = createServerConnection(connection, REPLAY_ID, replayData.size());
    auto context = Context::create(*server, mResourceProvider.get(), mMemoryManager.get());
    EXPECT_THAT(context, NotNull());
    EXPECT_TRUE(context->interpret());
    EXPECT_EQ(connection->out, expected);
}

TEST_F(ContextTest, PostDataErrorPop) {
    auto replayData = createReplayData(
            128, 1024, {0, 1, 2, 3, 4, 5, 6, 7}, {},
            {instruction(Interpreter::InstructionCode::PUSH_I, BaseType::ConstantPointer, 1),
             instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint8, 6),  // Wrong type
             instruction(Interpreter::InstructionCode::POST)});

    expectLoadReplay(mResourceProvider.get(), replayData);
    auto server = createServerConnection(REPLAY_ID, replayData.size());
    auto context = Context::create(*server, mResourceProvider.get(), mMemoryManager.get());

    EXPECT_THAT(context, NotNull());
    EXPECT_FALSE(context->interpret());
}

TEST_F(ContextTest, PostDataErrorPost) {
    auto replayData = createReplayData(
            128, 1024, {0, 1, 2, 3, 4, 5, 6, 7}, {},
            {instruction(Interpreter::InstructionCode::PUSH_I, BaseType::ConstantPointer, 1),
             instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint32, 6),
             instruction(Interpreter::InstructionCode::POST)});

    auto connection = new core::test::MockConnection();
    connection->out_limit = 7;
    expectLoadReplay(mResourceProvider.get(), replayData);
    auto server = createServerConnection(connection, REPLAY_ID, replayData.size());
    auto context = Context::create(*server, mResourceProvider.get(), mMemoryManager.get());
    EXPECT_THAT(context, NotNull());
    EXPECT_FALSE(context->interpret());
}

}  // namespace test
}  // namespace gapir
