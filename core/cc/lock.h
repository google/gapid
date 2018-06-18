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

#ifndef CORE_LOCK_H
#define CORE_LOCK_H

namespace core {

// Lock is a RAII helper that calls T::lock() on construction and T::unlock() on
// destruction.
template <typename T>
struct Lock {
 public:
  inline Lock(T* t);
  inline ~Lock();

 private:
  T* _;
};

template <typename T>
inline Lock<T>::Lock(T* t) : _(t) {
  _->lock();
};

template <typename T>
inline Lock<T>::~Lock() {
  _->unlock();
};

}  // namespace core

#endif  // CORE_LOCK_H
