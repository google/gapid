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

#include "interpreter.h"
#include "test_utilities.h"

#include <gtest/gtest.h>

#include <memory>
#include <vector>

namespace gapir {
namespace test {
namespace {

const uint32_t MEMORY_SIZE = 4096;
const uint32_t STACK_SIZE = 128;

template <typename T>
struct CheckTopOfStack {
    T expected_;
    bool operator()(uint32_t, Stack* stack, bool) {
        EXPECT_EQ(expected_, stack->pop<T>());
        return true;
    }
};

class InterpreterTest : public ::testing::Test {
protected:
    virtual void SetUp() {
        std::vector<uint32_t> memorySizes = {MEMORY_SIZE};
        mMemoryManager.reset(new MemoryManager(memorySizes));
        auto callback = [](Interpreter*, uint8_t) { return false; };
        mInterpreter.reset(new Interpreter(mMemoryManager.get(), STACK_SIZE, std::move(callback)));
    }

    std::unique_ptr<MemoryManager> mMemoryManager;
    std::unique_ptr<Interpreter> mInterpreter;
};
}  // anonymous namespace

TEST_F(InterpreterTest, PushIUint8) {
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<uint8_t>{210});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint8, 210),
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, PushIInt16Minus1) {
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<int16_t>{-1});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Int16, 0xffff),  // -1
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, PushIInt32Minus1) {
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<int32_t>{-1});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Int32, 0xfffff),  // -1
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, PushIFloat1) {
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<float>{1.0f});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Float, 0x7f),  // 1.0 exp
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, PushIDouble1) {
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<double>{1.0});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Double, 0x3ff),  // 1.0 exp
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, LoadC) {
    mMemoryManager->setReplayDataSize(10);
    uint8_t constantMemory[7] = {0x00, 0x00, 0x12, 0x34, 0x56, 0x78, 0x9a};
    uint8_t* constantBaseAddress = static_cast<uint8_t*>(mMemoryManager->getReplayAddress()) + 2;
    memcpy(constantBaseAddress, &constantMemory, sizeof(constantMemory));
    mMemoryManager->setConstantMemory({constantBaseAddress, sizeof(constantMemory)});

    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<uint16_t>{0x7856});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::LOAD_C, BaseType::Uint16, 4),
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, LoadV) {
    *static_cast<int32_t*>(mMemoryManager->volatileToAbsolute(784)) = -987654321;
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<int32_t>{-987654321});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::LOAD_V, BaseType::Int32, 784),
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, LoadConstantAddress) {
    mMemoryManager->setReplayDataSize(10);
    uint8_t constantMemory[7] = {0x00, 0x00, 0x12, 0x34, 0x56, 0x78, 0x9a};
    uint8_t* constantBaseAddress = static_cast<uint8_t*>(mMemoryManager->getReplayAddress()) + 2;
    memcpy(constantBaseAddress, &constantMemory, 7);
    mMemoryManager->setConstantMemory({constantBaseAddress, 7});

    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<uint16_t>{0x7856});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::ConstantPointer, 4),
            instruction(Interpreter::InstructionCode::LOAD, BaseType::Uint16, 0),
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, LoadVolatileAddress) {
    *static_cast<int32_t*>(mMemoryManager->volatileToAbsolute(784)) = -987654321;
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<int32_t>{-987654321});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::VolatilePointer, 784),
            instruction(Interpreter::InstructionCode::LOAD, BaseType::Int32, 0),
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, Pop) {
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<uint32_t>{123456});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint32, 123456),
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint16, 987),
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Int8, -123),
            instruction(Interpreter::InstructionCode::POP, 2),
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, StoreV) {
    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint32, 987654),
            instruction(Interpreter::InstructionCode::STORE_V, 124)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);

    EXPECT_EQ(987654, *static_cast<uint32_t*>(mMemoryManager->volatileToAbsolute(124)));
}

TEST_F(InterpreterTest, Store) {
    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint32, 987654),
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::VolatilePointer, 260),
            instruction(Interpreter::InstructionCode::STORE, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);

    EXPECT_EQ(987654, *static_cast<uint32_t*>(mMemoryManager->volatileToAbsolute(260)));
}

TEST_F(InterpreterTest, Copy) {
    mMemoryManager->setReplayDataSize(20);
    uint8_t constantMemory[10] = {0, 1, 2, 3, 4, 5, 6, 7, 8, 9};
    uint8_t* constantBaseAddress = static_cast<uint8_t*>(mMemoryManager->getReplayAddress()) + 2;
    memcpy(constantBaseAddress, &constantMemory, 10);
    mMemoryManager->setConstantMemory({constantBaseAddress, 10});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::ConstantPointer, 5),
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::VolatilePointer, 987),
            instruction(Interpreter::InstructionCode::COPY, 3)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);

    EXPECT_EQ(5, *static_cast<uint8_t*>(mMemoryManager->volatileToAbsolute(987)));
    EXPECT_EQ(6, *static_cast<uint8_t*>(mMemoryManager->volatileToAbsolute(988)));
    EXPECT_EQ(7, *static_cast<uint8_t*>(mMemoryManager->volatileToAbsolute(989)));
}

