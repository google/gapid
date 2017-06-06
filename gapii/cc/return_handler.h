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

#ifndef GAPII_RETURN_HANDLER_H
#define GAPII_RETURN_HANDLER_H

#include "abort_exception.h"

#include <atomic>
#include <string>
#include <typeindex>
#include <typeinfo>
#include <unordered_map>

namespace gapii {

class ReturnHandler {
 public:
  // Save the return value.
  template <typename T> void setReturnValue(const T& returnVal) {
    mReturnValues[std::type_index(typeid(T))] = new T(returnVal);
  }

  // Get back the saved return value and remove it.
  template <typename T> T getAndClearReturnValue() {
    auto tid = std::type_index(typeid(T));
    auto it = mReturnValues.find(tid);
    GAPID_ASSERT(it != mReturnValues.end() /* getAndClassReturnValue called, but hasReturnValue is false */);
    T returnVal{};
    T* val = static_cast<T*>(it->second);
    std::swap(*val, returnVal);
    delete val;
    mReturnValues.erase(it);
    return returnVal;
  }

  // Returns true if setReturnValue<T> was called and getAndClearReturnValue<T>
  // has not been called.
  template <typename T> bool hasReturnValue() const {
    return mReturnValues.find(std::type_index(typeid(T))) != mReturnValues.end();
  }
 private:
  std::unordered_map<std::type_index, void*> mReturnValues;
};

}  // namespace gapii

#endif // GAPII_SPY_BASE_H
