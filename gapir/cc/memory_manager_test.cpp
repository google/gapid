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

#include <gtest/gtest.h>

#include <memory>
#include <vector>

namespace gapir {
namespace test {
namespace {

const uint32_t MEMORY_SIZE = 4096;

class MemoryManagerTest : public ::testing::Test {
 protected:
  virtual void SetUp() {
    mMemoryAllocator =
        std::shared_ptr<MemoryAllocator>(new MemoryAllocator(MEMORY_SIZE));
    mMemoryManager.reset(new MemoryManager(mMemoryAllocator));
  }

  std::shared_ptr<MemoryAllocator> mMemoryAllocator;
  std::unique_ptr<MemoryManager> mMemoryManager;
};
}  // anonymous namespace

TEST_F(MemoryManagerTest, ConstantMemoryIsEmptyByDefault) {
  EXPECT_EQ(0, mMemoryManager->getConstantSize());
}

TEST_F(MemoryManagerTest, ConstantSizeIsCorrect) {
  // 64 bytes of opcodes, and 192 bytes of constants
  mMemoryManager->setReplayData(nullptr, 192, nullptr, 64);
  EXPECT_EQ(192, mMemoryManager->getConstantSize());
  EXPECT_EQ(64, mMemoryManager->getOpcodeSize());
}

TEST_F(MemoryManagerTest, ExplicitVolatileSizeIsUpdated) {
  const uint32_t volatileMemorySize = MEMORY_SIZE / 2;
  mMemoryManager->setVolatileMemory(volatileMemorySize);

  EXPECT_NE(MEMORY_SIZE, mMemoryManager->getVolatileSize());
  EXPECT_EQ(volatileMemorySize, mMemoryManager->getVolatileSize());
}

TEST_F(MemoryManagerTest, OutOfBoundsVolatileSizeFails) {
  EXPECT_TRUE(mMemoryManager->setVolatileMemory(MEMORY_SIZE));
  EXPECT_FALSE(mMemoryManager->setVolatileMemory(MEMORY_SIZE + 1));
  EXPECT_TRUE(mMemoryManager->setVolatileMemory(MEMORY_SIZE));
}

TEST_F(MemoryManagerTest, IsConstantAddressWorks) {
  const uint32_t constantMemorySize = 1024;
  uint8_t const_memory[constantMemorySize] = {};

  std::vector<uint8_t> constantMemory(constantMemorySize, 0);
  mMemoryManager->setReplayData(const_memory, constantMemorySize, nullptr, 0);
  uint8_t* constantBase = const_cast<uint8_t*>(
      static_cast<const uint8_t*>(mMemoryManager->getConstantAddress()));
  memcpy(constantBase, &constantMemory.front(), constantMemory.size());

  EXPECT_TRUE(mMemoryManager->isConstantAddress(constantBase));
  EXPECT_TRUE(
      mMemoryManager->isConstantAddress(constantBase + constantMemorySize / 2));
  EXPECT_FALSE(
      mMemoryManager->isConstantAddress(constantBase + constantMemorySize));
  EXPECT_FALSE(mMemoryManager->isConstantAddress(constantBase - 1));
  EXPECT_FALSE(
      mMemoryManager->isConstantAddress(mMemoryManager->volatileToAbsolute(0)));
}

TEST_F(MemoryManagerTest, IsVolatileAddressWorks) {
  uint32_t volatileSize = 1024;

  mMemoryManager->setVolatileMemory(volatileSize);
  uint8_t* volatileBase =
      static_cast<uint8_t*>(mMemoryManager->volatileToAbsolute(0));

  EXPECT_TRUE(mMemoryManager->isVolatileAddress(volatileBase));
  EXPECT_TRUE(
      mMemoryManager->isVolatileAddress(volatileBase + volatileSize / 2));
  EXPECT_FALSE(mMemoryManager->isVolatileAddress(volatileBase + volatileSize));
  EXPECT_FALSE(mMemoryManager->isVolatileAddress(volatileBase - 1));
  EXPECT_FALSE(
      mMemoryManager->isVolatileAddress(mMemoryManager->constantToAbsolute(0)));
}

TEST_F(MemoryManagerTest, AbsoluteToVolatile) {
  mMemoryManager->setVolatileMemory(MEMORY_SIZE / 2);

  EXPECT_EQ(
      10,
      mMemoryManager->absoluteToVolatile(
          static_cast<uint8_t*>(mMemoryManager->getVolatileAddress()) + 10));
}

TEST_F(MemoryManagerTest, AbsoluteToConstant) {
  const uint32_t constantMemorySize = 1024;
  uint8_t const_memory[constantMemorySize] = {};

  std::vector<uint8_t> constantMemory(constantMemorySize, 0);
  mMemoryManager->setReplayData(const_memory, constantMemorySize, nullptr, 0);
  uint8_t* constantBase = const_cast<uint8_t*>(
      static_cast<const uint8_t*>(mMemoryManager->getConstantAddress()));

  memcpy(constantBase, &constantMemory.front(), constantMemory.size());
  EXPECT_EQ(
      10,
      mMemoryManager->absoluteToConstant(
          static_cast<const uint8_t*>(mMemoryManager->constantToAbsolute(0)) +
          10));
}

}  // namespace test
}  // namespace gapir
