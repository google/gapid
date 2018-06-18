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

#include "base_type.h"

#include "core/cc/log.h"

namespace gapir {

uint32_t baseTypeSize(BaseType type) {
  static_assert(sizeof(uint8_t) == 1, "Size of a uint8_t must be 1!");
  switch (type) {
    case BaseType::Bool:
      return sizeof(bool);
    case BaseType::Int8:
      return sizeof(int8_t);
    case BaseType::Int16:
      return sizeof(int16_t);
    case BaseType::Int32:
      return sizeof(int32_t);
    case BaseType::Int64:
      return sizeof(int64_t);
    case BaseType::Uint8:
      return sizeof(uint8_t);
    case BaseType::Uint16:
      return sizeof(uint16_t);
    case BaseType::Uint32:
      return sizeof(uint32_t);
    case BaseType::Uint64:
      return sizeof(uint64_t);
    case BaseType::Float:
      return sizeof(float);
    case BaseType::Double:
      return sizeof(double);
    case BaseType::AbsolutePointer:
      return sizeof(void*);
    case BaseType::ConstantPointer:
    case BaseType::VolatilePointer:
      return sizeof(uint32_t);
    default:
      GAPID_FATAL("Invalid BaseType: %d", int(type));
      return 0;
  }
}

const char* baseTypeName(BaseType type) {
  switch (type) {
    case BaseType::Bool:
      return "bool";
    case BaseType::Int8:
      return "int8";
    case BaseType::Int16:
      return "int16";
    case BaseType::Int32:
      return "int32";
    case BaseType::Int64:
      return "int64";
    case BaseType::Uint8:
      return "uint8";
    case BaseType::Uint16:
      return "uint16";
    case BaseType::Uint32:
      return "uint32";
    case BaseType::Uint64:
      return "uint64";
    case BaseType::Float:
      return "float";
    case BaseType::Double:
      return "double";
    case BaseType::AbsolutePointer:
      return "absolute pointer";
    case BaseType::ConstantPointer:
      return "constant pointer";
    case BaseType::VolatilePointer:
      return "volatile pointer";
    default:
      GAPID_FATAL("Invalid BaseType: %d", int(type));
      return "unknown";
  }
}

}  // namespace gapir
