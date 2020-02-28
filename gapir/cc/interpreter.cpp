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

#define DEBUG_OPCODE(name, value) GAPID_VERBOSE(name)
#define DEBUG_OPCODE_26(name, value) \
  GAPID_VERBOSE(name "(%#010x)", value& DATA_MASK26)
#define DEBUG_OPCODE_TY_20(name, value)                  \
  GAPID_VERBOSE(name "(%#010x, %s)", value& DATA_MASK20, \
                baseTypeName(extractType(value)))

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
inline void sum(Stack& stack, uint32_t count) {
  T v = 0;
  for (uint32_t i = 0; i < count; i++) {
    v = sum2(v, stack.pop<T>());
  }
  stack.push(v);
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
      mLabel(0) {}

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
    const uint32_t opcode = mInstructions[mCurrentInstruction];
    Result ret;
    {
      InstructionCode code =
          static_cast<InstructionCode>(opcode >> OPCODE_BIT_SHIFT);
      switch (code) {
        case InstructionCode::CALL:
          DEBUG_OPCODE_26("CALL", opcode);
          ret = this->call(opcode);
          break;
        case InstructionCode::PUSH_I:
          DEBUG_OPCODE_TY_20("PUSH_I", opcode);
          ret = this->pushI(opcode);
          break;
        case InstructionCode::LOAD_C:
          DEBUG_OPCODE_TY_20("LOAD_C", opcode);
          ret = this->loadC(opcode);
          break;
        case InstructionCode::LOAD_V:
          DEBUG_OPCODE_TY_20("LOAD_V", opcode);
          ret = this->loadV(opcode);
          break;
        case InstructionCode::LOAD:
          DEBUG_OPCODE_TY_20("LOAD", opcode);
          ret = this->load(opcode);
          break;
        case InstructionCode::POP:
          DEBUG_OPCODE_26("POP", opcode);
          ret = this->pop(opcode);
          break;
        case InstructionCode::STORE_V:
          DEBUG_OPCODE_26("STORE_V", opcode);
          ret = this->storeV(opcode);
          break;
        case InstructionCode::STORE:
          DEBUG_OPCODE("STORE", opcode);
          ret = this->store(opcode);
          break;
        case InstructionCode::RESOURCE:
          DEBUG_OPCODE_26("RESOURCE", opcode);
          ret = this->resource(opcode);
          break;
        case InstructionCode::POST:
          DEBUG_OPCODE("POST", opcode);
          ret = this->post(opcode);
          break;
        case InstructionCode::COPY:
          DEBUG_OPCODE_26("COPY", opcode);
          ret = this->copy(opcode);
          break;
        case InstructionCode::CLONE:
          DEBUG_OPCODE_26("CLONE", opcode);
          ret = this->clone(opcode);
          break;
        case InstructionCode::STRCPY:
          DEBUG_OPCODE_26("STRCPY", opcode);
          ret = this->strcpy(opcode);
          break;
        case InstructionCode::EXTEND:
          DEBUG_OPCODE_26("EXTEND", opcode);
          ret = this->extend(opcode);
          break;
        case InstructionCode::ADD:
          DEBUG_OPCODE_26("ADD", opcode);
          ret = this->add(opcode);
          break;
        case InstructionCode::LABEL:
          DEBUG_OPCODE_26("LABEL", opcode);
          ret = this->label(opcode);
          break;
        case InstructionCode::SWITCH_THREAD:
          DEBUG_OPCODE_26("SWITCH_THREAD", opcode);
          ret = this->switchThread(opcode);
          break;
        case InstructionCode::JUMP_LABEL:
          DEBUG_OPCODE_26("JUMP_LABEL", opcode);
          ret = this->jumpLabel(opcode);
          break;
        case InstructionCode::JUMP_NZ:
          DEBUG_OPCODE_26("JUMP_NZ", opcode);
          ret = this->jumpNZ(opcode);
          break;
        case InstructionCode::JUMP_Z:
          DEBUG_OPCODE_26("JUMP_Z", opcode);
          ret = this->jumpZ(opcode);
          break;
        case InstructionCode::NOTIFICATION:
          DEBUG_OPCODE("NOTIFICATION", opcode);
          ret = this->notification(opcode);
          break;
        case InstructionCode::WAIT:
          DEBUG_OPCODE("WAIT", opcode);
          ret = this->wait(opcode);
          break;
        default:
          GAPID_WARNING("Unknown opcode! %#010x", opcode);
          ret = ERROR;
      }
    }

    switch (ret) {
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
      case SUCCESS:
        break;
    }
  }

  mExecResult.set_value(SUCCESS);
}

