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

#include "context.h"
#include "base_type.h"
#include "interpreter.h"
#include "memory_manager.h"
#include "mock_replay_service.h"
#include "mock_resource_loader.h"
#include "resource_loader.h"
#include "test_utilities.h"

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
    mMemoryAllocator =
        std::shared_ptr<MemoryAllocator>(new MemoryAllocator(MEMORY_SIZE));
    mMemoryManager.reset(new MemoryManager(mMemoryAllocator));
    mResourceLoader.reset(new StrictMock<MockResourceLoader>());
    mSrv.reset(new MockReplayService());
  }

  std::shared_ptr<MemoryAllocator> mMemoryAllocator;
  std::unique_ptr<MemoryManager> mMemoryManager;
  std::unique_ptr<StrictMock<MockResourceLoader>> mResourceLoader;
  std::unique_ptr<MockReplayService> mSrv;
};
}  // anonymous namespace

TEST_F(ContextTest, Create) {
  auto payload = createPayload(0, 0, {}, {}, {});

  EXPECT_CALL(*mSrv, getPayload("payload"))
      .WillOnce(Return(ByMove(std::move(payload))));
  core::CrashHandler crash_handler;
  auto context = Context::create(mSrv.get(), crash_handler,
                                 mResourceLoader.get(), mMemoryManager.get());
  EXPECT_THAT(context, NotNull());
  EXPECT_TRUE(context->initialize("payload"));
}

TEST_F(ContextTest, CreateErrorReplayRequest) {
  // Failed to load
  EXPECT_CALL(*mSrv, getPayload("payload"))
      .WillOnce(
          Return(ByMove(std::unique_ptr<ReplayService::Payload>(nullptr))));

  core::CrashHandler crash_handler;
  auto context = Context::create(mSrv.get(), crash_handler,
                                 mResourceLoader.get(), mMemoryManager.get());

  EXPECT_FALSE(context->initialize("payload"));
}

TEST_F(ContextTest, CreateErrorVolatileMemory) {
  auto payload = createPayload(0, MEMORY_SIZE + 1, {}, {}, {});

  EXPECT_CALL(*mSrv, getPayload("payload"))
      .WillOnce(Return(ByMove(std::move(payload))));
  core::CrashHandler crash_handler;
  auto context = Context::create(mSrv.get(), crash_handler,
                                 mResourceLoader.get(), mMemoryManager.get());
  EXPECT_THAT(context, NotNull());
  EXPECT_FALSE(context->initialize("payload"));
}

TEST_F(ContextTest, LoadResource) {
  auto payload =
      createPayload(128, 1024, {}, {A},
                    {instruction(Interpreter::InstructionCode::PUSH_I,
                                 BaseType::VolatilePointer, 0),
                     instruction(Interpreter::InstructionCode::RESOURCE, 0)});
  std::vector<uint8_t> resourceA{1, 2, 3, 4};

  EXPECT_CALL(*mSrv, getPayload("payload"))
      .WillOnce(Return(ByMove(std::move(payload))));
  EXPECT_CALL(*mResourceLoader, load(Pointee(Eq(A)), 1, _, 4))
      .WillOnce(DoAll(WithArg<2>(SetVoidPointee(resourceA)), Return(true)));

  core::CrashHandler crash_handler;
  auto context = Context::create(mSrv.get(), crash_handler,
                                 mResourceLoader.get(), mMemoryManager.get());

  EXPECT_THAT(context, NotNull());

  context->initialize("payload");
  EXPECT_TRUE(context->interpret());
  auto res = (uint8_t*)mMemoryManager->volatileToAbsolute(0);
  EXPECT_THAT(resourceA, ElementsAreArray(res, resourceA.size()));
}

TEST_F(ContextTest, LoadResourcePopFailed) {
  auto payload =
      createPayload(128, 1024, {}, {A},
                    {instruction(Interpreter::InstructionCode::RESOURCE, 0)});

  EXPECT_CALL(*mSrv, getPayload("payload"))
      .WillOnce(Return(ByMove(std::move(payload))));
  core::CrashHandler crash_handler;
  auto context = Context::create(mSrv.get(), crash_handler,
                                 mResourceLoader.get(), mMemoryManager.get());

  EXPECT_THAT(context, NotNull());
  context->initialize("payload");
  EXPECT_FALSE(context->interpret());
}

TEST_F(ContextTest, LoadResourceGetFailed) {
  auto payload =
      createPayload(128, 1024, {}, {A},
                    {instruction(Interpreter::InstructionCode::PUSH_I,
                                 BaseType::VolatilePointer, 0),
                     instruction(Interpreter::InstructionCode::RESOURCE, 0)});

  EXPECT_CALL(*mSrv, getPayload("payload"))
      .WillOnce(Return(ByMove(std::move(payload))));
  EXPECT_CALL(*mResourceLoader, load(Pointee(Eq(A)), 1, _, 4))
      .WillOnce(Return(false));

  core::CrashHandler crash_handler;
  auto context = Context::create(mSrv.get(), crash_handler,
                                 mResourceLoader.get(), mMemoryManager.get());
  EXPECT_THAT(context, NotNull());
  context->initialize("payload");
  EXPECT_FALSE(context->interpret());
}

