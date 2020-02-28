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

#include "stack.h"
#include "memory_manager.h"
#include "test_utilities.h"

#include <gtest/gtest.h>

#include <memory>

namespace gapir {
namespace test {
namespace {

const uint32_t MEMORY_SIZE = 4096;
const uint32_t STACK_CAPACITY = 128;
const uint32_t CONSTANT_SIZE = 128;

class StackTest : public ::testing::Test {
 protected:
  virtual void SetUp() {
    mMemoryAllocator =
        std::shared_ptr<MemoryAllocator>(new MemoryAllocator(MEMORY_SIZE));
    mMemoryManager.reset(new MemoryManager(mMemoryAllocator));
    mStack.reset(new Stack(STACK_CAPACITY, mMemoryManager.get()));

    mMemoryManager->setVolatileMemory(4096);
    mMemoryManager->setReplayData(const_memory, constantMemorySize, nullptr, 0);
  }

  static const uint32_t constantMemorySize = 1024;
  uint8_t const_memory[constantMemorySize] = {};
  std::shared_ptr<MemoryAllocator> mMemoryAllocator;
  std::unique_ptr<MemoryManager> mMemoryManager;
  std::unique_ptr<Stack> mStack;
};

}  // anonymous namespace

TEST_F(StackTest, GetType) {
  mStack->push<uint32_t>(123);
  EXPECT_EQ(std::type_index(typeid(uint32_t)), mStack->typeAtTop());
  mStack->discard(1);

  mStack->push<void*>(nullptr);
  EXPECT_EQ(std::type_index(typeid(void*)), mStack->typeAtTop());
  mStack->discard(1);
}

TEST_F(StackTest, PushValue) {
  uint32_t x = 123456789;
  mStack->push(x);
  EXPECT_EQ(123456789, mStack->pop<uint32_t>());
}

TEST_F(StackTest, PopVolatilePtrWithoutConvert) {
  uint32_t offset = 0x123;
  mStack->push(Stack::VolatilePointer(offset));

  Stack::VolatilePointer pointer = mStack->pop<Stack::VolatilePointer>();
  EXPECT_EQ(offset, pointer.getOffset());
}

TEST_F(StackTest, PopVolatilePtrWithConvert) {
  uint32_t offset = 0x123;
  mStack->push(Stack::VolatilePointer(offset));

  const void* pointer = mStack->pop<const void*>();
  EXPECT_EQ(mMemoryManager->volatileToAbsolute(offset), pointer);
}

TEST_F(StackTest, PopConstantPtrWithoutConvert) {
  uint32_t offset = 0x12;
  mStack->push(Stack::ConstantPointer(offset));

  Stack::ConstantPointer pointer = mStack->pop<Stack::ConstantPointer>();
  EXPECT_EQ(offset, pointer.getOffset());
}

TEST_F(StackTest, PopConstantPtrWithConvert) {
  uint32_t offset = 0x12;
  mStack->push(Stack::ConstantPointer(offset));

  const void* pointer = mStack->pop<const void*>();
  EXPECT_EQ(mMemoryManager->constantToAbsolute(offset), pointer);
}

TEST_F(StackTest, Discard) {
  mStack->push<uint32_t>(1234);
  mStack->push<uint32_t>(2345);
  mStack->push<uint32_t>(3356);

  mStack->discard(2);
  EXPECT_EQ(1234, mStack->pop<uint32_t>());
}

TEST_F(StackTest, SequentialDiscard) {
  mStack->push<uint32_t>(12);
  mStack->push<uint32_t>(123);
  mStack->push<uint32_t>(234);
  mStack->discard(1);
  mStack->push<uint32_t>(345);
  mStack->discard(1);

  EXPECT_EQ(123, mStack->pop<uint32_t>());
}

TEST_F(StackTest, Clone) {
  mStack->push<uint32_t>(1234);
  mStack->push<uint32_t>(2345);

  mStack->clone(1);

  EXPECT_EQ(1234, mStack->pop<uint32_t>());
  EXPECT_EQ(2345, mStack->pop<uint32_t>());
  EXPECT_EQ(1234, mStack->pop<uint32_t>());
}

TEST_F(StackTest, PushPop) {
  mStack->push<uint32_t>(123);
  ASSERT_EQ(123, mStack->pop<uint32_t>());
}

TEST_F(StackTest, PopVolatilePointer) {
  uint32_t a = 0;
  mStack->push(Stack::VolatilePointer(a));
  EXPECT_EQ(mMemoryManager->volatileToAbsolute(0), mStack->pop<void*>());
}

TEST_F(StackTest, PopConstantPointer) {
  uint32_t a = 0;
  mStack->push(Stack::ConstantPointer(a));
  EXPECT_EQ(mMemoryManager->constantToAbsolute(0), mStack->pop<const void*>());
}

TEST_F(StackTest, PopAbsolutePointer) {
  mStack->push<void*>(nullptr);
  EXPECT_EQ(nullptr, mStack->pop<void*>());
}

}  // namespace test
}  // namespace gapir