BaseType Interpreter::extractType(uint32_t opcode) const {
  return BaseType((opcode & TYPE_MASK) >> TYPE_BIT_SHIFT);
}

uint32_t Interpreter::extract20bitData(uint32_t opcode) const {
  return opcode & DATA_MASK20;
}

uint32_t Interpreter::extract26bitData(uint32_t opcode) const {
  return opcode & DATA_MASK26;
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

  uint64_t data = extract20bitData(opcode);
  GAPID_WARNING("data: %u", data);

  switch (type) {
    case BaseType::Bool:
      mStack.push(data != 0);
      break;

    case BaseType::Int8:
      mStack.push<int8_t>(data);
      break;

    case BaseType::Int16:
      mStack.push<int16_t>(data);
      break;

    case BaseType::Int32: {
      int32_t val =
          data | ((data & 0x80000) != 0
                      ? 0xfff00000ULL
                      : 0x0);  // Sign extend the 20 bit immediate value
      mStack.push(val);
    } break;

    case BaseType::Int64: {
      int64_t val =
          data | ((data & 0x80000) != 0
                      ? 0xfffffffffff00000ULL
                      : 0x0);  // Sign extend the 20 bit immediate value
      mStack.push(val);
    } break;

    case BaseType::Uint8:
      mStack.push<uint8_t>(data);
      break;

    case BaseType::Uint16:
      mStack.push<uint16_t>(data);
      break;

    case BaseType::Uint32:
      mStack.push<uint32_t>(data);
      break;

    case BaseType::Uint64:
      mStack.push<uint64_t>(data);
      break;

    case BaseType::Float: {
      // Shifting the value into the exponent for floating point types
      data <<= 23;
      float val = 0;
      memcpy(&val, &data, sizeof(float));
      mStack.push(val);
    } break;

    case BaseType::Double: {
      // Shifting the value into the exponent for floating point types
      data <<= 52;
      double val = 0;
      memcpy(&val, &data, sizeof(double));
      mStack.push(val);
    } break;

    case BaseType::AbsolutePointer: {
      void* val = nullptr;
      memcpy(&val, &data, sizeof(void*));
      mStack.push(val);
    } break;

    case BaseType::ConstantPointer:
      mStack.push(Stack::ConstantPointer(data));
      break;

    case BaseType::VolatilePointer:
      mStack.push(Stack::VolatilePointer(data));
      break;

    default:
      GAPID_FATAL("Unhandled type in PushI");
      break;
  }

  return SUCCESS;
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

  switch (type) {
    case BaseType::Bool:
      mStack.push(*(bool*)address);
      break;

    case BaseType::Int8:
      mStack.push(*(int8_t*)address);
      break;

    case BaseType::Int16:
      mStack.push(*(int16_t*)address);
      break;

    case BaseType::Int32:
      mStack.push(*(int32_t*)address);
      break;

    case BaseType::Int64:
      mStack.push(*(int64_t*)address);
      break;

    case BaseType::Uint8:
      mStack.push(*(uint8_t*)address);
      break;

    case BaseType::Uint16:
      mStack.push(*(uint16_t*)address);
      break;

    case BaseType::Uint32:
      mStack.push(*(uint32_t*)address);
      break;

    case BaseType::Uint64:
      mStack.push(*(uint64_t*)address);
      break;

    case BaseType::Float:
      mStack.push(*(float*)address);
      break;

    case BaseType::Double:
      mStack.push(*(double*)address);
      break;

    case BaseType::AbsolutePointer:
      mStack.push(*(void**)address);
      break;

    case BaseType::ConstantPointer:
      mStack.push(Stack::ConstantPointer(*(int32_t*)address));
      break;

    case BaseType::VolatilePointer:
      mStack.push(Stack::VolatilePointer(*(int32_t*)address));
      break;

    default:
      GAPID_FATAL("Unhandled type in LoadC");
      break;
  }

  return SUCCESS;
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

  switch (type) {
    case BaseType::Bool:
      mStack.push(*(bool*)address);
      break;

    case BaseType::Int8:
      mStack.push(*(int8_t*)address);
      break;

    case BaseType::Int16:
      mStack.push(*(int16_t*)address);
      break;

    case BaseType::Int32:
      mStack.push(*(int32_t*)address);
      break;

    case BaseType::Int64:
      mStack.push(*(int64_t*)address);
      break;

    case BaseType::Uint8:
      mStack.push(*(uint8_t*)address);
      break;

    case BaseType::Uint16:
      mStack.push(*(uint16_t*)address);
      break;

    case BaseType::Uint32:
      mStack.push(*(uint32_t*)address);
      break;

    case BaseType::Uint64:
      mStack.push(*(uint64_t*)address);
      break;

    case BaseType::Float:
      mStack.push(*(float*)address);
      break;

    case BaseType::Double:
      mStack.push(*(double*)address);
      break;

    case BaseType::AbsolutePointer:
      mStack.push(*(void**)address);
      break;

    case BaseType::ConstantPointer:
      mStack.push(Stack::ConstantPointer(*(int32_t*)address));
      break;

    case BaseType::VolatilePointer:
      mStack.push(Stack::VolatilePointer(*(int32_t*)address));
      break;

    default:
      GAPID_FATAL("Unhandled type in LoadV");
      break;
  }

  return SUCCESS;
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

  switch (type) {
    case BaseType::Bool:
      mStack.push(*(bool*)address);
      break;

    case BaseType::Int8:
      mStack.push(*(int8_t*)address);
      break;

    case BaseType::Int16:
      mStack.push(*(int16_t*)address);
      break;

    case BaseType::Int32:
      mStack.push(*(int32_t*)address);
      break;

    case BaseType::Int64:
      mStack.push(*(int64_t*)address);
      break;

    case BaseType::Uint8:
      mStack.push(*(uint8_t*)address);
      break;

    case BaseType::Uint16:
      mStack.push(*(uint16_t*)address);
      break;

    case BaseType::Uint32:
      mStack.push(*(uint32_t*)address);
      break;

    case BaseType::Uint64:
      mStack.push(*(uint64_t*)address);
      break;

    case BaseType::Float:
      mStack.push(*(float*)address);
      break;

    case BaseType::Double:
      mStack.push(*(double*)address);
      break;

    case BaseType::AbsolutePointer:
      mStack.push(*(void**)address);
      break;

    case BaseType::ConstantPointer:
      mStack.push(Stack::ConstantPointer(*(uint32_t*)address));
      break;

    case BaseType::VolatilePointer:
      mStack.push(Stack::VolatilePointer(*(uint32_t*)address));
      break;

    default:
      GAPID_FATAL("Unhandled type in Load");
      break;
  }

  return SUCCESS;
}

