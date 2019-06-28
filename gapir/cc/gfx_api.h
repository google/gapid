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

#ifndef GAPIR_GFX_API_H
#define GAPIR_GFX_API_H

#include <stdint.h>

#include "function_table.h"

namespace gapir {

class Stack;

// Api is the abstract base class to all graphics APIs.
class Api {
 public:
  virtual ~Api() {}
  // Returns the unique identifier of the graphics API.
  // The pointer is guaranteed to be constant for all instances of the API.
  virtual const char* id() const = 0;
  // Returns the index of the graphics API.
  virtual uint8_t index() const = 0;
  // The function table for the API.
  FunctionTable mFunctions;
};

// LazyResolved is for resolving indirect commands only when the commands are
// going to be called. It takes a resolver callback, which will be used for
// command resolving for the first time the command is to be called. And caches
// the resolved pointer.
template <typename FuncPtr>
class LazyResolved {
 public:
  LazyResolved() : resolve_(nullptr), ptr_(nullptr) {}
  LazyResolved(std::nullptr_t) : LazyResolved() {}
  explicit LazyResolved(std::function<void*()> resolver)
      : resolve_(resolver), ptr_(nullptr) {}
  // Pass forward the arguments to the command, if the function has never been
  // resolved before, resolve it first.
  template <typename... Args>
  typename std::result_of<FuncPtr(Args...)>::type operator()(Args&&... args) {
    if (!ptr_) {
      ptr_ = reinterpret_cast<FuncPtr>(resolve_());
    }
    return ptr_(std::forward<Args>(args)...);
  }
  // Overloaded not-equal nullptr comparison. Returns true if the underlying
  // function can be resolved and is not nullptr.
  bool operator!=(std::nullptr_t) {
    if (!resolve_) {
      return true;
    }
    if (!ptr_) {
      ptr_ = reinterpret_cast<FuncPtr>(resolve_());
    }
    return ptr_ != nullptr;
  }
  // Overloaded boolean comparison. Returns true if the underlying function can
  // be resolved and is not nullptr.
  operator bool() {
    if (!resolve_) {
      return false;
    }
    if (!ptr_) {
      ptr_ = reinterpret_cast<FuncPtr>(resolve_());
    }
    return ptr_ != nullptr;
  }

 private:
  // The function resolving callback.
  std::function<void*()> resolve_;
  // The cached function pointer to the underlying function.
  FuncPtr ptr_;
};

}  // namespace gapir

#endif  // GAPIR_GFX_API_H