TEST_F(ContextTest, PostData) {
  auto payload = createPayload(
      128, 1024, {0, 1, 2, 3, 4, 5, 6, 7}, {},
      {instruction(Interpreter::InstructionCode::PUSH_I,
                   BaseType::ConstantPointer, 1),
       instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint32, 6),
       instruction(Interpreter::InstructionCode::POST)});
  std::vector<uint8_t> expected;
  pushBytes(&expected, {1, 2, 3, 4, 5, 6});
  std::vector<uint8_t> actual;

  EXPECT_CALL(*mSrv, getPayload("payload"))
      .WillOnce(Return(ByMove(std::move(payload))));
  EXPECT_CALL(*mSrv, mockedSendPosts(NotNull()))
      .WillOnce(Invoke([&actual](ReplayService::Posts* posts) -> bool {
        for (size_t i = 0; i < posts->piece_count(); i++) {
          actual.resize(actual.size() + posts->piece_size(i));
          memcpy(&actual[actual.size() - posts->piece_size(i)],
                 posts->piece_data(i), posts->piece_size(i));
        }
        return true;
      }));

  core::CrashHandler crash_handler;
  auto context = Context::create(mSrv.get(), crash_handler,
                                 mResourceLoader.get(), mMemoryManager.get());
  EXPECT_THAT(context, NotNull());
  context->initialize("payload");
  EXPECT_TRUE(context->interpret());
  EXPECT_THAT(actual, ContainerEq(expected));
}

TEST_F(ContextTest, PostDataErrorPop) {
  auto payload =
      createPayload(128, 1024, {0, 1, 2, 3, 4, 5, 6, 7}, {},
                    {instruction(Interpreter::InstructionCode::PUSH_I,
                                 BaseType::ConstantPointer, 1),
                     instruction(Interpreter::InstructionCode::PUSH_I,
                                 BaseType::Uint8, 6),  // Wrong type
                     instruction(Interpreter::InstructionCode::POST)});

  EXPECT_CALL(*mSrv, getPayload("payload"))
      .WillOnce(Return(ByMove(std::move(payload))));
  core::CrashHandler crash_handler;
  auto context = Context::create(mSrv.get(), crash_handler,
                                 mResourceLoader.get(), mMemoryManager.get());

  EXPECT_THAT(context, NotNull());
  context->initialize("payload");
  EXPECT_FALSE(context->interpret());
}

TEST_F(ContextTest, PostDataErrorPost) {
  auto payload = createPayload(
      128, 1024, {0, 1, 2, 3, 4, 5, 6, 7}, {},
      {instruction(Interpreter::InstructionCode::PUSH_I,
                   BaseType::ConstantPointer, 1),
       instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint32, 6),
       instruction(Interpreter::InstructionCode::POST)});

  EXPECT_CALL(*mSrv, getPayload("payload"))
      .WillOnce(Return(ByMove(std::move(payload))));
  EXPECT_CALL(*mSrv, mockedSendPosts(NotNull())).WillOnce(Return(false));

  core::CrashHandler crash_handler;
  auto context = Context::create(mSrv.get(), crash_handler,
                                 mResourceLoader.get(), mMemoryManager.get());
  EXPECT_THAT(context, NotNull());
  context->initialize("payload");
  EXPECT_FALSE(context->interpret());
}

TEST_F(ContextTest, Notification) {
  const uint8_t api_index = 0xAB;
  const int severity = LOG_LEVEL_ERROR;
  const std::string msg = "notification test";
  // Invoke onDebugMessage() during interpreting POST instruction.
  auto payload = createPayload(
      128, 1024, {0, 1, 2, 3, 4, 5, 6, 7}, {},
      {instruction(Interpreter::InstructionCode::PUSH_I,
                   BaseType::ConstantPointer, 1),
       instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint32, 6),
       instruction(Interpreter::InstructionCode::POST)});

  EXPECT_CALL(*mSrv, getPayload("payload"))
      .WillOnce(Return(ByMove(std::move(payload))));
  core::CrashHandler crash_handler;
  auto context = Context::create(mSrv.get(), crash_handler,
                                 mResourceLoader.get(), mMemoryManager.get());
  EXPECT_THAT(context, NotNull());
  context->initialize("payload");

  EXPECT_CALL(*mSrv, mockedSendPosts(NotNull()))
      .WillOnce(Invoke([&context, &msg](ReplayService::Posts* posts) -> bool {
        context->onDebugMessage(severity, api_index, msg.c_str());
        return true;
      }));
  EXPECT_CALL(*mSrv, sendErrorMsg(0, severity, api_index, 0, msg, IsNull(), 0))
      .WillOnce(Return(true));

  EXPECT_TRUE(context->interpret());
}

}  // namespace test
}  // namespace gapir