Interpreter::Result Interpreter::pop(uint32_t opcode) {
  mStack.discard(extract26bitData(opcode));
  return SUCCESS;
}

Interpreter::Result Interpreter::storeV(uint32_t opcode) {
  void* address = mMemoryManager->volatileToAbsolute(extract26bitData(opcode));
  mStack.popTo(address);
  return SUCCESS;
}

Interpreter::Result Interpreter::store(uint32_t opcode) {
  void* address = mStack.pop<void*>();
  if (!isWriteAddress(address)) {
    GAPID_WARNING("Error: store not write address %p", address);
    return ERROR;
  }
  mStack.popTo(address);
  return SUCCESS;
}

Interpreter::Result Interpreter::resource(uint32_t opcode) {
  auto id = extract26bitData(opcode);
  mStack.push<uint32_t>(id);
  return this->call(Interpreter::RESOURCE_FUNCTION_ID);
}

Interpreter::Result Interpreter::post(uint32_t opcode) {
  return this->call(Interpreter::POST_FUNCTION_ID);
}

Interpreter::Result Interpreter::notification(uint32_t opcode) {
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
  return SUCCESS;
}

Interpreter::Result Interpreter::clone(uint32_t opcode) {
  mStack.clone(extract26bitData(opcode));
  return SUCCESS;
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
  return SUCCESS;
}

