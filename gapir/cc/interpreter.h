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

#ifndef GAPIR_INTERPRETER_H
#define GAPIR_INTERPRETER_H

#include "function_table.h"
#include "stack.h"
#include "thread_pool.h"

#include "gapir/cc/replay_service.h"
#include "gapir/replay_service/vm.h"

#include "core/cc/crash_handler.h"

#include <stdint.h>

#include <functional>
#include <future>
#include <unordered_map>
#include <utility>

namespace gapir {

class MemoryManager;

// Implementation of a (fix sized) stack based virtual machine to interpret the
// instructions in the given opcode stream.
class Interpreter {
 public:
  // The type of the callback function for requesting to register an api's
  // renderer functions to this interpreter. Taking in the pointer for the
  // interpreter and the api index, the callback is expected to populate the
  // renderer function for the given api index in the interpreter. It should
  // return true if the request is fulfilled.
  using ApiRequestCallback = std::function<bool(Interpreter*, uint8_t)>;
  using CheckReplayStatusCallback =
      std::function<void(uint64_t, uint32_t, uint32_t)>;

  using InstructionCode = vm::Opcode;

  enum : uint32_t {
    // The API index to use for global builtin functions.
    GLOBAL_INDEX = 0,
  };

  // Function ids for implementation specific functions and special debugging
  // functions. These functions shouldn't be called by the opcode stream
  enum FunctionIds : uint16_t {
    // Custom function Ids
    POST_FUNCTION_ID = 0xff00,
    RESOURCE_FUNCTION_ID = 0xff01,
    NOTIFICATION_FUNCTION_ID = 0xff02,
    WAIT_FUNCTION_ID = 0xff03,
    // Debug function Ids
    PRINT_STACK_FUNCTION_ID = 0xff80,
    // 0xff81..0xffff reserved for synthetic functions
  };

  // Creates a new interpreter with the specified memory manager (for resolving
  // memory addresses) and with the specified maximum stack size
  Interpreter(core::CrashHandler& crash_handler,
              const MemoryManager* memory_manager, uint32_t stack_depth);

  void setApiRequestCallback(ApiRequestCallback callback);

  // Register a call back function for interpreter to report replay status.
  void setCheckReplayStatusCallback(CheckReplayStatusCallback callback);

  // Registers a builtin function to the builtin function table.
  void registerBuiltin(uint8_t api, FunctionTable::Id, FunctionTable::Function);

  // Assigns the function table as the renderer functions to use for the given
  // api.
  void setRendererFunctions(uint8_t api, FunctionTable* functionTable);

  // Runs the interpreter on the instruction list specified by the pointer and
  // by its size.
  bool run(const uint32_t* instructions, uint32_t count);

  // Resets the interpreter to be able to continue running instructions
  // from this point.
  void resetInstructions();

  // Scan the instructions building the jump destination table.
  bool updateJumpTable(uint32_t jumpLabel);

  // Registers an API instance if it has not already been done.
  bool registerApi(uint8_t api);

  // Returns the last reached label value.
  inline uint32_t getLabel() const;

 private:
  void exec();

  enum : uint32_t {
    TYPE_MASK = 0x03f00000U,
    FUNCTION_ID_MASK = 0x0000ffffU,
    API_INDEX_MASK = 0x000f0000U,
    PUSH_RETURN_MASK = 0x01000000U,
    DATA_MASK20 = 0x000fffffU,
    DATA_MASK26 = 0x03ffffffU,
    API_BIT_SHIFT = 16,
    TYPE_BIT_SHIFT = 20,
    OPCODE_BIT_SHIFT = 26,
  };

  enum Result {
    SUCCESS,
    ERROR,
    CHANGE_THREAD,
  };

  // Get type information out from an opcode. The type is always stored in the
  // 7th to 13th MSB (both inclusive) of the opcode
  BaseType extractType(uint32_t opcode) const;

  // Get 20 bit data out from an opcode located in the 20 LSB of the opcode.
  uint32_t extract20bitData(uint32_t opcode) const;

  // Get 26 bit data out from an opcode located in the 26 LSB of the opcode.
  uint32_t extract26bitData(uint32_t opcode) const;

