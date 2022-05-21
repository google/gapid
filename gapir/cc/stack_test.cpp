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

void fillStack(Stack* stack, uint32_t n) {
  for (unsigned i = 0; i < n; ++i) {
    stack->push<uint32_t>(0);
  }
}
}  // anonymous namespace

TEST_F(StackTest, IsValid) { EXPECT_TRUE(mStack->isValid()); }

TEST_F(StackTest, GetType) {
  mStack->push<uint32_t>(123);
  EXPECT_EQ(BaseType::Uint32, mStack->getTopType());
  EXPECT_TRUE(mStack->isValid());
  mStack->discard(1);
  EXPECT_TRUE(mStack->isValid());

  mStack->push<void*>(nullptr);
  EXPECT_EQ(BaseType::AbsolutePointer, mStack->getTopType());
  EXPECT_TRUE(mStack->isValid());
  mStack->discard(1);
  EXPECT_TRUE(mStack->isValid());
}

TEST_F(StackTest, GetTypeErrorEmptyStack) {
  mStack->getTopType();
  EXPECT_FALSE(mStack->isValid());

  mStack->getTopType();
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, PushValue) {
  uint32_t x = 123456789;
  mStack->pushValue(BaseType::Uint32, x);
  EXPECT_EQ(123456789, mStack->pop<uint32_t>());
  EXPECT_TRUE(mStack->isValid());
}

TEST_F(StackTest, PushValueErrorStackOverflow) {
  fillStack(mStack.get(), STACK_CAPACITY);
  EXPECT_TRUE(mStack->isValid());

  uint32_t x = 123456789;
  mStack->pushValue(BaseType::Uint32, x);
  EXPECT_FALSE(mStack->isValid());

  mStack->pushValue(BaseType::Uint32, x);
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, PopVolatilePtrWithoutConvert) {
  uint32_t offset = 0x123;
  mStack->pushValue(BaseType::VolatilePointer, offset);

  uint32_t pointer = mStack->popBaseValue();
  EXPECT_TRUE(mStack->isValid());

  EXPECT_EQ(offset, pointer);
}

TEST_F(StackTest, PopVolatilePtrWithConvert) {
  uint32_t offset = 0x123;
  std::cerr << "mStack->isValid() = " << mStack->isValid() << "\n";
  mStack->pushValue(BaseType::VolatilePointer, offset);
  std::cerr << "mStack->isValid() = " << mStack->isValid() << "\n";

  const void* pointer = mStack->popVolatile<const void*>();
  std::cerr << "mStack->isValid() = " << mStack->isValid() << "\n";
  EXPECT_TRUE(mStack->isValid());

  EXPECT_EQ(mMemoryManager->volatileToAbsolute(offset), pointer);
}

TEST_F(StackTest, PopConstantPtrWithoutConvert) {
  uint32_t offset = 0x12;
  mStack->pushValue(BaseType::ConstantPointer, offset);

  uint32_t pointer = mStack->popBaseValue();
  EXPECT_TRUE(mStack->isValid());

  EXPECT_EQ(offset, pointer);
}

TEST_F(StackTest, PopConstantPtrWithConvert) {
  uint32_t offset = 0x12;
  mStack->pushValue(BaseType::ConstantPointer, offset);

  const void* pointer = mStack->popConstant<const void*>();
  EXPECT_TRUE(mStack->isValid());

  EXPECT_EQ(mMemoryManager->constantToAbsolute(offset), pointer);
}

TEST_F(StackTest, PopErrorEmptyStack) {
  mStack->pop<uint32_t>();
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, PopConstantErrorEmptyStack) {
  mStack->popConstant<const void*>();
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, PopVolatileErrorEmptyStack) {
  mStack->popVolatile<void*>();
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, PopBaseValueErrorEmptyStack) {
  mStack->popBaseValue();
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, Discard) {
  mStack->push<uint32_t>(1234);
  mStack->push<uint32_t>(2345);
  mStack->push<uint32_t>(3356);
  EXPECT_TRUE(mStack->isValid());

  mStack->discard(2);
  EXPECT_TRUE(mStack->isValid());
  EXPECT_EQ(1234, mStack->pop<uint32_t>());
}

