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

#ifndef GAPIR_FUNCTION_TABLE_H
#define GAPIR_FUNCTION_TABLE_H

#include "core/cc/log.h"

#include <stdint.h>
#include <functional>
#include <unordered_map>

namespace gapir {

class Stack;

// FunctionTable provides a mapping of function id to a VM function.
class FunctionTable {
 public:
  // General signature for functions callable by the interpreter with a function
  // call instruction. The first argument is a pointer to the stack of the
  // Virtual Machine and the second argument is true if the caller expect the
  // return value of the function to be pushed to the stack. The function should
  // return true if the function call was successful, false otherwise.
  typedef std::function<bool(uint32_t, Stack*, bool)> Function;

  FunctionTable() : mFunctions(new Function[65536]) {
    for (int i = 0; i < 65536; ++i) mFunctions[i] = nullptr;
  }
  ~FunctionTable() { delete[] mFunctions; }

  // The function identifier. These are part of the protocol between the server
  // and the replay system, and so must remain consistent.
  typedef uint16_t Id;

  // Inserts a function into the table.
  inline void insert(Id id, Function func);

  // Returns a function from the table, or nullptr if there is no function with
  // the specified identifier.
  inline Function* lookup(Id id);

 private:
  // Array of the actual function implementations by function ID
  // This is stored as an array rather than a map because many lookups are
  // performed at replay time and this becomes a bottleneck when stored as a
  // map. The 64k entries (limit mandated elsewhere in the code by the vm
  // bytecode instruction packing) are small enough that storing them as an
  // array isn't a problem. These are dynamically allocated because
  // windows fails at compile time with an inline array.
  Function* mFunctions;
};

inline FunctionTable::Function* FunctionTable::lookup(Id id) {
  Function& ret = mFunctions[id];
  if (ret == nullptr) {
    return nullptr;
  }
  return &ret;
}

inline void FunctionTable::insert(Id id, Function func) {
  if (mFunctions[id] != NULL) {
    GAPID_FATAL("Duplicate functions inserted into table");
  }
  mFunctions[id] = func;
}

}  // namespace gapir

#endif  // GAPIR_FUNCTION_TABLE_H