  // Implementation of the opcodes supported by the interpreter.
  Result call(uint32_t opcode);
  Result pushI(uint32_t opcode);
  Result loadC(uint32_t opcode);
  Result loadV(uint32_t opcode);
  Result load(uint32_t opcode);
  Result pop(uint32_t opcode);
  Result storeV(uint32_t opcode);
  Result store();
  Result resource(uint32_t);
  Result post();
  Result copy(uint32_t opcode);
  Result clone(uint32_t opcode);
  Result strcpy(uint32_t opcode);
  Result extend(uint32_t opcode);
  Result add(uint32_t opcode);
  Result label(uint32_t opcode);
  Result switchThread(uint32_t opcode);
  Result jumpLabel(uint32_t opcode);
  Result jumpNZ(uint32_t opcode);
  Result jumpZ(uint32_t opcode);
  Result notification();
  Result wait(uint32_t opcode);

  // Returns true, if address..address+size(type) is "constant" memory.
  bool isConstantAddressForType(const void* address, BaseType type) const;

  // Returns true, if address..address+size(type) is "volatile" memory.
  bool isVolatileAddressForType(const void* address, BaseType type) const;

  // Returns false, if address is known not safe to read from.
  bool isReadAddress(const void* address) const;

  // Returns false, if address is known not safe to write to.
  bool isWriteAddress(void* address) const;

  // Interpret one specific opcode.
  Result interpret(uint32_t opcode);

  // The crash handler used for catching and reporting crashes.
  core::CrashHandler& mCrashHandler;

  // Memory manager which managing the memory used during the interpretation
  const MemoryManager* mMemoryManager;

  // The builtin functions. The size of this array is specified by the number of
  // supported APIs which in-turn is defined by the packing of the vm bytecode
  // (4 bits = 16 values)
  FunctionTable mBuiltins[16];

  // The current renderer functions. The size of this array is specified by the
  // number of supported APIs which in-turn is defined by the packing of the vm
  // bytecode (4 bits = 16 values)
  FunctionTable* mRendererFunctions[16] = {};

  // Callback function for requesting renderer functions for an unknown api.
  ApiRequestCallback apiRequestCallback;

  // Callback function for checking replay progress and send back info to GAPIS
  // at right time.
  CheckReplayStatusCallback checkReplayStatusCallback;

  // The stack of the Virtual Machine.
  Stack mStack;

  // The list of instructions.
  const uint32_t* mInstructions;

  // The total number of instructions.
  uint32_t mInstructionCount;

  // The index of the current instruction.
  uint32_t mCurrentInstruction;

  // The next thread execution should continue on.
  uint32_t mNextThread;

  // The last reached label value.
  uint32_t mLabel;

  // The result of the thread-chained exec() calls.
  std::promise<Result> mExecResult;

  // The thread pool used to interpret on different threads.
  ThreadPool mThreadPool;

  // Jump ID to instruction ID
  std::map<uint32_t, uint32_t> mJumpLabels;
};

inline bool Interpreter::isConstantAddressForType(const void* address,
                                                  BaseType type) const {
  // Treat all pointer types as sizeof(void*)
  size_t size = isPointerType(type) ? sizeof(void*) : baseTypeSize(type);
  return mMemoryManager->isConstantAddressWithSize(address, size);
}

inline bool Interpreter::isVolatileAddressForType(const void* address,
                                                  BaseType type) const {
  return mMemoryManager->isVolatileAddressWithSize(address, baseTypeSize(type));
}

inline bool Interpreter::isReadAddress(const void* address) const {
  return address != nullptr &&
         !mMemoryManager->isNotObservedAbsoluteAddress(address);
}

inline bool Interpreter::isWriteAddress(void* address) const {
  return address != nullptr &&
         !mMemoryManager->isNotObservedAbsoluteAddress(address) &&
         !mMemoryManager->isConstantAddress(address);
}

inline uint32_t Interpreter::getLabel() const { return mLabel; }

}  // namespace gapir

#endif  // GAPIR_INTERPRETER_H