TEST_F(InterpreterTest, Clone) {
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<uint32_t>{123456});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint32, 123456),
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint16, 987),
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Int8, -123),
            instruction(Interpreter::InstructionCode::CLONE, 2),
            instruction(Interpreter::InstructionCode::CALL, 0),
            instruction(Interpreter::InstructionCode::POP, 2),
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, ExtendInt32) {
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<int32_t>{0x76543210});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Int32, 0x1d),
            instruction(Interpreter::InstructionCode::EXTEND, 0x2543210),
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, ExtendFloat) {
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<float>{1.1f});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Float, 0x7f),
            instruction(Interpreter::InstructionCode::EXTEND, 0x8ccccd),
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, ExtendDouble) {
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<double>{1.4});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Double, 0x3ff),
            instruction(Interpreter::InstructionCode::EXTEND, 0x1999999),
            instruction(Interpreter::InstructionCode::EXTEND, 0x2666666),
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, Add2xUint32) {
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<uint32_t>{15});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint32, 5),
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Uint32, 10),
            instruction(Interpreter::InstructionCode::ADD, 2),
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, Add3xFloat) {
    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<float>{3.5});

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Float, 0x7f),  // 1.0 exp
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Float, 0x7e),  // 0.5 exp
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::Float, 0x80),  // 2.0 exp
            instruction(Interpreter::InstructionCode::ADD, 3),
            instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
}

TEST_F(InterpreterTest, Strcpy) {
    mMemoryManager->setReplayDataSize(20);
    const char* constantMemory = "abc";
    uint8_t* constantBaseAddress = static_cast<uint8_t*>(mMemoryManager->getReplayAddress());
    memcpy(constantBaseAddress, constantMemory, 4);
    mMemoryManager->setConstantMemory({constantBaseAddress, 4});

    uint8_t* volatileMemory = static_cast<uint8_t*>(mMemoryManager->volatileToAbsolute(100));

    memset(volatileMemory, 'x', 5);

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::ConstantPointer, 0),
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::VolatilePointer, 100),
            instruction(Interpreter::InstructionCode::STRCPY, 10)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);

    EXPECT_EQ('a', volatileMemory[0]);
    EXPECT_EQ('b', volatileMemory[1]);
    EXPECT_EQ('c', volatileMemory[2]);
    EXPECT_EQ(0x0, volatileMemory[3]);
    EXPECT_EQ(0x0, volatileMemory[4]);
}

TEST_F(InterpreterTest, StrcpyShortBuffer) {
    mMemoryManager->setReplayDataSize(20);
    const char* constantMemory = "abcdef";
    uint8_t* constantBaseAddress = static_cast<uint8_t*>(mMemoryManager->getReplayAddress());
    memcpy(constantBaseAddress, constantMemory, 7);
    mMemoryManager->setConstantMemory({constantBaseAddress, 7});

    uint8_t* volatileMemory = static_cast<uint8_t*>(mMemoryManager->volatileToAbsolute(100));

    memset(volatileMemory, 'x', 8);

    std::vector<uint32_t> instructions{
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::ConstantPointer, 0),
            instruction(Interpreter::InstructionCode::PUSH_I, BaseType::VolatilePointer, 100),
            instruction(Interpreter::InstructionCode::STRCPY, 5)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);

    EXPECT_EQ('a', volatileMemory[0]);
    EXPECT_EQ('b', volatileMemory[1]);
    EXPECT_EQ('c', volatileMemory[2]);
    EXPECT_EQ('d', volatileMemory[3]);
    EXPECT_EQ(0x0, volatileMemory[4]);
    EXPECT_EQ('x', volatileMemory[5]);
}

TEST_F(InterpreterTest, Post) {
    uint32_t callCount = 0;
    auto post = [&callCount](uint32_t, Stack*, bool) {
        ++callCount;
        return true;
    };

    mInterpreter->registerBuiltin(0, Interpreter::POST_FUNCTION_ID, post);

    std::vector<uint32_t> instructions{instruction(Interpreter::InstructionCode::POST)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
    EXPECT_EQ(1, callCount);
}

TEST_F(InterpreterTest, Resource) {
    uint32_t callCount = 0;
    auto resource = [&callCount](uint32_t, Stack* stack, bool) {
        ++callCount;
        return true;
    };

    mInterpreter->registerBuiltin(0, 0, CheckTopOfStack<uint32_t>{123});
    mInterpreter->registerBuiltin(0, Interpreter::RESOURCE_FUNCTION_ID, resource);

    std::vector<uint32_t> instructions{instruction(Interpreter::InstructionCode::RESOURCE, 123),
                                       instruction(Interpreter::InstructionCode::CALL, 0)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_TRUE(res);
    EXPECT_EQ(1, callCount);
}

TEST_F(InterpreterTest, InvalidOpcode) {
    std::vector<uint32_t> instructions{63U << 26};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_FALSE(res);
}

TEST_F(InterpreterTest, InvalidFunctionId) {
    std::vector<uint32_t> instructions{instruction(Interpreter::InstructionCode::CALL, 0xffff)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_FALSE(res);
}

TEST_F(InterpreterTest, UnknownApi) {
    std::vector<uint32_t> instructions{instruction(Interpreter::InstructionCode::CALL, 1 << 16)};
    bool res = mInterpreter->run(instructions.data(), instructions.size());
    EXPECT_FALSE(res);
}

}  // namespace test
}  // namespace gapir
