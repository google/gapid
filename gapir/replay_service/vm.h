/*
 * Copyright (C) 2018 Google Inc.
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

#ifndef GAPIR_SERVICE_VM_H
#define GAPIR_SERVICE_VM_H

#include <stdint.h>

namespace gapir {
namespace vm {

// Opcodes for the different instructions.
// The codes have to be consistent with the codes on the server side.
enum class Opcode {
  CALL = 0,
  PUSH_I = 1,
  LOAD_C = 2,
  LOAD_V = 3,
  LOAD = 4,
  POP = 5,
  STORE_V = 6,
  STORE = 7,
  RESOURCE = 8,
  POST = 9,
  COPY = 10,
  CLONE = 11,
  STRCPY = 12,
  EXTEND = 13,
  ADD = 14,
  LABEL = 15,
  SWITCH_THREAD = 16,
  JUMP_LABEL = 17,
  JUMP_NZ = 18,
  JUMP_Z = 19,
  NOTIFICATION = 20,
  WAIT = 21,
  INLINE_RESOURCE = 22,
  NUM_OPCODES = 23,
};

// Unique ID for each supported data type. The ID have to fit into 6 bits (0-63)
// to fit into the opcode stream and the values have to be consistent with the
// values on the server side
enum class Type {
  Bool = 0,
  Int8 = 1,
  Int16 = 2,
  Int32 = 3,
  Int64 = 4,
  Uint8 = 5,
  Uint16 = 6,
  Uint32 = 7,
  Uint64 = 8,
  Float = 9,
  Double = 10,
  AbsolutePointer = 11,
  ConstantPointer = 12,
  VolatilePointer = 13,
};

}  // namespace vm
}  // namespace gapir

#endif  // GAPIR_SERVICE_VM_H
