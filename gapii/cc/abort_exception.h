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

#ifndef GAPII_ABORT_EXCEPTION_H
#define GAPII_ABORT_EXCEPTION_H

#include "core/cc/target.h"

#include <exception>
#include <string>

namespace gapii {

class AbortException : public std::exception {
 public:
  enum Category {
    NORMAL, // Stop processing of the current atom as dictated by the spec.
    ASSERT, // Internal error - there is a problem that we need to address.
  };

  AbortException(Category cat, const std::string&& message) : mCat(cat), mMsg(message) {}
  Category category() const { return mCat; }
  const std::string message() const { return mMsg; }

  const char* what() const throw() { return "gapii::AbortException"; }
 private:
  Category mCat;
  std::string mMsg;
};

inline void spyAssert(bool cond, const char* message) {
  if (UNLIKELY(!cond)) {
    throw gapii::AbortException(AbortException::ASSERT, message);
  }
}

template <typename Ptr>
auto checkNotNull(Ptr ptr) -> decltype(*ptr) {
  spyAssert(ptr != nullptr, "Null pointer");
  return *ptr;
}

}  // namespace gapii

#endif  // GAPII_ABORT_EXCEPTION_H
