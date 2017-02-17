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

#ifndef CORE_STATIC_ARRAY_H
#define CORE_STATIC_ARRAY_H

#include "assert.h"

namespace core {

// StaticArray represents a fixed size array with implicit conversions to and
// from T[N].
template <typename T, int N>
class StaticArray {
  typedef T ARRAY_TY[N];
public:

  inline StaticArray();
  inline StaticArray(const StaticArray<T, N>& other);
  inline StaticArray(T arr[N]);
  inline StaticArray(std::initializer_list<T> l);

  inline operator T*();
  inline operator const T*() const;

private:
  T mData[N];
};

template <typename T, int N>
inline StaticArray<T, N>::StaticArray() : mData{} {}

template <typename T, int N>
inline StaticArray<T, N>::StaticArray(const StaticArray<T, N>& other) {
    for (int i = 0; i < N; i++) {
        mData[i] = other[i];
    }
}

template <typename T, int N>
inline StaticArray<T, N>::StaticArray(T arr[N]) : mData{} {
    for (int i = 0; i < N; i++) {
        mData[i] = arr[i];
    }
}

template <typename T, int N>
inline StaticArray<T, N>::StaticArray(std::initializer_list<T> l) : mData{} {
    GAPID_ASSERT(l.size() == N);
    for (int i = 0; i < N; i++) {
        mData[i] = l.begin()[i];
    }
}

template <typename T, int N>
inline StaticArray<T, N>::operator T*() {
    return &mData[0];
}

template <typename T, int N>
inline StaticArray<T, N>::operator const T*() const {
    return &mData[0];
}

}  // namespace core

#endif  // CORE_STATIC_ARRAY_H
