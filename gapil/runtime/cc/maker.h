// Copyright (C) 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#ifndef __GAPIL_RUNTIME_ZERO_H__
#define __GAPIL_RUNTIME_ZERO_H__

#include <type_traits>
#include <utility>

namespace core {
class Arena;
}  // namespace core

namespace gapil {

// Maker is a template helper used by gapil::make() and gapil::inplace_new().
template <typename T, bool TAKES_ARENA>
struct Maker;

template <typename T>
struct Maker<T, true> {
  template <typename... ARGS>
  static inline T make(core::Arena* a, ARGS&&... args) {
    return T(a, std::forward<ARGS>(args)...);
  }

  template <typename... ARGS>
  static inline void inplace_new(T* ptr, core::Arena* a, ARGS&&... args) {
    new (ptr) T(a, std::forward<ARGS>(args)...);
  }
};

template <typename T>
struct Maker<T, false> {
  template <typename... ARGS>
  static inline T make(core::Arena*, ARGS&&... args) {
    return T(std::forward<ARGS>(args)...);
  }

  template <typename... ARGS>
  static inline void inplace_new(T* ptr, core::Arena* a, ARGS&&... args) {
    new (ptr) T(std::forward<ARGS>(args)...);
  }
};

// Special case for void* that can incorrectly convert the arena* to the target
// type.
template <>
struct Maker<void*, true> {
  static inline void* make(core::Arena* a) { return nullptr; }
  static inline void inplace_new(void** ptr, core::Arena* a) { *ptr = nullptr; }
};
template <>
struct Maker<const void*, true> {
  static inline void* make(core::Arena* a) { return nullptr; }
  static inline void inplace_new(void** ptr, core::Arena* a) { *ptr = nullptr; }
};

// Special case for bool, because std::is_constructible<bool, core::Arena*>
// is true, but without an arg, we want it to return false.
template <>
struct Maker<bool, true> {
  static inline bool make(core::Arena* a) { return false; }
  static inline void inplace_new(bool* ptr, core::Arena* a) { *ptr = false; }
};

// make returns a T constructed by the list of args.
// If T has a core::Arena* as the first constructor parameter then a is
// prepended to the list of arguments.
template <typename T, typename... ARGS>
inline T make(core::Arena* a, ARGS&&... args) {
  return Maker<typename std::remove_cv<T>::type,
               std::is_constructible<T, core::Arena*, ARGS...>::value>::
      make(a, std::forward<ARGS>(args)...);
}

// inplace_new constructs a T at ptr using the list of args.
// If T has a core::Arena* as the first constructor parameter then a is
// prepended to the list of arguments.
template <typename T, typename... ARGS>
inline void inplace_new(T* ptr, core::Arena* a, ARGS&&... args) {
  Maker<typename std::remove_cv<T>::type,
        std::is_constructible<T, core::Arena*, ARGS...>::value>::
      inplace_new(ptr, a, std::forward<ARGS>(args)...);
}

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_ZERO_H__
