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

#ifndef GAPIR_BASE_TYPE_H
#define GAPIR_BASE_TYPE_H

#include <stdint.h>

#include <type_traits>

#include "gapir/replay_service/vm.h"

namespace gapir {

using BaseType = vm::Type;

// Return the size of the underlying type for the given BaseType
uint32_t baseTypeSize(BaseType type);

// Return the name of the given BaseType
const char* baseTypeName(BaseType type);

inline bool isValid(BaseType type) {
  return type >= BaseType::Bool && type <= BaseType::VolatilePointer;
}

// Provide the BaseType value corresponding to the type specified in T.
// For pointers the corresponding base type is AbsolutePointer
// For enums the corresponding base type is uint32_t
template <typename T, typename = void>
struct TypeToBaseType;

template <typename T>
struct TypeToBaseType<T*> {
  static const BaseType type = BaseType::AbsolutePointer;
};

template <>
struct TypeToBaseType<bool> {
  static const BaseType type = BaseType::Bool;
};
template <>
struct TypeToBaseType<int8_t> {
  static const BaseType type = BaseType::Int8;
};
template <>
struct TypeToBaseType<int16_t> {
  static const BaseType type = BaseType::Int16;
};
template <>
struct TypeToBaseType<int32_t> {
  static const BaseType type = BaseType::Int32;
};
template <>
struct TypeToBaseType<int64_t> {
  static const BaseType type = BaseType::Int64;
};
template <>
struct TypeToBaseType<uint8_t> {
  static const BaseType type = BaseType::Uint8;
};
template <>
struct TypeToBaseType<uint16_t> {
  static const BaseType type = BaseType::Uint16;
};
template <>
struct TypeToBaseType<uint32_t> {
  static const BaseType type = BaseType::Uint32;
};
template <>
struct TypeToBaseType<uint64_t> {
  static const BaseType type = BaseType::Uint64;
};
template <>
struct TypeToBaseType<float> {
  static const BaseType type = BaseType::Float;
};
template <>
struct TypeToBaseType<double> {
  static const BaseType type = BaseType::Double;
};

template <typename T>
struct TypeToBaseType<T,
                      typename std::enable_if<std::is_enum<T>::value>::type> {
  static const BaseType type = BaseType::Uint32;
};

// isPointerType returns true if values of 'type' translate to a pointer.
inline bool isPointerType(BaseType type) {
  switch (type) {
    case BaseType::AbsolutePointer:
    case BaseType::ConstantPointer:
    case BaseType::VolatilePointer:
      return true;
    default:
      return false;
  }
}

}  // namespace gapir

#endif  // GAPIR_BASE_TYPE_H
