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
#include "memory_manager.h"

#include "core/cc/crash_handler.h"
#include "core/cc/log.h"

#define __STDC_FORMAT_MACROS
#include <inttypes.h>

#if !defined(_MSC_VER) || defined(__GNUC__)
// If compiling with MSVC, (rather than MSYS)
// unistd does not exist.
#include <unistd.h>
#endif  // !defined(_MSC_VER) || defined(__GNUC__)

#include <utility>
#include <vector>

namespace gapir {

namespace {

template <typename T>
inline T sum2(T a, T b) {
  return a + b;
}
template <typename T>
inline T* sum2(T* a, T* b) {
  return reinterpret_cast<T*>(reinterpret_cast<uintptr_t>(a) +
                              reinterpret_cast<uintptr_t>(b));
}

template <typename T>
inline bool sum(Stack& stack, uint32_t count) {
  T v = 0;
  for (uint32_t i = 0; i < count; i++) {
    v = sum2(v, stack.pop<T>());
  }
  stack.push(v);
  return stack.isValid();
}

}  // anonymous namespace

Interpreter::Interpreter(core::CrashHandler& crash_handler,
                         const MemoryManager* memory_manager,
                         uint32_t stack_depth)
    :

      mCrashHandler(crash_handler),
      mMemoryManager(memory_manager),
      mStack(stack_depth, mMemoryManager),
      mInstructions(nullptr),
      mInstructionCount(0),
      mCurrentInstruction(0),
      mNextThread(0),
      mLabel(0) {
  registerBuiltin(GLOBAL_INDEX, PRINT_STACK_FUNCTION_ID,
                  [](uint32_t, Stack* stack, bool) {
                    stack->printStack();
                    return true;
                  });
}

void Interpreter::setApiRequestCallback(ApiRequestCallback callback) {
  apiRequestCallback = std::move(callback);
}

void Interpreter::setCheckReplayStatusCallback(
    CheckReplayStatusCallback callback) {
  checkReplayStatusCallback = std::move(callback);
}

void Interpreter::registerBuiltin(uint8_t api, FunctionTable::Id id,
                                  FunctionTable::Function func) {
  mBuiltins[api].insert(id, func);
}

void Interpreter::setRendererFunctions(uint8_t api,
                                       FunctionTable* functionTable) {
  if (functionTable != nullptr) {
    mRendererFunctions[api] = functionTable;
  } else {
    mRendererFunctions.erase(api);
  }
}

void Interpreter::resetInstructions() {
  mInstructions = nullptr;
  mInstructionCount = 0;
  mCurrentInstruction = 0;
  mJumpLabels.clear();
}

bool Interpreter::updateJumpTable(uint32_t jumpLabel) {
  uint32_t instruction = 0;

  if (mJumpLabels.rbegin() != mJumpLabels.rend()) {
    instruction = mJumpLabels.rbegin()->second + 1;
  }

  for (; instruction < mInstructionCount; instruction++) {
    auto opcode = mInstructions[instruction];

    InstructionCode code =
        static_cast<InstructionCode>(opcode >> OPCODE_BIT_SHIFT);

    if (code == InstructionCode::JUMP_LABEL) {
      auto jump_id = extract26bitData(opcode);
      mJumpLabels[jump_id] = instruction;

      if (jump_id == jumpLabel) {
        return true;
      }
    }
  }

  return false;
}

bool Interpreter::run(const uint32_t* instructions, uint32_t count) {
  GAPID_ASSERT(mInstructions == nullptr);
  GAPID_ASSERT(mInstructionCount == 0);
  GAPID_ASSERT(mCurrentInstruction == 0);
  GAPID_ASSERT(mJumpLabels.empty());

  mInstructions = instructions;
  mInstructionCount = count;

  // Reset the promise here, otherwise this may throw.
  mExecResult = std::promise<Result>();
  auto unregisterHandler = mCrashHandler.registerHandler(
      [this](const std::string& minidumpPath, bool succeeded) {
        GAPID_ERROR("--- CRASH DURING REPLAY ---");
        GAPID_ERROR("LAST COMMAND:     %d", mLabel);
        GAPID_ERROR("LAST INSTRUCTION: %d", mCurrentInstruction);
      });

  exec();
  unregisterHandler();

  return mExecResult.get_future().get() == SUCCESS;
}

void Interpreter::exec() {
  for (; mCurrentInstruction < mInstructionCount; mCurrentInstruction++) {
    switch (interpret(mInstructions[mCurrentInstruction])) {
      case SUCCESS:
        break;
      case ERROR:
        GAPID_WARNING(
            "Interpreter stopped because of an interpretation error at opcode "
            "%u (%u). "
            "Last reached label: %d",
            mCurrentInstruction, mInstructions[mCurrentInstruction], mLabel);
        mExecResult.set_value(ERROR);
        return;
      case CHANGE_THREAD: {
        auto next_thread = mNextThread;
        mCurrentInstruction++;
        mThreadPool.enqueue(next_thread, [this] { this->exec(); });
        return;
      }
    }
  }
  mExecResult.set_value(SUCCESS);
}

uint32_t Interpreter::extract6bitData(uint32_t opcode) const {
  return (opcode & TYPE_MASK) >> TYPE_BIT_SHIFT;
}

uint32_t Interpreter::extract20bitData(uint32_t opcode) const {
  return opcode & DATA_MASK20;
}

uint32_t Interpreter::extract26bitData(uint32_t opcode) const {
  return opcode & DATA_MASK26;
}

BaseType Interpreter::extractType(uint32_t opcode) const {
  return BaseType(extract6bitData(opcode));
}

bool Interpreter::registerApi(uint8_t api) {
  return apiRequestCallback(this, api);
}

Interpreter::Result Interpreter::call(uint32_t opcode) {
  auto id = opcode & FUNCTION_ID_MASK;
  auto api = (opcode & API_INDEX_MASK) >> API_BIT_SHIFT;
  auto func = mBuiltins[api].lookup(id);
  auto label = getLabel();
  if (checkReplayStatusCallback) {
    checkReplayStatusCallback(label, mInstructionCount, mCurrentInstruction);
  }
  if (func == nullptr) {
    if (mRendererFunctions.count(api) > 0) {
      func = mRendererFunctions[api]->lookup(id);
    } else {
      if (apiRequestCallback && apiRequestCallback(this, api)) {
        func = mRendererFunctions[api]->lookup(id);
      } else {
        GAPID_WARNING("[%u]Error setting up renderer functions for api: %u",
                      label, api);
      }
    }
  }
  if (func == nullptr) {
    GAPID_WARNING("[%u]Invalid function id(%u), in api(%d)", label, id, api);
    return ERROR;
  }
  if (!(*func)(getLabel(), &mStack, (opcode & PUSH_RETURN_MASK) != 0)) {
    GAPID_WARNING("[%u]Error raised when calling function with id: %u", label,
                  id);
    return ERROR;
  }
  return SUCCESS;
}

Interpreter::Result Interpreter::pushI(uint32_t opcode) {
  BaseType type = extractType(opcode);
  if (!isValid(type)) {
    GAPID_WARNING("Error: pushI basic type invalid %d", (int)type);
    return ERROR;
  }
  Stack::BaseValue data = extract20bitData(opcode);
  switch (type) {
    // Sign extension for signed types
    case BaseType::Int32:
    case BaseType::Int64:
      if (data & 0x80000) {
        data |= 0xfffffffffff00000ULL;
      }
      break;
    // Shifting the value into the exponent for floating point types
    case BaseType::Float:
      data <<= 23;
      break;
    case BaseType::Double:
      data <<= 52;
      break;
    default:
      break;
  }
  mStack.pushValue(type, data);
  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::loadC(uint32_t opcode) {
  BaseType type = extractType(opcode);
  if (!isValid(type)) {
    GAPID_WARNING("Error: loadC basic type invalid %u", (unsigned int)type);
    return ERROR;
  }
  const void* address =
      mMemoryManager->constantToAbsolute(extract20bitData(opcode));
  if (!isConstantAddressForType(address, type)) {
    GAPID_WARNING("Error: loadC not constant address %p", address);
    return ERROR;
  }
  mStack.pushFrom(type, address);
  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::loadV(uint32_t opcode) {
  BaseType type = extractType(opcode);
  if (!isValid(type)) {
    GAPID_WARNING("Error: loadV basic type invalid %u", (unsigned int)type);
    return ERROR;
  }
  const void* address =
      mMemoryManager->volatileToAbsolute(extract20bitData(opcode));
  if (!isVolatileAddressForType(address, type)) {
    GAPID_WARNING("Error: loadV not volatile address %p", address);
    return ERROR;
  }
  mStack.pushFrom(type, address);
  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::load(uint32_t opcode) {
  BaseType type = extractType(opcode);
  if (!isValid(type)) {
    GAPID_WARNING("Error: load basic type invalid %u", (unsigned int)type);
    return ERROR;
  }
  const void* address = mStack.pop<const void*>();
  if (!isReadAddress(address)) {
    GAPID_WARNING("Error: load not readable address %p", address);
    return ERROR;
  }
  mStack.pushFrom(type, address);
  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::pop(uint32_t opcode) {
  mStack.discard(extract26bitData(opcode));
  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::storeV(uint32_t opcode) {
  void* address = mMemoryManager->volatileToAbsolute(extract26bitData(opcode));
  if (!isVolatileAddressForType(address, mStack.getTopType())) {
    GAPID_WARNING("Error: storeV not volatile address %p", address);
    return ERROR;
  }

  mStack.popTo(address);
  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::store() {
  void* address = mStack.pop<void*>();
  if (!isWriteAddress(address)) {
    GAPID_WARNING("Error: store not write address %p", address);
    return ERROR;
  }
  mStack.popTo(address);
  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::resource(uint32_t opcode) {
  mStack.push<uint32_t>(extract26bitData(opcode));
  return this->call(Interpreter::RESOURCE_FUNCTION_ID);
}

Interpreter::Result Interpreter::inlineResource(uint32_t opcode) {
  unsigned int numValuePatchUps = extract6bitData(opcode);
  unsigned int dataSize = extract20bitData(opcode);

  void* destination = mStack.pop<void*>();
  memcpy(destination, &mInstructions[mCurrentInstruction + 1],
         (size_t)dataSize);

  auto roundedDataSize = (dataSize / 4) + (dataSize % 4 != 0 ? 1 : 0);

  for (unsigned int i = 0; i < numValuePatchUps; ++i) {
    uint32_t destination =
        mInstructions[mCurrentInstruction + 1 + roundedDataSize + i * 2];
    uint32_t value =
        mInstructions[mCurrentInstruction + 1 + roundedDataSize + i * 2 + 1];

    *((void**)mMemoryManager->volatileToAbsolute(destination)) =
        (void*)mMemoryManager->volatileToAbsolute(value);
  }

  uint32_t numPointerPatchUps =
      mInstructions[mCurrentInstruction + 1 + roundedDataSize +
                    numValuePatchUps * 2];

  for (unsigned int i = 0; i < numPointerPatchUps; ++i) {
    uint32_t destination =
        mInstructions[mCurrentInstruction + 1 + roundedDataSize +
                      numValuePatchUps * 2 + 1 + i * 2];

    uint32_t source = mInstructions[mCurrentInstruction + 1 + roundedDataSize +
                                    numValuePatchUps * 2 + 1 + i * 2 + 1];

    *((void**)mMemoryManager->volatileToAbsolute(destination)) =
        *((void**)mMemoryManager->volatileToAbsolute(source));
  }

  unsigned int inlineData =
      roundedDataSize + numValuePatchUps * 2 + 1 + numPointerPatchUps * 2;

  mCurrentInstruction = mCurrentInstruction + inlineData;
  return SUCCESS;
}

Interpreter::Result Interpreter::post() {
  return this->call(Interpreter::POST_FUNCTION_ID);
}

Interpreter::Result Interpreter::notification() {
  return this->call(Interpreter::NOTIFICATION_FUNCTION_ID);
}

Interpreter::Result Interpreter::wait(uint32_t opcode) {
  mStack.push<uint32_t>(extract26bitData(opcode));
  return this->call(Interpreter::WAIT_FUNCTION_ID);
}

Interpreter::Result Interpreter::copy(uint32_t opcode) {
  uint32_t count = extract26bitData(opcode);
  void* target = mStack.pop<void*>();
  const void* source = mStack.pop<const void*>();
  if (!isWriteAddress(target)) {
    GAPID_WARNING("Error: copy target is invalid %p %" PRIu32, target, count);
    return ERROR;
  }
  if (!isReadAddress(source)) {
    GAPID_WARNING("Error: copy source is invalid %p %" PRIu32, target, count);
    return ERROR;
  }
  if (source == nullptr) {
    GAPID_WARNING("Error: copy source address is null");
    return ERROR;
  }
  if (target == nullptr) {
    GAPID_WARNING("Error: copy destination address is null");
    return ERROR;
  }
  memcpy(target, source, count);
  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::clone(uint32_t opcode) {
  mStack.clone(extract26bitData(opcode));
  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::strcpy(uint32_t opcode) {
  uint32_t count = extract26bitData(opcode);
  char* target = mStack.pop<char*>();
  const char* source = mStack.pop<const char*>();
  // Requires that the whole count is available, even if source is shorter.
  if (!isWriteAddress(target)) {
    GAPID_WARNING("Error: copy target is invalid %p %d", target, count);
    return ERROR;
  }
  if (!isReadAddress(source)) {
    GAPID_WARNING("Error: copy source is invalid %p %d", target, count);
    return ERROR;
  }
  if (source == nullptr) {
    GAPID_WARNING("Error: strcpy source address is null");
    return ERROR;
  }
  if (target == nullptr) {
    GAPID_WARNING("Error: strcpy destination address is null");
    return ERROR;
  }
  uint32_t i;
  for (i = 0; i < count - 1; i++) {
    char c = source[i];
    if (c == 0) {
      break;
    }
    target[i] = c;
  }
  for (; i < count; i++) {
    target[i] = 0;
  }
  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::extend(uint32_t opcode) {
  uint32_t data = extract26bitData(opcode);
  auto type = mStack.getTopType();
  auto value = mStack.popBaseValue();
  switch (type) {
    // Masking out the mantissa end extending it with the new bits for floating
    // point types
    case BaseType::Float: {
      value |= (data & 0x007fffffULL);
      break;
    }
    case BaseType::Double: {
      uint64_t exponent = value & 0xfff0000000000000ULL;
      value <<= 26;
      value |= data;
      value &= 0x000fffffffffffffULL;
      value |= exponent;
      break;
    }
    // Extending the value with 26 new LSB
    default: {
      value = (value << 26) | data;
      break;
    }
  }
  mStack.pushValue(type, value);
  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::add(uint32_t opcode) {
  uint32_t count = extract26bitData(opcode);
  if (count < 2) {
    return mStack.isValid() ? SUCCESS : ERROR;
  }
  auto type = mStack.getTopType();
  bool ok = false;
  switch (type) {
    case BaseType::Int8: {
      ok = sum<int8_t>(mStack, count);
      break;
    }
    case BaseType::Int16: {
      ok = sum<int16_t>(mStack, count);
      break;
    }
    case BaseType::Int32: {
      ok = sum<int32_t>(mStack, count);
      break;
    }
    case BaseType::Int64: {
      ok = sum<int64_t>(mStack, count);
      break;
    }
    case BaseType::Uint8: {
      ok = sum<uint8_t>(mStack, count);
      break;
    }
    case BaseType::Uint16: {
      ok = sum<uint16_t>(mStack, count);
      break;
    }
    case BaseType::Uint32: {
      ok = sum<uint32_t>(mStack, count);
      break;
    }
    case BaseType::Uint64: {
      ok = sum<uint64_t>(mStack, count);
      break;
    }
    case BaseType::Float: {
      ok = sum<float>(mStack, count);
      break;
    }
    case BaseType::Double: {
      ok = sum<double>(mStack, count);
      break;
    }
    case BaseType::AbsolutePointer: {
      ok = sum<void*>(mStack, count);
      break;
    }
    case BaseType::ConstantPointer: {
      ok = sum<void*>(mStack, count);
      break;
    }
    default:
      GAPID_WARNING("Cannot add values of type %s", baseTypeName(type));
      return ERROR;
  }
  return ok ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::label(uint32_t opcode) {
  mLabel = extract26bitData(opcode);
  return SUCCESS;
}

Interpreter::Result Interpreter::switchThread(uint32_t opcode) {
  auto thread = extract26bitData(opcode);
  GAPID_DEBUG("Switch thread %d -> %d", mNextThread, thread);
  mNextThread = thread;
  return CHANGE_THREAD;
}

Interpreter::Result Interpreter::jumpLabel(uint32_t opcode) {
  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::jumpNZ(uint32_t opcode) {
  auto jump_id = extract26bitData(opcode);
  auto should_jump = mStack.pop<int32_t>();

  if (!mStack.isEmpty()) {
    GAPID_WARNING("Error: stack is not empty before jumping to label %d",
                  jump_id);
    return ERROR;
  }

  if (should_jump != 0) {
    if (mJumpLabels.find(jump_id) == mJumpLabels.end() &&
        updateJumpTable(jump_id) == false) {
      GAPID_WARNING("Error: unknown jumpLabel %i", jump_id);
    }

    GAPID_VERBOSE("JUMP TAKEN");
    // The -1 on the following line is present because the program counter
    // is going to step forwards after this instruction is complete.
    mCurrentInstruction = mJumpLabels[jump_id] - 1;
  } else {
    GAPID_VERBOSE("JUMP NOT TAKEN");
  }

  return mStack.isValid() ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::jumpZ(uint32_t opcode) {
  auto jump_id = extract26bitData(opcode);
  auto should_jump = mStack.pop<int32_t>();

  if (!mStack.isEmpty()) {
    GAPID_WARNING("Error: stack is not empty before jumping to label %d",
                  jump_id);
    return ERROR;
  }

  if (should_jump == 0) {
    if (mJumpLabels.find(jump_id) == mJumpLabels.end() &&
        updateJumpTable(jump_id) == false) {
      GAPID_WARNING("Error: unknown jumpLabel %i", jump_id);
    }

    GAPID_VERBOSE("JUMP TAKEN");
    // The -1 on the following line is present because the program counter
    // is going to step forwards after this instruction is complete.
    mCurrentInstruction = mJumpLabels[jump_id] - 1;
  } else {
    GAPID_VERBOSE("JUMP NOT TAKEN");
  }

  return mStack.isValid() ? SUCCESS : ERROR;
}

#define DEBUG_OPCODE(name, value) GAPID_VERBOSE(name)
#define DEBUG_OPCODE_26(name, value) \
  GAPID_VERBOSE(name "(%#010x)", value& DATA_MASK26)
#define DEBUG_OPCODE_TY_20(name, value)                  \
  GAPID_VERBOSE(name "(%#010x, %s)", value& DATA_MASK20, \
                baseTypeName(extractType(value)))

Interpreter::Result Interpreter::interpret(uint32_t opcode) {
  InstructionCode code =
      static_cast<InstructionCode>(opcode >> OPCODE_BIT_SHIFT);
  switch (code) {
    case InstructionCode::CALL:
      DEBUG_OPCODE_26("CALL", opcode);
      return this->call(opcode);
    case InstructionCode::PUSH_I:
      DEBUG_OPCODE_TY_20("PUSH_I", opcode);
      return this->pushI(opcode);
    case InstructionCode::LOAD_C:
      DEBUG_OPCODE_TY_20("LOAD_C", opcode);
      return this->loadC(opcode);
    case InstructionCode::LOAD_V:
      DEBUG_OPCODE_TY_20("LOAD_V", opcode);
      return this->loadV(opcode);
    case InstructionCode::LOAD:
      DEBUG_OPCODE_TY_20("LOAD", opcode);
      return this->load(opcode);
    case InstructionCode::POP:
      DEBUG_OPCODE_26("POP", opcode);
      return this->pop(opcode);
    case InstructionCode::STORE_V:
      DEBUG_OPCODE_26("STORE_V", opcode);
      return this->storeV(opcode);
    case InstructionCode::STORE:
      DEBUG_OPCODE("STORE", opcode);
      return this->store();
    case InstructionCode::RESOURCE:
      DEBUG_OPCODE_26("RESOURCE", opcode);
      return this->resource(opcode);
    case InstructionCode::INLINE_RESOURCE:
      DEBUG_OPCODE_26("INLINE_RESOURCE", opcode);
      return this->inlineResource(opcode);
    case InstructionCode::POST:
      DEBUG_OPCODE("POST", opcode);
      return this->post();
    case InstructionCode::COPY:
      DEBUG_OPCODE_26("COPY", opcode);
      return this->copy(opcode);
    case InstructionCode::CLONE:
      DEBUG_OPCODE_26("CLONE", opcode);
      return this->clone(opcode);
    case InstructionCode::STRCPY:
      DEBUG_OPCODE_26("STRCPY", opcode);
      return this->strcpy(opcode);
    case InstructionCode::EXTEND:
      DEBUG_OPCODE_26("EXTEND", opcode);
      return this->extend(opcode);
    case InstructionCode::ADD:
      DEBUG_OPCODE_26("ADD", opcode);
      return this->add(opcode);
    case InstructionCode::LABEL:
      DEBUG_OPCODE_26("LABEL", opcode);
      return this->label(opcode);
    case InstructionCode::SWITCH_THREAD:
      DEBUG_OPCODE_26("SWITCH_THREAD", opcode);
      return this->switchThread(opcode);
    case InstructionCode::JUMP_LABEL:
      DEBUG_OPCODE_26("JUMP_LABEL", opcode);
      return this->jumpLabel(opcode);
    case InstructionCode::JUMP_NZ:
      DEBUG_OPCODE_26("JUMP_NZ", opcode);
      return this->jumpNZ(opcode);
    case InstructionCode::JUMP_Z:
      DEBUG_OPCODE_26("JUMP_Z", opcode);
      return this->jumpZ(opcode);
    case InstructionCode::NOTIFICATION:
      DEBUG_OPCODE("NOTIFICATION", opcode);
      return this->notification();
    case InstructionCode::WAIT:
      DEBUG_OPCODE("WAIT", opcode);
      return this->wait(opcode);
    default:
      GAPID_WARNING("Unknown opcode! %#010x", opcode);
      return ERROR;
  }
}

#undef DEBUG_OPCODE

}  // namespace gapir
