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

#include "replay_request.h"
#include "memory_manager.h"
#include "mock_replay_service.h"
#include "mock_resource_loader.h"
#include "test_utilities.h"

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
  std::vector<uint8_t> constantMemory = {'A', 'B', 'C', 'D',
                                         'E', 'F', 'G', 'H'};
  std::vector<Resource> resources{{"ZYX", 16}, {"1234", 32}};
  std::vector<uint32_t> instructionList{0, 1, 2};

  auto payload = createPayload(stackSize, volatileMemorySize, constantMemory,
                               resources, instructionList);

  auto mock_srv = std::unique_ptr<MockReplayService>(new MockReplayService());

  EXPECT_CALL(*mock_srv, getPayload("payload"))
      .WillOnce(Return(ByMove(std::move(payload))));

  std::shared_ptr<MemoryAllocator> mMemoryAllocator(
      new MemoryAllocator(MEMORY_SIZE));

  std::unique_ptr<MemoryManager> memoryManager(
      new MemoryManager(mMemoryAllocator));

  auto replayRequest =
      ReplayRequest::create(mock_srv.get(), "payload", memoryManager.get());

  EXPECT_THAT(replayRequest, NotNull());

  EXPECT_EQ(stackSize, replayRequest->getStackSize());
  EXPECT_EQ(volatileMemorySize, replayRequest->getVolatileMemorySize());
  EXPECT_EQ(resources, replayRequest->getResources());
  EXPECT_THAT(
      constantMemory,
      ElementsAreArray((uint8_t*)(replayRequest->getConstantMemory().first),
                       replayRequest->getConstantMemory().second));
  EXPECT_THAT(instructionList,
              ElementsAreArray(replayRequest->getInstructionList().first,
                               replayRequest->getInstructionList().second));
}

TEST(ReplayRequestTestStatic, CreateErrorGet) {
  auto mock_srv = std::unique_ptr<MockReplayService>(new MockReplayService());
  EXPECT_CALL(*mock_srv, getPayload("payload"))
      .WillOnce(Return(ByMove(nullptr)));

  std::shared_ptr<MemoryAllocator> mMemoryAllocator(
      new MemoryAllocator(MEMORY_SIZE));
  std::unique_ptr<MemoryManager> memoryManager(
      new MemoryManager(mMemoryAllocator));

  auto replayRequest =
      ReplayRequest::create(mock_srv.get(), "payload", memoryManager.get());

  EXPECT_EQ(nullptr, replayRequest);
}

}  // namespace test
}  // namespace gapir