Interpreter::Result Interpreter::extend(uint32_t opcode) {
  uint32_t data = extract26bitData(opcode);
  const std::type_index& type = mStack.typeAtTop();

  bool success = false;
  if (type == typeid(Stack::VolatilePointer)) {
    auto val = mStack.pop<Stack::VolatilePointer>();
    uint32_t offset = val.getOffset();
    offset = (offset << 26) | data;
    mStack.push(Stack::VolatilePointer(offset));
    success = true;
  } else if (type == typeid(void*)) {
    auto val = mStack.pop<void*>();
    if (sizeof(void*) == 8) {
      uint64_t* ptr = (uint64_t*)&val;
      *ptr = (*ptr << 26) | data;
    } else if (sizeof(void*) == 4) {
      uint32_t* ptr = (uint32_t*)&val;
      *ptr = (*ptr << 26) | data;
    } else {
      GAPID_FATAL(
          "sizeof(void*) isn't 4 or 8? Please update Interpreter::extend().");
    }
    mStack.push(val);
    success = true;
  } else if (type == typeid(uint32_t)) {
    auto val = mStack.pop<uint32_t>();
    val = (val << 26) | data;
    mStack.push(val);
    success = true;
  } else if (type == typeid(int32_t)) {
    auto val = mStack.pop<int32_t>();
    val = (val << 26) | data;
    mStack.push(val);
    success = true;
  } else if (type == typeid(uint64_t)) {
    auto val = mStack.pop<uint64_t>();
    val = (val << 26) | data;
    mStack.push(val);
    success = true;
  } else if (type == typeid(int8_t)) {
    auto val = mStack.pop<int8_t>();
    val = (val << 26) | data;
    mStack.push(val);
    success = true;
  } else if (type == typeid(int16_t)) {
    auto val = mStack.pop<int16_t>();
    val = (val << 26) | data;
    mStack.push(val);
    success = true;
  } else if (type == typeid(int64_t)) {
    auto val = mStack.pop<int64_t>();
    val = (val << 26) | data;
    mStack.push(val);
    success = true;
  } else if (type == typeid(uint8_t)) {
    auto val = mStack.pop<uint8_t>();
    val = (val << 26) | data;
    mStack.push(val);
    success = true;
  } else if (type == typeid(uint16_t)) {
    auto val = mStack.pop<uint16_t>();
    val = (val << 26) | data;
    mStack.push(val);
    success = true;
  } else if (type == typeid(float)) {
    auto val = mStack.pop<float>();
    uint32_t* ptr = (uint32_t*)&val;
    *ptr |= (data & 0x007fffffULL);
    mStack.push(val);
    success = true;
  } else if (type == typeid(double)) {
    auto val = mStack.pop<double>();
    uint64_t* ptr = (uint64_t*)&val;
    uint64_t exponent = *ptr & 0xfff0000000000000ULL;
    *ptr <<= 26;
    *ptr |= data;
    *ptr &= 0x000fffffffffffffULL;
    *ptr |= exponent;
    mStack.push(val);
    success = true;
  } else if (type == typeid(Stack::ConstantPointer)) {
    auto val = mStack.pop<Stack::ConstantPointer>();
    uint32_t offset = val.getOffset();
    offset = (offset << 26) | data;
    mStack.push(Stack::ConstantPointer(offset));
    success = true;
  } else {
    GAPID_FATAL("Don't know how to extend this stack data type.");
  }

  return success ? SUCCESS : ERROR;
}

Interpreter::Result Interpreter::add(uint32_t opcode) {
  uint32_t count = extract26bitData(opcode);
  if (count < 2) {
    return SUCCESS;
  }

  const std::type_index& type = mStack.typeAtTop();

  if (type == typeid(void*)) {
    sum<void*>(mStack, count);
  } else if (type == typeid(int32_t)) {
    sum<int32_t>(mStack, count);
  } else if (type == typeid(int8_t)) {
    sum<int8_t>(mStack, count);
  } else if (type == typeid(int16_t)) {
    sum<int16_t>(mStack, count);
  } else if (type == typeid(int64_t)) {
    sum<int64_t>(mStack, count);
  } else if (type == typeid(uint8_t)) {
    sum<uint8_t>(mStack, count);
  } else if (type == typeid(uint16_t)) {
    sum<uint16_t>(mStack, count);
  } else if (type == typeid(uint32_t)) {
    sum<uint32_t>(mStack, count);
  } else if (type == typeid(uint64_t)) {
    sum<uint64_t>(mStack, count);
  } else if (type == typeid(float)) {
    sum<float>(mStack, count);
  } else if (type == typeid(double)) {
    sum<double>(mStack, count);
  }

  return SUCCESS;
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

Interpreter::Result Interpreter::jumpLabel(uint32_t opcode) { return SUCCESS; }

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

  return SUCCESS;
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

  return SUCCESS;
}

#undef DEBUG_OPCODE

}  // namespace gapir