TEST_F(StackTest, SequentialDiscard) {
  mStack->push<uint32_t>(12);
  mStack->push<uint32_t>(123);
  mStack->push<uint32_t>(234);
  mStack->discard(1);
  mStack->push<uint32_t>(345);
  mStack->discard(1);

  EXPECT_TRUE(mStack->isValid());
  EXPECT_EQ(123, mStack->pop<uint32_t>());
}

TEST_F(StackTest, DiscardErrorStackUnderflow) {
  mStack->push<uint32_t>(1234);
  EXPECT_TRUE(mStack->isValid());

  mStack->discard(2);
  EXPECT_FALSE(mStack->isValid());

  mStack->discard(1);
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, Clone) {
  mStack->push<uint32_t>(1234);
  mStack->push<uint32_t>(2345);
  EXPECT_TRUE(mStack->isValid());

  mStack->clone(1);
  EXPECT_TRUE(mStack->isValid());

  EXPECT_EQ(1234, mStack->pop<uint32_t>());
  EXPECT_EQ(2345, mStack->pop<uint32_t>());
  EXPECT_EQ(1234, mStack->pop<uint32_t>());
  EXPECT_TRUE(mStack->isValid());
}

TEST_F(StackTest, CloneErrorNonExistantIndex) {
  mStack->clone(1);
  EXPECT_FALSE(mStack->isValid());

  mStack->clone(1);
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, CloneErrorOutsideOfCapacity) {
  mStack->push<uint32_t>(1234);
  mStack->push<uint32_t>(2345);
  EXPECT_TRUE(mStack->isValid());

  mStack->clone(3);
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, CloneErrorStackOverflow) {
  fillStack(mStack.get(), STACK_CAPACITY);
  EXPECT_TRUE(mStack->isValid());

  mStack->clone(1);
  EXPECT_FALSE(mStack->isValid());

  mStack->clone(1);
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, PushPop) {
  mStack->push<uint32_t>(123);
  EXPECT_TRUE(mStack->isValid());

  ASSERT_EQ(123, mStack->pop<uint32_t>());
  EXPECT_TRUE(mStack->isValid());
}

TEST_F(StackTest, PopVolatilePointer) {
  uint32_t a = 0;
  mStack->pushValue(BaseType::VolatilePointer, a);
  EXPECT_TRUE(mStack->isValid());
  EXPECT_EQ(mMemoryManager->volatileToAbsolute(0), mStack->pop<void*>());
}

TEST_F(StackTest, PopConstantPointer) {
  uint32_t a = 0;
  mStack->pushValue(BaseType::ConstantPointer, a);
  EXPECT_TRUE(mStack->isValid());
  EXPECT_EQ(mMemoryManager->constantToAbsolute(0), mStack->pop<const void*>());
}

TEST_F(StackTest, PopAbsolutePointer) {
  mStack->push<void*>(nullptr);
  EXPECT_TRUE(mStack->isValid());
  EXPECT_EQ(nullptr, mStack->pop<void*>());
}

TEST_F(StackTest, PopErrorStackUnderflow) {
  mStack->pop<uint32_t>();
  EXPECT_FALSE(mStack->isValid());

  mStack->pop<uint32_t>();
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, PopErrorTypeMissmatch) {
  mStack->push<uint16_t>(123);
  EXPECT_TRUE(mStack->isValid());

  mStack->pop<uint32_t>();
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, PopErrorMismatchingPointerType) {
  mStack->push<uint32_t>(123);
  EXPECT_TRUE(mStack->isValid());

  mStack->pop<void*>();
  EXPECT_FALSE(mStack->isValid());
}

TEST_F(StackTest, PushErrorOverCapacity) {
  fillStack(mStack.get(), STACK_CAPACITY);
  EXPECT_TRUE(mStack->isValid());

  mStack->push<uint32_t>(1);
  EXPECT_FALSE(mStack->isValid());

  mStack->push<uint32_t>(1);
  EXPECT_FALSE(mStack->isValid());
}

}  // namespace test
}  // namespace gapir
